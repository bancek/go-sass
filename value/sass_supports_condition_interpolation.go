// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/interpolation.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SupportsInterpolation is a condition that is itself an interpolation,
// evaluated and re-parsed before testing.
type SupportsInterpolation struct {
	// Expression is the interpolated condition source.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewSupportsInterpolation creates a condition from an interpolated expression.
func NewSupportsInterpolation(expression Expression, span sasscommon.FileSpan) *SupportsInterpolation {
	return &SupportsInterpolation{Expression: expression, span: span}
}

func (i *SupportsInterpolation) Span() (sasscommon.FileSpan, error) { return i.span, nil }
func (i *SupportsInterpolation) IsAstNode()                         {}
func (i *SupportsInterpolation) IsSassNode()                        {}
func (i *SupportsInterpolation) IsSupportsCondition()               {}

// ToInterpolation wraps the interpolated expression as a single-part
// interpolation.
func (i *SupportsInterpolation) ToInterpolation() (*Interpolation, error) {
	return NewInterpolation(
		[]any{i.Expression},
		[]*sasscommon.FileSpan{&i.span},
		i.span,
	)
}

// WithSpan returns a copy of this condition covering span.
func (i *SupportsInterpolation) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsInterpolation(i.Expression, span)
}
func (i *SupportsInterpolation) String() (string, error) {
	exprStr, err := i.Expression.String()
	if err != nil {
		return "", err
	}
	return "#{" + exprStr + "}", nil
}
