// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/keyframe_block.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssKeyframeBlock is a modifiable version of CssKeyframeBlock for
// use in the evaluation step.
//
// Matches Dart: ModifiableCssKeyframeBlock.
type ModifiableCssKeyframeBlock struct {
	node     baseParentNode
	selector sasscommon.CssValue[[]string]
}

// NewModifiableCssKeyframeBlockFrom creates a modifiable keyframe block from a
// non-modifiable CssKeyframeBlock.
//
// Matches Dart: modifiable wrapper construction.
func NewModifiableCssKeyframeBlockFrom(node CssKeyframeBlock) (*ModifiableCssKeyframeBlock, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssKeyframeBlock(node.Selector(), span), nil
}

// NewModifiableCssKeyframeBlock creates a modifiable keyframe block (for
// example "10%") with the given selector.
//
// Matches Dart: ModifiableCssKeyframeBlock constructor.
func NewModifiableCssKeyframeBlock(
	selector sasscommon.CssValue[[]string],
	span sasscommon.FileSpan,
) *ModifiableCssKeyframeBlock {
	b := &ModifiableCssKeyframeBlock{
		node:     baseParentNode{node: baseNode{span: span}},
		selector: selector,
	}
	b.node.self = b
	return b
}

// Selector returns the keyframe selector (offsets like "from", "50%").
//
// Matches Dart: CssKeyframeBlock.selector.
func (b *ModifiableCssKeyframeBlock) Selector() sasscommon.CssValue[[]string] { return b.selector }

func (b *ModifiableCssKeyframeBlock) Span() (sasscommon.FileSpan, error)  { return b.node.Span() }
func (b *ModifiableCssKeyframeBlock) IsAstNode()                          {}
func (b *ModifiableCssKeyframeBlock) IsCssNode()                          {}
func (b *ModifiableCssKeyframeBlock) IsGroupEnd() bool                    { return b.node.IsGroupEnd() }
func (b *ModifiableCssKeyframeBlock) SetIsGroupEnd(v bool)                { b.node.SetIsGroupEnd(v) }
func (b *ModifiableCssKeyframeBlock) setParent(p ModifiableCssParentNode) { b.node.setParent(p) }
func (b *ModifiableCssKeyframeBlock) setIndexInParent(i int)              { b.node.setIndexInParent(i) }
func (b *ModifiableCssKeyframeBlock) indexInParent() int                  { return b.node.indexInParent() }
func (b *ModifiableCssKeyframeBlock) Parent() CssParentNode               { return b.node.Parent() }
func (b *ModifiableCssKeyframeBlock) HasFollowingSibling() bool           { return b.node.HasFollowingSibling() }
func (b *ModifiableCssKeyframeBlock) Remove() error                       { return b.node.Remove() }
func (b *ModifiableCssKeyframeBlock) Children() []CssNode                 { return b.node.Children() }
func (b *ModifiableCssKeyframeBlock) ChildrenLen() int                    { return b.node.ChildrenLen() }
func (b *ModifiableCssKeyframeBlock) LastChild() ModifiableCssNode        { return b.node.LastChild() }
func (b *ModifiableCssKeyframeBlock) IsChildless() bool                   { return b.node.IsChildless() }
func (b *ModifiableCssKeyframeBlock) IsCssParentNode()                    {}
func (b *ModifiableCssKeyframeBlock) AddChild(child ModifiableCssNode) error {
	return b.node.AddChild(child)
}
func (b *ModifiableCssKeyframeBlock) ClearChildren()          { b.node.ClearChildren() }
func (b *ModifiableCssKeyframeBlock) removeChildAt(index int) { b.node.removeChildAt(index) }

func (b *ModifiableCssKeyframeBlock) IsInvisible() bool {
	for _, child := range b.Children() {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

func (b *ModifiableCssKeyframeBlock) IsInvisibleOtherThanBogusCombinators() bool {
	for _, child := range b.Children() {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

func (b *ModifiableCssKeyframeBlock) IsInvisibleHidingComments() bool {
	for _, child := range b.Children() {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

// EqualsIgnoringChildren reports whether both blocks share an equal selector
// list, ignoring children.
//
// Matches Dart: ModifiableCssKeyframeBlock.equalsIgnoringChildren.
func (b *ModifiableCssKeyframeBlock) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	other2, ok := any(other).(*ModifiableCssKeyframeBlock)
	if !ok {
		return false, nil
	}
	// listEquals from Dart's package:collection — structural list comparison
	return b.selector.Equal(other2.selector), nil
}

func (b *ModifiableCssKeyframeBlock) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssKeyframeBlock(b)
	return err
}

func (b *ModifiableCssKeyframeBlock) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssKeyframeBlock(b)
}

func (b *ModifiableCssKeyframeBlock) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssKeyframeBlock(b)
}

func (b *ModifiableCssKeyframeBlock) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssKeyframeBlock(b)
}

// CopyWithoutChildren returns a copy with the same selector and no children.
//
// Matches Dart: ModifiableCssKeyframeBlock.copyWithoutChildren.
func (b *ModifiableCssKeyframeBlock) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return NewModifiableCssKeyframeBlock(b.selector, b.node.node.span), nil
}
