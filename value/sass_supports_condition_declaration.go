// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/declaration.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// SupportsDeclaration is a condition that selects for browsers where a given
// declaration is supported.
type SupportsDeclaration struct {
	// Name is the declaration name under test.
	Name Expression
	// Value is the declaration value under test.
	Value Expression
	span  sasscommon.FileSpan
}

// NewSupportsDeclaration creates a (name: value) support test.
func NewSupportsDeclaration(name, value Expression, span sasscommon.FileSpan) *SupportsDeclaration {
	return &SupportsDeclaration{Name: name, Value: value, span: span}
}

func (d *SupportsDeclaration) Span() (sasscommon.FileSpan, error) { return d.span, nil }
func (d *SupportsDeclaration) IsAstNode()                         {}
func (d *SupportsDeclaration) IsSassNode()                        {}
func (d *SupportsDeclaration) IsSupportsCondition()               {}

// ToInterpolation flattens the condition into source-equivalent text,
// keeping unquoted names and already-interpolated values intact.
func (d *SupportsDeclaration) ToInterpolation() (*Interpolation, error) {
	nameSpan, err := d.Name.Span()
	if err != nil {
		return nil, err
	}
	buf := &InterpolationBuffer{}
	beforeSpan, err := d.span.Before(nameSpan)
	if err != nil {
		return nil, err
	}
	before, err := beforeSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(before)
	if se, ok := d.Name.(*StringExpression); ok && !se.HasQuotes {
		buf.AddInterpolation(se.Text)
	} else {
		buf.Add(d.Name, nameSpan)
	}
	valueSpan, err := d.Value.Span()
	if err != nil {
		return nil, err
	}
	betweenSpan, err := nameSpan.Between(valueSpan)
	if err != nil {
		return nil, err
	}
	between, err := betweenSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(between)
	if interp := d.Value.SourceInterpolation(); interp != nil {
		buf.AddInterpolation(interp)
	} else {
		buf.Add(d.Value, valueSpan)
	}
	afterSpan, err := d.span.After(valueSpan)
	if err != nil {
		return nil, err
	}
	after, err := afterSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(after)
	return buf.Interpolation(d.span)
}

// WithSpan returns a copy of this condition covering span.
func (d *SupportsDeclaration) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsDeclaration(d.Name, d.Value, span)
}
func (d *SupportsDeclaration) String() (string, error) {
	nameStr, err := d.Name.String()
	if err != nil {
		return "", err
	}
	valStr, err := d.Value.String()
	if err != nil {
		return "", err
	}
	return "(" + nameStr + ": " + valStr + ")", nil
}

// isCustomProperty reports whether the tested name is a custom property.
// Only names parsed as plain text count; an interpolated name that happens
// to serialize with a -- prefix does not.
func (d *SupportsDeclaration) isCustomProperty() bool {
	if se, ok := any(d.Name).(*StringExpression); ok && !se.HasQuotes {
		return strings.HasPrefix(se.Text.InitialPlain(), "--")
	}
	return false
}
