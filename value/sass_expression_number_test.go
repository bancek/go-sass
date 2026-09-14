package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNumberExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42px"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	unit := "px"
	expr := NewNumberExpression(42, span, &unit)

	if expr.Value != 42 {
		t.Errorf("Value = %v, want 42", expr.Value)
	}
	if *expr.Unit != "px" {
		t.Errorf("Unit = %v, want 'px'", *expr.Unit)
	}
}

func TestNumberExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42px"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	unit := "px"
	expr := NewNumberExpression(42, span, &unit)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "42px" {
		t.Errorf("String() = %q, want '42px'", got)
	}
}

func TestNumberExpressionUnitless(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	expr := NewNumberExpression(42, span, nil)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("String() = %q, want '42'", got)
	}
}

func TestNumberExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("42"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	expr := NewNumberExpression(42, span, nil)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
