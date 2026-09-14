// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/color.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ColorExpression is a color literal.
//
// Matches Dart: ColorExpression
type ColorExpression struct {
	// Value is the value of this color.
	Value *SassColor
	span  sasscommon.FileSpan
}

// NewColorExpression creates a color literal with the given value and span.
//
// Matches Dart: ColorExpression.new
func NewColorExpression(value *SassColor, span sasscommon.FileSpan) *ColorExpression {
	return &ColorExpression{Value: value, span: span}
}

func (e *ColorExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *ColorExpression) SourceInterpolation() *Interpolation { return nil }
func (e *ColorExpression) IsExpression()                       {}
func (e *ColorExpression) IsSassNode()                         {}
func (e *ColorExpression) IsAstNode()                          {}
func (e *ColorExpression) String() (string, error)             { return e.Value.String() }
