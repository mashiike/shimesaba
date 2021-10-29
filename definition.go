package shimesaba

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
)

type Definition struct {
	id              string
	timeFrame       time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	objectives []*MetricComparator
}

func NewDefinition(cfg *DefinitionConfig) (*Definition, error) {
	objectives := make([]*MetricComparator, 0, len(cfg.Objectives))
	for _, ocfg := range cfg.Objectives {
		comparator, err := NewMetricComparator(ocfg.Expr)
		if err != nil {
			return nil, err
		}
		objectives = append(objectives, comparator)
	}
	return &Definition{
		id:              cfg.ID,
		timeFrame:       cfg.DurationTimeFrame(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSize,
		objectives:      objectives,
	}, nil
}

func (d *Definition) ID() string {
	return d.id
}

func (d *Definition) CreateRepoorts(ctx context.Context, metrics Metrics) ([]*Report, error) {
	upFlag := make(map[time.Time]bool)
	for _, o := range d.objectives {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		isUp, err := o.Eval(metrics, metrics.StartAt(), metrics.EndAt())
		if err != nil {
			return nil, err
		}
		for t, f := range isUp {
			if u, ok := upFlag[t]; ok {
				upFlag[t] = u && f
			} else {
				upFlag[t] = f
			}
		}
	}
	outerStartAt := timeutils.TruncTime(metrics.StartAt().Add(d.timeFrame+d.calculate), d.calculate)
	outerEndAt := timeutils.TruncTime(metrics.EndAt(), d.calculate)
	outerIter := timeutils.NewIterator(outerStartAt, outerEndAt, d.calculate)
	outerIter.SetEnableOverWindow(true)
	log.Printf("[info] definition[%s] calculate range %s ~ %s\n", d.id, outerStartAt, outerEndAt)
	aggInterval := metrics.AggregationInterval()
	reports := make([]*Report, 0)
	for outerIter.HasNext() {
		curAt, _ := outerIter.Next()
		var upTime, faiureTime time.Duration
		var deltaFaiureTime time.Duration

		report := &Report{
			DefinitionID:     d.id,
			DataPoint:        curAt,
			TimeFrameStartAt: curAt.Add(-d.timeFrame),
			TimeFrameEndAt:   curAt.Add(-time.Nanosecond),
			ErrorBudgetSize:  time.Duration(d.errorBudgetSize * float64(d.timeFrame)),
		}
		innerIter := timeutils.NewIterator(report.TimeFrameStartAt, report.TimeFrameEndAt, aggInterval)
		for innerIter.HasNext() {
			t, _ := innerIter.Next()
			if isUp, ok := upFlag[t]; ok && !isUp {
				log.Printf("[debug] definition[%s]<%s> is failure", d.id, t)
				faiureTime += aggInterval
				if report.DataPoint.Sub(t) < d.calculate {
					deltaFaiureTime += aggInterval
				}
			} else {
				upTime += aggInterval
			}
		}
		if upTime+faiureTime != d.timeFrame {
			log.Printf("[warn] definition[%s]<%s> up_time<%s> + faiure_time<%s> != time_frame<%s> maybe drop data point\n", d.id, curAt, upTime, faiureTime, d.timeFrame)
			upTime = d.timeFrame - faiureTime
		}
		report.UpTime = upTime
		report.FailureTime = faiureTime
		report.ErrorBudget = report.ErrorBudgetSize - faiureTime
		report.ErrorBudgetConsumption = deltaFaiureTime
		log.Printf("[debug] %s\n", report)
		reports = append(reports, report)
	}
	return reports, nil
}

type Report struct {
	DefinitionID           string
	DataPoint              time.Time
	TimeFrameStartAt       time.Time
	TimeFrameEndAt         time.Time
	DeltaUpTime            time.Duration
	DeltaFaiureTime        time.Duration
	UpTime                 time.Duration
	FailureTime            time.Duration
	ErrorBudgetSize        time.Duration
	ErrorBudget            time.Duration
	ErrorBudgetConsumption time.Duration
}

func (r *Report) String() string {
	return fmt.Sprintf("definition[%s]<%s~%s> up_time=%s faiure_time=%s error_budget=%s(usage:%f)", r.DefinitionID, r.TimeFrameStartAt, r.TimeFrameEndAt, r.UpTime, r.FailureTime, r.ErrorBudget, r.ErrorBudgetUsageRate()*100.0)
}

func (r *Report) ErrorBudgetUsageRate() float64 {
	if r.ErrorBudget >= 0 {
		return 1.0 - float64(r.ErrorBudget)/float64(r.ErrorBudgetSize)
	}
	return -float64(r.ErrorBudget-r.ErrorBudgetSize) / float64(r.ErrorBudgetSize)
}

func (r *Report) ErrorBudgetConsumptionRate() float64 {
	return float64(r.ErrorBudgetConsumption) / float64(r.ErrorBudgetSize)
}

func (r *Report) MarshalJSON() ([]byte, error) {
	d := struct {
		DefinitionID               string    `json:"definition_id" yaml:"definition_id"`
		DataPoint                  time.Time `json:"data_point" yaml:"data_point"`
		TimeFrameStartAt           time.Time `json:"time_frame_start_at" yaml:"time_frame_start_at"`
		TimeFrameEndAt             time.Time `json:"time_frame_end_at" yaml:"time_frame_end_at"`
		UpTime                     float64   `json:"up_time" yaml:"up_time"`
		FailureTime                float64   `json:"failure_time" yaml:"failure_time"`
		ErrorBudgetSize            float64   `json:"error_budget_size" yaml:"error_budget_size"`
		ErrorBudget                float64   `json:"error_budget" yaml:"error_budget"`
		ErrorBudgetUsageRate       float64   `json:"error_budget_usage_rate" yaml:"error_budget_usage_rate"`
		ErrorBudgetConsumption     float64   `json:"error_budget_consumption" yaml:"error_budget_consumption"`
		ErrorBudgetConsumptionRate float64   `json:"error_budget_consumption_rate" yaml:"error_budget_consumption_rate"`
	}{
		DefinitionID:               r.DefinitionID,
		DataPoint:                  r.DataPoint,
		TimeFrameStartAt:           r.TimeFrameStartAt,
		TimeFrameEndAt:             r.TimeFrameEndAt,
		UpTime:                     r.UpTime.Minutes(),
		FailureTime:                r.FailureTime.Minutes(),
		ErrorBudgetSize:            r.ErrorBudgetSize.Minutes(),
		ErrorBudget:                r.ErrorBudget.Minutes(),
		ErrorBudgetUsageRate:       r.ErrorBudgetUsageRate(),
		ErrorBudgetConsumption:     r.ErrorBudgetConsumption.Minutes(),
		ErrorBudgetConsumptionRate: r.ErrorBudgetConsumptionRate(),
	}
	return json.Marshal(d)
}
