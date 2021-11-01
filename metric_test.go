package shimesaba_test

import (
	"testing"
	"time"

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
