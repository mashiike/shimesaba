package timeutils

import "time"

type Iterator struct {
	startAt          time.Time
	endAt            time.Time
	tick             time.Duration
	curAt            time.Time
	enableOverWindow bool
}

func NewIterator(startAt, endAt time.Time, tick time.Duration) *Iterator {
	return &Iterator{
		startAt:          startAt,
		endAt:            endAt,
		curAt:            startAt,
		tick:             tick,
		enableOverWindow: false,
	}
}

func (iter *Iterator) remaining() time.Duration {
	return iter.endAt.Sub(iter.curAt)
}

func (iter *Iterator) nextTick() time.Duration {
	if remaining := iter.remaining(); !iter.enableOverWindow && remaining < iter.tick {
		return remaining
	}
	return iter.tick
}

func (iter *Iterator) HasNext() bool {
	remaining := iter.remaining()
	if remaining > 0 {
		return true
	}
	return iter.enableOverWindow && remaining == 0
}

func (iter *Iterator) Next() (time.Time, time.Time) {
	nextTick := iter.nextTick()
	curStartAt := iter.curAt
	curEndAt := iter.curAt.Add(nextTick)
	iter.curAt = curEndAt
	return curStartAt, curEndAt.Add(-time.Nanosecond)
}

func (iter *Iterator) SetEnableOverWindow(flag bool) {
	iter.enableOverWindow = flag
}
