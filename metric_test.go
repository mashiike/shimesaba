package shimesaba_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mashiike/evaluator"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestMetricGetValue(t *testing.T) {
	appendValues := loadTupleFromCSV(t, "testdata/dummy1.csv")
	expectedStartAt := time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC)
	expectedEndAt := time.Date(2021, time.October, 1, 0, 2, 59, 999999999, time.UTC)
	expectedEndAt5min := time.Date(2021, time.October, 1, 0, 4, 59, 999999999, time.UTC)
	cases := []struct {
		cfg             *shimesaba.MetricConfig
		appendValues    []timeValueTuple
		expected        map[time.Time]float64
		expectedStartAt time.Time
		expectedEndAt   time.Time
	}{
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "max_values",
				AggregationInterval: "1m",
				AggregationMethod:   "max",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 0.9,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): 32.0,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "total_values",
				AggregationInterval: "1m",
				AggregationMethod:   "total",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 2,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): 33.1,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "avg_values",
				AggregationInterval: "1m",
				AggregationMethod:   "avg",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 2.0 / 3.0,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): 33.1 / 3.0,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "max_agg_5min_values",
				AggregationInterval: "5m",
				AggregationMethod:   "max",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 32,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt5min,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "default_agg_5min_values",
				AggregationInterval: "5m",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 32,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt5min,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "hoge_agg_5min_values",
				AggregationInterval: "5m",
				AggregationMethod:   "hoge",
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 32,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt5min,
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "avg_values",
				AggregationInterval: "1m",
				AggregationMethod:   "avg",
				InterpolatedValue:   Float64(0.0),
			},
			appendValues: appendValues,
			expected: map[time.Time]float64{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): 2.0 / 3.0,
				time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC): 0.0,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): 33.1 / 3.0,
			},
			expectedStartAt: expectedStartAt,
			expectedEndAt:   expectedEndAt,
		},
	}
	for _, c := range cases {
		t.Run(c.cfg.ID, func(t *testing.T) {
			metric := shimesaba.NewMetric(c.cfg)
			for _, tv := range c.appendValues {
				err := metric.AppendValue(tv.Time, tv.Value)
				require.NoError(t, err)
			}
			actualStartAt := metric.StartAt()
			require.EqualValues(t, c.expectedStartAt, actualStartAt)
			actualEndAt := metric.EndAt()
			require.EqualValues(t, c.expectedEndAt, actualEndAt)
			actual := metric.GetValues(actualStartAt, actualEndAt)
			require.EqualValues(t, c.expected, actual)
		})
	}

}

func Float64(f float64) *float64 {
	return &f
}

func TestMetricsGetVariables(t *testing.T) {
	metricsConfigs := []struct {
		cfg          *shimesaba.MetricConfig
		appendValues []timeValueTuple
	}{
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "request_count",
				AggregationInterval: "10m",
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
	actual := metrics.GetVariables(metrics.StartAt(), metrics.EndAt())
	expected := map[time.Time]evaluator.Variables{
		time.Date(2021, 9, 30, 23, 50, 0, 0, time.UTC): {
			"error_count":   1,
			"request_count": 220,
		},
		time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC): {
			"error_count":   2,
			"request_count": 1530,
		},
		time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC): {
			"error_count":   103,
			"request_count": 1650,
		},
		time.Date(2021, 10, 1, 0, 20, 0, 0, time.UTC): {
			"error_count":   0,
			"request_count": 330,
		},
	}
	t.Log("actual:")
	actualJSON, _ := json.MarshalIndent(actual, "", "  ")
	t.Log(string(actualJSON))
	t.Log("expected:")
	expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
	t.Log(string(expectedJSON))
	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}
