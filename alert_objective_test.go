package shimesaba_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestAlertObjective(t *testing.T) {
	restore := flextime.Fix(time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC))
	defer restore()
	alerts := shimesaba.Alerts{
		{
			MonitorID: "hogera",
			OpenedAt:  time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
			ClosedAt:  ptrTime(time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC)),
		},
		{
			MonitorID: "fugara",
			OpenedAt:  time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			ClosedAt:  ptrTime(time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC)),
		},
		{
			MonitorID: "fugara",
			OpenedAt:  time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
			ClosedAt:  ptrTime(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC)),
		},
		{
			MonitorID: "hogera",
			OpenedAt:  time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC),
			ClosedAt:  nil,
		},
	}
	cases := []struct {
		cfg      *shimesaba.AlertObjectiveConfig
		expected map[time.Time]bool
	}{
		{
			cfg: &shimesaba.AlertObjectiveConfig{
				MonitorID: "hogera",
			},
			expected: map[time.Time]bool{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC): false,
			},
		},
		{
			cfg: &shimesaba.AlertObjectiveConfig{
				MonitorID: "fugara",
			},
			expected: map[time.Time]bool{
				time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC): true,
				time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC): true,
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC): false,
				time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC): true,
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			obj := shimesaba.NewAlertObjective(c.cfg)
			actual, err := obj.NewReliabilityCollection(time.Minute, alerts)
			require.NoError(t, err)
			expected, _ := shimesaba.NewReliabilityCollection([]*shimesaba.Reliability{
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC), time.Minute, c.expected),
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC), time.Minute, c.expected),
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC), time.Minute, c.expected),
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC), time.Minute, c.expected),
				shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), time.Minute, c.expected),
			})
			require.EqualValues(t, expected, actual)
		})
	}

}

func ptrTime(t time.Time) *time.Time {
	return &t
}
