// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"github.com/bancek/go-sass/sasscommon"
)

// dart-source: lib/src/ast/css/modifiable/media_rule.dart

// ModifiableCssMediaRule is a modifiable version of CssMediaRule for use in
// the evaluation step.
//
// Matches Dart: ModifiableCssMediaRule.
type ModifiableCssMediaRule struct {
	node    baseParentNode
	queries []CssMediaQuery
}

// NewModifiableCssMediaRuleFrom creates a modifiable media rule from a
// non-modifiable CssMediaRule.
//
// Matches Dart: modifiable wrapper construction.
func NewModifiableCssMediaRuleFrom(node CssMediaRule) (*ModifiableCssMediaRule, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssMediaRule(node.Queries(), span)
}

// NewModifiableCssMediaRule creates a modifiable @media rule; queries must
// be non-empty.
//
// Matches Dart: ModifiableCssMediaRule constructor (queries argument).
func NewModifiableCssMediaRule(
	queries []CssMediaQuery,
	span sasscommon.FileSpan,
) (*ModifiableCssMediaRule, error) {
	if len(queries) == 0 {
		return nil, &sasscommon.ArgumentError{Name: "queries", Message: "may not be empty."}
	}
	q := make([]CssMediaQuery, len(queries))
	copy(q, queries)
	r := &ModifiableCssMediaRule{
		node:    baseParentNode{node: baseNode{span: span}},
		queries: q,
	}
	r.node.self = r
	return r, nil
}

// Queries returns the rule's media queries.
//
// Matches Dart: CssMediaRule.queries.
func (r *ModifiableCssMediaRule) Queries() []CssMediaQuery { return r.queries }

func (r *ModifiableCssMediaRule) Span() (sasscommon.FileSpan, error)  { return r.node.Span() }
func (r *ModifiableCssMediaRule) IsAstNode()                          {}
func (r *ModifiableCssMediaRule) IsCssNode()                          {}
func (r *ModifiableCssMediaRule) IsGroupEnd() bool                    { return r.node.IsGroupEnd() }
func (r *ModifiableCssMediaRule) SetIsGroupEnd(v bool)                { r.node.SetIsGroupEnd(v) }
func (r *ModifiableCssMediaRule) setParent(p ModifiableCssParentNode) { r.node.setParent(p) }
func (r *ModifiableCssMediaRule) setIndexInParent(i int)              { r.node.setIndexInParent(i) }
func (r *ModifiableCssMediaRule) indexInParent() int                  { return r.node.indexInParent() }
func (r *ModifiableCssMediaRule) Parent() CssParentNode               { return r.node.Parent() }
func (r *ModifiableCssMediaRule) HasFollowingSibling() bool           { return r.node.HasFollowingSibling() }
func (r *ModifiableCssMediaRule) Remove() error                       { return r.node.Remove() }
func (r *ModifiableCssMediaRule) Children() []CssNode                 { return r.node.Children() }
func (r *ModifiableCssMediaRule) ChildrenLen() int                    { return r.node.ChildrenLen() }
func (r *ModifiableCssMediaRule) LastChild() ModifiableCssNode        { return r.node.LastChild() }
func (r *ModifiableCssMediaRule) IsChildless() bool                   { return r.node.IsChildless() }
func (r *ModifiableCssMediaRule) IsCssParentNode()                    {}
func (r *ModifiableCssMediaRule) AddChild(child ModifiableCssNode) error {
	return r.node.AddChild(child)
}
func (r *ModifiableCssMediaRule) ClearChildren()          { r.node.ClearChildren() }
func (r *ModifiableCssMediaRule) removeChildAt(index int) { r.node.removeChildAt(index) }

// IsInvisible returns true if all children are invisible.
//
// A media rule contributes no output when every child is invisible.
// Matches Dart: CssMediaRule invisibility via EveryCssVisitor.
func (r *ModifiableCssMediaRule) IsInvisible() bool {
	for _, child := range r.Children() {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

func (r *ModifiableCssMediaRule) IsInvisibleOtherThanBogusCombinators() bool {
	for _, child := range r.Children() {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

func (r *ModifiableCssMediaRule) IsInvisibleHidingComments() bool {
	for _, child := range r.Children() {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

// EqualsIgnoringChildren reports whether both rules hold equal query lists.
//
// Matches Dart: ModifiableCssMediaRule.equalsIgnoringChildren.
func (r *ModifiableCssMediaRule) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	other2, ok := any(other).(*ModifiableCssMediaRule)
	if !ok {
		return false, nil
	}
	if len(r.queries) != len(other2.queries) {
		return false, nil
	}
	for i := range r.queries {
		a, b := &r.queries[i], &other2.queries[i]
		if !MediaQueriesEqual(a, b) {
			return false, nil
		}
	}
	return true, nil
}

func (r *ModifiableCssMediaRule) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssMediaRule(r)
	return err
}

func (r *ModifiableCssMediaRule) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssMediaRule(r)
}

func (r *ModifiableCssMediaRule) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssMediaRule(r)
}

func (r *ModifiableCssMediaRule) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssMediaRule(r)
}

// CopyWithoutChildren returns a copy sharing the query list shape with no children.
//
// Matches Dart: ModifiableCssMediaRule.copyWithoutChildren (shallow copy).
func (r *ModifiableCssMediaRule) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return NewModifiableCssMediaRule(r.queries, r.node.node.span)
}
