package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func calcSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

func TestIsCalculationSafeNumber(t *testing.T) {
	expr := NewNumberExpression(1.0, calcSpan("1", 0, 1), nil)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("number should be calculation safe")
	}
}

func TestIsCalculationSafeFunction(t *testing.T) {
	s := calcSpan("calc()", 0, 6)
	args := NewArgumentListEmpty(s)
	expr := NewFunctionExpression("calc", args, s, nil)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("function should be calculation safe")
	}
}

func TestIsCalculationSafeVariable(t *testing.T) {
	expr := NewVariableExpression("x", calcSpan("$x", 0, 2), nil)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("variable should be calculation safe")
	}
}

func TestIsCalculationSafeNull(t *testing.T) {
	expr := NewNullExpression(calcSpan("null", 0, 4))
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("null should not be calculation safe")
	}
}

func TestIsCalculationSafeBoolean(t *testing.T) {
	expr := NewBooleanExpression(true, calcSpan("true", 0, 4))
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("boolean should not be calculation safe")
	}
}

func TestIsCalculationSafeColor(t *testing.T) {
	color, _ := NewColorRGB(1, 2, 3, 1)
	expr := NewColorExpression(color, calcSpan("red", 0, 3))
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("color should not be calculation safe")
	}
}

func TestIsCalculationSafeBinaryOpPlus(t *testing.T) {
	left := NewNumberExpression(1.0, calcSpan("1 + 2", 0, 1), nil)
	right := NewNumberExpression(2.0, calcSpan("1 + 2", 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("numeric plus should be calculation safe")
	}
}

func TestIsCalculationSafeBinaryOpEquals(t *testing.T) {
	left := NewNumberExpression(1.0, calcSpan("1 == 2", 0, 1), nil)
	right := NewNumberExpression(2.0, calcSpan("1 == 2", 5, 6), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorEquals, left, right)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("equals should not be calculation safe")
	}
}

func TestIsCalculationSafeListSpace2Plus(t *testing.T) {
	s := calcSpan("1 2", 0, 3)
	contents := []Expression{
		NewNumberExpression(1.0, calcSpan("1", 0, 1), nil),
		NewNumberExpression(2.0, calcSpan("2", 2, 3), nil),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, s, false)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("space-separated unbracketed list with >=2 elements should be calculation safe")
	}
}

func TestIsCalculationSafeListComma(t *testing.T) {
	contents := []Expression{
		NewNumberExpression(1.0, calcSpan("1, 2", 0, 1), nil),
		NewNumberExpression(2.0, calcSpan("1, 2", 3, 4), nil),
	}
	expr := NewListExpression(contents, ListSeparatorComma, calcSpan("1, 2", 0, 4), false)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("comma-separated list should not be calculation safe")
	}
}

func TestIsCalculationSafeListBracketed(t *testing.T) {
	s := calcSpan("[1 2]", 0, 5)
	contents := []Expression{
		NewNumberExpression(1.0, calcSpan("1", 1, 2), nil),
		NewNumberExpression(2.0, calcSpan("2", 3, 4), nil),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, s, true)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("bracketed list should not be calculation safe")
	}
}

func TestIsCalculationSafeListSingle(t *testing.T) {
	s := calcSpan("1", 0, 1)
	contents := []Expression{
		NewNumberExpression(1.0, calcSpan("1", 0, 1), nil),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, s, false)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("single-element list should not be calculation safe")
	}
}

func TestIsCalculationSafeStringUnquoted(t *testing.T) {
	s := calcSpan("foo", 0, 3)
	text := NewInterpolationPlain("foo", s)
	expr := NewStringExpression(text, false)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if !safe {
		t.Error("unquoted identifier-like string should be calculation safe")
	}
}

func TestIsCalculationSafeStringQuoted(t *testing.T) {
	s := calcSpan("\"foo\"", 0, 5)
	text := NewInterpolationPlain("foo", s)
	expr := NewStringExpression(text, true)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("quoted string should not be calculation safe")
	}
}

func TestIsCalculationSafeStringImportant(t *testing.T) {
	s := calcSpan("!important", 0, 10)
	text := NewInterpolationPlain("!important", s)
	expr := NewStringExpression(text, false)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("!important string should not be calculation safe")
	}
}

func TestIsCalculationSafeUnaryOperation(t *testing.T) {
	s := calcSpan("-1", 0, 2)
	inner := NewNumberExpression(1.0, calcSpan("1", 1, 2), nil)
	expr := NewUnaryOperationExpression(UnaryOperatorMinus, inner, s)
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("unary operation should not be calculation safe")
	}
}

func TestIsCalculationSafeSelector(t *testing.T) {
	expr := NewSelectorExpression(calcSpan("&", 0, 1))
	safe, err := expr.AcceptBool(NewIsCalculationSafeVisitor())
	if err != nil {
		t.Fatal(err)
	}
	if safe {
		t.Error("selector should not be calculation safe")
	}
}
