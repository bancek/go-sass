// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package value

import (
	"net/url"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func visitorSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

// =============================================================================
// RecursiveStatementVisitor tests
// =============================================================================

func TestRecursiveStatementVisitChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, span, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	children := []Statement{vd}
	err = v.visitChildren(children)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementVisitChildrenEmpty(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	err := v.visitChildren([]Statement{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementStylesheet(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, span, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	ss := NewStylesheet([]Statement{vd}, span)
	_, err = ss.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementAtRuleNilChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@foo;", 0, 5)
	name := NewInterpolationPlain("foo", visitorSpan("foo", 1, 4))
	ar := NewAtRule(name, span, nil, nil)
	_, err := ar.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementAtRuleEmptyChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@foo { }", 0, 8)
	name := NewInterpolationPlain("foo", visitorSpan("foo", 1, 4))
	ar := NewAtRule(name, span, nil, []Statement{})
	_, err := ar.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementAtRuleWithChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@foo { }", 0, 8)
	name := NewInterpolationPlain("foo", visitorSpan("foo", 1, 4))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	ar := NewAtRule(name, span, nil, []Statement{vd})
	_, err = ar.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementDeclarationNilChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("color: red;", 0, 10)
	name := NewInterpolationPlain("color", visitorSpan("color", 0, 5))
	val := NewNullExpression(visitorSpan("null", 7, 10))
	d := NewDeclaration(name, val, span)
	_, err := d.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementDeclarationWithChildren(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("color: red { }", 0, 14)
	name := NewInterpolationPlain("color", visitorSpan("color", 0, 5))
	val := NewNullExpression(visitorSpan("null", 7, 9))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDeclarationNested(name, []Statement{vd}, span, val)
	_, err = d.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementIfRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@if true { }", 0, 12)
	trueExpr := NewBooleanExpression(true, visitorSpan("true", 4, 8))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	childExpr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", childExpr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	clause := NewIfClause(trueExpr, []Statement{vd})
	ifRule := NewIfRule([]*IfClause{clause}, span, nil)
	_, err = ifRule.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementIfRuleWithElse(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@if true { } @else { }", 0, 21)
	trueExpr := NewBooleanExpression(true, visitorSpan("true", 4, 8))
	vd, err := NewVariableDeclaration("x", NewNullExpression(visitorSpan("null", 0, 4)), visitorSpan("$x: 1;", 0, 6), nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	clause := NewIfClause(trueExpr, []Statement{vd})
	elseClause := NewElseClause([]Statement{vd})
	ifRule := NewIfRule([]*IfClause{clause}, span, elseClause)
	_, err = ifRule.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementIncludeContent(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("+foo", 0, 4)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	contentSpan := visitorSpan("{ }", 0, 3)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cb := NewContentBlock(params, nil, contentSpan)
	ir := NewIncludeRule("foo", args, span, nil, cb)
	_, err := ir.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementIncludeNoContent(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("+foo", 0, 4)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	ir := NewIncludeRule("foo", args, span, nil, nil)
	_, err := ir.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementFunctionRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@function f() { }", 0, 17)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	fr := NewFunctionRule("f", params, []Statement{vd}, span, nil)
	_, err = fr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementMixinRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@mixin m() { }", 0, 14)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	mr := NewMixinRule("m", params, []Statement{vd}, span, nil)
	_, err = mr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementLeafNodes(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("test", 0, 4)
	expr := NewNullExpression(visitorSpan("null", 0, 4))

	tests := []struct {
		name string
		stmt Statement
	}{
		{"ContentRule", NewContentRule(NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0)), span)},
		{"DebugRule", NewDebugRule(expr, span)},
		{"ErrorRule", NewErrorRule(expr, span)},
		{"WarnRule", NewWarnRule(expr, span)},
		{"ReturnRule", NewReturnRule(expr, span)},
		{"ExtendRule", NewExtendRule(NewInterpolationPlain("a", visitorSpan("a", 0, 1)), span, false)},
		{"ImportRule", NewImportRule(nil, span)},
		{"LoudComment", NewLoudComment(NewInterpolationPlain("/* */", span))},
		{"SilentComment", NewSilentComment("//", span)},
		{"UseRule", &UseRule{span: span}},
		{"ForwardRule", &ForwardRule{span: span, url: &url.URL{Scheme: "sass", Opaque: "test"}}},
		{"VariableDeclaration", mustVarDecl("x", expr, span)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.stmt.AcceptVoid(v)
			if err != nil {
				t.Errorf("AcceptVoid on %s: unexpected error: %v", tt.name, err)
			}
		})
	}
}

func TestRecursiveStatementContentBlock(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("{ }", 0, 3)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	cb := NewContentBlock(params, []Statement{vd}, span)
	_, err = cb.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementStyleRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan(".a { }", 0, 6)
	sel := NewInterpolationPlain(".a", visitorSpan(".a", 0, 2))
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	sr := NewStyleRule(sel, []Statement{vd}, span)
	_, err = sr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementMediaRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@media screen { }", 0, 17)
	query := NewInterpolationPlain("screen", visitorSpan("screen", 7, 13))
	mr := NewMediaRule(query, nil, span)
	_, err := mr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementSupportsRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@supports (a: b) { }", 0, 21)
	decl := NewSupportsDeclaration(
		NewStringExpressionPlain("a", visitorSpan("a", 11, 12), false),
		NewStringExpressionPlain("b", visitorSpan("b", 14, 15), false),
		visitorSpan("(a: b)", 10, 16),
	)
	sr := NewSupportsRule(decl, nil, span)
	_, err := sr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementEachRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@each $x in (a, b) { }", 0, 23)
	listExpr := NewListExpression(
		[]Expression{
			NewStringExpressionPlain("a", visitorSpan("a", 12, 13), true),
			NewStringExpressionPlain("b", visitorSpan("b", 15, 16), true),
		},
		ListSeparatorComma,
		visitorSpan("(a, b)", 11, 17),
		true,
	)
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	er := NewEachRule([]string{"x"}, listExpr, []Statement{vd}, span)
	_, err = er.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementForRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@for $i from 1 through 3 { }", 0, 30)
	fromExpr := NewNumberExpression(1, visitorSpan("1", 13, 14), nil)
	toExpr := NewNumberExpression(3, visitorSpan("3", 23, 24), nil)
	fr := NewForRule("i", fromExpr, toExpr, nil, span, true)
	_, err := fr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementWhileRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@while true { }", 0, 14)
	condExpr := NewBooleanExpression(true, visitorSpan("true", 7, 11))
	wr := NewWhileRule(condExpr, nil, span)
	_, err := wr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveStatementAtRootRule(t *testing.T) {
	v := &RecursiveStatementVisitor{}
	span := visitorSpan("@at-root { }", 0, 11)
	childSpan := visitorSpan("$x: 1;", 0, 6)
	expr := NewNullExpression(visitorSpan("null", 0, 4))
	vd, err := NewVariableDeclaration("x", expr, childSpan, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	ar := NewAtRootRule([]Statement{vd}, span, nil)
	_, err = ar.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// RecursiveAstVisitor tests
// =============================================================================

func TestRecursiveAstDebugRuleExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@debug $x;", 0, 10)
	expr := NewVariableExpression("x", visitorSpan("$x", 7, 9), nil)
	dr := NewDebugRule(expr, span)
	_, err := dr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstVariableDeclarationExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("$x: 1 + 2;", 0, 10)
	left := NewNumberExpression(1, visitorSpan("1", 4, 5), nil)
	right := NewNumberExpression(2, visitorSpan("2", 8, 9), nil)
	binExpr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)
	vd, err := NewVariableDeclaration("x", binExpr, span, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = vd.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstAtRuleInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@foo bar { }", 0, 12)
	name := NewInterpolationPlain("foo", visitorSpan("foo", 1, 4))
	value := NewInterpolationPlain("bar", visitorSpan("bar", 5, 8))
	ar := NewAtRule(name, span, value, nil)
	_, err := ar.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstDeclarationInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("color: red;", 0, 10)
	name := NewInterpolationPlain("color", visitorSpan("color", 0, 5))
	val := NewStringExpressionPlain("red", visitorSpan("red", 7, 10), false)
	d := NewDeclaration(name, val, span)
	_, err := d.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstContentRuleArgumentList(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@content($a);", 0, 13)
	argSpan := visitorSpan("$a", 9, 11)
	expr := NewVariableExpression("a", argSpan, nil)
	args := NewArgumentList(
		[]Expression{expr},
		nil,
		nil,
		sasscommon.NewSimpleFileSpan(nil, 0, 0),
		nil,
		nil,
	)
	cr := NewContentRule(args, span)
	_, err := cr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstIncludeArgumentList(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("+foo(1)", 0, 7)
	argSpan := visitorSpan("1", 5, 6)
	expr := NewNumberExpression(1, argSpan, nil)
	args := NewArgumentList(
		[]Expression{expr},
		nil,
		nil,
		sasscommon.NewSimpleFileSpan(nil, 0, 0),
		nil,
		nil,
	)
	ir := NewIncludeRule("foo", args, span, nil, nil)
	_, err := ir.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstBinaryOperationExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	left := NewNumberExpression(1, visitorSpan("1", 0, 1), nil)
	right := NewNumberExpression(2, visitorSpan("2", 4, 5), nil)
	binExpr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)
	// Visit expression directly via AcceptVoid
	_, err := binExpr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstListExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("(a, b)", 0, 6)
	items := []Expression{
		NewStringExpressionPlain("a", visitorSpan("a", 1, 2), true),
		NewStringExpressionPlain("b", visitorSpan("b", 4, 5), true),
	}
	le := NewListExpression(items, ListSeparatorComma, span, true)
	_, err := le.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstMapExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("(a: 1)", 0, 6)
	pairs := []struct {
		Key   Expression
		Value Expression
	}{
		{Key: NewStringExpressionPlain("a", visitorSpan("a", 1, 2), false), Value: NewNumberExpression(1, visitorSpan("1", 4, 5), nil)},
	}
	me := NewMapExpression(pairs, span)
	_, err := me.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstParenthesizedExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("(1)", 0, 3)
	inner := NewNumberExpression(1, visitorSpan("1", 1, 2), nil)
	pe := NewParenthesizedExpression(inner, span)
	_, err := pe.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstUnaryOperationExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("-1", 0, 2)
	operand := NewNumberExpression(1, visitorSpan("1", 1, 2), nil)
	ue := NewUnaryOperationExpression(UnaryOperatorMinus, operand, span)
	_, err := ue.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstFunctionExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	args := NewArgumentList(
		nil,
		nil,
		nil,
		sasscommon.NewSimpleFileSpan(nil, 0, 0),
		nil,
		nil,
	)
	fe := NewFunctionExpression("foo", args, visitorSpan("foo()", 0, 5), nil)
	_, err := fe.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstInterpolatedFunctionExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("#{f}()", 0, 6)
	name := NewInterpolationPlain("f", visitorSpan("f", 2, 3))
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	ife := NewInterpolatedFunctionExpression(name, args, span)
	_, err := ife.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstIfExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("if(true, a, b)", 0, 14)
	cond := NewBooleanExpression(true, visitorSpan("true", 3, 7))
	thenExpr := NewStringExpressionPlain("a", visitorSpan("a", 9, 10), true)
	elseExpr := NewStringExpressionPlain("b", visitorSpan("b", 12, 13), true)
	branches := []IfBranch{
		{Condition: &IfConditionSass{Expression: cond, span: visitorSpan("true", 3, 7)}, Expression: thenExpr},
		{Condition: nil, Expression: elseExpr},
	}
	ife, err := NewIfExpression(branches, span)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ife.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstSupportsConditionOperation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("(a: b) and (c: d)", 0, 17)
	left := NewSupportsDeclaration(
		NewStringExpressionPlain("a", visitorSpan("a", 1, 2), false),
		NewStringExpressionPlain("b", visitorSpan("b", 4, 5), false),
		visitorSpan("(a: b)", 0, 6),
	)
	right := NewSupportsDeclaration(
		NewStringExpressionPlain("c", visitorSpan("c", 11, 12), false),
		NewStringExpressionPlain("d", visitorSpan("d", 14, 15), false),
		visitorSpan("(c: d)", 10, 16),
	)
	op := NewSupportsOperation(left, right, BooleanOperatorAnd, span)
	se := NewSupportsExpression(op)
	_, err := se.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstSupportsConditionNegation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("not (a: b)", 0, 10)
	inner := NewSupportsDeclaration(
		NewStringExpressionPlain("a", visitorSpan("a", 5, 6), false),
		NewStringExpressionPlain("b", visitorSpan("b", 8, 9), false),
		visitorSpan("(a: b)", 4, 10),
	)
	neg := NewSupportsNegation(inner, span)
	se := NewSupportsExpression(neg)
	_, err := se.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstSupportsConditionInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("#{$x}", 0, 5)
	expr := NewVariableExpression("x", visitorSpan("$x", 2, 4), nil)
	interp := NewSupportsInterpolation(expr, span)
	se := NewSupportsExpression(interp)
	_, err := se.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstInterpolatedAttributeSelector(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("[name]", 0, 6)
	name := &InterpolatedQualifiedName{
		Name: NewInterpolationPlain("name", visitorSpan("name", 1, 5)),
		span: visitorSpan("name", 1, 5),
	}
	as := NewInterpolatedAttributeSelector(name, span)
	_, err := as.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstInterpolatedClassSelector(t *testing.T) {
	v := &RecursiveAstVisitor{}
	name := NewInterpolationPlain("foo", visitorSpan("foo", 0, 4))
	cs, err := NewInterpolatedClassSelector(name)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cs.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstInterpolatedCompoundSelector(t *testing.T) {
	v := &RecursiveAstVisitor{}
	cs, err := NewInterpolatedClassSelector(
		NewInterpolationPlain("a", visitorSpan("a", 0, 2)),
	)
	if err != nil {
		t.Fatal(err)
	}
	compound, err := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{cs})
	if err != nil {
		t.Fatal(err)
	}
	_, err = compound.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstForwardRuleConfiguration(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@forward 'foo' with ($x: 1)", 0, 27)
	u := &url.URL{Scheme: "sass", Opaque: "foo"}
	expr := NewNumberExpression(1, visitorSpan("1", 23, 24), nil)
	cv := NewConfiguredVariable("x", expr, span, false)
	fr := NewForwardRule(u, span, nil, []*ConfiguredVariable{cv})
	_, err := fr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstUseRuleConfiguration(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@use 'foo' with ($x: 1)", 0, 23)
	u, err := url.Parse("sass:foo")
	if err != nil {
		t.Fatal(err)
	}
	expr := NewNumberExpression(1, visitorSpan("1", 19, 20), nil)
	cv := NewConfiguredVariable("x", expr, span, false)
	ur, err := NewUseRule(u, nil, span, []*ConfiguredVariable{cv})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ur.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstImportRule(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@import 'foo';", 0, 14)
	staticImp := NewStaticImport(
		NewInterpolationPlain("foo", visitorSpan("foo", 9, 12)),
		visitorSpan("'foo'", 8, 13),
		nil,
	)
	ir := NewImportRule([]Import{staticImp}, span)
	_, err := ir.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstImportRuleWithModifiers(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@import 'foo' screen;", 0, 21)
	staticImp := NewStaticImport(
		NewInterpolationPlain("foo", visitorSpan("foo", 9, 12)),
		visitorSpan("'foo' screen", 8, 20),
		NewInterpolationPlain("screen", visitorSpan("screen", 14, 20)),
	)
	ir := NewImportRule([]Import{staticImp}, span)
	_, err := ir.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstFunctionRuleParameters(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@function f($x: 1) { }", 0, 21)
	paramSpan := visitorSpan("$x: 1", 13, 18)
	defaultVal := NewNumberExpression(1, visitorSpan("1", 17, 18), nil)
	p := NewParameter("x", paramSpan, defaultVal)
	pl := NewParameterList([]*Parameter{p}, paramSpan, nil)
	fr := NewFunctionRule("f", pl, nil, span, nil)
	_, err := fr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstMixinRuleParameters(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@mixin m($x: 1) { }", 0, 19)
	paramSpan := visitorSpan("$x: 1", 9, 14)
	defaultVal := NewNumberExpression(1, visitorSpan("1", 13, 14), nil)
	p := NewParameter("x", paramSpan, defaultVal)
	pl := NewParameterList([]*Parameter{p}, paramSpan, nil)
	mr := NewMixinRule("m", pl, nil, span, nil)
	_, err := mr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstStyleRuleSelector(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan(".a { }", 0, 6)
	sel := NewInterpolationPlain(".a", visitorSpan(".a", 0, 2))
	sr := NewStyleRule(sel, nil, span)
	_, err := sr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstLoudCommentInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("/* */", 0, 5)
	text := NewInterpolationPlain("/* */", span)
	lc := NewLoudComment(text)
	_, err := lc.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstMediaRuleInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("@media screen { }", 0, 17)
	query := NewInterpolationPlain("screen", visitorSpan("screen", 7, 13))
	mr := NewMediaRule(query, nil, span)
	_, err := mr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstStringExpressionInterpolation(t *testing.T) {
	v := &RecursiveAstVisitor{}
	text := NewInterpolationPlain("hello", visitorSpan("hello", 0, 7))
	se := NewStringExpression(text, true)
	_, err := se.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstBoolColorNullNumberLeafExpressions(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("leaf", 0, 4)

	tests := []Expression{
		NewBooleanExpression(true, span),
		NewNullExpression(span),
		NewNumberExpression(1, span, nil),
	}
	// ColorExpression needs a SassColor
	color, err := NewColorRGB(1, 2, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	tests = append(tests, NewColorExpression(color, span))
	tests = append(tests, NewSelectorExpression(span))
	tests = append(tests, NewValueExpression(nil, span))
	tests = append(tests, NewVariableExpression("x", span, nil))

	for _, expr := range tests {
		_, err := expr.AcceptVoid(v)
		if err != nil {
			t.Errorf("AcceptVoid on %T: unexpected error: %v", expr, err)
		}
	}
}

func TestRecursiveAstWarnErrorReturnExpressions(t *testing.T) {
	v := &RecursiveAstVisitor{}
	span := visitorSpan("test", 0, 4)
	expr := NewNumberExpression(1, visitorSpan("1", 0, 1), nil)

	wr := NewWarnRule(expr, span)
	_, err := wr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}

	er := NewErrorRule(expr, span)
	_, err = er.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}

	rr := NewReturnRule(expr, span)
	_, err = rr.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursiveAstLegacyIfExpression(t *testing.T) {
	v := &RecursiveAstVisitor{}
	args := NewArgumentList(
		[]Expression{NewNumberExpression(1, visitorSpan("1", 0, 1), nil)},
		nil,
		nil,
		sasscommon.NewSimpleFileSpan(nil, 0, 0),
		nil,
		nil,
	)
	lie := NewLegacyIfExpression(args, visitorSpan("if(1, a, b)", 0, 12))
	_, err := lie.AcceptVoid(v)
	if err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// StatementSearchVisitor tests
// =============================================================================

func TestStatementSearchFindsMatch(t *testing.T) {
	v := &StatementSearchVisitor{}
	span := visitorSpan("@warn true;", 0, 10)
	expr := NewBooleanExpression(true, visitorSpan("true", 6, 10))
	wr := NewWarnRule(expr, span)
	ss := NewStylesheet([]Statement{wr}, span)
	result, err := ss.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor should return false for leaf nodes (WarnRule)")
	}
}

func TestStatementSearchReturnsFalseDefault(t *testing.T) {
	v := &StatementSearchVisitor{}
	span := visitorSpan("@debug $x;", 0, 10)
	expr := NewVariableExpression("x", visitorSpan("$x", 7, 9), nil)
	dr := NewDebugRule(expr, span)
	result, err := dr.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor should return false by default for DebugRule")
	}
}

func TestStatementSearchContentRuleDefault(t *testing.T) {
	v := &StatementSearchVisitor{}
	span := visitorSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)
	result, err := cr.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor should return false by default for ContentRule")
	}
}

func TestStatementSearchContentRuleFunc(t *testing.T) {
	v := &StatementSearchVisitor{
		ContentRuleFunc: func(cr *ContentRule) (bool, error) {
			return true, nil
		},
	}
	span := visitorSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)
	result, err := cr.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("StatementSearchVisitor ContentRuleFunc should return true when callback returns true")
	}
}

func TestStatementSearchContentRuleFuncFalse(t *testing.T) {
	v := &StatementSearchVisitor{
		ContentRuleFunc: func(cr *ContentRule) (bool, error) {
			return false, nil
		},
	}
	span := visitorSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)
	result, err := cr.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor ContentRuleFunc should return false when callback returns false")
	}
}

func TestStatementSearchIncludeContent(t *testing.T) {
	v := &StatementSearchVisitor{}
	span := visitorSpan("+foo", 0, 4)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	contentSpan := visitorSpan("{ }", 0, 3)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cb := NewContentBlock(params, nil, contentSpan)
	ir := NewIncludeRule("foo", args, span, nil, cb)
	result, err := ir.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor should return false for IncludeRule with content")
	}
}

func TestStatementSearchIncludeNoContent(t *testing.T) {
	v := &StatementSearchVisitor{}
	span := visitorSpan("+foo", 0, 4)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	ir := NewIncludeRule("foo", args, span, nil, nil)
	result, err := ir.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("StatementSearchVisitor should return false for IncludeRule without content")
	}
}

// =============================================================================
// FindDependencies tests
// =============================================================================

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic("bad URL in test: " + raw)
	}
	return u
}

func TestFindDependenciesUseRule(t *testing.T) {
	span := visitorSpan("@use 'foo';", 0, 11)
	u := mustParseURL("file:///foo")
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	ss := NewStylesheet([]Statement{ur}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	if _, ok := report.Uses[u.String()]; !ok {
		t.Errorf("expected uses to contain %q, got %v", u.String(), report.Uses)
	}
}

func TestFindDependenciesForwardRule(t *testing.T) {
	span := visitorSpan("@forward 'bar';", 0, 15)
	u := mustParseURL("file:///bar")
	fr := NewForwardRule(u, span, nil, nil)
	ss := NewStylesheet([]Statement{fr}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	if _, ok := report.Forwards[u.String()]; !ok {
		t.Errorf("expected forwards to contain %q, got %v", u.String(), report.Forwards)
	}
}

func TestFindDependenciesBuiltInExcluded(t *testing.T) {
	span := visitorSpan("@use 'sass:color';", 0, 18)
	u := mustParseURL("sass:color")
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	ss := NewStylesheet([]Statement{ur}, span)
	report := FindDependencies(ss)

	if len(report.Uses) != 0 {
		t.Errorf("expected 0 uses for built-in, got %d", len(report.Uses))
	}
}

func TestFindDependenciesMetaLoadCss(t *testing.T) {
	span := visitorSpan("@use 'sass:meta';", 0, 17)
	metaUrl := mustParseURL("sass:meta")
	namespace := "meta"
	metaUse, err := NewUseRule(metaUrl, &namespace, span, nil)
	if err != nil {
		t.Fatal(err)
	}

	includeSpan := visitorSpan("@include meta.load-css('baz');", 0, 29)
	argSpan := visitorSpan("'baz'", 20, 25)
	argText := NewInterpolationPlain("baz", visitorSpan("baz", 21, 24))
	strExpr := NewStringExpression(argText, true)
	args := NewArgumentList(
		[]Expression{strExpr},
		nil,
		nil,
		argSpan,
		nil,
		nil,
	)
	inc := NewIncludeRule("load-css", args, includeSpan, &namespace, nil)

	ss := NewStylesheet([]Statement{metaUse, inc}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	u := mustParseURL("baz")
	if _, ok := report.MetaLoadCss[u.String()]; !ok {
		t.Errorf("expected metaLoadCss to contain %q, got %v", u.String(), report.MetaLoadCss)
	}
}

func TestFindDependenciesImportRule(t *testing.T) {
	span := visitorSpan("@import 'file';", 0, 15)
	u := mustParseURL("file:///file")
	di := NewDynamicImport(u.String(), visitorSpan("'file'", 8, 14))
	ir := NewImportRule([]Import{di}, span)
	ss := NewStylesheet([]Statement{ir}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	if _, ok := report.Imports[u.String()]; !ok {
		t.Errorf("expected imports to contain %q, got %v", u.String(), report.Imports)
	}
}

func TestFindDependenciesModulesMethod(t *testing.T) {
	span := visitorSpan("@use 'mod';", 0, 11)
	u := mustParseURL("file:///mod")
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	ss := NewStylesheet([]Statement{ur}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	modules := report.Modules()
	found := false
	for _, m := range modules {
		if m.String() == u.String() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Modules to contain %q, got %v", u.String(), modules)
	}
}

func TestFindDependenciesAllMethod(t *testing.T) {
	span := visitorSpan("@use 'mod'; @import 'imp';", 0, 24)
	u := mustParseURL("file:///mod")
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	di := NewDynamicImport("file:///imp", visitorSpan("'imp'", 18, 23))
	ir := NewImportRule([]Import{di}, span)
	ss := NewStylesheet([]Statement{ur, ir}, span)
	report := FindDependencies(ss)

	if report == nil {
		t.Fatal("FindDependencies returned nil")
	}
	all := report.All()
	foundUse := false
	foundImp := false
	for _, u := range all {
		switch u.String() {
		case "file:///mod":
			foundUse = true
		case "file:///imp":
			foundImp = true
		}
	}
	if !foundUse || !foundImp {
		t.Errorf("expected All to contain both file:///mod and file:///imp, got %v", all)
	}
}

func TestFindDependenciesSkipBodies(t *testing.T) {
	// Dependencies in function/mixin bodies should NOT be found
	span := visitorSpan("@function f() { @use 'inner'; @return 1; }", 0, 40)
	params := NewParameterListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	innerSpan := visitorSpan("@use 'inner';", 0, 13)
	innerU := mustParseURL("file:///inner")
	innerUse, err := NewUseRule(innerU, nil, innerSpan, nil)
	if err != nil {
		t.Fatal(err)
	}
	fr := NewFunctionRule("f", params, []Statement{innerUse}, span, nil)
	ss := NewStylesheet([]Statement{fr}, span)
	report := FindDependencies(ss)

	if len(report.Uses) != 0 {
		t.Errorf("expected 0 uses inside function body, got %d", len(report.Uses))
	}
	if len(report.Forwards) != 0 {
		t.Errorf("expected 0 forwards inside function body, got %d", len(report.Forwards))
	}
	if len(report.Imports) != 0 {
		t.Errorf("expected 0 imports inside function body, got %d", len(report.Imports))
	}
}

func mustVarDecl(name string, expr Expression, span sasscommon.FileSpan) *VariableDeclaration {
	vd, err := NewVariableDeclaration(name, expr, span, nil, false, false, nil)
	if err != nil {
		panic(err)
	}
	return vd
}
