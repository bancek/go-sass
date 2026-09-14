// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func replaceSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

func spanRef(s sasscommon.FileSpan) *sasscommon.FileSpan { return &s }

// testReplaceExpressionVisitor wraps ReplaceExpressionVisitor to set the
// visitor field (routing recursion through the wrapper), mirroring how
// makeExpressionCalculationSafe is used in production.
type testReplaceExpressionVisitor struct {
	ReplaceExpressionVisitor
}

func newTestReplaceExpressionVisitor() *testReplaceExpressionVisitor {
	v := &testReplaceExpressionVisitor{}
	v.ReplaceExpressionVisitor.visitor = v
	return v
}

// =============================================================================
// ReplaceExpressionVisitor tests
// =============================================================================

func TestReplaceExpressionBinaryOperation(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	left := NewBooleanExpression(true, replaceSpan("true", 0, 4))
	right := NewBooleanExpression(false, replaceSpan("false", 0, 5))
	node := NewBinaryOperationExpression(BinaryOperatorAnd, left, right)

	result, err := v.VisitBinaryOperationExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	bin, ok := result.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected *BinaryOperationExpression, got %T", result)
	}
	if bin.Operator != BinaryOperatorAnd {
		t.Errorf("operator should be preserved")
	}
}

func TestReplaceExpressionBoolean(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	node := NewBooleanExpression(true, replaceSpan("true", 0, 4))

	result, err := v.VisitBooleanExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	if result != node {
		t.Error("boolean should return same node (identity)")
	}
}

func TestReplaceExpressionNull(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	node := NewNullExpression(replaceSpan("null", 0, 4))

	result, err := v.VisitNullExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	if result != node {
		t.Error("null should return same node (identity)")
	}
}

func TestReplaceExpressionNumber(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	node := NewNumberExpression(1.0, replaceSpan("1", 0, 1), nil)

	result, err := v.VisitNumberExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	if result != node {
		t.Error("number should return same node (identity)")
	}
}

func TestReplaceExpressionVariable(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	node := NewVariableExpression("x", replaceSpan("$x", 0, 2), nil)

	result, err := v.VisitVariableExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	if result != node {
		t.Error("variable should return same node (identity)")
	}
}

func TestReplaceExpressionFunction(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("fn(1)", 0, 5)
	args := NewArgumentList(
		[]Expression{NewNumberExpression(1.0, replaceSpan("1", 3, 4), nil)},
		nil,
		nil,
		replaceSpan("(1)", 2, 5),
		nil,
		nil,
	)
	node := NewFunctionExpression("fn", args, span, nil)

	result, err := v.VisitFunctionExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	fe, ok := result.(*FunctionExpression)
	if !ok {
		t.Fatalf("expected *FunctionExpression, got %T", result)
	}
	if fe.OriginalName != "fn" {
		t.Errorf("name should be preserved, got %q", fe.OriginalName)
	}
}

func TestReplaceExpressionList(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("(a, b)", 0, 6)
	items := []Expression{
		NewStringExpressionPlain("a", replaceSpan("a", 1, 2), true),
		NewStringExpressionPlain("b", replaceSpan("b", 4, 5), true),
	}
	node := NewListExpression(items, ListSeparatorComma, span, true)

	result, err := v.VisitListExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	le, ok := result.(*ListExpression)
	if !ok {
		t.Fatalf("expected *ListExpression, got %T", result)
	}
	if len(le.Contents) != 2 {
		t.Errorf("expected 2 items, got %d", len(le.Contents))
	}
}

func TestReplaceExpressionMap(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("(a: 1)", 0, 6)
	pairs := []struct {
		Key   Expression
		Value Expression
	}{
		{
			Key:   NewStringExpressionPlain("a", replaceSpan("a", 1, 2), false),
			Value: NewNumberExpression(1, replaceSpan("1", 4, 5), nil),
		},
	}
	node := NewMapExpression(pairs, span)

	result, err := v.VisitMapExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	me, ok := result.(*MapExpression)
	if !ok {
		t.Fatalf("expected *MapExpression, got %T", result)
	}
	if len(me.Pairs) != 1 {
		t.Errorf("expected 1 pair, got %d", len(me.Pairs))
	}
}

