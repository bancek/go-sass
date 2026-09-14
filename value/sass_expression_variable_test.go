package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestVariableExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$var"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewVariableExpression("var", span, nil)

	if expr.Name() != "var" {
		t.Errorf("Name() = %q, want 'var'", expr.Name())
	}
	if expr.Namespace() != nil {
		t.Error("Namespace() should be nil")
	}
	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestVariableExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$var"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewVariableExpression("var", span, nil)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$var" {
		t.Errorf("String() = %q, want '$var'", got)
	}
}

func TestVariableExpressionInterpolated(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a.#{$expr}.b"), nil)
	span := sasscommon.NewFileSpan(fs, 4, 9)
	expr := NewVariableExpression("expr", span, nil)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$expr" {
		t.Errorf("String() = %q, want '$expr'", got)
	}
}

func TestVariableExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$var"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewVariableExpression("var", span, nil)

	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
