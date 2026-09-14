// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (plain-CSS re-evaluation
// sections: visitCssStylesheet, visitCssStyleRule, visitCssMediaRule,
// visitCssDeclaration, visitCssAtRule, visitCssComment, visitCssImport,
// visitCssKeyframeBlock, visitCssSupportsRule)

import (
	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/unvendor"
	"github.com/bancek/go-sass/value"
)

// Plain-CSS re-evaluation.
//
// These methods run when evaluating already-compiled CSS: stylesheets pulled
// in via @import that themselves contain @use rules, and CSS included via the
// load-css() function. Such a module is first compiled to CSS (it must be
// evaluated exactly once and may be reused elsewhere), then that CSS is
// executed more or less as though it were Sass — it cannot be injected into
// the output tree as-is because the @import may be nested inside other rules.
// Matches Dart: the "Plain CSS" section intro.

// withStyleRule runs callback with rule as the current style rule, restoring
// the previous one afterwards.
//
// Matches Dart: _withStyleRule. Installs the rule that defines the current
// parent selector for the callback's duration.
func (v *EvaluateVisitor) withStyleRule(rule value.CssStyleRule, callback func() error) error {
	oldRule := v.styleRuleIgnoringAtRoot
	v.styleRuleIgnoringAtRoot = rule
	err := callback()
	v.styleRuleIgnoringAtRoot = oldRule
	return err
}

// --- CSS visitor methods ---
//
// These are used when re-evaluating already-compiled CSS (specifically from
// @import of files that contain @use). Each mirrors a Sass-statement visitor
// noted inline; keep the pair in sync.

// EvaluateVisitor implements CssVisitor[struct{}], so CSS node acceptance
// dispatches to these methods via node.AcceptVoid(v). The evaluator itself
// defines no Sass-AST visitor interfaces; these nine methods are the only
// visitor-shaped dispatch it carries, serving the CSS re-evaluation phase.

