// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/replace_expression.dart

import (
	"fmt"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

// ReplaceExpressionVisitor recursively traverses each expression in a
// SassScript AST and rebuilds it with the values returned by nested
// recursion. Leaf expressions (booleans, colors, nulls, numbers, selectors,
// values, variables) are returned as-is; composite nodes are rebuilt from
// their rewritten children.
//
// Beyond the ExpressionVisitor methods it offers three overridable helpers
// — visitArgumentList, visitSupportsCondition, visitInterpolation — that the
// default visits call to cover argument lists, supports conditions, and
// interpolations.
//
// Matches Dart: ReplaceExpressionVisitor mixin. The visitor field works
// around Go embedding: a method calling itself would otherwise keep the
// embedded receiver and miss overrides, so recursion dispatches through this
// field set to the outer struct, matching Dart's mixin this-resolution.
type ReplaceExpressionVisitor struct {
	// visitor is the effective visitor used for recursive expression dispatch.
	//
	// Go embedding promotes methods but self-calls keep the embedded type as
	// receiver. When a method on ReplaceExpressionVisitor calls a method on
	// itself, the receiver is always *ReplaceExpressionVisitor — even if an
	// outer struct (like makeExpressionCalculationSafe) embeds it and overrides
	// methods. Setting [visitor] to the outer struct routes recursion through
	// it, so overrides are found.
	//
	// This matches Dart's mixin semantics: in Dart, `this` inside a mixin
	// method resolves to the concrete class.
	//
	// If nil, recursion uses v (the receiver).
	visitor ExpressionVisitor[Expression]
}

// replace recurses into expr through the effective visitor, so overrides on
// the outer struct are honored. The visitor field must be set before use.
//
// Matches Dart: node.accept(this) with mixin this-resolution
func (v *ReplaceExpressionVisitor) replace(expr Expression) (Expression, error) {
	vis := v.visitor
	if vis == nil {
		return nil, fmt.Errorf("BUG: ReplaceExpressionVisitor.replace called with nil visitor")
	}
	return expr.AcceptExpr(vis)
}

// replaceCondition recurses into an if() condition through this visitor.
func (v *ReplaceExpressionVisitor) replaceCondition(cond IfConditionExpression) (IfConditionExpression, error) {
	return cond.AcceptIfConditionExpression(v)
}

// VisitBinaryOperationExpression rebuilds the operation from its rewritten
// operands, dropping slash-interpretation (matching Dart's plain
// constructor call).
func (v *ReplaceExpressionVisitor) VisitBinaryOperationExpression(node *BinaryOperationExpression) (Expression, error) {
	left, err := v.replace(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := v.replace(node.Right)
	if err != nil {
		return nil, err
	}
	return &BinaryOperationExpression{
		Operator:    node.Operator,
		Left:        left,
		Right:       right,
		allowsSlash: false,
	}, nil
}

// VisitBooleanExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitBooleanExpression(node *BooleanExpression) (Expression, error) {
	return node, nil
}

// VisitColorExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitColorExpression(node *ColorExpression) (Expression, error) {
	return node, nil
}

// VisitFunctionExpression rebuilds the call from its rewritten argument
// list, preserving the original underscore spelling and namespace.
func (v *ReplaceExpressionVisitor) VisitFunctionExpression(node *FunctionExpression) (Expression, error) {
	args, err := v.visitArgumentList(node.Arguments())
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewFunctionExpression(
		node.OriginalName,
		args,
		span,
		node.Namespace,
	), nil
}

// VisitInterpolatedFunctionExpression rebuilds the call from its rewritten
// name interpolation and argument list.
func (v *ReplaceExpressionVisitor) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (Expression, error) {
	name, err := v.visitInterpolation(node.Name)
	if err != nil {
		return nil, err
	}
	args, err := v.visitArgumentList(node.Arguments())
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewInterpolatedFunctionExpression(
		name,
		args,
		span,
	), nil
}

// VisitLegacyIfExpression rebuilds the legacy if() from its rewritten
// argument list.
func (v *ReplaceExpressionVisitor) VisitLegacyIfExpression(node *LegacyIfExpression) (Expression, error) {
	args, err := v.visitArgumentList(node.Arguments())
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewLegacyIfExpression(args, span), nil
}

