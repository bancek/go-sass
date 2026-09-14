// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/*.dart (Accept methods)

// Each group below forwards one expression type to its matching Visit
// method for all four result shapes (Value, bool, struct{}, Expression).
// Dart documents accept() per class as "calls the appropriate visit method";
// those docs are collapsed onto Expression.AcceptBool (see
// sass_expression.go) instead of repeated seventy-two times here.
//
// The IsCalculationSafe/IsPlainCss methods at the bottom route through the
// predicate visitors, matching Dart's isCalculationSafe/isPlainCss getters.

func (e *BinaryOperationExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitBinaryOperationExpression(e)
}

func (e *BinaryOperationExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitBinaryOperationExpression(e)
}

func (e *BinaryOperationExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitBinaryOperationExpression(e)
}

func (e *BinaryOperationExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitBinaryOperationExpression(e)
}

func (e *BooleanExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitBooleanExpression(e)
}

func (e *BooleanExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitBooleanExpression(e)
}

func (e *BooleanExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitBooleanExpression(e)
}

func (e *BooleanExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitBooleanExpression(e)
}

func (e *ColorExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitColorExpression(e)
}

func (e *ColorExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitColorExpression(e)
}

func (e *ColorExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitColorExpression(e)
}

func (e *ColorExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitColorExpression(e)
}

func (e *InterpolatedFunctionExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitInterpolatedFunctionExpression(e)
}

func (e *InterpolatedFunctionExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitInterpolatedFunctionExpression(e)
}

func (e *InterpolatedFunctionExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitInterpolatedFunctionExpression(e)
}

func (e *InterpolatedFunctionExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitInterpolatedFunctionExpression(e)
}

func (e *FunctionExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitFunctionExpression(e)
}

func (e *FunctionExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitFunctionExpression(e)
}

func (e *FunctionExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitFunctionExpression(e)
}

func (e *FunctionExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitFunctionExpression(e)
}

func (e *IfExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitIfExpression(e)
}

func (e *IfExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfExpression(e)
}

func (e *IfExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfExpression(e)
}

func (e *IfExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitIfExpression(e)
}

func (e *LegacyIfExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitLegacyIfExpression(e)
}

func (e *LegacyIfExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitLegacyIfExpression(e)
}

func (e *LegacyIfExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitLegacyIfExpression(e)
}

func (e *LegacyIfExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitLegacyIfExpression(e)
}

func (e *ListExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitListExpression(e)
}

func (e *ListExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitListExpression(e)
}

func (e *ListExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitListExpression(e)
}

func (e *ListExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitListExpression(e)
}

func (e *MapExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitMapExpression(e)
}

func (e *MapExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitMapExpression(e)
}

func (e *MapExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitMapExpression(e)
}

func (e *MapExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitMapExpression(e)
}

func (e *NullExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitNullExpression(e)
}

func (e *NullExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitNullExpression(e)
}

func (e *NullExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitNullExpression(e)
}

func (e *NullExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitNullExpression(e)
}

func (e *NumberExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitNumberExpression(e)
}

func (e *NumberExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitNumberExpression(e)
}

func (e *NumberExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitNumberExpression(e)
}

func (e *NumberExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitNumberExpression(e)
}

func (e *ParenthesizedExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitParenthesizedExpression(e)
}

func (e *ParenthesizedExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitParenthesizedExpression(e)
}

func (e *ParenthesizedExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitParenthesizedExpression(e)
}

func (e *ParenthesizedExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitParenthesizedExpression(e)
}

func (e *SelectorExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitSelectorExpression(e)
}

func (e *SelectorExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitSelectorExpression(e)
}

func (e *SelectorExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitSelectorExpression(e)
}

func (e *SelectorExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitSelectorExpression(e)
}

func (e *StringExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitStringExpression(e)
}

func (e *StringExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitStringExpression(e)
}

func (e *StringExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitStringExpression(e)
}

func (e *StringExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitStringExpression(e)
}

func (e *SupportsExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitSupportsExpression(e)
}

func (e *SupportsExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitSupportsExpression(e)
}

func (e *SupportsExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitSupportsExpression(e)
}

func (e *SupportsExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitSupportsExpression(e)
}

func (e *UnaryOperationExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitUnaryOperationExpression(e)
}

func (e *UnaryOperationExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitUnaryOperationExpression(e)
}

func (e *UnaryOperationExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitUnaryOperationExpression(e)
}

func (e *UnaryOperationExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitUnaryOperationExpression(e)
}

func (e *ValueExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitValueExpression(e)
}

func (e *ValueExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitValueExpression(e)
}

func (e *ValueExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitValueExpression(e)
}

func (e *ValueExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitValueExpression(e)
}

func (e *VariableExpression) AcceptValue(v ExpressionVisitor[Value]) (Value, error) {
	return v.VisitVariableExpression(e)
}

func (e *VariableExpression) AcceptBool(v ExpressionVisitor[bool]) (bool, error) {
	return v.VisitVariableExpression(e)
}

func (e *VariableExpression) AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitVariableExpression(e)
}

func (e *VariableExpression) AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error) {
	return v.VisitVariableExpression(e)
}

// IsCalculationSafe and IsPlainCss implementations.
//
// Matches Dart: Expression.isCalculationSafe, Expression.isPlainCss

func (e *BinaryOperationExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *BinaryOperationExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *BooleanExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *BooleanExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *ColorExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *ColorExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *FunctionExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *FunctionExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *IfExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *IfExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *LegacyIfExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *LegacyIfExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *ListExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *ListExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *MapExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *MapExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *NullExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *NullExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *NumberExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *NumberExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *ParenthesizedExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *ParenthesizedExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *SelectorExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *SelectorExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *StringExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *StringExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *SupportsExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *SupportsExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *UnaryOperationExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *UnaryOperationExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *ValueExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *ValueExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
func (e *VariableExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *VariableExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}

func (e *InterpolatedFunctionExpression) IsCalculationSafe() (bool, error) {
	return e.AcceptBool(NewIsCalculationSafeVisitor())
}
func (e *InterpolatedFunctionExpression) IsPlainCss(allowInterpolation bool) (bool, error) {
	return e.AcceptBool(NewIsPlainCssVisitor(allowInterpolation))
}
