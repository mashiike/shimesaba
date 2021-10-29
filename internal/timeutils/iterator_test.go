package timeutils_test

import (
	"fmt"
	"testing"
	"time"

	timeutil "github.com/mashiike/shimesaba/internal/timeutils"
	"github.com/stretchr/testify/require"
)

type timeTuple struct {
	StartAt, EndAt time.Time
}

func (t timeTuple) GoString() string {
	return fmt.Sprintf("[%s ~ %s]", t.StartAt, t.EndAt)
}

func TestIterator(t *testing.T) {
	cases := []struct {
		startAt          time.Time
		endAt            time.Time
		tick             time.Duration
		enableOverWindow bool
		expected         []timeTuple
	}{
		{
			startAt: time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
			endAt:   time.Date(2021, time.October, 1, 5, 0, 0, 0, time.UTC),
			tick:    time.Hour,
			expected: []timeTuple{

				{
					StartAt: time.Date(2021, time.October, 1, 0, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 59, 59, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 1, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 1, 59, 59, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 2, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 2, 59, 59, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 3, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 3, 59, 59, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 4, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 4, 59, 59, 999999999, time.UTC),
				},
			},
		},
		{
			startAt: time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
			endAt:   time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			tick:    25 * time.Second,
			expected: []timeTuple{

				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 24, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 25, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 49, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 50, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 59, 999999999, time.UTC),
				},
			},
		},
		{
			startAt:          time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
			endAt:            time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			enableOverWindow: true,
			tick:             25 * time.Second,
			expected: []timeTuple{

				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 24, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 25, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 49, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 50, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 2, 14, 999999999, time.UTC),
				},
			},
		},
		{
			startAt:          time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
			endAt:            time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
			enableOverWindow: true,
			tick:             30 * time.Second,
			expected: []timeTuple{

				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 29, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 1, 30, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 1, 59, 999999999, time.UTC),
				},
				{
					StartAt: time.Date(2021, time.October, 1, 0, 2, 0, 0, time.UTC),
					EndAt:   time.Date(2021, time.October, 1, 0, 2, 29, 999999999, time.UTC),
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s~%s[tick=%s,over=%v]", c.startAt, c.endAt, c.tick, c.enableOverWindow), func(t *testing.T) {
			iter := timeutil.NewIterator(
				c.startAt,
				c.endAt,
				c.tick,
			)
			iter.SetEnableOverWindow(c.enableOverWindow)
			actual := make([]timeTuple, 0)
			for iter.HasNext() {
				startAt, endAt := iter.Next()
				actual = append(actual, timeTuple{
					StartAt: startAt,
					EndAt:   endAt,
				})
			}
			require.EqualValues(t, c.expected, actual)
		})
	}

}
