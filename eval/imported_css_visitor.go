// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/async_evaluate.dart (_ImportedCssVisitor)

import (
	"fmt"

	"github.com/bancek/go-sass/value"
)

// importedCssVisitor adds @imported CSS nodes to the root stylesheet,
// implementing CssVisitor[struct{}].
//
// An imported stylesheet cannot be evaluated with the importing stylesheet as
// its root, because it may @use modules that must be injected before the
// imported CSS. The visitor's own CSS handling cannot be reused either: it
// would attach the parent selector when the @import is nested, but that
// selector was already attached when the imported stylesheet was evaluated.
type importedCssVisitor struct {
	v *EvaluateVisitor
}

// VisitCssAtRule adds an at-rule, bubbling through style rules unless the
// rule is childless and attaches directly.
func (iv *importedCssVisitor) VisitCssAtRule(node value.CssAtRule) (struct{}, error) {
	if node.IsChildless() {
		return struct{}{}, iv.v.addChild(node.(value.ModifiableCssNode))
	}
	return struct{}{}, iv.v.addChildThrough(node.(value.ModifiableCssNode), func(child value.CssNode) bool {
		_, ok := child.(value.CssStyleRule)
		return ok
	})
}

// VisitCssComment adds a comment directly to the current parent.
func (iv *importedCssVisitor) VisitCssComment(node value.CssComment) (struct{}, error) {
	return struct{}{}, iv.v.addChild(node.(value.ModifiableCssNode))
}

// VisitCssDeclaration adds a declaration directly to the current parent.
func (iv *importedCssVisitor) VisitCssDeclaration(node value.CssDeclaration) (struct{}, error) {
	return struct{}{}, iv.v.addChild(node.(value.ModifiableCssNode))
}

// VisitCssImport adds an import, keeping @import order at the root. Nested
// imports attach directly to the current parent; root-level imports extend
// the leading import run (advancing endOfImports), while an import after
// other output is stashed in outOfOrderImports for insertion after that run.
func (iv *importedCssVisitor) VisitCssImport(node value.CssImport) (struct{}, error) {
	if iv.v.parent != iv.v.root {
		return struct{}{}, iv.v.addChild(node.(value.ModifiableCssNode))
	}
	if iv.v.endOfImports == len(iv.v.root.Children()) {
		if err := iv.v.addChild(node.(value.ModifiableCssNode)); err != nil {
			return struct{}{}, err
		}
		iv.v.endOfImports++
		return struct{}{}, nil
	} else {
		iv.v.outOfOrderImports = append(iv.v.outOfOrderImports, node.(*value.ModifiableCssImport))
		return struct{}{}, nil
	}
}

// VisitCssKeyframeBlock is unreachable: keyframe blocks are always nested
// inside keyframe rules, never visited at this level, so calling it is a bug.
func (iv *importedCssVisitor) VisitCssKeyframeBlock(_ value.CssKeyframeBlock) (struct{}, error) {
	return struct{}{}, fmt.Errorf("visitCssKeyframeBlock() should never be called")
}

// VisitCssMediaRule adds a media rule, bubbling through style rules and
// through media rules only when the node's queries were already merged with
// the ambient ones. Re-merging is a no-op for merged queries but would fail
// for unmerged ones, hence the upfront check.
func (iv *importedCssVisitor) VisitCssMediaRule(node value.CssMediaRule) (struct{}, error) {
	hasBeenMerged := iv.v.mediaQueries == nil ||
		iv.v.mergeMediaQueries(iv.v.mediaQueries, queriesToPtrs(node.Queries())) != nil
	return struct{}{}, iv.v.addChildThrough(node.(value.ModifiableCssNode), func(child value.CssNode) bool {
		if _, ok := child.(value.CssStyleRule); ok {
			return true
		}
		if hasBeenMerged {
			if _, ok := child.(value.CssMediaRule); ok {
				return true
			}
		}
		return false
	})
}

// VisitCssStyleRule adds a style rule, bubbling through style rules so nested
// imports land under the right parent.
func (iv *importedCssVisitor) VisitCssStyleRule(node value.CssStyleRule) (struct{}, error) {
	return struct{}{}, iv.v.addChildThrough(node.(value.ModifiableCssNode), func(child value.CssNode) bool {
		_, ok := child.(value.CssStyleRule)
		return ok
	})
}

// VisitCssStylesheet visits each child of an imported stylesheet in order,
// routing through the per-node methods above.
func (iv *importedCssVisitor) VisitCssStylesheet(node *value.CssStylesheet) (struct{}, error) {
	for _, child := range node.Children() {
		if _, err := child.AcceptVoid(iv); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

// VisitCssSupportsRule adds a supports rule, bubbling through style rules.
func (iv *importedCssVisitor) VisitCssSupportsRule(node value.CssSupportsRule) (struct{}, error) {
	return struct{}{}, iv.v.addChildThrough(node.(value.ModifiableCssNode), func(child value.CssNode) bool {
		_, ok := child.(value.CssStyleRule)
		return ok
	})
}