// VisitCssStylesheet re-evaluates an already-compiled CSS stylesheet node by
// node. The stylesheet itself contributes no output node.
//
// Matches Dart: visitCssStylesheet.
func (v *EvaluateVisitor) VisitCssStylesheet(node *value.CssStylesheet) (struct{}, error) {
	for _, child := range node.Children() {
		if _, err := child.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

// VisitCssStyleRule re-evaluates a frozen CSS style rule, merging its selector
// with the enclosing style rule and registering it with the extension store.
//
// Matches Dart: visitCssStyleRule. NOTE: this logic is largely duplicated in
// VisitStyleRule; mirror changes there. Style rules are rejected inside nested
// declarations and inside keyframe blocks. The selector merges with the
// enclosing style rule — merging unless there is no enclosing rule, skipping
// the merge when the enclosing rule came from plain CSS, and otherwise
// merging unless this rule came from plain CSS and contains a parent selector
// — via NestWithin with an implicit parent (unless @at-root excludes style
// rules) while preserving plain-CSS parent selectors. The new rule runs under
// withParent (style rules bubble) plus withStyleRule, atRootExcludingStyleRule
// clears for the duration and restores after, and when there was no enclosing
// style rule the parent's last child is marked as a group end.
func (v *EvaluateVisitor) VisitCssStyleRule(node value.CssStyleRule) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitStyleRule; mirror
	// changes there.

	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	if v.declarationName != "" {
		return struct{}{}, v.exception("Style rules may not be used within nested declarations.", &nodeSpan)
	} else if v.inKeyframes {
		if _, ok := v.parent.(*value.ModifiableCssKeyframeBlock); ok {
			return struct{}{}, v.exception("Style rules may not be used within keyframe blocks.", &nodeSpan)
		}
	}

	sr := v.styleRule()
	var merge bool
	if sr == nil {
		merge = true
	} else if sr.FromPlainCss() {
		// An enclosing plain-CSS rule already carries its full selector.
		merge = false
	} else {
		containsParent, err := node.Selector().ContainsParentSelector()
		if err != nil {
			return struct{}{}, err
		}
		merge = !node.FromPlainCss() || !containsParent
	}

	var originalSelector *value.SelectorList
	if merge {
		implicitParent := !v.atRootExcludingStyleRule
		var parent *value.SelectorList
		if sr != nil {
			parent = sr.OriginalSelector()
		}
		var err error
		originalSelector, err = node.Selector().NestWithin(parent, implicitParent, node.FromPlainCss())
		if err != nil {
			return struct{}{}, err
		}
	} else {
		originalSelector = node.Selector()
	}

	resultBox, err := v.extensionStore.AddSelector(originalSelector, v.mediaQueries)
	if err != nil {
		return struct{}{}, err
	}

	rule := value.NewModifiableCssStyleRule(resultBox, nodeSpan, originalSelector, node.FromPlainCss())

	oldAtRootExcludingStyleRule := v.atRootExcludingStyleRule
	v.atRootExcludingStyleRule = false

	// Only bubble through style rules when merging; a non-merged rule nests
	// in place.
	var through func(value.CssNode) bool
	if merge {
		through = func(n value.CssNode) bool {
			_, ok := n.(value.CssStyleRule)
			return ok
		}
	}

	err = v.withParent(rule, func() error {
		return v.withStyleRule(rule, func() error {
			for _, child := range node.Children() {
				if _, err := child.AcceptVoid(v); err != nil {
					return err
				}
			}
			return nil
		})
	}, through, new(bool))

	v.atRootExcludingStyleRule = oldAtRootExcludingStyleRule

	// A top-level rule group ends here: mark the parent's last child so
	// serialization groups selectors correctly.
	if v.styleRule() == nil {
		children := v.parent.Children()
		if len(children) > 0 {
			if lastChild, ok := children[len(children)-1].(value.ModifiableCssNode); ok {
				lastChild.SetIsGroupEnd(true)
			}
		}
	}

	return struct{}{}, err
}

// VisitCssMediaRule re-evaluates a frozen CSS @media rule, merging its queries
// with the enclosing media context and bubbling through style rules.
//
// Matches Dart: visitCssMediaRule. NOTE: this logic is largely duplicated in
// VisitMediaRule; mirror changes there. Rejected inside nested declarations.
// When plain-CSS nesting already applies, the rule nests as-is with no merging
// or bubbling. Otherwise its queries merge with the current media context —
// fully-merged-away queries emit nothing — and the merged set plus its sources
// run under withMediaQueries. The through predicate passes style rules plus
// media rules whose queries are all in the merged source set, so bubbling one
// query through another is safe. When a style rule is current, it is copied
// childless into the media rule first so declarations immediately inside
// @media have somewhere to go (for example "a {@media screen {b: c}}"
// produces "@media screen {a {b: c}}").
func (v *EvaluateVisitor) VisitCssMediaRule(node value.CssMediaRule) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitMediaRule; mirror
	// changes there.

	if v.declarationName != "" {
		span, err := node.Span()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, v.exception("Media rules may not be used within nested declarations.", &span)
	}

	// If the user has already opted into plain CSS nesting, don't bother with
	// any merging or bubbling; this rule is already only usable by browsers
	// that support nesting natively anyway.
	if v.hasCssNesting() {
		span, err := node.Span()
		if err != nil {
			return struct{}{}, err
		}
		rule, err := value.NewModifiableCssMediaRule(node.Queries(), span)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, v.withParent(rule, func() error {
			for _, child := range node.Children() {
				if _, err := child.AcceptVoid(v); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(bool))
	}

	var mergedQueries []*value.CssMediaQuery
	if v.mediaQueries != nil {
		mergedQueries = v.mergeMediaQueries(v.mediaQueries, queriesToPtrs(node.Queries()))
	}
	// Everything merged away: nothing to emit.
	if mergedQueries != nil && len(mergedQueries) == 0 {
		return struct{}{}, nil
	}

	var ruleQueries []value.CssMediaQuery
	var mergedSources *linkedhashmap.LinkedHashSet[*value.CssMediaQuery]
	if mergedQueries != nil {
		ruleQueries = queriesFromPtrs(mergedQueries)
		mergedSources = linkedhashmap.NewLinkedHashSet[*value.CssMediaQuery](value.MediaQueryHashEqual)
		for _, q := range v.mediaQueries {
			mergedSources.Add(q)
		}
		if v.mediaQuerySources != nil {
			for q := range v.mediaQuerySources.Keys() {
				mergedSources.Add(q)
			}
		}
		for _, q := range queriesToPtrs(node.Queries()) {
			mergedSources.Add(q)
		}
	} else {
		ruleQueries = node.Queries()
	}
	span, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	rule, err := value.NewModifiableCssMediaRule(ruleQueries, span)
	if err != nil {
		return struct{}{}, err
	}

	through := func(n value.CssNode) bool {
		if _, ok := n.(value.CssStyleRule); ok {
			return true
		}
		if mergedSources.Len() > 0 {
			if mr, ok := n.(value.CssMediaRule); ok {
				mrQueries := mr.Queries()
				allContained := true
				for i := range mrQueries {
					if !mergedSources.Contains(&mrQueries[i]) {
						allContained = false
						break
					}
				}
				if allContained {
					return true
				}
			}
		}
		return false
	}

	return struct{}{}, v.withParent(rule, func() error {
		return v.withMediaQueries(ruleQueries, mergedSources, func() error {
			if sr := v.styleRule(); sr != nil {
				// If in a style rule, copy it into the media query so that
				// declarations immediately inside @media have somewhere to go.
				// Dart: if (_styleRule case ModifiableCssStyleRule styleRule);
				// styleRule() only yields that concrete type when non-nil.
				modSr, ok := sr.(*value.ModifiableCssStyleRule)
				if ok {
					newParent, err := modSr.CopyWithoutChildren()
					if err != nil {
						return err
					}
					return v.withParent(newParent, func() error {
						for _, child := range node.Children() {
							if _, err := child.AcceptVoid(v); err != nil {
								return err
							}
						}
						return nil
					}, nil, new(bool))
				}
			}
			for _, child := range node.Children() {
				if _, err := child.AcceptVoid(v); err != nil {
					return err
				}
			}
			return nil
		})
	}, through, new(bool))
}

// VisitCssDeclaration copies a frozen CSS declaration into the modifiable
// output tree.
//
// Matches Dart: visitCssDeclaration. NOTE: this logic is largely duplicated
// in VisitDeclaration; mirror changes there. Copies the parent after a
// following sibling first, preserving the parsed-as-SassScript flag and the
// value span used for source maps.
func (v *EvaluateVisitor) VisitCssDeclaration(node value.CssDeclaration) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitDeclaration; mirror
	// changes there.
	if err := v.copyParentAfterSibling(); err != nil {
		return struct{}{}, err
	}
	span := node.ValueSpanForMap()
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	child, err := value.NewModifiableCssDeclaration(
		node.Name(),
		node.Value(),
		nodeSpan,
		node.ParsedAsSassScript(),
		&span,
	)
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, v.addChild(child)
}

// VisitCssAtRule re-evaluates a frozen CSS at-rule, bubbling it through
// style-rule parents.
//
// Matches Dart: visitCssAtRule. NOTE: this logic is largely duplicated in
// VisitAtRule; mirror changes there. Rejected inside nested declarations. A
// childless rule copies straight into the current parent. Otherwise the
// inKeyframes/inUnknownAtRule flags derive from the unvendored rule name and
// restore afterwards, and — unless plain-CSS nesting already applies, in which
// case no merging or bubbling happens — children run under withParent with a
// through predicate that only passes style rules. No unknown-at-rule check is
// needed here because the previous compilation already bubbled the at-rule to
// the root.
func (v *EvaluateVisitor) VisitCssAtRule(node value.CssAtRule) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitAtRule; mirror changes
	// there.

	if v.declarationName != "" {
		span, err := node.Span()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, v.exception("At-rules may not be used within nested declarations.", &span)
	}

	if node.IsChildless() {
		if err := v.copyParentAfterSibling(); err != nil {
			return struct{}{}, err
		}
		span, err := node.Span()
		if err != nil {
			return struct{}{}, err
		}
		if err := v.addChild(value.NewModifiableCssAtRule(node.Name(), span, node.IsChildless(), node.Value())); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}

	wasInKeyframes := v.inKeyframes
	wasInUnknownAtRule := v.inUnknownAtRule
	// Dart compares node.name.value directly without lowercasing; unvendor
	// handles vendor prefixes, so prefixed keyframes take the keyframes path.
	if unvendor.Unvendor(node.Name().Value) == "keyframes" {
		v.inKeyframes = true
	} else {
		v.inUnknownAtRule = true
	}

	// If the user has already opted into plain CSS nesting, don't bother with
	// any merging or bubbling; this rule is already only usable by browsers
	// that support nesting natively anyway.
	span, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	rule := value.NewModifiableCssAtRule(node.Name(), span, node.IsChildless(), node.Value())
	if v.hasCssNesting() {
		err := v.withParent(rule, func() error {
			for _, child := range node.Children() {
				if _, err := child.AcceptVoid(v); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(bool))
		v.inUnknownAtRule = wasInUnknownAtRule
		v.inKeyframes = wasInKeyframes
		return struct{}{}, err
	}

	err = v.withParent(rule, func() error {
		for _, child := range node.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return err
			}
		}
		return nil
	}, func(n value.CssNode) bool {
		_, ok := n.(value.CssStyleRule)
		return ok
	}, new(bool))

	v.inUnknownAtRule = wasInUnknownAtRule
	v.inKeyframes = wasInKeyframes
	return struct{}{}, err
}

