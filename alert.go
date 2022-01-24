package shimesaba

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

type Alert struct {
	Monitor  *Monitor
	HostID   string
	OpenedAt time.Time
	ClosedAt *time.Time
	Reason   string

	mu    sync.Mutex
	cache Reliabilities
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
func (alert *Alert) WithHostID(hostID string) *Alert {
	return &Alert{
		Monitor:  alert.Monitor,
		OpenedAt: alert.OpenedAt,
		ClosedAt: alert.ClosedAt,
		HostID:   hostID,
		Reason:   alert.Reason,
	}
}

func (alert *Alert) WithReason(reason string) *Alert {
	return &Alert{
		Monitor:  alert.Monitor,
		OpenedAt: alert.OpenedAt,
		ClosedAt: alert.ClosedAt,
		HostID:   alert.HostID,
		Reason:   reason,
	}
}

func (alert *Alert) String() string {
	return fmt.Sprintf("alert[%s:%s] %s ~ %s",
		alert.Monitor.ID(),
		alert.Monitor.Name(),
		alert.OpenedAt,
		alert.ClosedAt,
	)
}

func (alert *Alert) endAt() time.Time {
	if alert.ClosedAt != nil {
		return *alert.ClosedAt
	}
	return flextime.Now().Add(time.Minute)
}

func (alert *Alert) EvaluateReliabilities(timeFrame time.Duration) (Reliabilities, error) {
	log.Printf("[debug] EvaluateReliabilities alert=%s", alert)
	alert.mu.Lock()
	defer alert.mu.Unlock()
	if alert.cache != nil {
		log.Printf("[debug] return cache alert=%s", alert)
		return alert.cache, nil
	}
	if os.Getenv("SHIMESABA_ENABLE_REASSESSMENT") != "" {
		if reliabilities, ok := alert.Monitor.EvaluateReliabilities(
			alert.HostID,
			timeFrame,
			alert.OpenedAt.Add(-15*time.Minute),
			alert.endAt(),
		); ok {
			log.Printf("[notice] applying SLO reassessment as an experimental feature for Monitor %s.", alert.Monitor.name)
			alert.cache = reliabilities
			return reliabilities, nil
		}
	}
	var startAt, endAt time.Time
	var isNoViolation IsNoViolationCollection
	if correctionTime, ok := alert.CorrectionTime(); ok {
		log.Printf("[notice] applying SLO Violation time %s, to %s", correctionTime, alert.Monitor.name)
		startAt = alert.OpenedAt
		endAt = alert.endAt()
		isNoViolation = make(IsNoViolationCollection, correctionTime/time.Minute)
		iter := timeutils.NewIterator(startAt, alert.OpenedAt.Add(correctionTime), time.Minute)
		for iter.HasNext() {
			t, _ := iter.Next()
			isNoViolation[t] = false
		}
	} else {
		isNoViolation, startAt, endAt = alert.newIsNoViolation()
	}

	reliabilities, err := isNoViolation.NewReliabilities(timeFrame, startAt, endAt)
	if err != nil {
		return nil, err
	}
	alert.cache = reliabilities
	return reliabilities, nil
}

const correctionKeyword = "downtime:"

func (alert *Alert) CorrectionTime() (time.Duration, bool) {
	i := strings.Index(alert.Reason, correctionKeyword)
	if i < 0 {
		return 0, false
	}
	str := alert.Reason[i+len(correctionKeyword):]
	j := strings.IndexRune(str, ' ')
	if j >= 0 {
		str = str[:j]
	}
	d, err := timeutils.ParseDuration(str)
	if err != nil {
		log.Printf("[debug] try parse correction time failed:%s", err)
		return 0, false
	}
	return d, true
}

func (alert *Alert) newIsNoViolation() (isNoViolation IsNoViolationCollection, startAt, endAt time.Time) {
	startAt = alert.OpenedAt
	endAt = alert.endAt()

	isNoViolation = make(IsNoViolationCollection, endAt.Sub(startAt)/time.Minute)
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
