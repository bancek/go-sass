// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/return_rule.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ReturnRule is a @return rule.
//
// It exits the enclosing function body with a return value.
type ReturnRule struct {
	// Expression is the value returned to the function caller.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewReturnRule creates a @return rule returning expression.
func NewReturnRule(expression Expression, span sasscommon.FileSpan) *ReturnRule {
	return &ReturnRule{Expression: expression, span: span}
}

func (r *ReturnRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ReturnRule) IsStatement()                       {}
func (r *ReturnRule) IsSassNode()                        {}
func (r *ReturnRule) IsAstNode()                         {}
func (r *ReturnRule) String() (string, error) {
	exprStr, err := r.Expression.String()
	if err != nil {
		return "", err
	}
	return "@return " + exprStr + ";", nil
}