// VisitCssComment copies a frozen CSS comment into the modifiable output tree.
//
// Matches Dart: visitCssComment. NOTE: this logic is largely duplicated in
// VisitLoudComment; mirror changes there. Comments may appear between CSS
// imports, so when the current parent is the root and endOfImports still
// points at the end of the root's children, it advances past the new comment.
func (v *EvaluateVisitor) VisitCssComment(node value.CssComment) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitLoudComment; mirror
	// changes there.

	// Comments are allowed to appear between CSS imports.
	if v.parent == v.root && v.endOfImports == v.root.ChildrenLen() {
		v.endOfImports++
	}

	if err := v.copyParentAfterSibling(); err != nil {
		return struct{}{}, err
	}
	comment, err := value.NewModifiableCssCommentFrom(node)
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, v.addChild(comment)
}

// VisitCssImport copies a frozen CSS @import into the modifiable output tree.
//
// Matches Dart: visitCssImport. NOTE: this logic is largely duplicated in the
// static-import path; mirror changes there. A nested import copies into the
// current parent (after copying the parent past any following sibling); a
// root-level import in order extends endOfImports, while an out-of-order
// root-level import defers into outOfOrderImports.
func (v *EvaluateVisitor) VisitCssImport(node value.CssImport) (struct{}, error) {
	// NOTE: this logic is largely duplicated in _visitStaticImport; mirror
	// changes there.

	modifiableNode, err := value.NewModifiableCssImportFrom(node)
	if err != nil {
		return struct{}{}, err
	}
	if v.parent != v.root {
		if err := v.copyParentAfterSibling(); err != nil {
			return struct{}{}, err
		}
		if err := v.parent.AddChild(modifiableNode); err != nil {
			return struct{}{}, err
		}
	} else if v.endOfImports == v.root.ChildrenLen() {
		if err := v.root.AddChild(modifiableNode); err != nil {
			return struct{}{}, err
		}
		v.endOfImports++
	} else {
		v.outOfOrderImports = append(v.outOfOrderImports, modifiableNode)
	}
	return struct{}{}, nil
}

