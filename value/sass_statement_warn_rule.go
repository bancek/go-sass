// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/warn_rule.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// WarnRule is a @warn rule.
//
// It evaluates its expression and reports the result as a compilation
// warning, usually a string telling the user about something risky.
type WarnRule struct {
	// Expression is the value printed as the warning message.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewWarnRule creates a @warn rule reporting expression.
func NewWarnRule(expression Expression, span sasscommon.FileSpan) *WarnRule {
	return &WarnRule{Expression: expression, span: span}
}

func (r *WarnRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *WarnRule) IsStatement()                       {}
func (r *WarnRule) IsSassNode()                        {}
func (r *WarnRule) IsAstNode()                         {}
func (r *WarnRule) String() (string, error) {
	exprStr, err := r.Expression.String()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@warn %s;", exprStr), nil
}