// VisitIfExpression recurses into each branch's condition and expression.
//
// Matches Dart: IfExpression dispatch via the ReplaceExpressionVisitor mixin,
// and Rust: replace_expression.rs visit_if.
func (v *ReplaceExpressionVisitor) VisitIfExpression(node *IfExpression) (Expression, error) {
	newBranches := make([]IfBranch, len(node.Branches))
	for i, branch := range node.Branches {
		if branch.Condition != nil {
			cond, err := v.replaceCondition(branch.Condition)
			if err != nil {
				return nil, err
			}
			newBranches[i].Condition = cond
		}
		expr, err := v.replace(branch.Expression)
		if err != nil {
			return nil, err
		}
		newBranches[i].Expression = expr
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewIfExpression(newBranches, span)
}

// VisitListExpression rebuilds the list from its rewritten contents,
// preserving separator and brackets.
func (v *ReplaceExpressionVisitor) VisitListExpression(node *ListExpression) (Expression, error) {
	newContents := make([]Expression, len(node.Contents))
	for i, item := range node.Contents {
		var err error
		newContents[i], err = v.replace(item)
		if err != nil {
			return nil, err
		}
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewListExpression(newContents, node.Separator, span, node.HasBrackets), nil
}

// VisitMapExpression rebuilds the map from its rewritten keys and values.
func (v *ReplaceExpressionVisitor) VisitMapExpression(node *MapExpression) (Expression, error) {
	newPairs := make([]struct {
		Key   Expression
		Value Expression
	}, len(node.Pairs))
	for i, pair := range node.Pairs {
		key, err := v.replace(pair.Key)
		if err != nil {
			return nil, err
		}
		value, err := v.replace(pair.Value)
		if err != nil {
			return nil, err
		}
		newPairs[i] = struct {
			Key   Expression
			Value Expression
		}{
			Key:   key,
			Value: value,
		}
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewMapExpression(newPairs, span), nil
}

// VisitNullExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitNullExpression(node *NullExpression) (Expression, error) {
	return node, nil
}

// VisitNumberExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitNumberExpression(node *NumberExpression) (Expression, error) {
	return node, nil
}

// VisitParenthesizedExpression rebuilds the wrapper from its rewritten
// inner expression.
func (v *ReplaceExpressionVisitor) VisitParenthesizedExpression(node *ParenthesizedExpression) (Expression, error) {
	expr, err := v.replace(node.Expression)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewParenthesizedExpression(expr, span), nil
}

// VisitSelectorExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitSelectorExpression(node *SelectorExpression) (Expression, error) {
	return node, nil
}

// VisitStringExpression rebuilds the string from its rewritten text
// interpolation, preserving quoting.
func (v *ReplaceExpressionVisitor) VisitStringExpression(node *StringExpression) (Expression, error) {
	text, err := v.visitInterpolation(node.Text)
	if err != nil {
		return nil, err
	}
	return NewStringExpression(text, node.HasQuotes), nil
}

// VisitSupportsExpression rebuilds the expression from its rewritten
// supports condition.
func (v *ReplaceExpressionVisitor) VisitSupportsExpression(node *SupportsExpression) (Expression, error) {
	cond, err := v.visitSupportsCondition(node.Condition)
	if err != nil {
		return nil, err
	}
	return NewSupportsExpression(cond), nil
}

// VisitUnaryOperationExpression rebuilds the operation from its rewritten
// operand.
func (v *ReplaceExpressionVisitor) VisitUnaryOperationExpression(node *UnaryOperationExpression) (Expression, error) {
	operand, err := v.replace(node.Operand)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewUnaryOperationExpression(
		node.Operator,
		operand,
		span,
	), nil
}

// VisitValueExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitValueExpression(node *ValueExpression) (Expression, error) {
	return node, nil
}

// VisitVariableExpression returns leaf nodes unchanged.
func (v *ReplaceExpressionVisitor) VisitVariableExpression(node *VariableExpression) (Expression, error) {
	return node, nil
}

// if() condition expressions: each rebuilds its condition node from
// rewritten children.

// VisitIfConditionParenthesized rebuilds the wrapper from its rewritten
// inner condition.
func (v *ReplaceExpressionVisitor) VisitIfConditionParenthesized(node *IfConditionParenthesized) (IfConditionExpression, error) {
	cond, err := v.replaceCondition(node.Expression)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewIfConditionParenthesized(cond, span), nil
}

// VisitIfConditionNegation rebuilds the negation from its rewritten inner
// condition.
func (v *ReplaceExpressionVisitor) VisitIfConditionNegation(node *IfConditionNegation) (IfConditionExpression, error) {
	cond, err := v.replaceCondition(node.Expression)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewIfConditionNegation(cond, span), nil
}

