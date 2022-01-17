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
		shimesaba.NewAlert(
			&shimesaba.Monitor{
				ID: "hogera",
			},
			time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC)),
		),
		shimesaba.NewAlert(
			&shimesaba.Monitor{
				ID: "fugara",
			},
			time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC)),
		),
		shimesaba.NewAlert(
			&shimesaba.Monitor{
				ID: "fugara",
			},
			time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC)),
		),
	}
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), alerts.EndAt())
	alerts = append(alerts, shimesaba.NewAlert(
		&shimesaba.Monitor{
			ID: "hogera",
		},
		time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
		nil,
	))
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC), alerts.EndAt())
}
