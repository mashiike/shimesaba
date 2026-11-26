package shimesaba_test

import (
	"errors"
	"testing"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

// metricMockClient returns monitors and metric values for the reassessment
// (try_reassessment) evaluators.
type metricMockClient struct {
	shimesaba.MackerelClient
	t        *testing.T
	monitors []mackerel.Monitor
	values   []mackerel.MetricValue
	err      error
}

func (m *metricMockClient) FindMonitors() ([]mackerel.Monitor, error) {
	return m.monitors, nil
}

func (m *metricMockClient) FetchHostMetricValues(hostID string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error) {
	require.Equal(m.t, "dummyHostID", hostID)
	require.Equal(m.t, "custom.dummy", metricName)
	return m.values, m.err
}

func (m *metricMockClient) FetchServiceMetricValues(serviceName string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error) {
	require.Equal(m.t, "shimesaba", serviceName)
	require.Equal(m.t, "custom.dummy", metricName)
	return m.values, m.err
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func TestMonitorEvaluateReliabilitiesWithMetrics(t *testing.T) {
	startAt := time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.Add(10 * time.Minute)
	// one value per minute: 00:00 -> 0, 00:01 -> 1, ..., 00:09 -> 9
	values := make([]mackerel.MetricValue, 0, 10)
	for i := 0; i < 10; i++ {
		values = append(values, mackerel.MetricValue{
			Name:  "custom.dummy",
			Time:  startAt.Add(time.Duration(i) * time.Minute).Unix(),
			Value: float64(i),
		})
	}

	cases := []struct {
		name            string
		operator        string
		warning         *float64
		critical        *float64
		values          []mackerel.MetricValue
		err             error
		expectedOK      bool
		expectedFailure time.Duration
	}{
		{
			name:            "> warning",
			operator:        ">",
			warning:         ptrFloat64(7),
			expectedOK:      true,
			expectedFailure: 2 * time.Minute, // 8, 9
		},
		{
			name:            "> critical only",
			operator:        ">",
			critical:        ptrFloat64(8),
			expectedOK:      true,
			expectedFailure: 1 * time.Minute, // 9
		},
		{
			name:            "> warning and critical",
			operator:        ">",
			warning:         ptrFloat64(6),
			critical:        ptrFloat64(8),
			expectedOK:      true,
			expectedFailure: 3 * time.Minute, // 7, 8, 9
		},
		{
			name:            "< warning",
			operator:        "<",
			warning:         ptrFloat64(2),
			expectedOK:      true,
			expectedFailure: 2 * time.Minute, // 0, 1
		},
		{
			name:            "< critical only",
			operator:        "<",
			critical:        ptrFloat64(1),
			expectedOK:      true,
			expectedFailure: 1 * time.Minute, // 0
		},
		{
			name:            "< warning and critical",
			operator:        "<",
			warning:         ptrFloat64(3),
			critical:        ptrFloat64(1),
			expectedOK:      true,
			expectedFailure: 3 * time.Minute, // 0, 1, 2
		},
		{
			name:            "no violation",
			operator:        ">",
			warning:         ptrFloat64(100),
			critical:        ptrFloat64(200),
			expectedOK:      true,
			expectedFailure: 0,
		},
		{
			name:     "non float values are ignored",
			operator: ">",
			warning:  ptrFloat64(0),
			values: []mackerel.MetricValue{
				{Name: "custom.dummy", Time: startAt.Add(time.Minute).Unix(), Value: "not a number"},
			},
			expectedOK:      true,
			expectedFailure: 0,
		},
		{
			name:       "unknown operator",
			operator:   "==",
			warning:    ptrFloat64(1),
			expectedOK: false,
		},
		{
			name:       "fetch metrics failed",
			operator:   ">",
			warning:    ptrFloat64(1),
			err:        errors.New("dummy error"),
			expectedOK: false,
		},
	}

	for _, c := range cases {
		monitorValues := c.values
		if monitorValues == nil {
			monitorValues = values
		}
		monitors := map[string]mackerel.Monitor{
			"host": &mackerel.MonitorHostMetric{
				ID: "hostMonitorID", Name: "Dummy Host Metric Monitor", Type: "host",
				Metric: "custom.dummy", Operator: c.operator, Warning: c.warning, Critical: c.critical,
			},
			"service": &mackerel.MonitorServiceMetric{
				ID: "serviceMonitorID", Name: "Dummy Service Metric Monitor", Type: "service",
				Service: "shimesaba", Metric: "custom.dummy", Operator: c.operator, Warning: c.warning, Critical: c.critical,
			},
		}
		for kind, monitor := range monitors {
			t.Run(kind+"/"+c.name, func(t *testing.T) {
				client := &metricMockClient{
					t:        t,
					monitors: []mackerel.Monitor{monitor},
					values:   monitorValues,
					err:      c.err,
				}
				repo := shimesaba.NewRepository(client)
				found, err := repo.FindMonitors()
				require.NoError(t, err)
				require.Len(t, found, 1)

				var reliabilities shimesaba.Reliabilities
				var ok bool
				require.NotPanics(t, func() {
					reliabilities, ok = found[0].EvaluateReliabilities("dummyHostID", 5*time.Minute, startAt, endAt)
				})
				require.Equal(t, c.expectedOK, ok)
				if !c.expectedOK {
					return
				}
				var failureTime, upTime time.Duration
				for _, r := range reliabilities {
					failureTime += r.FailureTime()
					upTime += r.UpTime()
				}
				require.Equal(t, c.expectedFailure, failureTime)
				require.Equal(t, 10*time.Minute-c.expectedFailure, upTime)
			})
		}
	}
}

func TestMonitorWithoutEvaluator(t *testing.T) {
	m := shimesaba.NewMonitor("id", "name", "check")
	reliabilities, ok := m.EvaluateReliabilities("dummyHostID", time.Minute, time.Now(), time.Now().Add(time.Minute))
	require.False(t, ok)
	require.Nil(t, reliabilities)
}
