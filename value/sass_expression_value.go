// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/value.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ValueExpression is an expression that directly embeds a Value.
//
// This is never built by the parser. It is only used when ASTs are
// constructed dynamically, as for the `call()` function.
//
// Matches Dart: ValueExpression
type ValueExpression struct {
	// Value is the embedded value.
	Value Value
	span  sasscommon.FileSpan
}

// NewValueExpression creates an expression embedding value with the given
// span.
//
// Matches Dart: ValueExpression.new
func NewValueExpression(value Value, span sasscommon.FileSpan) *ValueExpression {
	return &ValueExpression{Value: value, span: span}
}

func (e *ValueExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *ValueExpression) SourceInterpolation() *Interpolation { return nil }
func (e *ValueExpression) IsExpression()                       {}
func (e *ValueExpression) IsSassNode()                         {}
func (e *ValueExpression) IsAstNode()                          {}
func (e *ValueExpression) String() (string, error)             { return e.Value.String() }
