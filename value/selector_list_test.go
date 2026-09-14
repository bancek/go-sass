package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func makeCompoundForList() (*CompoundSelector, error) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	return NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
}

func makeComplexForList() (*ComplexSelector, error) {
	compound, err := makeCompoundForList()
	if err != nil {
		return nil, err
	}
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	return NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
}

func TestNewSelectorList(t *testing.T) {
	complex, err := makeComplexForList()
	if err != nil {
		t.Fatal(err)
	}
	list, err := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(list.Components))
	}
}

func TestNewSelectorListEmpty(t *testing.T) {
	_, err := NewSelectorList([]*ComplexSelector{}, sasscommon.BogusSpan)
	if err == nil {
		t.Error("Expected error for empty components")
	}
}

func TestSelectorListIsInvisible(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	if list.IsInvisible() {
		t.Error("SelectorList with visible complex should not be invisible")
	}
}

func TestSelectorListIsBogus(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	if list.IsBogus() {
		t.Error("SelectorList with non-bogus complex should not be bogus")
	}
}

func TestSelectorListIsUseless(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	if list.IsUseless() {
		t.Error("SelectorList should not be useless")
	}
}

func TestSelectorListContainsParentSelector(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	got, err := list.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Should not contain parent selector")
	}

	parent := NewParentSelector(sasscommon.BogusSpan, nil)
	compound, _ := NewCompoundSelector([]SimpleSelector{parent}, sasscommon.BogusSpan)
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	complex2, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	list2, _ := NewSelectorList([]*ComplexSelector{complex2}, sasscommon.BogusSpan)
	got, err = list2.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Should contain parent selector")
	}
}

func TestSelectorListHashCode(t *testing.T) {
	complex1, _ := makeComplexForList()
	complex2, _ := makeComplexForList()
	list1, _ := NewSelectorList([]*ComplexSelector{complex1}, sasscommon.BogusSpan)
	list2, _ := NewSelectorList([]*ComplexSelector{complex2}, sasscommon.BogusSpan)
	if list1.HashCode() != list2.HashCode() {
		t.Error("Same construction should produce same hash")
	}
}

func TestSelectorListString(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	got, err := list.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ".foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestSelectorListAssertNotBogus(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	if err := list.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestSelectorListWithAdditionalCombinators(t *testing.T) {
	complex, _ := makeComplexForList()
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)

	comb := sasscommon.NewCssValue(CombinatorChild, sasscommon.BogusSpan)
	result, err := list.WithAdditionalCombinators([]sasscommon.CssValue[Combinator]{comb})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Components) != 1 {
		t.Error("Should still have one component")
	}
}
