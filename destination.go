package shimesaba

import "fmt"

type Destination struct {
	ServiceName       string
	MetricPrefix      string
	MetricSuffix      string
	MetricTypeNames   map[DestinationMetricType]string
	MetricTypeEnabled map[DestinationMetricType]bool
}

func NewDestination(cfg *DestinationConfig) *Destination {
	ret := &Destination{
		ServiceName:       cfg.ServiceName,
		MetricPrefix:      cfg.MetricPrefix,
		MetricSuffix:      cfg.MetricSuffix,
		MetricTypeNames:   make(map[DestinationMetricType]string),
		MetricTypeEnabled: make(map[DestinationMetricType]bool),
	}
	if cfg.Metrics == nil {
		return ret
	}
	for _, metricType := range DestinationMetricTypeValues() {
		if metricCfg, ok := cfg.Metrics[metricType.ID()]; ok {
			ret.MetricTypeNames[metricType] = metricCfg.MetricTypeName
			if metricCfg.Enabled == nil {
				ret.MetricTypeEnabled[metricType] = metricType.DefaultEnabled()
			} else {
				ret.MetricTypeEnabled[metricType] = *metricCfg.Enabled
			}
		}
	}
	return ret
}

func (d *Destination) MetricName(metricType DestinationMetricType) string {
	if d.MetricTypeNames == nil {
		return fmt.Sprintf("%s.%s.%s", d.MetricPrefix, metricType.DefaultTypeName(), d.MetricSuffix)
	}
	if name, ok := d.MetricTypeNames[metricType]; ok {
		return fmt.Sprintf("%s.%s.%s", d.MetricPrefix, name, d.MetricSuffix)
	}
	return fmt.Sprintf("%s.%s.%s", d.MetricPrefix, metricType.DefaultTypeName(), d.MetricSuffix)
}

func (d *Destination) MetricEnabled(metricType DestinationMetricType) bool {
	if d.MetricTypeEnabled == nil {
		return true
	}
	if enabled, ok := d.MetricTypeEnabled[metricType]; ok {
		return enabled
	}
	return false
}
