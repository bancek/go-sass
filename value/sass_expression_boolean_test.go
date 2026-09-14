package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestBooleanExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("true"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewBooleanExpression(true, span)

	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestBooleanExpressionTrueString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("true"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewBooleanExpression(true, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("String() = %q, want 'true'", got)
	}
}

func TestBooleanExpressionFalseString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("false"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 5)
	expr := NewBooleanExpression(false, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "false" {
		t.Errorf("String() = %q, want 'false'", got)
	}
}

func TestBooleanExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("true"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewBooleanExpression(true, span)

	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
