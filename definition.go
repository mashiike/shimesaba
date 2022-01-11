package shimesaba

import (
	"context"
	"sort"
	"time"

	"github.com/mashiike/evaluator"
)

//Definition is SLI/SLO Definition
type Definition struct {
	id              string
	serviceName     string
	timeFrame       time.Duration
	calculate       time.Duration
	errorBudgetSize float64

	exprObjectives []*ExprObjective
	objectives     []evaluator.Comparator
}

//NewDefinition creates Definition from DefinitionConfig
func NewDefinition(cfg *DefinitionConfig) (*Definition, error) {
	exprObjectives := make([]*ExprObjective, 0, len(cfg.Objectives))
	objectives := make([]evaluator.Comparator, 0, len(cfg.Objectives))
	for _, objCfg := range cfg.Objectives {
		exprObjectives = append(exprObjectives, NewExprObjective(objCfg.GetComparator()))
		objectives = append(objectives, objCfg.GetComparator())
	}
	return &Definition{
		id:              cfg.ID,
		serviceName:     cfg.ServiceName,
		timeFrame:       cfg.DurationTimeFrame(),
		calculate:       cfg.DurationCalculate(),
		errorBudgetSize: cfg.ErrorBudgetSize,
		objectives:      objectives,
		exprObjectives:  exprObjectives,
	}, nil
}

// ID returns DefinitionConfig.id
func (d *Definition) ID() string {
	return d.id
}

// CreateReports returns Report with Metrics
func (d *Definition) CreateReports(ctx context.Context, metrics Metrics) ([]*Report, error) {
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
	reports := NewReports(d.id, d.serviceName, d.errorBudgetSize, d.timeFrame, reliabilityCollection)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].DataPoint.Before(reports[j].DataPoint)
	})
	return reports, nil
}
