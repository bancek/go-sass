// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"net/url"

	"github.com/bancek/go-sass/sasscommon"
)

// dart-source: lib/src/ast/css/stylesheet.dart

// CssStylesheet is a plain CSS stylesheet. This is the root plain CSS node.
//
// Its parent is always nil and it is never childless. Matches Dart:
// CssStylesheet.
type CssStylesheet struct {
	node     CssNodeBase
	children []CssNode
}

// NewCssStylesheet creates a stylesheet containing the given children.
//
// Children are copied; the list view is unmodifiable in Dart. Matches Dart:
// CssStylesheet constructor.
func NewCssStylesheet(children []CssNode, span sasscommon.FileSpan) *CssStylesheet {
	c := make([]CssNode, len(children))
	copy(c, children)
	return &CssStylesheet{
		node:     CssNodeBase{span: span},
		children: c,
	}
}

// NewCssStylesheetEmpty creates an empty stylesheet with the given source URL.
//
// Matches Dart: CssStylesheet.empty({Object? url})
func NewCssStylesheetEmpty(sourceURL *url.URL) *CssStylesheet {
	file := sasscommon.NewFileSource([]byte{}, sourceURL)
	return &CssStylesheet{
		node:     CssNodeBase{span: sasscommon.NewFileSpan(file, 0, 0)},
		children: []CssNode{},
	}
}

func (s *CssStylesheet) Parent() CssParentNode { return nil }
func (s *CssStylesheet) IsGroupEnd() bool      { return false }
func (s *CssStylesheet) IsChildless() bool     { return false }
func (s *CssStylesheet) Children() []CssNode   { return s.children }

func (s *CssStylesheet) Span() (sasscommon.FileSpan, error) { return s.node.Span() }
func (s *CssStylesheet) IsAstNode()                         {}
func (s *CssStylesheet) IsCssNode()                         {}
func (s *CssStylesheet) IsCssParentNode()                   {}
func (s *CssStylesheet) IsInvisible() bool {
	for _, child := range s.children {
		if !child.IsInvisible() {
			return false
		}
	}
	return true
}

func (s *CssStylesheet) IsInvisibleHidingComments() bool {
	for _, child := range s.children {
		if !child.IsInvisibleHidingComments() {
			return false
		}
	}
	return true
}

func (s *CssStylesheet) IsInvisibleOtherThanBogusCombinators() bool {
	for _, child := range s.children {
		if !child.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

func (s *CssStylesheet) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssStylesheet(s)
}

func (s *CssStylesheet) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssStylesheet(s)
}
