// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/modifiable/node.dart

// ModifiableCssNode is the interface for all modifiable CSS AST nodes.
//
// Matches Dart: ModifiableCssNode (extends CssNode). Almost all CSS nodes
// are these modifiable wrappers under the covers during evaluation; the
// frozen CssXxx types are used elsewhere to enforce that mutation stays in
// the evaluation step.
type ModifiableCssNode interface {
	CssNode
	SetIsGroupEnd(bool)
	AcceptModifiableVoid(visitor ModifiableCssVisitor) error
	AcceptCloneModifiableCssNode(visitor CloneCssVisitor) (ModifiableCssNode, error)
	Remove() error
	HasFollowingSibling() bool
	setParent(p ModifiableCssParentNode)
	setIndexInParent(i int)
	indexInParent() int
}

// ModifiableCssParentNode is the interface for modifiable CSS nodes that
// can have children.
//
// Matches Dart: ModifiableCssParentNode. Children are owned by the parent;
// EqualsIgnoringChildren compares distinguishing fields only, and
// CopyWithoutChildren clones those fields with an empty child list (a
// shallow copy: shared modifiable parts stay shared).
type ModifiableCssParentNode interface {
	ModifiableCssNode
	Children() []CssNode
	IsChildless() bool
	IsCssParentNode()
	AddChild(child ModifiableCssNode) error
	ClearChildren()
	removeChildAt(index int)
	EqualsIgnoringChildren(other ModifiableCssNode) (bool, error)
	CopyWithoutChildren() (ModifiableCssParentNode, error)
	// ChildrenLen returns the number of children without building the
	// []CssNode cache (mirrors Rust ModifiableCssNode::children_len).
	ChildrenLen() int
	// LastChild returns the last child without building the []CssNode cache,
	// or nil if empty (mirrors Rust ModifiableCssNode::last_child).
	LastChild() ModifiableCssNode
}

// baseNode provides the shared implementation for all modifiable CSS nodes.
//
// It holds the span, parent link, index-in-parent for O(1) removal, and the
// group-end flag for flattened nested trees. Matches Dart:
// ModifiableCssNode fields (_parent, _indexInParent, isGroupEnd).

// baseNode provides the shared implementation for all modifiable CSS nodes.
type baseNode struct {
	span       sasscommon.FileSpan
	parent     ModifiableCssParentNode
	idx        int
	isGroupEnd bool
}

func (n *baseNode) Span() (sasscommon.FileSpan, error)         { return n.span, nil }
func (n *baseNode) IsAstNode()                                 {}
func (n *baseNode) IsCssNode()                                 {}
func (n *baseNode) IsGroupEnd() bool                           { return n.isGroupEnd }
func (n *baseNode) SetIsGroupEnd(v bool)                       { n.isGroupEnd = v }
func (n *baseNode) IsInvisible() bool                          { return false }
func (n *baseNode) IsInvisibleHidingComments() bool            { return false }
func (n *baseNode) IsInvisibleOtherThanBogusCombinators() bool { return false }
func (n *baseNode) setParent(p ModifiableCssParentNode)        { n.parent = p }
func (n *baseNode) setIndexInParent(i int)                     { n.idx = i }
func (n *baseNode) indexInParent() int                         { return n.idx }
func (n *baseNode) Parent() CssParentNode                      { return n.parent }
func (n *baseNode) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	panic("AcceptModifiableVoid must be overridden by concrete types")
}

func (n *baseNode) AcceptCloneModifiableCssNode(visitor CloneCssVisitor) (ModifiableCssNode, error) {
	panic("AcceptCloneModifiableCssNode must be overridden by concrete types")
}

func (n *baseNode) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	panic("AcceptVoid must be overridden by concrete types")
}

func (n *baseNode) AcceptBool(v CssVisitor[bool]) (bool, error) {
	panic("AcceptBool must be overridden by concrete types")
}

// HasFollowingSibling reports whether a visible sibling follows this node,
// used to decide trailing separators during serialization.
//
// Matches Dart: ModifiableCssNode.hasFollowingSibling.
func (n *baseNode) HasFollowingSibling() bool {
	if n.parent == nil {
		return false
	}
	siblings := n.parent.Children()
	for i := n.idx + 1; i < len(siblings); i++ {
		if !siblings[i].IsInvisible() {
			return true
		}
	}
	return false
}

// Remove detaches this node from its parent, reindexing later siblings.
//
// It errors when the node has no parent. Matches Dart:
// ModifiableCssNode.remove.
func (n *baseNode) Remove() error {
	if n.parent == nil {
		return &sasscommon.StateError{Message: "Can't remove a node without a parent."}
	}
	i := n.idx
	n.parent.removeChildAt(i)
	n.parent = nil
	return nil
}

