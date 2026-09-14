package value

import (
	"strings"
	"testing"
)

// ======================================================================
// importSupportsQuery
// ======================================================================

func TestImportSupportsQueryNot(t *testing.T) {
	p := newTestParserExpr("not (color)")
	result, err := p.importSupportsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsNegation); !ok {
		t.Fatalf("expected SupportsNegation, got %T", result)
	}
}

func TestImportSupportsQueryParensDecl(t *testing.T) {
	// (display: flex) is a declaration inside parens
	p := newTestParserExpr("(display: flex)")
	result, err := p.importSupportsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsDeclaration); !ok {
		t.Fatalf("expected SupportsDeclaration, got %T", result)
	}
}

func TestImportSupportsQueryFunction(t *testing.T) {
	p := newTestParserExpr("selector(foo)")
	result, err := p.importSupportsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsFunction); !ok {
		t.Fatalf("expected SupportsFunction, got %T", result)
	}
}

func TestImportSupportsQueryDeclaration(t *testing.T) {
	p := newTestParserExpr("display: flex")
	result, err := p.importSupportsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsDeclaration); !ok {
		t.Fatalf("expected SupportsDeclaration, got %T", result)
	}
}

// ======================================================================
// supportsCondition
// ======================================================================

func TestSupportsConditionNot(t *testing.T) {
	p := newTestParserExpr("not (color)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsNegation); !ok {
		t.Fatalf("expected SupportsNegation, got %T", result)
	}
}

func TestSupportsConditionAnd(t *testing.T) {
	p := newTestParserExpr("(color) and (width > 0)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := result.(*SupportsOperation)
	if !ok {
		t.Fatalf("expected SupportsOperation, got %T", result)
	}
	if op.Operator != BooleanOperatorAnd {
		t.Errorf("operator = %v, want And", op.Operator)
	}
}

func TestSupportsConditionOr(t *testing.T) {
	p := newTestParserExpr("(color) or (monochrome)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := result.(*SupportsOperation)
	if !ok {
		t.Fatalf("expected SupportsOperation, got %T", result)
	}
	if op.Operator != BooleanOperatorOr {
		t.Errorf("operator = %v, want Or", op.Operator)
	}
}

func TestSupportsConditionMultiAnd(t *testing.T) {
	p := newTestParserExpr("(a) and (b) and (c)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsOperation); !ok {
		t.Fatalf("expected SupportsOperation, got %T", result)
	}
}

// ======================================================================
// supportsCondition via parens path
// ======================================================================

func TestSupportsFunction(t *testing.T) {
	p := newTestParserExpr("func(args)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := result.(*SupportsFunction)
	if !ok {
		t.Fatalf("expected SupportsFunction, got %T", result)
	}
	if plain := fn.Name.AsPlain(); plain == nil || *plain != "func" {
		t.Errorf("name = %v, want func", plain)
	}
}

func TestSupportsInterpolation(t *testing.T) {
	// #{$var} in parens → SupportsInterpolation
	p := newTestParserExpr("#{$var}")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsInterpolation); !ok {
		t.Fatalf("expected SupportsInterpolation, got %T", result)
	}
}

func TestSupportsParensNotInside(t *testing.T) {
	p := newTestParserExpr("(not (color))")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsNegation); !ok {
		t.Fatalf("expected SupportsNegation, got %T", result)
	}
}

func TestSupportsParensNested(t *testing.T) {
	// ((display: flex)) → nested parens around a declaration
	p := newTestParserExpr("((display: flex))")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsDeclaration); !ok {
		t.Fatalf("expected SupportsDeclaration, got %T", result)
	}
}

func TestSupportsDeclaration(t *testing.T) {
	p := newTestParserExpr("(display: flex)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsDeclaration); !ok {
		t.Fatalf("expected SupportsDeclaration, got %T", result)
	}
}

func TestSupportsDeclarationCustomProperty(t *testing.T) {
	p := newTestParserExpr("(--custom: value)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsDeclaration); !ok {
		t.Fatalf("expected SupportsDeclaration, got %T", result)
	}
}

func TestSupportsFallbackAnything(t *testing.T) {
	// (--foo) triggers unified fallback → SupportsAnything
	p := newTestParserExpr("(--foo)")
	result, err := p.supportsCondition(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*SupportsAnything); !ok {
		t.Fatalf("expected SupportsAnything, got %T", result)
	}
}

// ======================================================================
// Error cases
// ======================================================================

func TestSupportsErrorNotAsIdentifier(t *testing.T) {
	p := newTestParserExpr("not(args)")
	_, err := p.supportsConditionInParens()
	if err == nil {
		t.Fatal("expected error for 'not' as identifier")
	}
	want := strings.Join([]string{
		`Error: "not" is not a valid identifier here.`,
		`  ╷`,
		`1 │ not(args)`,
		`  │ ^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSupportsErrorMissingCloseParen(t *testing.T) {
	p := newTestParserExpr("(color")
	_, err := p.supportsCondition(false)
	if err == nil {
		t.Fatal("expected error for missing )")
	}
	want := "expected \")\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSupportsErrorMissingOpenParen(t *testing.T) {
	p := newTestParserExpr("}")
	_, err := p.supportsCondition(false)
	if err == nil {
		t.Fatal("expected error for missing (")
	}
	want := "expected \"(\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSupportsErrorExpectedCondition(t *testing.T) {
	// Multi-content identifier without parens
	p := newTestParserExpr("a b")
	_, err := p.supportsCondition(false)
	if err == nil {
		t.Fatal("expected error for bad condition")
	}
	want := strings.Join([]string{
		"Error: Expected @supports condition.",
		"  ╷",
		"1 │ a b",
		"  │ ^",
		"  ╵",
		"  - 1:1  root stylesheet",
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
