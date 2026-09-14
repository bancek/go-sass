// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/is_plain_css.dart

// IsPlainCssVisitor determines whether an expression is valid plain CSS
// that will produce the same result as it would in Sass.
//
// Use it through Expression.isPlainCss. It deliberately avoids a search
// visitor with a default-true fallback, so adding a new expression type
// without updating this visitor fails closed instead of silently passing.
//
// Matches Dart: IsPlainCssVisitor (lib/src/visitor/is_plain_css.dart)
type IsPlainCssVisitor struct {
	// allowInterpolation permits interpolated expressions as an exception,
	// even when they contain SassScript. It ports Dart's
	// _allowInterpolation field.
	allowInterpolation bool
}

// NewIsPlainCssVisitor creates a plain-CSS checker. When
// allowInterpolation is set, interpolated expressions are accepted as an
// exception even if they contain SassScript.
func NewIsPlainCssVisitor(allowInterpolation bool) *IsPlainCssVisitor {
	return &IsPlainCssVisitor{allowInterpolation: allowInterpolation}
}

// VisitBinaryOperationExpression always reports false: operations are
// SassScript, never plain CSS.
func (v *IsPlainCssVisitor) VisitBinaryOperationExpression(node *BinaryOperationExpression) (bool, error) {
	return false, nil
}

// VisitBooleanExpression always reports false: booleans have no plain-CSS
// literal form.
func (v *IsPlainCssVisitor) VisitBooleanExpression(node *BooleanExpression) (bool, error) {
	return false, nil
}

// VisitColorExpression always reports true: colors are plain CSS.
func (v *IsPlainCssVisitor) VisitColorExpression(node *ColorExpression) (bool, error) {
	return true, nil
}

// VisitFunctionExpression reports true for unprefixed functions whose
// argument list is itself plain CSS.
func (v *IsPlainCssVisitor) VisitFunctionExpression(node *FunctionExpression) (bool, error) {
	if node.Namespace != nil {
		return false, nil
	}
	return v.visitArgumentList(node.Arguments())
}

// VisitIfExpression reports true when every branch condition (where
// present) and every branch body is plain CSS.
func (v *IsPlainCssVisitor) VisitIfExpression(node *IfExpression) (bool, error) {
	for _, pair := range node.Branches {
		if pair.Condition != nil {
			plain, err := pair.Condition.AcceptBool(v)
			if err != nil {
				return false, err
			}
			if !plain {
				return false, nil
			}
		}
		plain, err := pair.Expression.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !plain {
			return false, nil
		}
	}
	return true, nil
}

// VisitInterpolatedFunctionExpression reports true only in interpolation
// mode with a plain-CSS argument list; otherwise interpolated names may
// hide SassScript.
func (v *IsPlainCssVisitor) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (bool, error) {
	if !v.allowInterpolation {
		return false, nil
	}
	return v.visitArgumentList(node.Arguments())
}

// VisitLegacyIfExpression always reports false: the legacy if() syntax is
// SassScript, never plain CSS.
func (v *IsPlainCssVisitor) VisitLegacyIfExpression(node *LegacyIfExpression) (bool, error) {
	return false, nil
}

// VisitListExpression reports true when every element is plain CSS. An
// empty unbracketed list is not valid CSS and reports false.
func (v *IsPlainCssVisitor) VisitListExpression(node *ListExpression) (bool, error) {
	if len(node.Contents) == 0 && !node.HasBrackets {
		return false, nil
	}
	for _, element := range node.Contents {
		plain, err := element.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !plain {
			return false, nil
		}
	}
	return true, nil
}

// VisitMapExpression always reports false: maps have no plain-CSS form.
func (v *IsPlainCssVisitor) VisitMapExpression(node *MapExpression) (bool, error) {
	return false, nil
}

// VisitNullExpression always reports false: null has no plain-CSS form.
func (v *IsPlainCssVisitor) VisitNullExpression(node *NullExpression) (bool, error) {
	return false, nil
}

// VisitNumberExpression always reports true: numbers are plain CSS.
func (v *IsPlainCssVisitor) VisitNumberExpression(node *NumberExpression) (bool, error) {
	return true, nil
}

