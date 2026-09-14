// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/unary_operation.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// UnaryOperator is a unary operator constant.
//
// Matches Dart: UnaryOperator enum. Each item carries its English name and
// its Sass syntax.
type UnaryOperator int

const (
	// UnaryOperatorPlus is the numeric identity operator, `+`.
	UnaryOperatorPlus UnaryOperator = iota
	// UnaryOperatorMinus is the numeric negation operator, `-`.
	UnaryOperatorMinus
	// UnaryOperatorDivide is the leading-slash operator, `/`, a historical
	// artifact.
	UnaryOperatorDivide
	// UnaryOperatorNot is the boolean negation operator, `not`.
	UnaryOperatorNot
)

// String returns the English name of the operator.
//
// Matches Dart: UnaryOperator.toString
func (op UnaryOperator) String() string {
	switch op {
	case UnaryOperatorPlus:
		return "plus"
	case UnaryOperatorMinus:
		return "minus"
	case UnaryOperatorDivide:
		return "divide"
	case UnaryOperatorNot:
		return "not"
	default:
		return fmt.Sprintf("UnaryOperator(%d)", op)
	}
}

// OperatorSyntax returns the Sass syntax for the operator.
//
// Matches Dart: UnaryOperator.operator
func (op UnaryOperator) OperatorSyntax() string {
	switch op {
	case UnaryOperatorPlus:
		return "+"
	case UnaryOperatorMinus:
		return "-"
	case UnaryOperatorDivide:
		return "/"
	case UnaryOperatorNot:
		return "not"
	default:
		return "?"
	}
}

// UnaryOperationExpression is a unary operator, as in `+$var` or `not fn()`.
//
// Matches Dart: UnaryOperationExpression
type UnaryOperationExpression struct {
	// Operator is the operator being invoked.
	Operator UnaryOperator
	// Operand is the operand.
	Operand Expression
	span    sasscommon.FileSpan
}

// NewUnaryOperationExpression creates a unary operation.
//
// Matches Dart: UnaryOperationExpression.new
func NewUnaryOperationExpression(operator UnaryOperator, operand Expression, span sasscommon.FileSpan) *UnaryOperationExpression {
	return &UnaryOperationExpression{
		Operator: operator,
		Operand:  operand,
		span:     span,
	}
}

func (e *UnaryOperationExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *UnaryOperationExpression) SourceInterpolation() *Interpolation { return nil }
func (e *UnaryOperationExpression) IsExpression()                       {}
func (e *UnaryOperationExpression) IsSassNode()                         {}
func (e *UnaryOperationExpression) IsAstNode()                          {}

// String renders the operator plus its operand, parenthesizing nested
// binary/unary operations and bracketless multi-element lists so the output
// re-parses with the same grouping. `not` is followed by a space.
func (e *UnaryOperationExpression) String() (string, error) {
	var buf strings.Builder

	buf.WriteString(e.Operator.OperatorSyntax())
	if e.Operator == UnaryOperatorNot {
		buf.WriteRune(' ')
	}

	needsParens := false
	if _, ok := any(e.Operand).(*BinaryOperationExpression); ok {
		needsParens = true
	} else if _, ok := any(e.Operand).(*UnaryOperationExpression); ok {
		needsParens = true
	} else if listExpr, ok := any(e.Operand).(*ListExpression); ok {
		if !listExpr.HasBrackets && len(listExpr.Contents) >= 2 {
			needsParens = true
		}
	}

	if needsParens {
		buf.WriteRune('(')
	}
	opStr, err := e.Operand.String()
	if err != nil {
		return "", err
	}
	buf.WriteString(opStr)
	if needsParens {
		buf.WriteRune(')')
	}

	return buf.String(), nil
}
