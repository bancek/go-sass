// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/css.dart

// CssVisitor visits plain CSS AST nodes.
//
// In Dart this extends ModifiableCssVisitor so one class serves both trees;
// in Go the modifiable visitor uses VisitModifiable* names to avoid method
// collisions. Matches Dart: CssVisitor.
type CssVisitor[T any] interface {
	VisitCssAtRule(CssAtRule) (T, error)
	VisitCssComment(CssComment) (T, error)
	VisitCssDeclaration(CssDeclaration) (T, error)
	VisitCssImport(CssImport) (T, error)
	VisitCssKeyframeBlock(CssKeyframeBlock) (T, error)
	VisitCssMediaRule(CssMediaRule) (T, error)
	VisitCssStyleRule(CssStyleRule) (T, error)
	VisitCssStylesheet(*CssStylesheet) (T, error)
	VisitCssSupportsRule(CssSupportsRule) (T, error)
}
