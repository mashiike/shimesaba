package shimesaba

import (
	"fmt"
	"time"
)

type Monitor struct {
	id          string
	name        string
	monitorType string
	evaluator   func(hostID string, timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, bool)
}

func NewMonitor(id, name, monitorType string) *Monitor {
	return &Monitor{
		id:          id,
		name:        name,
		monitorType: monitorType,
	}
}

func (m *Monitor) WithEvaluator(evaluator func(hostID string, timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, bool)) *Monitor {
	return &Monitor{
		id:          m.id,
		name:        m.name,
		monitorType: m.monitorType,
		evaluator:   evaluator,
	}
}

func (m *Monitor) ID() string {
	return m.id
}

func (m *Monitor) Name() string {
	return m.name
}

func (m *Monitor) Type() string {
	return m.monitorType
}

func (m *Monitor) String() string {
	return fmt.Sprintf("[%s]%s", m.monitorType, m.name)
}

func (m *Monitor) EvaluateReliabilities(hostID string, timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, bool) {
	if m.evaluator == nil {
		return nil, false
	}
	return m.evaluator(hostID, timeFrame, startAt, endAt)
}
