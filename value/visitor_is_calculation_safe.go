// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/is_calculation_safe.dart

import (
	"strings"
)

// IsCalculationSafeVisitor determines whether an expression is valid in a
// calculation context.
//
// Use it through Expression.isCalculationSafe. It deliberately avoids a
// search visitor with a default-true fallback, so adding a new expression
// type without updating this visitor fails closed instead of silently
// passing.
//
// Matches Dart: IsCalculationSafeVisitor
// (lib/src/visitor/is_calculation_safe.dart)
type IsCalculationSafeVisitor struct{}

// NewIsCalculationSafeVisitor creates a calculation-safety checker.
func NewIsCalculationSafeVisitor() *IsCalculationSafeVisitor {
	return &IsCalculationSafeVisitor{}
}

// VisitBinaryOperationExpression reports true for the four arithmetic
// operators with calculation-safe operands on both sides; every other
// operator is unsafe in a calculation.
func (v *IsCalculationSafeVisitor) VisitBinaryOperationExpression(node *BinaryOperationExpression) (bool, error) {
	switch node.Operator {
	case BinaryOperatorTimes, BinaryOperatorDividedBy,
		BinaryOperatorPlus, BinaryOperatorMinus:
		safe, err := node.Left.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !safe {
			return false, nil
		}
		return node.Right.AcceptBool(v)
	default:
		return false, nil
	}
}

// VisitBooleanExpression always reports false: booleans are not valid in
// a calculation.
func (v *IsCalculationSafeVisitor) VisitBooleanExpression(node *BooleanExpression) (bool, error) {
	return false, nil
}

// VisitColorExpression always reports false: colors are not valid in a
// calculation.
func (v *IsCalculationSafeVisitor) VisitColorExpression(node *ColorExpression) (bool, error) {
	return false, nil
}

// VisitFunctionExpression always reports true: any function call may be
// valid in a calculation (unknown functions are left for runtime).
func (v *IsCalculationSafeVisitor) VisitFunctionExpression(node *FunctionExpression) (bool, error) {
	return true, nil
}

// VisitIfExpression always reports true: conditional expressions are
// allowed in a calculation.
func (v *IsCalculationSafeVisitor) VisitIfExpression(node *IfExpression) (bool, error) {
	return true, nil
}

// VisitInterpolatedFunctionExpression always reports true: like plain
// function calls, interpolated calls may be valid in a calculation.
func (v *IsCalculationSafeVisitor) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (bool, error) {
	return true, nil
}

// VisitLegacyIfExpression always reports true: like if(), the legacy
// conditional is allowed in a calculation.
func (v *IsCalculationSafeVisitor) VisitLegacyIfExpression(node *LegacyIfExpression) (bool, error) {
	return true, nil
}

// VisitListExpression reports true for multi-element unbracketed
// space-separated lists whose elements are all calculation-safe; every
// other list shape is unsafe in a calculation.
func (v *IsCalculationSafeVisitor) VisitListExpression(node *ListExpression) (bool, error) {
	if node.Separator != ListSeparatorSpace {
		return false, nil
	}
	if node.HasBrackets {
		return false, nil
	}
	if len(node.Contents) <= 1 {
		return false, nil
	}
	for _, expr := range node.Contents {
		safe, err := expr.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !safe {
			return false, nil
		}
	}
	return true, nil
}

// VisitMapExpression always reports false: maps are not valid in a
// calculation.
func (v *IsCalculationSafeVisitor) VisitMapExpression(node *MapExpression) (bool, error) {
	return false, nil
}

// VisitNullExpression always reports false: null is not valid in a
// calculation.
func (v *IsCalculationSafeVisitor) VisitNullExpression(node *NullExpression) (bool, error) {
	return false, nil
}

// VisitNumberExpression always reports true: numbers are valid in a
// calculation.
func (v *IsCalculationSafeVisitor) VisitNumberExpression(node *NumberExpression) (bool, error) {
	return true, nil
}

// VisitParenthesizedExpression defers to the inner expression.
func (v *IsCalculationSafeVisitor) VisitParenthesizedExpression(node *ParenthesizedExpression) (bool, error) {
	return node.Expression.AcceptBool(v)
}

// VisitSelectorExpression always reports false: selectors are not valid
// in a calculation.
func (v *IsCalculationSafeVisitor) VisitSelectorExpression(node *SelectorExpression) (bool, error) {
	return false, nil
}

// VisitStringExpression reports true for unquoted identifier-like strings
// only. Quoted strings are never safe, and several unquoted shapes that
// parse as strings are excluded without a full identifier parse (cheaper
// than validating): !important markers, ID-style identifiers, unicode
// ranges (a plus in second position), and url() calls (a paren in fourth
// position).
func (v *IsCalculationSafeVisitor) VisitStringExpression(node *StringExpression) (bool, error) {
	if node.HasQuotes {
		return false, nil
	}

	// Exclude non-identifier constructs that are parsed as [StringExpression]s.
	// We could just check if they parse as valid identifiers, but this is
	// cheaper.
	text := node.Text.InitialPlain()
	return !strings.HasPrefix(text, "!") &&
		!strings.HasPrefix(text, "#") &&
		(len(text) <= 1 || text[1] != '+') &&
		(len(text) <= 3 || text[3] != '('), nil
}

// VisitSupportsExpression always reports false: supports conditions are
// not valid in a calculation.
func (v *IsCalculationSafeVisitor) VisitSupportsExpression(node *SupportsExpression) (bool, error) {
	return false, nil
}

// VisitUnaryOperationExpression always reports false: unary operations
// are not valid in a calculation.
func (v *IsCalculationSafeVisitor) VisitUnaryOperationExpression(node *UnaryOperationExpression) (bool, error) {
	return false, nil
}

// VisitValueExpression always reports false: an eager Sass value is not
// known to be calculation-safe.
func (v *IsCalculationSafeVisitor) VisitValueExpression(node *ValueExpression) (bool, error) {
	return false, nil
}

// VisitVariableExpression always reports true: variables may hold
// calculation-safe values (checked at runtime).
func (v *IsCalculationSafeVisitor) VisitVariableExpression(node *VariableExpression) (bool, error) {
	return true, nil
}

var _ ExpressionVisitor[bool] = (*IsCalculationSafeVisitor)(nil)
