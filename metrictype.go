package shimesaba

//go:generate go run github.com/alvaroloes/enumer -type=MetricType -text -json -linecomment
//MackerelでのMetricの種類
type MetricType int

const (
	HostMetric    MetricType = iota + 1 //host
	ServiceMetric                       //service
)
