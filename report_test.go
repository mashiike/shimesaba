package shimesaba_test

import (
	"testing"
	"time"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	cases := []struct {
		casename                           string
		report                             *shimesaba.Report
		expectedErrorBudgetUsageRate       float64
		expectedErrorBudgetConsumptionRate float64
	}{
		{
			casename: "size=100min,budget=99min",
			report: &shimesaba.Report{
				ErrorBudgetSize:        100 * time.Minute,
				ErrorBudget:            99 * time.Minute,
				ErrorBudgetConsumption: time.Minute,
			},
			expectedErrorBudgetUsageRate:       0.01,
			expectedErrorBudgetConsumptionRate: 0.01,
		},
		{
			casename: "size=100min,budget=-3min",
			report: &shimesaba.Report{
				ErrorBudgetSize:        100 * time.Minute,
				ErrorBudget:            -3 * time.Minute,
				ErrorBudgetConsumption: 99 * time.Minute,
			},
			expectedErrorBudgetUsageRate:       1.03,
			expectedErrorBudgetConsumptionRate: 0.99,
		},
	}
	epsilon := 0.00001
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			usageRate := c.report.ErrorBudgetUsageRate()
			t.Log(usageRate)
			require.InEpsilon(
				t,
				c.expectedErrorBudgetUsageRate,
				usageRate,
				epsilon,
				"usage rate",
			)
			consumptionRate := c.report.ErrorBudgetConsumptionRate()
			t.Log(consumptionRate)
			require.InEpsilon(
				t,
				c.expectedErrorBudgetConsumptionRate,
				consumptionRate,
				epsilon,
				"consumption rate",
			)
		})
	}
}
