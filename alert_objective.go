package shimesaba

import (
	"log"
	"strings"
	"time"

	"github.com/mashiike/shimesaba/internal/timeutils"
)

type AlertObjective struct {
	cfg *AlertObjectiveConfig
}

func NewAlertObjective(cfg *AlertObjectiveConfig) *AlertObjective {
	return &AlertObjective{cfg: cfg}
}

func (o AlertObjective) NewReliabilityCollection(timeFrame time.Duration, alerts Alerts, startAt, endAt time.Time) (ReliabilityCollection, error) {
	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	iter.SetEnableOverWindow(true)
	rc := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		rc = append(rc, NewReliability(cursorAt, timeFrame, nil))
	}
	reliabilities, err := NewReliabilityCollection(rc)
	if err != nil {
		return nil, err
	}
	for _, alert := range alerts {
		if !o.matchAlert(alert) {
			continue
		}
		tmp, err := alert.NewReliabilityCollection(timeFrame)
		if err != nil {
			return nil, err
		}
		reliabilities, err = reliabilities.MergeInRange(tmp, startAt, endAt)
		if err != nil {
			return nil, err
		}
	}
	return reliabilities, nil
}

func (o AlertObjective) matchAlert(alert *Alert) bool {
	log.Printf("[debug] try match %s vs %v", alert, o.cfg)
	if o.MatchMonitor(alert.Monitor) {
		log.Printf("[debug] match %s", alert)
		return true
	}
	return false
}

func (o AlertObjective) MatchMonitor(monitor *Monitor) bool {
	if o.cfg.MonitorID != "" {
		if monitor.ID != o.cfg.MonitorID {
			return false
		}
	}
	if o.cfg.MonitorNamePrefix != "" {
		if !strings.HasPrefix(monitor.Name, o.cfg.MonitorNamePrefix) {
			return false
		}
	}
	if o.cfg.MonitorNameSuffix != "" {
		if !strings.HasSuffix(monitor.Name, o.cfg.MonitorNameSuffix) {
			return false
		}
	}
	if o.cfg.MonitorType != "" {
		if !strings.EqualFold(monitor.Type, o.cfg.MonitorType) {
			return false
		}
	}
	return true
}
