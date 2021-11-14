package timeutils

import (
	"fmt"
	"strings"
	"time"
)

func DurationString(d time.Duration) string {
	days := uint64(d.Truncate(24*time.Hour).Hours() / 24.0)
	remining := d - d.Truncate(24*time.Hour)
	hours := uint64(remining.Truncate(time.Hour).Hours())
	remining = remining - remining.Truncate(time.Hour)
	minutes := uint64(remining.Truncate(time.Minute).Minutes())
	remining = remining - remining.Truncate(time.Minute)
	seconds := uint64(remining.Truncate(time.Second).Seconds())
	return durationString(days, hours, minutes, seconds)
}

func durationString(day, hours, minutes, seconds uint64) string {
	var builder strings.Builder
	if day > 0 {
		fmt.Fprintf(&builder, "%dd", day)
	}
	if hours > 0 {
		fmt.Fprintf(&builder, "%dh", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&builder, "%dm", minutes)
	}
	if seconds > 0 {
		fmt.Fprintf(&builder, "%ds", seconds)
	}
	return builder.String()
}
