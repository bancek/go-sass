// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/expression_to_calc.dart

import (
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

// ExpressionToCalc converts expression to an equivalent calc() call.
//
// This assumes expression already returns a number. It is intended for
// end-user messaging and may not produce directly evaluable expressions:
// the result wraps the calculation-safe rewrite in a calc() function.
//
// Matches Dart: expressionToCalc
func ExpressionToCalc(expression Expression) (*FunctionExpression, error) {
	calc := &makeExpressionCalculationSafe{}
	calc.ReplaceExpressionVisitor.visitor = calc
	expr, err := expression.AcceptExpr(calc)
	if err != nil {
		return nil, err
	}
	span, err := expression.Span()
	if err != nil {
		return nil, err
	}
	return NewFunctionExpression(
		"calc",
		NewArgumentList(
			[]Expression{expr},
			orderedmap.New[string, Expression](),
			map[string]sasscommon.FileSpan{},
			span,
			nil,
			nil,
		),
		span,
		nil,
	), nil
}

// makeExpressionCalculationSafe rewrites constructs that calc() cannot use
// into ones it can, delegating everything else to ReplaceExpressionVisitor.
//
// Matches Dart: _MakeExpressionCalculationSafe
type makeExpressionCalculationSafe struct {
	ReplaceExpressionVisitor
}

// VisitBinaryOperationExpression rewrites `%` (unsupported by calc(), with
// no browser-backed mod() alternative) into a math.max() call so the message
// still shows valid syntax; all other operators recurse normally.
func (v *makeExpressionCalculationSafe) VisitBinaryOperationExpression(node *BinaryOperationExpression) (Expression, error) {
	if node.Operator == BinaryOperatorModulo {
		// calc() doesn't support % for modulo but Sass doesn't yet support the
		// mod() calculation function because there's no browser support, so we
		// have to work around it by wrapping the call in a Sass function.
		ns := "math"
		span, err := node.Span()
		if err != nil {
			return nil, err
		}
		return NewFunctionExpression(
			"max",
			NewArgumentList(
				[]Expression{node},
				orderedmap.New[string, Expression](),
				map[string]sasscommon.FileSpan{},
				span,
				nil,
				nil,
			),
			span,
			&ns,
		), nil
	}
	return v.ReplaceExpressionVisitor.VisitBinaryOperationExpression(node)
}

// VisitInterpolatedFunctionExpression keeps interpolated calls as-is: their
// names are unknown until evaluation, so no calculation rewrite applies.
func (v *makeExpressionCalculationSafe) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (Expression, error) {
	return node, nil
}

// VisitIfExpression keeps if() as-is: branching cannot be lowered into
// calc() arithmetic.
func (v *makeExpressionCalculationSafe) VisitIfExpression(node *IfExpression) (Expression, error) {
	return node, nil
}

// VisitUnaryOperationExpression lowers calc()-unsupported unary operators:
// plus is dropped, minus becomes multiplication by -1, and non-numeric
// operators are kept so serialization can report a useful syntax error.
func (v *makeExpressionCalculationSafe) VisitUnaryOperationExpression(node *UnaryOperationExpression) (Expression, error) {
	switch node.Operator {
	case UnaryOperatorPlus:
		// calc() doesn't support unary operations.
		return node.Operand, nil
	case UnaryOperatorMinus:
		span, err := node.Span()
		if err != nil {
			return nil, err
		}
		bin := NewBinaryOperationExpression(BinaryOperatorTimes, NewNumberExpression(-1, span, nil), node.Operand)
		return bin, nil
	default:
		// Other unary operations don't produce numbers, so keep them as-is to
		// give the user a more useful syntax error after serialization.
		return v.ReplaceExpressionVisitor.VisitUnaryOperationExpression(node)
	}
}
