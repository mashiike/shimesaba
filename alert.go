package shimesaba

import (
	"fmt"
	"time"

	"github.com/Songmu/flextime"
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
