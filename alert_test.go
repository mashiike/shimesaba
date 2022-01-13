package shimesaba_test

import (
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestAlerts(t *testing.T) {
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
	}
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), alerts.EndAt())
	alerts = append(alerts, &shimesaba.Alert{
		MonitorID: "hogera",
		OpenedAt:  time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
		ClosedAt:  nil,
	})
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC), alerts.EndAt())
}
