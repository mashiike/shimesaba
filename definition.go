package shimesaba

import (
	"context"
	"log"
	"time"

	"github.com/mashiike/evaluator"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

//Definition is SLI/SLO Definition
type Definition struct {
	id              string
	serviceName     string
	timeFrame       time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	objectives []evaluator.Comparator
}

//NewDefinition creates Definition from DefinitionConfig
func NewDefinition(cfg *DefinitionConfig) (*Definition, error) {
	objectives := make([]evaluator.Comparator, 0, len(cfg.Objectives))
	for _, objCfg := range cfg.Objectives {
		objectives = append(objectives, objCfg.GetComparator())
	}
	return &Definition{
		id:              cfg.ID,
		serviceName:     cfg.ServiceName,
		timeFrame:       cfg.DurationTimeFrame(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSize,
		objectives:      objectives,
	}, nil
}

// ID returns DefinitionConfig.id
func (d *Definition) ID() string {
	return d.id
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, metrics Metrics) ([]*Report, error) {
	upFlag := make(map[time.Time]bool)
	for _, o := range d.objectives {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		isUp := MetricsComparate(o, metrics, metrics.StartAt(), metrics.EndAt())
		for t, f := range isUp {
			if u, ok := upFlag[t]; ok {
				upFlag[t] = u && f
			} else {
				upFlag[t] = f
			}
		}
	}
	outerStartAt := metrics.StartAt().Add(d.timeFrame + d.calculate).Truncate(d.calculate)
	outerEndAt := metrics.EndAt().Truncate(d.calculate)
	outerIter := timeutils.NewIterator(outerStartAt, outerEndAt, d.calculate)
	outerIter.SetEnableOverWindow(true)
	log.Printf("[info] definition[%s] calculate range %s ~ %s\n", d.id, outerStartAt, outerEndAt)
	aggInterval := metrics.AggregationInterval()
	reports := make([]*Report, 0)
	for outerIter.HasNext() {
		curAt, _ := outerIter.Next()
		var upTime, failureTime time.Duration
		var deltaFailureTime time.Duration

		report := NewReport(d.id, d.serviceName, curAt, d.timeFrame, d.errorBudgetSize)
		innerIter := timeutils.NewIterator(report.TimeFrameStartAt, report.TimeFrameEndAt, aggInterval)
		for innerIter.HasNext() {
			t, _ := innerIter.Next()
			if isUp, ok := upFlag[t]; ok && !isUp {
				failureTime += aggInterval
				if report.DataPoint.Sub(t) < d.calculate {
					deltaFailureTime += aggInterval
				}
			} else {
				upTime += aggInterval
			}
		}
		if upTime+failureTime != d.timeFrame {
			log.Printf("[warn] definition[%s]<%s> up_time<%s> + failure_time<%s> != time_frame<%s> maybe drop data point\n", d.id, curAt, upTime, failureTime, d.timeFrame)
			upTime = d.timeFrame - failureTime
		}
		report.SetTime(upTime, failureTime, deltaFailureTime)
		log.Printf("[debug] %s\n", report)
		reports = append(reports, report)
	}
	return reports, nil
}

func MetricsComparate(c evaluator.Comparator, metrics Metrics, startAt, endAt time.Time) map[time.Time]bool {
	variables := metrics.GetVariables(startAt, endAt)
	ret := make(map[time.Time]bool, len(variables))
	for t, v := range variables {
		b, err := c.Compare(v)
		if evaluator.IsDivideByZero(err) {
			continue
		}
		if err != nil {
			log.Printf("[warn] compare failed expr=%s time=%s reason=%s", c.String(), t, err)
			continue
		}
		ret[t] = b
	}
	return ret
}
