package shimesaba_test

import (
	"context"
	"testing"
	"time"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestRepositoryFetchVirtualAlerts(t *testing.T) {
	client := newMockMackerelClient(t)
	repo := shimesaba.NewRepository(client)

	cases := []struct {
		name        string
		serviceName string
		sloID       string
		startAt     time.Time
		endAt       time.Time
		expected    shimesaba.Alerts
	}{
		{
			name:        "SLO:*",
			serviceName: "shimesaba",
			sloID:       "hoge",
			startAt:     time.Date(2021, 10, 1, 0, 5, 0, 0, time.UTC),
			endAt:       time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC),
			expected: shimesaba.Alerts{
				{
					Reason:   "SLO:*",
					OpenedAt: time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC),
					ClosedAt: ptrTime(time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC)),
				},
			},
		},
		{
			name:        "SLO:availability,quarity ",
			serviceName: "shimesaba",
			sloID:       "availability",
			startAt:     time.Date(2021, 10, 1, 0, 5, 0, 0, time.UTC),
			endAt:       time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC),
			expected: shimesaba.Alerts{
				{
					Reason:   "SLO:*",
					OpenedAt: time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC),
					ClosedAt: ptrTime(time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC)),
				},
				{
					Reason:   "ALB Failures SLO:availability,quarity affected.",
					OpenedAt: time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC),
					ClosedAt: ptrTime(time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC)),
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vAlerts, err := repo.FetchVirtualAlerts(context.Background(), c.serviceName, c.sloID, c.startAt, c.endAt)
			require.NoError(t, err)
			require.EqualValues(t, c.expected, vAlerts)
		})
	}
}
