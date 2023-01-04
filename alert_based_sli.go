package shimesaba

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
	"golang.org/x/sync/errgroup"
)

type AlertBasedSLI struct {
	cfg *AlertBasedSLIConfig
}

func NewAlertBasedSLI(cfg *AlertBasedSLIConfig) *AlertBasedSLI {
	return &AlertBasedSLI{cfg: cfg}
}

var evaluateReliabilitiesWorkerNum int = 10

func init() {
	if str := os.Getenv("SHIMESABA_EVALUATE_RELIABILITIES_WORKER_NUM"); str != "" {
		i, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			panic(fmt.Errorf("SHIMESABA_EVALUATE_RELIABILITIES_WORKER_NUM can not parse as int: %w", err))
		}
		evaluateReliabilitiesWorkerNum = int(i)
		if evaluateReliabilitiesWorkerNum <= 0 {
			evaluateReliabilitiesWorkerNum = 1
		}
	}
}
func (o AlertBasedSLI) EvaluateReliabilities(timeFrame time.Duration, alerts Alerts, startAt, endAt time.Time) (Reliabilities, error) {
	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	iter.SetEnableOverWindow(true)
	rc := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		rc = append(rc, NewReliability(cursorAt, timeFrame, nil))
	}
	reliabilities, err := NewReliabilities(rc)
	if err != nil {
		return nil, err
	}

	inputQueue := make(chan *Alert, len(alerts))
	outputQueue := make(chan Reliabilities, evaluateReliabilitiesWorkerNum*2)
	quit := make(chan struct{})
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, egCtx := errgroup.WithContext(cancelCtx)
	for i := 0; i < evaluateReliabilitiesWorkerNum; i++ {
		//input workers
		workerID := i
		eg.Go(func() error {
			log.Printf("[debug] start EvaluateReliabilities input worker_id=%d", workerID)
			for {
				select {
				case <-egCtx.Done():
					log.Printf("[debug] end EvaluateReliabilities input worker_id=%d: %v", workerID, egCtx.Err())
					return egCtx.Err()
				case <-quit:
					log.Printf("[debug] end EvaluateReliabilities input worker_id=%d: quit", workerID)
					return nil
				case alert, ok := <-inputQueue:
					if !ok {
						log.Printf("[debug] end EvaluateReliabilities input worker_id=%d: success", workerID)
						return nil
					}
					log.Printf("[debug] worker_id=%d EvaluateReliabilities %s", workerID, alert.String())
					tmp, err := alert.EvaluateReliabilities(timeFrame, o.cfg.TryReassessment)
					if err != nil {
						log.Printf("[debug] end EvaluateReliabilities input worker_id=%d: EvaluateReliabilities err: %v", workerID, err)
						return err
					}
					outputQueue <- tmp
				}
			}
		})
	}
	var outputErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// output worker
		log.Printf("[debug] start EvaluateReliabilities output worker")
		defer wg.Done()
		log.Printf("[debug] end EvaluateReliabilities output worker")
		for {
			select {
			case <-cancelCtx.Done():
				log.Printf("[debug] end EvaluateReliabilities output worker: %v", cancelCtx.Err())
				return
			case <-quit:
				log.Printf("[debug] end EvaluateReliabilities output worker: quit")
				return
			case tmp, ok := <-outputQueue:
				if !ok {
					// Completed evaluation of all alerts
					log.Printf("[debug] end EvaluateReliabilities output worker: success")
					return
				}
				reliabilities, outputErr = reliabilities.MergeInRange(tmp, startAt, endAt)
				if outputErr != nil {
					log.Printf("[debug] end EvaluateReliabilities output worker: MergeInRange err: %v", err)
					return
				}
			}
		}
	}()

	for _, alert := range alerts {
		if !o.matchAlert(alert) {
			continue
		}
		inputQueue <- alert
	}
	close(inputQueue)

	// wait input wokers done
	if err := eg.Wait(); err != nil {
		// send quit to output worker and wait output woker done.
		close(quit)
		wg.Wait()
		return nil, err
	}

	// Evaluation of all alerts was completed. Close queue of ouptut and wait for merge process.
	close(outputQueue)
	wg.Wait()
	return reliabilities, nil
}

func (o AlertBasedSLI) matchAlert(alert *Alert) bool {
	if alert.IsVirtual() {
		return true
	}
	log.Printf("[debug] try match %s vs %v", alert, o.cfg)
	if o.MatchMonitor(alert.Monitor) {
		log.Printf("[debug] match %s", alert)
		return true
	}
	return false
}

func (o AlertBasedSLI) MatchMonitor(monitor *Monitor) bool {
	if o.cfg.MonitorID != "" {
		if monitor.ID() != o.cfg.MonitorID {
			return false
		}
	}
	if o.cfg.MonitorName != "" {
		if monitor.Name() != o.cfg.MonitorName {
			return false
		}
	}
	if o.cfg.MonitorNamePrefix != "" {
		if !strings.HasPrefix(monitor.Name(), o.cfg.MonitorNamePrefix) {
			return false
		}
	}
	if o.cfg.MonitorNameSuffix != "" {
		if !strings.HasSuffix(monitor.Name(), o.cfg.MonitorNameSuffix) {
			return false
		}
	}
	if o.cfg.MonitorType != "" {
		if !strings.EqualFold(monitor.Type(), o.cfg.MonitorType) {
			return false
		}
	}
	return true
}