// VisitIfConditionOperation rebuilds the and/or chain from its rewritten
// conditions.
func (v *ReplaceExpressionVisitor) VisitIfConditionOperation(node *IfConditionOperation) (IfConditionExpression, error) {
	newExprs := make([]IfConditionExpression, len(node.Expressions))
	for i, expr := range node.Expressions {
		var err error
		newExprs[i], err = v.replaceCondition(expr)
		if err != nil {
			return nil, err
		}
	}
	result, err := NewIfConditionOperation(newExprs, node.Op)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// VisitIfConditionFunction rebuilds the function-style condition from its
// rewritten name and argument interpolations.
func (v *ReplaceExpressionVisitor) VisitIfConditionFunction(node *IfConditionFunction) (IfConditionExpression, error) {
	name, err := v.visitInterpolation(node.Name)
	if err != nil {
		return nil, err
	}
	args, err := v.visitInterpolation(node.Arguments)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewIfConditionFunction(
		name,
		args,
		span,
	), nil
}

// VisitIfConditionSass rebuilds the sass() condition from its rewritten
// inner expression.
func (v *ReplaceExpressionVisitor) VisitIfConditionSass(node *IfConditionSass) (IfConditionExpression, error) {
	expr, err := v.replace(node.Expression)
	if err != nil {
		return nil, err
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return NewIfConditionSass(expr, span), nil
}

// VisitIfConditionRaw rebuilds the raw-text condition from its rewritten
// text interpolation.
func (v *ReplaceExpressionVisitor) VisitIfConditionRaw(node *IfConditionRaw) (IfConditionExpression, error) {
	text, err := v.visitInterpolation(node.Text)
	if err != nil {
		return nil, err
	}
	return NewIfConditionRaw(text), nil
}

// Protected helpers, overridable by embedding visitors.

// visitArgumentList rewrites every positional, named, rest, and keyword-rest
// argument while preserving names, name spans, and the invocation span.
//
// Matches Dart: ReplaceExpressionVisitor.visitArgumentList (protected)
func (v *ReplaceExpressionVisitor) visitArgumentList(invocation *ArgumentList) (*ArgumentList, error) {
	positional := make([]Expression, len(invocation.Positional))
	for i, expr := range invocation.Positional {
		var err error
		positional[i], err = v.replace(expr)
		if err != nil {
			return nil, err
		}
	}
	named := orderedmap.NewWithCapacity[string, Expression](invocation.Named.Len())
	namedSpans := make(map[string]sasscommon.FileSpan, len(invocation.NamedSpans))
	for name, expr := range invocation.Named.Entries() {
		replaced, err := v.replace(expr)
		if err != nil {
			return nil, err
		}
		named.Put(name, replaced)
		if span, ok := invocation.NamedSpans[name]; ok {
			namedSpans[name] = span
		}
	}
	var rest Expression
	if invocation.Rest != nil {
		var err error
		rest, err = v.replace(invocation.Rest)
		if err != nil {
			return nil, err
		}
	}
	var keywordRest Expression
	if invocation.KeywordRest != nil {
		var err error
		keywordRest, err = v.replace(invocation.KeywordRest)
		if err != nil {
			return nil, err
		}
	}
	span, err := invocation.Span()
	if err != nil {
		return nil, err
	}
	return NewArgumentList(positional, named, namedSpans, span, rest, keywordRest), nil
}

// visitSupportsCondition rewrites each side of operations and negations,
// the embedded expression of interpolations, and the name/value of
// declarations; unknown conditions are a bug.
//
// Matches Dart: ReplaceExpressionVisitor.visitSupportsCondition (protected)
func (v *ReplaceExpressionVisitor) visitSupportsCondition(condition SupportsCondition) (SupportsCondition, error) {
	if op, ok := condition.(*SupportsOperation); ok {
		left, err := v.visitSupportsCondition(op.Left)
		if err != nil {
			return nil, err
		}
		right, err := v.visitSupportsCondition(op.Right)
		if err != nil {
			return nil, err
		}
		span, err := op.Span()
		if err != nil {
			return nil, err
		}
		return NewSupportsOperation(
			left,
			right,
			op.Operator,
			span,
		), nil
	} else if neg, ok := condition.(*SupportsNegation); ok {
		inner, err := v.visitSupportsCondition(neg.Condition)
		if err != nil {
			return nil, err
		}
		span, err := neg.Span()
		if err != nil {
			return nil, err
		}
		return NewSupportsNegation(
			inner,
			span,
		), nil
	} else if interp, ok := condition.(*SupportsInterpolation); ok {
		replaced, err := v.replace(interp.Expression)
		if err != nil {
			return nil, err
		}
		span, err := interp.Span()
		if err != nil {
			return nil, err
		}
		return NewSupportsInterpolation(replaced, span), nil
	} else if decl, ok := condition.(*SupportsDeclaration); ok {
		replacedName, err := v.replace(decl.Name)
		if err != nil {
			return nil, err
		}
		replacedValue, err := v.replace(decl.Value)
		if err != nil {
			return nil, err
		}
		span, err := decl.Span()
		if err != nil {
			return nil, err
		}
		return NewSupportsDeclaration(replacedName, replacedValue, span), nil
	}
	return nil, &sasscommon.ArgumentError{
		Message: fmt.Sprintf("BUG: Unknown SupportsCondition %T", condition),
	}
}

// visitInterpolation rewrites embedded expressions in place and copies
// plain-text segments through, preserving per-element spans.
//
// Matches Dart: ReplaceExpressionVisitor.visitInterpolation (protected)
func (v *ReplaceExpressionVisitor) visitInterpolation(in *Interpolation) (*Interpolation, error) {
	newContents := make([]any, len(in.Contents))
	for i, content := range in.Contents {
		if expr, ok := content.(Expression); ok {
			replaced, err := v.replace(expr)
			if err != nil {
				return nil, err
			}
			newContents[i] = replaced
		} else {
			newContents[i] = content
		}
	}
	span, err := in.Span()
	if err != nil {
		return nil, err
	}
	interp, err := NewInterpolation(newContents, in.Spans(), span)
	if err != nil {
		return nil, err
	}
	return interp, nil
}

var _ ExpressionVisitor[Expression] = (*ReplaceExpressionVisitor)(nil)
var _ IfConditionExpressionVisitor[IfConditionExpression] = (*ReplaceExpressionVisitor)(nil)
