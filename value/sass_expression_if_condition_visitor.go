// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/if_condition_expression.dart

// IfConditionExpressionVisitor traverses if() condition expressions, with
// one method per condition type. The type parameter selects the result, as
// with ExpressionVisitor.
//
// Each Visit method is the collapsed owner of the corresponding Dart
// per-class accept() ("calls the appropriate visit method"); the Accept*
// methods on each condition forward here and are not documented individually.
type IfConditionExpressionVisitor[T any] interface {
	VisitIfConditionParenthesized(*IfConditionParenthesized) (T, error)
	VisitIfConditionNegation(*IfConditionNegation) (T, error)
	VisitIfConditionOperation(*IfConditionOperation) (T, error)
	VisitIfConditionFunction(*IfConditionFunction) (T, error)
	VisitIfConditionSass(*IfConditionSass) (T, error)
	VisitIfConditionRaw(*IfConditionRaw) (T, error)
}

func (c *IfConditionParenthesized) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionParenthesized(c)
}

func (c *IfConditionParenthesized) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionParenthesized(c)
}

func (c *IfConditionParenthesized) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionParenthesized(c)
}

func (c *IfConditionParenthesized) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionParenthesized(c)
}

func (c *IfConditionNegation) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionNegation(c)
}

func (c *IfConditionNegation) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionNegation(c)
}

func (c *IfConditionNegation) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionNegation(c)
}

func (c *IfConditionNegation) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionNegation(c)
}

func (c *IfConditionOperation) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionOperation(c)
}

func (c *IfConditionOperation) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionOperation(c)
}

func (c *IfConditionOperation) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionOperation(c)
}

func (c *IfConditionOperation) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionOperation(c)
}

func (c *IfConditionFunction) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionFunction(c)
}

func (c *IfConditionFunction) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionFunction(c)
}

func (c *IfConditionFunction) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionFunction(c)
}

func (c *IfConditionFunction) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionFunction(c)
}

func (c *IfConditionSass) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionSass(c)
}

func (c *IfConditionSass) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionSass(c)
}

func (c *IfConditionSass) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionSass(c)
}

func (c *IfConditionSass) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionSass(c)
}

func (c *IfConditionRaw) AcceptAny(v IfConditionExpressionVisitor[any]) (any, error) {
	return v.VisitIfConditionRaw(c)
}

func (c *IfConditionRaw) AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error) {
	return v.VisitIfConditionRaw(c)
}

func (c *IfConditionRaw) AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfConditionRaw(c)
}

func (c *IfConditionRaw) AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error) {
	return v.VisitIfConditionRaw(c)
}
