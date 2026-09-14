package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func span(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

func TestIsPlainCssColor(t *testing.T) {
	color, _ := NewColorRGB(1, 2, 3, 1)
	expr := NewColorExpression(color, span("green", 0, 5))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("color should be plain CSS")
	}
}

func TestIsPlainCssNumber(t *testing.T) {
	expr := NewNumberExpression(1.0, span("1", 0, 1), nil)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("number should be plain CSS")
	}
}

func TestIsPlainCssBoolean(t *testing.T) {
	expr := NewBooleanExpression(true, span("true", 0, 4))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("boolean should not be plain CSS")
	}
}

func TestIsPlainCssNull(t *testing.T) {
	expr := NewNullExpression(span("null", 0, 4))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("null should not be plain CSS")
	}
}

func TestIsPlainCssVariable(t *testing.T) {
	expr := NewVariableExpression("x", span("$x", 0, 2), nil)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("variable should not be plain CSS")
	}
}

func TestIsPlainCssSelector(t *testing.T) {
	expr := NewSelectorExpression(span("&", 0, 1))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("selector should not be plain CSS")
	}
}

func TestIsPlainCssMap(t *testing.T) {
	expr := NewMapExpression(nil, span("()", 0, 2))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("map should not be plain CSS")
	}
}

func TestIsPlainCssStringPlain(t *testing.T) {
	s := span("foo", 0, 3)
	text := NewInterpolationPlain("foo", s)
	expr := NewStringExpression(text, false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("plain string should be plain CSS")
	}
}

func TestIsPlainCssStringInterpolated(t *testing.T) {
	s := span("#{$x}", 0, 5)
	exprSpan := span("#{$x}", 2, 4)
	text, err := NewInterpolation(
		[]any{NewVariableExpression("x", exprSpan, nil)},
		[]*sasscommon.FileSpan{&exprSpan},
		s,
	)
	if err != nil {
		t.Fatal(err)
	}
	expr := NewStringExpression(text, false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("interpolated string should not be plain CSS when allowInterpolation=false")
	}
}

func TestIsPlainCssStringInterpolatedAllowed(t *testing.T) {
	s := span("#{$x}", 0, 5)
	exprSpan := span("#{$x}", 2, 4)
	text, err := NewInterpolation(
		[]any{NewVariableExpression("x", exprSpan, nil)},
		[]*sasscommon.FileSpan{&exprSpan},
		s,
	)
	if err != nil {
		t.Fatal(err)
	}
	expr := NewStringExpression(text, false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(true))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("interpolated string should be plain CSS when allowInterpolation=true")
	}
}

func TestIsPlainCssFunctionNoNamespace(t *testing.T) {
	s := span("rgb()", 0, 5)
	args := NewArgumentListEmpty(s)
	expr := NewFunctionExpression("rgb", args, s, nil)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("function without namespace should be plain CSS")
	}
}

func TestIsPlainCssFunctionWithNamespace(t *testing.T) {
	s := span("ns.rgb()", 0, 8)
	ns := "ns"
	args := NewArgumentListEmpty(s)
	expr := NewFunctionExpression("rgb", args, s, &ns)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("function with namespace should not be plain CSS")
	}
}

func TestIsPlainCssBinaryOperation(t *testing.T) {
	left := NewNumberExpression(1.0, span("1 + 2", 0, 1), nil)
	right := NewNumberExpression(2.0, span("1 + 2", 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("binary operation should not be plain CSS")
	}
}

func TestIsPlainCssParenthesized(t *testing.T) {
	color, _ := NewColorRGB(1, 2, 3, 1)
	inner := NewColorExpression(color, span("red", 0, 3))
	expr := NewParenthesizedExpression(inner, span("(red)", 0, 5))
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("parenthesized color should be plain CSS")
	}
}

func TestIsPlainCssListEmptyUnbracketed(t *testing.T) {
	s := span("", 0, 0)
	expr := NewListExpression(nil, ListSeparatorSpace, s, false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("empty unbracketed list should not be plain CSS")
	}
}

func TestIsPlainCssListAllNumbers(t *testing.T) {
	contents := []Expression{
		NewNumberExpression(1.0, span("1 2", 0, 1), nil),
		NewNumberExpression(2.0, span("1 2", 2, 3), nil),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, span("1 2", 0, 3), false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if !plain {
		t.Error("list of plain-CSS-safe elements should be plain CSS")
	}
}

func TestIsPlainCssListWithNonPlain(t *testing.T) {
	s := span("1 $x", 0, 4)
	contents := []Expression{
		NewNumberExpression(1.0, span("1", 0, 1), nil),
		NewVariableExpression("x", span("$x", 2, 4), nil),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, s, false)
	plain, err := expr.AcceptBool(NewIsPlainCssVisitor(false))
	if err != nil {
		t.Fatal(err)
	}
	if plain {
		t.Error("list with non-plain element should not be plain CSS")
	}
}
