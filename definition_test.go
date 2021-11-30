package shimesaba_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/mashiike/evaluator"
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
				AggregationInterval: "1",
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy3.csv"),
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy3",
				AggregationInterval: "1m",
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy3.csv"),
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "request_count",
				AggregationInterval: "1m",
				AggregationMethod:   "sum",
			},
			appendValues: loadTupleFromCSV(t, "testdata/request_count.csv"),
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "error_count",
				AggregationInterval: "1m",
				AggregationMethod:   "sum",
				InterpolatedValue:   Float64(0.0),
			},
			appendValues: loadTupleFromCSV(t, "testdata/error_count.csv"),
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
				TimeFrame:         "10m",
				CalculateInterval: "5m",
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
		{
			defCfg: &shimesaba.DefinitionConfig{
				ID:                "error_rate",
				TimeFrame:         "10m",
				CalculateInterval: "5m",
				ErrorBudgetSize:   0.3,
				Objectives: []*shimesaba.ObjectiveConfig{
					{
						Expr: "rate(error_count, request_count) <= 0.5",
					},
				},
			},
			expected: []*shimesaba.Report{
				{
					DefinitionID:           "error_rate",
					DataPoint:              time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 9, 59, 999999999, time.UTC),
					UpTime:                 10 * time.Minute,
					FailureTime:            0 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            3 * time.Minute,
					ErrorBudgetConsumption: 0,
				},
				{
					DefinitionID:           "error_rate",
					DataPoint:              time.Date(2021, 10, 01, 0, 15, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 5, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 14, 59, 999999999, time.UTC),
					UpTime:                 10 * time.Minute,
					FailureTime:            0 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            3 * time.Minute,
					ErrorBudgetConsumption: 0 * time.Minute,
				},
				{
					DefinitionID:           "error_rate",
					DataPoint:              time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 19, 59, 999999999, time.UTC),
					UpTime:                 9 * time.Minute,
					FailureTime:            1 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            2 * time.Minute,
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
			actual, err := def.CreateReports(context.Background(), metrics)
			require.NoError(t, err)
			t.Log("actual:")
			for _, a := range actual {
				bs, _ := json.MarshalIndent(a, "", "  ")
				t.Log(string(bs))
			}
			t.Log("expected:")
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

func TestMetricComparate(t *testing.T) {
	metricsConfigs := []struct {
		cfg          *shimesaba.MetricConfig
		appendValues []timeValueTuple
	}{
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy1",
				AggregationInterval: "1m",
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy1.csv"),
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy2",
				AggregationInterval: "1m",
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy2.csv"),
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
		expr     string
		expected map[time.Time]bool
	}{
		{
			expr: "dummy1 <= 0.9",
			expected: map[time.Time]bool{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): true,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): false,
			},
		},
		{
			expr: "dummy2 - dummy1 >= 1.0",
			expected: map[time.Time]bool{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): true,
			},
		},
		{
			expr: "rate(dummy1,dummy1) == 1.0",
			expected: map[time.Time]bool{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): true,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): true,
			},
		},
		{
			expr:     "dummy1/(dummy1-dummy1) > 1.0",
			expected: map[time.Time]bool{},
		},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			e, err := evaluator.New(c.expr)
			require.NoError(t, err)
			comparator, ok := e.AsComparator()
			require.EqualValues(t, true, ok)
			actual := shimesaba.MetricsComparate(comparator, metrics, metrics.StartAt(), metrics.EndAt())
			require.EqualValues(t, c.expected, actual)
		})
	}

}
