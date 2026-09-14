// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/every_css.dart

// EveryCssVisitor visits CSS nodes and returns true if all individual visit
// methods return true. Each method returns false by default.
//
// Parent kinds fold children with logical-and; leaves (comment, declaration,
// import) are always false. This backs the isInvisible queries. Matches Dart:
// EveryCssVisitor mixin (internal in Dart).
type EveryCssVisitor struct{}

func (v *EveryCssVisitor) VisitCssAtRule(node CssAtRule) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (v *EveryCssVisitor) VisitCssComment(CssComment) (bool, error) { return false, nil }

func (v *EveryCssVisitor) VisitCssDeclaration(CssDeclaration) (bool, error) { return false, nil }

func (v *EveryCssVisitor) VisitCssImport(CssImport) (bool, error) { return false, nil }

func (v *EveryCssVisitor) VisitCssKeyframeBlock(node CssKeyframeBlock) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (v *EveryCssVisitor) VisitCssMediaRule(node CssMediaRule) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (v *EveryCssVisitor) VisitCssStyleRule(node CssStyleRule) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (v *EveryCssVisitor) VisitCssStylesheet(node *CssStylesheet) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (v *EveryCssVisitor) VisitCssSupportsRule(node CssSupportsRule) (bool, error) {
	for _, child := range node.Children() {
		ok, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

var _ CssVisitor[bool] = (*EveryCssVisitor)(nil)
