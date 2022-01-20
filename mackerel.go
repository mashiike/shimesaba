package shimesaba

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/mashiike/shimesaba/internal/timeutils"
	retry "github.com/shogo82148/go-retry"
)

// MackerelClient is an abstraction interface for mackerel-client-go.Client
type MackerelClient interface {
	FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error)
	FetchHostMetricValues(hostID string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
	FetchServiceMetricValues(serviceName string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
	PostServiceMetricValues(serviceName string, metricValues []*mackerel.MetricValue) error

	FindDashboards() ([]*mackerel.Dashboard, error)
	FindDashboard(dashboardID string) (*mackerel.Dashboard, error)
	CreateDashboard(param *mackerel.Dashboard) (*mackerel.Dashboard, error)
	UpdateDashboard(dashboardID string, param *mackerel.Dashboard) (*mackerel.Dashboard, error)

	FindWithClosedAlerts() (*mackerel.AlertsResp, error)
	FindWithClosedAlertsByNextID(nextID string) (*mackerel.AlertsResp, error)
	GetMonitor(monitorID string) (mackerel.Monitor, error)
	FindMonitors() ([]mackerel.Monitor, error)
}

// Repository handles reading and writing data
type Repository struct {
	client MackerelClient

	mu          sync.Mutex
	monitorByID map[string]*Monitor
}

// NewRepository creates Repository
func NewRepository(client MackerelClient) *Repository {
	return &Repository{
		client:      client,
		monitorByID: make(map[string]*Monitor),
	}
}

const (
	fetchMetricMetricmit = 6 * time.Hour
)

// FetchMetric gets Metric using MetricConfig
func (repo *Repository) FetchMetric(ctx context.Context, cfg *MetricConfig, startAt time.Time, endAt time.Time) (*Metric, error) {
	iter := timeutils.NewIterator(startAt, endAt, fetchMetricMetricmit)
	m := NewMetric(cfg)

	var fetchMetricValues func(int64, int64) ([]mackerel.MetricValue, error)
	switch cfg.Type {
	case HostMetric:
		hosts, err := repo.client.FindHosts(&mackerel.FindHostsParam{
			Service: cfg.ServiceName,
			Roles:   cfg.Roles,
			Name:    cfg.HostName,
		})
		if err != nil {
			return nil, err
		}
		fetchMetricValues = func(from, to int64) ([]mackerel.MetricValue, error) {
			values := make([]mackerel.MetricValue, 0)
			for _, host := range hosts {
				log.Printf("[debug] call MackerelClient.FetchHostMetricValues(%s,%s,%s,%s)", host.ID, cfg.Name, time.Unix(from, 0), time.Unix(to, 0))
				v, err := repo.client.FetchHostMetricValues(host.ID, cfg.Name, from, to)
				if err != nil {
					return nil, err
				}
				values = append(values, v...)
			}
			return values, nil
		}
	case ServiceMetric:
		fetchMetricValues = func(from, to int64) ([]mackerel.MetricValue, error) {
			log.Printf("[debug] call MackerelClient.FetchServiceMetricValues(%s,%s,%s,%s)", cfg.ServiceName, cfg.Name, time.Unix(from, 0), time.Unix(to, 0))
			return repo.client.FetchServiceMetricValues(cfg.ServiceName, cfg.Name, from, to)
		}
	default:
		return nil, fmt.Errorf("metric type `%s` is unknown", cfg.Type)
	}

	for iter.HasNext() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		curStartat, curEndAt := iter.Next()
		values, err := fetchMetricValues(curStartat.Unix(), curEndAt.Unix())
		if err != nil {
			return nil, fmt.Errorf("metric=%s :%w", m.ID(), err)
		}
		for _, value := range values {
			if err := m.AppendValue(time.Unix(value.Time, 0), value.Value); err != nil {
				return nil, fmt.Errorf("metric=%s :%w", m.ID(), err)
			}
		}
		time.Sleep(500 * time.Microsecond)
	}
	return m, nil
}

