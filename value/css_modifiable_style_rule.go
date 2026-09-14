// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/style_rule.dart

import (
	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssStyleRule is a modifiable version of CssStyleRule for use in
// the evaluation step.
//
// The selector lives in a Box supplied by the extension store so @extend can
// swap it in place; OriginalSelector preserves the pre-extension selector
// and FromPlainCss marks rules from plain-CSS inputs. Matches Dart:
// ModifiableCssStyleRule.
type ModifiableCssStyleRule struct {
	node          baseParentNode
	selectorBox   *box.Box[*SelectorList]
	innerOriginal *SelectorList
	fromPlainCss  bool
}

// NewModifiableCssStyleRuleFrom creates a modifiable style rule from a
// non-modifiable CssStyleRule.
//
// The selector is sealed into a fresh box; the original selector and
// plain-CSS flag carry over. Matches Dart: modifiable wrapper construction.
func NewModifiableCssStyleRuleFrom(node CssStyleRule) (*ModifiableCssStyleRule, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssStyleRule(
		box.NewModifiableBox(node.Selector()).Seal(),
		span,
		node.OriginalSelector(),
		node.FromPlainCss(),
	), nil
}

// NewModifiableCssStyleRule creates a modifiable style rule from a selector
// box, span, original selector, and plain-CSS flag.
//
// A nil original selector defaults to the box value. Matches Dart:
// ModifiableCssStyleRule constructor.
func NewModifiableCssStyleRule(
	selBox *box.Box[*SelectorList],
	span sasscommon.FileSpan,
	originalSelector *SelectorList,
	fromPlainCss bool,
) *ModifiableCssStyleRule {
	orig := originalSelector
	if orig == nil {
		orig = selBox.Value()
	}
	r := &ModifiableCssStyleRule{
		node:          baseParentNode{node: baseNode{span: span}},
		selectorBox:   selBox,
		innerOriginal: orig,
		fromPlainCss:  fromPlainCss,
	}
	r.node.self = r
	return r
}

// Selector returns the current (possibly extended) selector.
//
// Matches Dart: CssStyleRule.selector.
func (r *ModifiableCssStyleRule) Selector() *SelectorList { return r.selectorBox.Value() }

// OriginalSelector returns the pre-extension selector.
//
// Matches Dart: CssStyleRule.originalSelector.
func (r *ModifiableCssStyleRule) OriginalSelector() *SelectorList { return r.innerOriginal }

// FromPlainCss reports whether the rule came from a plain CSS stylesheet.
//
// This is internal metadata in Dart (@internal); it is exported here only
// for the evaluator and cloner.
func (r *ModifiableCssStyleRule) FromPlainCss() bool { return r.fromPlainCss }

// SetSelector swaps the selector through the extension-store box.
//
// Matches Dart: selector box update during @extend.
func (r *ModifiableCssStyleRule) SetSelector(sel *SelectorList) {
	r.selectorBox.SetValue(sel)
}

func (r *ModifiableCssStyleRule) Span() (sasscommon.FileSpan, error)  { return r.node.Span() }
func (r *ModifiableCssStyleRule) IsAstNode()                          {}
func (r *ModifiableCssStyleRule) IsCssNode()                          {}
func (r *ModifiableCssStyleRule) IsGroupEnd() bool                    { return r.node.IsGroupEnd() }
func (r *ModifiableCssStyleRule) SetIsGroupEnd(v bool)                { r.node.SetIsGroupEnd(v) }
func (r *ModifiableCssStyleRule) setParent(p ModifiableCssParentNode) { r.node.setParent(p) }
func (r *ModifiableCssStyleRule) setIndexInParent(i int)              { r.node.setIndexInParent(i) }
func (r *ModifiableCssStyleRule) indexInParent() int                  { return r.node.indexInParent() }
func (r *ModifiableCssStyleRule) Parent() CssParentNode               { return r.node.Parent() }
func (r *ModifiableCssStyleRule) HasFollowingSibling() bool           { return r.node.HasFollowingSibling() }
func (r *ModifiableCssStyleRule) Remove() error                       { return r.node.Remove() }
func (r *ModifiableCssStyleRule) Children() []CssNode                 { return r.node.Children() }
func (r *ModifiableCssStyleRule) ChildrenLen() int                    { return r.node.ChildrenLen() }
func (r *ModifiableCssStyleRule) LastChild() ModifiableCssNode        { return r.node.LastChild() }
func (r *ModifiableCssStyleRule) IsChildless() bool                   { return r.node.IsChildless() }
func (r *ModifiableCssStyleRule) IsCssParentNode()                    {}
func (r *ModifiableCssStyleRule) AddChild(child ModifiableCssNode) error {
	return r.node.AddChild(child)
}
func (r *ModifiableCssStyleRule) ClearChildren()          { r.node.ClearChildren() }
func (r *ModifiableCssStyleRule) removeChildAt(index int) { r.node.removeChildAt(index) }

// IsInvisible returns whether this style rule is invisible in the output.
//
// Matches Dart: CssStyleRule.isInvisible
func (r *ModifiableCssStyleRule) IsInvisible() bool {
	if r.selectorBox.Value().IsInvisible() {
		return true
	}
	for _, child := range r.Children() {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

// Matches Dart: CssStyleRule.isInvisibleOtherThanBogusCombinators
func (r *ModifiableCssStyleRule) IsInvisibleOtherThanBogusCombinators() bool {
	if r.selectorBox.Value().IsInvisibleOtherThanBogusCombinators() {
		return true
	}
	for _, child := range r.Children() {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

// IsInvisibleHidingComments returns whether this style rule should be
// considered invisible even when comments are not hidden.
//
// Matches Dart: CssStyleRule.isInvisibleHidingComments
func (r *ModifiableCssStyleRule) IsInvisibleHidingComments() bool {
	if r.selectorBox.Value().IsInvisible() {
		return true
	}
	for _, child := range r.Children() {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

// EqualsIgnoringChildren reports whether both rules share an equal selector,
// ignoring children.
//
// Matches Dart: ModifiableCssStyleRule.equalsIgnoringChildren.
func (r *ModifiableCssStyleRule) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	other2, ok := any(other).(*ModifiableCssStyleRule)
	if !ok {
		return false, nil
	}
	return EqualSelectors(r.selectorBox.Value(), other2.selectorBox.Value()), nil
}

func (r *ModifiableCssStyleRule) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssStyleRule(r)
	return err
}

func (r *ModifiableCssStyleRule) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssStyleRule(r)
}

func (r *ModifiableCssStyleRule) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssStyleRule(r)
}

func (r *ModifiableCssStyleRule) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssStyleRule(r)
}

// CopyWithoutChildren returns a shallow copy with no children, sharing the
// selector box but resetting FromPlainCss to false (Dart's constructor
// default), used when copying rules into @media/at-rule bodies.
//
// Matches Dart: ModifiableCssStyleRule.copyWithoutChildren.
func (r *ModifiableCssStyleRule) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return newModifiableCssStyleRuleWithBox(r.selectorBox, r.node.node.span, r.innerOriginal, false), nil
}

func newModifiableCssStyleRuleWithBox(
	selectorBox *box.Box[*SelectorList],
	span sasscommon.FileSpan,
	originalSelector *SelectorList,
	fromPlainCss bool,
) *ModifiableCssStyleRule {
	orig := originalSelector
	if orig == nil {
		orig = selectorBox.Value()
	}
	r := &ModifiableCssStyleRule{
		node:          baseParentNode{node: baseNode{span: span}},
		selectorBox:   selectorBox,
		innerOriginal: orig,
		fromPlainCss:  fromPlainCss,
	}
	r.node.self = r
	return r
}
