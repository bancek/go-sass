// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/negation.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SupportsNegation inverts another condition with `not`.
type SupportsNegation struct {
	// Condition is the test being negated.
	Condition SupportsCondition
	span      sasscommon.FileSpan
}

// NewSupportsNegation creates the negation of condition.
func NewSupportsNegation(condition SupportsCondition, span sasscommon.FileSpan) *SupportsNegation {
	return &SupportsNegation{Condition: condition, span: span}
}

func (n *SupportsNegation) Span() (sasscommon.FileSpan, error) { return n.span, nil }
func (n *SupportsNegation) IsAstNode()                         {}
func (n *SupportsNegation) IsSassNode()                        {}
func (n *SupportsNegation) IsSupportsCondition()               {}

// ToInterpolation flattens the condition into source-equivalent text,
// preserving the `not` keyword around the nested condition.
func (n *SupportsNegation) ToInterpolation() (*Interpolation, error) {
	child, err := n.Condition.ToInterpolation()
	if err != nil {
		return nil, err
	}
	condSpan, err := n.Condition.Span()
	if err != nil {
		return nil, err
	}
	buf := &InterpolationBuffer{}
	beforeSpan, err := n.span.Before(condSpan)
	if err != nil {
		return nil, err
	}
	before, err := beforeSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(before)
	buf.AddInterpolation(child)
	afterSpan, err := n.span.After(condSpan)
	if err != nil {
		return nil, err
	}
	after, err := afterSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(after)
	return buf.Interpolation(n.span)
}

// WithSpan returns a copy of this condition covering span.
func (n *SupportsNegation) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsNegation(n.Condition, span)
}

func (n *SupportsNegation) String() (string, error) {
	condStr, err := n.Condition.String()
	if err != nil {
		return "", err
	}
	_, isNegation := n.Condition.(*SupportsNegation)
	_, isOperation := n.Condition.(*SupportsOperation)
	if isNegation || isOperation {
		return "not (" + condStr + ")", nil
	}
	return "not " + condStr, nil
}