// FetchMetrics gets metrics togethers
func (repo *Repository) FetchMetrics(ctx context.Context, cfgs MetricConfigs, startAt time.Time, endAt time.Time) (Metrics, error) {
	ms := make(Metrics)
	for _, cfg := range cfgs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		log.Printf("[info] start fetch metric_id=%s\n", cfg.ID)
		m, err := repo.FetchMetric(ctx, cfg, startAt, endAt)
		log.Printf("[info] finished fetch metric_id=%s\n", cfg.ID)
		if err != nil {
			return nil, err
		}
		ms.Set(m)
	}
	return ms, nil
}

// SaveReports posts Reports to Mackerel
func (repo *Repository) SaveReports(ctx context.Context, reports []*Report) error {
	services := make(map[string][]*mackerel.MetricValue)
	for _, report := range reports {
		values, ok := services[report.Destination.ServiceName]
		if !ok {
			values = make([]*mackerel.MetricValue, 0)
		}
		values = append(values, newMackerelMetricValuesFromReport(report)...)
		services[report.Destination.ServiceName] = values
	}
	for service, values := range services {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := repo.postServiceMetricValues(ctx, service, values); err != nil {
			return fmt.Errorf("post service `%s` metric values: %w", service, err)
		}
	}
	return nil
}

const batchSize = 100

var policy = retry.Policy{
	MinDelay: time.Second,
	MaxDelay: 10 * time.Second,
	MaxCount: 10,
}

func (repo *Repository) postServiceMetricValues(ctx context.Context, service string, values []*mackerel.MetricValue) error {
	size := len(values)
	for i := 0; i < size; i += batchSize {
		start, end := i, i+batchSize
		if size < end {
			end = size
		}
		log.Printf("[info] PostServiceMetricValues %s values[%d:%d]\n", service, start, end)
		err := policy.Do(ctx, func() error {
			err := repo.client.PostServiceMetricValues(service, values[start:end])
			if err != nil {
				log.Printf("[warn] PostServiceMetricValues retry because: %s\n", err)
			}
			return err
		})
		if err != nil {
			log.Printf("[warn] failed to PostServiceMetricValues service:%s %s\n", service, err)
		}
	}
	return nil
}

func newMackerelMetricValuesFromReport(report *Report) []*mackerel.MetricValue {
	values := make([]*mackerel.MetricValue, 0, 5)
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.ErrorBudgetMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.ErrorBudget.Minutes(),
	})
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.ErrorBudgetPercentageMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.ErrorBudgetUsageRate() * 100.0,
	})
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.ErrorBudgetConsumptionMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.ErrorBudgetConsumption.Minutes(),
	})
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.ErrorBudgetConsumptionPercentageMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.ErrorBudgetConsumptionRate() * 100.0,
	})
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.UpTimeMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.UpTime.Minutes(),
	})
	values = append(values, &mackerel.MetricValue{
		Name:  report.Destination.FailureMetricName(),
		Time:  report.DataPoint.Unix(),
		Value: report.FailureTime.Minutes(),
	})
	return values
}

//Dashboard is alias of mackerel.Dashboard
type Dashboard = mackerel.Dashboard

var ErrDashboardNotFound = errors.New("dashboard not found")

// FindDashboardID get Mackerel Dashboard ID from url or id
func (repo *Repository) FindDashboardID(dashboardIDOrURL string) (string, error) {
	dashboards, err := repo.client.FindDashboards()
	if err != nil {
		return "", err
	}
	for _, d := range dashboards {
		if d.ID == dashboardIDOrURL {
			return d.ID, nil
		}
		if d.URLPath == dashboardIDOrURL {
			return d.ID, nil
		}
	}
	return "", ErrDashboardNotFound
}

// FindDashboard get Mackerel Dashboard
func (repo *Repository) FindDashboard(dashboardIDOrURL string) (*Dashboard, error) {
	id, err := repo.FindDashboardID(dashboardIDOrURL)
	if err != nil {
		return nil, err
	}
	//Get Widgets
	dashboard, err := repo.client.FindDashboard(id)
	if err != nil {
		return nil, err
	}

	dashboard.ID = ""
	dashboard.CreatedAt = 0
	dashboard.UpdatedAt = 0
	return dashboard, nil
}

