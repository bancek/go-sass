package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func makeCompound() (*CompoundSelector, error) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	return NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
}

func makeComponent() (*ComplexSelectorComponent, *CompoundSelector, error) {
	compound, err := makeCompound()
	if err != nil {
		return nil, nil, err
	}
	return NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan), compound, nil
}

func TestNewComplexSelector(t *testing.T) {
	comp, compound, err := makeComponent()
	if err != nil {
		t.Fatal(err)
	}
	cs, err := NewComplexSelector(
		nil,
		[]*ComplexSelectorComponent{comp},
		sasscommon.BogusSpan,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(cs.Components))
	}
	if cs.Components[0].Selector != compound {
		t.Error("Component selector should match")
	}
}

func TestNewComplexSelectorEmpty(t *testing.T) {
	_, err := NewComplexSelector(nil, nil, sasscommon.BogusSpan, false)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestComplexSelectorIsInvisible(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if cs.IsInvisible() {
		t.Error("Should not be invisible")
	}
}

func TestComplexSelectorIsBogus(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if cs.IsBogus() {
		t.Error("Should not be bogus")
	}
}

func TestComplexSelectorIsUseless(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if cs.IsUseless() {
		t.Error("Should not be useless")
	}
}

func TestComplexSelectorSingleCompound(t *testing.T) {
	comp, compound, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	result := cs.SingleCompound()
	if result == nil {
		t.Fatal("Expected non-nil")
	}
	if result != compound {
		t.Error("Should return the single compound")
	}
}

func TestComplexSelectorSpecificity(t *testing.T) {
	class1 := NewClassSelector("foo", sasscommon.BogusSpan)
	class2 := NewClassSelector("bar", sasscommon.BogusSpan)
	compound1, _ := NewCompoundSelector([]SimpleSelector{class1}, sasscommon.BogusSpan)
	compound2, _ := NewCompoundSelector([]SimpleSelector{class2}, sasscommon.BogusSpan)
	comp1 := NewComplexSelectorComponent(compound1, nil, sasscommon.BogusSpan)
	comp2 := NewComplexSelectorComponent(compound2, nil, sasscommon.BogusSpan)

	cs, _ := NewComplexSelector(
		nil,
		[]*ComplexSelectorComponent{comp1, comp2},
		sasscommon.BogusSpan,
		false,
	)
	if sp := cs.Specificity(); sp != 2000 {
		t.Errorf("Specificity = %d, want 2000", sp)
	}
}

func TestComplexSelectorHashCode(t *testing.T) {
	comp1, _, _ := makeComponent()
	cs1, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp1}, sasscommon.BogusSpan, false)
	comp2, _, _ := makeComponent()
	cs2, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp2}, sasscommon.BogusSpan, false)
	if cs1.HashCode() != cs2.HashCode() {
		t.Error("Same construction should produce same hash")
	}
}

func TestComplexSelectorWithAdditionalCombinators(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)

	comb := sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)
	result := cs.WithAdditionalCombinators([]sasscommon.CssValue[Combinator]{comb}, false)
	if len(result.Components) != 1 {
		t.Error("Should still have one component")
	}
	if len(result.Components[0].Combinators) != 1 {
		t.Errorf("Component should have 1 combinator, got %d", len(result.Components[0].Combinators))
	}
}

func TestComplexSelectorConcatenate(t *testing.T) {
	comp1, _, _ := makeComponent()
	comp2, _, _ := makeComponent()
	cs1, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp1}, sasscommon.BogusSpan, false)
	cs2, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp2}, sasscommon.BogusSpan, false)

	result := cs1.Concatenate(cs2, sasscommon.BogusSpan, false)
	if len(result.Components) != 2 {
		t.Errorf("len(Components) = %d, want 2", len(result.Components))
	}
}

func TestComplexSelectorString(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	got, err := cs.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ".foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestComplexSelectorAssertNotBogus(t *testing.T) {
	comp, _, _ := makeComponent()
	cs, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if err := cs.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestComplexSelectorEquals(t *testing.T) {
	comp1, _, _ := makeComponent()
	cs1, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp1}, sasscommon.BogusSpan, false)
	comp2, _, _ := makeComponent()
	cs2, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp2}, sasscommon.BogusSpan, false)
	if !cs1.Equals(cs2) {
		t.Error("Same construction should be equal")
	}

	cs3, _ := NewComplexSelector(
		[]sasscommon.CssValue[Combinator]{sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)},
		[]*ComplexSelectorComponent{comp1},
		sasscommon.BogusSpan,
		false,
	)
	if cs1.Equals(cs3) {
		t.Error("Different leading combinators should not be equal")
	}
}

func TestComplexSelectorParse(t *testing.T) {
	cs, err := ComplexSelectorParse(".foo", nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cs == nil {
		t.Fatal("expected non-nil")
	}
	if len(cs.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(cs.Components))
	}
}

func TestComplexSelectorParseError(t *testing.T) {
	_, err := ComplexSelectorParse("", nil, true, nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
