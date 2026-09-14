// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/error_rule.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// ErrorRule is an @error rule.
//
// It evaluates its expression and aborts compilation with that message.
type ErrorRule struct {
	// Expression is evaluated to produce the error message.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewErrorRule creates an @error rule reporting the value of expression.
func NewErrorRule(expression Expression, span sasscommon.FileSpan) *ErrorRule {
	return &ErrorRule{Expression: expression, span: span}
}

func (r *ErrorRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ErrorRule) IsStatement()                       {}
func (r *ErrorRule) IsSassNode()                        {}
func (r *ErrorRule) IsAstNode()                         {}
func (r *ErrorRule) String() (string, error) {
	exprStr, err := r.Expression.String()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@error %s;", exprStr), nil
}
