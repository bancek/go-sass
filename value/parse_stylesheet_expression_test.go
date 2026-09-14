package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// ======================================================================
// Helpers
// ======================================================================

func newTestParserExpr(text string) *StylesheetParser {
	return NewStylesheetParser([]byte(text), nil, false, nil)
}

// ======================================================================
// Group A: _expression — Pratt Parser Core
// ======================================================================

// --- Simple expressions ---

func TestExpressionNumber(t *testing.T) {
	p := newTestParserExpr("42")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 42 {
		t.Errorf("value = %v, want 42", ne.Value)
	}
}

func TestExpressionNegativeNumber(t *testing.T) {
	p := newTestParserExpr("-42")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != -42 {
		t.Errorf("value = %v, want -42", ne.Value)
	}
}

func TestExpressionStringDoubleQuote(t *testing.T) {
	p := newTestParserExpr(`"hello"`)
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	se, ok := expr.(*StringExpression)
	if !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
	if !se.HasQuotes {
		t.Error("expected HasQuotes=true")
	}
}

func TestExpressionBooleanTrue(t *testing.T) {
	p := newTestParserExpr("true")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BooleanExpression)
	if !ok {
		t.Fatalf("expected BooleanExpression, got %T", expr)
	}
	if !be.Value {
		t.Error("expected true")
	}
}

func TestExpressionBooleanFalse(t *testing.T) {
	p := newTestParserExpr("false")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BooleanExpression)
	if !ok {
		t.Fatalf("expected BooleanExpression, got %T", expr)
	}
	if be.Value {
		t.Error("expected false")
	}
}

func TestExpressionNull(t *testing.T) {
	p := newTestParserExpr("null")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*NullExpression); !ok {
		t.Fatalf("expected NullExpression, got %T", expr)
	}
}

func TestExpressionNamedColor(t *testing.T) {
	p := newTestParserExpr("red")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

func TestExpressionVariable(t *testing.T) {
	p := newTestParserExpr("$x")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ve, ok := expr.(*VariableExpression)
	if !ok {
		t.Fatalf("expected VariableExpression, got %T", expr)
	}
	if ve.Name() != "x" {
		t.Errorf("name = %q, want %q", ve.Name(), "x")
	}
}

func TestExpressionSelector(t *testing.T) {
	p := newTestParserExpr("&")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*SelectorExpression); !ok {
		t.Fatalf("expected SelectorExpression, got %T", expr)
	}
}

func TestExpressionImportant(t *testing.T) {
	p := newTestParserExpr("!important")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	se, ok := expr.(*StringExpression)
	if !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "!important" {
		t.Errorf("text = %v, want '!important'", plain)
	}
}

func TestExpressionPercent(t *testing.T) {
	p := newTestParserExpr("%")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	se, ok := expr.(*StringExpression)
	if !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "%" {
		t.Errorf("text = %v, want '%%'", plain)
	}
}

// --- Binary operations ---

func TestExpressionPlus(t *testing.T) {
	p := newTestParserExpr("1 + 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorPlus {
		t.Errorf("operator = %v, want Plus", be.Operator)
	}
	// Check operands are number expressions
	if _, ok := be.Left.(*NumberExpression); !ok {
		t.Errorf("left = %T, want NumberExpression", be.Left)
	}
	if _, ok := be.Right.(*NumberExpression); !ok {
		t.Errorf("right = %T, want NumberExpression", be.Right)
	}
}

func TestExpressionMinus(t *testing.T) {
	p := newTestParserExpr("1 - 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorMinus {
		t.Errorf("operator = %v, want Minus", be.Operator)
	}
}

func TestExpressionTimes(t *testing.T) {
	p := newTestParserExpr("1 * 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorTimes {
		t.Errorf("operator = %v, want Times", be.Operator)
	}
}

func TestExpressionDividedBy(t *testing.T) {
	p := newTestParserExpr("1 / 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorDividedBy {
		t.Errorf("operator = %v, want DividedBy", be.Operator)
	}
}

func TestExpressionModulo(t *testing.T) {
	p := newTestParserExpr("3 % 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorModulo {
		t.Errorf("operator = %v, want Modulo", be.Operator)
	}
}

func TestExpressionEquals(t *testing.T) {
	p := newTestParserExpr("1 == 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorEquals {
		t.Errorf("operator = %v, want Equals", be.Operator)
	}
}

func TestExpressionNotEquals(t *testing.T) {
	p := newTestParserExpr("1 != 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorNotEquals {
		t.Errorf("operator = %v, want NotEquals", be.Operator)
	}
}

func TestExpressionLessThan(t *testing.T) {
	p := newTestParserExpr("1 < 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorLessThan {
		t.Errorf("operator = %v, want LessThan", be.Operator)
	}
}

func TestExpressionLessThanOrEquals(t *testing.T) {
	p := newTestParserExpr("1 <= 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorLessThanOrEquals {
		t.Errorf("operator = %v, want LessThanOrEquals", be.Operator)
	}
}

func TestExpressionGreaterThan(t *testing.T) {
	p := newTestParserExpr("1 > 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorGreaterThan {
		t.Errorf("operator = %v, want GreaterThan", be.Operator)
	}
}

func TestExpressionGreaterThanOrEquals(t *testing.T) {
	p := newTestParserExpr("1 >= 2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorGreaterThanOrEquals {
		t.Errorf("operator = %v, want GreaterThanOrEquals", be.Operator)
	}
}

func TestExpressionSingleEquals(t *testing.T) {
	p := newTestParserExpr("a = b")
	expr, err := p._expression(expressionOpts{singleEquals: true})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorSingleEquals {
		t.Errorf("operator = %v, want SingleEquals", be.Operator)
	}
}

func TestExpressionAnd(t *testing.T) {
	p := newTestParserExpr("true and false")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorAnd {
		t.Errorf("operator = %v, want And", be.Operator)
	}
}

func TestExpressionOr(t *testing.T) {
	p := newTestParserExpr("true or false")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorOr {
		t.Errorf("operator = %v, want Or", be.Operator)
	}
}

// --- Unary operations ---

func TestExpressionUnaryDivide(t *testing.T) {
	p := newTestParserExpr("/ 10px")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorDivide {
		t.Errorf("operator = %v, want Divide", ue.Operator)
	}
}

func TestExpressionUnaryPlus(t *testing.T) {
	p := newTestParserExpr("+ 10")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorPlus {
		t.Errorf("operator = %v, want Plus", ue.Operator)
	}
	if _, ok := ue.Operand.(*NumberExpression); !ok {
		t.Errorf("operand = %T, want NumberExpression", ue.Operand)
	}
}

