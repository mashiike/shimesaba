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
			shimesaba.NewMonitor(
				"hogera",
				"hogera.example.com",
				"external",
			),
			time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC)),
		),
		shimesaba.NewAlert(
			shimesaba.NewMonitor(
				"fugara",
				"fugara.example.com",
				"external",
			),
			time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC)),
		),
		shimesaba.NewAlert(
			shimesaba.NewMonitor(
				"fugara",
				"fugara.example.com",
				"external",
			),
			time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
			ptrTime(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC)),
		),
		shimesaba.NewVirtualAlert(
			"slo:hoge",
			time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC),
		),
	}
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), alerts.EndAt())
	alerts = append(alerts, shimesaba.NewAlert(
		shimesaba.NewMonitor(
			"hogera",
			"hogera.example.com",
			"external",
		),
		time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
		nil,
	))
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), alerts.StartAt())
	require.EqualValues(t, time.Date(2021, time.October, 1, 0, 6, 0, 0, time.UTC), alerts.EndAt())
}

func TestAlertEvaluateReliabilities(t *testing.T) {
	restore := flextime.Fix(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC))
	defer restore()
	cases := []struct {
		alert             *shimesaba.Alert
		timeFrame         time.Duration
		expectedGenerator func() shimesaba.Reliabilities
	}{
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"external",
				),
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
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"external",
				),
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
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"external",
				),
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
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"external",
				),
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC)),
			).WithReason("downtime:2m"),
			timeFrame: 5 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{
					time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 4, 0, 0, time.UTC): false,
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
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"external",
				),
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC)),
			).WithReason("downtime:0m"),
			timeFrame: 5 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{}
				expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
				})
				return expected
			},
		},
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor(
					"fugara",
					"fugara.example.com",
					"service",
				).WithEvaluator(func(hostID string, timeFrame time.Duration, startAt, endAt time.Time) (shimesaba.Reliabilities, bool) {
					isNoViolation := map[time.Time]bool{
						time.Date(2021, time.September, 30, 23, 58, 0, 0, time.UTC): false,
						time.Date(2021, time.September, 30, 23, 59, 0, 0, time.UTC): false,
						time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC):      false,
						time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC):      false,
						time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC):      false,
					}
					reliabilities, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
						shimesaba.NewReliability(time.Date(2021, time.September, 30, 23, 55, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
						shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
						shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
					})
					return reliabilities, true
				}),
				time.Date(2021, time.October, 1, 0, 3, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 8, 0, 0, time.UTC)),
			),
			timeFrame: 5 * time.Minute,
			expectedGenerator: func() shimesaba.Reliabilities {
				isNoViolation := map[time.Time]bool{
					time.Date(2021, time.September, 30, 23, 58, 0, 0, time.UTC): false,
					time.Date(2021, time.September, 30, 23, 59, 0, 0, time.UTC): false,
					time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC):      false,
					time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC):      false,
					time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC):      false,
				}
				expected, _ := shimesaba.NewReliabilities([]*shimesaba.Reliability{
					shimesaba.NewReliability(time.Date(2021, time.September, 30, 23, 55, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
					shimesaba.NewReliability(time.Date(2021, time.October, 1, 0, 5, 0, 0, time.UTC), 5*time.Minute, isNoViolation),
				})
				return expected
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			actual, err := c.alert.EvaluateReliabilities(c.timeFrame, true)
			require.NoError(t, err)
			require.EqualValues(t, c.expectedGenerator(), actual)
		})
	}
}

func TestAlertCorrectionTime(t *testing.T) {
	cases := []struct {
		alert      *shimesaba.Alert
		exceptedOk bool
		excepted   time.Duration
	}{
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor("test", "test", "external"),
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC)),
			).WithReason("Actual downtime:3m 5xx during this time, 5 cases."),
			exceptedOk: true,
			excepted:   3 * time.Minute,
		},
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor("test", "test", "external"),
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC)),
			).WithReason("Actual downtime:3m, 5xx during this time, 5 cases."),
			exceptedOk: false,
		},
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor("test", "test", "external"),
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC)),
			).WithReason("downtime:8m"),
			exceptedOk: true,
			excepted:   8 * time.Minute,
		},
		{
			alert: shimesaba.NewAlert(
				shimesaba.NewMonitor("test", "test", "external"),
				time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
				ptrTime(time.Date(2021, time.October, 1, 0, 7, 0, 0, time.UTC)),
			),
			exceptedOk: false,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			actual, ok := c.alert.CorrectionTime()
			require.EqualValues(t, c.exceptedOk, ok)
			require.EqualValues(t, c.excepted, actual)
		})
	}
}
