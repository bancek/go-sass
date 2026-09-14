package value

import (
	"testing"

	"github.com/bancek/go-sass/deprecation"
)

func newTestSelectorParser(text string) *SelectorParser {
	return NewSelectorParser(text, nil, nil, nil)
}

func newTestSelectorParserPlainCss(text string) *SelectorParser {
	plainCss := true
	allowParent := true
	return NewSelectorParser(text, nil, nil, &SelectorParserOptions{
		PlainCss:    &plainCss,
		AllowParent: &allowParent,
	})
}

func newTestSelectorParserNoParent(text string) *SelectorParser {
	allowParent := false
	return NewSelectorParser(text, nil, nil, &SelectorParserOptions{
		AllowParent: &allowParent,
	})
}

// ============================================================================
// Constructor
// ============================================================================

func TestSelectorParserConstructorDefaults(t *testing.T) {
	p := newTestSelectorParser("")
	if !p.allowParent {
		t.Error("expected allowParent=true by default")
	}
	if p.plainCss {
		t.Error("expected plainCss=false by default")
	}
}

func TestSelectorParserConstructorAllowParentFalse(t *testing.T) {
	p := newTestSelectorParserNoParent("")
	if p.allowParent {
		t.Error("expected allowParent=false")
	}
}

func TestSelectorParserConstructorPlainCssTrue(t *testing.T) {
	p := newTestSelectorParserPlainCss("")
	if !p.plainCss {
		t.Error("expected plainCss=true")
	}
}

// ============================================================================
// Parse (public API)
// ============================================================================

func TestParseValidSimple(t *testing.T) {
	p := newTestSelectorParser(".foo")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(list.Components))
	}
}

func TestParseValidMulti(t *testing.T) {
	p := newTestSelectorParser(".foo, .bar")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(list.Components))
	}
}

func TestParseErrorEmpty(t *testing.T) {
	p := newTestSelectorParser("")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseErrorTrailingGarbage(t *testing.T) {
	// ".foo bar" is a valid complex selector with descendant combinator
	// Use something clearly invalid as trailing garbage
	p := newTestSelectorParser(".foo bar {")
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for trailing garbage after valid selector")
	}
}

// ============================================================================
// ParseComplexSelector
// ============================================================================

func TestParseComplexSelectorValid(t *testing.T) {
	p := newTestSelectorParser(".foo")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if cs == nil {
		t.Fatal("expected non-nil ComplexSelector")
	}
}

func TestParseComplexSelectorErrorEmpty(t *testing.T) {
	p := newTestSelectorParser("")
	_, err := p.ParseComplexSelector()
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseComplexSelectorErrorTrailing(t *testing.T) {
	// ".foo bar" is a valid complex selector. Use { as trailing garbage.
	p := newTestSelectorParser(".foo {")
	_, err := p.ParseComplexSelector()
	if err == nil {
		t.Error("expected error for trailing garbage")
	}
}

// ============================================================================
// ParseCompoundSelector
// ============================================================================

func TestParseCompoundSelectorValid(t *testing.T) {
	p := newTestSelectorParser(".foo")
	cs, err := p.ParseCompoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	if cs == nil {
		t.Fatal("expected non-nil CompoundSelector")
	}
}

func TestParseCompoundSelectorErrorEmpty(t *testing.T) {
	p := newTestSelectorParser("")
	_, err := p.ParseCompoundSelector()
	if err == nil {
		t.Error("expected error for empty input")
	}
}

// ============================================================================
// ParseSimpleSelector
// ============================================================================

func TestParseSimpleSelectorClass(t *testing.T) {
	p := newTestSelectorParser(".foo")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	if class, ok := sel.(*ClassSelector); !ok {
		t.Errorf("expected ClassSelector, got %T", sel)
	} else {
		if class.Name != "foo" {
			t.Errorf("name = %q, want 'foo'", class.Name)
		}
	}
}

func TestParseSimpleSelectorErrorTrailing(t *testing.T) {
	p := newTestSelectorParser(".foo bar")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected 'unexpected token' error for trailing content")
	}
}

// ============================================================================
// _classSelector
// ============================================================================

func TestClassSelectorValid(t *testing.T) {
	p := newTestSelectorParser(".foo")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	class, ok := sel.(*ClassSelector)
	if !ok {
		t.Fatalf("expected ClassSelector, got %T", sel)
	}
	if class.Name != "foo" {
		t.Errorf("name = %q, want 'foo'", class.Name)
	}
}