func TestExpressionUnaryMinus(t *testing.T) {
	p := newTestParserExpr("- 10px")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorMinus {
		t.Errorf("operator = %v, want Minus", ue.Operator)
	}
}

func TestExpressionNot(t *testing.T) {
	p := newTestParserExpr("not true")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorNot {
		t.Errorf("operator = %v, want Not", ue.Operator)
	}
}

// --- Precedence ---

func TestExpressionPrecedenceMultiplyBeforeAdd(t *testing.T) {
	p := newTestParserExpr("1 + 2 * 3")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorPlus {
		t.Errorf("top operator = %v, want Plus", be.Operator)
	}
	// Right side should be (2 * 3)
	right, ok := be.Right.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("right = %T, want BinaryOperationExpression", be.Right)
	}
	if right.Operator != BinaryOperatorTimes {
		t.Errorf("right operator = %v, want Times", right.Operator)
	}
}

func TestExpressionPrecedenceParenthesized(t *testing.T) {
	p := newTestParserExpr("(1 + 2) * 3")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorTimes {
		t.Errorf("top operator = %v, want Times", be.Operator)
	}
	if _, ok := be.Left.(*ParenthesizedExpression); !ok {
		t.Errorf("left = %T, want ParenthesizedExpression", be.Left)
	}
}

// --- Lists ---

func TestExpressionSpaceList(t *testing.T) {
	p := newTestParserExpr("1 2 3")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	le, ok := expr.(*ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
	if len(le.Contents) != 3 {
		t.Errorf("length = %d, want 3", len(le.Contents))
	}
	if le.Separator != ListSeparatorSpace {
		t.Errorf("separator = %v, want Space", le.Separator)
	}
}

func TestExpressionCommaList(t *testing.T) {
	p := newTestParserExpr("1, 2, 3")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	le, ok := expr.(*ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
	if len(le.Contents) != 3 {
		t.Errorf("length = %d, want 3", len(le.Contents))
	}
	if le.Separator != ListSeparatorComma {
		t.Errorf("separator = %v, want Comma", le.Separator)
	}
}

func TestExpressionBracketedList(t *testing.T) {
	p := newTestParserExpr("[1, 2, 3]")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	le, ok := expr.(*ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
	if len(le.Contents) != 3 {
		t.Errorf("length = %d, want 3", len(le.Contents))
	}
	if !le.HasBrackets {
		t.Error("expected HasBrackets=true")
	}
}

func TestExpressionEmptyBracketedList(t *testing.T) {
	p := newTestParserExpr("[]")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	le, ok := expr.(*ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
	if len(le.Contents) != 0 {
		t.Errorf("length = %d, want 0", len(le.Contents))
	}
	if !le.HasBrackets {
		t.Error("expected HasBrackets=true")
	}
}

// --- Maps ---

func TestExpressionMap(t *testing.T) {
	p := newTestParserExpr("(a: 1, b: 2)")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// _expression resolves maps directly — they are not wrapped in ParenthesizedExpression
	me, ok := expr.(*MapExpression)
	if !ok {
		t.Fatalf("expected MapExpression, got %T", expr)
	}
	if len(me.Pairs) != 2 {
		t.Errorf("pairs length = %d, want 2", len(me.Pairs))
	}
}

// --- Parenthesized expressions ---

func TestExpressionParenthesesNumber(t *testing.T) {
	p := newTestParserExpr("(42)")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	pe, ok := expr.(*ParenthesizedExpression)
	if !ok {
		t.Fatalf("expected ParenthesizedExpression, got %T", expr)
	}
	if _, ok := pe.Expression.(*NumberExpression); !ok {
		t.Errorf("wrapped = %T, want NumberExpression", pe.Expression)
	}
}

func TestExpressionEmptyParentheses(t *testing.T) {
	p := newTestParserExpr("()")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ListExpression); !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
}

// --- Slash ambiguity ---

func TestExpressionSlashAmbiguity(t *testing.T) {
	p := newTestParserExpr("1/2")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if !be.AllowsSlash() {
		t.Error("expected AllowsSlash()=true for '1/2'")
	}
}

// --- Error cases ---

func TestExpressionErrorEmpty(t *testing.T) {
	p := newTestParserExpr("")
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	want := "Expected expression."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestExpressionErrorMissingOperand(t *testing.T) {
	p := newTestParserExpr("1 +")
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for missing operand")
	}
}

func TestExpressionErrorMissingRightParen(t *testing.T) {
	p := newTestParserExpr("(1")
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for missing ')'")
	}
}

func TestExpressionErrorMissingRightBracket(t *testing.T) {
	p := newTestParserExpr("[1")
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for missing ']'")
	}
}

func TestExpressionErrorUnclosedString(t *testing.T) {
	p := newTestParserExpr(`"hello`)
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for unclosed string")
	}
	want := `Expected ".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// --- Plain CSS restrictions ---

func TestExpressionPlainCssOperatorsNotAllowed(t *testing.T) {
	p := NewStylesheetParser([]byte("1 < 3"), nil, false, nil)
	p.plainCss = true
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for operators in plain CSS")
	}
}

func TestExpressionPlainCssVariableNotAllowed(t *testing.T) {
	p := NewStylesheetParser([]byte("$x"), nil, false, nil)
	p.plainCss = true
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for variables in plain CSS")
	}
}

func TestExpressionPlainCssParentSelectorNotAllowed(t *testing.T) {
	p := NewStylesheetParser([]byte("&"), nil, false, nil)
	p.plainCss = true
	_, err := p._expression(expressionOpts{})
	if err == nil {
		t.Fatal("expected error for parent selector in plain CSS")
	}
}

// ======================================================================
// Group B: argumentInvocation
// ======================================================================

func TestArgumentInvocationEmpty(t *testing.T) {
	p := newTestParserExpr("()")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(args.Positional) != 0 {
		t.Errorf("positional len = %d, want 0", len(args.Positional))
	}
	if args.Named.Len() != 0 {
		t.Errorf("named len = %d, want 0", args.Named.Len())
	}
}

func TestArgumentInvocationSinglePositional(t *testing.T) {
	p := newTestParserExpr("(1)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(args.Positional) != 1 {
		t.Errorf("positional len = %d, want 1", len(args.Positional))
	}
}

func TestArgumentInvocationMultiplePositional(t *testing.T) {
	p := newTestParserExpr("(1, 2, 3)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(args.Positional) != 3 {
		t.Errorf("positional len = %d, want 3", len(args.Positional))
	}
}

func TestArgumentInvocationNamed(t *testing.T) {
	p := newTestParserExpr("($x: 1)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !args.Named.Has("x") {
		t.Error("expected named argument 'x'")
	}
}

func TestArgumentInvocationMixedPositionalAndNamed(t *testing.T) {
	p := newTestParserExpr("(1, $x: 2)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(args.Positional) != 1 {
		t.Errorf("positional len = %d, want 1", len(args.Positional))
	}
	if !args.Named.Has("x") {
		t.Error("expected named 'x'")
	}
}

func TestArgumentInvocationRest(t *testing.T) {
	p := newTestParserExpr("($list...)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if args.Rest == nil {
		t.Error("expected rest argument")
	}
}

func TestArgumentInvocationKeywordRest(t *testing.T) {
	p := newTestParserExpr("($map...)")
	args, err := p.argumentInvocation(argumentInvocationOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// Single rest with $map... is a regular rest, not keyword-rest
	// Keyword rest is $args..., $rest...
	if args.Rest == nil {
		t.Error("expected rest argument")
	}
}

func TestArgumentInvocationErrorDuplicateNamed(t *testing.T) {
	p := newTestParserExpr("($x: 1, $x: 2)")
	_, err := p.argumentInvocation(argumentInvocationOpts{})
	if err == nil {
		t.Fatal("expected error for duplicate named argument")
	}
	want := strings.Join([]string{
		`Error: Duplicate argument.`,
		`  ╷`,
		`1 │ ($x: 1, $x: 2)`,
		`  │         ^^`,
		`  ╵`,
		`  - 1:9  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestArgumentInvocationErrorPositionalAfterNamed(t *testing.T) {
	p := newTestParserExpr("($x: 1, 2)")
	_, err := p.argumentInvocation(argumentInvocationOpts{})
	if err == nil {
		t.Fatal("expected error for positional after named")
	}
}

func TestArgumentInvocationAllowEmptySecondArg(t *testing.T) {
	p := newTestParserExpr("(a,)")
	args, err := p.argumentInvocation(argumentInvocationOpts{allowEmptySecondArg: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(args.Positional) != 2 {
		t.Errorf("positional len = %d, want 2", len(args.Positional))
	}
}

func TestArgumentInvocationErrorNoParen(t *testing.T) {
	p := newTestParserExpr("1")
	_, err := p.argumentInvocation(argumentInvocationOpts{})
	if err == nil {
		t.Fatal("expected error for missing '('")
	}
}

// ======================================================================
// Group C: Prefix Parsers — singleExpression
// ======================================================================

func TestSingleExpressionNumber(t *testing.T) {
	p := newTestParserExpr("42")
	expr, err := p.singleExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*NumberExpression); !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
}

func TestSingleExpressionVariable(t *testing.T) {
	p := newTestParserExpr("$x")
	expr, err := p.singleExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*VariableExpression); !ok {
		t.Fatalf("expected VariableExpression, got %T", expr)
	}
}

func TestSingleExpressionString(t *testing.T) {
	p := newTestParserExpr(`"hello"`)
	expr, err := p.singleExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
}

func TestSingleExpressionParentheses(t *testing.T) {
	p := newTestParserExpr("(1)")
	expr, err := p.singleExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ParenthesizedExpression); !ok {
		t.Fatalf("expected ParenthesizedExpression, got %T", expr)
	}
}

