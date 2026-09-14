package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestSelectorExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("&"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	expr := NewSelectorExpression(span)

	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestSelectorExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("&"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	expr := NewSelectorExpression(span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "&" {
		t.Errorf("String() = %q, want '&'", got)
	}
}

func TestSelectorExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("&"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	expr := NewSelectorExpression(span)

	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