func TestClassSelectorErrorInvalidIdentifier(t *testing.T) {
	p := newTestSelectorParser(".1foo")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for .1foo (digit after dot)")
	}
}

// ============================================================================
// _idSelector
// ============================================================================

func TestIDSelectorValid(t *testing.T) {
	p := newTestSelectorParser("#foo")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	id, ok := sel.(*IDSelector)
	if !ok {
		t.Fatalf("expected IDSelector, got %T", sel)
	}
	if id.Name != "foo" {
		t.Errorf("name = %q, want 'foo'", id.Name)
	}
}

func TestIDSelectorErrorMissingHash(t *testing.T) {
	// abc is a valid type selector, not an error case for #
	// Error case tested via _simpleSelector dispatch indirectly.
	p := newTestSelectorParser("abc")
	_, err := p.ParseSimpleSelector()
	if err != nil {
		t.Errorf("unexpected error for type selector 'abc': %v", err)
	}
}

// ============================================================================
// _placeholderSelector
// ============================================================================

func TestPlaceholderSelectorValid(t *testing.T) {
	p := newTestSelectorParser("%foo")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*PlaceholderSelector); !ok {
		t.Errorf("expected PlaceholderSelector, got %T", sel)
	}
}

func TestPlaceholderSelectorPlainCssError(t *testing.T) {
	p := newTestSelectorParserPlainCss("%foo")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for placeholder in plain CSS")
	}
}

// ============================================================================
// _parentSelector
// ============================================================================

func TestParentSelectorBare(t *testing.T) {
	p := newTestSelectorParser("&")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := sel.(*ParentSelector)
	if !ok {
		t.Fatalf("expected ParentSelector, got %T", sel)
	}
	if ps.Suffix != nil {
		t.Errorf("expected nil suffix, got %v", *ps.Suffix)
	}
}

func TestParentSelectorParserWithSuffix(t *testing.T) {
	p := newTestSelectorParser("&-suffix")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := sel.(*ParentSelector)
	if !ok {
		t.Fatalf("expected ParentSelector, got %T", sel)
	}
	if ps.Suffix == nil || *ps.Suffix != "-suffix" {
		t.Errorf("suffix = %v, want '-suffix'", ps.Suffix)
	}
}

func TestParentSelectorSuffixPlainCssError(t *testing.T) {
	p := newTestSelectorParserPlainCss("&-suffix")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for &-suffix in plain CSS")
	}
}

func TestParentSelectorNotAllowed(t *testing.T) {
	p := newTestSelectorParserNoParent("&")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for & when allowParent=false")
	}
}

// ============================================================================
// _attributeSelector
// ============================================================================

