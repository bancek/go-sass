// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/debug_rule.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// DebugRule is a @debug rule.
//
// It evaluates its expression and prints the resulting value to the logger
// for debugging, without affecting the compiled CSS.
type DebugRule struct {
	// Expression is the value to print.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewDebugRule creates a @debug rule that prints expression.
func NewDebugRule(expression Expression, span sasscommon.FileSpan) *DebugRule {
	return &DebugRule{Expression: expression, span: span}
}

func (r *DebugRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *DebugRule) IsStatement()                       {}
func (r *DebugRule) IsSassNode()                        {}
func (r *DebugRule) IsAstNode()                         {}
func (r *DebugRule) String() (string, error) {
	exprStr, err := r.Expression.String()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@debug %s;", exprStr), nil
}
