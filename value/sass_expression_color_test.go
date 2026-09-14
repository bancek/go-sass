package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestColorExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("#f00"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	color := &SassColor{}
	expr := NewColorExpression(color, span)

	if expr.Value != color {
		t.Error("Value should be the same color")
	}
	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestColorExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("red"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 3)
	color := &SassColor{}
	expr := NewColorExpression(color, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
