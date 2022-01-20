package shimesaba

import (
	"errors"
	"log"
	"sort"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
)

// Reliability represents a group of values related to reliability per tumbling window.
type Reliability struct {
	cursorAt      time.Time
	timeFrame     time.Duration
	isNoViolation IsNoViolationCollection
	upTime        time.Duration
	failureTime   time.Duration
}

type IsNoViolationCollection map[time.Time]bool

func (c IsNoViolationCollection) IsUp(t time.Time) bool {
	if isUp, ok := c[t]; ok && !isUp {
		return false
	}
	return true
}

func (c IsNoViolationCollection) NewReliabilities(timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, error) {
	startAt = startAt.Truncate(timeFrame)
	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	reliabilitySlice := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		reliabilitySlice = append(reliabilitySlice, NewReliability(cursorAt, timeFrame, c))
	}
	return NewReliabilities(reliabilitySlice)
}

func NewReliability(cursorAt time.Time, timeFrame time.Duration, isNoViolation IsNoViolationCollection) *Reliability {
	cursorAt = cursorAt.Truncate(timeFrame).Add(timeFrame).UTC()
	r := &Reliability{
		cursorAt:      cursorAt,
		timeFrame:     timeFrame,
		isNoViolation: isNoViolation,
	}
	r = r.Clone()
	r.calc()
	return r
}

func (r *Reliability) Clone() *Reliability {
	cloned := &Reliability{
		cursorAt:    r.cursorAt,
		timeFrame:   r.timeFrame,
		upTime:      r.upTime,
		failureTime: r.failureTime,
	}
	iter := timeutils.NewIterator(r.TimeFrameStartAt(), r.TimeFrameEndAt(), time.Minute)
	clonedIsNoViolation := make(IsNoViolationCollection, r.timeFrame/time.Minute)
	for iter.HasNext() {
		t, _ := iter.Next()
		clonedIsNoViolation[t] = r.isNoViolation.IsUp(t)
	}
	cloned.isNoViolation = clonedIsNoViolation
	return cloned
}

func (r *Reliability) calc() {
	iter := timeutils.NewIterator(r.TimeFrameStartAt(), r.TimeFrameEndAt(), time.Minute)
	var upTime, failureTime time.Duration
	for iter.HasNext() {
		t, _ := iter.Next()
		if r.isNoViolation.IsUp(t) {
			upTime += time.Minute
		} else {
			failureTime += time.Minute
		}
	}
	r.upTime = upTime
	r.failureTime = failureTime
}

//CursorAt is a representative value of the time shown by the tumbling window
func (r *Reliability) CursorAt() time.Time {
	return r.cursorAt
}

//TimeFrame is the size of the tumbling window
func (r *Reliability) TimeFrame() time.Duration {
	return r.timeFrame
}

//TimeFrameStartAt is the start time of the tumbling window
func (r *Reliability) TimeFrameStartAt() time.Time {
	return r.cursorAt.Add(-r.timeFrame)
}

//TimeFrameEndAt is the end time of the tumbling window
func (r *Reliability) TimeFrameEndAt() time.Time {
	return r.cursorAt.Add(-time.Nanosecond)
}

//UpTime is the uptime that can guarantee reliability.
func (r *Reliability) UpTime() time.Duration {
	return r.upTime
}

//FailureTime is the time when reliability could not be ensured, i.e. SLO was violated
func (r *Reliability) FailureTime() time.Duration {
	return r.failureTime
}

//Merge must be the same tumbling window
func (r *Reliability) Merge(other *Reliability) (*Reliability, error) {
	if r.cursorAt != other.cursorAt {
		return r, errors.New("mismatch cursorAt")
	}
	if r.timeFrame != other.timeFrame {
		return r, errors.New("mismatch timeFrame")
	}
	cloned := r.Clone()
	for t, isUp2 := range other.isNoViolation {
		cloned.isNoViolation[t] = r.isNoViolation.IsUp(t) && isUp2
	}
	cloned.calc()
	return cloned, nil
}

// Reliabilities is sortable
type Reliabilities []*Reliability

func NewReliabilities(s []*Reliability) (Reliabilities, error) {
	c := Reliabilities(s)
	sort.Sort(c)
	if c.Len() == 0 {
		return c, nil
	}
	timeFrame := c[0].TimeFrame()
	cursorAt := time.Unix(0, 0)
	for _, r := range c {
		if r.CursorAt() == cursorAt {
			return nil, errors.New("duplicate cursorAt")
		}
		cursorAt = r.CursorAt()
		if r.TimeFrame() != timeFrame {
			return nil, errors.New("multiple timeFrame")
		}
	}
	return c, nil
}

func (c Reliabilities) Len() int           { return len(c) }
func (c Reliabilities) Less(i, j int) bool { return c[i].CursorAt().After(c[j].CursorAt()) }
func (c Reliabilities) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func (c Reliabilities) Clone() Reliabilities {
	cloned := make(Reliabilities, 0, len(c))
	for _, r := range c {
		cloned = append(cloned, r.Clone())
	}
	sort.Sort(cloned)
	return cloned
}

func (c Reliabilities) CalcTime(cursor, n int) (upTime, failureTime, deltaFailureTime time.Duration) {
	deltaFailureTime = c[cursor].FailureTime()
	i := cursor
	for ; i < cursor+n && i < c.Len(); i++ {
		upTime += c[i].UpTime()
		failureTime += c[i].FailureTime()
	}
	log.Printf("[debug] CalcTime[%s~%s] = (%s, %s, %s)",
		c[cursor].TimeFrameStartAt(),
		c[i-1].TimeFrameEndAt(),
		upTime,
		failureTime,
		deltaFailureTime,
	)
	return
}

//TimeFrame is the size of the tumbling window
func (c Reliabilities) TimeFrame() time.Duration {
	if c.Len() == 0 {
		return 0
	}
	return c[0].TimeFrame()
}

//CursorAt is a representative value of the time shown by the tumbling window
func (c Reliabilities) CursorAt(i int) time.Time {
	if c.Len() == 0 {
		return time.Unix(0, 0)
	}
	return c[i].cursorAt
}

//Merge two collection
func (c Reliabilities) Merge(other Reliabilities) (Reliabilities, error) {
	return c.MergeInRange(other, time.Unix(0, 0).UTC(), time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC))
}

func (c Reliabilities) MergeInRange(other Reliabilities, startAt, endAt time.Time) (Reliabilities, error) {
	if len(other) == 0 {
		return c.Clone(), nil
	}
	if len(c) == 0 {
		return other.Clone(), nil
	}
	reliabilityByCursorAt := make(map[time.Time]*Reliability, len(c))
	for _, r := range c {
		if r.TimeFrameStartAt().Before(startAt) {
			continue
		}
		if r.TimeFrameStartAt().After(endAt) {
			continue
		}
		reliabilityByCursorAt[r.CursorAt()] = r.Clone()
	}
	for _, r := range other {
		if r.TimeFrameStartAt().Before(startAt) {
			continue
		}
		if r.TimeFrameStartAt().After(endAt) {
			continue
		}
		cursorAt := r.CursorAt()
		if base, ok := reliabilityByCursorAt[cursorAt]; ok {
			var err error
			reliabilityByCursorAt[cursorAt], err = base.Merge(r)
			if err != nil {
				return nil, err
			}
		} else {
			reliabilityByCursorAt[cursorAt] = r
		}
	}
	merged := make([]*Reliability, 0, len(reliabilityByCursorAt))
	for _, r := range reliabilityByCursorAt {
		merged = append(merged, r)
	}
	return NewReliabilities(merged)
}
