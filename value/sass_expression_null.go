// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/expression/null.dart

// NullExpression is a null literal.
//
// Matches Dart: NullExpression
type NullExpression struct {
	span sasscommon.FileSpan
}

// NewNullExpression creates a null literal covering span.
//
// Matches Dart: NullExpression.new
func NewNullExpression(span sasscommon.FileSpan) *NullExpression {
	return &NullExpression{span: span}
}

func (e *NullExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *NullExpression) SourceInterpolation() *Interpolation { return nil }
func (e *NullExpression) IsExpression()                       {}
func (e *NullExpression) IsSassNode()                         {}
func (e *NullExpression) IsAstNode()                          {}
func (e *NullExpression) String() (string, error)             { return "null", nil }
