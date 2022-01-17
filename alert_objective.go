package shimesaba

import (
	"log"
	"strings"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

type AlertObjective struct {
	cfg *AlertObjectiveConfig
}

func NewAlertObjective(cfg *AlertObjectiveConfig) *AlertObjective {
	return &AlertObjective{cfg: cfg}
}

func (o AlertObjective) NewReliabilityCollection(timeFrame time.Duration, alerts Alerts, startAt, endAt time.Time) (ReliabilityCollection, error) {
	isNoViolation := o.newIsNoViolation(alerts)

	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	iter.SetEnableOverWindow(true)
	reliabilitySlice := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		reliabilitySlice = append(reliabilitySlice, NewReliability(cursorAt, timeFrame, isNoViolation))
	}
	return NewReliabilityCollection(reliabilitySlice)
}

func (o AlertObjective) newIsNoViolation(alerts Alerts) map[time.Time]bool {
	now := flextime.Now().Add(time.Minute)
	isNoViolation := make(map[time.Time]bool)
	for _, alert := range alerts {
		if !o.matchAlert(alert) {
			continue
		}
		closedAt := now
		if alert.ClosedAt != nil {
			closedAt = *alert.ClosedAt
		}
		iter := timeutils.NewIterator(alert.OpenedAt, closedAt, time.Minute)
		for iter.HasNext() {
			t, _ := iter.Next()
			isNoViolation[t] = false
		}
	}
	return isNoViolation
}

func (o AlertObjective) matchAlert(alert *Alert) bool {
	log.Printf("[debug] try match %s vs %v", alert, o.cfg)
	if o.cfg.MonitorID != "" {
		if alert.Monitor.ID != o.cfg.MonitorID {
			return false
		}
	}
	if o.cfg.MonitorNamePrefix != "" {
		if !strings.HasPrefix(alert.Monitor.Name, o.cfg.MonitorNamePrefix) {
			return false
		}
	}
	if o.cfg.MonitorNameSuffix != "" {
		if !strings.HasSuffix(alert.Monitor.Name, o.cfg.MonitorNameSuffix) {
			return false
		}
	}
	if o.cfg.MonitorType != "" {
		if !strings.EqualFold(alert.Monitor.Type, o.cfg.MonitorType) {
			return false
		}
	}
	log.Printf("[debug] match %s", alert)
	return true
}
