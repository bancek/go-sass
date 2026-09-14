// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/declaration.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// Declaration is a style declaration, that is, a name: value pair.
//
// A declaration either holds a plain value or nests child statements, in
// which case Value may be nil.
type Declaration struct {
	parent ParentStatement
	// Name is the interpolated property name.
	Name *Interpolation
	// Value is the declared value, or nil for a nested declaration without one.
	Value Expression
	// parsedAsSassScript reports whether Value was parsed as SassScript.
	// Custom properties and plain-CSS function results keep the raw text
	// instead; see ParsedAsSassScript.
	parsedAsSassScript bool
	span               sasscommon.FileSpan
}

// NewDeclaration creates a declaration with no children and parsedAsSassScript
// set to true.
func NewDeclaration(name *Interpolation, value Expression, span sasscommon.FileSpan) *Declaration {
	return &Declaration{
		parent:             NewParentStatement(nil),
		Name:               name,
		Value:              value,
		span:               span,
		parsedAsSassScript: true,
	}
}

// NewDeclarationNotSassScript creates a declaration whose value is raw text
// rather than SassScript, used for custom properties and plain-CSS results.
func NewDeclarationNotSassScript(name *Interpolation, value *StringExpression, span sasscommon.FileSpan) *Declaration {
	return &Declaration{
		parent:             NewParentStatement(nil),
		Name:               name,
		Value:              value,
		span:               span,
		parsedAsSassScript: false,
	}
}

// NewDeclarationNested creates a declaration with children.
// children must be non-nil; Dart's Declaration.nested() requires non-null
// children and would throw at List.unmodifiable(null).
func NewDeclarationNested(name *Interpolation, children []Statement, span sasscommon.FileSpan, value Expression) *Declaration {
	if children == nil {
		panic("BUG: NewDeclarationNested called with nil children")
	}
	c := make([]Statement, len(children))
	copy(c, children)
	return &Declaration{
		parent:             NewParentStatement(c),
		Name:               name,
		Value:              value,
		span:               span,
		parsedAsSassScript: true,
	}
}

// ParsedAsSassScript reports whether the value was parsed as SassScript.
// Raw-text values (custom properties, plain-CSS results) report false and
// hold an unquoted StringExpression.
func (d *Declaration) ParsedAsSassScript() bool           { return d.parsedAsSassScript }
func (d *Declaration) Span() (sasscommon.FileSpan, error) { return d.span, nil }
func (d *Declaration) IsStatement()                       {}
func (d *Declaration) IsSassNode()                        {}
func (d *Declaration) IsAstNode()                         {}
func (d *Declaration) GetChildren() []Statement           { return d.parent.Children }
func (d *Declaration) HasDeclarations() bool              { return d.parent.HasDeclarations() }
func (d *Declaration) String() (string, error) {
	var builder strings.Builder
	nameStr, err := d.Name.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(nameStr)
	builder.WriteRune(':')

	if d.Value != nil {
		if d.parsedAsSassScript {
			builder.WriteRune(' ')
		}
		valStr, err := d.Value.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(valStr)
	}

	if d.parent.Children != nil {
		builder.WriteString(" {")
		parts := make([]string, len(d.parent.Children))
		for i, child := range d.parent.Children {
			s, err := statementChildString(child)
			if err != nil {
				return "", err
			}
			parts[i] = s
		}
		builder.WriteString(strings.Join(parts, " "))
		builder.WriteString("}")
	} else {
		builder.WriteRune(';')
	}
	return builder.String(), nil
}
