// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/modifiable/declaration.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ModifiableCssDeclaration is a modifiable version of CssDeclaration for use
// in the evaluation step.
//
// Custom-property values stay unparsed SassStrings while SassScript values
// hold parsed values; ValueSpanForMap points at the declaration site for
// source maps. Matches Dart: ModifiableCssDeclaration.
type ModifiableCssDeclaration struct {
	node               baseNode
	innerName          sasscommon.CssValue[string]
	innerValue         sasscommon.CssValue[Value]
	parsedAsSassScript bool
	valueSpanForMap    sasscommon.FileSpan
}

// NewModifiableCssDeclaration creates a modifiable declaration, validating
// that non-SassScript values hold an unquoted SassString.
//
// Matches Dart: ModifiableCssDeclaration constructor.
func NewModifiableCssDeclaration(
	name sasscommon.CssValue[string],
	val sasscommon.CssValue[Value],
	span sasscommon.FileSpan,
	parsedAsSassScript bool,
	valueSpanForMap *sasscommon.FileSpan,
) (*ModifiableCssDeclaration, error) {
	vs, err := val.Span()
	if err != nil {
		return nil, err
	}
	if valueSpanForMap != nil {
		vs = *valueSpanForMap
	}
	if !parsedAsSassScript {
		if _, ok := val.Value.(*SassString); !ok {
			return nil, &sasscommon.ArgumentError{Message: fmt.Sprintf("If parsedAsSassScript is false, value must contain a SassString (was `%s` of type %T).", val, val.Value)}
		}
	}
	return &ModifiableCssDeclaration{
		node:               baseNode{span: span},
		innerName:          name,
		innerValue:         val,
		parsedAsSassScript: parsedAsSassScript,
		valueSpanForMap:    vs,
	}, nil
}

// Name returns the declaration name.
//
// Matches Dart: CssDeclaration.name.
func (d *ModifiableCssDeclaration) Name() sasscommon.CssValue[string] { return d.innerName }

// Value returns the declaration value.
//
// Matches Dart: CssDeclaration.value.
func (d *ModifiableCssDeclaration) Value() sasscommon.CssValue[Value] { return d.innerValue }

// ValueSpanForMap returns the span emitted to source maps (the declaration
// site for variable-backed values).
//
// Matches Dart: CssDeclaration.valueSpanForMap.
func (d *ModifiableCssDeclaration) ValueSpanForMap() sasscommon.FileSpan { return d.valueSpanForMap }

// ParsedAsSassScript reports whether the value was parsed as SassScript.
//
// Matches Dart: CssDeclaration.parsedAsSassScript.
func (d *ModifiableCssDeclaration) ParsedAsSassScript() bool { return d.parsedAsSassScript }

// IsCustomProperty reports whether the name starts with "--".
//
// Matches Dart: CssDeclaration.isCustomProperty.
func (d *ModifiableCssDeclaration) IsCustomProperty() bool {
	return strings.HasPrefix(d.innerName.Value, "--")
}

func (d *ModifiableCssDeclaration) Span() (sasscommon.FileSpan, error) { return d.node.Span() }
func (d *ModifiableCssDeclaration) IsAstNode()                         {}
func (d *ModifiableCssDeclaration) IsCssNode()                         {}
func (d *ModifiableCssDeclaration) IsGroupEnd() bool                   { return d.node.IsGroupEnd() }
func (d *ModifiableCssDeclaration) SetIsGroupEnd(v bool)               { d.node.SetIsGroupEnd(v) }
func (d *ModifiableCssDeclaration) IsInvisible() bool                  { return d.node.IsInvisible() }
func (d *ModifiableCssDeclaration) IsInvisibleHidingComments() bool {
	return d.node.IsInvisibleHidingComments()
}
func (d *ModifiableCssDeclaration) IsInvisibleOtherThanBogusCombinators() bool {
	return d.node.IsInvisibleOtherThanBogusCombinators()
}
func (d *ModifiableCssDeclaration) setParent(p ModifiableCssParentNode) { d.node.setParent(p) }
func (d *ModifiableCssDeclaration) setIndexInParent(i int)              { d.node.setIndexInParent(i) }
func (d *ModifiableCssDeclaration) indexInParent() int                  { return d.node.indexInParent() }
func (d *ModifiableCssDeclaration) Parent() CssParentNode               { return d.node.Parent() }
func (d *ModifiableCssDeclaration) HasFollowingSibling() bool           { return d.node.HasFollowingSibling() }
func (d *ModifiableCssDeclaration) Remove() error                       { return d.node.Remove() }

func (d *ModifiableCssDeclaration) AcceptModifiableVoid(visitor ModifiableCssVisitor) error {
	_, err := visitor.VisitModifiableCssDeclaration(d)
	return err
}

func (d *ModifiableCssDeclaration) AcceptCloneModifiableCssNode(v CloneCssVisitor) (ModifiableCssNode, error) {
	return v.VisitCloneCssDeclaration(d)
}

func (d *ModifiableCssDeclaration) AcceptVoid(v CssVisitor[struct{}]) (struct{}, error) {
	return v.VisitCssDeclaration(d)
}

func (d *ModifiableCssDeclaration) AcceptBool(v CssVisitor[bool]) (bool, error) {
	return v.VisitCssDeclaration(d)
}

// String renders the declaration as "name: value;".
//
// Matches Dart: CssDeclaration toString via serialization.
func (d *ModifiableCssDeclaration) String() string {
	return d.innerName.String() + ": " + d.innerValue.String() + ";"
}
