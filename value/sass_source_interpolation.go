// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/source_interpolation.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SourceInterpolationVisitor builds an Interpolation that evaluates to the
// same text as the visited expression, slicing literal source spans so
// formatting is preserved.
//
// Use it through Expression.SourceInterpolation or
// IfConditionExpression.ToInterpolation. The buffer is set to nil when a
// node cannot be represented as valid CSS with interpolations; Result then
// returns nil.
//
// Matches Dart: SourceInterpolationVisitor
type SourceInterpolationVisitor struct {
	// buffer accumulates output, or nil once an unrepresentable node is
	// met. Nil is sticky: later visits keep it nil.
	buffer *InterpolationBuffer
}

// NewSourceInterpolationVisitor creates a visitor with a fresh buffer.
//
// Matches Dart: SourceInterpolationVisitor.new (implicit)
func NewSourceInterpolationVisitor() *SourceInterpolationVisitor {
	return &SourceInterpolationVisitor{buffer: &InterpolationBuffer{}}
}

// Result returns the built interpolation, or nil when the expression could
// not be represented as valid CSS with interpolations.
func (v *SourceInterpolationVisitor) Result() *Interpolation {
	if v.buffer == nil {
		return nil
	}
	interpolation, err := v.buffer.Interpolation(sasscommon.SimpleFileSpan{})
	if err != nil {
		return nil
	}
	return interpolation
}

