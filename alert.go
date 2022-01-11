package shimesaba

import "time"

type Alert struct {
	MonitorID string
	OpenedAt  time.Time
	ClosedAt  *time.Time
}

type Alerts []*Alert
