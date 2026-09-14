// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/parenthesized.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// ParenthesizedExpression is an expression wrapped in parentheses.
//
// Matches Dart: ParenthesizedExpression
type ParenthesizedExpression struct {
	// Expression is the internal expression.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewParenthesizedExpression creates a parenthesized wrapper around expr.
//
// Matches Dart: ParenthesizedExpression.new
func NewParenthesizedExpression(expr Expression, span sasscommon.FileSpan) *ParenthesizedExpression {
	return &ParenthesizedExpression{Expression: expr, span: span}
}

func (e *ParenthesizedExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *ParenthesizedExpression) SourceInterpolation() *Interpolation { return nil }
func (e *ParenthesizedExpression) IsExpression()                       {}
func (e *ParenthesizedExpression) IsSassNode()                         {}
func (e *ParenthesizedExpression) IsAstNode()                          {}
func (e *ParenthesizedExpression) String() (string, error) {
	exprStr, err := e.Expression.String()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s)", exprStr), nil
}
