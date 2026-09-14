package value

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

func exprCalcSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("calc test"), nil)
	return sasscommon.NewFileSpan(fs, 0, 9)
}

func TestExpressionToCalcWrapsInCalc(t *testing.T) {
	span := exprCalcSpan()
	expr := NewNumberExpression(42, span, nil)
	fe, err := ExpressionToCalc(expr)
	if err != nil {
		t.Fatal(err)
	}
	if fe.OriginalName != "calc" {
		t.Errorf("expected calc, got %q", fe.OriginalName)
	}
	if len(fe.Arguments().Positional) != 1 {
		t.Errorf("expected 1 argument, got %d", len(fe.Arguments().Positional))
	}
}

func TestExpressionToCalcModulo(t *testing.T) {
	span := exprCalcSpan()
	left := NewNumberExpression(5, span, nil)
	right := NewNumberExpression(3, span, nil)
	modExpr := NewBinaryOperationExpression(BinaryOperatorModulo, left, right)
	fe, err := ExpressionToCalc(modExpr)
	if err != nil {
		t.Fatal(err)
	}
	if fe.OriginalName != "calc" {
		t.Errorf("expected calc, got %q", fe.OriginalName)
	}
	inner, ok := fe.Arguments().Positional[0].(*FunctionExpression)
	if !ok {
		t.Fatalf("expected *FunctionExpression, got %T", fe.Arguments().Positional[0])
	}
	if inner.OriginalName != "max" {
		t.Errorf("expected max, got %q", inner.OriginalName)
	}
	if inner.Namespace == nil || *inner.Namespace != "math" {
		t.Error("expected math namespace")
	}
}

func TestExpressionToCalcIfPassesThrough(t *testing.T) {
	span := exprCalcSpan()
	branches := []IfBranch{
		{
			Condition:  NewIfConditionSass(NewBooleanExpression(true, span), span),
			Expression: NewNumberExpression(1, span, nil),
		},
		{
			Expression: NewNumberExpression(2, span, nil),
		},
	}
	ifExpr, err := NewIfExpression(branches, span)
	if err != nil {
		t.Fatal(err)
	}
	fe, err := ExpressionToCalc(ifExpr)
	if err != nil {
		t.Fatal(err)
	}
	inner, ok := fe.Arguments().Positional[0].(*IfExpression)
	if !ok {
		t.Fatalf("expected *IfExpression, got %T", fe.Arguments().Positional[0])
	}
	if len(inner.Branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(inner.Branches))
	}
}

func TestExpressionToCalcUnaryPlusStrips(t *testing.T) {
	span := exprCalcSpan()
	num := NewNumberExpression(5, span, nil)
	unaryPlus := NewUnaryOperationExpression(UnaryOperatorPlus, num, span)
	fe, err := ExpressionToCalc(unaryPlus)
	if err != nil {
		t.Fatal(err)
	}
	inner, ok := fe.Arguments().Positional[0].(*NumberExpression)
	if !ok {
		t.Fatalf("expected *NumberExpression, got %T", fe.Arguments().Positional[0])
	}
	if inner.Value != 5 {
		t.Errorf("expected 5, got %f", inner.Value)
	}
}

func TestExpressionToCalcUnaryMinus(t *testing.T) {
	span := exprCalcSpan()
	num := NewNumberExpression(5, span, nil)
	unaryMinus := NewUnaryOperationExpression(UnaryOperatorMinus, num, span)
	fe, err := ExpressionToCalc(unaryMinus)
	if err != nil {
		t.Fatal(err)
	}
	bin, ok := fe.Arguments().Positional[0].(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected *BinaryOperationExpression, got %T", fe.Arguments().Positional[0])
	}
	if bin.Operator != BinaryOperatorTimes {
		t.Errorf("expected times operator, got %v", bin.Operator)
	}
	left, ok := bin.Left.(*NumberExpression)
	if !ok {
		t.Fatalf("expected left *NumberExpression, got %T", bin.Left)
	}
	if left.Value != -1 {
		t.Errorf("expected left -1, got %f", left.Value)
	}
}

func TestExpressionToCalcInterpolatedFunctionPassesThrough(t *testing.T) {
	span := exprCalcSpan()
	name := NewInterpolationPlain("fn", span)
	args := NewArgumentList(nil, orderedmap.New[string, Expression](), nil, span, nil, nil)
	ifn := NewInterpolatedFunctionExpression(name, args, span)
	fe, err := ExpressionToCalc(ifn)
	if err != nil {
		t.Fatal(err)
	}
	inner, ok := fe.Arguments().Positional[0].(*InterpolatedFunctionExpression)
	if !ok {
		t.Fatalf("expected *InterpolatedFunctionExpression, got %T", fe.Arguments().Positional[0])
	}
	_ = inner
}
