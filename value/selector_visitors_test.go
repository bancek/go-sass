package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// findParentVisitor fully implements SelectorVisitor[bool] and returns true
// when it finds a ParentSelector.
type findParentVisitor struct{}

func (v *findParentVisitor) VisitParentSelector(*ParentSelector) (bool, error) { return true, nil }
func (v *findParentVisitor) VisitAttributeSelector(*AttributeSelector) (bool, error) {
	return false, nil
}
func (v *findParentVisitor) VisitClassSelector(*ClassSelector) (bool, error) { return false, nil }
func (v *findParentVisitor) VisitIDSelector(*IDSelector) (bool, error)       { return false, nil }
func (v *findParentVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (bool, error) {
	return false, nil
}
func (v *findParentVisitor) VisitTypeSelector(*TypeSelector) (bool, error) { return false, nil }
func (v *findParentVisitor) VisitUniversalSelector(*UniversalSelector) (bool, error) {
	return false, nil
}
func (v *findParentVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (bool, error) {
	if pseudo.Selector != nil {
		return pseudo.Selector.AcceptBool(v)
	}
	return false, nil
}
func (v *findParentVisitor) VisitCompoundSelector(compound *CompoundSelector) (bool, error) {
	for _, s := range compound.Components {
		r, err := s.AcceptBool(v)
		if err != nil || r {
			return r, err
		}
	}
	return false, nil
}
func (v *findParentVisitor) VisitComplexSelector(complex *ComplexSelector) (bool, error) {
	for _, comp := range complex.Components {
		r, err := v.VisitCompoundSelector(comp.Selector)
		if err != nil || r {
			return r, err
		}
	}
	return false, nil
}
func (v *findParentVisitor) VisitSelectorList(list *SelectorList) (bool, error) {
	for _, c := range list.Components {
		r, err := v.VisitComplexSelector(c)
		if err != nil || r {
			return r, err
		}
	}
	return false, nil
}

func TestFindParentVisitor(t *testing.T) {
	v := &findParentVisitor{}

	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	result, err := v.VisitCompoundSelector(compound)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("Should not find parent in ClassSelector-only compound")
	}

	parent := NewParentSelector(sasscommon.BogusSpan, nil)
	compound2, _ := NewCompoundSelector([]SimpleSelector{parent}, sasscommon.BogusSpan)
	result, err = v.VisitCompoundSelector(compound2)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Should find ParentSelector")
	}
}

func TestSearchVisitorBase(t *testing.T) {
	v := &SelectorSearchVisitor{}

	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	result, err := v.VisitCompoundSelector(compound)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("SearchVisitor base returns false for all leaves")
	}
}

func TestRecursiveSelectorVisitor(t *testing.T) {
	v := &RecursiveSelectorVisitor{}

	class := NewClassSelector("foo", sasscommon.BogusSpan)
	compound, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	complex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)

	_, err := v.VisitSelectorList(list)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnyVisitorBase(t *testing.T) {
	v := &AnySelectorVisitor{}

	parent := NewParentSelector(sasscommon.BogusSpan, nil)
	compound, _ := NewCompoundSelector([]SimpleSelector{parent}, sasscommon.BogusSpan)
	result, err := v.VisitCompoundSelector(compound)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("AnySelector base returns false for all leaves")
	}
}

func TestIsParentSelector(t *testing.T) {
	parent := NewParentSelector(sasscommon.BogusSpan, nil)
	if !IsParentSelector(parent) {
		t.Error("IsParentSelector(ParentSelector) should be true")
	}
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	if IsParentSelector(class) {
		t.Error("IsParentSelector(ClassSelector) should be false")
	}
}