// VisitParenthesizedExpression defers to the inner expression.
func (v *IsPlainCssVisitor) VisitParenthesizedExpression(node *ParenthesizedExpression) (bool, error) {
	return node.Expression.AcceptBool(v)
}

// VisitSelectorExpression always reports false: selector expressions are
// Sass constructs, not plain CSS values.
func (v *IsPlainCssVisitor) VisitSelectorExpression(node *SelectorExpression) (bool, error) {
	return false, nil
}

// VisitStringExpression reports true in interpolation mode, and otherwise
// only when the string text contains no interpolation.
func (v *IsPlainCssVisitor) VisitStringExpression(node *StringExpression) (bool, error) {
	return v.allowInterpolation || node.Text.IsPlain(), nil
}

// VisitSupportsExpression always reports false: supports conditions are
// not plain CSS values.
func (v *IsPlainCssVisitor) VisitSupportsExpression(node *SupportsExpression) (bool, error) {
	return false, nil
}

// VisitUnaryOperationExpression always reports false: operations are
// SassScript, never plain CSS.
func (v *IsPlainCssVisitor) VisitUnaryOperationExpression(node *UnaryOperationExpression) (bool, error) {
	return false, nil
}

// VisitValueExpression always reports false: a Sass value flowing through
// an expression slot is not known to be plain CSS.
func (v *IsPlainCssVisitor) VisitValueExpression(node *ValueExpression) (bool, error) {
	return false, nil
}

// VisitVariableExpression always reports false: variables are SassScript,
// never plain CSS.
func (v *IsPlainCssVisitor) VisitVariableExpression(node *VariableExpression) (bool, error) {
	return false, nil
}

// VisitIfConditionParenthesized defers to the inner condition expression.
func (v *IsPlainCssVisitor) VisitIfConditionParenthesized(node *IfConditionParenthesized) (bool, error) {
	return node.Expression.AcceptBool(v)
}

// VisitIfConditionNegation defers to the negated condition expression.
func (v *IsPlainCssVisitor) VisitIfConditionNegation(node *IfConditionNegation) (bool, error) {
	return node.Expression.AcceptBool(v)
}

// VisitIfConditionOperation reports true when every operand condition is
// plain CSS.
func (v *IsPlainCssVisitor) VisitIfConditionOperation(node *IfConditionOperation) (bool, error) {
	for _, expr := range node.Expressions {
		plain, err := expr.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !plain {
			return false, nil
		}
	}
	return true, nil
}

// VisitIfConditionFunction reports true in interpolation mode, and
// otherwise only when both the function name and its arguments are
// interpolation-free.
func (v *IsPlainCssVisitor) VisitIfConditionFunction(node *IfConditionFunction) (bool, error) {
	return v.allowInterpolation || (node.Name.IsPlain() && node.Arguments.IsPlain()), nil
}

// VisitIfConditionSass always reports false: an embedded Sass condition is
// SassScript, never plain CSS.
func (v *IsPlainCssVisitor) VisitIfConditionSass(node *IfConditionSass) (bool, error) {
	return false, nil
}

// VisitIfConditionRaw reports true in interpolation mode, and otherwise
// only when the raw text contains no interpolation.
func (v *IsPlainCssVisitor) VisitIfConditionRaw(node *IfConditionRaw) (bool, error) {
	return v.allowInterpolation || node.Text.IsPlain(), nil
}

// visitArgumentList reports whether an argument list holds only plain CSS:
// no named arguments, no rest argument, and plain positional arguments.
func (v *IsPlainCssVisitor) visitArgumentList(node *ArgumentList) (bool, error) {
	if node.Named.Len() > 0 || node.Rest != nil {
		return false, nil
	}
	for _, arg := range node.Positional {
		plain, err := arg.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if !plain {
			return false, nil
		}
	}
	return true, nil
}

var _ ExpressionVisitor[bool] = (*IsPlainCssVisitor)(nil)
var _ IfConditionExpressionVisitor[bool] = (*IsPlainCssVisitor)(nil)
