// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/number.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// NumberExpression is a number literal.
//
// Matches Dart: NumberExpression
type NumberExpression struct {
	// Value is the numeric value.
	Value float64
	// Unit is the number's unit, or nil for a unitless number.
	Unit *string
	span sasscommon.FileSpan
}

// NewNumberExpression creates a number literal with the given value, span,
// and optional unit.
//
// Matches Dart: NumberExpression.new
func NewNumberExpression(value float64, span sasscommon.FileSpan, unit *string) *NumberExpression {
	return &NumberExpression{Value: value, Unit: unit, span: span}
}

func (e *NumberExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *NumberExpression) SourceInterpolation() *Interpolation { return nil }
func (e *NumberExpression) IsExpression()                       {}
func (e *NumberExpression) IsSassNode()                         {}
func (e *NumberExpression) IsAstNode()                          {}
func (e *NumberExpression) String() (string, error)             { return NewSassNumber(e.Value, e.Unit).String() }
