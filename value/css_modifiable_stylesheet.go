// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/modifiable/stylesheet.dart

// ModifiableCssStylesheet is a modifiable version of CssStylesheet for use in
// the evaluation step.
//
// This is the root of the mutable output tree. Matches Dart:
// ModifiableCssStylesheet.
type ModifiableCssStylesheet struct {
	node baseParentNode
}

// NewModifiableCssStylesheet creates an empty modifiable stylesheet root.
//
// Matches Dart: ModifiableCssStylesheet constructor.
func NewModifiableCssStylesheet(span sasscommon.FileSpan) *ModifiableCssStylesheet {
	s := &ModifiableCssStylesheet{
		node: baseParentNode{node: baseNode{span: span}},
	}
	s.node.self = s
	return s
}

func (s *ModifiableCssStylesheet) Span() (sasscommon.FileSpan, error)  { return s.node.Span() }
func (s *ModifiableCssStylesheet) IsAstNode()                          {}
func (s *ModifiableCssStylesheet) IsCssNode()                          {}
func (s *ModifiableCssStylesheet) IsGroupEnd() bool                    { return s.node.IsGroupEnd() }
func (s *ModifiableCssStylesheet) SetIsGroupEnd(v bool)                { s.node.SetIsGroupEnd(v) }
func (s *ModifiableCssStylesheet) setParent(p ModifiableCssParentNode) { s.node.setParent(p) }
func (s *ModifiableCssStylesheet) setIndexInParent(i int)              { s.node.setIndexInParent(i) }
func (s *ModifiableCssStylesheet) indexInParent() int                  { return s.node.indexInParent() }
func (s *ModifiableCssStylesheet) Parent() CssParentNode               { return s.node.Parent() }
func (s *ModifiableCssStylesheet) HasFollowingSibling() bool           { return s.node.HasFollowingSibling() }
func (s *ModifiableCssStylesheet) Remove() error                       { return s.node.Remove() }
func (s *ModifiableCssStylesheet) Children() []CssNode                 { return s.node.Children() }
func (s *ModifiableCssStylesheet) ChildrenLen() int                    { return s.node.ChildrenLen() }
func (s *ModifiableCssStylesheet) LastChild() ModifiableCssNode        { return s.node.LastChild() }
func (s *ModifiableCssStylesheet) IsChildless() bool                   { return s.node.IsChildless() }
func (s *ModifiableCssStylesheet) IsCssParentNode()                    {}
func (s *ModifiableCssStylesheet) AddChild(child ModifiableCssNode) error {
	return s.node.AddChild(child)
}
func (s *ModifiableCssStylesheet) ClearChildren()          { s.node.ClearChildren() }
func (s *ModifiableCssStylesheet) removeChildAt(index int) { s.node.removeChildAt(index) }

func (s *ModifiableCssStylesheet) IsInvisible() bool {
	for _, child := range s.Children() {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

func (s *ModifiableCssStylesheet) IsInvisibleOtherThanBogusCombinators() bool {
	for _, child := range s.Children() {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

func (s *ModifiableCssStylesheet) IsInvisibleHidingComments() bool {
	for _, child := range s.Children() {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

// EqualsIgnoringChildren reports whether other is also a stylesheet;
// stylesheets carry no distinguishing fields beyond children.
//
// Matches Dart: ModifiableCssStylesheet.equalsIgnoringChildren.
func (s *ModifiableCssStylesheet) EqualsIgnoringChildren(other ModifiableCssNode) (bool, error) {
	_, ok := any(other).(*ModifiableCssStylesheet)
	return ok, nil
}

func (s *ModifiableCssStylesheet) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssStylesheet(s)
	return err
}

func (s *ModifiableCssStylesheet) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return s.node.AcceptCloneModifiableCssNode(v)
}

func (s *ModifiableCssStylesheet) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssStylesheet(s.ToCssStylesheet())
}

func (s *ModifiableCssStylesheet) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssStylesheet(s.ToCssStylesheet())
}

// CopyWithoutChildren returns an empty stylesheet with the same span.
//
// Matches Dart: ModifiableCssStylesheet.copyWithoutChildren.
func (s *ModifiableCssStylesheet) CopyWithoutChildren() (ModifiableCssParentNode, error) {
	return NewModifiableCssStylesheet(s.node.node.span), nil
}

// ToCssStylesheet converts this modifiable stylesheet to a non-modifiable
// CssStylesheet, suitable for use in evaluation results.
func (s *ModifiableCssStylesheet) ToCssStylesheet() *CssStylesheet {
	return NewCssStylesheet(s.Children(), s.node.node.span)
}
