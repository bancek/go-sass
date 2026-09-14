package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestParenthesizedExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(true)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)

	inner := NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 5))
	expr := NewParenthesizedExpression(inner, span)

	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestParenthesizedExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(true)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)

	inner := NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 5))
	expr := NewParenthesizedExpression(inner, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(true)" {
		t.Errorf("String() = %q, want '(true)'", got)
	}
}

func TestParenthesizedExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(x)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 3)

	inner := NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 2))
	expr := NewParenthesizedExpression(inner, span)

	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
