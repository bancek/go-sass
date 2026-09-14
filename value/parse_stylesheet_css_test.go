package value

import (
	"strings"
	"testing"
)

func newTestCssParser(text string) *CssParser {
	return NewCssParser([]byte(text), nil, false, nil)
}

// ======================================================================
// Group A: cssAtRule — forbidden Sass rules
// ======================================================================

func TestParseCssAtRuleForbidden(t *testing.T) {
	p := newTestCssParser("@mixin foo { }")
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for @mixin in plain CSS")
	}
	want := "This at-rule isn't allowed in plain CSS."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestParseCssAtRuleMediaAllowed(t *testing.T) {
	p := newTestCssParser("@media screen { }")
	_, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
}

// ======================================================================
// Group B: cssParentheses — single expression only
// ======================================================================

func TestParseCssParenthesesSimple(t *testing.T) {
	p := newTestCssParser("(1 + 2)")
	expr, err := p.parentheses()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ParenthesizedExpression); !ok {
		t.Errorf("expected ParenthesizedExpression, got %T", expr)
	}
}

// ======================================================================
// Group C: cssSilentComment error
// ======================================================================

func TestParseCssSilentComment(t *testing.T) {
	p := newTestCssParser("// comment\n")
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for silent comment in plain CSS")
	}
	want := "Silent comments aren't allowed in plain CSS."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ======================================================================
// Group D: cssNamespacedExpression error
// ======================================================================

func TestParseCssNamespaceError(t *testing.T) {
	p := newTestCssParser("ns.$var")
	_, err := p.namespacedExpression("ns", p.scanner.State())
	if err == nil {
		t.Fatal("expected error for namespace in plain CSS")
	}
}
