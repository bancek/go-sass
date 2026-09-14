// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/supports.dart

import "github.com/bancek/go-sass/sasscommon"

// SupportsExpression is an expression-level @supports condition.
//
// This appears only in the modifiers after a plain-CSS @import, without the
// function name wrapping the condition.
//
// Matches Dart: SupportsExpression
type SupportsExpression struct {
	// Condition is the condition itself.
	Condition SupportsCondition
}

// NewSupportsExpression creates an expression wrapping a @supports
// condition.
//
// Matches Dart: SupportsExpression.new
func NewSupportsExpression(condition SupportsCondition) *SupportsExpression {
	return &SupportsExpression{Condition: condition}
}

func (e *SupportsExpression) Span() (sasscommon.FileSpan, error)  { return e.Condition.Span() }
func (e *SupportsExpression) SourceInterpolation() *Interpolation { return nil }
func (e *SupportsExpression) IsExpression()                       {}
func (e *SupportsExpression) IsSassNode()                         {}
func (e *SupportsExpression) IsAstNode()                          {}
func (e *SupportsExpression) String() (string, error)             { return e.Condition.String() }