func TestAttributeSelectorBare(t *testing.T) {
	p := newTestSelectorParser(`[foo]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr, ok := sel.(*AttributeSelector)
	if !ok {
		t.Fatalf("expected AttributeSelector, got %T", sel)
	}
	if attr.Name.Name != "foo" {
		t.Errorf("name = %q, want 'foo'", attr.Name.Name)
	}
	if attr.Op != nil {
		t.Error("expected nil op for bare attribute")
	}
}

func TestAttributeSelectorEqualString(t *testing.T) {
	p := newTestSelectorParser(`[foo="bar"]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr, ok := sel.(*AttributeSelector)
	if !ok {
		t.Fatalf("expected AttributeSelector, got %T", sel)
	}
	if attr.Name.Name != "foo" {
		t.Errorf("name = %q, want 'foo'", attr.Name.Name)
	}
	if attr.Op == nil || *attr.Op != AttributeOperatorEqual {
		t.Error("expected Equal operator")
	}
	if attr.Value == nil || *attr.Value != "bar" {
		t.Errorf("value = %v, want 'bar'", attr.Value)
	}
}

func TestAttributeSelectorEqualIdent(t *testing.T) {
	p := newTestSelectorParser(`[foo=bar]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr, ok := sel.(*AttributeSelector)
	if !ok {
		t.Fatalf("expected AttributeSelector, got %T", sel)
	}
	if attr.Op == nil || *attr.Op != AttributeOperatorEqual {
		t.Error("expected Equal operator")
	}
	if attr.Value == nil || *attr.Value != "bar" {
		t.Errorf("value = %v, want 'bar'", attr.Value)
	}
}

func TestAttributeSelectorOperators(t *testing.T) {
	tests := []struct {
		input string
		want  AttributeOperator
	}{
		{`[foo~=bar]`, AttributeOperatorInclude},
		{`[foo|=bar]`, AttributeOperatorDash},
		{`[foo^=bar]`, AttributeOperatorPrefix},
		{`[foo$=bar]`, AttributeOperatorSuffix},
		{`[foo*=bar]`, AttributeOperatorSubstring},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := newTestSelectorParser(tt.input)
			sel, err := p.ParseSimpleSelector()
			if err != nil {
				t.Fatal(err)
			}
			attr, ok := sel.(*AttributeSelector)
			if !ok {
				t.Fatalf("expected AttributeSelector")
			}
			if attr.Op == nil || *attr.Op != tt.want {
				t.Errorf("op = %v, want %v", attr.Op, tt.want)
			}
		})
	}
}

func TestAttributeSelectorWithModifier(t *testing.T) {
	p := newTestSelectorParser(`[foo=bar i]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr, ok := sel.(*AttributeSelector)
	if !ok {
		t.Fatalf("expected AttributeSelector, got %T", sel)
	}
	if attr.Modifier == nil || *attr.Modifier != "i" {
		t.Errorf("modifier = %v, want 'i'", attr.Modifier)
	}
}

func TestAttributeSelectorErrorMissingClose(t *testing.T) {
	p := newTestSelectorParser(`[foo`)
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for missing ']'")
	}
}

// ============================================================================
// _attributeName
// ============================================================================

func checkAttributeName(t *testing.T, input string, name string, namespace *string) {
	t.Helper()
	p := newTestSelectorParser(input)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector for %q, got %T", input, sel)
	}
	if ty.Name.Name != name {
		t.Errorf("name = %q, want %q", ty.Name.Name, name)
	}
	if (ty.Name.Namespace == nil) != (namespace == nil) {
		t.Errorf("namespace presence mismatch: got %v, want %v", ty.Name.Namespace, namespace)
	}
	if namespace != nil && ty.Name.Namespace != nil && *ty.Name.Namespace != *namespace {
		t.Errorf("namespace = %q, want %q", *ty.Name.Namespace, *namespace)
	}
}

func TestAttributeNamePlain(t *testing.T) {
	checkAttributeName(t, "div", "div", nil)
}

func TestAttributeNameWildcardNamespace(t *testing.T) {
	wc := "*"
	checkAttributeName(t, "*|div", "div", &wc)
}

func TestAttributeNameEmptyNamespace(t *testing.T) {
	empty := ""
	checkAttributeName(t, "|div", "div", &empty)
}

func TestAttributeNameExplicitNamespace(t *testing.T) {
	ns := "ns"
	checkAttributeName(t, "ns|div", "div", &ns)
}