// VisitBinaryOperationExpression marks the expression unrepresentable:
// operators need evaluation, not source slicing.
func (v *SourceInterpolationVisitor) VisitBinaryOperationExpression(*BinaryOperationExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitBooleanExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitBooleanExpression(*BooleanExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitColorExpression copies the color's source text verbatim.
func (v *SourceInterpolationVisitor) VisitColorExpression(node *ColorExpression) (Value, error) {
	if v.buffer != nil {
		span, err := node.Span()
		if err != nil {
			return nil, err
		}
		text, err := span.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	return nil, nil
}

// VisitFunctionExpression marks the expression unrepresentable: calls need
// evaluation.
func (v *SourceInterpolationVisitor) VisitFunctionExpression(*FunctionExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitInterpolatedFunctionExpression copies the interpolated name and then
// the positional arguments with their surrounding source text.
func (v *SourceInterpolationVisitor) VisitInterpolatedFunctionExpression(node *InterpolatedFunctionExpression) (Value, error) {
	if v.buffer != nil {
		v.buffer.AddInterpolation(node.Name)
	}
	if err := v.visitArguments(node.Arguments()); err != nil {
		return nil, err
	}
	return nil, nil
}

// VisitIfExpression replays each branch's source: the text before each
// condition or else-expression, the condition itself, the text between the
// condition and its expression, and the expression.
func (v *SourceInterpolationVisitor) VisitIfExpression(node *IfExpression) (Value, error) {
	var lastSpan sasscommon.FileSpan
	for _, branch := range node.Branches {
		var err error
		var firstSpan sasscommon.FileSpan
		if branch.Condition != nil {
			firstSpan, err = branch.Condition.Span()
			if err != nil {
				return nil, err
			}
		} else {
			firstSpan, err = branch.Expression.Span()
			if err != nil {
				return nil, err
			}
		}
		if v.buffer != nil {
			if lastSpan == nil {
				sp, err := node.Span()
				if err != nil {
					return nil, err
				}
				beforeSpan, err := sp.Before(firstSpan)
				if err != nil {
					return nil, err
				}
				text, err := beforeSpan.SpanText()
				if err != nil {
					return nil, err
				}
				v.buffer.Write(text)
			} else {
				betweenSpan, err := lastSpan.Between(firstSpan)
				if err != nil {
					return nil, err
				}
				text, err := betweenSpan.SpanText()
				if err != nil {
					return nil, err
				}
				v.buffer.Write(text)
			}
		}
		if branch.Condition != nil {
			if _, err := branch.Condition.AcceptAny(v); err != nil {
				return nil, err
			}
			if v.buffer != nil {
				condSpan, err := branch.Condition.Span()
				if err != nil {
					return nil, err
				}
				exprSpan, err := branch.Expression.Span()
				if err != nil {
					return nil, err
				}
				betweenSpan, err := condSpan.Between(exprSpan)
				if err != nil {
					return nil, err
				}
				text, err := betweenSpan.SpanText()
				if err != nil {
					return nil, err
				}
				v.buffer.Write(text)
			}
		}
		if _, err := branch.Expression.AcceptValue(v); err != nil {
			return nil, err
		}
		lastSpan, err = branch.Expression.Span()
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// VisitLegacyIfExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitLegacyIfExpression(*LegacyIfExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitListExpression copies bracketed lists element by element with their
// surrounding brackets; bare single-element or empty lists have no stable
// source shape and are marked unrepresentable.
func (v *SourceInterpolationVisitor) VisitListExpression(node *ListExpression) (Value, error) {
	if len(node.Contents) <= 1 && !node.HasBrackets {
		v.buffer = nil
		return nil, nil
	}
	if node.HasBrackets && len(node.Contents) == 0 {
		if v.buffer != nil {
			span, err := node.Span()
			if err != nil {
				return nil, err
			}
			text, err := span.SpanText()
			if err != nil {
				return nil, err
			}
			v.buffer.Write(text)
		}
		return nil, nil
	}
	if node.HasBrackets && v.buffer != nil {
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		firstSpan, err := node.Contents[0].Span()
		if err != nil {
			return nil, err
		}
		beforeSpan, err := sp.Before(firstSpan)
		if err != nil {
			return nil, err
		}
		text, err := beforeSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	if err := writeListAndBetween(v, node.Contents, func(node Expression) error {
		_, err := node.AcceptValue(v)
		return err
	}); err != nil {
		return nil, err
	}
	if node.HasBrackets && v.buffer != nil {
		last := node.Contents[len(node.Contents)-1]
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		lastSpan, err := last.Span()
		if err != nil {
			return nil, err
		}
		afterSpan, err := sp.After(lastSpan)
		if err != nil {
			return nil, err
		}
		text, err := afterSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	return nil, nil
}

// VisitMapExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitMapExpression(*MapExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitNullExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitNullExpression(*NullExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitNumberExpression copies the number's source text verbatim.
func (v *SourceInterpolationVisitor) VisitNumberExpression(node *NumberExpression) (Value, error) {
	if v.buffer != nil {
		span, err := node.Span()
		if err != nil {
			return nil, err
		}
		text, err := span.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	return nil, nil
}

// VisitParenthesizedExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitParenthesizedExpression(*ParenthesizedExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitSelectorExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitSelectorExpression(*SelectorExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitStringExpression copies plain strings verbatim; strings with
// interpolation replay each element's source slice, embedding live
// expressions and copying literal gaps (including quote-adjacent text)
// around them.
func (v *SourceInterpolationVisitor) VisitStringExpression(node *StringExpression) (Value, error) {
	if node.Text.IsPlain() {
		if v.buffer != nil {
			span, err := node.Span()
			if err != nil {
				return nil, err
			}
			text, err := span.SpanText()
			if err != nil {
				return nil, err
			}
			v.buffer.Write(text)
		}
		return nil, nil
	}
	for i := 0; i < len(node.Text.Contents); i++ {
		span, err := node.Text.SpanForElement(i)
		if err != nil {
			return nil, err
		}
		switch content := node.Text.Contents[i].(type) {
		case Expression:
			if v.buffer != nil {
				if i == 0 {
					sp, err := node.Span()
					if err != nil {
						return nil, err
					}
					beforeSpan, err := sp.Before(span)
					if err != nil {
						return nil, err
					}
					text, err := beforeSpan.SpanText()
					if err != nil {
						return nil, err
					}
					v.buffer.Write(text)
				}
				v.buffer.Add(content, span)
				if i == len(node.Text.Contents)-1 {
					sp, err := node.Span()
					if err != nil {
						return nil, err
					}
					afterSpan, err := sp.After(span)
					if err != nil {
						return nil, err
					}
					text, err := afterSpan.SpanText()
					if err != nil {
						return nil, err
					}
					v.buffer.Write(text)
				}
			}
		default:
			if v.buffer != nil {
				text, err := span.SpanText()
				if err != nil {
					return nil, err
				}
				v.buffer.Write(text)
			}
		}
	}
	return nil, nil
}

// VisitSupportsExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitSupportsExpression(*SupportsExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitUnaryOperationExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitUnaryOperationExpression(*UnaryOperationExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitValueExpression marks the expression unrepresentable: embedded values
// have no source text to slice.
func (v *SourceInterpolationVisitor) VisitValueExpression(*ValueExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// VisitVariableExpression marks the expression unrepresentable.
func (v *SourceInterpolationVisitor) VisitVariableExpression(*VariableExpression) (Value, error) {
	v.buffer = nil
	return nil, nil
}

// if() condition expressions: each replays its own source slices around
// recursive visits, so the rebuilt interpolation keeps the original
// punctuation.

// VisitIfConditionParenthesized copies the parentheses around the inner
// condition.
func (v *SourceInterpolationVisitor) VisitIfConditionParenthesized(node *IfConditionParenthesized) (any, error) {
	if v.buffer != nil {
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		exprSpan, err := node.Expression.Span()
		if err != nil {
			return nil, err
		}
		beforeSpan, err := sp.Before(exprSpan)
		if err != nil {
			return nil, err
		}
		text, err := beforeSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	if _, err := node.Expression.AcceptAny(v); err != nil {
		return nil, err
	}
	if v.buffer != nil {
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		exprSpan, err := node.Expression.Span()
		if err != nil {
			return nil, err
		}
		afterSpan, err := sp.After(exprSpan)
		if err != nil {
			return nil, err
		}
		text, err := afterSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	return nil, nil
}

// VisitIfConditionNegation copies the "not " prefix before the inner
// condition.
func (v *SourceInterpolationVisitor) VisitIfConditionNegation(node *IfConditionNegation) (any, error) {
	if v.buffer != nil {
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		exprSpan, err := node.Expression.Span()
		if err != nil {
			return nil, err
		}
		beforeSpan, err := sp.Before(exprSpan)
		if err != nil {
			return nil, err
		}
		text, err := beforeSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	if _, err := node.Expression.AcceptAny(v); err != nil {
		return nil, err
	}
	return nil, nil
}

// VisitIfConditionOperation visits each operand, copying the source text
// between operands.
func (v *SourceInterpolationVisitor) VisitIfConditionOperation(node *IfConditionOperation) (any, error) {
	if err := writeListAndBetween(v, node.Expressions, func(node IfConditionExpression) error {
		_, err := node.AcceptAny(v)
		return err
	}); err != nil {
		return nil, err
	}
	return nil, nil
}

// VisitIfConditionFunction copies the name interpolation, the source text
// between the name and arguments, the arguments interpolation, and the
// trailing source text.
func (v *SourceInterpolationVisitor) VisitIfConditionFunction(node *IfConditionFunction) (any, error) {
	if v.buffer != nil {
		v.buffer.AddInterpolation(node.Name)
		nameSpan, err := node.Name.Span()
		if err != nil {
			return nil, err
		}
		argsSpan, err := node.Arguments.Span()
		if err != nil {
			return nil, err
		}
		betweenSpan, err := nameSpan.Between(argsSpan)
		if err != nil {
			return nil, err
		}
		text, err := betweenSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
		v.buffer.AddInterpolation(node.Arguments)
		sp, err := node.Span()
		if err != nil {
			return nil, err
		}
		afterSpan, err := sp.After(argsSpan)
		if err != nil {
			return nil, err
		}
		text, err = afterSpan.SpanText()
		if err != nil {
			return nil, err
		}
		v.buffer.Write(text)
	}
	return nil, nil
}

// VisitIfConditionSass marks the condition unrepresentable: sass()
// conditions need evaluation.
func (v *SourceInterpolationVisitor) VisitIfConditionSass(*IfConditionSass) (any, error) {
	v.buffer = nil
	return nil, nil
}

// VisitIfConditionRaw copies the raw text interpolation through.
func (v *SourceInterpolationVisitor) VisitIfConditionRaw(node *IfConditionRaw) (any, error) {
	if v.buffer != nil {
		v.buffer.AddInterpolation(node.Text)
	}
	return nil, nil
}

// Private helpers.

// visitArguments replays positional arguments with their surrounding source
// text. Calls with named arguments or rest arguments are not valid
// interpolated plain CSS, so they leave the buffer untouched for the caller
// to treat as unrepresentable.
func (v *SourceInterpolationVisitor) visitArguments(arguments *ArgumentList) error {
	if arguments.Named.Len() > 0 || arguments.Rest != nil {
		return nil
	}
	if v.buffer == nil {
		return nil
	}
	if len(arguments.Positional) == 0 {
		span, err := arguments.Span()
		if err != nil {
			return err
		}
		text, err := span.SpanText()
		if err != nil {
			return err
		}
		v.buffer.Write(text)
		return nil
	}
	sp, err := arguments.Span()
	if err != nil {
		return err
	}
	firstSpan, err := arguments.Positional[0].Span()
	if err != nil {
		return err
	}
	beforeSpan, err := sp.Before(firstSpan)
	if err != nil {
		return err
	}
	text, err := beforeSpan.SpanText()
	if err != nil {
		return err
	}
	v.buffer.Write(text)
	if err := writeListAndBetween(v, arguments.Positional, func(node Expression) error {
		_, err := node.AcceptValue(v)
		return err
	}); err != nil {
		return err
	}
	last := arguments.Positional[len(arguments.Positional)-1]
	lastSpan, err := last.Span()
	if err != nil {
		return err
	}
	afterSpan, err := sp.After(lastSpan)
	if err != nil {
		return err
	}
	text, err = afterSpan.SpanText()
	if err != nil {
		return err
	}
	v.buffer.Write(text)
	return nil
}

// writeListAndBetween visits each node and copies whatever source text lies
// between consecutive nodes. It stops early once the buffer is nil.
func writeListAndBetween[T sasscommon.AstNode](v *SourceInterpolationVisitor, nodes []T, visit func(T) error) error {
	var lastSpan sasscommon.FileSpan
	for _, node := range nodes {
		if lastSpan != nil {
			if v.buffer != nil {
				sp, err := node.Span()
				if err != nil {
					return err
				}
				betweenSpan, err := lastSpan.Between(sp)
				if err != nil {
					return err
				}
				text, err := betweenSpan.SpanText()
				if err != nil {
					return err
				}
				v.buffer.Write(text)
			}
		}
		if err := visit(node); err != nil {
			return err
		}
		if v.buffer == nil {
			return nil
		}
		var err error
		lastSpan, err = node.Span()
		if err != nil {
			return err
		}
	}
	return nil
}

var _ ExpressionVisitor[Value] = (*SourceInterpolationVisitor)(nil)
var _ IfConditionExpressionVisitor[any] = (*SourceInterpolationVisitor)(nil)
