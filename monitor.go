package shimesaba

import "fmt"

type Monitor struct {
	ID   string
	Name string
	Type string
}

func (m *Monitor) String() string {
	return fmt.Sprintf("[%s]%s", m.Type, m.Name)
}
