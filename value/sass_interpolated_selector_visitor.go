// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/interpolated_selector.dart

// InterpolatedSelectorVisitor traverses interpolated selector nodes.
//
// Each method handles one selector type and returns a caller-chosen result,
// so the same traversal shape serves resolving, searching, and effectful
// passes. Nodes dispatch through their AcceptVoid methods.
type InterpolatedSelectorVisitor[T any] interface {
	VisitAttributeSelector(*InterpolatedAttributeSelector) (T, error)
	VisitClassSelector(*InterpolatedClassSelector) (T, error)
	VisitComplexSelector(*InterpolatedComplexSelector) (T, error)
	VisitCompoundSelector(*InterpolatedCompoundSelector) (T, error)
	VisitIDSelector(*InterpolatedIDSelector) (T, error)
	VisitParentSelector(*InterpolatedParentSelector) (T, error)
	VisitPlaceholderSelector(*InterpolatedPlaceholderSelector) (T, error)
	VisitPseudoSelector(*InterpolatedPseudoSelector) (T, error)
	VisitSelectorList(*InterpolatedSelectorList) (T, error)
	VisitTypeSelector(*InterpolatedTypeSelector) (T, error)
	VisitUniversalSelector(*InterpolatedUniversalSelector) (T, error)
}

func (s *InterpolatedAttributeSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitAttributeSelector(s)
}

func (s *InterpolatedClassSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitClassSelector(s)
}

func (s *InterpolatedComplexSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitComplexSelector(s)
}

func (s *InterpolatedCompoundSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitCompoundSelector(s)
}

func (s *InterpolatedIDSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitIDSelector(s)
}

func (s *InterpolatedParentSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitParentSelector(s)
}

func (s *InterpolatedPlaceholderSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitPlaceholderSelector(s)
}

func (s *InterpolatedPseudoSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitPseudoSelector(s)
}

func (s *InterpolatedSelectorList) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitSelectorList(s)
}

func (s *InterpolatedTypeSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitTypeSelector(s)
}

func (s *InterpolatedUniversalSelector) AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitUniversalSelector(s)
}
