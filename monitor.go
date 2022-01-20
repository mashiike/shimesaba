package shimesaba

import (
	"fmt"
)

type Monitor struct {
	id          string
	name        string
	monitorType string
}

func NewMonitor(id, name, monitorType string) *Monitor {
	return &Monitor{
		id:          id,
		name:        name,
		monitorType: monitorType,
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
