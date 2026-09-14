package value

import (
	"strings"
	"testing"
)

// =============================================================================
// Stylesheet selectorList tests
// =============================================================================

func TestStylesheetSelectorListSingle(t *testing.T) {
	p := newTestStylesheetParser("a")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(list.Components))
	}
	s, err := list.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a" {
		t.Errorf("String() = %q, want %q", s, "a")
	}
}

func TestStylesheetSelectorListMultiple(t *testing.T) {
	p := newTestStylesheetParser("a, b, c")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 3 {
		t.Errorf("len(Components) = %d, want 3", len(list.Components))
	}
	s, err := list.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a, b, c" {
		t.Errorf("String() = %q, want %q", s, "a, b, c")
	}
}

func TestStylesheetSelectorListEmptyComma(t *testing.T) {
	p := newTestStylesheetParser("a,,b")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Errorf("len(Components) = %d, want 2", len(list.Components))
	}
}

func TestStylesheetSelectorListTrailingComma(t *testing.T) {
	p := newTestStylesheetParser("a,")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(list.Components))
	}
}

// =============================================================================
// Stylesheet complexSelector tests
// =============================================================================

func TestStylesheetComplexSelectorSingleCompound(t *testing.T) {
	p := newTestStylesheetParser("div")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "div" {
		t.Errorf("String() = %q, want %q", s, "div")
	}
}

func TestStylesheetComplexSelectorChild(t *testing.T) {
	p := newTestStylesheetParser("a > b")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a > b" {
		t.Errorf("String() = %q, want %q", s, "a > b")
	}
}

func TestStylesheetComplexSelectorNextSibling(t *testing.T) {
	p := newTestStylesheetParser("a + b")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a + b" {
		t.Errorf("String() = %q, want %q", s, "a + b")
	}
}

func TestStylesheetComplexSelectorFollowingSibling(t *testing.T) {
	p := newTestStylesheetParser("a ~ b")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a ~ b" {
		t.Errorf("String() = %q, want %q", s, "a ~ b")
	}
}

func TestStylesheetComplexSelectorDescendant(t *testing.T) {
	p := newTestStylesheetParser("a b")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a b" {
		t.Errorf("String() = %q, want %q", s, "a b")
	}
}

