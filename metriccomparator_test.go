package shimesaba_test

import (
	"testing"
	"time"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestMetricComparatorEval(t *testing.T) {
	metricsConfigs := []struct {
		cfg          *shimesaba.MetricConfig
		appendValues []timeValueTuple
	}{
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy1",
				AggregationInterval: 1,
				AggregationMethod:   "max",
			},
			appendValues: loadTupleFromCSV(t, "testdata/dummy1.csv"),
		},
		{
			cfg: &shimesaba.MetricConfig{
				ID:                  "dummy2",
				AggregationInterval: 1,
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
			expr:     "rate(dummy1,dummy1-dummy1) > 1.0",
			expected: map[time.Time]bool{},
		},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			mc, err := shimesaba.NewMetricComparator(c.expr)
			require.NoError(t, err)
			actual, err := mc.Eval(metrics, metrics.StartAt(), metrics.EndAt())
			require.NoError(t, err)
			require.EqualValues(t, c.expected, actual)
		})
	}

}
