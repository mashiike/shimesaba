package shimesaba

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strconv"
	"time"
)

// MetricComparator is a comparison using multiple metrics
type MetricComparator struct {
	leftExpr        metricExpr
	comparativeFunc func(float64, float64) bool
	rightValue      float64
}

// Reserved errors
var (
	ErrNotComparativeExpression = errors.New("this expr is not comparative")
	ErrExprRightNotLiteral      = errors.New("expr right side is value literal")
)

//NewMetricComparator creates MetricComparator from expr string
func NewMetricComparator(str string) (*MetricComparator, error) {
	expr, err := parser.ParseExpr(str)
	if err != nil {
		return nil, err
	}
	bexpr, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return nil, ErrNotComparativeExpression
	}
	comparativeFunc, ok := getComparativeFunc(bexpr.Op)
	if !ok {
		return nil, ErrNotComparativeExpression
	}
	rightValue, ok := parseAsFloat(bexpr.Y)
	if !ok {
		return nil, ErrExprRightNotLiteral
	}
	leftExpr, err := parseAsMtricExpr(bexpr.X)
	if err != nil {
		return nil, err
	}
	metricComparator := &MetricComparator{
		leftExpr:        leftExpr,
		comparativeFunc: comparativeFunc,
		rightValue:      rightValue,
	}
	return metricComparator, nil
}

// Eval performs a comparison
func (mc MetricComparator) Eval(metrics Metrics, startAt, endAt time.Time) (map[time.Time]bool, error) {
	values, err := mc.leftExpr.Eval(metrics, startAt, endAt)
	if err != nil {
		return nil, err
	}
	ret := make(map[time.Time]bool, len(values))
	for t, v := range values {
		ret[t] = mc.comparativeFunc(v, mc.rightValue)
	}
	return ret, nil
}

func getComparativeFunc(op token.Token) (func(float64, float64) bool, bool) {
	switch op {
	case token.EQL, token.ASSIGN: // == or =
		return func(f1, f2 float64) bool { return f1 == f2 }, true
	case token.LSS: // <
		return func(f1, f2 float64) bool { return f1 < f2 }, true
	case token.GTR: // >
		return func(f1, f2 float64) bool { return f1 > f2 }, true
	case token.NEQ: // >
		return func(f1, f2 float64) bool { return f1 != f2 }, true
	case token.LEQ: // <=
		return func(f1, f2 float64) bool { return f1 <= f2 }, true
	case token.GEQ: // >=
		return func(f1, f2 float64) bool { return f1 >= f2 }, true
	default:
		return nil, false
	}
}

func parseAsFloat(expr ast.Expr) (float64, bool) {
	blit, ok := expr.(*ast.BasicLit)
	if !ok {
		return 0.0, false
	}
	switch blit.Kind {
	case token.INT:
		v, err := strconv.ParseInt(blit.Value, 10, 64)
		if err != nil {
			log.Printf("[debug] internal parseAsFloat(%s) :%s", blit.Value, err)
			return 0.0, false
		}
		return float64(v), true
	case token.FLOAT:
		v, err := strconv.ParseFloat(blit.Value, 64)
		if err != nil {
			log.Printf("[debug] internal parseAsFloat(%s) :%s", blit.Value, err)
			return 0.0, false
		}
		return v, true
	default:
		return 0.0, false
	}
}

type metricExpr interface {
	Eval(metrics Metrics, startAt, endAt time.Time) (map[time.Time]float64, error)
}

type metricRefExpr string

func (e metricRefExpr) Eval(metrics Metrics, startAt, endAt time.Time) (map[time.Time]float64, error) {
	metric, ok := metrics.Get(string(e))
	if !ok {
		return nil, fmt.Errorf("metric `%s` not found", e)
	}
	return metric.GetValues(startAt, endAt), nil
}

func parseAsMtricExpr(expr ast.Expr) (metricExpr, error) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return parseAsFuncExpr(e)
	case *ast.Ident:
		return metricRefExpr(e.Name), nil
	case *ast.BinaryExpr:
		return parseAsBinalyOpExpr(e)
	default:
		return nil, fmt.Errorf("expr(%s) unknown expr type %T", expr, expr)
	}
}

type metricRateFuncExpr struct {
	Numerator   metricExpr
	Denominator metricExpr
}

func parseAsFuncExpr(expr *ast.CallExpr) (metricExpr, error) {
	funcID, ok := expr.Fun.(*ast.Ident)
	if !ok {
		return nil, errors.New("unknown func id")
	}
	args := make([]metricExpr, 0, len(expr.Args))
	for i, arg := range expr.Args {
		m, err := parseAsMtricExpr(arg)
		if err != nil {
			return nil, fmt.Errorf("%s() arg%d: %w", funcID.Name, i, err)
		}
		args = append(args, m)
	}
	switch funcID.Name {
	case "rate":
		if len(args) != 2 {
			return nil, fmt.Errorf("func_id:rate expected 2 args, provided %d args", len(args))
		}
		return &metricRateFuncExpr{
			Numerator:   args[0],
			Denominator: args[1],
		}, nil
	default:
		return nil, fmt.Errorf("func_id:%s is not implemented", funcID.Name)
	}

}

func (e *metricRateFuncExpr) Eval(metrics Metrics, startAt, endAt time.Time) (map[time.Time]float64, error) {
	numerator, err := e.Numerator.Eval(metrics, startAt, endAt)
	if err != nil {
		return nil, err
	}
	denominator, err := e.Denominator.Eval(metrics, startAt, endAt)
	if err != nil {
		return nil, err
	}
	ret := make(map[time.Time]float64, len(denominator))
	for t, vd := range denominator {
		if vd == 0.0 {
			continue
		}
		vn, ok := numerator[t]
		if !ok {
			continue
		}
		ret[t] = vn / vd
	}

	return ret, nil
}

type metricBinalyOpExpr struct {
	Left   metricExpr
	Right  metricExpr
	OpFunc func(float64, float64) float64
}

func parseAsBinalyOpExpr(expr *ast.BinaryExpr) (metricExpr, error) {
	left, err := parseAsMtricExpr(expr.X)
	if err != nil {
		return nil, err
	}
	right, err := parseAsMtricExpr(expr.Y)
	if err != nil {
		return nil, err
	}
	var opFunc func(float64, float64) float64
	switch expr.Op {
	case token.ADD: // +
		opFunc = func(f1, f2 float64) float64 { return f1 + f2 }
	case token.SUB: // -
		opFunc = func(f1, f2 float64) float64 { return f1 - f2 }
	case token.MUL: // *
		opFunc = func(f1, f2 float64) float64 { return f1 * f2 }
	case token.QUO: // /
		return &metricRateFuncExpr{
			Numerator:   left,
			Denominator: right,
		}, nil
	default:
		return nil, fmt.Errorf("parseAsBinalyOpExpr unknown op token %s", expr.Op)
	}
	e := &metricBinalyOpExpr{
		Left:   left,
		Right:  right,
		OpFunc: opFunc,
	}
	return e, nil
}

func (e *metricBinalyOpExpr) Eval(metrics Metrics, startAt, endAt time.Time) (map[time.Time]float64, error) {
	leftValue, err := e.Left.Eval(metrics, startAt, endAt)
	if err != nil {
		return nil, err
	}
	rightValue, err := e.Right.Eval(metrics, startAt, endAt)
	if err != nil {
		return nil, err
	}
	ret := make(map[time.Time]float64, len(leftValue))
	for t, vl := range leftValue {
		vr, ok := rightValue[t]
		if !ok {
			continue
		}
		ret[t] = e.OpFunc(vl, vr)
	}

	return ret, nil
}
