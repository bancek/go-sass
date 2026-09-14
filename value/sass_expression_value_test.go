package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestValueExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42px"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	unit := "px"
	val := NewSassNumber(42, &unit)
	expr := NewValueExpression(val, span)

	if expr.Value != val {
		t.Error("Value should be the same")
	}
}

func TestValueExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42px"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	unit := "px"
	val := NewSassNumber(42, &unit)
	expr := NewValueExpression(val, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "42px" {
		t.Errorf("String() = %q, want '42px'", got)
	}
}

func TestValueExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	val := NewSassNumber(1, nil)
	expr := NewValueExpression(val, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