// VisitCssKeyframeBlock re-evaluates a frozen keyframe block under a fresh
// modifiable parent.
//
// Matches Dart: visitCssKeyframeBlock. NOTE: this logic is largely duplicated
// in VisitStyleRule; mirror changes there. Children run under withParent with
// a through predicate that only passes style rules and no new environment
// scope for CSS children.
func (v *EvaluateVisitor) VisitCssKeyframeBlock(node value.CssKeyframeBlock) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitStyleRule; mirror
	// changes there.

	rule, err := value.NewModifiableCssKeyframeBlockFrom(node)
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, v.withParent(rule, func() error {
		for _, child := range node.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return err
			}
		}
		return nil
	}, func(n value.CssNode) bool { _, ok := n.(value.CssStyleRule); return ok }, new(bool))
}

// VisitCssSupportsRule re-evaluates a frozen CSS @supports rule, bubbling it
// through style rules.
//
// Matches Dart: visitCssSupportsRule. NOTE: this logic is largely duplicated
// in VisitSupportsRule; mirror changes there. Rejected inside nested
// declarations. Plain-CSS nesting skips merging and bubbling. Otherwise, when
// a style rule is current it is copied childless into the supports rule first
// so declarations immediately inside @supports have somewhere to go (for
// example "a {@supports (a: b) {b: c}}" produces "@supports (a: b) {a {b:
// c}}").
func (v *EvaluateVisitor) VisitCssSupportsRule(node value.CssSupportsRule) (struct{}, error) {
	// NOTE: this logic is largely duplicated in VisitSupportsRule; mirror
	// changes there.

	if v.declarationName != "" {
		span, err := node.Span()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, v.exception("Supports rules may not be used within nested declarations.", &span)
	}

	rule, err := value.NewModifiableCssSupportsRuleFrom(node)
	if err != nil {
		return struct{}{}, err
	}

	// If the user has already opted into plain CSS nesting, don't bother with
	// any merging or bubbling; this rule is already only usable by browsers
	// that support nesting natively anyway.
	if v.hasCssNesting() {
		return struct{}{}, v.withParent(rule, func() error {
			for _, child := range node.Children() {
				if _, err := child.AcceptVoid(v); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(bool))
	}

	return struct{}{}, v.withParent(rule, func() error {
		if sr := v.styleRule(); sr != nil {
			// If in a style rule, copy it into the supports rule so that
			// declarations immediately inside @supports have somewhere to go.
			// Dart: if (_styleRule case ModifiableCssStyleRule styleRule);
			// styleRule() only yields that concrete type when non-nil.
			modSr, ok := sr.(*value.ModifiableCssStyleRule)
			if ok {
				newParent, err := modSr.CopyWithoutChildren()
				if err != nil {
					return err
				}
				return v.withParent(newParent, func() error {
					for _, child := range node.Children() {
						if _, err := child.AcceptVoid(v); err != nil {
							return err
						}
					}
					return nil
				}, nil, nil)
			}
		}
		for _, child := range node.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return err
			}
		}
		return nil
	}, func(n value.CssNode) bool {
		_, ok := n.(value.CssStyleRule)
		return ok
	}, new(bool))
}
