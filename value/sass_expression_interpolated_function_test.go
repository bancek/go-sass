package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestInterpolatedFunctionExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("#{$fn}()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 8)
	name := NewInterpolationPlain("fn", sasscommon.NewSimpleFileSpan(fs, 2, 4))
	args := NewArgumentListEmpty(span)
	expr := NewInterpolatedFunctionExpression(name, args, span)

	if expr.Name != name {
		t.Error("Name should be the same interpolation")
	}
}

func TestInterpolatedFunctionExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("#{$fn}()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 8)
	name := NewInterpolationPlain("fn", sasscommon.NewSimpleFileSpan(fs, 2, 4))
	args := NewArgumentListEmpty(span)
	expr := NewInterpolatedFunctionExpression(name, args, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "fn()" {
		t.Errorf("String() = %q, want 'fn()'", got)
	}
}

func TestInterpolatedFunctionExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 3)
	name := NewInterpolationPlain("x", span)
	args := NewArgumentListEmpty(span)
	expr := NewInterpolatedFunctionExpression(name, args, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
