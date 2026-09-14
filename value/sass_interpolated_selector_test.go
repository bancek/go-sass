package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func iselSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("interpolated selector test"), nil)
	return sasscommon.NewFileSpan(fs, 0, 26)
}

func iselPlainInterp(name string) *Interpolation {
	return NewInterpolationPlain(name, iselSpan())
}

// iselMockVisitor records which visit method was called.
type iselMockVisitor struct {
	visited string
}

func (v *iselMockVisitor) VisitAttributeSelector(n *InterpolatedAttributeSelector) (struct{}, error) {
	v.visited = "Attribute"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitClassSelector(n *InterpolatedClassSelector) (struct{}, error) {
	v.visited = "Class"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitComplexSelector(n *InterpolatedComplexSelector) (struct{}, error) {
	v.visited = "Complex"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitCompoundSelector(n *InterpolatedCompoundSelector) (struct{}, error) {
	v.visited = "Compound"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitIDSelector(n *InterpolatedIDSelector) (struct{}, error) {
	v.visited = "ID"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitParentSelector(n *InterpolatedParentSelector) (struct{}, error) {
	v.visited = "Parent"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitPlaceholderSelector(n *InterpolatedPlaceholderSelector) (struct{}, error) {
	v.visited = "Placeholder"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitPseudoSelector(n *InterpolatedPseudoSelector) (struct{}, error) {
	v.visited = "Pseudo"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitSelectorList(n *InterpolatedSelectorList) (struct{}, error) {
	v.visited = "SelectorList"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitTypeSelector(n *InterpolatedTypeSelector) (struct{}, error) {
	v.visited = "Type"
	return struct{}{}, nil
}
func (v *iselMockVisitor) VisitUniversalSelector(n *InterpolatedUniversalSelector) (struct{}, error) {
	v.visited = "Universal"
	return struct{}{}, nil
}

// ---------------------------------------------------------------------------
// InterpolatedQualifiedName
// ---------------------------------------------------------------------------

func TestInterpolatedQualifiedNameConstruction(t *testing.T) {
	name := iselPlainInterp("div")
	span := iselSpan()
	qn := NewInterpolatedQualifiedName(name, span, nil)

	s, err := qn.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
	if qn.Namespace != nil {
		t.Error("namespace should be nil")
	}
}

func TestInterpolatedQualifiedNameWithNamespace(t *testing.T) {
	name := iselPlainInterp("div")
	span := iselSpan()
	ns := iselPlainInterp("svg")
	qn := NewInterpolatedQualifiedName(name, span, ns)

	if qn.Namespace == nil {
		t.Fatal("namespace should not be nil")
	}
}

func TestInterpolatedQualifiedNameString(t *testing.T) {
	name := iselPlainInterp("div")
	span := iselSpan()
	qn := NewInterpolatedQualifiedName(name, span, nil)

	s, err := qn.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "div" {
		t.Errorf("String() = %q, want %q", s, "div")
	}
}

func TestInterpolatedQualifiedNameStringWithNamespace(t *testing.T) {
	name := iselPlainInterp("div")
	span := iselSpan()
	ns := iselPlainInterp("svg")
	qn := NewInterpolatedQualifiedName(name, span, ns)

	s, err := qn.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "svg|div" {
		t.Errorf("String() = %q, want %q", s, "svg|div")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedClassSelector
// ---------------------------------------------------------------------------

func TestInterpolatedClassSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("foo")
	sel, err := NewInterpolatedClassSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameS, err := name.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameStart, err := nameS.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	selStart, err := s.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	if selStart.Offset != nameStart.Offset-1 {
		t.Errorf("span start = %d, expected %d (name span start - 1)",
			selStart.Offset, nameStart.Offset-1)
	}
}

func TestInterpolatedClassSelectorString(t *testing.T) {
	name := iselPlainInterp("foo")
	sel, err := NewInterpolatedClassSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != ".foo" {
		t.Errorf("String() = %q, want %q", s, ".foo")
	}
}

func TestInterpolatedClassSelectorAccept(t *testing.T) {
	name := iselPlainInterp("foo")
	sel, _ := NewInterpolatedClassSelector(name)
	var v iselMockVisitor
	_, err := sel.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "Class" {
		t.Errorf("expected Class, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// InterpolatedIDSelector
// ---------------------------------------------------------------------------

func TestInterpolatedIDSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("bar")
	sel, err := NewInterpolatedIDSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameS, err := name.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameStart, _ := nameS.StartLocation()
	selStart, _ := s.StartLocation()
	if selStart.Offset != nameStart.Offset-1 {
		t.Error("span should extend one char left for hash prefix")
	}
}

func TestInterpolatedIDSelectorString(t *testing.T) {
	name := iselPlainInterp("bar")
	sel, err := NewInterpolatedIDSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "#bar" {
		t.Errorf("String() = %q, want %q", s, "#bar")
	}
}

func TestInterpolatedIDSelectorAccept(t *testing.T) {
	name := iselPlainInterp("bar")
	sel, _ := NewInterpolatedIDSelector(name)
	var v iselMockVisitor
	_, err := sel.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "ID" {
		t.Errorf("expected ID, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// InterpolatedPlaceholderSelector
// ---------------------------------------------------------------------------

func TestInterpolatedPlaceholderSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("baz")
	sel, err := NewInterpolatedPlaceholderSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameS, err := name.Span()
	if err != nil {
		t.Fatal(err)
	}
	nameStart, _ := nameS.StartLocation()
	selStart, _ := s.StartLocation()
	if selStart.Offset != nameStart.Offset-1 {
		t.Error("span should extend one char left for percent prefix")
	}
}

func TestInterpolatedPlaceholderSelectorString(t *testing.T) {
	name := iselPlainInterp("baz")
	sel, err := NewInterpolatedPlaceholderSelector(name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "%baz" {
		t.Errorf("String() = %q, want %q", s, "%baz")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedTypeSelector
// ---------------------------------------------------------------------------

func TestInterpolatedTypeSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("div")
	span := iselSpan()
	qn := NewInterpolatedQualifiedName(name, span, nil)
	sel := NewInterpolatedTypeSelector(qn)

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedUniversalSelector
// ---------------------------------------------------------------------------

func TestInterpolatedUniversalSelectorConstruction(t *testing.T) {
	span := iselSpan()
	sel := NewInterpolatedUniversalSelector(span, nil)

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestInterpolatedUniversalSelectorWithNamespace(t *testing.T) {
	span := iselSpan()
	ns := iselPlainInterp("svg")
	sel := NewInterpolatedUniversalSelector(span, ns)

	if sel.Namespace == nil {
		t.Fatal("namespace should not be nil")
	}
}

func TestInterpolatedUniversalSelectorString(t *testing.T) {
	span := iselSpan()
	sel := NewInterpolatedUniversalSelector(span, nil)

	s, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "*" {
		t.Errorf("String() = %q, want %q", s, "*")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedPseudoSelector
// ---------------------------------------------------------------------------

func TestInterpolatedPseudoSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("hover")
	span := iselSpan()
	sel := NewInterpolatedPseudoSelector(name, span, false, nil, nil)

	if !sel.IsSyntacticClass {
		t.Error("hover should be a pseudo-class")
	}
}

func TestInterpolatedPseudoSelectorElement(t *testing.T) {
	name := iselPlainInterp("before")
	span := iselSpan()
	sel := NewInterpolatedPseudoSelector(name, span, true, nil, nil)

	if !sel.IsSyntacticElement() {
		t.Error("before should be a pseudo-element")
	}
}

func TestInterpolatedPseudoSelectorWithArgument(t *testing.T) {
	name := iselPlainInterp("nth-child")
	span := iselSpan()
	arg := iselPlainInterp("2n+1")
	sel := NewInterpolatedPseudoSelector(name, span, false, arg, nil)

	if sel.Argument == nil {
		t.Fatal("argument should not be nil")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedParentSelector
// ---------------------------------------------------------------------------

func TestInterpolatedParentSelectorConstruction(t *testing.T) {
	span := iselSpan()
	sel := NewInterpolatedParentSelector(span, nil)

	s, err := sel.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestInterpolatedParentSelectorString(t *testing.T) {
	span := iselSpan()
	sel := NewInterpolatedParentSelector(span, nil)

	s, err := sel.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "&" {
		t.Errorf("String() = %q, want %q", s, "&")
	}
}

func TestInterpolatedParentSelectorWithSuffix(t *testing.T) {
	span := iselSpan()
	suffix := iselPlainInterp("suffix")
	sel := NewInterpolatedParentSelector(span, suffix)

	if sel.Suffix == nil {
		t.Fatal("suffix should not be nil")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedCompoundSelector
// ---------------------------------------------------------------------------

func TestInterpolatedCompoundSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	components := []InterpolatedSimpleSelector{class}
	compound, err := NewInterpolatedCompoundSelector(components)
	if err != nil {
		t.Fatal(err)
	}
	if len(compound.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(compound.Components))
	}
}

func TestInterpolatedCompoundSelectorEmpty(t *testing.T) {
	_, err := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{})
	if err == nil {
		t.Error("expected error for empty components")
	}
}

func TestInterpolatedCompoundSelectorAccept(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	var v iselMockVisitor
	_, err := compound.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "Compound" {
		t.Errorf("expected Compound, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// InterpolatedComplexSelectorComponent
// ---------------------------------------------------------------------------

func TestInterpolatedComplexSelectorComponentConstruction(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	span := iselSpan()
	c := NewInterpolatedComplexSelectorComponent(compound, span, nil)

	s, err := c.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

// ---------------------------------------------------------------------------
// InterpolatedComplexSelector
// ---------------------------------------------------------------------------

func TestInterpolatedComplexSelectorConstruction(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	span := iselSpan()
	comp := NewInterpolatedComplexSelectorComponent(compound, span, nil)
	cs, err := NewInterpolatedComplexSelector([]*InterpolatedComplexSelectorComponent{comp}, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cs.LeadingCombinator != nil {
		t.Error("leading combinator should be nil")
	}
}

func TestInterpolatedComplexSelectorEmpty(t *testing.T) {
	span := iselSpan()
	_, err := NewInterpolatedComplexSelector([]*InterpolatedComplexSelectorComponent{}, span, nil)
	if err == nil {
		t.Error("expected error for empty components with nil leading combinator")
	}
}

func TestInterpolatedComplexSelectorAccept(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	span := iselSpan()
	comp := NewInterpolatedComplexSelectorComponent(compound, span, nil)
	cs, _ := NewInterpolatedComplexSelector([]*InterpolatedComplexSelectorComponent{comp}, span, nil)
	var v iselMockVisitor
	_, err := cs.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "Complex" {
		t.Errorf("expected Complex, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// InterpolatedSelectorList
// ---------------------------------------------------------------------------

func TestInterpolatedSelectorListConstruction(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	span := iselSpan()
	comp := NewInterpolatedComplexSelectorComponent(compound, span, nil)
	cs, _ := NewInterpolatedComplexSelector([]*InterpolatedComplexSelectorComponent{comp}, span, nil)
	list, err := NewInterpolatedSelectorList([]*InterpolatedComplexSelector{cs}, span)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(list.Components))
	}
}

func TestInterpolatedSelectorListEmpty(t *testing.T) {
	span := iselSpan()
	_, err := NewInterpolatedSelectorList([]*InterpolatedComplexSelector{}, span)
	if err == nil {
		t.Error("expected error for empty components")
	}
}

func TestInterpolatedSelectorListAccept(t *testing.T) {
	name := iselPlainInterp("foo")
	class, _ := NewInterpolatedClassSelector(name)
	compound, _ := NewInterpolatedCompoundSelector([]InterpolatedSimpleSelector{class})
	span := iselSpan()
	comp := NewInterpolatedComplexSelectorComponent(compound, span, nil)
	cs, _ := NewInterpolatedComplexSelector([]*InterpolatedComplexSelectorComponent{comp}, span, nil)
	list, _ := NewInterpolatedSelectorList([]*InterpolatedComplexSelector{cs}, span)
	var v iselMockVisitor
	_, err := list.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "SelectorList" {
		t.Errorf("expected SelectorList, got %s", v.visited)
	}
}
