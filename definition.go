package shimesaba

import (
	"context"
	"sort"
	"time"
)

//Definition is SLI/SLO Definition
type Definition struct {
	id              string
	serviceName     string
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
		id:              cfg.ID,
		serviceName:     cfg.ServiceName,
		timeFrame:       cfg.DurationTimeFrame(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSize,
		exprObjectives:  exprObjectives,
		alertObjectives: alertObjectives,
	}, nil
}

// ID returns DefinitionConfig.id
func (d *Definition) ID() string {
	return d.id
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, metrics Metrics, alerts Alerts) ([]*Report, error) {
	var reliabilityCollection ReliabilityCollection
	for _, o := range d.exprObjectives {
		rc, err := o.NewReliabilityCollection(d.calculate, metrics)
		if err != nil {
			return nil, err
		}
		reliabilityCollection, err = reliabilityCollection.Merge(rc)
		if err != nil {
			return nil, err
		}
	}
	for _, o := range d.alertObjectives {
		rc, err := o.NewReliabilityCollection(d.calculate, alerts)
		if err != nil {
			return nil, err
		}
		reliabilityCollection, err = reliabilityCollection.Merge(rc)
		if err != nil {
			return nil, err
		}
	}
	reports := NewReports(d.id, d.serviceName, d.errorBudgetSize, d.timeFrame, reliabilityCollection)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].DataPoint.Before(reports[j].DataPoint)
	})
	return reports, nil
}

func (d *Definition) ExprObjectives() []string {
	objectives := make([]string, 0, len(d.exprObjectives))
	for _, obj := range d.exprObjectives {
		objectives = append(objectives, obj.String())
	}
	return objectives
}