func TestStylesheetComplexSelectorLeadingCombinator(t *testing.T) {
	p := newTestStylesheetParser("> a")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if complex.LeadingCombinator == nil {
		t.Fatal("expected leading combinator")
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	_ = complex
	_ = s
}

func TestStylesheetComplexSelectorTrailingCombinatorError(t *testing.T) {
	p := newTestStylesheetParser("a >")
	_, err := p.complexSelector(false, true, false)
	if err == nil {
		t.Fatal("expected error for trailing combinator")
	}
	want := `expected selector.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetComplexSelectorEmptyError(t *testing.T) {
	p := newTestStylesheetParser("")
	_, err := p.complexSelector(false, true, true)
	if err == nil {
		t.Fatal("expected error for empty selector")
	}
	want := `expected selector.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetComplexSelectorMultipleCombinators(t *testing.T) {
	p := newTestStylesheetParser("a > b + c ~ d")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := complex.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a > b + c ~ d" {
		t.Errorf("String() = %q, want %q", s, "a > b + c ~ d")
	}
}

func TestStylesheetComplexSelectorAmpersandMidCompoundError(t *testing.T) {
	p := newTestStylesheetParser("a&b")
	_, err := p.complexSelector(false, true, true)
	if err == nil {
		t.Fatal("expected error for & in middle of compound")
	}
	want := `"&" may only used at the beginning of a compound selector.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// =============================================================================
// Stylesheet compoundSelector tests
// =============================================================================

func TestStylesheetCompoundSelectorSingle(t *testing.T) {
	p := newTestStylesheetParser("div")
	comp, err := p.compoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	if comp == nil {
		t.Fatal("expected non-nil compound")
	}
}

func TestStylesheetCompoundSelectorMultipleSimples(t *testing.T) {
	p := newTestStylesheetParser("div.foo#bar")
	comp, err := p.compoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	s, err := comp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "div.foo#bar" {
		t.Errorf("String() = %q, want %q", s, "div.foo#bar")
	}
}

// =============================================================================
// Stylesheet isSimpleSelectorStart tests
// =============================================================================

func TestStylesheetIsSimpleSelectorStart(t *testing.T) {
	tests := []struct {
		ch   rune
		want bool
	}{
		{'*', true},
		{'[', true},
		{'.', true},
		{'#', true},
		{'%', true},
		{':', true},
		{'a', false},
		{'-', false},
	}
	for _, tt := range tests {
		p := newTestStylesheetParser("")
		got := p.isSimpleSelectorStart(int(tt.ch))
		if got != tt.want {
			t.Errorf("isSimpleSelectorStart(%q) = %v, want %v", tt.ch, got, tt.want)
		}
	}
}

// =============================================================================
// Stylesheet simpleSelector tests
// =============================================================================

func TestStylesheetSimpleSelectorClass(t *testing.T) {
	p := newTestStylesheetParser(".foo")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ".foo" {
		t.Errorf("String() = %q, want %q", str, ".foo")
	}
}

func TestStylesheetSimpleSelectorID(t *testing.T) {
	p := newTestStylesheetParser("#bar")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "#bar" {
		t.Errorf("String() = %q, want %q", str, "#bar")
	}
}

func TestStylesheetSimpleSelectorType(t *testing.T) {
	p := newTestStylesheetParser("div")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "div" {
		t.Errorf("String() = %q, want %q", str, "div")
	}
}

func TestStylesheetSimpleSelectorUniversal(t *testing.T) {
	p := newTestStylesheetParser("*")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "*" {
		t.Errorf("String() = %q, want %q", str, "*")
	}
}

func TestStylesheetSimpleSelectorParentAllowed(t *testing.T) {
	p := newTestStylesheetParser("&")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "&" {
		t.Errorf("String() = %q, want %q", str, "&")
	}
}

func TestStylesheetSimpleSelectorParentNotAllowedError(t *testing.T) {
	p := newTestStylesheetParser("&")
	_, err := p.simpleSelector(false)
	if err == nil {
		t.Fatal("expected error for parent selector when not allowed")
	}
	want := strings.Join([]string{
		`Error: Parent selectors aren't allowed here.`,
		`  ╷`,
		`1 │ &`,
		`  │ ^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetSimpleSelectorPlaceholder(t *testing.T) {
	p := newTestStylesheetParser("%x")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "%x" {
		t.Errorf("String() = %q, want %q", str, "%x")
	}
}

func TestStylesheetSimpleSelectorPseudoClass(t *testing.T) {
	p := newTestStylesheetParser(":hover")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":hover" {
		t.Errorf("String() = %q, want %q", str, ":hover")
	}
}

func TestStylesheetSimpleSelectorPseudoElement(t *testing.T) {
	p := newTestStylesheetParser("::before")
	s, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "::before" {
		t.Errorf("String() = %q, want %q", str, "::before")
	}
}

// =============================================================================
// Stylesheet attributeSelector tests
// =============================================================================

func TestStylesheetAttributeSelectorBare(t *testing.T) {
	p := newTestStylesheetParser("[href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[href]" {
		t.Errorf("String() = %q, want %q", str, "[href]")
	}
}

func TestStylesheetAttributeSelectorWithValue(t *testing.T) {
	p := newTestStylesheetParser("[href=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[href=val]" {
		t.Errorf("String() = %q, want %q", str, "[href=val]")
	}
}

func TestStylesheetAttributeSelectorWithOperator(t *testing.T) {
	p := newTestStylesheetParser("[data-value~=x]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[data-value~=x]" {
		t.Errorf("String() = %q, want %q", str, "[data-value~=x]")
	}
}

func TestStylesheetAttributeSelectorWithQuotedValue(t *testing.T) {
	p := newTestStylesheetParser(`[href="val"]`)
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != `[href="val"]` {
		t.Errorf("String() = %q, want %q", str, `[href="val"]`)
	}
}

func TestStylesheetAttributeSelectorWithModifier(t *testing.T) {
	p := newTestStylesheetParser("[href=val i]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[href=val i]" {
		t.Errorf("String() = %q, want %q", str, "[href=val i]")
	}
}

func TestStylesheetAttributeSelectorAllOperators(t *testing.T) {
	tests := []struct{ input, want string }{
		{"[a=b]", "[a=b]"},
		{"[a~=b]", "[a~=b]"},
		{"[a|=b]", "[a|=b]"},
		{"[a^=b]", "[a^=b]"},
		{"[a$=b]", "[a$=b]"},
		{"[a*=b]", "[a*=b]"},
	}
	for _, tt := range tests {
		p := newTestStylesheetParser(tt.input)
		sel, err := p.attributeSelector()
		if err != nil {
			t.Fatalf("%s: %v", tt.input, err)
		}
		str, _ := sel.String()
		if str != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.input, str, tt.want)
		}
	}
}

func TestStylesheetAttributeSelectorWildcardNamespace(t *testing.T) {
	p := newTestStylesheetParser("[*|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[*|href]" {
		t.Errorf("String() = %q, want %q", str, "[*|href]")
	}
}

func TestStylesheetAttributeSelectorExplicitNamespace(t *testing.T) {
	p := newTestStylesheetParser("[ns|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[ns|href]" {
		t.Errorf("String() = %q, want %q", str, "[ns|href]")
	}
}

func TestStylesheetAttributeSelectorNoNamespace(t *testing.T) {
	p := newTestStylesheetParser("[|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "[|href]" {
		t.Errorf("String() = %q, want %q", str, "[|href]")
	}
}

// =============================================================================
// Stylesheet classSelector tests
// =============================================================================

func TestStylesheetClassSelector(t *testing.T) {
	p := newTestStylesheetParser(".foo")
	sel, err := p.classSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ".foo" {
		t.Errorf("String() = %q, want %q", str, ".foo")
	}
}

// =============================================================================
// Stylesheet idSelector tests
// =============================================================================

func TestStylesheetIDSelector(t *testing.T) {
	p := newTestStylesheetParser("#bar")
	sel, err := p.idSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "#bar" {
		t.Errorf("String() = %q, want %q", str, "#bar")
	}
}

// =============================================================================
// Stylesheet placeholderSelector tests
// =============================================================================

func TestStylesheetPlaceholderSelector(t *testing.T) {
	p := newTestStylesheetParser("%foo")
	sel, err := p.placeholderSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "%foo" {
		t.Errorf("String() = %q, want %q", str, "%foo")
	}
}

// =============================================================================
// Stylesheet parentSelector tests
// =============================================================================

func TestStylesheetParentSelector(t *testing.T) {
	p := newTestStylesheetParser("&")
	sel, err := p.parentSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "&" {
		t.Errorf("String() = %q, want %q", str, "&")
	}
}

func TestStylesheetParentSelectorSuffix(t *testing.T) {
	p := newTestStylesheetParser("&-suffix")
	sel, err := p.parentSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "&-suffix" {
		t.Errorf("String() = %q, want %q", str, "&-suffix")
	}
}

// =============================================================================
// Stylesheet pseudoSelector tests
// =============================================================================

func TestStylesheetPseudoSelectorClass(t *testing.T) {
	p := newTestStylesheetParser(":hover")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":hover" {
		t.Errorf("String() = %q, want %q", str, ":hover")
	}
}

func TestStylesheetPseudoSelectorElement(t *testing.T) {
	p := newTestStylesheetParser("::before")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "::before" {
		t.Errorf("String() = %q, want %q", str, "::before")
	}
}

func TestStylesheetPseudoSelectorWithSelectorArg(t *testing.T) {
	p := newTestStylesheetParser(":not(.foo)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":not(.foo)" {
		t.Errorf("String() = %q, want %q", str, ":not(.foo)")
	}
}

func TestStylesheetPseudoSelectorWithDeclarationArg(t *testing.T) {
	p := newTestStylesheetParser(":lang(en)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":lang(en)" {
		t.Errorf("String() = %q, want %q", str, ":lang(en)")
	}
}

func TestStylesheetPseudoSelectorNthChild(t *testing.T) {
	p := newTestStylesheetParser(":nth-child(2n+1)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":nth-child(2n+1)" {
		t.Errorf("String() = %q, want %q", str, ":nth-child(2n+1)")
	}
}

func TestStylesheetPseudoSelectorNthChildWithSelector(t *testing.T) {
	p := newTestStylesheetParser(":nth-child(1 of .foo)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":nth-child(1 of .foo)" {
		t.Errorf("String() = %q, want %q", str, ":nth-child(1 of .foo)")
	}
}

// =============================================================================
// Stylesheet typeOrUniversalSelector tests
// =============================================================================

func TestStylesheetTypeOrUniversalWildcard(t *testing.T) {
	p := newTestStylesheetParser("*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "*" {
		t.Errorf("String() = %q, want %q", str, "*")
	}
}

func TestStylesheetTypeOrUniversalWildcardNamespace(t *testing.T) {
	p := newTestStylesheetParser("*|*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "*|*" {
		t.Errorf("String() = %q, want %q", str, "*|*")
	}
}

func TestStylesheetTypeOrUniversalWildcardNamespaceType(t *testing.T) {
	p := newTestStylesheetParser("*|div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "*|div" {
		t.Errorf("String() = %q, want %q", str, "*|div")
	}
}

func TestStylesheetTypeOrUniversalEmptyNamespaceWildcard(t *testing.T) {
	p := newTestStylesheetParser("|*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "|*" {
		t.Errorf("String() = %q, want %q", str, "|*")
	}
}

func TestStylesheetTypeOrUniversalEmptyNamespaceType(t *testing.T) {
	p := newTestStylesheetParser("|div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "|div" {
		t.Errorf("String() = %q, want %q", str, "|div")
	}
}

func TestStylesheetTypeOrUniversalNamespacedType(t *testing.T) {
	p := newTestStylesheetParser("ns|div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "ns|div" {
		t.Errorf("String() = %q, want %q", str, "ns|div")
	}
}

func TestStylesheetTypeOrUniversalNamespacedWildcard(t *testing.T) {
	p := newTestStylesheetParser("ns|*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "ns|*" {
		t.Errorf("String() = %q, want %q", str, "ns|*")
	}
}

func TestStylesheetTypeOrUniversalBareType(t *testing.T) {
	p := newTestStylesheetParser("div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "div" {
		t.Errorf("String() = %q, want %q", str, "div")
	}
}

// =============================================================================
// === ADDITIONAL TESTS: Error paths (exact message assertions)
// =============================================================================

// --- Format errors (multiline, box-drawing chars) ---

func TestStylesheetSimpleSelectorPlaceholderPlainCssError(t *testing.T) {
	p := newTestStylesheetParser("%x")
	p.plainCss = true
	_, err := p.simpleSelector(true)
	if err == nil {
		t.Fatal("expected error for placeholder in plain CSS")
	}
	want := strings.Join([]string{
		`Error: Placeholder selectors aren't allowed in plain CSS.`,
		`  ╷`,
		`1 │ %x`,
		`  │ ^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetSimpleSelectorPlaceholderPlainCssErrorWithPrefix(t *testing.T) {
	p := newTestStylesheetParser("a %x")
	p.plainCss = true
	_, err := p.complexSelector(false, true, true)
	if err == nil {
		t.Fatal("expected error for placeholder in plain CSS")
	}
	want := strings.Join([]string{
		`Error: Placeholder selectors aren't allowed in plain CSS.`,
		`  ╷`,
		`1 │ a %x`,
		`  │   ^^`,
		`  ╵`,
		`  - 1:3  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// --- Scan errors (single-line) ---

func TestStylesheetParentSelectorSuffixPlainCssError(t *testing.T) {
	p := newTestStylesheetParser("&-suffix")
	p.plainCss = true
	_, err := p.parentSelector()
	if err == nil {
		t.Fatal("expected error for parent suffix in plain CSS")
	}
	want := `Parent selectors can't have suffixes in plain CSS.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetAttributeSelectorInvalidOperatorError(t *testing.T) {
	p := newTestStylesheetParser("[href!val]")
	_, err := p.attributeSelector()
	if err == nil {
		t.Fatal("expected error for invalid attribute operator")
	}
	want := `Expected "]".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetComplexSelectorLeadingCombinatorDeniedError(t *testing.T) {
	p := newTestStylesheetParser("> a")
	_, err := p.complexSelector(false, false, true)
	if err == nil {
		t.Fatal("expected error when leading combinator denied")
	}
	want := `expected selector.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStylesheetComplexSelectorTrailingCombinatorPlainCssError(t *testing.T) {
	p := newTestStylesheetParser("a >")
	p.plainCss = true
	_, err := p.complexSelector(false, true, true)
	if err == nil {
		t.Fatal("expected error for trailing combinator in plain CSS")
	}
	want := `expected selector.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// =============================================================================
// === ADDITIONAL TESTS: complex_selector structural assertions
// =============================================================================

func TestStylesheetComplexSelectorLeadingCombinatorChild(t *testing.T) {
	p := newTestStylesheetParser("> a")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if complex.LeadingCombinator == nil {
		t.Fatal("expected leading combinator")
	}
	if complex.LeadingCombinator.Value != CombinatorChild {
		t.Errorf("expected Child combinator, got %v", complex.LeadingCombinator.Value)
	}
}

func TestStylesheetComplexSelectorLeadingCombinatorNextSibling(t *testing.T) {
	p := newTestStylesheetParser("+ a")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if complex.LeadingCombinator == nil {
		t.Fatal("expected leading combinator")
	}
	if complex.LeadingCombinator.Value != CombinatorNextSibling {
		t.Errorf("expected NextSibling combinator, got %v", complex.LeadingCombinator.Value)
	}
}

func TestStylesheetComplexSelectorLeadingCombinatorFollowingSibling(t *testing.T) {
	p := newTestStylesheetParser("~ a")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if complex.LeadingCombinator == nil {
		t.Fatal("expected leading combinator")
	}
	if complex.LeadingCombinator.Value != CombinatorFollowingSibling {
		t.Errorf("expected FollowingSibling combinator, got %v", complex.LeadingCombinator.Value)
	}
}

func TestStylesheetComplexSelectorCombinatorTypesInOrder(t *testing.T) {
	p := newTestStylesheetParser("a > b + c ~ d")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(complex.Components) != 4 {
		t.Fatalf("expected 4 components, got %d", len(complex.Components))
	}
	expected := []Combinator{CombinatorChild, CombinatorNextSibling, CombinatorFollowingSibling}
	for i := 0; i < 3; i++ {
		comp := complex.Components[i]
		if comp.Combinator == nil {
			t.Errorf("component %d: expected combinator %v, got nil", i, expected[i])
		} else if comp.Combinator.Value != expected[i] {
			t.Errorf("component %d: expected combinator %v, got %v", i, expected[i], comp.Combinator.Value)
		}
	}
}

func TestStylesheetComplexSelectorDescendantChain(t *testing.T) {
	p := newTestStylesheetParser("a b c")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(complex.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(complex.Components))
	}
	for i, comp := range complex.Components {
		if i < len(complex.Components)-1 && comp.Combinator != nil {
			t.Errorf("component %d: expected nil combinator (descendant), got %v", i, comp.Combinator)
		}
	}
}

func TestStylesheetComplexSelectorChildCombinatorComponent(t *testing.T) {
	p := newTestStylesheetParser("a > b")
	complex, err := p.complexSelector(false, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(complex.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(complex.Components))
	}
	if complex.Components[0].Combinator == nil {
		t.Fatal("expected combinator on first component")
	}
	if complex.Components[0].Combinator.Value != CombinatorChild {
		t.Errorf("expected Child combinator, got %v", complex.Components[0].Combinator.Value)
	}
}

// =============================================================================
// === ADDITIONAL TESTS: simple_selector dispatch
// =============================================================================

func TestStylesheetSimpleSelectorHashWithBraceGoesToTypeOrUniversal(t *testing.T) {
	p := newTestStylesheetParser("#{$x}")
	sel, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*InterpolatedIDSelector); ok {
		t.Error("expected type/universal selector, not ID selector")
	}
}

func TestStylesheetSimpleSelectorPlaceholderNotPlainCss(t *testing.T) {
	p := newTestStylesheetParser("%x")
	sel, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*InterpolatedPlaceholderSelector); !ok {
		t.Error("expected InterpolatedPlaceholderSelector")
	}
}

func TestStylesheetSimpleSelectorAttributeDispatch(t *testing.T) {
	p := newTestStylesheetParser("[href]")
	sel, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*InterpolatedAttributeSelector); !ok {
		t.Error("expected InterpolatedAttributeSelector")
	}
}

func TestStylesheetSimpleSelectorParent(t *testing.T) {
	p := newTestStylesheetParser("&")
	sel, err := p.simpleSelector(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*InterpolatedParentSelector); !ok {
		t.Error("expected InterpolatedParentSelector")
	}
}

func TestStylesheetSimpleSelectorParentPlainCssDisallowed(t *testing.T) {
	p := newTestStylesheetParser("&")
	p.plainCss = true
	_, err := p.simpleSelector(false)
	if err == nil {
		t.Fatal("expected error for parent in plain CSS with allowParent=false")
	}
	want := strings.Join([]string{
		`Error: Parent selectors aren't allowed here.`,
		`  ╷`,
		`1 │ &`,
		`  │ ^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// =============================================================================
// === ADDITIONAL TESTS: attribute_selector structural assertions
// =============================================================================

func TestStylesheetAttributeSelectorModifierPresent(t *testing.T) {
	p := newTestStylesheetParser("[href=val i]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Modifier == nil {
		t.Fatal("expected modifier to be present")
	}
	s, err := sel.Modifier.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "i" {
		t.Errorf("modifier = %q, want %q", s, "i")
	}
}

func TestStylesheetAttributeSelectorBareNoOpNoValue(t *testing.T) {
	p := newTestStylesheetParser("[href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op != nil {
		t.Error("expected Op to be nil for bare attribute")
	}
	if sel.Value != nil {
		t.Error("expected Value to be nil for bare attribute")
	}
}

func TestStylesheetAttributeSelectorQuotedValue(t *testing.T) {
	p := newTestStylesheetParser(`[href="val"]`)
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Value == nil {
		t.Fatal("expected Value to be present")
	}
	s, err := sel.Value.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != `"val"` {
		t.Errorf("value = %q, want %q", s, `"val"`)
	}
}

func TestStylesheetAttributeSelectorOperatorInclude(t *testing.T) {
	p := newTestStylesheetParser("[href~=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op == nil {
		t.Fatal("expected Op to be present")
	}
	if sel.Op.Value != AttributeOperatorInclude {
		t.Errorf("expected Include operator, got %v", sel.Op.Value)
	}
}

func TestStylesheetAttributeSelectorOperatorDash(t *testing.T) {
	p := newTestStylesheetParser("[href|=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op == nil {
		t.Fatal("expected Op to be present")
	}
	if sel.Op.Value != AttributeOperatorDash {
		t.Errorf("expected Dash operator, got %v", sel.Op.Value)
	}
}

func TestStylesheetAttributeSelectorOperatorPrefix(t *testing.T) {
	p := newTestStylesheetParser("[href^=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op == nil {
		t.Fatal("expected Op to be present")
	}
	if sel.Op.Value != AttributeOperatorPrefix {
		t.Errorf("expected Prefix operator, got %v", sel.Op.Value)
	}
}

func TestStylesheetAttributeSelectorOperatorSuffix(t *testing.T) {
	p := newTestStylesheetParser("[href$=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op == nil {
		t.Fatal("expected Op to be present")
	}
	if sel.Op.Value != AttributeOperatorSuffix {
		t.Errorf("expected Suffix operator, got %v", sel.Op.Value)
	}
}

func TestStylesheetAttributeSelectorOperatorSubstring(t *testing.T) {
	p := newTestStylesheetParser("[href*=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Op == nil {
		t.Fatal("expected Op to be present")
	}
	if sel.Op.Value != AttributeOperatorSubstring {
		t.Errorf("expected Substring operator, got %v", sel.Op.Value)
	}
}

// =============================================================================
// === ADDITIONAL TESTS: attribute_name logic
// =============================================================================

func TestStylesheetAttributeNameWildcardNs(t *testing.T) {
	p := newTestStylesheetParser("[*|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	name := sel.Name
	if name.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := name.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "*" {
		t.Errorf("namespace = %q, want %q", ns, "*")
	}
	n, err := name.Name.String()
	if err != nil {
		t.Fatal(err)
	}
	if n != "href" {
		t.Errorf("name = %q, want %q", n, "href")
	}
}

func TestStylesheetAttributeNameEmptyNs(t *testing.T) {
	p := newTestStylesheetParser("[|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	name := sel.Name
	if name.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := name.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "" {
		t.Errorf("namespace = %q, want empty", ns)
	}
}

func TestStylesheetAttributeNameExplicitNs(t *testing.T) {
	p := newTestStylesheetParser("[ns|href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	name := sel.Name
	if name.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := name.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "ns" {
		t.Errorf("namespace = %q, want %q", ns, "ns")
	}
	n, err := name.Name.String()
	if err != nil {
		t.Fatal(err)
	}
	if n != "href" {
		t.Errorf("name = %q, want %q", n, "href")
	}
}

func TestStylesheetAttributeNameNoNs(t *testing.T) {
	p := newTestStylesheetParser("[href]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Name.Namespace != nil {
		t.Error("expected no namespace")
	}
}

func TestStylesheetAttributeNamePipeEqualsDisambig(t *testing.T) {
	p := newTestStylesheetParser("[href|=val]")
	sel, err := p.attributeSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel.Name.Namespace != nil {
		t.Error("expected no namespace for [href|=val] — |= is operator, not namespace")
	}
	n, err := sel.Name.Name.String()
	if err != nil {
		t.Fatal(err)
	}
	if n != "href" {
		t.Errorf("name = %q, want %q", n, "href")
	}
}

// =============================================================================
// === ADDITIONAL TESTS: pseudo_selector expanded
// =============================================================================

func TestStylesheetPseudoSelectorNthLastChild(t *testing.T) {
	p := newTestStylesheetParser(":nth-last-child(2n)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":nth-last-child(2n)" {
		t.Errorf("String() = %q, want %q", str, ":nth-last-child(2n)")
	}
}

func TestStylesheetPseudoSelectorIsList(t *testing.T) {
	p := newTestStylesheetParser(":is(.a, .b)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":is(.a, .b)" {
		t.Errorf("String() = %q, want %q", str, ":is(.a, .b)")
	}
}

func TestStylesheetPseudoSelectorWhere(t *testing.T) {
	p := newTestStylesheetParser(":where(.x)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":where(.x)" {
		t.Errorf("String() = %q, want %q", str, ":where(.x)")
	}
}

func TestStylesheetPseudoSelectorHasCombinator(t *testing.T) {
	p := newTestStylesheetParser(":has(div > a)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":has(div > a)" {
		t.Errorf("String() = %q, want %q", str, ":has(div > a)")
	}
}

func TestStylesheetPseudoSelectorSlotted(t *testing.T) {
	p := newTestStylesheetParser("::slotted(.x)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "::slotted(.x)" {
		t.Errorf("String() = %q, want %q", str, "::slotted(.x)")
	}
}

func TestStylesheetPseudoSelectorEmptyArg(t *testing.T) {
	p := newTestStylesheetParser(":lang()")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":lang()" {
		t.Errorf("String() = %q, want %q", str, ":lang()")
	}
}

func TestStylesheetPseudoSelectorVendorPrefixed(t *testing.T) {
	p := newTestStylesheetParser(":-moz-any(.x)")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != ":-moz-any(.x)" {
		t.Errorf("String() = %q, want %q", str, ":-moz-any(.x)")
	}
}

func TestStylesheetPseudoSelectorStructuralClassVsElement(t *testing.T) {
	p := newTestStylesheetParser(":hover")
	sel, err := p.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	if !sel.IsSyntacticClass {
		t.Error(":hover should be isSyntacticClass=true")
	}
	if sel.IsSyntacticElement() {
		t.Error(":hover should be isSyntacticElement=false")
	}

	p2 := newTestStylesheetParser("::before")
	sel2, err := p2.pseudoSelector()
	if err != nil {
		t.Fatal(err)
	}
	if sel2.IsSyntacticClass {
		t.Error("::before should be isSyntacticClass=false")
	}
	if !sel2.IsSyntacticElement() {
		t.Error("::before should be isSyntacticElement=true")
	}
}

// =============================================================================
// === ADDITIONAL TESTS: type_or_universal structural assertions
// =============================================================================

func TestStylesheetTypeOrUniversalWildcardNoNs(t *testing.T) {
	p := newTestStylesheetParser("*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	u, ok := sel.(*InterpolatedUniversalSelector)
	if !ok {
		t.Fatal("expected InterpolatedUniversalSelector")
	}
	if u.Namespace != nil {
		t.Error("expected no namespace")
	}
}

func TestStylesheetTypeOrUniversalNamespacedTypeAsType(t *testing.T) {
	p := newTestStylesheetParser("ns|div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*InterpolatedTypeSelector)
	if !ok {
		t.Fatal("expected InterpolatedTypeSelector")
	}
	if ty.Name.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := ty.Name.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "ns" {
		t.Errorf("namespace = %q, want %q", ns, "ns")
	}
}

func TestStylesheetTypeOrUniversalWildcardNamespaceStar(t *testing.T) {
	p := newTestStylesheetParser("*|*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	u, ok := sel.(*InterpolatedUniversalSelector)
	if !ok {
		t.Fatal("expected InterpolatedUniversalSelector")
	}
	if u.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := u.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "*" {
		t.Errorf("namespace = %q, want %q", ns, "*")
	}
}

func TestStylesheetTypeOrUniversalEmptyNsStar(t *testing.T) {
	p := newTestStylesheetParser("|*")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	u, ok := sel.(*InterpolatedUniversalSelector)
	if !ok {
		t.Fatal("expected InterpolatedUniversalSelector")
	}
	if u.Namespace == nil {
		t.Fatal("expected namespace")
	}
	ns, err := u.Namespace.String()
	if err != nil {
		t.Fatal(err)
	}
	if ns != "" {
		t.Errorf("namespace = %q, want empty", ns)
	}
}

func TestStylesheetTypeOrUniversalBareTypeAsType(t *testing.T) {
	p := newTestStylesheetParser("div")
	sel, err := p.typeOrUniversalSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*InterpolatedTypeSelector)
	if !ok {
		t.Fatal("expected InterpolatedTypeSelector")
	}
	if ty.Name.Namespace != nil {
		t.Error("expected no namespace for bare type")
	}
	n, err := ty.Name.Name.String()
	if err != nil {
		t.Fatal(err)
	}
	if n != "div" {
		t.Errorf("name = %q, want %q", n, "div")
	}
}

// =============================================================================
// === ADDITIONAL TESTS: selector_list
// =============================================================================

func TestStylesheetSelectorListLineBreak(t *testing.T) {
	p := newTestStylesheetParser("a,\nb")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(list.Components))
	}
}

func TestStylesheetSelectorListComplexWithCombinators(t *testing.T) {
	p := newTestStylesheetParser("a > b, c + d")
	list, err := p.selectorList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(list.Components))
	}
	str, err := list.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "a > b, c + d" {
		t.Errorf("String() = %q, want %q", str, "a > b, c + d")
	}
}

// =============================================================================
// === ADDITIONAL TESTS: compound_selector
// =============================================================================

func TestStylesheetCompoundSelectorAllTypes(t *testing.T) {
	p := newTestStylesheetParser("div.foo#bar:hover")
	comp, err := p.compoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := comp.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "div.foo#bar:hover" {
		t.Errorf("String() = %q, want %q", str, "div.foo#bar:hover")
	}
}

func TestStylesheetCompoundSelectorNamespacedType(t *testing.T) {
	p := newTestStylesheetParser("*|div.foo")
	comp, err := p.compoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	str, err := comp.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "*|div.foo" {
		t.Errorf("String() = %q, want %q", str, "*|div.foo")
	}
}
