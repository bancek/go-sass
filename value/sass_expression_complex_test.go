package value

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

func TestIfExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("if(true: 1; else: 2)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 20)

	branches := []IfBranch{
		{
			Condition:  NewIfConditionSass(NewBooleanExpression(true, span), span),
			Expression: NewNumberExpression(1, span, nil),
		},
		{
			Condition:  nil,
			Expression: NewNumberExpression(2, span, nil),
		},
	}
	expr, err := NewIfExpression(branches, span)
	if err != nil {
		t.Fatal(err)
	}

	if len(expr.Branches) != 2 {
		t.Errorf("len(Branches) = %d, want 2", len(expr.Branches))
	}
}

func TestIfExpressionEmptyBranchesError(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(""), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 0)
	_, err := NewIfExpression(nil, span)
	if err == nil {
		t.Fatal("expected error for empty branches")
	}
}

func TestIfExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	branches := []IfBranch{{
		Condition:  NewIfConditionSass(NewBooleanExpression(true, span), span),
		Expression: NewNumberExpression(1, span, nil),
	}}
	expr, err := NewIfExpression(branches, span)
	if err != nil {
		t.Fatal(err)
	}
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}

func TestLegacyIfExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("if(1, 2, 3)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 10)
	args := NewArgumentList(
		[]Expression{NewNumberExpression(1, span, nil), NewNumberExpression(2, span, nil), NewNumberExpression(3, span, nil)},
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil, nil,
	)
	expr := NewLegacyIfExpression(args, span)

	if expr.Arguments() != args {
		t.Error("Arguments() should return the same argument list")
	}
}

func TestLegacyIfExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("if(1, 2, 3)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 10)
	args := NewArgumentList(
		[]Expression{NewNumberExpression(1, span, nil), NewNumberExpression(2, span, nil), NewNumberExpression(3, span, nil)},
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil, nil,
	)
	expr := NewLegacyIfExpression(args, span)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "if(1, 2, 3)" {
		t.Errorf("String() = %q, want 'if(1, 2, 3)'", got)
	}
}

func TestLegacyIfExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	args := NewArgumentList(
		[]Expression{NewBooleanExpression(true, span)},
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil, nil,
	)
	expr := NewLegacyIfExpression(args, span)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}

func TestSupportsExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@supports (a) {}"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 18)
	cond := NewSupportsInterpolation(NewBooleanExpression(true, span), span)
	expr := NewSupportsExpression(cond)

	if expr.Condition != cond {
		t.Error("Condition should be the same")
	}
}

func TestSupportsExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	cond := NewSupportsInterpolation(NewBooleanExpression(true, span), span)
	expr := NewSupportsExpression(cond)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
