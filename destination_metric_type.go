package shimesaba

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

func (t DestinationMetricType) ID() string {
	return t.String()
}

func (t DestinationMetricType) DefaultTypeName() string {
	return t.String()
}

func (t DestinationMetricType) DefaultEnabled() bool {
	switch t {
	case UpTime, FailureTime:
		return false
	default:
		return true
	}
}
