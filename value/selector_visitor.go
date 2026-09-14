// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/selector.dart

// SelectorVisitor traverses selector AST nodes, with one visit method per
// concrete selector type. The type parameter selects the result: bool for
// predicate searches, struct{} for effect-only traversals, and
// *ParentSelector for parent-selector searches.
type SelectorVisitor[T any] interface {
	VisitAttributeSelector(*AttributeSelector) (T, error)
	VisitClassSelector(*ClassSelector) (T, error)
	VisitComplexSelector(*ComplexSelector) (T, error)
	VisitCompoundSelector(*CompoundSelector) (T, error)
	VisitIDSelector(*IDSelector) (T, error)
	VisitSelectorList(*SelectorList) (T, error)
	VisitParentSelector(*ParentSelector) (T, error)
	VisitPlaceholderSelector(*PlaceholderSelector) (T, error)
	VisitPseudoSelector(*PseudoSelector) (T, error)
	VisitTypeSelector(*TypeSelector) (T, error)
	VisitUniversalSelector(*UniversalSelector) (T, error)
}
