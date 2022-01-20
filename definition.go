package shimesaba

import (
	"context"
	"log"
	"sort"
	"time"
)

//Definition is SLI/SLO Definition
type Definition struct {
	id              string
	destination     *Destination
	timeFrame       time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	exprObjectives  []*ExprObjective
	alertObjectives []*AlertObjective
}

//NewDefinition creates Definition from DefinitionConfig
func NewDefinition(cfg *DefinitionConfig) (*Definition, error) {
	exprObjectives := make([]*ExprObjective, 0, len(cfg.Objectives))
	alertObjectives := make([]*AlertObjective, 0, len(cfg.Objectives))
	for _, objCfg := range cfg.Objectives {
		switch objCfg.Type() {
		case "expr":
			exprObjectives = append(exprObjectives, NewExprObjective(objCfg.GetComparator()))
		case "alert":
			alertObjectives = append(alertObjectives, NewAlertObjective(objCfg.Alert))
		}
	}
	return &Definition{
		id: cfg.ID,
		destination: &Destination{
			ServiceName:  cfg.ServiceName,
			MetricPrefix: cfg.MetricPrefix,
			MetricSuffix: cfg.MetricSuffix,
		},
		timeFrame:       cfg.DurationTimeFrame(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSizeParcentage(),
		exprObjectives:  exprObjectives,
		alertObjectives: alertObjectives,
	}, nil
}

// ID returns DefinitionConfig.id
func (d *Definition) ID() string {
	return d.id
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, metrics Metrics, alerts Alerts, startAt, endAt time.Time) ([]*Report, error) {
	log.Printf("[debug] original report range = %s ~ %s", startAt, endAt)
	startAt = startAt.Truncate(d.calculate)
	endAt = endAt.Add(+time.Nanosecond).Truncate(d.calculate).Add(-time.Nanosecond)
	log.Printf("[debug] truncate report range = %s ~ %s", startAt, endAt)
	log.Printf("[debug] timeFrame = %s, calcurateInterval = %s", d.timeFrame, d.calculate)
	var Reliabilities Reliabilities
	log.Printf("[debug] expr objective count = %d", len(d.exprObjectives))
	for _, o := range d.exprObjectives {
		rc, err := o.NewReliabilities(d.calculate, metrics, startAt, endAt)
		if err != nil {
			return nil, err
		}
		Reliabilities, err = Reliabilities.Merge(rc)
		if err != nil {
			return nil, err
		}
	}
	log.Printf("[debug] alert objective count = %d", len(d.alertObjectives))
	for _, o := range d.alertObjectives {
		rc, err := o.NewReliabilities(d.calculate, alerts, startAt, endAt)
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
	reports := NewReports(d.id, d.destination, d.errorBudgetSize, d.timeFrame, Reliabilities)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].DataPoint.Before(reports[j].DataPoint)
	})
	log.Printf("[debug] created %d reports", len(reports))
	return reports, nil
}

func (d *Definition) ExprObjectives() []string {
	objectives := make([]string, 0, len(d.exprObjectives))
	for _, obj := range d.exprObjectives {
		objectives = append(objectives, obj.String())
	}
	return objectives
}

func (d *Definition) AlertObjectives(monitors []*Monitor) []*Monitor {
	matched := make(map[string]*Monitor)
	for _, m := range monitors {
		for _, obj := range d.alertObjectives {
			if obj.MatchMonitor(m) {
				matched[m.ID] = m
			}
		}
	}
	objectiveMonitors := make([]*Monitor, 0, len(matched))
	for _, monitor := range matched {
		objectiveMonitors = append(objectiveMonitors, monitor)
	}
	return objectiveMonitors
}
