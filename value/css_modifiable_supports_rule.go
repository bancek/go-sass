// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/supports_rule.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssSupportsRule is a modifiable version of CssSupportsRule for use
// in the evaluation step.
//
// Matches Dart: ModifiableCssSupportsRule.
type ModifiableCssSupportsRule struct {
	node      baseParentNode
	condition sasscommon.CssValue[string]
}

// NewModifiableCssSupportsRuleFrom creates a modifiable supports rule from a
// non-modifiable CssSupportsRule.
//
// Matches Dart: modifiable wrapper construction.
func NewModifiableCssSupportsRuleFrom(node CssSupportsRule) (*ModifiableCssSupportsRule, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssSupportsRule(node.Condition(), span), nil
}

// NewModifiableCssSupportsRule creates a modifiable @supports rule with the
// given condition.
//
// Matches Dart: ModifiableCssSupportsRule constructor.
func NewModifiableCssSupportsRule(
	condition sasscommon.CssValue[string],
	span sasscommon.FileSpan,
) *ModifiableCssSupportsRule {
	r := &ModifiableCssSupportsRule{
		node:      baseParentNode{node: baseNode{span: span}},
		condition: condition,
	}
	r.node.self = r
	return r
}

// Condition returns the @supports condition.
//
// Matches Dart: CssSupportsRule.condition.
func (r *ModifiableCssSupportsRule) Condition() sasscommon.CssValue[string] { return r.condition }

func (r *ModifiableCssSupportsRule) Span() (sasscommon.FileSpan, error)  { return r.node.Span() }
func (r *ModifiableCssSupportsRule) IsAstNode()                          {}
func (r *ModifiableCssSupportsRule) IsCssNode()                          {}
func (r *ModifiableCssSupportsRule) IsGroupEnd() bool                    { return r.node.IsGroupEnd() }
func (r *ModifiableCssSupportsRule) SetIsGroupEnd(v bool)                { r.node.SetIsGroupEnd(v) }
func (r *ModifiableCssSupportsRule) setParent(p ModifiableCssParentNode) { r.node.setParent(p) }
func (r *ModifiableCssSupportsRule) setIndexInParent(i int)              { r.node.setIndexInParent(i) }
func (r *ModifiableCssSupportsRule) indexInParent() int                  { return r.node.indexInParent() }
func (r *ModifiableCssSupportsRule) Parent() CssParentNode               { return r.node.Parent() }
func (r *ModifiableCssSupportsRule) HasFollowingSibling() bool           { return r.node.HasFollowingSibling() }
func (r *ModifiableCssSupportsRule) Remove() error                       { return r.node.Remove() }
func (r *ModifiableCssSupportsRule) Children() []CssNode                 { return r.node.Children() }
func (r *ModifiableCssSupportsRule) ChildrenLen() int                    { return r.node.ChildrenLen() }
func (r *ModifiableCssSupportsRule) LastChild() ModifiableCssNode        { return r.node.LastChild() }
func (r *ModifiableCssSupportsRule) IsChildless() bool                   { return r.node.IsChildless() }
func (r *ModifiableCssSupportsRule) IsCssParentNode()                    {}
func (r *ModifiableCssSupportsRule) AddChild(child ModifiableCssNode) error {
	return r.node.AddChild(child)
}
func (r *ModifiableCssSupportsRule) ClearChildren()          { r.node.ClearChildren() }
func (r *ModifiableCssSupportsRule) removeChildAt(index int) { r.node.removeChildAt(index) }

func (r *ModifiableCssSupportsRule) IsInvisible() bool {
	for _, child := range r.Children() {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

func (r *ModifiableCssSupportsRule) IsInvisibleOtherThanBogusCombinators() bool {
	for _, child := range r.Children() {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

func (r *ModifiableCssSupportsRule) IsInvisibleHidingComments() bool {
	for _, child := range r.Children() {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

// EqualsIgnoringChildren reports whether both rules share an equal condition.
//
// Matches Dart: ModifiableCssSupportsRule.equalsIgnoringChildren.
func (r *ModifiableCssSupportsRule) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	other2, ok := any(other).(*ModifiableCssSupportsRule)
	if !ok {
		return false, nil
	}
	return r.condition.Equal(other2.condition), nil
}

func (r *ModifiableCssSupportsRule) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssSupportsRule(r)
	return err
}

func (r *ModifiableCssSupportsRule) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssSupportsRule(r)
}

func (r *ModifiableCssSupportsRule) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssSupportsRule(r)
}

func (r *ModifiableCssSupportsRule) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssSupportsRule(r)
}

// CopyWithoutChildren returns a copy with the same condition and no children.
//
// Matches Dart: ModifiableCssSupportsRule.copyWithoutChildren.
func (r *ModifiableCssSupportsRule) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return NewModifiableCssSupportsRule(r.condition, r.node.node.span), nil
}
