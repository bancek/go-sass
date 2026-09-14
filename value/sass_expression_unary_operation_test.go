package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestUnaryOperationExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("-1"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	inner := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 1, 2), nil)
	expr := NewUnaryOperationExpression(UnaryOperatorMinus, inner, span)

	if expr.Operator != UnaryOperatorMinus {
		t.Errorf("Operator = %v, want Minus", expr.Operator)
	}
}

func TestUnaryOperationExpressionStringMinus(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("-1"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	inner := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 1, 2), nil)
	expr := NewUnaryOperationExpression(UnaryOperatorMinus, inner, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "-1" {
		t.Errorf("String() = %q, want '-1'", got)
	}
}

func TestUnaryOperationExpressionStringPlus(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("+1"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	inner := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 1, 2), nil)
	expr := NewUnaryOperationExpression(UnaryOperatorPlus, inner, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "+1" {
		t.Errorf("String() = %q, want '+1'", got)
	}
}

func TestUnaryOperationExpressionStringNot(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("not true"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 8)
	inner := NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 4, 8))
	expr := NewUnaryOperationExpression(UnaryOperatorNot, inner, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "not true" {
		t.Errorf("String() = %q, want 'not true'", got)
	}
}

func TestUnaryOperationExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("-1"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	inner := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 1, 2), nil)
	expr := NewUnaryOperationExpression(UnaryOperatorMinus, inner, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
