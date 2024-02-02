package shimesaba_test

import (
	"errors"
	"testing"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

type mockMackerelClient struct {
	shimesaba.MackerelClient
	posted []*mackerel.MetricValue
	t      *testing.T
}

func newMockMackerelClient(t *testing.T) *mockMackerelClient {
	t.Helper()
	return &mockMackerelClient{
		t: t,
	}
}

func (m *mockMackerelClient) GetOrg() (*mackerel.Org, error) {
	return &mackerel.Org{
		Name: "dummy",
	}, nil
}

func (m *mockMackerelClient) FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error) {
	require.EqualValues(
		m.t,
		&mackerel.FindHostsParam{
			Service: "shimesaba",
			Name:    "dummy-alb",
		},
		param,
	)
	return []*mackerel.Host{
		{
			ID: "dummyHostID",
		},
	}, nil
}

func (m *mockMackerelClient) PostServiceMetricValues(serviceName string, metricValues []*mackerel.MetricValue) error {
	require.Equal(m.t, "shimesaba", serviceName)
	m.posted = append(m.posted, metricValues...)
	return nil
}

func (m *mockMackerelClient) FindWithClosedAlerts() (*mackerel.AlertsResp, error) {
	return &mackerel.AlertsResp{
		Alerts: []*mackerel.Alert{
			{
				ID:        "dummyID20211001-001900",
				Status:    "WARNING",
				MonitorID: "dummyMonitorID",
				OpenedAt:  time.Date(2021, 10, 1, 0, 19, 0, 0, time.UTC).Unix(),
				Value:     0.01,
				Type:      "service",
			},
			{
				ID:        "dummyID20211001-00200",
				Status:    "WARNING",
				MonitorID: "dummyCheckMonitorID",
				OpenedAt:  time.Date(2021, 10, 1, 0, 17, 0, 0, time.UTC).Unix(),
				Value:     0.01,
				Type:      "check",
			},
		},
		NextID: "dummyNextID",
	}, nil
}

func (m *mockMackerelClient) FindWithClosedAlertsByNextID(nextID string) (*mackerel.AlertsResp, error) {
	require.Equal(m.t, "dummyNextID", nextID)
	return &mackerel.AlertsResp{
		Alerts: []*mackerel.Alert{
			{
				ID:        "dummyID20211001-001000",
				Status:    "OK",
				MonitorID: "dummyMonitorID",
				OpenedAt:  time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC).Unix(),
				ClosedAt:  time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC).Unix(),
				Value:     0.01,
				Type:      "service",
			},
		},
		NextID: "",
	}, nil
}

func (m *mockMackerelClient) GetMonitor(monitorID string) (mackerel.Monitor, error) {
	switch monitorID {
	case "dummyMonitorID":
		return &mackerel.MonitorServiceMetric{
			ID:   monitorID,
			Name: "Dummy Service Metric Monitor",
			Type: "service",
		}, nil
	case "dummyCheckMonitorID":
		return nil, &mackerel.APIError{
			StatusCode: 400,
			Message:    "Cannot get a check monitor",
		}
	default:
		require.Equal(m.t, "dummyMonitorID", monitorID)
		return nil, errors.New("unexpected monitorID")
	}
}

var graphAnnotations = []mackerel.GraphAnnotation{
	{
		ID:          "xxxxxxxxxxx",
		Title:       "hogehogehoge",
		Description: "fugafugafuga",
		From:        time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC).Unix(),
		To:          time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC).Unix(),
	},
	{
		ID:          "yyyyyyyyyyy",
		Title:       "hogehogehoge",
		Description: "SLO:*",
		From:        time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC).Unix(),
		To:          time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC).Unix(),
	},
	{
		ID:          "zzzzzzzzzzz",
		Title:       "hogehogehoge",
		Description: "ALB Failures SLO:availability,quarity affected.",
		From:        time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC).Unix(),
		To:          time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC).Unix(),
	},
}

func (m *mockMackerelClient) FindGraphAnnotations(service string, from int64, to int64) ([]mackerel.GraphAnnotation, error) {
	require.Equal(m.t, "shimesaba", service)

	return graphAnnotations, nil
}
