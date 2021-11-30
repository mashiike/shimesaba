package shimesaba

import (
	"context"
	"encoding/json"
	"fmt"
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

		report := &Report{
			DefinitionID:     d.id,
			ServiceName:      d.serviceName,
			DataPoint:        curAt,
			TimeFrameStartAt: curAt.Add(-d.timeFrame),
			TimeFrameEndAt:   curAt.Add(-time.Nanosecond),
			ErrorBudgetSize:  time.Duration(d.errorBudgetSize * float64(d.timeFrame)).Truncate(time.Minute),
		}
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
		report.UpTime = upTime
		report.FailureTime = failureTime
		report.ErrorBudget = (report.ErrorBudgetSize - failureTime).Truncate(time.Minute)
		report.ErrorBudgetConsumption = deltaFailureTime.Truncate(time.Minute)
		log.Printf("[debug] %s\n", report)
		reports = append(reports, report)
	}
	return reports, nil
}

// Report has SLI/ SLO/ErrorBudget numbers in one rolling window
type Report struct {
	DefinitionID           string
	ServiceName            string
	DataPoint              time.Time
	TimeFrameStartAt       time.Time
	TimeFrameEndAt         time.Time
	UpTime                 time.Duration
	FailureTime            time.Duration
	ErrorBudgetSize        time.Duration
	ErrorBudget            time.Duration
	ErrorBudgetConsumption time.Duration
}

// String implements fmt.Stringer
func (r *Report) String() string {
	return fmt.Sprintf("definition[%s][%s]<%s~%s> up_time=%s failure_time=%s error_budget=%s(usage:%f)", r.DefinitionID, r.DataPoint, r.TimeFrameStartAt, r.TimeFrameEndAt, r.UpTime, r.FailureTime, r.ErrorBudget, r.ErrorBudgetUsageRate()*100.0)
}

// ErrorBudgetUsageRate returns (1.0 - ErrorBudget/ErrorBudgetSize)
func (r *Report) ErrorBudgetUsageRate() float64 {
	if r.ErrorBudget >= 0 {
		return 1.0 - float64(r.ErrorBudget)/float64(r.ErrorBudgetSize)
	}
	return -float64(r.ErrorBudget-r.ErrorBudgetSize) / float64(r.ErrorBudgetSize)
}

// ErrorBudgetConsumptionRate returns ErrorBudgetConsumption/ErrorBudgetSize
func (r *Report) ErrorBudgetConsumptionRate() float64 {
	return float64(r.ErrorBudgetConsumption) / float64(r.ErrorBudgetSize)
}

// MarshalJSON implements json.Marshaler
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
