package shimesaba

import "log"

type DestinationMetricType int

//go:generate go install github.com/dmarkham/enumer
//go:generate enumer -type=DestinationMetricType -yaml -linecomment -transform=snake -output destination_metric_type_enumer.go

const (
	ErrorBudget DestinationMetricType = iota
	ErrorBudgetRemainingPercentage
	ErrorBudgetPercentage
	ErrorBudgetConsumption
	ErrorBudgetConsumptionPercentage
	UpTime //uptime
	FailureTime
)

func (t DestinationMetricType) DefaultEnabled() bool {
	switch t {
	case UpTime, FailureTime:
		log.Printf("[warn] In the near future the default value of `enabled` for `%s` metrics will be false, please specify explicitly in config", t.String())
		return true
	default:
		return true
	}
}
