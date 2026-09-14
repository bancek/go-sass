// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/selector_search.dart

// SelectorSearchVisitor finds the first matching node in a selector tree.
//
// To search for a specific selector type, embed or compose this visitor and
// override the corresponding Visit method to return (true, nil). Use a
// side-channel field on the composed struct to capture the found value.
//
// Unlike Dart's generic search visitor, which threads a nullable result
// through, this port specializes to bool: composite visit methods short
// circuit on the first true and descend into pseudo-selector arguments.
// Each leaf method returns false by default.
type SelectorSearchVisitor struct{}

func (v *SelectorSearchVisitor) VisitAttributeSelector(*AttributeSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitClassSelector(*ClassSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitIDSelector(*IDSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitParentSelector(*ParentSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitTypeSelector(*TypeSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitUniversalSelector(*UniversalSelector) (bool, error) {
	return false, nil
}

func (v *SelectorSearchVisitor) VisitComplexSelector(complex *ComplexSelector) (bool, error) {
	for _, component := range complex.Components {
		result, err := v.VisitCompoundSelector(component.Selector)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *SelectorSearchVisitor) VisitCompoundSelector(compound *CompoundSelector) (bool, error) {
	for _, simple := range compound.Components {
		result, err := simple.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *SelectorSearchVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (bool, error) {
	if pseudo.Selector != nil {
		return pseudo.Selector.AcceptBool(v)
	}
	return false, nil
}

func (v *SelectorSearchVisitor) VisitSelectorList(list *SelectorList) (bool, error) {
	for _, complex := range list.Components {
		result, err := v.VisitComplexSelector(complex)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

var _ SelectorVisitor[bool] = (*SelectorSearchVisitor)(nil)
