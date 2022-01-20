package shimesaba

import (
	"fmt"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

type Alert struct {
	Monitor  *Monitor
	OpenedAt time.Time
	ClosedAt *time.Time
}

func NewAlert(monitor *Monitor, openedAt time.Time, closedAt *time.Time) *Alert {
	if closedAt != nil {
		tmp := closedAt.Truncate(time.Minute).UTC()
		closedAt = &tmp
	}
	return &Alert{
		Monitor:  monitor,
		OpenedAt: openedAt.Truncate(time.Minute).UTC(),
		ClosedAt: closedAt,
	}
}

func (alert *Alert) String() string {
	return fmt.Sprintf("alert[%s:%s] %s ~ %s",
		alert.Monitor.ID,
		alert.Monitor.Name,
		alert.OpenedAt,
		alert.ClosedAt,
	)
}

func (alert *Alert) NewReliabilityCollection(timeFrame time.Duration) (ReliabilityCollection, error) {
	isNoViolation, startAt, endAt := alert.newIsNoViolation()
	startAt = startAt.Truncate(timeFrame)
	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	reliabilitySlice := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		reliabilitySlice = append(reliabilitySlice, NewReliability(cursorAt, timeFrame, isNoViolation))
	}
	return NewReliabilityCollection(reliabilitySlice)
}

func (alert *Alert) newIsNoViolation() (isNoViolation map[time.Time]bool, startAt, endAt time.Time) {
	startAt = alert.OpenedAt
	endAt = flextime.Now().Add(time.Minute)
	if alert.ClosedAt != nil {
		endAt = *alert.ClosedAt
	}

	isNoViolation = make(map[time.Time]bool, endAt.Sub(startAt)/time.Minute)
	iter := timeutils.NewIterator(startAt, endAt, time.Minute)
	for iter.HasNext() {
		t, _ := iter.Next()
		isNoViolation[t] = false
	}
	return
}

type Alerts []*Alert

func (alerts Alerts) StartAt() time.Time {
	startAt := flextime.Now()
	for _, alert := range alerts {
		if alert.OpenedAt.Before(startAt) {
			startAt = alert.OpenedAt
		}
	}
	return startAt
}
func (alerts Alerts) EndAt() time.Time {
	endAt := time.Unix(0, 0)
	for _, alert := range alerts {
		if alert.ClosedAt == nil {
			return flextime.Now()
		}
		if alert.ClosedAt.After(endAt) {
			endAt = *alert.ClosedAt
		}
	}
	return endAt
}
