package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestMapExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(a: b)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	pairs := []struct {
		Key   Expression
		Value Expression
	}{
		{NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 1, 2), false),
			NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 4, 5), false)},
	}
	expr := NewMapExpression(pairs, span)

	if len(expr.Pairs) != 1 {
		t.Errorf("len(Pairs) = %d, want 1", len(expr.Pairs))
	}
}

func TestMapExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(a: b)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	pairs := []struct {
		Key   Expression
		Value Expression
	}{
		{NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 1, 2), false),
			NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 4, 5), false)},
	}
	expr := NewMapExpression(pairs, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(a: b)" {
		t.Errorf("String() = %q, want '(a: b)'", got)
	}
}

func TestMapExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(a: b)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	expr := NewMapExpression(nil, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
