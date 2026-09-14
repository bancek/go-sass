// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/expression.dart

// ExpressionVisitor traverses SassScript expression AST nodes, with one
// method per expression type. The type parameter selects the result: boolean
// predicates, value-producing evaluation, effect-only walks, and
// expression-rewriting passes each instantiate it with a different result
// type.
//
// Each Visit method is the collapsed owner of the corresponding Dart
// per-class accept() ("calls the appropriate visit method"); individual
// expression Accept methods forward here.
type ExpressionVisitor[T any] interface {
	VisitBinaryOperationExpression(*BinaryOperationExpression) (T, error)
	VisitBooleanExpression(*BooleanExpression) (T, error)
	VisitColorExpression(*ColorExpression) (T, error)
	VisitInterpolatedFunctionExpression(*InterpolatedFunctionExpression) (T, error)
	VisitFunctionExpression(*FunctionExpression) (T, error)
	VisitIfExpression(*IfExpression) (T, error)
	VisitLegacyIfExpression(*LegacyIfExpression) (T, error)
	VisitListExpression(*ListExpression) (T, error)
	VisitMapExpression(*MapExpression) (T, error)
	VisitNullExpression(*NullExpression) (T, error)
	VisitNumberExpression(*NumberExpression) (T, error)
	VisitParenthesizedExpression(*ParenthesizedExpression) (T, error)
	VisitSelectorExpression(*SelectorExpression) (T, error)
	VisitStringExpression(*StringExpression) (T, error)
	VisitSupportsExpression(*SupportsExpression) (T, error)
	VisitUnaryOperationExpression(*UnaryOperationExpression) (T, error)
	VisitValueExpression(*ValueExpression) (T, error)
	VisitVariableExpression(*VariableExpression) (T, error)
}
