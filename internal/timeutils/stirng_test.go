package timeutils_test

import (
	"testing"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
	"github.com/stretchr/testify/require"
)

func TestDurationString(t *testing.T) {
	cases := []struct {
		d        time.Duration
		expected string
	}{
		{
			expected: "1m",
			d:        time.Minute,
		},
		{
			expected: "1h1m",
			d:        time.Hour + time.Minute,
		},
		{
			expected: "1d",
			d:        24 * time.Hour,
		},
		{
			expected: "1d1m3s",
			d:        24*time.Hour + time.Minute + 3*time.Second,
		},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			actual := timeutils.DurationString(c.d)
			require.EqualValues(t, c.expected, actual)
		})
	}
}
