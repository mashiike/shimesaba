package shimesaba

import (
	"encoding/json"
	"fmt"
	"time"
)

// Report has SLI/SLO/ErrorBudget numbers in one rolling window
type Report struct {
	DefinitionID           string
	Destination            *Destination
	DataPoint              time.Time
	TimeFrameStartAt       time.Time
	TimeFrameEndAt         time.Time
	UpTime                 time.Duration
	FailureTime            time.Duration
	ErrorBudgetSize        time.Duration
	ErrorBudget            time.Duration
	ErrorBudgetConsumption time.Duration
}

func NewReport(definitionID string, destination *Destination, cursorAt time.Time, timeFrame time.Duration, errorBudgetSize float64) *Report {
	report := &Report{
		DefinitionID:     definitionID,
		Destination:      destination,
		DataPoint:        cursorAt,
		TimeFrameStartAt: cursorAt.Add(-timeFrame),
		TimeFrameEndAt:   cursorAt.Add(-time.Nanosecond),
		ErrorBudgetSize:  time.Duration(errorBudgetSize * float64(timeFrame)).Truncate(time.Minute),
	}
	return report
}

func NewReports(definitionID string, destination *Destination, errorBudgetSize float64, timeFrame time.Duration, reliability Reliabilities) []*Report {
	if reliability.Len() == 0 {
		return make([]*Report, 0)
	}
	n := int(timeFrame / reliability.TimeFrame())
	numReports := reliability.Len() - n + 1
	reports := make([]*Report, 0, numReports)

	for i := 0; i < numReports; i++ {
		report := NewReport(
			definitionID,
			destination,
			reliability.CursorAt(i),
			timeFrame,
			errorBudgetSize,
		)
		report.SetTime(reliability.CalcTime(i, n))
		reports = append(reports, report)
	}

	return reports
}

func (r *Report) SetTime(upTime time.Duration, failureTime time.Duration, deltaFailureTime time.Duration) {
	r.UpTime = upTime
	r.FailureTime = failureTime
	r.ErrorBudget = (r.ErrorBudgetSize - failureTime).Truncate(time.Minute)
	r.ErrorBudgetConsumption = deltaFailureTime.Truncate(time.Minute)
}

// String implements fmt.Stringer
func (r *Report) String() string {
	return fmt.Sprintf(
		"error budget report[id=`%s`,data_point=`%s`]: size=%0.4f[min], remaining=%0.4f[min](%0.1f%%), consumption=%0.4f[min](%0.1f%%)",
		r.DefinitionID, r.DataPoint.Format(time.RFC3339),
		r.ErrorBudgetSize.Minutes(),
		r.ErrorBudget.Minutes(), r.ErrorBudgetUsageRate()*100.0,
		r.ErrorBudgetConsumption.Minutes(), r.ErrorBudgetConsumptionRate()*100.0,
	)
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

func (r *Report) GetDestinationMetricValue(metricType DestinationMetricType) float64 {
	switch metricType {
	case ErrorBudget:
		return r.ErrorBudget.Minutes()
	case ErrorBudgetRemainingPercentage:
		return (1.0 - r.ErrorBudgetUsageRate()) * 100.0
	case ErrorBudgetPercentage:
		return r.ErrorBudgetUsageRate() * 100.0
	case ErrorBudgetConsumption:
		return r.ErrorBudgetConsumption.Minutes()
	case ErrorBudgetConsumptionPercentage:
		return r.ErrorBudgetConsumptionRate() * 100.0
	case UpTime:
		return r.UpTime.Minutes()
	case FailureTime:
		return r.FailureTime.Minutes()
	}
	panic(fmt.Sprintf("unknown metric type %v", metricType))
}
