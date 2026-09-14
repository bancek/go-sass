package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// --- unifyUniversalAndElement tests ---

func TestUnifyUniversalAndElementBothType(t *testing.T) {
	s1 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	s2 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := unifyUniversalAndElement(s1, s2)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", result)
	}
	if ts.Name.Name != "div" {
		t.Errorf("Name = %q, want div", ts.Name.Name)
	}
}

func TestUnifyUniversalAndElementUniversalWildcardNSType(t *testing.T) {
	ns := "*"
	u := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	t2 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := unifyUniversalAndElement(u, t2)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", result)
	}
	if ts.Name.Name != "div" {
		t.Errorf("Name = %q, want div", ts.Name.Name)
	}
}

func TestUnifyUniversalAndElementTypeUniversalWildcardNS(t *testing.T) {
	t1 := NewTypeSelector(NewQualifiedName("span"), sasscommon.BogusSpan)
	ns := "*"
	u := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	result, err := unifyUniversalAndElement(t1, u)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", result)
	}
	if ts.Name.Name != "span" {
		t.Errorf("Name = %q, want span", ts.Name.Name)
	}
}

func TestUnifyUniversalAndElementConflictingNS(t *testing.T) {
	ns1 := "svg"
	ns2 := "html"
	u1 := NewUniversalSelector(sasscommon.BogusSpan, &ns1)
	u2 := NewUniversalSelector(sasscommon.BogusSpan, &ns2)
	result, err := unifyUniversalAndElement(u1, u2)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("should be nil for conflicting namespaces")
	}
}

func TestUnifyUniversalAndElementUniversalNilNameType(t *testing.T) {
	u := NewUniversalSelector(sasscommon.BogusSpan, nil)
	t2 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := unifyUniversalAndElement(u, t2)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", result)
	}
	if ts.Name.Name != "div" {
		t.Errorf("Name = %q, want div", ts.Name.Name)
	}
}

func TestUnifyUniversalAndElementBothUniversal(t *testing.T) {
	ns := "svg"
	u1 := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	u2 := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	result, err := unifyUniversalAndElement(u1, u2)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	us, ok := result.(*UniversalSelector)
	if !ok {
		t.Fatalf("expected UniversalSelector, got %T", result)
	}
	if us.Namespace == nil || *us.Namespace != "svg" {
		t.Errorf("Namespace = %v, want svg", us.Namespace)
	}
}

func TestUnifyUniversalAndElementNameConflict(t *testing.T) {
	t1 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	t2 := NewTypeSelector(NewQualifiedName("span"), sasscommon.BogusSpan)
	result, err := unifyUniversalAndElement(t1, t2)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("should be nil for conflicting names")
	}
}

func TestUnifyUniversalAndElementUniversalRHS(t *testing.T) {
	t1 := NewTypeSelector(NewQualifiedName("span"), sasscommon.BogusSpan)
	u := NewUniversalSelector(sasscommon.BogusSpan, nil)
	result, err := unifyUniversalAndElement(t1, u)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("expected TypeSelector, got %T", result)
	}
	if ts.Name.Name != "span" {
		t.Errorf("Name = %q, want span", ts.Name.Name)
	}
}

// --- UnifyComplex tests ---

func makeComplex(name string) *ComplexSelector {
	class := NewClassSelector(name, sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	return cs
}

func TestUnifyComplexSingle(t *testing.T) {
	a := makeComplex("foo")
	result, err := UnifyComplex([]*ComplexSelector{a}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func TestUnifyComplexUseless(t *testing.T) {
	bogusLC := []sasscommon.CssValue[Combinator]{
		sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan),
		sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan),
	}
	a := makeComplex("foo")
	useless, _ := NewComplexSelector(bogusLC, a.Components, sasscommon.BogusSpan, false)
	result, err := UnifyComplex([]*ComplexSelector{useless, a}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("should be nil for useless selector")
	}
}

func TestUnifyComplexTwoIdentical(t *testing.T) {
	a := makeComplex("foo")
	b := makeComplex("foo")
	result, err := UnifyComplex([]*ComplexSelector{a, b}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("should not be nil")
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

// --- CompoundIsSuperselector tests ---

func TestCompoundIsSuperselectorSame(t *testing.T) {
	a, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo should be superselector of .foo")
	}
}

func TestCompoundIsSuperselectorDifferent(t *testing.T) {
	a, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("bar", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error(".foo should not be superselector of .bar")
	}
}

func TestCompoundIsSuperselectorExtraClass(t *testing.T) {
	// .foo should be superselector of .foo.bar (less specific → matches more)
	a, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{
		NewClassSelector("foo", sasscommon.BogusSpan),
		NewClassSelector("bar", sasscommon.BogusSpan),
	}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo should be superselector of .foo.bar")
	}
}

func TestCompoundIsSuperselectorTypeWithClass(t *testing.T) {
	// div should be superselector of div.foo
	a, _ := NewCompoundSelector([]SimpleSelector{NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{
		NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan),
		NewClassSelector("foo", sasscommon.BogusSpan),
	}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("div should be superselector of div.foo")
	}
}

func TestCompoundIsSuperselectorTypeNotSuperOfClassed(t *testing.T) {
	// div.foo should NOT be superselector of div (more specific → matches fewer)
	a, _ := NewCompoundSelector([]SimpleSelector{
		NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan),
		NewClassSelector("foo", sasscommon.BogusSpan),
	}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("div.foo should not be superselector of div")
	}
}

