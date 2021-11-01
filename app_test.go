package shimesaba_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
			var buf bytes.Buffer
			logger.Setup(&buf, "debug")
			defer func() {
				t.Log(buf.String())
				logger.Setup(os.Stderr, "info")
			}()
			cfg := shimesaba.NewDefaultConfig()
			cfg.Load("testdata/simple.yaml")
			client := newMockMackerelClient(t)
			app, err := shimesaba.NewWithMackerelClient(client, cfg)
			require.NoError(t, err, "create app")
			restore := flextime.Set(time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC))
			defer restore()
			err = app.Run(context.Background(), shimesaba.BackfillOption(backfill))
			require.NoError(t, err, "run app")

			excepted := map[string]int{
				"shimesaba.error_budget.latency":                        backfill,
				"shimesaba.error_budget_consumption.latency":            backfill,
				"shimesaba.error_budget_consumption_percentage.latency": backfill,
				"shimesaba.error_budget_percentage.latency":             backfill,
				"shimesaba.failure_time.latency":                        backfill,
				"shimesaba.uptime.latency":                              backfill,
			}
			actual := make(map[string]int)
			for _, v := range client.posted {
				if _, ok := actual[v.Name]; !ok {
					actual[v.Name] = 0
				}
				actual[v.Name]++
			}
			require.EqualValues(t, excepted, actual)
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
func (m *mockMackerelClient) FetchHostMetricValues(hostID string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error) {
	require.Equal(m.t, "dummyHostID", hostID)
	require.Equal(m.t, "custom.alb.response.time_p90", metricName)
	ret := make([]mackerel.MetricValue, 0)
	for _, tv := range m.hostMetricData {
		t := tv.Time.Unix()
		if t < from || t > to {
			continue
		}
		ret = append(ret, mackerel.MetricValue{
			Name:  metricName,
			Time:  t,
			Value: tv.Value,
		})
	}
	return ret, nil
}
func (m *mockMackerelClient) FetchServiceMetricValues(serviceName string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error) {
	require.Equal(m.t, "shimesaba", serviceName)
	require.Equal(m.t, "component.dummy.response_time", metricName)
	ret := make([]mackerel.MetricValue, 0)
	for _, tv := range m.serviceMetricData {
		t := tv.Time.Unix()
		if t < from || t > to {
			continue
		}
		ret = append(ret, mackerel.MetricValue{
			Name:  metricName,
			Time:  t,
			Value: tv.Value,
		})
	}
	return ret, nil
}
func (m *mockMackerelClient) PostServiceMetricValues(serviceName string, metricValues []*mackerel.MetricValue) error {
	require.Equal(m.t, "shimesaba", serviceName)
	m.posted = metricValues
	return nil
}
