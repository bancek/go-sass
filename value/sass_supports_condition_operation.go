// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/operation.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SupportsOperation joins two conditions with `and` or `or`.
type SupportsOperation struct {
	// Left is the left-hand operand.
	Left SupportsCondition
	// Right is the right-hand operand.
	Right SupportsCondition
	// Operator is the and/or connective.
	Operator BooleanOperator
	span     sasscommon.FileSpan
}

// NewSupportsOperation creates a binary condition joining left and right
// with operator.
func NewSupportsOperation(left, right SupportsCondition, operator BooleanOperator, span sasscommon.FileSpan) *SupportsOperation {
	return &SupportsOperation{Left: left, Right: right, Operator: operator, span: span}
}

func (o *SupportsOperation) Span() (sasscommon.FileSpan, error) { return o.span, nil }
func (o *SupportsOperation) IsAstNode()                         {}
func (o *SupportsOperation) IsSassNode()                        {}
func (o *SupportsOperation) IsSupportsCondition()               {}

// ToInterpolation flattens the condition into source-equivalent text,
// preserving the literal around and between the two operands.
func (o *SupportsOperation) ToInterpolation() (*Interpolation, error) {
	left, err := o.Left.ToInterpolation()
	if err != nil {
		return nil, err
	}
	right, err := o.Right.ToInterpolation()
	if err != nil {
		return nil, err
	}
	leftSpan, err := o.Left.Span()
	if err != nil {
		return nil, err
	}
	rightSpan, err := o.Right.Span()
	if err != nil {
		return nil, err
	}
	buf := &InterpolationBuffer{}
	beforeSpan, err := o.span.Before(leftSpan)
	if err != nil {
		return nil, err
	}
	before, err := beforeSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(before)
	buf.AddInterpolation(left)
	betweenSpan, err := leftSpan.Between(rightSpan)
	if err != nil {
		return nil, err
	}
	between, err := betweenSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(between)
	buf.AddInterpolation(right)
	afterSpan, err := o.span.After(rightSpan)
	if err != nil {
		return nil, err
	}
	after, err := afterSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(after)
	return buf.Interpolation(o.span)
}

// WithSpan returns a copy of this condition covering span.
func (o *SupportsOperation) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsOperation(o.Left, o.Right, o.Operator, span)
}

func (o *SupportsOperation) String() (string, error) {
	leftStr, err := parenthesize(o.Left, o.Operator)
	if err != nil {
		return "", err
	}
	rightStr, err := parenthesize(o.Right, o.Operator)
	if err != nil {
		return "", err
	}
	return leftStr + " " + o.Operator.String() + " " + rightStr, nil
}

// parenthesize renders condition, adding parentheses when it is a negation
// or a same-operator nesting that would otherwise read ambiguously.
func parenthesize(condition SupportsCondition, op BooleanOperator) (string, error) {
	condStr, err := condition.String()
	if err != nil {
		return "", err
	}
	_, isNegation := condition.(*SupportsNegation)
	if isNegation {
		return "(" + condStr + ")", nil
	}
	if so, isOperation := condition.(*SupportsOperation); isOperation && so.Operator == op {
		return "(" + condStr + ")", nil
	}
	return condStr, nil
}
