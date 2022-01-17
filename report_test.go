package shimesaba_test

import (
	"encoding/json"
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

func TestNewReports(t *testing.T) {

	allTimeIsNoViolation := map[time.Time]bool{

		time.Date(2022, 1, 6, 8, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 8, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 8, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 10, 38, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 40, 0, 0, time.UTC): false,
	}
	tumblingWindowTimeFrame := time.Hour
	c, _ := shimesaba.NewReliabilityCollection(
		[]*shimesaba.Reliability{
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 8, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
		},
	)
	actual := shimesaba.NewReports("test", "test", "test", 0.05, 2*time.Hour, c)
	expected := []*shimesaba.Report{
		{
			DefinitionID:           "test",
			ServiceName:            "test",
			MetricPrefix:           "test",
			DataPoint:              time.Date(2022, 1, 6, 11, 0, 0, 0, time.UTC),
			TimeFrameStartAt:       time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC),
			TimeFrameEndAt:         time.Date(2022, 1, 6, 11, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
			ErrorBudgetSize:        6 * time.Minute,
			UpTime:                 (57 + 58) * time.Minute,
			FailureTime:            (3 + 2) * time.Minute,
			ErrorBudget:            1 * time.Minute,
			ErrorBudgetConsumption: 3 * time.Minute,
		},
		{
			DefinitionID:           "test",
			ServiceName:            "test",
			MetricPrefix:           "test",
			DataPoint:              time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC),
			TimeFrameStartAt:       time.Date(2022, 1, 6, 8, 0, 0, 0, time.UTC),
			TimeFrameEndAt:         time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
			ErrorBudgetSize:        6 * time.Minute,
			UpTime:                 (58 + 59) * time.Minute,
			FailureTime:            (2 + 1) * time.Minute,
			ErrorBudget:            3 * time.Minute,
			ErrorBudgetConsumption: 2 * time.Minute,
		},
	}
	for i, a := range actual {
		bs, _ := json.MarshalIndent(a, "", "  ")
		t.Logf("actual[%d]:%s", i, string(bs))
	}
	for i, e := range expected {
		bs, _ := json.MarshalIndent(e, "", "  ")
		t.Logf("expected[%d]:%s", i, string(bs))
	}
	require.EqualValues(t, expected, actual)
}
