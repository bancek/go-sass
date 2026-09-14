package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewComplexSelectorComponent(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	c := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	if c.Selector != compound {
		t.Error("Selector should match constructor argument")
	}
}

func TestComplexSelectorComponentHashCode(t *testing.T) {
	class1 := NewClassSelector("foo", sasscommon.BogusSpan)
	compound1, _ := NewCompoundSelector([]SimpleSelector{class1}, sasscommon.BogusSpan)
	c1 := NewComplexSelectorComponent(compound1, nil, sasscommon.BogusSpan)

	class2 := NewClassSelector("foo", sasscommon.BogusSpan)
	compound2, _ := NewCompoundSelector([]SimpleSelector{class2}, sasscommon.BogusSpan)
	c2 := NewComplexSelectorComponent(compound2, nil, sasscommon.BogusSpan)

	if c1.HashCode() != c2.HashCode() {
		t.Error("Same fields should produce same hash")
	}
}

func TestComplexSelectorComponentWithAdditionalCombinators(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	c := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)

	comb := sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)
	result := c.WithAdditionalCombinators([]sasscommon.CssValue[Combinator]{comb})
	if len(result.Combinators) != 1 {
		t.Errorf("len(Combinators) = %d, want 1", len(result.Combinators))
	}
	if result.Combinators[0].Value != CombinatorChild {
		t.Error("Combinator should be Child")
	}
}

func TestComplexSelectorComponentWithAdditionalCombinatorsEmpty(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	c := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)

	result := c.WithAdditionalCombinators(nil)
	if len(result.Combinators) != 0 {
		t.Error("Should not add empty combinators")
	}
}

func TestComplexSelectorComponentEqual(t *testing.T) {
	class1 := NewClassSelector("foo", sasscommon.BogusSpan)
	compound1, _ := NewCompoundSelector([]SimpleSelector{class1}, sasscommon.BogusSpan)
	c1 := NewComplexSelectorComponent(compound1, nil, sasscommon.BogusSpan)

	class2 := NewClassSelector("foo", sasscommon.BogusSpan)
	compound2, _ := NewCompoundSelector([]SimpleSelector{class2}, sasscommon.BogusSpan)
	c2 := NewComplexSelectorComponent(compound2, nil, sasscommon.BogusSpan)

	if !c1.Equal(c2) {
		t.Error("Same selector + combinators should be equal")
	}

	class3 := NewClassSelector("bar", sasscommon.BogusSpan)
	compound3, _ := NewCompoundSelector([]SimpleSelector{class3}, sasscommon.BogusSpan)
	c3 := NewComplexSelectorComponent(compound3, nil, sasscommon.BogusSpan)

	if c1.Equal(c3) {
		t.Error("Different selectors should not be equal")
	}
}
