package shimesaba

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	mackerel "github.com/mackerelio/mackerel-client-go"
	retry "github.com/shogo82148/go-retry"
)

// MackerelClient is an abstraction interface for mackerel-client-go.Client
type MackerelClient interface {
	GetOrg() (*mackerel.Org, error)
	FindHosts(param *mackerel.FindHostsParam) ([]*mackerel.Host, error)
	FetchHostMetricValues(hostID string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
	FetchServiceMetricValues(serviceName string, metricName string, from int64, to int64) ([]mackerel.MetricValue, error)
	PostServiceMetricValues(serviceName string, metricValues []*mackerel.MetricValue) error

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

	alertMu        sync.Mutex
	alertCache     Alerts
	alertCurrentAt time.Time
	alertNextID    string
}

// NewRepository creates Repository
func NewRepository(client MackerelClient) *Repository {
	return &Repository{
		client:      client,
		monitorByID: make(map[string]*Monitor),
		alertCache:  make(Alerts, 0, 100),
	}
}

func (repo *Repository) GetOrgName(ctx context.Context) (string, error) {
	org, err := repo.client.GetOrg()
	if err != nil {
		return "", err
	}
	return org.Name, nil
}

const (
	fetchMetricMetricmit = 6 * time.Hour
)

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

// FetchAlerts retrieves alerts for a specified period of time
func (repo *Repository) FetchAlerts(ctx context.Context, startAt time.Time, endAt time.Time) (Alerts, error) {
	repo.alertMu.Lock()
	defer repo.alertMu.Unlock()

	if len(repo.alertCache) == 0 {
		if err := repo.fetchAlertsInitial(ctx); err != nil {
			return nil, err
		}
	}
	for startAt.Before(repo.alertCurrentAt) && repo.alertNextID != "" {
		if err := repo.fetchAlertsIncremental(ctx); err != nil {
			return nil, err
		}
	}
	alerts := make(Alerts, 0, 100)
	for _, alert := range repo.alertCache {
		if alert.OpenedAt.After(endAt) {
			continue
		}
		if alert.OpenedAt.Before(startAt) {
			break
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (repo *Repository) fetchAlertsInitial(ctx context.Context) error {
	log.Printf("[debug] call MackerelClient.FindWithClosedAlerts()")
	resp, err := repo.client.FindWithClosedAlerts()
	if err != nil {
		return err
	}
	converted, err := repo.convertAlerts(resp)
	if err != nil {
		return err
	}
	repo.alertCache = append(repo.alertCache, converted...)
	currentAt := flextime.Now()
	if len(repo.alertCache) != 0 {
		currentAt = repo.alertCache[len(repo.alertCache)-1].OpenedAt
	}
	repo.alertCurrentAt = currentAt
	repo.alertNextID = resp.NextID
	return nil
}

func (repo *Repository) fetchAlertsIncremental(ctx context.Context) error {
	log.Printf("[debug] call MackerelClient.FindWithClosedAlertsByNextID(%s)", repo.alertNextID)
	resp, err := repo.client.FindWithClosedAlertsByNextID(repo.alertNextID)
	if err != nil {
		return err
	}
	converted, err := repo.convertAlerts(resp)
	if err != nil {
		return err
	}
	repo.alertCache = append(repo.alertCache, converted...)

	if len(converted) != 0 {
		repo.alertCurrentAt = converted[len(converted)-1].OpenedAt
		repo.alertNextID = resp.NextID
	}
	return nil
}

func (repo *Repository) convertAlerts(resp *mackerel.AlertsResp) ([]*Alert, error) {
	alerts := make([]*Alert, 0, len(resp.Alerts))
	for _, alert := range resp.Alerts {
		if alert.MonitorID == "" {
			continue
		}
		openedAt := time.Unix(alert.OpenedAt, 0)
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
		a = a.WithHostID(alert.HostID).WithReason(alert.Reason)
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
	case *mackerel.MonitorServiceMetric:
		m = m.WithEvaluator(func(_ string, timeFrame time.Duration, startAt, endAt time.Time) (Reliabilities, bool) {
			log.Printf("[debug] try evaluate service metric, service=%s monitor=`%s` time=%s~%s", monitor.Service, monitor.Name, startAt, endAt)
			metrics, err := repo.client.FetchServiceMetricValues(monitor.Service, monitor.Metric, startAt.Unix(), endAt.Unix())
			if err != nil {
				log.Printf("[debug] FetchServiceMetricValues failed: %s", err)
				log.Printf("[warn] monitor `%s`, can not get service metric = `%s`, reliability reassessment based on metric is not enabled.", monitor.Name, monitor.Metric)
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
							log.Printf("[debug] monitor `%s`, SLO Violation, service=`%s`, time=`%s`,  value[%f] > warning[%f]", monitor.Name, monitor.Service, cursorAt, value, *monitor.Warning)
							continue
						}
					}
					if monitor.Critical != nil {
						if value > *monitor.Critical {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, service=`%s`, time=`%s`,  value[%f] > critical[%f]", monitor.Name, monitor.Service, cursorAt, value, *monitor.Critical)
							continue
						}
					}
				case "<":
					if monitor.Warning != nil {
						if value < *monitor.Warning {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, service=`%s`, time=`%s`,  value[%f] < warning[%f]", monitor.Name, monitor.Service, cursorAt, value, *monitor.Warning)
							continue
						}
					}
					if monitor.Critical != nil {
						if value < *monitor.Critical {
							isNoViolation[cursorAt] = false
							log.Printf("[debug] monitor `%s`, SLO Violation, service=`%s`, time=`%s`,  value[%f] < critical[%f]", monitor.Name, monitor.Service, cursorAt, value, *monitor.Warning)
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
