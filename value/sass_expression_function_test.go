package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestFunctionExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("fn()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	args := NewArgumentListEmpty(span)
	expr := NewFunctionExpression("fn", args, span, nil)

	if expr.OriginalName != "fn" {
		t.Errorf("OriginalName = %q, want 'fn'", expr.OriginalName)
	}
	if expr.Name != "fn" {
		t.Errorf("Name = %q, want 'fn'", expr.Name)
	}
}

func TestFunctionExpressionUnderscoreNormalization(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("fn_name()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 10)
	args := NewArgumentListEmpty(span)
	expr := NewFunctionExpression("fn_name", args, span, nil)

	if expr.Name != "fn-name" {
		t.Errorf("Name = %q, want 'fn-name'", expr.Name)
	}
}

func TestFunctionExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("fn()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	args := NewArgumentListEmpty(span)
	expr := NewFunctionExpression("fn", args, span, nil)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "fn()" {
		t.Errorf("String() = %q, want 'fn()'", got)
	}
}

func TestFunctionExpressionWithNamespace(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("ns.fn()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 7)
	args := NewArgumentListEmpty(span)
	ns := "ns"
	expr := NewFunctionExpression("ns.fn", args, span, &ns)

	if expr.Namespace == nil || *expr.Namespace != "ns" {
		t.Errorf("Namespace = %v, want 'ns'", expr.Namespace)
	}
}

func TestFunctionExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("fn()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	args := NewArgumentListEmpty(span)
	expr := NewFunctionExpression("fn", args, span, nil)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