func TestAttributeNamePipeEqualsDisambiguation(t *testing.T) {
	// Pipe followed by = is NOT a namespace separator
	p := newTestSelectorParser(`[ns|=bar]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr, ok := sel.(*AttributeSelector)
	if !ok {
		t.Fatalf("expected AttributeSelector")
	}
	// ns|= is the dash operator
	if attr.Op == nil || *attr.Op != AttributeOperatorDash {
		t.Errorf("expected Dash operator for |=, got %v", attr.Op)
	}
}

// ============================================================================
// _attributeOperator
// ============================================================================

func TestAttributeOperatorEqual(t *testing.T) {
	p := newTestSelectorParser(`[foo=bar]`)
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	attr := sel.(*AttributeSelector)
	if attr.Op == nil || *attr.Op != AttributeOperatorEqual {
		t.Errorf("expected Equal, got %v", attr.Op)
	}
}

func TestAttributeOperatorInvalid(t *testing.T) {
	p := newTestSelectorParser(`[foo!bar]`)
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

// ============================================================================
// _typeOrUniversalSelector
// ============================================================================

func TestUniversalSelectorStar(t *testing.T) {
	p := newTestSelectorParser("*")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sel.(*UniversalSelector); !ok {
		t.Errorf("expected UniversalSelector, got %T", sel)
	}
}

func TestUniversalSelectorWildcardNamespace(t *testing.T) {
	p := newTestSelectorParser("*|*")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	us, ok := sel.(*UniversalSelector)
	if !ok {
		t.Fatalf("expected UniversalSelector, got %T", sel)
	}
	if us.Namespace == nil || *us.Namespace != "*" {
		t.Errorf("namespace = %v, want '*'", us.Namespace)
	}
}

func TestTypeSelectorWithWildcardNamespace(t *testing.T) {
	p := newTestSelectorParser("*|div")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", sel)
	}
	if ty.Name.Namespace == nil || *ty.Name.Namespace != "*" {
		t.Errorf("namespace = %v, want '*'", ty.Name.Namespace)
	}
}

func TestUniversalSelectorEmptyNamespace(t *testing.T) {
	p := newTestSelectorParser("|*")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	us, ok := sel.(*UniversalSelector)
	if !ok {
		t.Fatalf("expected UniversalSelector, got %T", sel)
	}
	if us.Namespace == nil || *us.Namespace != "" {
		t.Errorf("namespace = %v, want ''", us.Namespace)
	}
}

func TestTypeSelectorEmptyNamespace(t *testing.T) {
	p := newTestSelectorParser("|div")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", sel)
	}
	if ty.Name.Namespace == nil || *ty.Name.Namespace != "" {
		t.Errorf("namespace = %v, want ''", ty.Name.Namespace)
	}
}

func TestTypeSelectorPlain(t *testing.T) {
	p := newTestSelectorParser("div")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ty, ok := sel.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", sel)
	}
	if ty.Name.Name != "div" {
		t.Errorf("name = %q, want 'div'", ty.Name.Name)
	}
	if ty.Name.Namespace != nil {
		t.Error("expected nil namespace")
	}
}

// ============================================================================
// _isSimpleSelectorStart
// ============================================================================

func TestIsSimpleSelectorStartStar(t *testing.T) {
	p := newTestSelectorParser("*")
	if !p._isSimpleSelectorStart('*') {
		t.Error("* should be a simple selector start")
	}
}

func TestIsSimpleSelectorStartBrackets(t *testing.T) {
	p := newTestSelectorParser("[")
	for _, ch := range []int{'[', '.', '#', '%', ':'} {
		if !p._isSimpleSelectorStart(ch) {
			t.Errorf("%c should be a simple selector start", ch)
		}
	}
}

func TestIsSimpleSelectorStartAmpersandPlainCss(t *testing.T) {
	p := newTestSelectorParserPlainCss("&")
	if !p._isSimpleSelectorStart('&') {
		t.Error("& should be a simple selector start in plainCss mode")
	}
}

func TestIsSimpleSelectorStartAmpersandNonPlainCss(t *testing.T) {
	p := newTestSelectorParser("&")
	if p._isSimpleSelectorStart('&') {
		t.Error("& should NOT be a simple selector start in non-plainCss mode")
	}
}

func TestIsSimpleSelectorStartNonMatch(t *testing.T) {
	p := newTestSelectorParser("a")
	if p._isSimpleSelectorStart('a') {
		t.Error("letter should not be a simple selector start")
	}
}

// ============================================================================
// _aNPlusB
// ============================================================================

func TestAnPlusBEven(t *testing.T) {
	// Test via :nth-child(even) pseudo selector.
	p := newTestSelectorParser(":nth-child(even)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "even" {
		t.Errorf("argument = %v, want 'even'", ps.Argument)
	}
}

func TestAnPlusBOdd(t *testing.T) {
	p := newTestSelectorParser(":nth-child(odd)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "odd" {
		t.Errorf("argument = %v, want 'odd'", ps.Argument)
	}
}

func TestAnPlusBPositiveN(t *testing.T) {
	p := newTestSelectorParser(":nth-child(+n)")
	// +n parses via sign prefix
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "+n" {
		t.Errorf("argument = %v, want '+n'", ps.Argument)
	}
}

func TestAnPlusBNegativeN(t *testing.T) {
	p := newTestSelectorParser(":nth-child(-n)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "-n" {
		t.Errorf("argument = %v, want '-n'", ps.Argument)
	}
}

func TestAnPlusBTwoN(t *testing.T) {
	p := newTestSelectorParser(":nth-child(2n)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "2n" {
		t.Errorf("argument = %v, want '2n'", ps.Argument)
	}
}

func TestAnPlusBTwoNPlusOne(t *testing.T) {
	p := newTestSelectorParser(":nth-child(2n+1)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "2n+1" {
		t.Errorf("argument = %v, want '2n+1'", ps.Argument)
	}
}

func TestAnPlusBBareNumber(t *testing.T) {
	p := newTestSelectorParser(":nth-child(5)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "5" {
		t.Errorf("argument = %v, want '5'", ps.Argument)
	}
}

func TestAnPlusBNPlusThree(t *testing.T) {
	p := newTestSelectorParser(":nth-child(n+3)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "n+3" {
		t.Errorf("argument = %v, want 'n+3'", ps.Argument)
	}
}

func TestAnPlusBErrorExpectedNumber(t *testing.T) {
	p := newTestSelectorParser(":nth-child(n+x)")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for n+x")
	}
}

// ============================================================================
// _pseudoSelector
// ============================================================================

func TestPseudoSelectorHover(t *testing.T) {
	p := newTestSelectorParser(":hover")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := sel.(*PseudoSelector)
	if !ok {
		t.Fatalf("expected PseudoSelector, got %T", sel)
	}
	if ps.Name != "hover" {
		t.Errorf("name = %q, want 'hover'", ps.Name)
	}
	if ps.Argument != nil {
		t.Error("expected nil argument")
	}
	if ps.Selector != nil {
		t.Error("expected nil selector")
	}
}

func TestPseudoSelectorPseudoElement(t *testing.T) {
	p := newTestSelectorParser("::before")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := sel.(*PseudoSelector)
	if !ok {
		t.Fatalf("expected PseudoSelector, got %T", sel)
	}
	if !ps.IsElement() {
		t.Error("expected element=true for ::before")
	}
}

func TestPseudoSelectorNot(t *testing.T) {
	p := newTestSelectorParser(":not(.foo)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := sel.(*PseudoSelector)
	if !ok {
		t.Fatalf("expected PseudoSelector, got %T", sel)
	}
	if ps.Selector == nil {
		t.Error("expected non-nil selector for :not()")
	}
}

func TestPseudoSelectorIs(t *testing.T) {
	p := newTestSelectorParser(":is(.foo)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Selector == nil {
		t.Error("expected non-nil selector for :is()")
	}
}

func TestPseudoSelectorSlotted(t *testing.T) {
	p := newTestSelectorParser("::slotted(.foo)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if !ps.IsElement() {
		t.Error("expected element=true for ::slotted")
	}
	if ps.Selector == nil {
		t.Error("expected non-nil selector for ::slotted()")
	}
}

func TestPseudoSelectorNthChildWithOf(t *testing.T) {
	p := newTestSelectorParser(":nth-child(2n+1 of .foo)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Selector == nil {
		t.Error("expected non-nil selector for :nth-child(... of ...)")
	}
}

func TestPseudoSelectorLang(t *testing.T) {
	p := newTestSelectorParser(":lang(en-us)")
	sel, err := p.ParseSimpleSelector()
	if err != nil {
		t.Fatal(err)
	}
	ps := sel.(*PseudoSelector)
	if ps.Argument == nil || *ps.Argument != "en-us" {
		t.Errorf("argument = %v, want 'en-us'", ps.Argument)
	}
	if ps.Selector != nil {
		t.Error("expected nil selector for :lang()")
	}
}

func TestPseudoSelectorErrorMissingCloseParen(t *testing.T) {
	p := newTestSelectorParser(":hover(")
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for missing )")
	}
}

// ============================================================================
// _compoundSelector
// ============================================================================

func TestCompoundSelectorSingle(t *testing.T) {
	p := newTestSelectorParser(".foo")
	cs, err := p.ParseCompoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(cs.Components))
	}
}

func TestCompoundSelectorMulti(t *testing.T) {
	p := newTestSelectorParser("a.b#c")
	cs, err := p.ParseCompoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(cs.Components))
	}
}

func TestCompoundSelectorAllowParent(t *testing.T) {
	p := newTestSelectorParser("&.foo")
	cs, err := p.ParseCompoundSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(cs.Components))
	}
	// First should be ParentSelector
	if _, ok := cs.Components[0].(*ParentSelector); !ok {
		t.Errorf("first component should be ParentSelector, got %T", cs.Components[0])
	}
}

func TestCompoundSelectorAllowParentFalse(t *testing.T) {
	p := newTestSelectorParserNoParent("&.foo")
	_, err := p.ParseCompoundSelector()
	if err == nil {
		t.Error("expected error for & when allowParent=false")
	}
}

// ============================================================================
// _complexSelector
// ============================================================================

func TestComplexSelectorParserSingleCompound(t *testing.T) {
	p := newTestSelectorParser(".foo")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(cs.Components))
	}
}

func TestComplexSelectorChildCombinator(t *testing.T) {
	p := newTestSelectorParser(".foo > .bar")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorNextSiblingCombinator(t *testing.T) {
	p := newTestSelectorParser(".foo + .bar")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorFollowingSiblingCombinator(t *testing.T) {
	p := newTestSelectorParser(".foo ~ .bar")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorDescendant(t *testing.T) {
	p := newTestSelectorParser(".foo .bar")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorLeadingCombinators(t *testing.T) {
	p := newTestSelectorParser("> .foo")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.LeadingCombinators) == 0 {
		t.Error("expected leading combinators for '> .foo'")
	}
}

func TestComplexSelectorTrailingCombinatorPlainCss(t *testing.T) {
	p := newTestSelectorParserPlainCss(".foo >")
	_, err := p.ParseComplexSelector()
	if err == nil {
		t.Error("expected error for trailing combinator in plain CSS")
	}
}

func TestComplexSelectorTrailingCombinatorNonPlainCss(t *testing.T) {
	p := newTestSelectorParser(".foo >")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(cs.Components))
	}
	if cs.LeadingCombinators == nil {
		t.Error("trailing > should NOT add leading combinators after first compound")
	}
}

func TestComplexSelectorLeadingCombinatorsOnly(t *testing.T) {
	// > alone parses as a complex selector with only leading combinators and no components
	p := newTestSelectorParser(">")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.LeadingCombinators) == 0 {
		t.Error("expected leading combinators for '>'")
	}
	if len(cs.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorAmpersandSecondPositionError(t *testing.T) {
	// "a b &" — & is consumed as a third compound with a ParentSelector
	p := newTestSelectorParser("a b &")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 3 {
		t.Fatalf("expected 3 compounds, got %d", len(cs.Components))
	}
}

func TestComplexSelectorMultipleCombinators(t *testing.T) {
	p := newTestSelectorParser("a > b + c")
	cs, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(cs.Components))
	}
}

func TestComplexSelectorDeprecationCallback(t *testing.T) {
	// Verify the parser constructor stores and accepts the deprecation callback option
	var called bool
	p := NewSelectorParser(".foo", nil, nil, &SelectorParserOptions{
		AllowParent: ptrBool(true),
		WarnDeprecationFn: func(msg string, dep *deprecation.Deprecation) error {
			called = true
			return nil
		},
	})
	_, err := p.ParseComplexSelector()
	if err != nil {
		t.Fatal(err)
	}
	_ = called
}

func ptrBool(b bool) *bool { return &b }

// ============================================================================
// _selectorList
// ============================================================================

func TestSelectorListSingle(t *testing.T) {
	p := newTestSelectorParser(".foo")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(list.Components))
	}
}

func TestSelectorListComma(t *testing.T) {
	p := newTestSelectorParser(".foo, .bar")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(list.Components))
	}
}

func TestSelectorListConsecutiveCommas(t *testing.T) {
	p := newTestSelectorParser(".foo,, .bar")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components after consecutive commas, got %d", len(list.Components))
	}
}

func TestSelectorListCommaLineBreak(t *testing.T) {
	p := newTestSelectorParser(".foo,\n.bar")
	list, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(list.Components))
	}
	if !list.Components[1].LineBreak {
		t.Error("expected lineBreak=true for selector after newline")
	}
}

func TestSelectorListErrorEmpty(t *testing.T) {
	p := newTestSelectorParser("")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for empty selector list")
	}
}

// ============================================================================
// Error propagation tests
// ============================================================================

func TestReadCharErrorMidSelector(t *testing.T) {
	// Input that starts valid but hits an unexpected end mid-parse
	p := newTestSelectorParser(`[foo="unterminated`)
	_, err := p.ParseSimpleSelector()
	if err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestScannerErrorPropagation(t *testing.T) {
	p := newTestSelectorParser(`div.`)
	// Starts with type selector "div", then "." but no ident after
	_, err := p.ParseCompoundSelector()
	if err == nil {
		t.Error("expected error for . without identifier")
	}
}