func TestReplaceExpressionString(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	text := NewInterpolationPlain("hello", replaceSpan("hello", 1, 6))
	node := NewStringExpression(text, true)

	result, err := v.VisitStringExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	se, ok := result.(*StringExpression)
	if !ok {
		t.Fatalf("expected *StringExpression, got %T", result)
	}
	if se.HasQuotes != true {
		t.Error("has_quotes should be preserved")
	}
}

func TestReplaceExpressionParenthesized(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("(1)", 0, 3)
	inner := NewNumberExpression(1.0, replaceSpan("1", 1, 2), nil)
	node := NewParenthesizedExpression(inner, span)

	result, err := v.VisitParenthesizedExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*ParenthesizedExpression); !ok {
		t.Fatalf("expected *ParenthesizedExpression, got %T", result)
	}
}

func TestReplaceExpressionSupportsCondition(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("(a: b) and (c: d)", 0, 17)
	left := NewSupportsDeclaration(
		NewStringExpressionPlain("a", replaceSpan("a", 1, 2), false),
		NewStringExpressionPlain("b", replaceSpan("b", 4, 5), false),
		replaceSpan("(a: b)", 0, 6),
	)
	right := NewSupportsDeclaration(
		NewStringExpressionPlain("c", replaceSpan("c", 11, 12), false),
		NewStringExpressionPlain("d", replaceSpan("d", 14, 15), false),
		replaceSpan("(c: d)", 10, 16),
	)
	node := NewSupportsExpression(NewSupportsOperation(left, right, BooleanOperatorAnd, span))

	result, err := v.VisitSupportsExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	se, ok := result.(*SupportsExpression)
	if !ok {
		t.Fatalf("expected *SupportsExpression, got %T", result)
	}
	if _, ok := se.Condition.(*SupportsOperation); !ok {
		t.Fatalf("expected *SupportsOperation in condition, got %T", se.Condition)
	}
}

func TestReplaceExpressionInterpolation(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("a #{$x} b", 0, 9)
	parts := []any{
		"a ",
		NewVariableExpression("x", replaceSpan("$x", 4, 6), nil),
		" b",
	}
	spans := []*sasscommon.FileSpan{nil, spanRef(replaceSpan("#{$x}", 2, 8)), nil}
	interp, err := NewInterpolation(parts, spans, span)
	if err != nil {
		t.Fatal(err)
	}
	node := NewStringExpression(interp, false)

	result, err := v.VisitStringExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	se, ok := result.(*StringExpression)
	if !ok {
		t.Fatalf("expected *StringExpression, got %T", result)
	}
	if se.Text == nil {
		t.Fatal("text should not be nil")
	}
}

func TestReplaceExpressionIf(t *testing.T) {
	v := newTestReplaceExpressionVisitor()
	span := replaceSpan("if(sass($x); 1; else 2)", 0, 23)
	condExpr := NewVariableExpression("x", replaceSpan("$x", 6, 8), nil)
	cond := NewIfConditionSass(condExpr, replaceSpan("sass($x)", 3, 11))
	thenExpr := NewNumberExpression(1.0, replaceSpan("1", 13, 14), nil)
	elseExpr := NewNumberExpression(2.0, replaceSpan("22", 20, 22), nil)
	node, err := NewIfExpression([]IfBranch{
		{Condition: cond, Expression: thenExpr},
		{Expression: elseExpr},
	}, span)
	if err != nil {
		t.Fatal(err)
	}

	result, err := v.VisitIfExpression(node)
	if err != nil {
		t.Fatal(err)
	}
	ie, ok := result.(*IfExpression)
	if !ok {
		t.Fatalf("expected *IfExpression, got %T", result)
	}
	if len(ie.Branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(ie.Branches))
	}
	if ie.Branches[0].Condition == nil {
		t.Fatal("expected first branch to have a condition")
	}
	if _, ok := ie.Branches[0].Condition.(*IfConditionSass); !ok {
		t.Fatalf("expected *IfConditionSass, got %T", ie.Branches[0].Condition)
	}
	if _, ok := ie.Branches[0].Expression.(*NumberExpression); !ok {
		t.Fatalf("expected *NumberExpression, got %T", ie.Branches[0].Expression)
	}
	if ie.Branches[1].Condition != nil {
		t.Fatal("expected else branch to have nil condition")
	}
}
