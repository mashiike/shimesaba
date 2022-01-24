package shimesaba_test

import (
	"testing"
	"time"

	"github.com/mashiike/evaluator"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestExprObjective(t *testing.T) {
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
			obj := shimesaba.NewExprObjective(comparator)
			actual, err := obj.EvaluateReliabilities(
				time.Minute,
				metrics,
				time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			)
			require.NoError(t, err)
			expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC), time.Minute, c.expected),
			})
			require.EqualValues(t, expected, actual)
		})
	}

}
