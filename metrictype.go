package shimesaba

//go:generate go run github.com/alvaroloes/enumer -type=MetricType -text -json -linecomment
// MetricType is the type of metric in Mackerel
type MetricType int

// Reserved value
const (
	HostMetric    MetricType = iota + 1 //host
	ServiceMetric                       //service
)