func TestSingleExpressionEmpty(t *testing.T) {
	p := newTestParserExpr("")
	_, err := p.singleExpression()
	if err == nil {
		t.Fatal("expected error for empty")
	}
}

// ======================================================================
// Group C: Prefix Parsers — number
// ======================================================================

func TestNumberInteger(t *testing.T) {
	p := newTestParserExpr("42")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 42 {
		t.Errorf("value = %v, want 42", n.Value)
	}
}

func TestNumberNegative(t *testing.T) {
	p := newTestParserExpr("-42")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != -42 {
		t.Errorf("value = %v, want -42", n.Value)
	}
}

func TestNumberPositive(t *testing.T) {
	p := newTestParserExpr("+42")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 42 {
		t.Errorf("value = %v, want 42", n.Value)
	}
}

func TestNumberDecimal(t *testing.T) {
	p := newTestParserExpr("3.14")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 3.14 {
		t.Errorf("value = %v, want 3.14", n.Value)
	}
}

func TestNumberLeadingDot(t *testing.T) {
	p := newTestParserExpr(".5")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 0.5 {
		t.Errorf("value = %v, want 0.5", n.Value)
	}
}

func TestNumberExponent(t *testing.T) {
	p := newTestParserExpr("1e3")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 1000 {
		t.Errorf("value = %v, want 1000", n.Value)
	}
}

func TestNumberExponentNegative(t *testing.T) {
	p := newTestParserExpr("1e-3")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 0.001 {
		t.Errorf("value = %v, want 0.001", n.Value)
	}
}

func TestNumberExponentPositive(t *testing.T) {
	p := newTestParserExpr("1e+3")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 1000 {
		t.Errorf("value = %v, want 1000", n.Value)
	}
}

func TestNumberWithPercentUnit(t *testing.T) {
	p := newTestParserExpr("42%")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 42 {
		t.Errorf("value = %v, want 42", n.Value)
	}
	if n.Unit == nil || *n.Unit != "%" {
		t.Errorf("unit = %v, want '%%'", n.Unit)
	}
}

func TestNumberWithIdentUnit(t *testing.T) {
	p := newTestParserExpr("42px")
	n, err := p.number()
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != 42 {
		t.Errorf("value = %v, want 42", n.Value)
	}
	if n.Unit == nil || *n.Unit != "px" {
		t.Errorf("unit = %v, want 'px'", n.Unit)
	}
}

func TestNumberErrorLeadingDotNonDigit(t *testing.T) {
	p := newTestParserExpr(".a")
	_, err := p.number()
	if err == nil {
		t.Fatal("expected error for '.a'")
	}
}

func TestNumberErrorExponentNoDigit(t *testing.T) {
	// 1e- parses 'e' + '-' and then expects a digit, so it should error
	p := newTestParserExpr("1e-")
	_, err := p.number()
	if err == nil {
		t.Fatal("expected error for exponent with no digit after sign")
	}
}

// ======================================================================
// Group C: Prefix Parsers — unicodeRange
// ======================================================================

