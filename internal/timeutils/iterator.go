package timeutils

import "time"

// Iterator achieves a loop for a specified period of time
type Iterator struct {
	startAt          time.Time
	endAt            time.Time
	tick             time.Duration
	curAt            time.Time
	enableOverWindow bool
}

//NewIterator create Iterator
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

// HasNext is a loop continuation condition
func (iter *Iterator) HasNext() bool {
	remaining := iter.remaining()
	if remaining > 0 {
		return true
	}
	return iter.enableOverWindow && remaining == 0
}

// Next returns the current rolling window and recommends Iterator to the next window
func (iter *Iterator) Next() (time.Time, time.Time) {
	nextTick := iter.nextTick()
	curStartAt := iter.curAt
	curEndAt := iter.curAt.Add(nextTick)
	iter.curAt = curEndAt
	return curStartAt, curEndAt.Add(-time.Nanosecond)
}

//SetEnableOverWindow affects the Iterator's end condition and specifies whether to allow the end time of the rolling window to exceed the end time of the specified period.
func (iter *Iterator) SetEnableOverWindow(flag bool) {
	iter.enableOverWindow = flag
}
