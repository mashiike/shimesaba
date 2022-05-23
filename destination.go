package shimesaba

import "fmt"

type Destination struct {
	ServiceName  string
	MetricPrefix string
	MetricSuffix string
}

func (d *Destination) ErrorBudgetMetricName() string {
	return fmt.Sprintf("%s.error_budget.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) ErrorBudgetRemainingPercentageMetricName() string {
	return fmt.Sprintf("%s.error_budget_remaining_percentage.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) ErrorBudgetPercentageMetricName() string {
	return fmt.Sprintf("%s.error_budget_percentage.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) ErrorBudgetConsumptionMetricName() string {
	return fmt.Sprintf("%s.error_budget_consumption.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) ErrorBudgetConsumptionPercentageMetricName() string {
	return fmt.Sprintf("%s.error_budget_consumption_percentage.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) UpTimeMetricName() string {
	return fmt.Sprintf("%s.uptime.%s", d.MetricPrefix, d.MetricSuffix)
}

func (d *Destination) FailureMetricName() string {
	return fmt.Sprintf("%s.failure_time.%s", d.MetricPrefix, d.MetricSuffix)
}
