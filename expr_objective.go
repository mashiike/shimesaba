package shimesaba

import (
	"log"
	"time"

	"github.com/mashiike/evaluator"
	"github.com/mashiike/shimesaba/internal/timeutils"
)

type ExprObjective struct {
	expr evaluator.Comparator
}

func NewExprObjective(expr evaluator.Comparator) *ExprObjective {
	return &ExprObjective{expr: expr}
}

func (o *ExprObjective) NewReliabilityCollection(timeFrame time.Duration, metrics Metrics, startAt, endAt time.Time) (ReliabilityCollection, error) {
	isNoViolation := o.newIsNoViolation(metrics)
	iter := timeutils.NewIterator(startAt, endAt, timeFrame)
	iter.SetEnableOverWindow(true)
	reliabilitySlice := make([]*Reliability, 0)
	for iter.HasNext() {
		cursorAt, _ := iter.Next()
		reliabilitySlice = append(reliabilitySlice, NewReliability(cursorAt, timeFrame, isNoViolation))
	}
	return NewReliabilityCollection(reliabilitySlice)
}

func (o *ExprObjective) newIsNoViolation(metrics Metrics) map[time.Time]bool {
	variables := metrics.GetVariables(metrics.StartAt(), metrics.EndAt())
	ret := make(map[time.Time]bool, len(variables))
	for t, v := range variables {
		b, err := o.expr.Compare(v)
		if evaluator.IsDivideByZero(err) {
			continue
		}
		if err != nil {
			log.Printf("[warn] compare failed expr=%s time=%s reason=%s", o.expr.String(), t, err)
			continue
		}
		ret[t] = b
	}
	return ret
}

func (o *ExprObjective) String() string {
	return o.expr.String()
}
