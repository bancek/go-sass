// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/recursive_ast.dart

// RecursiveAstVisitor walks every statement and expression in a Sass AST.
//
// It extends RecursiveStatementVisitor so that each statement method also
// descends into its inline expressions, interpolations, argument lists,
// supports conditions, and selector components. Embed it and override
// individual Visit methods to observe specific nodes.
//
// The shared helpers visitArgumentList, visitSupportsCondition,
// visitInterpolation, and visitQualifiedName recurse into those subtrees;
// overrides usually call them to keep the default traversal.
type RecursiveAstVisitor struct {
	*RecursiveStatementVisitor
}

func (v *RecursiveAstVisitor) VisitAtRootRule(node *AtRootRule) (struct{}, error) {
	if node.Query != nil {
		if err := v.visitInterpolation(node.Query); err != nil {
			return struct{}{}, err
		}
	}
	return v.RecursiveStatementVisitor.VisitAtRootRule(node)
}

func (v *RecursiveAstVisitor) VisitAtRule(node *AtRule) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	if node.Value != nil {
		if err := v.visitInterpolation(node.Value); err != nil {
			return struct{}{}, err
		}
	}
	return v.RecursiveStatementVisitor.VisitAtRule(node)
}

func (v *RecursiveAstVisitor) VisitContentRule(node *ContentRule) (struct{}, error) {
	if err := v.visitArgumentList(node.Arguments); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitDebugRule(node *DebugRule) (struct{}, error) {
	if err := v.visitExpression(node.Expression); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitDeclaration(node *Declaration) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	if node.Value != nil {
		if err := v.visitExpression(node.Value); err != nil {
			return struct{}{}, err
		}
	}
	return v.RecursiveStatementVisitor.VisitDeclaration(node)
}

func (v *RecursiveAstVisitor) VisitEachRule(node *EachRule) (struct{}, error) {
	if err := v.visitExpression(node.List); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitEachRule(node)
}

func (v *RecursiveAstVisitor) VisitErrorRule(node *ErrorRule) (struct{}, error) {
	if err := v.visitExpression(node.Expression); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitExtendRule(node *ExtendRule) (struct{}, error) {
	if err := v.visitInterpolation(node.Selector); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitForRule(node *ForRule) (struct{}, error) {
	if err := v.visitExpression(node.From); err != nil {
		return struct{}{}, err
	}
	if err := v.visitExpression(node.To); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitForRule(node)
}

func (v *RecursiveAstVisitor) VisitForwardRule(node *ForwardRule) (struct{}, error) {
	for _, variable := range node.Configuration {
		if err := v.visitExpression(variable.Expression); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIfRule(node *IfRule) (struct{}, error) {
	for _, clause := range node.Clauses {
		if err := v.visitExpression(clause.Expression); err != nil {
			return struct{}{}, err
		}
		for _, child := range clause.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return struct{}{}, err
			}
		}
	}
	if node.LastClause != nil {
		for _, child := range node.LastClause.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return struct{}{}, err
			}
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitImportRule(node *ImportRule) (struct{}, error) {
	for _, imp := range node.Imports {
		if s, ok := imp.(*StaticImport); ok {
			if err := v.visitInterpolation(s.URL); err != nil {
				return struct{}{}, err
			}
			if s.Modifiers != nil {
				if err := v.visitInterpolation(s.Modifiers); err != nil {
					return struct{}{}, err
				}
			}
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIncludeRule(node *IncludeRule) (struct{}, error) {
	if err := v.visitArgumentList(node.Arguments()); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitIncludeRule(node)
}

func (v *RecursiveAstVisitor) VisitLoudComment(node *LoudComment) (struct{}, error) {
	if err := v.visitInterpolation(node.Text); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitMediaRule(node *MediaRule) (struct{}, error) {
	if err := v.visitInterpolation(node.Query); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitMediaRule(node)
}

func (v *RecursiveAstVisitor) VisitReturnRule(node *ReturnRule) (struct{}, error) {
	if err := v.visitExpression(node.Expression); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitStyleRule(node *StyleRule) (struct{}, error) {
	if node.Selector != nil {
		if err := v.visitInterpolation(node.Selector); err != nil {
			return struct{}{}, err
		}
	}
	return v.RecursiveStatementVisitor.VisitStyleRule(node)
}

func (v *RecursiveAstVisitor) VisitSupportsRule(node *SupportsRule) (struct{}, error) {
	if err := v.visitSupportsCondition(node.Condition); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitSupportsRule(node)
}

func (v *RecursiveAstVisitor) VisitUseRule(node *UseRule) (struct{}, error) {
	for _, variable := range node.Configuration {
		if err := v.visitExpression(variable.Expression); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitVariableDeclaration(node *VariableDeclaration) (struct{}, error) {
	if err := v.visitExpression(node.Expression); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitWarnRule(node *WarnRule) (struct{}, error) {
	if err := v.visitExpression(node.Expression); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitWhileRule(node *WhileRule) (struct{}, error) {
	if err := v.visitExpression(node.Condition); err != nil {
		return struct{}{}, err
	}
	return v.RecursiveStatementVisitor.VisitWhileRule(node)
}

// Expressions

func (v *RecursiveAstVisitor) visitExpression(expr Expression) error {
	_, err := expr.AcceptVoid(v)
	return err
}

func (v *RecursiveAstVisitor) VisitBinaryOperationExpression(node *BinaryOperationExpression) (struct{}, error) {
	if _, err := node.Left.AcceptVoid(v); err != nil {
		return struct{}{}, err
	}
	if _, err := node.Right.AcceptVoid(v); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitBooleanExpression(*BooleanExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitColorExpression(*ColorExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitFunctionExpression(node *FunctionExpression) (struct{}, error) {
	if err := v.visitArgumentList(node.Arguments()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIfExpression(node *IfExpression) (struct{}, error) {
	for _, branch := range node.Branches {
		if branch.Condition != nil {
			if _, err := branch.Condition.AcceptVoid(v); err != nil {
				return struct{}{}, err
			}
		}
		if _, err := branch.Expression.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	if err := v.visitArgumentList(node.Arguments()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitLegacyIfExpression(node *LegacyIfExpression) (struct{}, error) {
	if err := v.visitArgumentList(node.Arguments()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitListExpression(node *ListExpression) (struct{}, error) {
	for _, item := range node.Contents {
		if _, err := item.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitMapExpression(node *MapExpression) (struct{}, error) {
	for _, pair := range node.Pairs {
		if _, err := pair.Key.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
		if _, err := pair.Value.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitNullExpression(*NullExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitNumberExpression(*NumberExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitParenthesizedExpression(node *ParenthesizedExpression) (struct{}, error) {
	if _, err := node.Expression.AcceptVoid(v); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitSelectorExpression(*SelectorExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitStringExpression(node *StringExpression) (struct{}, error) {
	if err := v.visitInterpolation(node.Text); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitSupportsExpression(node *SupportsExpression) (struct{}, error) {
	if err := v.visitSupportsCondition(node.Condition); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitUnaryOperationExpression(node *UnaryOperationExpression) (struct{}, error) {
	if _, err := node.Operand.AcceptVoid(v); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitValueExpression(*ValueExpression) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitVariableExpression(*VariableExpression) (struct{}, error) {
	return struct{}{}, nil
}

// if() condition expressions

func (v *RecursiveAstVisitor) VisitIfConditionParenthesized(node *IfConditionParenthesized) (struct{}, error) {
	_, err := node.Expression.AcceptVoid(v)
	return struct{}{}, err
}

func (v *RecursiveAstVisitor) VisitIfConditionNegation(node *IfConditionNegation) (struct{}, error) {
	_, err := node.Expression.AcceptVoid(v)
	return struct{}{}, err
}

func (v *RecursiveAstVisitor) VisitIfConditionOperation(node *IfConditionOperation) (struct{}, error) {
	for _, expr := range node.Expressions {
		if _, err := expr.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIfConditionFunction(node *IfConditionFunction) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	if err := v.visitInterpolation(node.Arguments); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIfConditionSass(node *IfConditionSass) (struct{}, error) {
	_, err := node.Expression.AcceptVoid(v)
	return struct{}{}, err
}

func (v *RecursiveAstVisitor) VisitIfConditionRaw(node *IfConditionRaw) (struct{}, error) {
	if err := v.visitInterpolation(node.Text); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

// Interpolated selectors

func (v *RecursiveAstVisitor) VisitAttributeSelector(node *InterpolatedAttributeSelector) (struct{}, error) {
	if err := v.visitQualifiedName(node.Name); err != nil {
		return struct{}{}, err
	}
	if node.Value != nil {
		if err := v.visitInterpolation(node.Value); err != nil {
			return struct{}{}, err
		}
	}
	if node.Modifier != nil {
		if err := v.visitInterpolation(node.Modifier); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitClassSelector(node *InterpolatedClassSelector) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitComplexSelector(node *InterpolatedComplexSelector) (struct{}, error) {
	for _, component := range node.Components {
		if _, err := v.VisitCompoundSelector(component.Selector); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitCompoundSelector(node *InterpolatedCompoundSelector) (struct{}, error) {
	for _, simple := range node.Components {
		if _, err := simple.AcceptVoid(v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitIDSelector(node *InterpolatedIDSelector) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitParentSelector(node *InterpolatedParentSelector) (struct{}, error) {
	if node.Suffix != nil {
		if err := v.visitInterpolation(node.Suffix); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitPlaceholderSelector(node *InterpolatedPlaceholderSelector) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitPseudoSelector(node *InterpolatedPseudoSelector) (struct{}, error) {
	if err := v.visitInterpolation(node.Name); err != nil {
		return struct{}{}, err
	}
	if node.Argument != nil {
		if err := v.visitInterpolation(node.Argument); err != nil {
			return struct{}{}, err
		}
	}
	if node.Selector != nil {
		if _, err := v.VisitSelectorList(node.Selector); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitSelectorList(node *InterpolatedSelectorList) (struct{}, error) {
	for _, component := range node.Components {
		if _, err := v.VisitComplexSelector(component); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitTypeSelector(node *InterpolatedTypeSelector) (struct{}, error) {
	if err := v.visitQualifiedName(node.Name); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveAstVisitor) VisitUniversalSelector(node *InterpolatedUniversalSelector) (struct{}, error) {
	if node.Namespace != nil {
		if err := v.visitInterpolation(node.Namespace); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

// Protected methods

func (v *RecursiveAstVisitor) visitCallableDeclaration(node CallableDeclaration) (struct{}, error) {
	for _, param := range node.Parameters().Parameters {
		if param.DefaultValue != nil {
			if err := v.visitExpression(param.DefaultValue); err != nil {
				return struct{}{}, err
			}
		}
	}
	return v.RecursiveStatementVisitor.visitCallableDeclaration(node)
}

func (v *RecursiveAstVisitor) visitArgumentList(invocation *ArgumentList) error {
	for _, expr := range invocation.Positional {
		if err := v.visitExpression(expr); err != nil {
			return err
		}
	}
	for _, expr := range invocation.Named.Entries() {
		if err := v.visitExpression(expr); err != nil {
			return err
		}
	}
	if invocation.Rest != nil {
		if err := v.visitExpression(invocation.Rest); err != nil {
			return err
		}
	}
	if invocation.KeywordRest != nil {
		if err := v.visitExpression(invocation.KeywordRest); err != nil {
			return err
		}
	}
	return nil
}

func (v *RecursiveAstVisitor) visitSupportsCondition(condition SupportsCondition) error {
	switch c := condition.(type) {
	case *SupportsOperation:
		if err := v.visitSupportsCondition(c.Left); err != nil {
			return err
		}
		if err := v.visitSupportsCondition(c.Right); err != nil {
			return err
		}
	case *SupportsNegation:
		if err := v.visitSupportsCondition(c.Condition); err != nil {
			return err
		}
	case *SupportsInterpolation:
		if err := v.visitExpression(c.Expression); err != nil {
			return err
		}
	case *SupportsDeclaration:
		if err := v.visitExpression(c.Name); err != nil {
			return err
		}
		if err := v.visitExpression(c.Value); err != nil {
			return err
		}
	}
	return nil
}

func (v *RecursiveAstVisitor) visitInterpolation(in *Interpolation) error {
	for _, content := range in.Contents {
		if expr, ok := content.(Expression); ok {
			if err := v.visitExpression(expr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *RecursiveAstVisitor) visitQualifiedName(node *InterpolatedQualifiedName) error {
	if node.Namespace != nil {
		if err := v.visitInterpolation(node.Namespace); err != nil {
			return err
		}
	}
	if err := v.visitInterpolation(node.Name); err != nil {
		return err
	}
	return nil
}

// Ensure RecursiveAstVisitor implements the visitor interfaces.
var _ StatementVisitor[struct{}] = (*RecursiveAstVisitor)(nil)
var _ ExpressionVisitor[struct{}] = (*RecursiveAstVisitor)(nil)
var _ IfConditionExpressionVisitor[struct{}] = (*RecursiveAstVisitor)(nil)
var _ InterpolatedSelectorVisitor[struct{}] = (*RecursiveAstVisitor)(nil)