// SaveDashboard post Mackerel Dashboard
func (repo *Repository) SaveDashboard(ctx context.Context, dashboard *Dashboard) error {
	id, err := repo.FindDashboardID(dashboard.URLPath)
	if err == nil {
		log.Printf("[debug] update dashboard id=%s url=%s", id, dashboard.URLPath)
		after, err := repo.client.UpdateDashboard(id, dashboard)
		if err != nil {
			return err
		}
		log.Printf("[info] updated dashboard id=%s url=%s updated_at=%s", after.ID, after.URLPath, time.Unix(after.UpdatedAt, 0).String())
	}
	if err == ErrDashboardNotFound {
		log.Printf("[debug] create dashboard url=%s", dashboard.URLPath)
		after, err := repo.client.CreateDashboard(dashboard)
		if err != nil {
			return err
		}
		log.Printf("[info] updated dashboard id=%s url=%s updated_at=%s", after.ID, after.URLPath, time.Unix(after.CreatedAt, 0).String())
	}
	return err

}

// FetchAlerts retrieves alerts for a specified period of time
func (repo *Repository) FetchAlerts(ctx context.Context, startAt time.Time, endAt time.Time) (Alerts, error) {
	alerts := make(Alerts, 0, 100)
	log.Printf("[debug] call MackerelClient.FindWithClosedAlerts()")
	resp, err := repo.client.FindWithClosedAlerts()
	if err != nil {
		return nil, err
	}
	converted, err := repo.convertAlerts(resp, endAt)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, converted...)
	currentAt := flextime.Now()
	if len(alerts) != 0 {
		currentAt = alerts[len(alerts)-1].OpenedAt
	}
	for startAt.Before(currentAt) && resp.NextID != "" {
		log.Printf("[debug] call MackerelClient.FindWithClosedAlertsByNextID(%s)", resp.NextID)
		resp, err = repo.client.FindWithClosedAlertsByNextID(resp.NextID)
		if err != nil {
			return nil, err
		}
		converted, err := repo.convertAlerts(resp, endAt)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, converted...)
		if len(alerts) != 0 {
			currentAt = alerts[len(alerts)-1].OpenedAt
		}
	}
	return alerts, nil
}

func (repo *Repository) convertAlerts(resp *mackerel.AlertsResp, endAt time.Time) ([]*Alert, error) {
	alerts := make([]*Alert, 0, len(resp.Alerts))
	for _, alert := range resp.Alerts {
		if alert.MonitorID == "" {
			continue
		}
		openedAt := time.Unix(alert.OpenedAt, 0)
		if openedAt.After(endAt) {
			continue
		}
		var closedAt *time.Time
		if alert.Status == "OK" {
			tmpClosedAt := time.Unix(alert.ClosedAt, 0)
			closedAt = &tmpClosedAt
		}
		monitor, err := repo.getMonitor(alert.MonitorID)
		if err != nil {
			return nil, err
		}
		a := NewAlert(
			monitor,
			openedAt,
			closedAt,
		)
		a = a.WithHostID(alert.HostID)
		log.Printf("[debug] %s", a)
		alerts = append(alerts, a)
	}
	return alerts, nil
}

func (repo *Repository) getMonitor(id string) (*Monitor, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if monitor, ok := repo.monitorByID[id]; ok {
		return monitor, nil
	}
	log.Printf("[debug] call GetMonitor(%s)", id)
	monitor, err := repo.client.GetMonitor(id)
	if err != nil {
		return nil, err
	}
	log.Printf("[debug] catch monitor[%s] = %#v", id, monitor)
	repo.monitorByID[id] = repo.convertMonitor(monitor)
	return repo.monitorByID[id], nil
}

func (repo *Repository) FindMonitors() ([]*Monitor, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	log.Printf("[debug] call FindMonitors()")
	monitors, err := repo.client.FindMonitors()
	if err != nil {
		return nil, err
	}
	ret := make([]*Monitor, 0, len(monitors))
	for _, m := range monitors {
		monitor := repo.convertMonitor(m)
		repo.monitorByID[monitor.ID()] = monitor
		ret = append(ret, monitor)
	}
	return ret, nil
}