// baseParentNode provides the shared implementation for modifiable parent nodes.
//
// Children are stored as modifiable nodes with a cached CssNode view that is
// invalidated on AddChild/ClearChildren/removeChildAt. Matches Dart:
// ModifiableCssParentNode (_children, children view, addChild/clearChildren).
type baseParentNode struct {
	node       baseNode                // named field (was anonymous embed)
	self       ModifiableCssParentNode // points to the outer concrete type
	childNodes []ModifiableCssNode
	childCache []CssNode // nil = dirty, rebuilt on Children()
}

func (p *baseParentNode) Span() (sasscommon.FileSpan, error) { return p.node.Span() }
func (p *baseParentNode) IsAstNode()                         {}
func (p *baseParentNode) IsCssNode()                         {}
func (p *baseParentNode) IsGroupEnd() bool                   { return p.node.IsGroupEnd() }
func (p *baseParentNode) SetIsGroupEnd(v bool)               { p.node.SetIsGroupEnd(v) }
func (p *baseParentNode) IsInvisible() bool                  { return p.node.IsInvisible() }
func (p *baseParentNode) IsInvisibleHidingComments() bool    { return p.node.IsInvisibleHidingComments() }
func (p *baseParentNode) IsInvisibleOtherThanBogusCombinators() bool {
	return p.node.IsInvisibleOtherThanBogusCombinators()
}
func (p *baseParentNode) setParent(parent ModifiableCssParentNode) { p.node.setParent(parent) }
func (p *baseParentNode) setIndexInParent(i int)                   { p.node.setIndexInParent(i) }
func (p *baseParentNode) indexInParent() int                       { return p.node.indexInParent() }
func (p *baseParentNode) Parent() CssParentNode                    { return p.node.Parent() }
func (p *baseParentNode) AcceptModifiableVoid(v ModifiableCssVisitor) error {
	return p.node.AcceptModifiableVoid(v)
}
func (p *baseParentNode) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return p.node.AcceptCloneModifiableCssNode(v)
}
func (p *baseParentNode) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return p.node.AcceptVoid(v)
}
func (p *baseParentNode) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return p.node.AcceptBool(v)
}
func (p *baseParentNode) HasFollowingSibling() bool { return p.node.HasFollowingSibling() }
func (p *baseParentNode) Remove() error             { return p.node.Remove() }

// Children returns the cached child view, rebuilding it after mutations.
//
// Matches Dart: ModifiableCssParentNode.children.
func (p *baseParentNode) Children() []CssNode {
	if p.childCache == nil {
		result := make([]CssNode, len(p.childNodes))
		for i, child := range p.childNodes {
			result[i] = child
		}
		p.childCache = result
	}
	return p.childCache
}

func (p *baseParentNode) ChildrenLen() int { return len(p.childNodes) }

func (p *baseParentNode) LastChild() ModifiableCssNode {
	if len(p.childNodes) == 0 {
		return nil
	}
	return p.childNodes[len(p.childNodes)-1]
}

func (p *baseParentNode) IsChildless() bool { return false }
func (p *baseParentNode) IsCssParentNode()  {}
func (p *baseParentNode) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	panic("EqualsIgnoringChildren must be overridden by concrete types")
}
func (p *baseParentNode) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	panic("CopyWithoutChildren must be overridden by concrete types")
}

// AddChild appends child, wiring its parent link and index and dropping the
// child cache.
//
// Matches Dart: ModifiableCssParentNode.addChild.
func (p *baseParentNode) AddChild(child ModifiableCssNode) error {
	parent := ModifiableCssParentNode(p.self)
	if parent == nil {
		parent = p
	}
	child.setParent(parent)
	child.setIndexInParent(len(p.childNodes))
	p.childNodes = append(p.childNodes, child)
	p.childCache = nil
	return nil
}

// ClearChildren detaches all children in both directions and empties the list.
//
// Matches Dart: ModifiableCssParentNode.clearChildren.
func (p *baseParentNode) ClearChildren() {
	for _, child := range p.childNodes {
		child.setParent(nil)
		child.setIndexInParent(0)
	}
	p.childNodes = nil
	p.childCache = nil
}

// removeChildAt splices out the child at index and reindexes its successors.
//
// Matches Dart: parent._children.removeAt plus index fixup in remove().
func (p *baseParentNode) removeChildAt(index int) {
	p.childNodes = append(p.childNodes[:index], p.childNodes[index+1:]...)
	for i := index; i < len(p.childNodes); i++ {
		p.childNodes[i].setIndexInParent(i)
	}
	p.childCache = nil
}
