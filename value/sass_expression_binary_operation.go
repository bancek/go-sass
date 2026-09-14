// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/binary_operation.dart

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// BinaryOperator is a binary operator constant, as in `1 + 2` or
// `$this and $other`.
//
// Matches Dart: BinaryOperator enum. Each item carries its English name, its
// Sass syntax, and its precedence (tighter binding means higher precedence).
type BinaryOperator int

const (
	// BinaryOperatorSingleEquals is the legacy `=` operator.
	BinaryOperatorSingleEquals BinaryOperator = iota
	// BinaryOperatorOr is the disjunction operator, `or`.
	BinaryOperatorOr
	// BinaryOperatorAnd is the conjunction operator, `and`.
	BinaryOperatorAnd
	// BinaryOperatorEquals is the equality operator, `==`.
	BinaryOperatorEquals
	// BinaryOperatorNotEquals is the inequality operator, `!=`.
	BinaryOperatorNotEquals
	// BinaryOperatorGreaterThan is the greater-than operator, `>`.
	BinaryOperatorGreaterThan
	// BinaryOperatorGreaterThanOrEquals is the greater-than-or-equal-to
	// operator, `>=`.
	BinaryOperatorGreaterThanOrEquals
	// BinaryOperatorLessThan is the less-than operator, `<`.
	BinaryOperatorLessThan
	// BinaryOperatorLessThanOrEquals is the less-than-or-equal-to operator,
	// `<=`.
	BinaryOperatorLessThanOrEquals
	// BinaryOperatorPlus is the addition operator, `+`.
	BinaryOperatorPlus
	// BinaryOperatorMinus is the subtraction operator, `-`.
	BinaryOperatorMinus
	// BinaryOperatorTimes is the multiplication operator, `*`.
	BinaryOperatorTimes
	// BinaryOperatorDividedBy is the division operator, `/`.
	BinaryOperatorDividedBy
	// BinaryOperatorModulo is the modulo operator, `%`.
	BinaryOperatorModulo
)

// Name returns the English name of the operator, such as "plus".
//
// Matches Dart: BinaryOperator.name
func (op BinaryOperator) Name() string {
	switch op {
	case BinaryOperatorSingleEquals:
		return "single equals"
	case BinaryOperatorOr:
		return "or"
	case BinaryOperatorAnd:
		return "and"
	case BinaryOperatorEquals:
		return "equals"
	case BinaryOperatorNotEquals:
		return "not equals"
	case BinaryOperatorGreaterThan:
		return "greater than"
	case BinaryOperatorGreaterThanOrEquals:
		return "greater than or equals"
	case BinaryOperatorLessThan:
		return "less than"
	case BinaryOperatorLessThanOrEquals:
		return "less than or equals"
	case BinaryOperatorPlus:
		return "plus"
	case BinaryOperatorMinus:
		return "minus"
	case BinaryOperatorTimes:
		return "times"
	case BinaryOperatorDividedBy:
		return "divided by"
	case BinaryOperatorModulo:
		return "modulo"
	default:
		return fmt.Sprintf("BinaryOperator(%d)", op)
	}
}

// String returns the English name of the operator.
//
// Matches Dart: BinaryOperator.toString
func (op BinaryOperator) String() string { return op.Name() }

// Precedence returns how tightly the operator binds: higher binds tighter.
// Single equals is 0, or/and are 1/2, equality 3, comparisons 4, additive 5,
// multiplicative 6.
//
// Matches Dart: BinaryOperator.precedence
func (op BinaryOperator) Precedence() int {
	switch op {
	case BinaryOperatorSingleEquals:
		return 0
	case BinaryOperatorOr:
		return 1
	case BinaryOperatorAnd:
		return 2
	case BinaryOperatorEquals, BinaryOperatorNotEquals:
		return 3
	case BinaryOperatorGreaterThan, BinaryOperatorGreaterThanOrEquals, BinaryOperatorLessThan, BinaryOperatorLessThanOrEquals:
		return 4
	case BinaryOperatorPlus, BinaryOperatorMinus:
		return 5
	case BinaryOperatorTimes, BinaryOperatorDividedBy, BinaryOperatorModulo:
		return 6
	default:
		return 0
	}
}

