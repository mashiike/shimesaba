package timeutils_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
	"github.com/stretchr/testify/require"
)

func TestTruncTime(t *testing.T) {
	cases := []struct {
		base     time.Time
		d        time.Duration
		expected time.Time
	}{
		{
			base:     time.Date(2021, time.October, 1, 2, 5, 4, 3, time.UTC),
			d:        time.Hour,
			expected: time.Date(2021, time.October, 1, 2, 0, 0, 0, time.UTC),
		},
		{
			base:     time.Date(2021, time.October, 1, 2, 5, 4, 3, time.UTC),
			d:        time.Minute,
			expected: time.Date(2021, time.October, 1, 2, 5, 0, 0, time.UTC),
		},
		{
			base:     time.Date(2021, time.October, 1, 2, 5, 4, 3, time.UTC),
			d:        time.Second,
			expected: time.Date(2021, time.October, 1, 2, 5, 4, 0, time.UTC),
		},
		{
			base:     time.Date(2021, time.October, 1, 2, 5, 4, 123456789, time.UTC),
			d:        time.Millisecond,
			expected: time.Date(2021, time.October, 1, 2, 5, 4, 123000000, time.UTC),
		},
		{
			base:     time.Date(2021, time.October, 1, 2, 5, 4, 123456789, time.UTC),
			d:        0,
			expected: time.Date(2021, time.October, 1, 2, 5, 4, 123456789, time.UTC),
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("TruncTime(%s,%s)", c.base, c.d), func(t *testing.T) {
			actual := timeutils.TruncTime(c.base, c.d)
			require.EqualValues(t, c.expected, actual)
		})
	}
}
