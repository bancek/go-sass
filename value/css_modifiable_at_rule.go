// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/at_rule.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssAtRule is a modifiable version of CssAtRule for use in the
// evaluation step.
//
// Unknown at-rules carry a name, optional value, and the childless flag
// distinguishing `@foo;` from `@foo {}`. Matches Dart: ModifiableCssAtRule.
type ModifiableCssAtRule struct {
	node       baseParentNode
	innerName  sasscommon.CssValue[string]
	innerValue *sasscommon.CssValue[string]
	childless  bool
}

// NewModifiableCssAtRule creates a modifiable unknown at-rule.
//
// Matches Dart: ModifiableCssAtRule constructor.
func NewModifiableCssAtRule(
	name sasscommon.CssValue[string],
	span sasscommon.FileSpan,
	childless bool,
	value *sasscommon.CssValue[string],
) *ModifiableCssAtRule {
	r := &ModifiableCssAtRule{
		node:       baseParentNode{node: baseNode{span: span}},
		innerName:  name,
		innerValue: value,
		childless:  childless,
	}
	r.node.self = r
	return r
}

// Name returns the at-rule name.
//
// Matches Dart: CssAtRule.name.
func (r *ModifiableCssAtRule) Name() sasscommon.CssValue[string] { return r.innerName }

// Value returns the at-rule value, or nil when absent.
//
// Matches Dart: CssAtRule.value.
func (r *ModifiableCssAtRule) Value() *sasscommon.CssValue[string] { return r.innerValue }

// IsChildless reports whether the rule ends with ";" rather than a block.
//
// A childless rule implies no children, but an empty block is not childless.
// Matches Dart: CssAtRule.isChildless.
func (r *ModifiableCssAtRule) IsChildless() bool { return r.childless }

func (r *ModifiableCssAtRule) Span() (sasscommon.FileSpan, error) { return r.node.Span() }
func (r *ModifiableCssAtRule) IsAstNode()                         {}
func (r *ModifiableCssAtRule) IsCssNode()                         {}
func (r *ModifiableCssAtRule) IsGroupEnd() bool                   { return r.node.IsGroupEnd() }
func (r *ModifiableCssAtRule) SetIsGroupEnd(v bool)               { r.node.SetIsGroupEnd(v) }
func (r *ModifiableCssAtRule) IsInvisible() bool                  { return r.node.IsInvisible() }
func (r *ModifiableCssAtRule) IsInvisibleHidingComments() bool {
	return r.node.IsInvisibleHidingComments()
}
func (r *ModifiableCssAtRule) IsInvisibleOtherThanBogusCombinators() bool {
	return r.node.IsInvisibleOtherThanBogusCombinators()
}
func (r *ModifiableCssAtRule) setParent(p ModifiableCssParentNode) { r.node.setParent(p) }
func (r *ModifiableCssAtRule) setIndexInParent(i int)              { r.node.setIndexInParent(i) }
func (r *ModifiableCssAtRule) indexInParent() int                  { return r.node.indexInParent() }
func (r *ModifiableCssAtRule) Parent() CssParentNode               { return r.node.Parent() }
func (r *ModifiableCssAtRule) HasFollowingSibling() bool           { return r.node.HasFollowingSibling() }
func (r *ModifiableCssAtRule) Remove() error                       { return r.node.Remove() }
func (r *ModifiableCssAtRule) Children() []CssNode                 { return r.node.Children() }
func (r *ModifiableCssAtRule) ChildrenLen() int                    { return r.node.ChildrenLen() }
func (r *ModifiableCssAtRule) LastChild() ModifiableCssNode        { return r.node.LastChild() }
func (r *ModifiableCssAtRule) IsCssParentNode()                    {}
func (r *ModifiableCssAtRule) ClearChildren()                      { r.node.ClearChildren() }
func (r *ModifiableCssAtRule) removeChildAt(index int)             { r.node.removeChildAt(index) }

// AddChild appends a child, rejecting children for childless rules.
//
// Matches Dart: childless guard in at-rule evaluation.
func (r *ModifiableCssAtRule) AddChild(child ModifiableCssNode) error {
	if r.childless {
		return &sasscommon.ArgumentError{Message: "Cannot add a child to a childless at-rule."}
	}
	return r.node.AddChild(child)
}

// EqualsIgnoringChildren reports whether name, value, and childless all match.
//
// Matches Dart: ModifiableCssAtRule.equalsIgnoringChildren.
func (r *ModifiableCssAtRule) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	other2, ok := any(other).(*ModifiableCssAtRule)
	if !ok {
		return false, nil
	}
	return r.innerName.Equal(other2.innerName) &&
		(r.innerValue == nil && other2.innerValue == nil ||
			r.innerValue != nil && other2.innerValue != nil && r.innerValue.Equal(*other2.innerValue)) &&
		r.childless == other2.childless, nil
}

func (r *ModifiableCssAtRule) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssAtRule(r)
	return err
}

func (r *ModifiableCssAtRule) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssAtRule(r)
}

func (r *ModifiableCssAtRule) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssAtRule(r)
}

func (r *ModifiableCssAtRule) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssAtRule(r)
}

// CopyWithoutChildren returns a copy with the same name/value/childless and
// no children (shallow copy).
//
// Matches Dart: ModifiableCssAtRule.copyWithoutChildren.
func (r *ModifiableCssAtRule) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return NewModifiableCssAtRule(r.innerName, r.node.node.span, r.childless, r.innerValue), nil
}