// OperatorSyntax returns the Sass syntax for the operator, such as "==" or
// "and".
//
// Matches Dart: BinaryOperator.operator
func (op BinaryOperator) OperatorSyntax() string {
	switch op {
	case BinaryOperatorSingleEquals:
		return "="
	case BinaryOperatorOr:
		return "or"
	case BinaryOperatorAnd:
		return "and"
	case BinaryOperatorEquals:
		return "=="
	case BinaryOperatorNotEquals:
		return "!="
	case BinaryOperatorGreaterThan:
		return ">"
	case BinaryOperatorGreaterThanOrEquals:
		return ">="
	case BinaryOperatorLessThan:
		return "<"
	case BinaryOperatorLessThanOrEquals:
		return "<="
	case BinaryOperatorPlus:
		return "+"
	case BinaryOperatorMinus:
		return "-"
	case BinaryOperatorTimes:
		return "*"
	case BinaryOperatorDividedBy:
		return "/"
	case BinaryOperatorModulo:
		return "%"
	default:
		return "?"
	}
}

// IsAssociative reports whether the operator has the associative property:
// or, and, plus, and times.
//
// Matches Dart: BinaryOperator.isAssociative
func (op BinaryOperator) IsAssociative() bool {
	switch op {
	case BinaryOperatorOr, BinaryOperatorAnd, BinaryOperatorPlus, BinaryOperatorTimes:
		return true
	default:
		return false
	}
}

// BinaryOperationExpression is a binary operation, as in `1 + 2` or
// `$this and $other`.
type BinaryOperationExpression struct {
	// Operator is the operator being invoked.
	Operator BinaryOperator
	// Left is the left-hand operand.
	Left Expression
	// Right is the right-hand operand.
	Right Expression
	// allowsSlash records whether a dividedBy operation may be read as
	// slash-separated numbers rather than division.
	//
	// Matches Dart: BinaryOperationExpression.allowsSlash (internal).
	allowsSlash bool
}

// NewBinaryOperationExpression creates a binary operation with the given
// operator and operands. The result never allows slash interpretation.
//
// Matches Dart: BinaryOperationExpression.new
func NewBinaryOperationExpression(operator BinaryOperator, left, right Expression) *BinaryOperationExpression {
	return &BinaryOperationExpression{
		Operator:    operator,
		Left:        left,
		Right:       right,
		allowsSlash: false,
	}
}

// NewBinaryOperationExpressionSlash creates a dividedBy operation that may be
// interpreted as slash-separated numbers.
//
// Matches Dart: BinaryOperationExpression.slash (internal)
func NewBinaryOperationExpressionSlash(left, right Expression) *BinaryOperationExpression {
	return &BinaryOperationExpression{
		Operator:    BinaryOperatorDividedBy,
		Left:        left,
		Right:       right,
		allowsSlash: true,
	}
}

// AllowsSlash reports whether this dividedBy operation may be interpreted
// as slash-separated numbers.
func (e *BinaryOperationExpression) AllowsSlash() bool { return e.allowsSlash }

func (e *BinaryOperationExpression) Span() (sasscommon.FileSpan, error) {
	// Avoid building intermediate spans for chained binary operations by
	// descending to the left-most and right-most operands first.
	left := e.Left
	for {
		if binOp, ok := any(left).(*BinaryOperationExpression); ok {
			left = binOp.Left
		} else {
			break
		}
	}

	right := e.Right
	for {
		if binOp, ok := any(right).(*BinaryOperationExpression); ok {
			right = binOp.Right
		} else {
			break
		}
	}

	leftSpan, err := left.Span()
	if err != nil {
		return nil, err
	}
	rightSpan, err := right.Span()
	if err != nil {
		return nil, err
	}
	return leftSpan.Expand(rightSpan)
}

