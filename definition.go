package shimesaba

import (
	"context"
	"log"
	"sort"
	"time"
)

//Definition is SLO Definition
type Definition struct {
	id              string
	destination     *Destination
	rollingPeriod   time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	alertObjectives []*AlertObjective
}

//NewDefinition creates Definition from SLOConfig
func NewDefinition(cfg *SLOConfig) (*Definition, error) {
	alertObjectives := make([]*AlertObjective, 0, len(cfg.AlertBasedSLI))
	for _, cfg := range cfg.AlertBasedSLI {
		alertObjectives = append(alertObjectives, NewAlertObjective(cfg))
	}
	return &Definition{
		id: cfg.ID,
		destination: &Destination{
			ServiceName:  cfg.Destination.ServiceName,
			MetricPrefix: cfg.Destination.MetricPrefix,
			MetricSuffix: cfg.Destination.MetricSuffix,
		},
		rollingPeriod:   cfg.DurationRollingPeriod(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSizeParcentage(),
		alertObjectives: alertObjectives,
	}, nil
}

// ID returns SLOConfig.id
func (d *Definition) ID() string {
	return d.id
}

type DataProvider interface {
	FetchAlerts(ctx context.Context, startAt time.Time, endAt time.Time) (Alerts, error)
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, provider DataProvider, now time.Time, backfill int) ([]*Report, error) {
	startAt := d.StartAt(now, backfill)
	alerts, err := provider.FetchAlerts(ctx, startAt, now)
	if err != nil {
		return nil, err
	}
	return d.CreateReportsWithAlertsAndPeriod(ctx, alerts, d.StartAt(now, backfill), now)
}

func (d *Definition) CreateReportsWithAlertsAndPeriod(ctx context.Context, alerts Alerts, startAt, endAt time.Time) ([]*Report, error) {
	log.Printf("[debug] original report range = %s ~ %s", startAt, endAt)
	startAt = startAt.Truncate(d.calculate)
	endAt = endAt.Add(+time.Nanosecond).Truncate(d.calculate).Add(-time.Nanosecond)
	log.Printf("[debug] truncate report range = %s ~ %s", startAt, endAt)
	log.Printf("[debug] timeFrame = %s, calcurateInterval = %s", d.rollingPeriod, d.calculate)
	var Reliabilities Reliabilities
	log.Printf("[debug] alert objective count = %d", len(d.alertObjectives))
	for _, o := range d.alertObjectives {
		rc, err := o.EvaluateReliabilities(d.calculate, alerts, startAt, endAt)
		if err != nil {
			return nil, err
		}
		Reliabilities, err = Reliabilities.Merge(rc)
		if err != nil {
			return nil, err
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

func (d *Definition) AlertObjectives(monitors []*Monitor) []*Monitor {
	matched := make(map[string]*Monitor)
	for _, m := range monitors {
		for _, obj := range d.alertObjectives {
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
