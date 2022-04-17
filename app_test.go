package shimesaba_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba"
	"github.com/mashiike/shimesaba/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestAppWithMock(t *testing.T) {
	backfillCounts := []int{3, 4, 5}
	for _, backfill := range backfillCounts {
		t.Run(fmt.Sprintf("backfill=%d", backfill), func(t *testing.T) {
			cases := []struct {
				configFile string
				expected   map[string]int
			}{
				{
					configFile: "testdata/alert_source.yaml",
					expected: map[string]int{
						"shimesaba.error_budget.alerts":                        backfill,
						"shimesaba.error_budget_consumption.alerts":            backfill,
						"shimesaba.error_budget_consumption_percentage.alerts": backfill,
						"shimesaba.error_budget_percentage.alerts":             backfill,
						"shimesaba.failure_time.alerts":                        backfill,
						"shimesaba.uptime.alerts":                              backfill,
					},
				},
			}
			for _, c := range cases {
				t.Run(c.configFile, func(t *testing.T) {
					var buf bytes.Buffer
					logger.Setup(&buf, "debug")
					defer func() {
						t.Log(buf.String())
						logger.Setup(os.Stderr, "info")
					}()
					cfg := shimesaba.NewDefaultConfig()
					cfg.Load(c.configFile)
					client := newMockMackerelClient(t)
					app, err := shimesaba.NewWithMackerelClient(client, cfg)
					require.NoError(t, err, "create app")
					restore := flextime.Set(time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC))
					defer restore()
					err = app.Run(context.Background(), shimesaba.BackfillOption(backfill))
					require.NoError(t, err, "run app")

					actual := make(map[string]int)
					for _, v := range client.posted {
						if _, ok := actual[v.Name]; !ok {
							actual[v.Name] = 0
						}
						actual[v.Name]++
					}
					require.EqualValues(t, c.expected, actual)
				})
			}
		})
	}
}

type mockMackerelClient struct {
	shimesaba.MackerelClient
	hostMetricData    []timeValueTuple
	serviceMetricData []timeValueTuple
	posted            []*mackerel.MetricValue
	t                 *testing.T
}

func newMockMackerelClient(t *testing.T) *mockMackerelClient {
	t.Helper()
	return &mockMackerelClient{
		hostMetricData:    loadTupleFromCSV(t, "testdata/dummy3.csv"),
		serviceMetricData: loadTupleFromCSV(t, "testdata/dummy4.csv"),
		t:                 t,
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
	require.Equal(m.t, "dummyMonitorID", monitorID)
	return &mackerel.MonitorServiceMetric{
		ID:   monitorID,
		Name: "Dummy Service Metric Monitor",
		Type: "service",
	}, nil
}

type timeValueTuple struct {
	Time  time.Time
	Value interface{}
}

func loadTupleFromCSV(t *testing.T, path string) []timeValueTuple {
	t.Helper()
	fp, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	reader := csv.NewReader(fp)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := records[0]
	var timeIndex, valueIndex int
	for i, h := range header {
		if strings.HasPrefix(h, "time") {
			timeIndex = i
			continue
		}
		if strings.HasSuffix(h, "value") {
			valueIndex = i
			continue
		}
	}
	ret := make([]timeValueTuple, 0, len(records)-1)
	for _, record := range records[1:] {

		tt, err := time.Parse(time.RFC3339Nano, record[timeIndex])
		if err != nil {
			t.Fatal(err)
		}
		tv, err := strconv.ParseFloat(record[valueIndex], 64)
		if err != nil {
			t.Fatal(err)
		}
		ret = append(ret, timeValueTuple{
			Time:  tt,
			Value: tv,
		})
	}
	return ret
}
