// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/expression/selector.dart

// SelectorExpression is a parent selector reference, `&`.
//
// Matches Dart: SelectorExpression
type SelectorExpression struct {
	span sasscommon.FileSpan
}

// NewSelectorExpression creates a parent selector reference covering span.
//
// Matches Dart: SelectorExpression.new
func NewSelectorExpression(span sasscommon.FileSpan) *SelectorExpression {
	return &SelectorExpression{span: span}
}

func (e *SelectorExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *SelectorExpression) SourceInterpolation() *Interpolation { return nil }
func (e *SelectorExpression) IsExpression()                       {}
func (e *SelectorExpression) IsSassNode()                         {}
func (e *SelectorExpression) IsAstNode()                          {}
func (e *SelectorExpression) String() (string, error)             { return "&", nil }
