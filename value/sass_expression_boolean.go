// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/boolean.dart

import (
	"strconv"

	"github.com/bancek/go-sass/sasscommon"
)

// BooleanExpression is a boolean literal, `true` or `false`.
//
// Matches Dart: BooleanExpression
type BooleanExpression struct {
	// Value is the value of this expression.
	Value bool
	span  sasscommon.FileSpan
}

// NewBooleanExpression creates a boolean literal with the given value and
// span.
//
// Matches Dart: BooleanExpression.new
func NewBooleanExpression(value bool, span sasscommon.FileSpan) *BooleanExpression {
	return &BooleanExpression{Value: value, span: span}
}

func (e *BooleanExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *BooleanExpression) SourceInterpolation() *Interpolation { return nil }
func (e *BooleanExpression) IsExpression()                       {}
func (e *BooleanExpression) IsSassNode()                         {}
func (e *BooleanExpression) IsAstNode()                          {}
func (e *BooleanExpression) String() (string, error)             { return strconv.FormatBool(e.Value), nil }