func (repo *Repository) convertMonitor(monitor mackerel.Monitor) *Monitor {
	m := NewMonitor(
		monitor.MonitorID(),
		monitor.MonitorName(),
		monitor.MonitorType(),
	)
	switch monitor := monitor.(type) {
	case *mackerel.MonitorHostMetric:
		m = m.WithEvaluator(func(hostID string, timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, bool) {
			log.Printf("[debug] try evaluate host metric, host_id=`%s`, monitor=`%s` time=%s~%s", hostID, monitor.Name, startAt, endAt)
			metrics, err := repo.client.FetchHostMetricValues(hostID, monitor.Metric, startAt.Unix(), endAt.Unix())
			if err != nil {
				log.Printf("[debug] FetchHostMetricValues failed: %s", err)
				log.Printf("[warn] monitor `%s`, can not get host metric = `%s`, reliability reassessment based on metric is not enabled.", monitor.Name, monitor.Metric)
				return nil, false
			}
			isNoViolation := make(IsNoViolationCollection, endAt.Sub(startAt)/time.Minute)
			for _, metric := range metrics {
				cursorAt := time.Unix(metric.Time, 0).UTC()
				value, ok := metric.Value.(float64)
				if !ok {
					continue
				}
				switch monitor.Operator {
				case ">":
					if monitor.Warning != nil {
						if value > *monitor.Warning {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, host_id=`%s`, time=`%s`,  value[%f] > warning[%f]", monitor.Name, hostID, cursorAt, value, *monitor.Warning)
							continue
						}
					}
					if monitor.Critical != nil {
						if value > *monitor.Critical {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, hostId=`%s`, time=`%s`,  value[%f] > critical[%f]", monitor.Name, hostID, cursorAt, value, *monitor.Critical)
							continue
						}
					}
				case "<":
					if monitor.Warning != nil {
						if value < *monitor.Warning {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, hostId=`%s`, time=`%s`,  value[%f] < warning[%f]", monitor.Name, hostID, cursorAt, value, *monitor.Warning)
							continue
						}
					}
					if monitor.Critical != nil {
						if value < *monitor.Critical {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, hostId=`%s`, time=`%s`,  value[%f] < critical[%f]", monitor.Name, hostID, cursorAt, value, *monitor.Warning)
							continue
						}
					}
				default:
					log.Printf("[warn] monitor `%s`, unknown operator `%s`, reliability reassessment based on metric is not enabled.", monitor.Name, monitor.Operator)
					return nil, false
				}
			}
			reliabilities, err := isNoViolation.NewReliabilities(timeFrame, startAt, endAt)
			if err != nil {
				log.Printf("[debug] NewReliabilities failed: %s", err)
				log.Printf("[warn] monitor `%s`, reliability reassessment based on metric is not enabled.", monitor.Name)
				return nil, false
			}
			return reliabilities, true
		})
	}
	return m
}

func (repo *Repository) WithDryRun() *Repository {
	return &Repository{
		client: DryRunMackerelClient{
			MackerelClient: repo.client,
		},
		monitorByID: repo.monitorByID,
	}
}

type DryRunMackerelClient struct {
	MackerelClient
}

func (c DryRunMackerelClient) PostServiceMetricValues(serviceName string, metricValues []*mackerel.MetricValue) error {
	for _, value := range metricValues {
		log.Printf("[notice] **DRY RUN** action=PostServiceMetricValue, service=`%s`, metricName=`%s`, time=`%s`, value=`%f` ", serviceName, value.Name, time.Unix(value.Time, 0).UTC(), value.Value)
	}
	return nil
}

func (c DryRunMackerelClient) CreateDashboard(param *mackerel.Dashboard) (*mackerel.Dashboard, error) {
	dashboard, err := dashboardToString(param)
	if err != nil {
		return nil, err
	}
	log.Printf("[notice] **DRY RUN** action=CreateDashboard, dashboard=%s", dashboard)
	return param, nil
}

func (c DryRunMackerelClient) UpdateDashboard(dashboardID string, param *mackerel.Dashboard) (*mackerel.Dashboard, error) {
	dashboard, err := dashboardToString(param)
	if err != nil {
		return nil, err
	}
	log.Printf("[notice] **DRY RUN** action=UpdateDashboard, dashboard_id=`%s`, dashboard=%s", dashboardID, dashboard)
	return param, nil
}

func dashboardToString(param *mackerel.Dashboard) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(param); err != nil {
		return "", err
	}
	return buf.String(), nil
}
