package shimesaba_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Songmu/flextime"
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
	restore := flextime.Fix(time.Date(2021, 10, 01, 0, 22, 0, 0, time.UTC))
	defer restore()
	alerts := shimesaba.Alerts{
		{
			MonitorID: "hogera",
			OpenedAt:  time.Date(2021, 10, 1, 0, 3, 0, 0, time.UTC),
			ClosedAt:  ptrTime(time.Date(2021, 10, 1, 0, 9, 0, 0, time.UTC)),
		},
		{
			MonitorID: "hogera",
			OpenedAt:  time.Date(2021, time.October, 1, 0, 15, 0, 0, time.UTC),
			ClosedAt:  nil,
		},
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
		{
			defCfg: &shimesaba.DefinitionConfig{
				ID:                "alert_and_metric_mixing",
				TimeFrame:         "10m",
				CalculateInterval: "5m",
				ErrorBudgetSize:   0.3,
				Objectives: []*shimesaba.ObjectiveConfig{
					{
						Expr: "rate(error_count, request_count) <= 0.5",
					},
					{
						Alert: &shimesaba.AlertObjectiveConfig{
							MonitorID: "hogera",
						},
					},
				},
			},
			expected: []*shimesaba.Report{
				{
					DefinitionID:           "alert_and_metric_mixing",
					DataPoint:              time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 9, 59, 999999999, time.UTC),
					UpTime:                 4 * time.Minute,
					FailureTime:            6 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -3 * time.Minute,
					ErrorBudgetConsumption: 4 * time.Minute,
				},
				{
					DefinitionID:           "alert_and_metric_mixing",
					DataPoint:              time.Date(2021, 10, 01, 0, 15, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 5, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 14, 59, 999999999, time.UTC),
					UpTime:                 6 * time.Minute,
					FailureTime:            4 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -1 * time.Minute,
					ErrorBudgetConsumption: 0 * time.Minute,
				},
				{
					DefinitionID:           "alert_and_metric_mixing",
					DataPoint:              time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
					TimeFrameStartAt:       time.Date(2021, 10, 01, 0, 10, 0, 0, time.UTC),
					TimeFrameEndAt:         time.Date(2021, 10, 01, 0, 19, 59, 999999999, time.UTC),
					UpTime:                 5 * time.Minute,
					FailureTime:            5 * time.Minute,
					ErrorBudgetSize:        3 * time.Minute,
					ErrorBudget:            -2 * time.Minute,
					ErrorBudgetConsumption: 5 * time.Minute,
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
			actual, err := def.CreateReports(context.Background(), metrics, alerts,
				time.Date(2021, 10, 01, 0, 0, 0, 0, time.UTC),
				time.Date(2021, 10, 01, 0, 20, 0, 0, time.UTC),
			)
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