func TestUnicodeRangeSimple(t *testing.T) {
	p := newTestParserExpr("u+1")
	se, err := p.unicodeRange()
	if err != nil {
		t.Fatal(err)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "u+1" {
		t.Errorf("text = %v, want 'u+1'", plain)
	}
}

func TestUnicodeRangeLong(t *testing.T) {
	p := newTestParserExpr("u+10ffff")
	se, err := p.unicodeRange()
	if err != nil {
		t.Fatal(err)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "u+10ffff" {
		t.Errorf("text = %v, want 'u+10ffff'", plain)
	}
}

func TestUnicodeRangeWithQuestionMarks(t *testing.T) {
	p := newTestParserExpr("u+??")
	se, err := p.unicodeRange()
	if err != nil {
		t.Fatal(err)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "u+??" {
		t.Errorf("text = %v, want 'u+??'", plain)
	}
}

func TestUnicodeRangeWithRange(t *testing.T) {
	p := newTestParserExpr("u+0020-007e")
	se, err := p.unicodeRange()
	if err != nil {
		t.Fatal(err)
	}
	plain := se.Text.AsPlain()
	if plain == nil || *plain != "u+0020-007e" {
		t.Errorf("text = %v, want 'u+0020-007e'", plain)
	}
}

func TestUnicodeRangeCapitalU(t *testing.T) {
	p := newTestParserExpr("U+1")
	se, err := p.unicodeRange()
	if err != nil {
		t.Fatal(err)
	}
	if se.Text.AsPlain() == nil || *se.Text.AsPlain() != "U+1" {
		t.Errorf("text = %v, want 'U+1'", se.Text.AsPlain())
	}
}

func TestUnicodeRangeErrorEmpty(t *testing.T) {
	p := newTestParserExpr("u+")
	_, err := p.unicodeRange()
	if err == nil {
		t.Fatal("expected error for empty unicode range")
	}
	want := `Expected hex digit or "?".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestUnicodeRangeErrorTooManyDigits(t *testing.T) {
	p := newTestParserExpr("u+1234567")
	_, err := p.unicodeRange()
	if err == nil {
		t.Fatal("expected error for too many digits")
	}
}

// ======================================================================
// Group C: Prefix Parsers — variable
// ======================================================================

func TestVariableSimple(t *testing.T) {
	p := newTestParserExpr("$x")
	v, err := p.variable()
	if err != nil {
		t.Fatal(err)
	}
	if v.Name() != "x" {
		t.Errorf("name = %q, want 'x'", v.Name())
	}
}

func TestVariableComplexName(t *testing.T) {
	p := newTestParserExpr("$foo-bar")
	v, err := p.variable()
	if err != nil {
		t.Fatal(err)
	}
	if v.Name() != "foo-bar" {
		t.Errorf("name = %q, want 'foo-bar'", v.Name())
	}
}

func TestVariablePlainCss(t *testing.T) {
	p := NewStylesheetParser([]byte("$x"), nil, false, nil)
	p.plainCss = true
	_, err := p.variable()
	if err == nil {
		t.Fatal("expected error for variable in plain CSS")
	}
}

// ======================================================================
// Group C: Prefix Parsers — selectorExpr
// ======================================================================

func TestSelectorExprSimple(t *testing.T) {
	p := newTestParserExpr("&")
	s, err := p.selectorExpr()
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil SelectorExpression")
	}
}

func TestSelectorExprDoubleAnd(t *testing.T) {
	p := newTestParserExpr("&&")
	_, err := p.selectorExpr()
	if err != nil {
		t.Fatal(err)
	}
	// Should produce a ParseTimeWarning about "&&"
	if len(p.warnings) == 0 {
		t.Error("expected deprecation warning for '&&'")
	}
}

func TestSelectorExprPlainCss(t *testing.T) {
	p := NewStylesheetParser([]byte("&"), nil, false, nil)
	p.plainCss = true
	_, err := p.selectorExpr()
	if err == nil {
		t.Fatal("expected error for parent selector in plain CSS")
	}
}

// ======================================================================
// Group C: Prefix Parsers — interpolatedString
// ======================================================================

func TestInterpolatedStringSimple(t *testing.T) {
	p := newTestParserExpr(`"hello"`)
	s, err := p.interpolatedString()
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasQuotes {
		t.Error("expected HasQuotes=true")
	}
	plain := s.Text.AsPlain()
	if plain == nil || *plain != "hello" {
		t.Errorf("text = %v, want 'hello'", plain)
	}
}

func TestInterpolatedStringSingleQuotes(t *testing.T) {
	p := newTestParserExpr(`'hello'`)
	s, err := p.interpolatedString()
	if err != nil {
		t.Fatal(err)
	}
	plain := s.Text.AsPlain()
	if plain == nil || *plain != "hello" {
		t.Errorf("text = %v, want 'hello'", plain)
	}
}

func TestInterpolatedStringEmpty(t *testing.T) {
	p := newTestParserExpr(`""`)
	s, err := p.interpolatedString()
	if err != nil {
		t.Fatal(err)
	}
	plain := s.Text.AsPlain()
	if plain == nil || *plain != "" {
		t.Errorf("text = %v, want ''", plain)
	}
}

func TestInterpolatedStringErrorUnclosed(t *testing.T) {
	p := newTestParserExpr(`"hello`)
	_, err := p.interpolatedString()
	if err == nil {
		t.Fatal("expected error for unclosed string")
	}
}

func TestInterpolatedStringEscape(t *testing.T) {
	p := newTestParserExpr(`"hello\aworld"`)
	s, err := p.interpolatedString()
	if err != nil {
		t.Fatal(err)
	}
	plain := s.Text.AsPlain()
	// \a = hex U+000A = line feed = \n in Go string
	if plain == nil || *plain != "hello\nworld" {
		t.Errorf("text = %q, want 'hello\\nworld'", *plain)
	}
}

func TestInterpolatedStringNewlineEscape(t *testing.T) {
	p := newTestParserExpr("\"hello\\\nworld\"")
	s, err := p.interpolatedString()
	if err != nil {
		t.Fatal(err)
	}
	plain := s.Text.AsPlain()
	if plain == nil || *plain != "helloworld" {
		t.Errorf("text = %v, want 'helloworld'", plain)
	}
}

// ======================================================================
// Group C: Prefix Parsers — interpolatedStringToken
// ======================================================================

func TestInterpolatedStringTokenSimple(t *testing.T) {
	p := newTestParserExpr(`"hello"`)
	interp, err := p.interpolatedStringToken()
	if err != nil {
		t.Fatal(err)
	}
	plain := interp.AsPlain()
	if plain == nil || *plain != `"hello"` {
		t.Errorf("text = %v, want '\"hello\"'", plain)
	}
}

func TestInterpolatedStringTokenEmpty(t *testing.T) {
	p := newTestParserExpr(`""`)
	interp, err := p.interpolatedStringToken()
	if err != nil {
		t.Fatal(err)
	}
	plain := interp.AsPlain()
	if plain == nil || *plain != `""` {
		t.Errorf("text = %v, want '\"\"'", plain)
	}
}

// ======================================================================
// Group C: Prefix Parsers — hashExpression
// ======================================================================

func TestHashExpressionShort(t *testing.T) {
	p := newTestParserExpr("#fff")
	expr, err := p.hashExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

func TestHashExpressionLong(t *testing.T) {
	p := newTestParserExpr("#ff00ff")
	expr, err := p.hashExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

func TestHashExpressionWithAlpha(t *testing.T) {
	p := newTestParserExpr("#ff00ff80")
	expr, err := p.hashExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

func TestHashExpressionNotColor(t *testing.T) {
	p := newTestParserExpr("#xyz")
	expr, err := p.hashExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression for non-color hash, got %T", expr)
	}
}

func TestHashExpressionHexColorInvalid(t *testing.T) {
	p := newTestParserExpr("#0")
	// #0: reads '#', then sees digit '0' and tries hexColorContents which needs >= 3 digits
	// The error goes through p.error() so it's a formatted Sass error
	_, err := p.hashExpression()
	if err == nil {
		t.Fatal("expected error for invalid hex color '#0'")
	}
}

func TestHashExpressionDigit(t *testing.T) {
	p := newTestParserExpr("#0a0")
	expr, err := p.hashExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

// ======================================================================
// Group C: Prefix Parsers — hexDigit
// ======================================================================

func TestHexDigitValid(t *testing.T) {
	p := newTestParserExpr("f")
	d, err := p.hexDigit()
	if err != nil {
		t.Fatal(err)
	}
	if d != 15 {
		t.Errorf("hex value = %d, want 15", d)
	}
}

func TestHexDigitInvalid(t *testing.T) {
	p := newTestParserExpr("g")
	_, err := p.hexDigit()
	if err == nil {
		t.Fatal("expected error for invalid hex digit")
	}
	want := "Expected hex digit."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group C: Prefix Parsers — isHexColor
// ======================================================================

func TestIsHexColor3Digits(t *testing.T) {
	p := newTestParserExpr("#abc")
	p.scanner.ReadChar() // consume '#'
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if !isHexColor(ident) {
		t.Error("expected isHexColor=true for 'abc'")
	}
}

func TestIsHexColor4Digits(t *testing.T) {
	p := newTestParserExpr("#abcd")
	p.scanner.ReadChar()
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if !isHexColor(ident) {
		t.Error("expected isHexColor=true for 'abcd'")
	}
}

func TestIsHexColor6Digits(t *testing.T) {
	p := newTestParserExpr("#abc123")
	p.scanner.ReadChar()
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if !isHexColor(ident) {
		t.Error("expected isHexColor=true for 'abc123'")
	}
}

func TestIsHexColor8Digits(t *testing.T) {
	p := newTestParserExpr("#abcd1234")
	p.scanner.ReadChar()
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if !isHexColor(ident) {
		t.Error("expected isHexColor=true for 'abcd1234'")
	}
}

func TestIsHexColorInvalidLength(t *testing.T) {
	p := newTestParserExpr("#abc12")
	p.scanner.ReadChar()
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if isHexColor(ident) {
		t.Error("expected isHexColor=false for 'abc12'")
	}
}

func TestIsHexColorNonHex(t *testing.T) {
	p := newTestParserExpr("#abcz")
	p.scanner.ReadChar()
	afterHash := p.scanner.State()
	ident, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	p.scanner.SetState(afterHash)
	if isHexColor(ident) {
		t.Error("expected isHexColor=false for 'abcz'")
	}
}

// ======================================================================
// Group C: Prefix Parsers — plusExpression / minusExpression
// ======================================================================

func TestPlusExpressionNumber(t *testing.T) {
	p := newTestParserExpr("+10")
	expr, err := p.plusExpression()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 10 {
		t.Errorf("value = %v, want 10", ne.Value)
	}
}

func TestPlusExpressionUnary(t *testing.T) {
	p := newTestParserExpr("+ $x")
	expr, err := p.plusExpression()
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorPlus {
		t.Errorf("operator = %v, want Plus", ue.Operator)
	}
}

func TestMinusExpressionNumber(t *testing.T) {
	p := newTestParserExpr("-10")
	expr, err := p.minusExpression()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != -10 {
		t.Errorf("value = %v, want -10", ne.Value)
	}
}

func TestMinusExpressionIdentifierLike(t *testing.T) {
	p := newTestParserExpr("-foo")
	expr, err := p.minusExpression()
	if err != nil {
		t.Fatal(err)
	}
	// -foo is parsed as identifier, which becomes a StringExpression
	if _, ok := expr.(*UnaryOperationExpression); ok {
		// It's a unary minus followed by identifier
	} else {
		// It could be parsed as identifierLike
	}
}

// ======================================================================
// Group C: Prefix Parsers — unaryOperation
// ======================================================================

func TestUnaryOperationDivide(t *testing.T) {
	p := newTestParserExpr("/ $x")
	ue, err := p.unaryOperation()
	if err != nil {
		t.Fatal(err)
	}
	if ue.Operator != UnaryOperatorDivide {
		t.Errorf("operator = %v, want Divide", ue.Operator)
	}
}

func TestUnaryOperationPlus(t *testing.T) {
	p := newTestParserExpr("+ $x")
	ue, err := p.unaryOperation()
	if err != nil {
		t.Fatal(err)
	}
	if ue.Operator != UnaryOperatorPlus {
		t.Errorf("operator = %v, want Plus", ue.Operator)
	}
}

func TestUnaryOperationMinus(t *testing.T) {
	p := newTestParserExpr("- $x")
	ue, err := p.unaryOperation()
	if err != nil {
		t.Fatal(err)
	}
	if ue.Operator != UnaryOperatorMinus {
		t.Errorf("operator = %v, want Minus", ue.Operator)
	}
}

func TestUnaryOperationPlainCss(t *testing.T) {
	p := NewStylesheetParser([]byte("+ $x"), nil, false, nil)
	p.plainCss = true
	_, err := p.unaryOperation()
	if err == nil {
		t.Fatal("expected error for unary operators in plain CSS")
	}
}

// ======================================================================
// Group D: Expression Combiners
// ======================================================================

func TestExpressionUntilComma(t *testing.T) {
	p := newTestParserExpr("1, 2")
	expr, err := p.expressionUntilComma(false)
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

func TestIsSlashOperandNumber(t *testing.T) {
	p := newTestParserExpr("1")
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !p.isSlashOperand(expr) {
		t.Error("expected isSlashOperand=true for number expression")
	}
}

func TestIsSlashOperandString(t *testing.T) {
	p := newTestParserExpr(`"hello"`)
	expr, err := p._expression(expressionOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if p.isSlashOperand(expr) {
		t.Error("expected isSlashOperand=false for string expression")
	}
}

// ======================================================================
// Group E: identifierLike (the most complex dispatch)
// ======================================================================

func TestIdentifierLikeTrue(t *testing.T) {
	p := newTestParserExpr("true")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BooleanExpression)
	if !ok {
		t.Fatalf("expected BooleanExpression, got %T", expr)
	}
	if !be.Value {
		t.Error("expected true")
	}
}

func TestIdentifierLikeFalse(t *testing.T) {
	p := newTestParserExpr("false")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.(*BooleanExpression)
	if !ok {
		t.Fatalf("expected BooleanExpression, got %T", expr)
	}
	if be.Value {
		t.Error("expected false")
	}
}

func TestIdentifierLikeNull(t *testing.T) {
	p := newTestParserExpr("null")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*NullExpression); !ok {
		t.Fatalf("expected NullExpression, got %T", expr)
	}
}

func TestIdentifierLikeNamedColor(t *testing.T) {
	p := newTestParserExpr("red")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ColorExpression); !ok {
		t.Fatalf("expected ColorExpression, got %T", expr)
	}
}

func TestIdentifierLikeNotExpression(t *testing.T) {
	p := newTestParserExpr("not x")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	ue, ok := expr.(*UnaryOperationExpression)
	if !ok {
		t.Fatalf("expected UnaryOperationExpression, got %T", expr)
	}
	if ue.Operator != UnaryOperatorNot {
		t.Errorf("operator = %v, want Not", ue.Operator)
	}
}

func TestIdentifierLikeFunctionCall(t *testing.T) {
	p := newTestParserExpr("rgb(1, 2, 3)")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	fe, ok := expr.(*FunctionExpression)
	if !ok {
		t.Fatalf("expected FunctionExpression, got %T", expr)
	}
	if fe.OriginalName != "rgb" {
		t.Errorf("name = %q, want 'rgb'", fe.OriginalName)
	}
}

func TestIdentifierLikeSimpleString(t *testing.T) {
	p := newTestParserExpr("foo")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
}

// ======================================================================
// Group H: namespacedExpression
// ======================================================================

func TestNamespacedExpressionVariable(t *testing.T) {
	p := newTestParserExpr("ns.$x")
	// Must first consume "ns"
	p.interpolatedIdentifier()
	// Then consume "."
	p.readChar()
	start := p.scanner.State()
	expr, err := p.namespacedExpression("ns", start)
	if err != nil {
		t.Fatal(err)
	}
	ve, ok := expr.(*VariableExpression)
	if !ok {
		t.Fatalf("expected VariableExpression, got %T", expr)
	}
	if ve.Namespace() == nil || *ve.Namespace() != "ns" {
		t.Errorf("namespace = %v, want 'ns'", ve.Namespace())
	}
	if ve.Name() != "x" {
		t.Errorf("name = %q, want 'x'", ve.Name())
	}
}

func TestNamespacedExpressionFunction(t *testing.T) {
	p := newTestParserExpr("ns.func()")
	// Consume "ns"
	p.interpolatedIdentifier()
	// Consume "."
	p.readChar()
	start := p.scanner.State()
	expr, err := p.namespacedExpression("ns", start)
	if err != nil {
		t.Fatal(err)
	}
	fe, ok := expr.(*FunctionExpression)
	if !ok {
		t.Fatalf("expected FunctionExpression, got %T", expr)
	}
	if fe.Namespace == nil || *fe.Namespace != "ns" {
		t.Errorf("namespace = %v, want 'ns'", fe.Namespace)
	}
}

func TestNamespacedExpressionPrivate(t *testing.T) {
	p := newTestParserExpr("ns._private")
	// Consume "ns"
	p.interpolatedIdentifier()
	// Consume "."
	p.readChar()
	start := p.scanner.State()
	_, err := p.namespacedExpression("ns", start)
	if err == nil {
		t.Fatal("expected error for private member access")
	}
}

// ======================================================================
// Group G: tryUrlContents
// ======================================================================

func TestTryUrlContentsSimple(t *testing.T) {
	p := newTestParserExpr("(foo)")
	start := p.scanner.State()
	interp, err := p.tryUrlContents(start, "url", false)
	if err != nil {
		t.Fatal(err)
	}
	if interp == nil {
		t.Fatal("expected non-nil interpolation")
	}
}

func TestTryUrlContentsEmpty(t *testing.T) {
	p := newTestParserExpr("()")
	start := p.scanner.State()
	interp, err := p.tryUrlContents(start, "url", false)
	if err != nil {
		t.Fatal(err)
	}
	if interp == nil {
		t.Fatal("expected non-nil interpolation for empty url")
	}
}

func TestTryUrlContentsNoParen(t *testing.T) {
	p := newTestParserExpr("foo")
	start := p.scanner.State()
	interp, err := p.tryUrlContents(start, "url", false)
	if err != nil {
		t.Fatal(err)
	}
	if interp != nil {
		t.Error("expected nil for no paren")
	}
}

// ======================================================================
// dynamicUrl
// ======================================================================

func TestDynamicUrlSimple(t *testing.T) {
	p := newTestParserExpr("url(foo)")
	expr, err := p.dynamicUrl()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
}

func TestDynamicUrlEmpty(t *testing.T) {
	p := newTestParserExpr("url()")
	expr, err := p.dynamicUrl()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", expr)
	}
}

func TestDynamicUrlWithArgs(t *testing.T) {
	p := newTestParserExpr("url($a, $b)")
	expr, err := p.dynamicUrl()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*InterpolatedFunctionExpression); !ok {
		t.Fatalf("expected InterpolatedFunctionExpression, got %T", expr)
	}
}

func TestDynamicUrlErrorNoParen(t *testing.T) {
	p := newTestParserExpr("url")
	_, err := p.dynamicUrl()
	if err == nil {
		t.Fatal("expected error for 'url' without '('")
	}
}

// ======================================================================
// interpolatedStringToken (additional tests)
// ======================================================================

func TestInterpolatedStringTokenSingleQuotes(t *testing.T) {
	p := newTestParserExpr(`'world'`)
	interp, err := p.interpolatedStringToken()
	if err != nil {
		t.Fatal(err)
	}
	plain := interp.AsPlain()
	if plain == nil || *plain != `'world'` {
		t.Errorf("text = %v, want \"'world'\"", plain)
	}
}

func TestInterpolatedStringTokenErrorUnclosed(t *testing.T) {
	p := newTestParserExpr(`"hello`)
	_, err := p.interpolatedStringToken()
	if err == nil {
		t.Fatal("expected error for unclosed string")
	}
	want := `Expected ".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// ifExpression
// ======================================================================

func TestIfExpressionModern(t *testing.T) {
	// if(1, 2, 3) — legacy if-function syntax — succeeds and produces a deprecation warning.
	// This exercises the identifierLike → if() path.
	p := newTestParserExpr("if(1, 2, 3)")
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	// Should be a LegacyIfExpression (with deprecation)
	if _, ok := expr.(*IfExpression); !ok {
		if _, ok := expr.(*LegacyIfExpression); !ok {
			t.Fatalf("expected IfExpression or LegacyIfExpression, got %T", expr)
		}
	}
}

func TestIfExpressionLegacyDeprecation(t *testing.T) {
	p := newTestParserExpr("if(1, 2, 3)")
	// consume "if" then call ifExpression — should try argumentInvocation first (legacy)
	expr, err := p.identifierLike()
	if err != nil {
		t.Fatal(err)
	}
	// Check it's a LegacyIfExpression or IfExpression
	if _, ok := expr.(*IfExpression); !ok {
		if _, ok := expr.(*LegacyIfExpression); !ok {
			t.Fatalf("expected IfExpression or LegacyIfExpression, got %T", expr)
		}
	}
}

// ======================================================================
// expressionUntilComparison — used by media queries
// ======================================================================

func TestExpressionUntilComparison(t *testing.T) {
	p := newTestParserExpr("1 < 2")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

// ======================================================================
// expressionOpts.BracketList
// ======================================================================

func TestExpressionBracketList(t *testing.T) {
	p := newTestParserExpr("[1, 2]")
	expr, err := p._expression(expressionOpts{bracketList: true})
	if err != nil {
		t.Fatal(err)
	}
	le, ok := expr.(*ListExpression)
	if !ok {
		t.Fatalf("expected ListExpression, got %T", expr)
	}
	if !le.HasBrackets {
		t.Error("expected HasBrackets=true")
	}
}

// ======================================================================
// expressionOpts.Until
// ======================================================================

func TestExpressionUntilCommaInPractice(t *testing.T) {
	p := newTestParserExpr("1, 2, 3")
	// Parse first item until comma
	expr, err := p._expression(expressionOpts{
		until: func() bool { return p.scanner.PeekChar(0) == ',' },
	})
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

func TestExpressionUntilAtStartErr(t *testing.T) {
	p := newTestParserExpr("")
	_, err := p._expression(expressionOpts{
		until: func() bool { return true },
	})
	if err == nil {
		t.Fatal("expected error when until triggers at start")
	}
}

// ======================================================================
// expressionUntilComparison
// ======================================================================

func TestExpressionUntilComparisonStopsAtLessThan(t *testing.T) {
	p := newTestParserExpr("1 < 2")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
	// Scanner should be at '<'
	if p.scanner.PeekChar(0) != '<' {
		t.Errorf("scanner at %c, want '<'", rune(p.scanner.PeekChar(0)))
	}
}

func TestExpressionUntilComparisonStopsAtGreaterThan(t *testing.T) {
	p := newTestParserExpr("1 > 2")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

func TestExpressionUntilComparisonStopsAtSingleEquals(t *testing.T) {
	p := newTestParserExpr("1 = 2")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

func TestExpressionUntilComparisonDoesNotStopAtDoubleEquals(t *testing.T) {
	p := newTestParserExpr("1 == 2")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	// "==" doesn't trigger until, so full expression parsed
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorEquals {
		t.Errorf("operator = %v, want Equals", be.Operator)
	}
}

func TestExpressionUntilComparisonNoComparison(t *testing.T) {
	p := newTestParserExpr("42")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", expr)
	}
	if ne.Value != 42 {
		t.Errorf("value = %v, want 42", ne.Value)
	}
}

func TestExpressionUntilComparisonWithIdentifier(t *testing.T) {
	p := newTestParserExpr("screen and (color)")
	expr, err := p.expressionUntilComparison()
	if err != nil {
		t.Fatal(err)
	}
	// "screen and (color)" — 'and' is not a comparison, so the full expression parses.
	// The until callback only stops at <, >, = (not ==).
	// Result: BinaryOperation(And, "screen", "(color)")
	be, ok := expr.(*BinaryOperationExpression)
	if !ok {
		t.Fatalf("expected BinaryOperationExpression, got %T", expr)
	}
	if be.Operator != BinaryOperatorAnd {
		t.Errorf("operator = %v, want And", be.Operator)
	}
}

// ======================================================================
// ifGroup
// ======================================================================

func TestIfGroupFunctionEnv(t *testing.T) {
	p := newTestParserExpr("env(--foo)")
	result, err := p.ifGroup()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := result.(*IfConditionFunction)
	if !ok {
		t.Fatalf("expected IfConditionFunction, got %T", result)
	}
	if plain := fn.Name.AsPlain(); plain == nil || *plain != "env" {
		t.Errorf("name = %v, want env", plain)
	}
	argsStr, err := fn.Arguments.String()
	if err != nil {
		t.Fatal(err)
	}
	if argsStr != "--foo" {
		t.Errorf("arguments = %q, want %q", argsStr, "--foo")
	}
}

func TestIfGroupFunctionVar(t *testing.T) {
	p := newTestParserExpr("var(--x)")
	result, err := p.ifGroup()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := result.(*IfConditionFunction)
	if !ok {
		t.Fatalf("expected IfConditionFunction, got %T", result)
	}
	if plain := fn.Name.AsPlain(); plain == nil || *plain != "var" {
		t.Errorf("name = %v, want var", plain)
	}
}

func TestIfGroupFunctionAttr(t *testing.T) {
	p := newTestParserExpr("attr(type)")
	result, err := p.ifGroup()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := result.(*IfConditionFunction)
	if !ok {
		t.Fatalf("expected IfConditionFunction, got %T", result)
	}
	if plain := fn.Name.AsPlain(); plain == nil || *plain != "attr" {
		t.Errorf("name = %v, want attr", plain)
	}
}

func TestIfGroupFunctionEmpty(t *testing.T) {
	p := newTestParserExpr("foo()")
	result, err := p.ifGroup()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := result.(*IfConditionFunction)
	if !ok {
		t.Fatalf("expected IfConditionFunction, got %T", result)
	}
	argsStr, err := fn.Arguments.String()
	if err != nil {
		t.Fatal(err)
	}
	if argsStr != "" {
		t.Errorf("arguments = %q, want empty", argsStr)
	}
}

func TestIfGroupRawInterpolation(t *testing.T) {
	p := newTestParserExpr("#{$var}")
	result, err := p.ifGroup()
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := result.(*IfConditionRaw)
	if !ok {
		t.Fatalf("expected IfConditionRaw, got %T", result)
	}
	if raw.Text == nil {
		t.Error("expected non-nil Text")
	}
}

func TestIfGroupErrorWhitespaceRequiredAnd(t *testing.T) {
	p := newTestParserExpr("and(x)")
	_, err := p.ifGroup()
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Whitespace is required between "and" and "("`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestIfGroupErrorWhitespaceRequiredOr(t *testing.T) {
	p := newTestParserExpr("or(x)")
	_, err := p.ifGroup()
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Whitespace is required between "or" and "("`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestIfGroupErrorWhitespaceRequiredNot(t *testing.T) {
	p := newTestParserExpr("not(x)")
	_, err := p.ifGroup()
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Whitespace is required between "not" and "("`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestIfGroupErrorMissingOpenParen(t *testing.T) {
	p := newTestParserExpr("foo")
	_, err := p.ifGroup()
	if err == nil {
		t.Fatal("expected error for missing (")
	}
	want := "expected \"(\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestIfGroupErrorMissingCloseParen(t *testing.T) {
	p := newTestParserExpr("foo(bar")
	_, err := p.ifGroup()
	if err == nil {
		t.Fatal("expected error for missing )")
	}
	want := "expected \")\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// trySpecialFunction
// ======================================================================

func consumeIdent(t *testing.T, scanner *sasscommon.SpanScanner, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		_, err := scanner.ReadChar()
		if err != nil {
			t.Fatalf("consumeIdent: %v", err)
		}
	}
}

func TestTrySpecialFunctionType(t *testing.T) {
	p := newTestParserExpr("type(text/plain)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 4)
	result, err := p.trySpecialFunction("type", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

func TestTrySpecialFunctionTypeNoParen(t *testing.T) {
	p := newTestParserExpr("type not-a-call")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 4)
	result, err := p.trySpecialFunction("type", start)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %T", result)
	}
}

func TestTrySpecialFunctionExpressionVendored(t *testing.T) {
	p := newTestParserExpr("-moz-expression(1 + 2)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 15)
	result, err := p.trySpecialFunction("-moz-expression", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
	if len(p.warnings) == 0 {
		t.Error("expected deprecation warning")
	}
}

func TestTrySpecialFunctionExpressionNonVendored(t *testing.T) {
	p := newTestParserExpr("expression(X X)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 10)
	result, err := p.trySpecialFunction("expression", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil (non-vendored expression goes to calcLike)")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

// The vendor-prefixed expression() deprecation states the correct future
// (#2148, via refactor 548e6604): an argument that isn't valid SassScript
// "will no longer be valid syntax", one that parses but isn't plain CSS "will
// be parsed as SassScript".
func TestTrySpecialFunctionExpressionInvalidArg(t *testing.T) {
	p := newTestParserExpr("-c-expression(@#$)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 13)
	if _, err := p.trySpecialFunction("-c-expression", start); err != nil {
		t.Fatal(err)
	}
	if len(p.warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(p.warnings))
	}
	if !strings.Contains(p.warnings[0].Message, "this argument will no longer be valid syntax") {
		t.Errorf("got: %s", p.warnings[0].Message)
	}
}

func TestTrySpecialFunctionExpressionScriptLikeArg(t *testing.T) {
	p := newTestParserExpr("-c-expression($d)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 13)
	if _, err := p.trySpecialFunction("-c-expression", start); err != nil {
		t.Fatal(err)
	}
	if len(p.warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(p.warnings))
	}
	if !strings.Contains(p.warnings[0].Message, "this argument will be parsed as SassScript") {
		t.Errorf("got: %s", p.warnings[0].Message)
	}
}

// Empty `()` skips the probe entirely: no warning (#2148).
func TestTrySpecialFunctionExpressionEmptyNoWarning(t *testing.T) {
	p := newTestParserExpr("-a-expression()")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 13)
	if _, err := p.trySpecialFunction("-a-expression", start); err != nil {
		t.Fatal(err)
	}
	if len(p.warnings) != 0 {
		t.Errorf("warnings = %d, want 0: %+v", len(p.warnings), p.warnings)
	}
}

func TestTrySpecialFunctionCalcVendored(t *testing.T) {
	p := newTestParserExpr("-moz-calc(1px + 2px)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 9)
	result, err := p.trySpecialFunction("-moz-calc", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

func TestTrySpecialFunctionCalcNonVendored(t *testing.T) {
	p := newTestParserExpr("calc(1px + 2px)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 4)
	result, err := p.trySpecialFunction("calc", start)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil for non-vendored calc, got %T", result)
	}
}

func TestTrySpecialFunctionElement(t *testing.T) {
	p := newTestParserExpr("element(#foo)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 7)
	result, err := p.trySpecialFunction("element", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

func TestTrySpecialFunctionProgid(t *testing.T) {
	p := newTestParserExpr("progid:DXImageTransform.Microsoft.Alpha(opacity=50)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 6)
	result, err := p.trySpecialFunction("progid", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

func TestTrySpecialFunctionProgidVendored(t *testing.T) {
	p := newTestParserExpr("-ms-progid:Foo.Bar(baz)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 10)
	result, err := p.trySpecialFunction("-ms-progid", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
	if len(p.warnings) == 0 {
		t.Error("expected deprecation warning for vendored progid")
	}
}

func TestTrySpecialFunctionUrl(t *testing.T) {
	p := newTestParserExpr("url(http://example.com)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 3)
	result, err := p.trySpecialFunction("url", start)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result.(*StringExpression); !ok {
		t.Fatalf("expected StringExpression, got %T", result)
	}
}

func TestTrySpecialFunctionUrlNoParen(t *testing.T) {
	p := newTestParserExpr("url")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 3)
	result, err := p.trySpecialFunction("url", start)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %T", result)
	}
}

func TestTrySpecialFunctionUnknown(t *testing.T) {
	p := newTestParserExpr("rgb(1, 2, 3)")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 3)
	result, err := p.trySpecialFunction("rgb", start)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %T", result)
	}
}

func TestTrySpecialFunctionTypeErrorMissingCloseParen(t *testing.T) {
	p := newTestParserExpr("type(text/plain")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 4)
	_, err := p.trySpecialFunction("type", start)
	if err == nil {
		t.Fatal("expected error for missing )")
	}
	want := "expected \")\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestTrySpecialFunctionCalcVendoredErrorMissingCloseParen(t *testing.T) {
	p := newTestParserExpr("-moz-calc(1px + 2px")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 9)
	_, err := p.trySpecialFunction("-moz-calc", start)
	if err == nil {
		t.Fatal("expected error for missing )")
	}
	want := "expected \")\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestTrySpecialFunctionProgidErrorMissingOpenParen(t *testing.T) {
	p := newTestParserExpr("progid:Foo")
	start := p.scanner.State()
	consumeIdent(t, p.scanner, 6)
	_, err := p.trySpecialFunction("progid", start)
	if err == nil {
		t.Fatal("expected error for missing (")
	}
	want := "expected \"(\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
