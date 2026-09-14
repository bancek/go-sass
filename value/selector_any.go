// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/any_selector.dart

// AnySelectorVisitor reports whether any selector in a Sass selector tree
// satisfies a predicate. It is meant to be embedded: override one leaf visit
// method to return true and the composite visit methods propagate the hit up
// through compound, complex, and list levels, descending into the selector
// arguments of pseudo-selectors. Every method returns false by default.
type AnySelectorVisitor struct{}

// VisitComplexSelector returns true when any compound component matches.
func (v *AnySelectorVisitor) VisitComplexSelector(complex *ComplexSelector) (bool, error) {
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

// VisitCompoundSelector returns true when any simple component matches.
func (v *AnySelectorVisitor) VisitCompoundSelector(compound *CompoundSelector) (bool, error) {
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

// VisitPseudoSelector descends into the pseudo-selector's selector argument,
// if it has one; a bare pseudo without arguments never matches.
func (v *AnySelectorVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (bool, error) {
	if pseudo.Selector == nil {
		return false, nil
	}
	return pseudo.Selector.AcceptBool(v)
}

// VisitSelectorList returns true when any complex component matches.
func (v *AnySelectorVisitor) VisitSelectorList(list *SelectorList) (bool, error) {
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

func (v *AnySelectorVisitor) VisitAttributeSelector(*AttributeSelector) (bool, error) {
	return false, nil
}
func (v *AnySelectorVisitor) VisitClassSelector(*ClassSelector) (bool, error)   { return false, nil }
func (v *AnySelectorVisitor) VisitIDSelector(*IDSelector) (bool, error)         { return false, nil }
func (v *AnySelectorVisitor) VisitParentSelector(*ParentSelector) (bool, error) { return false, nil }
func (v *AnySelectorVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (bool, error) {
	return false, nil
}
func (v *AnySelectorVisitor) VisitTypeSelector(*TypeSelector) (bool, error) { return false, nil }
func (v *AnySelectorVisitor) VisitUniversalSelector(*UniversalSelector) (bool, error) {
	return false, nil
}

var _ SelectorVisitor[bool] = (*AnySelectorVisitor)(nil)
