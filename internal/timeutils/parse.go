package timeutils

import (
	"errors"
	"log"
	"strconv"
	"time"
)

func ParseDuration(str string) (time.Duration, error) {
	if d, err := strconv.ParseUint(str, 10, 64); err == nil {
		log.Printf("[warn] Setting an interval without a unit is deprecated. Please write `%s` as` %sm`", str, str)
		return time.Duration(d) * time.Minute, nil
	}

	days, parts := trimDay(str)
	if parts != "" {
		d, err := time.ParseDuration(parts)
		return days + d, err
	}
	if days == 0 {
		return 0, errors.New("invalid format")
	}
	return days, nil
}

func trimDay(str string) (time.Duration, string) {
	var val int64
	for i, c := range str {
		if '0' <= c && c <= '9' {
			v := int64(c - '0')
			val = val*10 + v
			continue
		}
		if c == 'd' {
			return time.Duration(val) * 24 * time.Hour, str[i+1:]
		}
		return 0, str
	}
	return 0, str
}