func TestCompoundIsSuperselectorClassSuperOfType(t *testing.T) {
	// .foo should be superselector of div.foo (class-only matches more elements)
	a, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	b, _ := NewCompoundSelector([]SimpleSelector{
		NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan),
		NewClassSelector("foo", sasscommon.BogusSpan),
	}, sasscommon.BogusSpan)
	result, err := CompoundIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo should be superselector of div.foo")
	}
}

// --- ComplexIsSuperselector tests ---

func makeComps(names ...string) []*ComplexSelectorComponent {
	var comps []*ComplexSelectorComponent
	for _, name := range names {
		compound, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector(name, sasscommon.BogusSpan)}, sasscommon.BogusSpan)
		comps = append(comps, NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan))
	}
	return comps
}

func TestComplexIsSuperselectorSame(t *testing.T) {
	a := makeComps("foo")
	b := makeComps("foo")
	result, err := complexIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo should be superselector of .foo")
	}
}

func TestComplexIsSuperselectorDescendantOfChild(t *testing.T) {
	// .foo .bar (descendant) should be superselector of .foo > .bar (child)
	// because descendant matches any depth while child matches only immediate
	aC := makeComps("foo")
	aC = append(aC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	combChild := []sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)}
	bC := makeComps("foo")
	if len(bC) > 0 {
		bC[len(bC)-1].Combinators = combChild
	}
	bC = append(bC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	result, err := complexIsSuperselector(aC, bC)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo .bar should be superselector of .foo > .bar")
	}
}

func newCompound(name string) *CompoundSelector {
	c, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector(name, sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	return c
}

func TestComplexIsSuperselectorChildNotSuperOfDescendant(t *testing.T) {
	// .foo > .bar should NOT be superselector of .foo .bar
	combChild := []sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)}
	aC := makeComps("foo")
	if len(aC) > 0 {
		aC[len(aC)-1].Combinators = combChild
	}
	aC = append(aC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	bC := makeComps("foo")
	bC = append(bC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	result, err := complexIsSuperselector(aC, bC)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error(".foo > .bar should NOT be superselector of .foo .bar")
	}
}

func TestComplexIsSuperselectorNextSibling(t *testing.T) {
	combPlus := []sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorNextSibling, sasscommon.BogusSpan)}
	aC := makeComps("foo")
	if len(aC) > 0 {
		aC[len(aC)-1].Combinators = combPlus
	}
	aC = append(aC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	bC := makeComps("foo")
	if len(bC) > 0 {
		bC[len(bC)-1].Combinators = combPlus
	}
	bC = append(bC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	result, err := complexIsSuperselector(aC, bC)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo + .bar should be superselector of .foo + .bar")
	}
}

func TestComplexIsSuperselectorFollowingOfNext(t *testing.T) {
	combTilde := []sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorFollowingSibling, sasscommon.BogusSpan)}
	aC := makeComps("foo")
	if len(aC) > 0 {
		aC[len(aC)-1].Combinators = combTilde
	}
	aC = append(aC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	combPlus := []sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorNextSibling, sasscommon.BogusSpan)}
	bC := makeComps("foo")
	if len(bC) > 0 {
		bC[len(bC)-1].Combinators = combPlus
	}
	bC = append(bC, NewComplexSelectorComponent(
		newCompound("bar"), nil, sasscommon.BogusSpan))
	result, err := complexIsSuperselector(aC, bC)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo ~ .bar should be superselector of .foo + .bar")
	}
}

// --- ListIsSuperselector tests ---

func TestListIsSuperselectorExtraItem(t *testing.T) {
	a := []*ComplexSelector{makeComplex("foo"), makeComplex("bar")}
	b := []*ComplexSelector{makeComplex("foo")}
	result, err := ListIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error(".foo, .bar should be superselector of .foo")
	}
}

func TestListIsSuperselectorNotSuper(t *testing.T) {
	a := []*ComplexSelector{makeComplex("foo")}
	b := []*ComplexSelector{makeComplex("foo"), makeComplex("bar")}
	result, err := ListIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error(".foo should NOT be superselector of .foo, .bar")
	}
}

func TestListIsSuperselectorMissingItem(t *testing.T) {
	a := []*ComplexSelector{makeComplex("foo"), makeComplex("bar")}
	b := []*ComplexSelector{makeComplex("foo"), makeComplex("bar"), makeComplex("baz")}
	result, err := ListIsSuperselector(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error(".foo, .bar should NOT be superselector of .foo, .bar, .baz")
	}
}

