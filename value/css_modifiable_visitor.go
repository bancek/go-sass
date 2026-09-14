// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/modifiable_css.dart

// ModifiableCssVisitor visits modifiable CSS AST nodes.
//
// In Dart, CssVisitor<T> extends ModifiableCssVisitor<T>, so a single class
// can implement both. In Go, methods are prefixed with VisitModifiable* to
// avoid name collisions with CssVisitor, since Go cannot overload method
// names by parameter type. Matches Dart: ModifiableCssVisitor.
type ModifiableCssVisitor interface {
	VisitModifiableCssAtRule(*ModifiableCssAtRule) (struct{}, error)
	VisitModifiableCssComment(*ModifiableCssComment) (struct{}, error)
	VisitModifiableCssDeclaration(*ModifiableCssDeclaration) (struct{}, error)
	VisitModifiableCssImport(*ModifiableCssImport) (struct{}, error)
	VisitModifiableCssKeyframeBlock(*ModifiableCssKeyframeBlock) (struct{}, error)
	VisitModifiableCssMediaRule(*ModifiableCssMediaRule) (struct{}, error)
	VisitModifiableCssStyleRule(*ModifiableCssStyleRule) (struct{}, error)
	VisitModifiableCssStylesheet(*ModifiableCssStylesheet) (struct{}, error)
	VisitModifiableCssSupportsRule(*ModifiableCssSupportsRule) (struct{}, error)
}

// CloneCssVisitor visits modifiable CSS AST nodes and returns cloned copies.
// Used by clone_css.go to implement deep copying with extension store mapping.
// Matches Dart's _CloneCssVisitor implementing CssVisitor<ModifiableCssNode>.
//
// Each VisitClone* method deep-copies one node kind, rebinding selectors
// through the clone map.
type CloneCssVisitor interface {
	VisitCloneCssAtRule(*ModifiableCssAtRule) (ModifiableCssNode, error)
	VisitCloneCssComment(*ModifiableCssComment) (ModifiableCssNode, error)
	VisitCloneCssDeclaration(*ModifiableCssDeclaration) (ModifiableCssNode, error)
	VisitCloneCssImport(*ModifiableCssImport) (ModifiableCssNode, error)
	VisitCloneCssKeyframeBlock(*ModifiableCssKeyframeBlock) (ModifiableCssNode, error)
	VisitCloneCssMediaRule(*ModifiableCssMediaRule) (ModifiableCssNode, error)
	VisitCloneCssStyleRule(*ModifiableCssStyleRule) (ModifiableCssNode, error)
	VisitCloneCssSupportsRule(*ModifiableCssSupportsRule) (ModifiableCssNode, error)
}
