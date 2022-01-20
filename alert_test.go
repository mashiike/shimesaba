package shimesaba_test

import (
	"fmt"
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

func TestAlertCalculateReliabilities(t *testing.T) {
	restore := flextime.Fix(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC))
	defer restore()
	cases := []struct {
		alert             *shimesaba.Alert
		timeFrame         time.Duration
		expectedGenerator func() shimesaba.Reliabilities
	}{
		{
			alert: shimesaba.NewAlert(
				&shimesaba.Monitor{
					ID: "fugara",
				},
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC)),
			),
			timeFrame: 5 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{
					time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC): false,
				}
				expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
				})
				return expected
			},
		},
		{
			alert: shimesaba.NewAlert(
				&shimesaba.Monitor{
					ID: "fugara",
				},
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC)),
			),
			timeFrame: 5 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{
					time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC): false,
				}
				expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
				})
				return expected
			},
		},
		{
			alert: shimesaba.NewAlert(
				&shimesaba.Monitor{
					ID: "fugara",
				},
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				nil,
			),
			timeFrame: 2 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{
					time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC): true,
					time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC): false,
				}
				expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC), 2*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC), 2*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC), 2*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC), 2*time.Minute, isNoViolation),
				})
				return expected
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			actual, err := c.alert.CalculateReliabilities(c.timeFrame)
			require.NoError(t, err)
			require.EqualValues(t, c.expectedGenerator(), actual)
		})
	}
}