// --- Weave tests ---

func TestWeaveSingle(t *testing.T) {
	a := makeComplex("foo")
	result, err := Weave([]*ComplexSelector{a}, sasscommon.BogusSpan, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func TestWeaveSingleForceLineBreak(t *testing.T) {
	a := makeComplex("foo")
	flb := true
	result, err := Weave([]*ComplexSelector{a}, sasscommon.BogusSpan, &flb)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if !result[0].LineBreak {
		t.Error("forceLineBreak should set line_break")
	}
}

func TestWeaveSingleAlreadyLineBreak(t *testing.T) {
	a := makeComplex("foo")
	a.LineBreak = true
	flb := true
	result, err := Weave([]*ComplexSelector{a}, sasscommon.BogusSpan, &flb)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if !result[0].LineBreak {
		t.Error("should preserve existing line_break")
	}
	// It should be the same pointer since line_break was already true
	if a != result[0] {
		t.Log("new complex created (expected, force LB with existing LB creates copy)")
	}
}

// --- NestWithin tests ---

func makeComplexForNest(name string) *ComplexSelector {
	return makeComplex(name)
}

func TestNestWithinNilParent(t *testing.T) {
	a, _ := NewSelectorList([]*ComplexSelector{makeComplexForNest("foo")}, sasscommon.BogusSpan)
	result, err := a.NestWithin(nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != a {
		t.Error("nest_within with nil parent should return self")
	}
}

func TestNestWithinImplicitParent(t *testing.T) {
	a, _ := NewSelectorList([]*ComplexSelector{makeComplexForNest("foo")}, sasscommon.BogusSpan)
	parent, _ := NewSelectorList([]*ComplexSelector{makeComplexForNest("div")}, sasscommon.BogusSpan)
	result, err := a.NestWithin(parent, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Components) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Components))
	}
}

func TestNestWithinParentSelector(t *testing.T) {
	ps := NewParentSelector(sasscommon.BogusSpan, nil)
	compound, _ := NewCompoundSelector([]SimpleSelector{ps}, sasscommon.BogusSpan)
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	complex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	a, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	pComp, _ := NewCompoundSelector([]SimpleSelector{NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	pCompC := NewComplexSelectorComponent(pComp, nil, sasscommon.BogusSpan)
	pComplex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{pCompC}, sasscommon.BogusSpan, false)
	parentList, _ := NewSelectorList([]*ComplexSelector{pComplex}, sasscommon.BogusSpan)
	result, err := a.NestWithin(parentList, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Components) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Components))
	}
}

func stringPtr(s string) *string { return &s }

func TestNestWithinParentSelectorWithSuffix(t *testing.T) {
	suffix := "bar"
	ps := NewParentSelector(sasscommon.BogusSpan, &suffix)
	compound, _ := NewCompoundSelector([]SimpleSelector{ps}, sasscommon.BogusSpan)
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	complex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	a, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	pComp, _ := NewCompoundSelector([]SimpleSelector{NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	pCompC := NewComplexSelectorComponent(pComp, nil, sasscommon.BogusSpan)
	pComplex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{pCompC}, sasscommon.BogusSpan, false)
	parentList, _ := NewSelectorList([]*ComplexSelector{pComplex}, sasscommon.BogusSpan)
	result, err := a.NestWithin(parentList, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Components) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Components))
	}
	// Should produce divbar (TypeSelector with suffix appended)
	lastCompounds := result.Components[0].Components[len(result.Components[0].Components)-1].Selector.Components
	if len(lastCompounds) != 1 {
		t.Fatalf("expected 1 simple (divbar), got %d", len(lastCompounds))
	}
	if ts, ok := lastCompounds[0].(*TypeSelector); ok {
		if ts.Name.Name != "divbar" {
			t.Errorf("Name = %q, want divbar", ts.Name.Name)
		}
	} else {
		t.Errorf("expected TypeSelector, got %T", lastCompounds[0])
	}
}

// --- ptrEq helper tests ---

func TestPtrEqBothNil(t *testing.T) {
	if !ptrEq(nil, nil) {
		t.Error("both nil should be equal")
	}
}

func TestPtrEqOneNil(t *testing.T) {
	s := "hello"
	if ptrEq(nil, &s) {
		t.Error("nil and non-nil should not be equal")
	}
	if ptrEq(&s, nil) {
		t.Error("non-nil and nil should not be equal")
	}
}

func TestPtrEqSameValue(t *testing.T) {
	a := "hello"
	b := "hello"
	if !ptrEq(&a, &b) {
		t.Error("same values should be equal")
	}
}

func TestPtrEqDifferentValues(t *testing.T) {
	a := "hello"
	b := "world"
	if ptrEq(&a, &b) {
		t.Error("different values should not be equal")
	}
}
