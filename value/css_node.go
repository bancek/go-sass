// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/node.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// CssNode is a statement in a plain CSS syntax tree.
//
// Matches Dart: CssNode. Parent is nil for the stylesheet root; IsGroupEnd
// marks nodes from the tail of a flattened nested tree. The invisibility
// queries drive output elision (loud comments count as visible unless hidden
// variants are asked).
type CssNode interface {
	sasscommon.AstNode
	Parent() CssParentNode
	IsGroupEnd() bool
	IsInvisible() bool
	IsInvisibleHidingComments() bool
	IsInvisibleOtherThanBogusCombinators() bool
	IsCssNode()
	AcceptVoid(v CssVisitor[struct{}]) (struct{}, error)
	AcceptBool(v CssVisitor[bool]) (bool, error)
}

// CssParentNode is a CssNode that can have child statements.
//
// IsChildless distinguishes `@foo;` from `@foo {}`: childless implies no
// children but not vice versa. Matches Dart: CssParentNode.
type CssParentNode interface {
	CssNode
	Children() []CssNode
	IsChildless() bool
	IsCssParentNode()
}

// CssNodeBase provides the common embedded implementation for CSS nodes.
//
// Frozen nodes embed this for span storage and default-visible behavior;
// modifiable nodes use baseNode/baseParentNode instead. Matches Dart:
// CssNode default isInvisible getters (via _IsInvisibleVisitor).
type CssNodeBase struct {
	span sasscommon.FileSpan
}

// NewCssNodeBase creates the shared base with the given span.
//
// Matches Dart: CssNode span storage.
func NewCssNodeBase(span sasscommon.FileSpan) CssNodeBase {
	return CssNodeBase{span: span}
}

func (b CssNodeBase) Span() (sasscommon.FileSpan, error)         { return b.span, nil }
func (b CssNodeBase) IsAstNode()                                 {}
func (b CssNodeBase) IsCssNode()                                 {}
func (b CssNodeBase) IsCssParentNode()                           {}
func (b CssNodeBase) IsInvisible() bool                          { return false }
func (b CssNodeBase) IsInvisibleHidingComments() bool            { return false }
func (b CssNodeBase) IsInvisibleOtherThanBogusCombinators() bool { return false }