// OperatorSpan returns the span covering only the operator: the gap between
// the end of the left operand and the start of the right operand when both
// share a source file, falling back to the full expression span otherwise.
//
// Matches Dart: BinaryOperationExpression.operatorSpan (internal)
func (e *BinaryOperationExpression) OperatorSpan() (sasscommon.FileSpan, error) {
	leftSpan, err := e.Left.Span()
	if err != nil {
		return nil, err
	}
	rightSpan, err := e.Right.Span()
	if err != nil {
		return nil, err
	}

	leftURL, err := leftSpan.SourceURL()
	if err != nil {
		return nil, err
	}
	rightURL, err := rightSpan.SourceURL()
	if err != nil {
		return nil, err
	}

	if sameSourceURL(leftURL, rightURL) {
		leftEnd, err := leftSpan.EndLocation()
		if err != nil {
			return nil, err
		}
		rightStart, err := rightSpan.StartLocation()
		if err != nil {
			return nil, err
		}
		if leftEnd.Offset < rightStart.Offset {
			file, err := leftSpan.File()
			if err != nil {
				return nil, err
			}
			return sasscommon.NewSimpleFileSpan(
				file,
				leftEnd.Offset,
				rightStart.Offset,
			).Trim()
		}
	}
	return e.Span()
}

// sameSourceURL reports whether two operand spans come from the same source,
// treating two nil URLs as the same file.
func sameSourceURL(a, b *url.URL) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.String() == b.String()
}

func (e *BinaryOperationExpression) SourceInterpolation() *Interpolation { return nil }
func (e *BinaryOperationExpression) IsExpression()                       {}
func (e *BinaryOperationExpression) IsSassNode()                         {}
func (e *BinaryOperationExpression) IsAstNode()                          {}

// String renders the operation with minimal parentheses: an operand is
// wrapped only when a nested binary operation would otherwise bind
// differently (lower precedence on the left; lower-or-equal, non-associative
// repeats on the right) or when a bracketless multi-element list would
// parse ambiguously.
func (e *BinaryOperationExpression) String() (string, error) {
	var buf strings.Builder

	leftNeedsParens := false
	if leftBinOp, ok := any(e.Left).(*BinaryOperationExpression); ok {
		leftNeedsParens = leftBinOp.Operator.Precedence() < e.Operator.Precedence()
	} else if leftList, ok := any(e.Left).(*ListExpression); ok {
		if !leftList.HasBrackets && len(leftList.Contents) >= 2 {
			leftNeedsParens = true
		}
	}
	if leftNeedsParens {
		buf.WriteRune('(')
	}
	leftStr, err := e.Left.String()
	if err != nil {
		return "", err
	}
	buf.WriteString(leftStr)
	if leftNeedsParens {
		buf.WriteRune(')')
	}

	buf.WriteRune(' ')
	buf.WriteString(e.Operator.OperatorSyntax())
	buf.WriteRune(' ')

	rightNeedsParens := false
	if rightBinOp, ok := any(e.Right).(*BinaryOperationExpression); ok {
		rightNeedsParens = rightBinOp.Operator.Precedence() <= e.Operator.Precedence() &&
			!(rightBinOp.Operator == e.Operator && rightBinOp.Operator.IsAssociative())
	} else if rightList, ok := any(e.Right).(*ListExpression); ok {
		if !rightList.HasBrackets && len(rightList.Contents) >= 2 {
			rightNeedsParens = true
		}
	}
	if rightNeedsParens {
		buf.WriteRune('(')
	}
	rightStr, err := e.Right.String()
	if err != nil {
		return "", err
	}
	buf.WriteString(rightStr)
	if rightNeedsParens {
		buf.WriteRune(')')
	}

	return buf.String(), nil
}
