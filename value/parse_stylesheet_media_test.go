package value

import (
	"strings"
	"testing"
)

// ======================================================================
// mediaQueryList
// ======================================================================

func TestMediaQueryListSimpleType(t *testing.T) {
	p := newTestParserExpr("screen")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if plain := result.AsPlain(); plain == nil || *plain != "screen" {
		t.Errorf("AsPlain = %v, want screen", plain)
	}
}

func TestMediaQueryListComma(t *testing.T) {
	p := newTestParserExpr("screen, print")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	s, err := result.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "screen, print" {
		t.Errorf("String = %q, want %q", s, "screen, print")
	}
}

func TestMediaQueryListTypeAndFeature(t *testing.T) {
	p := newTestParserExpr("screen and (color)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if result.AsPlain() != nil {
		t.Error("expected non-plain interpolation (contains expressions)")
	}
}

func TestMediaQueryListNot(t *testing.T) {
	p := newTestParserExpr("not screen")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	s, err := result.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "not screen" {
		t.Errorf("String = %q, want %q", s, "not screen")
	}
}

func TestMediaQueryListOnly(t *testing.T) {
	p := newTestParserExpr("only screen and (color)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if result.AsPlain() != nil {
		t.Error("expected non-plain interpolation (contains expressions)")
	}
}

// ======================================================================
// mediaInParens (exercised via mediaQueryList)
// ======================================================================

func TestMediaInParensFeatureComparison(t *testing.T) {
	p := newTestParserExpr("(width > 100px)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if result.AsPlain() != nil {
		t.Error("expected non-plain (contains expression)")
	}
}

func TestMediaInParensColon(t *testing.T) {
	p := newTestParserExpr("(width: 100px)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if result.AsPlain() != nil {
		t.Error("expected non-plain (contains expression)")
	}
}

func TestMediaInParensAnd(t *testing.T) {
	p := newTestParserExpr("(width > 100px) and (height > 100px)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	s, err := result.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "and") {
		t.Errorf("expected 'and' in: %q", s)
	}
}

func TestMediaInParensOr(t *testing.T) {
	p := newTestParserExpr("(width > 100px) or (height > 100px)")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	s, err := result.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "or") {
		t.Errorf("expected 'or' in: %q", s)
	}
}

func TestMediaInParensNested(t *testing.T) {
	p := newTestParserExpr("((width > 100px))")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	if result.AsPlain() != nil {
		t.Error("expected non-plain (contains expression)")
	}
}

func TestMediaInParensNot(t *testing.T) {
	p := newTestParserExpr("(not (color))")
	result, err := p.mediaQueryList()
	if err != nil {
		t.Fatal(err)
	}
	s, err := result.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "not") {
		t.Errorf("expected 'not' in: %q", s)
	}
}

// ======================================================================
// Error cases
// ======================================================================

func TestMediaErrorMissingCloseParen(t *testing.T) {
	p := newTestParserExpr("(width > 100px")
	_, err := p.mediaQueryList()
	if err == nil {
		t.Fatal("expected error for missing )")
	}
	want := "expected \")\"."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestMediaErrorTypeAndNoValue(t *testing.T) {
	p := newTestParserExpr("screen and")
	_, err := p.mediaQueryList()
	if err == nil {
		t.Fatal("expected error after 'and'")
	}
	want := "Expected whitespace."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
