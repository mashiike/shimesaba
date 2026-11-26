package shimesaba_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

// fakeMackerelAPI is a minimal HTTP server that speaks the subset of the Mackerel API
// that shimesaba uses. Unlike mockMackerelClient, it lets the real mackerel-client-go
// build requests and decode responses, so updates of mackerel-client-go are covered.
type fakeMackerelAPI struct {
	t      *testing.T
	server *httptest.Server
	apiKey string

	mu            sync.Mutex
	alertRequests int
	posted        map[string][]mackerel.MetricValue
}

const fakeMackerelAPIMaxAlertRequests = 10

func newFakeMackerelAPI(t *testing.T) *fakeMackerelAPI {
	t.Helper()
	f := &fakeMackerelAPI{
		t:      t,
		apiKey: "dummy-api-key",
		posted: make(map[string][]mackerel.MetricValue),
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeMackerelAPI) client(t *testing.T) *mackerel.Client {
	t.Helper()
	client, err := mackerel.NewClientWithOptions(f.apiKey, f.server.URL, false)
	require.NoError(t, err)
	return client
}

func (f *fakeMackerelAPI) handle(w http.ResponseWriter, r *http.Request) {
	if got := r.Header.Get("X-Api-Key"); got != f.apiKey {
		f.t.Errorf("unexpected X-Api-Key header: %q", got)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v0/org":
		f.writeJSON(w, map[string]any{"name": "dummy"})
	case r.Method == http.MethodGet && r.URL.Path == "/api/v0/alerts":
		f.handleAlerts(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v0/monitors/dummyMonitorID":
		io.WriteString(w, `{"monitor":{"id":"dummyMonitorID","name":"Dummy Service Metric Monitor","type":"service","service":"shimesaba","metric":"custom.dummy","operator":">","warning":1,"critical":2,"duration":1}}`)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v0/graph-annotations":
		require.Equal(f.t, "shimesaba", r.URL.Query().Get("service"))
		f.writeJSON(w, map[string]any{"graphAnnotations": graphAnnotations})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v0/services/shimesaba/tsdb":
		var values []mackerel.MetricValue
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			f.t.Errorf("decode posted metrics: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		for _, v := range values {
			f.posted[v.Name] = append(f.posted[v.Name], v)
		}
		f.mu.Unlock()
		f.writeJSON(w, map[string]any{"success": true})
	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleAlerts returns three pages of alerts.
// The second page has only an alert without monitorId, which shimesaba skips.
func (f *fakeMackerelAPI) handleAlerts(w http.ResponseWriter, r *http.Request) {
	require.Equal(f.t, "true", r.URL.Query().Get("withClosed"))
	f.mu.Lock()
	f.alertRequests++
	n := f.alertRequests
	f.mu.Unlock()
	if n > fakeMackerelAPIMaxAlertRequests {
		f.t.Errorf("too many alert requests (%d): paging does not seem to advance", n)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	unix := func(min int) int64 {
		return time.Date(2021, 10, 1, 0, min, 0, 0, time.UTC).Unix()
	}
	switch nextID := r.URL.Query().Get("nextId"); nextID {
	case "":
		fmt.Fprintf(w, `{"alerts":[
			{"id":"dummyID20211001-001900","status":"WARNING","monitorId":"dummyMonitorID","type":"service","value":0.01,"openedAt":%d},
			{"id":"dummyID20211001-00200","status":"WARNING","monitorId":"dummyCheckMonitorID","type":"check","hostId":"dummyHostID","openedAt":%d}
		],"nextId":"page2"}`, unix(19), unix(17))
	case "page2":
		fmt.Fprintf(w, `{"alerts":[
			{"id":"noMonitorAlert","status":"OK","type":"connectivity","openedAt":%d,"closedAt":%d}
		],"nextId":"page3"}`, unix(12), unix(13))
	case "page3":
		fmt.Fprintf(w, `{"alerts":[
			{"id":"dummyID20211001-001000","status":"OK","monitorId":"dummyMonitorID","type":"service","value":0.01,"openedAt":%d,"closedAt":%d}
		]}`, unix(10), unix(15))
	default:
		f.t.Errorf("unexpected nextId: %s", nextID)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (f *fakeMackerelAPI) writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		f.t.Errorf("encode response: %s", err)
	}
}

func TestRepositoryWithMackerelAPI(t *testing.T) {
	api := newFakeMackerelAPI(t)
	repo := shimesaba.NewRepository(api.client(t))
	ctx := context.Background()

	orgName, err := repo.GetOrgName(ctx)
	require.NoError(t, err)
	require.Equal(t, "dummy", orgName)

	alerts, err := repo.FetchAlerts(ctx,
		time.Date(2021, 10, 1, 0, 5, 0, 0, time.UTC),
		time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	for _, a := range alerts {
		a.Monitor = a.Monitor.WithEvaluator(nil)
	}
	require.EqualValues(t, shimesaba.Alerts{
		{
			OpenedAt: time.Date(2021, 10, 1, 0, 19, 0, 0, time.UTC),
			Monitor:  shimesaba.NewMonitor("dummyMonitorID", "Dummy Service Metric Monitor", "service"),
		},
		{
			OpenedAt: time.Date(2021, 10, 1, 0, 17, 0, 0, time.UTC),
			Monitor:  shimesaba.NewMonitor("dummyCheckMonitorID", "check monitor dummyCheckMonitorID", "check"),
			HostID:   "dummyHostID",
		},
		{
			OpenedAt: time.Date(2021, 10, 1, 0, 10, 0, 0, time.UTC),
			Monitor:  shimesaba.NewMonitor("dummyMonitorID", "Dummy Service Metric Monitor", "service"),
			ClosedAt: ptrTime(time.Date(2021, 10, 1, 0, 15, 0, 0, time.UTC)),
		},
	}, alerts)
	require.Equal(t, 3, api.alertRequests, "alert pages must be fetched once each")

	vAlerts, err := repo.FetchVirtualAlerts(ctx, "shimesaba", "availability",
		time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Len(t, vAlerts, 2, "`SLO:*` and `SLO:availability,...` annotations")
}

// TestAppWithMackerelAPI runs the same scenario as TestAppWithMock through the real
// mackerel-client-go, and checks that the posted metrics are the same.
func TestAppWithMackerelAPI(t *testing.T) {
	// The other testdata configs have an error budget below 1m (5m * 0.1 truncated to 0m),
	// which makes the percentage metrics NaN/Inf. They can't be encoded as JSON,
	// so they are not used here.
	configFiles := []string{
		"testdata/app_api_test.yaml",
	}
	for _, configFile := range configFiles {
		t.Run(configFile, func(t *testing.T) {
			restore := flextime.Set(time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC))
			defer restore()

			run := func(client shimesaba.MackerelClient) {
				t.Helper()
				cfg := shimesaba.NewDefaultConfig()
				require.NoError(t, cfg.Load(configFile))
				app, err := shimesaba.NewWithMackerelClient(client, cfg)
				require.NoError(t, err)
				require.NoError(t, app.Run(context.Background(), shimesaba.BackfillOption(3)))
			}

			mock := newMockMackerelClient(t)
			run(mock)
			expected := make(map[string][]mackerel.MetricValue)
			for _, v := range mock.posted {
				expected[v.Name] = append(expected[v.Name], mackerel.MetricValue{Name: v.Name, Time: v.Time, Value: v.Value})
			}

			api := newFakeMackerelAPI(t)
			run(api.client(t))

			require.NotEmpty(t, api.posted)
			require.Equal(t, len(expected), len(api.posted))
			for name, values := range expected {
				actual := api.posted[name]
				require.Len(t, actual, len(values), name)
				for i, v := range values {
					require.False(t, math.IsNaN(v.Value.(float64)) || math.IsInf(v.Value.(float64), 0), "%s must be finite", name)
					require.Equal(t, v.Time, actual[i].Time, name)
					require.InDelta(t, v.Value, actual[i].Value, 1e-9, name)
				}
			}
		})
	}
}

func TestDryRunDoesNotPost(t *testing.T) {
	restore := flextime.Set(time.Date(2021, 10, 1, 0, 21, 0, 0, time.UTC))
	defer restore()

	api := newFakeMackerelAPI(t)
	cfg := shimesaba.NewDefaultConfig()
	require.NoError(t, cfg.Load("testdata/app_test.yaml"))
	app, err := shimesaba.NewWithMackerelClient(api.client(t), cfg)
	require.NoError(t, err)
	require.NoError(t, app.Run(context.Background(),
		shimesaba.BackfillOption(3),
		shimesaba.DryRunOption(true),
		shimesaba.DumpReportsOption(true),
	))
	require.Empty(t, api.posted)
}
