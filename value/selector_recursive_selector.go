// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/recursive_selector.dart

// RecursiveSelectorVisitor walks every component of a selector tree for its
// side effects. It is meant to be embedded: override one visit method and the
// composite visit methods still drive the traversal into compound components
// and pseudo-selector arguments. Leaf visit methods do nothing by default.
//
// Unlike Dart's void mixin, the Go port returns (struct{}, error) so
// traversal failures propagate through the AcceptVoid family.
type RecursiveSelectorVisitor struct{}

func (v *RecursiveSelectorVisitor) VisitAttributeSelector(*AttributeSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitClassSelector(*ClassSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitIDSelector(*IDSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitParentSelector(*ParentSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitTypeSelector(*TypeSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitUniversalSelector(*UniversalSelector) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitComplexSelector(complex *ComplexSelector) (struct{}, error) {
	for _, component := range complex.Components {
		if _, err := v.VisitCompoundSelector(component.Selector); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitCompoundSelector(compound *CompoundSelector) (struct{}, error) {
	for _, simple := range compound.Components {
		if _, err := simple.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (struct{}, error) {
	if pseudo.Selector != nil {
		return pseudo.Selector.AcceptVoid(v)
	}
	return struct{}{}, nil
}

func (v *RecursiveSelectorVisitor) VisitSelectorList(list *SelectorList) (struct{}, error) {
	for _, complex := range list.Components {
		if _, err := v.VisitComplexSelector(complex); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

var _ SelectorVisitor[struct{}] = (*RecursiveSelectorVisitor)(nil)
