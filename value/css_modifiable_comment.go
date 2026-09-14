// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/modifiable/comment.dart

// ModifiableCssComment is a modifiable version of CssComment for use in the
// evaluation step.
//
// Matches Dart: ModifiableCssComment.
type ModifiableCssComment struct {
	node baseNode
	text string
}

// NewModifiableCssCommentFrom creates a modifiable comment from a
// non-modifiable CssComment.
//
// Matches Dart: modifiable wrapper construction.
func NewModifiableCssCommentFrom(node CssComment) (*ModifiableCssComment, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssComment(node.Text(), span), nil
}

// NewModifiableCssComment creates a modifiable comment with the given text
// (including /* and */).
//
// Matches Dart: ModifiableCssComment constructor.
func NewModifiableCssComment(text string, span sasscommon.FileSpan) *ModifiableCssComment {
	return &ModifiableCssComment{
		node: baseNode{span: span},
		text: text,
	}
}

// Text returns the comment text including delimiters.
//
// Matches Dart: CssComment.text.
func (c *ModifiableCssComment) Text() string { return c.text }

// IsPreserved returns whether this comment is preserved (starts with /*!).
//
// Matches Dart: ModifiableCssComment.isPreserved
func (c *ModifiableCssComment) IsPreserved() bool {
	return c.text[2] == '!'
}

func (c *ModifiableCssComment) Span() (sasscommon.FileSpan, error) { return c.node.Span() }
func (c *ModifiableCssComment) IsAstNode()                         {}
func (c *ModifiableCssComment) IsCssNode()                         {}
func (c *ModifiableCssComment) IsGroupEnd() bool                   { return c.node.IsGroupEnd() }
func (c *ModifiableCssComment) SetIsGroupEnd(v bool)               { c.node.SetIsGroupEnd(v) }
func (c *ModifiableCssComment) IsInvisibleOtherThanBogusCombinators() bool {
	return c.node.IsInvisibleOtherThanBogusCombinators()
}
func (c *ModifiableCssComment) setParent(p ModifiableCssParentNode) { c.node.setParent(p) }
func (c *ModifiableCssComment) setIndexInParent(i int)              { c.node.setIndexInParent(i) }
func (c *ModifiableCssComment) indexInParent() int                  { return c.node.indexInParent() }
func (c *ModifiableCssComment) Parent() CssParentNode               { return c.node.Parent() }
func (c *ModifiableCssComment) HasFollowingSibling() bool           { return c.node.HasFollowingSibling() }
func (c *ModifiableCssComment) Remove() error                       { return c.node.Remove() }

func (c *ModifiableCssComment) IsInvisible() bool { return false }

// IsInvisibleHidingComments returns true if the comment is not preserved.
// Matches Dart: CssComment.isInvisibleHidingComments
func (c *ModifiableCssComment) IsInvisibleHidingComments() bool { return !c.IsPreserved() }

func (c *ModifiableCssComment) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssComment(c)
	return err
}

func (c *ModifiableCssComment) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssComment(c)
}

func (c *ModifiableCssComment) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssComment(c)
}

func (c *ModifiableCssComment) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssComment(c)
}
