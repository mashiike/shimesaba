package shimesaba

import (
	"log"
	"time"

	"github.com/mashiike/evaluator"
)

type ExprObjective struct {
	expr evaluator.Comparator
}

func NewExprObjective(expr evaluator.Comparator) *ExprObjective {
	return &ExprObjective{expr: expr}
}

func (o *ExprObjective) EvaluateReliabilities(timeFrame time.Duration, metrics Metrics, startAt, endAt time.Time) (Reliabilities, error) {
	isNoViolation := o.newIsNoViolation(metrics)
	return isNoViolation.NewReliabilities(timeFrame, startAt, endAt)
}

func (o *ExprObjective) newIsNoViolation(metrics Metrics) IsNoViolationCollection {
	variables := metrics.GetVariables(metrics.StartAt(), metrics.EndAt())
	ret := make(IsNoViolationCollection, len(variables))
	for t, v := range variables {
		b, err := o.expr.Compare(v)
		if evaluator.IsDivideByZero(err) {
			continue
		}
		if err != nil {
			log.Printf("[warn] compare failed expr=%s time=%s reason=%s", o.expr.String(), t, err)
			continue
		}
		if !b {
			log.Printf("[debug] SLO violation, expr=`%s`, time=`%s`", o.expr, t.UTC())
		}
		ret[t.UTC()] = b
	}
	return ret
}

func (o *ExprObjective) String() string {
	return o.expr.String()
}
