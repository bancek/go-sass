package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func supportsSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

// --- SupportsAnything ---

func TestSupportsAnythingConstruction(t *testing.T) {
	span := supportsSpan("(foo)", 0, 5)
	contents := NewInterpolationPlain("foo", supportsSpan("foo", 1, 4))
	cond := NewSupportsAnything(contents, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsAnythingString(t *testing.T) {
	span := supportsSpan("(foo)", 0, 5)
	contents := NewInterpolationPlain("foo", supportsSpan("foo", 1, 4))
	cond := NewSupportsAnything(contents, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(foo)" {
		t.Errorf("String() = %q, want %q", got, "(foo)")
	}
}

func TestSupportsAnythingWithSpan(t *testing.T) {
	span1 := supportsSpan("(foo)", 0, 5)
	span2 := supportsSpan("(bar)", 0, 5)
	contents := NewInterpolationPlain("foo", supportsSpan("foo", 1, 4))
	cond := NewSupportsAnything(contents, span1)
	replaced := cond.WithSpan(span2)

	got, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

// --- SupportsDeclaration ---

func TestSupportsDeclarationConstruction(t *testing.T) {
	span := supportsSpan("(a: b)", 0, 6)
	name := NewStringExpressionPlain("a", supportsSpan("a", 1, 2), false)
	val := NewStringExpressionPlain("b", supportsSpan("b", 4, 5), false)
	cond := NewSupportsDeclaration(name, val, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsDeclarationString(t *testing.T) {
	span := supportsSpan("(a: b)", 0, 6)
	name := NewStringExpressionPlain("a", supportsSpan("a", 1, 2), false)
	val := NewStringExpressionPlain("b", supportsSpan("b", 4, 5), false)
	cond := NewSupportsDeclaration(name, val, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(a: b)" {
		t.Errorf("String() = %q, want %q", got, "(a: b)")
	}
}

func TestSupportsDeclarationWithSpan(t *testing.T) {
	span1 := supportsSpan("(a: b)", 0, 6)
	span2 := supportsSpan("(x: y)", 0, 6)
	name := NewStringExpressionPlain("a", supportsSpan("a", 1, 2), false)
	val := NewStringExpressionPlain("b", supportsSpan("b", 4, 5), false)
	cond := NewSupportsDeclaration(name, val, span1)
	replaced := cond.WithSpan(span2)
	_, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
}

// --- SupportsFunction ---

func TestSupportsFunctionConstruction(t *testing.T) {
	span := supportsSpan("fn(args)", 0, 8)
	nameInterp := NewInterpolationPlain("fn", supportsSpan("fn", 0, 2))
	argsInterp := NewInterpolationPlain("args", supportsSpan("args", 3, 7))
	cond := NewSupportsFunction(nameInterp, argsInterp, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsFunctionString(t *testing.T) {
	span := supportsSpan("fn(args)", 0, 8)
	nameInterp := NewInterpolationPlain("fn", supportsSpan("fn", 0, 2))
	argsInterp := NewInterpolationPlain("args", supportsSpan("args", 3, 7))
	cond := NewSupportsFunction(nameInterp, argsInterp, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "fn(args)" {
		t.Errorf("String() = %q, want %q", got, "fn(args)")
	}
}

func TestSupportsFunctionWithSpan(t *testing.T) {
	span1 := supportsSpan("fn(args)", 0, 8)
	span2 := supportsSpan("fn(x)", 0, 5)
	nameInterp := NewInterpolationPlain("fn", supportsSpan("fn", 0, 2))
	argsInterp := NewInterpolationPlain("args", supportsSpan("args", 3, 7))
	cond := NewSupportsFunction(nameInterp, argsInterp, span1)
	replaced := cond.WithSpan(span2)
	_, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
}

// --- SupportsInterpolation ---

func TestSupportsInterpolationConstruction(t *testing.T) {
	span := supportsSpan("#{x}", 0, 4)
	expr := NewBooleanExpression(true, supportsSpan("true", 2, 3))
	cond := NewSupportsInterpolation(expr, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsInterpolationString(t *testing.T) {
	span := supportsSpan("#{x}", 0, 4)
	expr := NewBooleanExpression(true, supportsSpan("true", 2, 3))
	cond := NewSupportsInterpolation(expr, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "#{true}" {
		t.Errorf("String() = %q, want %q", got, "#{true}")
	}
}

func TestSupportsInterpolationWithSpan(t *testing.T) {
	span1 := supportsSpan("#{x}", 0, 4)
	span2 := supportsSpan("#{y}", 0, 4)
	expr := NewBooleanExpression(true, supportsSpan("x", 2, 3))
	cond := NewSupportsInterpolation(expr, span1)
	replaced := cond.WithSpan(span2)
	_, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
}

// --- SupportsNegation ---

func TestSupportsNegationConstruction(t *testing.T) {
	span := supportsSpan("not (foo)", 0, 9)
	inner := NewSupportsAnything(
		NewInterpolationPlain("foo", supportsSpan("foo", 5, 8)),
		supportsSpan("(foo)", 4, 9),
	)
	cond := NewSupportsNegation(inner, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsNegationString(t *testing.T) {
	span := supportsSpan("not (foo)", 0, 9)
	inner := NewSupportsAnything(
		NewInterpolationPlain("foo", supportsSpan("foo", 5, 8)),
		supportsSpan("(foo)", 4, 9),
	)
	cond := NewSupportsNegation(inner, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "not (foo)" {
		t.Errorf("String() = %q, want %q", got, "not (foo)")
	}
}

func TestSupportsNegationWithSpan(t *testing.T) {
	span1 := supportsSpan("not (foo)", 0, 9)
	span2 := supportsSpan("not (bar)", 0, 9)
	inner := NewSupportsAnything(
		NewInterpolationPlain("foo", supportsSpan("foo", 5, 8)),
		supportsSpan("(foo)", 4, 9),
	)
	cond := NewSupportsNegation(inner, span1)
	replaced := cond.WithSpan(span2)
	_, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSupportsNegationNegatedInner(t *testing.T) {
	span := supportsSpan("not (not x)", 0, 11)
	expr := NewBooleanExpression(true, supportsSpan("x", 9, 10))
	innerInner := NewSupportsInterpolation(expr, supportsSpan("#{x}", 9, 10))
	innerNeg := NewSupportsNegation(innerInner, supportsSpan("not x", 5, 10))
	cond := NewSupportsNegation(innerNeg, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "not (not #{true})" {
		t.Errorf("String() = %q, want %q", got, "not (not #{true})")
	}
}

func TestSupportsNegationOperationInner(t *testing.T) {
	span := supportsSpan("not (a and b)", 0, 13)
	left := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("a", 5, 6)),
		supportsSpan("#{a}", 5, 6),
	)
	right := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("b", 11, 12)),
		supportsSpan("#{b}", 11, 12),
	)
	innerOp := NewSupportsOperation(left, right, BooleanOperatorAnd, supportsSpan("a and b", 5, 12))
	cond := NewSupportsNegation(innerOp, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "not (#{true} and #{true})" {
		t.Errorf("String() = %q, want %q", got, "not (#{true} and #{true})")
	}
}

// --- SupportsOperation ---

func TestSupportsOperationConstruction(t *testing.T) {
	span := supportsSpan("a and b", 0, 7)
	left := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("a", 0, 1)),
		supportsSpan("#{a}", 0, 1),
	)
	right := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("b", 6, 7)),
		supportsSpan("#{b}", 6, 7),
	)
	cond := NewSupportsOperation(left, right, BooleanOperatorAnd, span)

	got, err := cond.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}

func TestSupportsOperationString(t *testing.T) {
	span := supportsSpan("a and b", 0, 7)
	left := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("a", 0, 1)),
		supportsSpan("#{a}", 0, 1),
	)
	right := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("b", 6, 7)),
		supportsSpan("#{b}", 6, 7),
	)
	cond := NewSupportsOperation(left, right, BooleanOperatorAnd, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "#{true} and #{true}" {
		t.Errorf("String() = %q, want %q", got, "#{true} and #{true}")
	}
}

func TestSupportsOperationWithSpan(t *testing.T) {
	span1 := supportsSpan("a and b", 0, 7)
	span2 := supportsSpan("x or y", 0, 6)
	left := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("a", 0, 1)),
		supportsSpan("#{a}", 0, 1),
	)
	right := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("b", 6, 7)),
		supportsSpan("#{b}", 6, 7),
	)
	cond := NewSupportsOperation(left, right, BooleanOperatorAnd, span1)
	replaced := cond.WithSpan(span2)
	_, err := replaced.Span()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSupportsOperationSameOp(t *testing.T) {
	span := supportsSpan("(a and b) and c", 0, 15)
	innerLeft := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("a", 1, 2)),
		supportsSpan("#{a}", 1, 2),
	)
	innerRight := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("b", 7, 8)),
		supportsSpan("#{b}", 7, 8),
	)
	inner := NewSupportsOperation(innerLeft, innerRight, BooleanOperatorAnd, supportsSpan("a and b", 1, 8))
	right := NewSupportsInterpolation(
		NewBooleanExpression(true, supportsSpan("c", 14, 15)),
		supportsSpan("#{c}", 14, 15),
	)
	cond := NewSupportsOperation(inner, right, BooleanOperatorAnd, span)

	got, err := cond.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(#{true} and #{true}) and #{true}" {
		t.Errorf("String() = %q, want %q", got, "(#{true} and #{true}) and #{true}")
	}
}
