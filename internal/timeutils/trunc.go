package timeutils

import "time"

func TruncTime(t time.Time, d time.Duration) time.Time {
	if d == 0 {
		return t
	}
	return time.Unix(0, (t.UnixNano()/int64(d))*int64(d)).In(t.Location())
}
