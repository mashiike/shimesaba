package shimesaba_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestDefinition(t *testing.T) {
	metricsConfigs := []struct {
		cfg          *shimesaba.MetricConfig
		appendValues []timeValueTuple
	}{
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy3",
				AggregationInterval: 1,
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy3.csv"),
		},
	}
	metrics := make(shimesaba.Metrics)
	for _, cfg := range metricsConfigs {
		metric := shimesaba.NewMetric(cfg.cfg)
		for _, tv := range cfg.appendValues {
			metric.AppendValue(tv.Time, tv.Value)
		}
		metrics.Set(metric)
	}
	cases := []struct {
		defCfg   *shimesaba.DefinitionConfig
		expected []*shimesaba.Report
	}{
		{
			defCfg: &shimesaba.DefinitionConfig{
				ID:                "test1",
				TimeFrame:         10,
				CalculateInterval: 5,
				ErrorBudgetSize:   0.3,
				Objectives: []*shimesaba.ObjectiveConfig{
					{
						Expr: "dummy3 < 1.0",
					},
				},
			},
			expected: []*shimesaba.Report{
				{
					DefinitionID:           "test1",
					DataPoint:              time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 9, 59, 999999999, time.UTC),
					UpTime:                 8 * time.Minute,
					FailureTime:            2 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            1 * time.Minute,
					ErrorBudgetConsumption: 0,
				},
				{
					DefinitionID:           "test1",
					DataPoint:              time.Date(2021, 10, 01, 0, 15, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 5, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 14, 59, 999999999, time.UTC),
					UpTime:                 6 * time.Minute,
					FailureTime:            4 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -1 * time.Minute,
					ErrorBudgetConsumption: 4 * time.Minute,
				},
				{
					DefinitionID:           "test1",
					DataPoint:              time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 19, 59, 999999999, time.UTC),
					UpTime:                 5 * time.Minute,
					FailureTime:            5 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -2 * time.Minute,
					ErrorBudgetConsumption: 1 * time.Minute,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.defCfg.ID, func(t *testing.T) {
			var buf bytes.Buffer
			logger.Setup(&buf, "debug")
			defer func() {
				t.Log(buf.String())
				logger.Setup(os.Stderr, "info")
			}()
			def, err := shimesaba.NewDefinition(c.defCfg)
			require.NoError(t, err)
			actual, err := def.CreateRepoorts(context.Background(), metrics)
			require.NoError(t, err)
			t.Log("actual:")
			for _, a := range actual {
				bs, _ := json.MarshalIndent(a, "", "  ")
				t.Log(string(bs))
			}
			t.Log("excepted:")
			for _, e := range c.expected {
				bs, _ := json.MarshalIndent(e, "", "  ")
				t.Log(string(bs))
			}
			require.EqualValues(t, c.expected, actual)
		})
	}

}

func TestReport(t *testing.T) {
	cases := []struct {
		casename                           string
		report                             *shimesaba.Report
		expectedErrorBudgetUsageRate       float64
		exceptedErrorBudgetConsumptionRate float64
	}{
		{
			casename: "size=100min,budget=99min",
			report: &shimesaba.Report{
				ErrorBudgetSize:        100 * time.Minute,
				ErrorBudget:            99 * time.Minute,
				ErrorBudgetConsumption: time.Minute,
			},
			expectedErrorBudgetUsageRate:       0.01,
			exceptedErrorBudgetConsumptionRate: 0.01,
		},
		{
			casename: "size=100min,budget=-3min",
			report: &shimesaba.Report{
				ErrorBudgetSize:        100 * time.Minute,
				ErrorBudget:            -3 * time.Minute,
				ErrorBudgetConsumption: 99 * time.Minute,
			},
			expectedErrorBudgetUsageRate:       1.03,
			exceptedErrorBudgetConsumptionRate: 0.99,
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
				c.exceptedErrorBudgetConsumptionRate,
				consumptionRate,
				epsilon,
				"consumption rate",
			)
		})
	}
}
