// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/import.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssImport is a modifiable version of CssImport for use in the
// evaluation step.
//
// Matches Dart: ModifiableCssImport.
type ModifiableCssImport struct {
	node      baseNode
	url       sasscommon.CssValue[string]
	modifiers *sasscommon.CssValue[string]
}

// NewModifiableCssImportFrom creates a modifiable import from a
// non-modifiable CssImport.
//
// Matches Dart: modifiable wrapper construction.
func NewModifiableCssImportFrom(node CssImport) (*ModifiableCssImport, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewModifiableCssImport(node.URL(), span, node.Modifiers()), nil
}

// NewModifiableCssImport creates a modifiable @import with url and optional modifiers.
//
// Matches Dart: ModifiableCssImport constructor.
func NewModifiableCssImport(
	url sasscommon.CssValue[string],
	span sasscommon.FileSpan,
	modifiers *sasscommon.CssValue[string],
) *ModifiableCssImport {
	return &ModifiableCssImport{
		node:      baseNode{span: span},
		url:       url,
		modifiers: modifiers,
	}
}

// URL returns the imported URL.
//
// Matches Dart: CssImport.url.
func (i *ModifiableCssImport) URL() sasscommon.CssValue[string] { return i.url }

// Modifiers returns the @import modifiers (media queries, supports, layer),
// or nil when absent.
//
// Matches Dart: CssImport.modifiers.
func (i *ModifiableCssImport) Modifiers() *sasscommon.CssValue[string] { return i.modifiers }

func (i *ModifiableCssImport) Span() (sasscommon.FileSpan, error) { return i.node.Span() }
func (i *ModifiableCssImport) IsAstNode()                         {}
func (i *ModifiableCssImport) IsCssNode()                         {}
func (i *ModifiableCssImport) IsGroupEnd() bool                   { return i.node.IsGroupEnd() }
func (i *ModifiableCssImport) SetIsGroupEnd(v bool)               { i.node.SetIsGroupEnd(v) }
func (i *ModifiableCssImport) IsInvisible() bool                  { return i.node.IsInvisible() }
func (i *ModifiableCssImport) IsInvisibleHidingComments() bool {
	return i.node.IsInvisibleHidingComments()
}
func (i *ModifiableCssImport) IsInvisibleOtherThanBogusCombinators() bool {
	return i.node.IsInvisibleOtherThanBogusCombinators()
}
func (i *ModifiableCssImport) setParent(p ModifiableCssParentNode) { i.node.setParent(p) }
func (i *ModifiableCssImport) setIndexInParent(idx int)            { i.node.setIndexInParent(idx) }
func (i *ModifiableCssImport) indexInParent() int                  { return i.node.indexInParent() }
func (i *ModifiableCssImport) Parent() CssParentNode               { return i.node.Parent() }
func (i *ModifiableCssImport) HasFollowingSibling() bool           { return i.node.HasFollowingSibling() }
func (i *ModifiableCssImport) Remove() error                       { return i.node.Remove() }

func (i *ModifiableCssImport) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssImport(i)
	return err
}

func (i *ModifiableCssImport) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssImport(i)
}

func (i *ModifiableCssImport) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssImport(i)
}

func (i *ModifiableCssImport) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssImport(i)
}
