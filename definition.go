package shimesaba

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"
)

// Definition is SLO Definition
type Definition struct {
	id              string
	destination     *Destination
	rollingPeriod   time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	alertBasedSLIs []*AlertBasedSLI
}

// NewDefinition creates Definition from SLOConfig
func NewDefinition(cfg *SLOConfig) (*Definition, error) {
	AlertBasedSLIs := make([]*AlertBasedSLI, 0, len(cfg.AlertBasedSLI))
	for _, cfg := range cfg.AlertBasedSLI {
		AlertBasedSLIs = append(AlertBasedSLIs, NewAlertBasedSLI(cfg))
	}
	return &Definition{
		id:              cfg.ID,
		destination:     NewDestination(cfg.Destination),
		rollingPeriod:   cfg.DurationRollingPeriod(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSizePercentage(),
		alertBasedSLIs:  AlertBasedSLIs,
	}, nil
}

// ID returns SLOConfig.id
func (d *Definition) ID() string {
	return d.id
}

type DataProvider interface {
	FetchAlerts(ctx context.Context, startAt time.Time, endAt time.Time) (Alerts, error)
	FetchVirtualAlerts(ctx context.Context, serviceName string, sloID string, startAt time.Time, endAt time.Time) (Alerts, error)
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, provider DataProvider, now time.Time, backfill int) ([]*Report, error) {
	startAt := d.StartAt(now, backfill)
	alerts, err := provider.FetchAlerts(ctx, startAt, now)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %w", err)
	}
	log.Printf("[debug] get %d alerts", len(alerts))
	valerts, err := provider.FetchVirtualAlerts(ctx, d.destination.ServiceName, d.id, startAt, now)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch virtual alerts: %w", err)
	}
	log.Printf("[debug] get %d virtual alerts", len(valerts))
	alerts = append(alerts, valerts...)
	reports, err := d.CreateReportsWithAlertsAndPeriod(ctx, alerts, startAt, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create reports: %w", err)
	}
	return reports, nil
}

func (d *Definition) CreateReportsWithAlertsAndPeriod(ctx context.Context, alerts Alerts, startAt, endAt time.Time) ([]*Report, error) {
	log.Printf("[debug] original report range = %s ~ %s", startAt, endAt)
	startAt = startAt.Truncate(d.calculate)
	endAt = endAt.Add(+time.Nanosecond).Truncate(d.calculate).Add(-time.Nanosecond)
	log.Printf("[debug] truncate report range = %s ~ %s", startAt, endAt)
	log.Printf("[debug] timeFrame = %s, calculateInterval = %s", d.rollingPeriod, d.calculate)
	var Reliabilities Reliabilities
	log.Printf("[debug] alert based SLI count = %d", len(d.alertBasedSLIs))
	for i, o := range d.alertBasedSLIs {
		rc, err := o.EvaluateReliabilities(d.calculate, alerts, startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate reliabilities for alert_based_sli[%d]: %w", i, err)
		}
		Reliabilities, err = Reliabilities.Merge(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to merge reliabilities for alert_based_sli[%d]: %w", i, err)
		}
	}
	for _, r := range Reliabilities {
		log.Printf("[debug] reliability[%s~%s] =  (%s, %s)", r.TimeFrameStartAt(), r.TimeFrameEndAt(), r.UpTime(), r.FailureTime())
	}
	reports := NewReports(d.id, d.destination, d.errorBudgetSize, d.rollingPeriod, Reliabilities)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].DataPoint.Before(reports[j].DataPoint)
	})
	log.Printf("[debug] created %d reports", len(reports))
	return reports, nil
}

func (d *Definition) AlertBasedSLIs(monitors []*Monitor) []*Monitor {
	matched := make(map[string]*Monitor)
	for _, m := range monitors {
		for _, obj := range d.alertBasedSLIs {
			if obj.MatchMonitor(m) {
				matched[m.ID()] = m
			}
		}
	}
	objectiveMonitors := make([]*Monitor, 0, len(matched))
	for _, monitor := range matched {
		objectiveMonitors = append(objectiveMonitors, monitor)
	}
	return objectiveMonitors
}

func (d *Definition) StartAt(now time.Time, backfill int) time.Time {
	return now.Truncate(d.calculate).Add(-(time.Duration(backfill) * d.calculate) - d.rollingPeriod)
}
