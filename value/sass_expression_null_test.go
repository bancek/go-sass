package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNullExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("null"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewNullExpression(span)

	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestNullExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("null"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewNullExpression(span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "null" {
		t.Errorf("String() = %q, want 'null'", got)
	}
}

func TestNullExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("null"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewNullExpression(span)

	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
