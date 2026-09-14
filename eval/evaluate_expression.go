// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (expression visitors)

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// calcFunctionNames lists the CSS math functions whose operands may form a
// slash-separated number. A namespaced call or a user-defined function with
// the same name disqualifies the operand (see operandAllowsSlash). The same
// list is also tracked for plain-CSS analysis on the parser side, so changes
// here should be mirrored there.
var calcFunctionNames = map[string]struct{}{
	"calc": {}, "clamp": {}, "hypot": {}, "sin": {}, "cos": {},
	"tan": {}, "asin": {}, "acos": {}, "atan": {}, "sqrt": {},
	"exp": {}, "sign": {}, "mod": {}, "rem": {}, "atan2": {},
	"pow": {}, "log": {}, "calc-size": {},
}

// --- Expression visitors ---

// VisitValueExpression returns the literal value unchanged.
func (v *EvaluateVisitor) VisitValueExpression(expr *value.ValueExpression) (value.Value, error) {
	if expr.Value == nil {
		panic("ValueExpression does not contain a valid value")
	}
	return expr.Value, nil
}

// VisitVariableExpression looks up a variable (with optional namespace) in the
// environment. The lookup runs under the expression span so environment errors
// point at the use site; a miss reports "Undefined variable." at that span.
func (v *EvaluateVisitor) VisitVariableExpression(expr *value.VariableExpression) (value.Value, error) {
	result, err := addExceptionSpan(v, expr, func() (value.Value, error) {
		return v.env.GetVariable(expr.Name(), expr.Namespace())
	}, nil)
	if err != nil {
		return nil, err
	}
	if result != nil {
		return result, nil
	}
	span, err := expr.Span()
	if err != nil {
		return nil, err
	}
	return nil, v.exception("Undefined variable.", &span)
}

// VisitStringExpression evaluates a string with interpolation. It does not go
// through performInterpolation because the raw text of embedded values is
// needed rather than their semantic values: plain text is spliced as-is,
// embedded SassString values contribute their text directly, and every other
// value is serialized unquoted. Evaluation clears inSupportsDeclaration so
// interpolations never inherit the surrounding supports-declaration context.
func (v *EvaluateVisitor) VisitStringExpression(expr *value.StringExpression) (value.Value, error) {
	oldInSupportsDeclaration := v.inSupportsDeclaration
	v.inSupportsDeclaration = false
	defer func() { v.inSupportsDeclaration = oldInSupportsDeclaration }()

	if expr.Text == nil {
		return &value.SassString{Text: "", HasQuotes: expr.HasQuotes}, nil
	}
	if expr.Text.IsPlain() {
		return &value.SassString{Text: *expr.Text.AsPlain(), HasQuotes: expr.HasQuotes}, nil
	}

	var buf strings.Builder
	for _, part := range expr.Text.Contents {
		if s, ok := part.(string); ok {
			buf.WriteString(s)
		} else if expr, ok := part.(value.Expression); ok {
			result, err := v.eval(expr)
			if err != nil {
				return nil, err
			}
			if s, ok := result.(*value.SassString); ok {
				buf.WriteString(s.Text)
			} else {
				text, err := v.serialize(result, expr, false)
				if err != nil {
					return nil, err
				}
				buf.WriteString(text)
			}
		} else {
			return nil, &sasscommon.ArgumentError{
				Message: fmt.Sprintf("Unknown interpolation value type %T.", part),
			}
		}
	}
	return &value.SassString{Text: buf.String(), HasQuotes: expr.HasQuotes}, nil
}

// VisitNumberExpression converts a numeric literal to a SassNumber, carrying
// the source unit when one is present and producing a unitless number
// otherwise.
func (v *EvaluateVisitor) VisitNumberExpression(expr *value.NumberExpression) (value.Value, error) {
	if expr.Unit != nil {
		return value.NewSingleUnitNumber(expr.Value, *expr.Unit), nil
	}
	return value.NewUnitlessNumber(expr.Value), nil
}

// VisitBooleanExpression returns the shared singleton for a boolean literal.
func (v *EvaluateVisitor) VisitBooleanExpression(expr *value.BooleanExpression) (value.Value, error) {
	if expr.Value {
		return value.SassTrue, nil
	}
	return value.SassFalse, nil
}

// VisitNullExpression returns the shared null singleton.
func (v *EvaluateVisitor) VisitNullExpression(expr *value.NullExpression) (value.Value, error) {
	return value.Null, nil
}

// VisitColorExpression returns the literal color unchanged.
func (v *EvaluateVisitor) VisitColorExpression(expr *value.ColorExpression) (value.Value, error) {
	if expr.Value == nil {
		panic("ColorExpression does not contain a valid color")
	}
	return expr.Value, nil
}

// VisitListExpression evaluates each element in order and packs the results
// into a SassList preserving the source separator and brackets.
func (v *EvaluateVisitor) VisitListExpression(expr *value.ListExpression) (value.Value, error) {
	contents := make([]value.Value, 0, len(expr.Contents))
	for _, element := range expr.Contents {
		val, err := v.eval(element)
		if err != nil {
			return nil, err
		}
		contents = append(contents, val)
	}
	list, err := value.NewSassList(contents, value.ListSeparator(expr.Separator), expr.HasBrackets)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// VisitMapExpression evaluates each key and value pair into a SassMap. A
// repeated key is a MultiSpan error labeling the new occurrence "second key"
// and the original "first key", so the key source nodes are tracked alongside
// the map while it is built.
func (v *EvaluateVisitor) VisitMapExpression(expr *value.MapExpression) (value.Value, error) {
	m := value.EmptySassMap()
	// Tracks each key's source node so duplicate-key errors can span both the
	// first and second occurrence.
	keyNodes := linkedhashmap.New[value.Value, sasscommon.AstNode](value.ValueEquals)
	for _, pair := range expr.Pairs {
		key, err := v.eval(pair.Key)
		if err != nil {
			return nil, err
		}
		val, err := v.eval(pair.Value)
		if err != nil {
			return nil, err
		}

		if _, ok := m.Get(key); ok {
			oldKeyNode, _ := keyNodes.Get(key)
			sp, err := pair.Key.Span()
			if err != nil {
				return nil, err
			}
			oldSp, err := oldKeyNode.Span()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.MultiSpanSassRuntimeException{
				Message:        "Duplicate key.",
				Span:           sp,
				PrimaryLabel:   "second key",
				SecondarySpans: map[sasscommon.FileSpan]string{oldSp: "first key"},
				Trace:          v.stackTrace(sp),
			}
		}
		keyNodes.Put(key, pair.Key)
		m.Set(key, val)
	}
	return m, nil
}

// VisitBinaryOperationExpression evaluates a binary operator. Plain-CSS
// stylesheets reject every operator except `=` and `/` at the operator span.
// The whole body runs under the expression span so operand failures point at
// the operation; `or`/`and` short-circuit, leaving the right side unevaluated
// once the left side decides the result.
func (v *EvaluateVisitor) VisitBinaryOperationExpression(expr *value.BinaryOperationExpression) (value.Value, error) {
	// Plain-CSS stylesheets only permit `=` (custom-property syntax) and `/`
	// (which may be a plain slash); any other operator is rejected up front.
	if v.stylesheet.IsPlainCss() &&
		expr.Operator != value.BinaryOperatorSingleEquals &&
		expr.Operator != value.BinaryOperatorDividedBy {
		operatorSpan, err := expr.OperatorSpan()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Operators aren't allowed in plain CSS.", &operatorSpan)
	}

	return addExceptionSpan(v, expr, func() (value.Value, error) {
		left, err := v.eval(expr.Left)
		if err != nil {
			return nil, err
		}

		switch expr.Operator {
		case value.BinaryOperatorSingleEquals:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.SingleEquals(right)

		case value.BinaryOperatorOr:
			if left.IsTruthy() {
				return left, nil
			}
			return v.eval(expr.Right)

		case value.BinaryOperatorAnd:
			if left.IsTruthy() {
				return v.eval(expr.Right)
			}
			return left, nil

		case value.BinaryOperatorEquals:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			if left.Equals(right) {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil

		case value.BinaryOperatorNotEquals:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			if left.Equals(right) {
				return value.SassFalse, nil
			}
			return value.SassTrue, nil

		case value.BinaryOperatorDividedBy:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return v.slashDivide(left, right, expr)

		case value.BinaryOperatorGreaterThan:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.GreaterThan(right)

		case value.BinaryOperatorGreaterThanOrEquals:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.GreaterThanOrEquals(right)

		case value.BinaryOperatorLessThan:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.LessThan(right)

		case value.BinaryOperatorLessThanOrEquals:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.LessThanOrEquals(right)

		case value.BinaryOperatorPlus:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.Plus(right)

		case value.BinaryOperatorMinus:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.Minus(right)

		case value.BinaryOperatorTimes:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.Times(right)

		case value.BinaryOperatorModulo:
			right, err := v.eval(expr.Right)
			if err != nil {
				return nil, err
			}
			return left.Modulo(right)

		default:
			panic(fmt.Sprintf("unknown operator: %v", expr.Operator))
		}
	}, nil)
}

// slashDivide returns the result of the SassScript `/` operation between left
// and right in node. When the node allows slash separators and both operands
// qualify, the quotient keeps the slash-separated form; two plain numbers
// divide with a slash-div deprecation warning pointing at math.div/calc()
// replacements; any other operand combination divides without warning.
func (v *EvaluateVisitor) slashDivide(left, right value.Value, node *value.BinaryOperationExpression) (value.Value, error) {
	result, err := left.DividedBy(right)
	if err != nil {
		return nil, err
	}

	leftNum, leftIsNum := left.(value.SassNumber)
	rightNum, rightIsNum := right.(value.SassNumber)

	if leftIsNum && rightIsNum && node.AllowsSlash() {
		leftAllowsSlash, err := v.operandAllowsSlash(node.Left)
		if err != nil {
			return nil, err
		}
		rightAllowsSlash, err := v.operandAllowsSlash(node.Right)
		if err != nil {
			return nil, err
		}
		if leftAllowsSlash && rightAllowsSlash {
			return result.(value.SassNumber).WithSlash(leftNum, rightNum), nil
		}
	}

	if !leftIsNum || !rightIsNum {
		return result, nil
	}

	// The recommendation rewrites nested `/` operations as math.div calls and
	// renders parenthesized operands from their source text, matching the
	// deprecation message format.
	var recommendation func(expr value.Expression) (string, error)
	recommendation = func(expr value.Expression) (string, error) {
		switch e := expr.(type) {
		case *value.BinaryOperationExpression:
			if e.Operator == value.BinaryOperatorDividedBy {
				left, err := recommendation(e.Left)
				if err != nil {
					return "", err
				}
				right, err := recommendation(e.Right)
				if err != nil {
					return "", err
				}
				return "math.div(" + left + ", " + right + ")", nil
			}
		case *value.ParenthesizedExpression:
			return e.Expression.String()
		}
		return expr.String()
	}

	sp, spanErr := node.Span()
	if spanErr != nil {
		return nil, spanErr
	}
	calcExpr, err := value.ExpressionToCalc(node)
	if err != nil {
		return nil, err
	}
	recStr, recErr := recommendation(node)
	if recErr != nil {
		return nil, recErr
	}
	calcStr, calcErr := calcExpr.String()
	if calcErr != nil {
		return nil, calcErr
	}
	if err := v.warn(
		"Using / for division outside of calc() is deprecated "+
			"and will be removed in Dart Sass 2.0.0.\n"+
			"\n"+
			"Recommendation: "+recStr+" or "+
			calcStr+"\n"+
			"\n"+
			"More info and automated migrator: "+
			"https://sass-lang.com/d/slash-div",
		sp,
		deprecation.SlashDiv,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// operandAllowsSlash returns whether node can be used as a component of a
// slash-separated number. Although most of this is resolved at parse time, an
// operand only counts as slash-safe once evaluated: a known CSS math function
// qualifies only when it is unnamespaced and no user-defined function
// shadows its name, since evaluation may still turn it into a calculation.
func (v *EvaluateVisitor) operandAllowsSlash(node value.Expression) (bool, error) {
	fn, ok := node.(*value.FunctionExpression)
	if !ok {
		return true, nil
	}
	if fn.Namespace != nil {
		return false, nil
	}
	_, inSet := calcFunctionNames[strings.ToLower(fn.Name)]
	if !inSet {
		return false, nil
	}
	declared, err := v.env.GetFunction(fn.Name, nil)
	if err != nil {
		return false, err
	}
	return declared == nil, nil
}

// VisitUnaryOperationExpression evaluates the operand first, then applies the
// operator under the expression span so failures point at the operation.
func (v *EvaluateVisitor) VisitUnaryOperationExpression(expr *value.UnaryOperationExpression) (value.Value, error) {
	operand, err := v.eval(expr.Operand)
	if err != nil {
		return nil, err
	}

	return addExceptionSpan(v, expr, func() (value.Value, error) {
		switch expr.Operator {
		case value.UnaryOperatorPlus:
			return operand.UnaryPlus()

		case value.UnaryOperatorMinus:
			return operand.UnaryMinus()

		case value.UnaryOperatorDivide:
			return operand.UnaryDivide()

		case value.UnaryOperatorNot:
			return operand.UnaryNot()

		default:
			panic(fmt.Sprintf("unknown unary operator: %v", expr.Operator))
		}
	}, nil)
}

// VisitFunctionExpression evaluates a function call. Plain-CSS stylesheets skip
// the environment lookup entirely; names starting with `--` are always plain
// CSS, and a namespaced name with no definition reports "Undefined function."
// Calls to min/max/round/abs whose positional arguments are all
// calculation-safe stay as calculations with legacy unit behavior, while other
// known CSS math names evaluate as calculations directly. Anything else falls
// back to a built-in and finally to a plain-CSS call; a mixin used as a
// function is rejected. The call runs with inFunction set and its errors
// wrapped at the call span.
func (v *EvaluateVisitor) VisitFunctionExpression(expr *value.FunctionExpression) (value.Value, error) {
	var fn sasscallable.Callable
	if !v.stylesheet.IsPlainCss() {
		var err error
		fn, err = addExceptionSpan(v, expr, func() (sasscallable.Callable, error) {
			return v.env.GetFunction(expr.Name, expr.Namespace)
		}, nil)
		if err != nil {
			return nil, err
		}
	}

	if fn == nil || strings.HasPrefix(expr.OriginalName, "--") {
		if expr.Namespace != nil {
			span, spanErr := expr.Span()
			if spanErr != nil {
				return nil, spanErr
			}
			return nil, v.exception("Undefined function.", &span)
		}

		if isCssMathFunction(expr.Name) {
			// min/max/round/abs keep the legacy Sass unit behavior when every
			// positional argument is calculation-safe, so the legacy function
			// name is threaded through; all other math functions evaluate as
			// plain calculations.
			lower := strings.ToLower(expr.Name)
			if lower == "min" || lower == "max" || lower == "round" || lower == "abs" {
				if expr.Arguments().Named.Len() == 0 &&
					expr.Arguments().Rest == nil &&
					allCalculationSafe(expr.Arguments().Positional) {
					return v.visitCalculation(expr, lower)
				}
			} else {
				return v.visitCssMathFunction(expr)
			}
		}

		if !v.stylesheet.IsPlainCss() {
			if bfn, ok := v.builtInFunctions[expr.Name]; ok {
				fn = bfn
			} else {
				fn = functions.NewPlainCssCallable(expr.OriginalName)
			}
		} else {
			fn = functions.NewPlainCssCallable(expr.OriginalName)
		}
	}

	if udFn, ok := fn.(*functions.UserDefinedCallable); ok {
		if _, isMixin := udFn.Declaration().(*value.MixinRule); isMixin {
			return nil, sasscommon.NewSassScriptException("Mixin used as function.", nil)
		}
	}

	oldInFunction := v.inFunction
	v.inFunction = true
	defer func() { v.inFunction = oldInFunction }()

	// Every callable shape (built-in, user-defined, plain CSS) is invoked
	// through the same invokeCallable dispatch.
	return addErrorSpan(v, expr, func() (value.Value, error) {
		return v.invokeCallable(fn, expr, expr.Arguments())
	})
}

// isCssMathFunction returns whether name is a CSS math function,
// matched case-insensitively.
func isCssMathFunction(name string) bool {
	return cssMathFunctionNames[strings.ToLower(name)]
}

// allCalculationSafe returns whether every expression in the list can appear
// in a calculation. An empty list counts as safe.
func allCalculationSafe(exprs []value.Expression) bool {
	for _, expr := range exprs {
		safe, err := expr.IsCalculationSafe()
		if err != nil || !safe {
			return false
		}
	}
	return true
}

// visitCssMathFunction evaluates a CSS math function call as a calculation
// with no legacy function behavior.
func (v *EvaluateVisitor) visitCssMathFunction(expr *value.FunctionExpression) (value.Value, error) {
	return v.visitCalculation(expr, "")
}

// visitCalculation evaluates a CSS math function by recursively walking its
// arguments through visitCalculationExpression.
//
// When inLegacySassFunction names a function, unitless numbers may be added
// to and subtracted from numbers with units for backwards compatibility with
// the old global min(), max(), round(), and abs(); the name is used for
// deprecation warnings. Keyword and rest arguments are always rejected, and
// inside a supports declaration the result stays unsimplified.
func (v *EvaluateVisitor) visitCalculation(expr *value.FunctionExpression, inLegacySassFunction string) (value.Value, error) {
	name := strings.ToLower(expr.Name)
	args := expr.Arguments()

	// Calculations take positionals only; keyword and rest arguments are
	// rejected before the arity check below.
	if args.Named.Len() > 0 {
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Keyword arguments can't be used with calculations.", &sp)
	}
	if args.Rest != nil {
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Rest arguments can't be used with calculations.", &sp)
	}
	if len(args.Positional) == 0 {
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Missing argument.", &sp)
	}
	maxArgs := calculationMaxArgs(name)
	if maxArgs > 0 && len(args.Positional) > maxArgs {
		msg := fmt.Sprintf(
			"Only %d %s allowed, but %d %s passed.",
			maxArgs,
			util.Pluralize("argument", maxArgs, nil),
			len(args.Positional),
			util.Pluralize("was", len(args.Positional), new("were")),
		)
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception(msg, &sp)
	}

	// Each positional argument is evaluated as a calculation component.
	evaluatedArgs := make([]any, len(args.Positional))
	for i, arg := range args.Positional {
		val, err := v.visitCalculationExpression(arg, inLegacySassFunction)
		if err != nil {
			// Calculation-operand failures surface as script errors; attach
			// the call span and stack trace here since the inner evaluation
			// has no call-site context of its own.
			if sse, ok := errors.AsType[*sasscommon.SassScriptException](err); ok {
				sp, spErr := expr.Span()
				if spErr != nil {
					return nil, spErr
				}
				return nil, sasscommon.ThrowWithTrace(
					&sasscommon.SassRuntimeException{
						Message: sse.Error(), Span: sp, Trace: v.stackTrace(sp),
					}, sse)
			}
			return nil, err
		}
		evaluatedArgs[i] = val
	}

	if v.inSupportsDeclaration {
		return value.NewUnsimplified(name, evaluatedArgs...), nil
	}

	oldCallableNode := v.ec.CallableNode()
	v.ec.SetCallableNode(expr)
	defer func() { v.ec.SetCallableNode(oldCallableNode) }()

	var result value.Value
	var calcErr error
	switch name {
	case "calc":
		result, calcErr = value.NewCalc(evaluatedArgs[0])
	case "min":
		result, calcErr = value.NewMin(evaluatedArgs...)
	case "max":
		result, calcErr = value.NewMax(evaluatedArgs...)
	case "clamp":
		result, calcErr = value.NewClamp(getArg(evaluatedArgs, 0), getArg(evaluatedArgs, 1), getArg(evaluatedArgs, 2))
	case "round":
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		result, calcErr = value.NewRoundInternal(
			getArg(evaluatedArgs, 0),
			getArg(evaluatedArgs, 1),
			getArg(evaluatedArgs, 2),
			pointerOrNil(inLegacySassFunction),
			func(message string, deprecation *deprecation.Deprecation) error {
				return v.warn(message, sp, deprecation)
			},
		)
	case "mod":
		result, calcErr = value.NewMod(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	case "rem":
		result, calcErr = value.NewRem(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	case "sin":
		result, calcErr = value.NewSin(evaluatedArgs[0])
	case "cos":
		result, calcErr = value.NewCos(evaluatedArgs[0])
	case "tan":
		result, calcErr = value.NewTan(evaluatedArgs[0])
	case "asin":
		result, calcErr = value.NewAsin(evaluatedArgs[0])
	case "acos":
		result, calcErr = value.NewAcos(evaluatedArgs[0])
	case "atan":
		result, calcErr = value.NewAtan(evaluatedArgs[0])
	case "atan2":
		result, calcErr = value.NewAtan2(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	case "pow":
		result, calcErr = value.NewPow(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	case "sqrt":
		result, calcErr = value.NewSqrt(evaluatedArgs[0])
	case "hypot":
		result, calcErr = value.NewHypot(evaluatedArgs...)
	case "log":
		result, calcErr = value.NewLog(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	case "exp":
		result, calcErr = value.NewExp(evaluatedArgs[0])
	case "abs":
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		result, calcErr = value.NewAbsInternal(evaluatedArgs[0],
			func(message string, deprecation *deprecation.Deprecation) error {
				return v.warn(message, sp, deprecation)
			},
		)
	case "sign":
		result, calcErr = value.NewSign(evaluatedArgs[0])
	case "calc-size":
		result, calcErr = value.NewCalcSize(evaluatedArgs[0], getArg(evaluatedArgs, 1))
	default:
		sp, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception(fmt.Sprintf(`Unknown calculation name "%s".`, expr.Name), &sp)
	}
	if calcErr != nil {
		if sse, ok := errors.AsType[*sasscommon.SassScriptException](calcErr); ok {
			// Simplification throws for incompatible units, but the evaluated
			// arguments have lost their spans; re-verify against the original
			// argument nodes so the error can label each offending operand.
			if strings.Contains(sse.Message, "compatible") {
				if err := v.verifyCompatibleNumbersExpressions(evaluatedArgs, args.Positional); err != nil {
					return nil, err
				}
			}
			sp, spErr := expr.Span()
			if spErr != nil {
				return nil, spErr
			}
			return nil, sasscommon.ThrowWithTrace(
				&sasscommon.SassRuntimeException{
					Message: sse.Error(), Span: sp, Trace: v.stackTrace(sp),
				}, sse)
		}
		return nil, calcErr
	}
	return result, nil
}

// calculationMaxArgs returns the maximum number of positional arguments a CSS
// calculation function accepts, or 0 for functions with no upper bound. This
// backs the arity check: every calculation still requires at least one
// argument.
func calculationMaxArgs(name string) int {
	switch name {
	case "calc", "sqrt", "sin", "cos", "tan", "asin", "acos", "atan", "abs", "exp", "sign":
		return 1
	case "min", "max", "hypot":
		return 0
	case "pow", "atan2", "log", "mod", "rem", "calc-size":
		return 2
	case "round", "clamp":
		return 3
	default:
		return 0
	}
}

// pointerOrNil converts an empty legacy-function name to nil so callers can
// distinguish "no legacy function" from a named one.
func pointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// visitCalculationExpression evaluates node as a component of a calculation.
//
// It returns any rather than Value because operations may produce
// CalculationOperation nodes, not just values. Parenthesized results keep
// their parentheses when they wrap to strings; recognized constant names
// (pi, e, infinity, -infinity, nan) become numbers while other safe strings
// interpolate to text. Binary operations check operator whitespace and run
// under the operation span; space-separated lists of three or more elements
// join to text after an adjacency check, re-parenthesizing operations that
// came from parenthesized source. Anything else must be provably unsafe to
// reach the final error.
func (v *EvaluateVisitor) visitCalculationExpression(node value.Expression, inLegacySassFunction string) (any, error) {
	switch n := node.(type) {
	case *value.ParenthesizedExpression:
		inner, err := v.visitCalculationExpression(n.Expression, inLegacySassFunction)
		if err != nil {
			return nil, err
		}
		if s, ok := inner.(*value.SassString); ok {
			return &value.SassString{Text: "(" + s.Text + ")", HasQuotes: false}, nil
		}
		return inner, nil

	case *value.StringExpression:
		safe, err := n.IsCalculationSafe()
		if err != nil {
			return nil, err
		}
		if !safe {
			break
		}
		if n.HasQuotes {
			// Calculation-safe strings are unquoted by construction; a quoted
			// one here indicates a compiler bug.
			panic("BUG: Unexpected quoted string in calculation-safe expression.")
		}
		plain := n.Text.AsPlain()
		if plain == nil {
			result, err := v.performInterpolation(n.Text, nil)
			if err != nil {
				return nil, err
			}
			return &value.SassString{Text: result, HasQuotes: false}, nil
		}
		switch strings.ToLower(*plain) {
		case "pi":
			return value.NewSassNumber(math.Pi, nil), nil
		case "e":
			return value.NewSassNumber(math.E, nil), nil
		case "infinity":
			return value.NewSassNumber(math.Inf(1), nil), nil
		case "-infinity":
			return value.NewSassNumber(math.Inf(-1), nil), nil
		case "nan":
			return value.NewSassNumber(math.NaN(), nil), nil
		default:
			result, err := v.performInterpolation(n.Text, nil)
			if err != nil {
				return nil, err
			}
			return &value.SassString{Text: result, HasQuotes: false}, nil
		}

	case *value.BinaryOperationExpression:
		// `+` and `-` read as math only with surrounding whitespace; without
		// it they may be part of a larger token (e.g. a signed number).
		if err := v.checkWhitespaceAroundCalculationOperator(n); err != nil {
			return nil, err
		}
		// The whole operation, including operand evaluation, runs under the
		// binary node's span so operand failures point at the operation.
		return addExceptionSpan(v, n, func() (any, error) {
			left, err := v.visitCalculationExpression(n.Left, inLegacySassFunction)
			if err != nil {
				return nil, err
			}
			right, err := v.visitCalculationExpression(n.Right, inLegacySassFunction)
			if err != nil {
				return nil, err
			}
			opSpan, err := n.OperatorSpan()
			if err != nil {
				return nil, err
			}
			op, err := v.binaryOperatorToCalculationOperator(n.Operator, opSpan)
			if err != nil {
				return nil, err
			}
			var legacyFn *string
			if inLegacySassFunction != "" {
				legacyFn = &inLegacySassFunction
			}
			nSpan, err := n.Span()
			if err != nil {
				return nil, err
			}
			result, operr := value.OperateInternal(
				op, left, right,
				legacyFn,
				!v.inSupportsDeclaration,
				func(message string, deprecation *deprecation.Deprecation) error {
					return v.warn(message, nSpan, deprecation)
				},
			)
			if operr != nil {
				return nil, operr
			}
			return result, nil
		}, nil)

	case *value.NumberExpression, *value.VariableExpression, *value.FunctionExpression, *value.LegacyIfExpression:
		result, err := v.eval(node)
		if err != nil {
			return nil, err
		}
		switch r := result.(type) {
		case value.SassNumber:
			return r, nil
		case *value.SassCalculation:
			return r, nil
		case *value.SassString:
			if !r.HasQuotes {
				return r, nil
			}
		}
		resultStr, err := result.String()
		if err != nil {
			return nil, err
		}
		span, spanErr := node.Span()
		if spanErr != nil {
			return nil, spanErr
		}
		return nil, &sasscommon.SassRuntimeException{
			Message: fmt.Sprintf("Value %s can't be used in a calculation.", resultStr),
			Span:    span,
			Trace:   v.stackTrace(span),
		}

	case *value.ListExpression:
		if !n.HasBrackets && n.Separator == value.ListSeparatorSpace && len(n.Contents) >= 2 {
			var elements []any
			for _, elem := range n.Contents {
				val, err := v.visitCalculationExpression(elem, inLegacySassFunction)
				if err != nil {
					return nil, err
				}
				elements = append(elements, val)
			}

			// Adjacent non-string elements mean an operator is missing.
			if err := v.checkAdjacentCalculationValues(elements, n); err != nil {
				return nil, err
			}

			// Operations that were parenthesized in source stay parenthesized
			// in the joined text so their grouping survives.
			for i := range elements {
				if _, ok := elements[i].(*value.CalculationOperation); ok {
					if _, isParen := n.Contents[i].(*value.ParenthesizedExpression); isParen {
						elemStr, err := value.SprintAny(elements[i])
						if err != nil {
							return nil, err
						}
						elements[i] = &value.SassString{
							Text:      "(" + elemStr + ")",
							HasQuotes: false,
						}
					}
				}
			}

			var parts []string
			for _, e := range elements {
				s, err := value.SprintAny(e)
				if err != nil {
					return nil, err
				}
				parts = append(parts, s)
			}
			return &value.SassString{Text: strings.Join(parts, " "), HasQuotes: false}, nil
		}
	}

	// Anything reaching here is not one of the evaluable shapes above; it is an
	// error unless the expression claims to be calculation-safe, which would
	// mean a shape was missed.
	safe, err := node.IsCalculationSafe()
	if err != nil {
		return nil, err
	}
	if safe {
		panic("Expected expression not to be calculation-safe.")
	}
	span, spanErr := node.Span()
	if spanErr != nil {
		return nil, spanErr
	}
	return nil, &sasscommon.SassRuntimeException{
		Message: "This expression can't be used in a calculation.",
		Span:    span,
		Trace:   v.stackTrace(span),
	}
}

// checkWhitespaceAroundCalculationOperator returns an error when node uses `+`
// or `-` in a calculation without surrounding whitespace. Only those two
// operators need the check; `*` and `/` are unambiguous.
func (v *EvaluateVisitor) checkWhitespaceAroundCalculationOperator(node *value.BinaryOperationExpression) error {
	if node.Operator != value.BinaryOperatorPlus && node.Operator != value.BinaryOperatorMinus {
		return nil
	}

	leftSpan, err := node.Left.Span()
	if err != nil {
		return err
	}
	rightSpan, err := node.Right.Span()
	if err != nil {
		return err
	}

	// Both operands always parse from a single file, so mismatched or inverted
	// source positions should be impossible; bail out quietly rather than
	// crashing oddly if that assumption ever breaks.
	leftURL, err := leftSpan.SourceURL()
	if err != nil {
		return err
	}
	rightURL, err := rightSpan.SourceURL()
	if err != nil {
		return err
	}
	if (leftURL == nil) != (rightURL == nil) || (leftURL != nil && leftURL.String() != rightURL.String()) {
		return nil
	}
	leftEndLoc, err := leftSpan.EndLocation()
	if err != nil {
		return err
	}
	rightStartLoc, err := rightSpan.StartLocation()
	if err != nil {
		return err
	}
	leftEnd := leftEndLoc.Offset
	rightStart := rightStartLoc.Offset
	if leftEnd >= rightStart {
		return nil
	}

	leftFile, err := leftSpan.File()
	if err != nil {
		return err
	}
	textBetween := leftFile.GetText(leftEnd, rightStart)
	if len(textBetween) == 0 {
		return nil
	}
	opSpan2, err := node.OperatorSpan()
	if err != nil {
		return err
	}
	first := rune(textBetween[0])
	last := rune(textBetween[len(textBetween)-1])
	if !(unicode.IsSpace(first) || first == '/') || !(unicode.IsSpace(last) || last == '/') {
		return &sasscommon.SassRuntimeException{
			Message: "\"+\" and \"-\" must be surrounded by whitespace in calculations.",
			Span:    opSpan2,
			Trace:   v.stackTrace(opSpan2),
		}
	}
	return nil
}

// binaryOperatorToCalculationOperator maps a SassScript operator to its
// calculation counterpart. Operators with no calculation meaning (comparisons,
// equality, `%`) are rejected at the operator span.
func (v *EvaluateVisitor) binaryOperatorToCalculationOperator(op value.BinaryOperator, operatorSpan sasscommon.FileSpan) (value.CalculationOperator, error) {
	switch op {
	case value.BinaryOperatorPlus:
		return value.CalculationOperatorPlus, nil
	case value.BinaryOperatorMinus:
		return value.CalculationOperatorMinus, nil
	case value.BinaryOperatorTimes:
		return value.CalculationOperatorTimes, nil
	case value.BinaryOperatorDividedBy:
		return value.CalculationOperatorDividedBy, nil
	default:
		return 0, &sasscommon.SassRuntimeException{
			Message: "This operation can't be used in a calculation.",
			Span:    operatorSpan,
			Trace:   v.stackTrace(operatorSpan),
		}
	}
}

// checkAdjacentCalculationValues returns an error when elements holds two
// adjacent non-string values, which means an operator is missing between them.
// Strings act as separators, so only runs of non-strings are examined.
func (v *EvaluateVisitor) checkAdjacentCalculationValues(elements []any, node *value.ListExpression) error {
	if len(elements) <= 1 {
		return nil
	}

	for i := 1; i < len(elements); i++ {
		previous := elements[i-1]
		current := elements[i]
		if _, ok := previous.(*value.SassString); ok {
			continue
		}
		if _, ok := current.(*value.SassString); ok {
			continue
		}

		prevNode := node.Contents[i-1]
		currNode := node.Contents[i]
		if isUnaryOrNegative(currNode) {
			// `calc(1 -2)` parses as a space-separated list whose second
			// element is a unary operator or a negative number. Reporting a
			// missing operator would mislead, so narrow the span to the sign
			// and report the whitespace rule instead.
			currNodeSpan, err := currNode.Span()
			if err != nil {
				return err
			}
			subspan, err := currNodeSpan.Subspan(0, 1)
			if err != nil {
				return err
			}
			return &sasscommon.SassRuntimeException{
				Message: "\"+\" and \"-\" must be surrounded by whitespace in calculations.",
				Span:    subspan,
				Trace:   v.stackTrace(subspan),
			}
		}
		// Otherwise the operator is genuinely missing; span both nodes so the
		// error covers the gap.
		prevNodeSpan, err := prevNode.Span()
		if err != nil {
			return err
		}
		currNodeSpan, err := currNode.Span()
		if err != nil {
			return err
		}
		expanded, err := prevNodeSpan.Expand(currNodeSpan)
		if err != nil {
			return err
		}
		return &sasscommon.SassRuntimeException{
			Message: "Missing math operator.",
			Span:    expanded,
			Trace:   v.stackTrace(expanded),
		}
	}

	return nil
}

// isUnaryOrNegative returns whether node is a unary plus/minus expression or
// a negative number literal, the two shapes a missing-whitespace sign can
// take after parsing.
func isUnaryOrNegative(node value.Expression) bool {
	if ue, ok := node.(*value.UnaryOperationExpression); ok {
		return ue.Operator == value.UnaryOperatorMinus || ue.Operator == value.UnaryOperatorPlus
	}
	if ne, ok := node.(*value.NumberExpression); ok {
		return ne.Value < 0
	}
	return false
}

// getArg returns the evaluated calculation argument at index i, or nil when
// the function was called with fewer arguments.
func getArg(args []any, i int) any {
	if i < len(args) {
		return args[i]
	}
	return nil
}

// verifyCompatibleNumbersExpressions checks that the numbers in args all have
// units usable in CSS calculations, using nodesWithSpans (which parallels
// args) for error spans. Complex units are rejected per argument; pairwise
// incompatible units report a MultiSpan error labeling each operand. This
// logic is largely duplicated in the calculation constructors, and most
// changes here should be reflected there as well.
func (v *EvaluateVisitor) verifyCompatibleNumbersExpressions(args []any, nodesWithSpans []value.Expression) error {
	for i, arg := range args {
		if num, ok := arg.(value.SassNumber); ok && num.HasComplexUnits() {
			sp, err := nodesWithSpans[i].Span()
			if err != nil {
				return err
			}
			str, err := num.String()
			if err != nil {
				return err
			}
			return &sasscommon.SassRuntimeException{
				Message: fmt.Sprintf("Number %s isn't compatible with CSS calculations.", str),
				Span:    sp,
				Trace:   v.stackTrace(sp),
			}
		}
	}

	for i := 0; i < len(args)-1; i++ {
		num1, ok := args[i].(value.SassNumber)
		if !ok {
			continue
		}
		for j := i + 1; j < len(args); j++ {
			num2, ok := args[j].(value.SassNumber)
			if !ok {
				continue
			}
			if num1.HasPossiblyCompatibleUnits(num2) {
				continue
			}
			spI, err := nodesWithSpans[i].Span()
			if err != nil {
				return err
			}
			spJ, err := nodesWithSpans[j].Span()
			if err != nil {
				return err
			}
			str1, err := num1.String()
			if err != nil {
				return err
			}
			str2, err := num2.String()
			if err != nil {
				return err
			}
			return &sasscommon.MultiSpanSassRuntimeException{
				Message:        fmt.Sprintf("%s and %s are incompatible.", str1, str2),
				Span:           spI,
				PrimaryLabel:   str1,
				SecondarySpans: map[sasscommon.FileSpan]string{spJ: str2},
				Trace:          v.stackTrace(spI),
			}
		}
	}
	return nil
}

// VisitIfExpression evaluates the modern if() function. Branches with Sass
// conditions decide immediately: the first true branch evaluates and returns,
// while a true branch after CSS conditions appends an `else` entry. Branches
// with CSS conditions accumulate as strings and render as an unquoted
// `if(cond: value; ...)` using each value's CSS form. No true branch at all
// yields null.
func (v *EvaluateVisitor) VisitIfExpression(expr *value.IfExpression) (value.Value, error) {
	type resultPair struct {
		condition string
		exprVal   value.Value
	}
	var results []resultPair

	for _, branch := range expr.Branches {
		var result any = true
		if branch.Condition != nil {
			var err error
			result, err = branch.Condition.AcceptAny(v)
			if err != nil {
				return nil, err
			}
		}

		switch r := result.(type) {
		case string:
			results = append(results, resultPair{r, nil})
			exprVal, err := v.eval(branch.Expression)
			if err != nil {
				return nil, err
			}
			results[len(results)-1].exprVal = exprVal

		case bool:
			if r && results != nil {
				// Some previous condition was a CSS function, accumulate.
				exprVal, err := v.eval(branch.Expression)
				if err != nil {
					return nil, err
				}
				results = append(results, resultPair{"else", exprVal})
			} else if r {
				return v.eval(branch.Expression)
			}
		}
	}

	if results == nil {
		return value.Null, nil
	}

	var parts []string
	for _, r := range results {
		// Branch values render in CSS form, not inspect form, so e.g. strings
		// keep the quoting the stylesheet would emit.
		valStr, err := r.exprVal.ToCssString(true)
		if err != nil {
			return nil, err
		}
		parts = append(parts, r.condition+": "+valStr)
	}
	return &value.SassString{Text: "if(" + strings.Join(parts, "; ") + ")", HasQuotes: false}, nil
}

// VisitIfConditionSass evaluates a Sass condition to its truthiness, yielding
// a bool for the if() dispatch.
func (v *EvaluateVisitor) VisitIfConditionSass(node *value.IfConditionSass) (any, error) {
	val, err := node.Expression.AcceptValue(v)
	if err != nil {
		return nil, err
	}
	return val.IsTruthy(), nil
}

// VisitIfConditionParenthesized evaluates a parenthesized if() condition,
// re-wrapping CSS text in parentheses and passing bools through.
func (v *EvaluateVisitor) VisitIfConditionParenthesized(node *value.IfConditionParenthesized) (any, error) {
	result, err := node.Expression.AcceptAny(v)
	if err != nil {
		return nil, err
	}
	if s, ok := result.(string); ok {
		return "(" + s + ")", nil
	}
	return result, nil
}

// VisitIfConditionNegation evaluates a `not` if() condition, prefixing CSS
// text and negating bools.
func (v *EvaluateVisitor) VisitIfConditionNegation(node *value.IfConditionNegation) (any, error) {
	result, err := node.Expression.AcceptAny(v)
	if err != nil {
		return nil, err
	}
	if s, ok := result.(string); ok {
		return "not " + s, nil
	}
	return !result.(bool), nil
}

// VisitIfConditionOperation evaluates an `and`/`or` if() condition. Bool
// operands short-circuit (`and` fails on the first false, `or` succeeds on
// the first true) while CSS operands accumulate as strings. A lone surviving
// parenthesized operand sheds its parentheses, which is always valid because
// parentheses hold an if-group and the operation itself is an if-group;
// anything left joins with the operator.
func (v *EvaluateVisitor) VisitIfConditionOperation(node *value.IfConditionOperation) (any, error) {
	var values []struct {
		expr value.IfConditionExpression
		str  string
	}
	for _, e := range node.Expressions {
		result, err := e.AcceptAny(v)
		if err != nil {
			return nil, err
		}
		if s, ok := result.(string); ok {
			values = append(values, struct {
				expr value.IfConditionExpression
				str  string
			}{e, s})
		} else if node.Op == value.BooleanOperatorAnd {
			if !result.(bool) {
				return false, nil
			}
		} else {
			if result.(bool) {
				return true, nil
			}
		}
	}
	if len(values) == 0 {
		return node.Op == value.BooleanOperatorAnd, nil
	}
	// A sole surviving parenthesized operand drops its outer parentheses.
	if len(values) == 1 {
		if _, ok := values[0].expr.(*value.IfConditionParenthesized); ok {
			s := values[0].str
			if len(s) >= 2 {
				return s[1 : len(s)-1], nil
			}
		}
	}
	var parts []string
	for _, p := range values {
		parts = append(parts, p.str)
	}
	opStr := " and "
	if node.Op == value.BooleanOperatorOr {
		opStr = " or "
	}
	return strings.Join(parts, opStr), nil
}

// VisitIfConditionFunction interpolates an unknown function in an if()
// condition to CSS text.
func (v *EvaluateVisitor) VisitIfConditionFunction(node *value.IfConditionFunction) (any, error) {
	name, err := v.performInterpolation(node.Name, nil)
	if err != nil {
		return nil, err
	}
	args, err := v.performInterpolation(node.Arguments, nil)
	if err != nil {
		return nil, err
	}
	return name + "(" + args + ")", nil
}

// VisitIfConditionRaw interpolates raw if() condition text to CSS text.
func (v *EvaluateVisitor) VisitIfConditionRaw(node *value.IfConditionRaw) (any, error) {
	return v.performInterpolation(node.Text, nil)
}

// VisitLegacyIfExpression evaluates the legacy if($condition, $if-true,
// $if-false) function. Macro arguments are evaluated before arity checking so
// defaults and rest args resolve first; the condition may come positionally
// or by name, and the winning branch evaluates with slash-separation stripped
// against the branch node for deprecation spans.
func (v *EvaluateVisitor) VisitLegacyIfExpression(expr *value.LegacyIfExpression) (value.Value, error) {
	// Arguments evaluate as a macro call so defaults, rest, and keyword rest
	// resolve before the arity check.
	positional, named, err := v.evaluateMacroArguments(expr.Arguments(), expr)
	if err != nil {
		return nil, err
	}

	// The arity check reports against the legacy if() declaration at the call
	// span.
	names := make(map[string]struct{})
	for k := range named.Keys() {
		names[k] = struct{}{}
	}
	if err := value.LegacyIfDeclaration.Verify(len(positional), names); err != nil {
		span, spanErr := expr.Span()
		if spanErr != nil {
			return nil, spanErr
		}
		return nil, v.exception(err.Error(), &span)
	}

	// Each branch resolves positionally first, then by name.
	var conditionExpr value.Expression
	if len(positional) > 0 {
		conditionExpr = positional[0]
	} else {
		conditionExpr, _ = named.Get("condition")
	}

	var ifTrueExpr value.Expression
	if len(positional) > 1 {
		ifTrueExpr = positional[1]
	} else {
		ifTrueExpr, _ = named.Get("if-true")
	}

	var ifFalseExpr value.Expression
	if len(positional) > 2 {
		ifFalseExpr = positional[2]
	} else {
		ifFalseExpr, _ = named.Get("if-false")
	}

	// Truthiness picks the branch; the winner evaluates with slash-separation
	// stripped, using the branch node (not the whole call) for spans.
	condition, err := v.eval(conditionExpr)
	if err != nil {
		return nil, err
	}

	var chosenExpr value.Expression
	if condition.IsTruthy() {
		chosenExpr = ifTrueExpr
	} else {
		chosenExpr = ifFalseExpr
	}

	// The winner evaluates with slash-separation stripped, using the branch
	// node (not the whole call) for spans.
	result, err := v.eval(chosenExpr)
	if err != nil {
		return nil, err
	}
	return v.withoutSlash(result, chosenExpr)
}

// VisitParenthesizedExpression evaluates the inner expression. Parentheses
// are rejected in plain-CSS stylesheets; otherwise they are transparent.
func (v *EvaluateVisitor) VisitParenthesizedExpression(expr *value.ParenthesizedExpression) (value.Value, error) {
	if v.stylesheet.IsPlainCss() {
		span, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Parentheses aren't allowed in plain CSS.", &span)
	}
	return v.eval(expr.Expression)
}

// VisitSelectorExpression returns the enclosing style rule's original selector
// as a Sass list, or null when evaluated outside a style rule. At-root context
// is ignored so selectors still resolve inside @at-root.
func (v *EvaluateVisitor) VisitSelectorExpression(expr *value.SelectorExpression) (value.Value, error) {
	sr := v.styleRuleIgnoringAtRoot
	if sr != nil {
		return sr.OriginalSelector().AsSassList()
	}
	return value.Null, nil
}

// VisitInterpolatedFunctionExpression evaluates a call whose name contains
// interpolation. The interpolated name always denotes a plain-CSS function;
// the call runs with inFunction set and its errors wrapped at the call span.
func (v *EvaluateVisitor) VisitInterpolatedFunctionExpression(expr *value.InterpolatedFunctionExpression) (value.Value, error) {
	name, err := v.performInterpolation(expr.Name, nil)
	if err != nil {
		return nil, err
	}
	fn := functions.NewPlainCssCallable(name)

	oldInFunction := v.inFunction
	v.inFunction = true
	defer func() { v.inFunction = oldInFunction }()

	// Like VisitFunctionExpression, every callable shape goes through the same
	// invokeCallable dispatch.
	return addErrorSpan(v, expr, func() (value.Value, error) {
		return v.invokeCallable(fn, expr, expr.Arguments())
	})
}

// visitSupportsCondition evaluates a @supports condition to plain CSS text.
// Operations join their parenthesized operands with the operator; declarations
// evaluate with inSupportsDeclaration set so nested calculations stay
// unsimplified, and omit the space after the colon for custom properties.
func (v *EvaluateVisitor) visitSupportsCondition(condition value.SupportsCondition) (string, error) {
	switch c := condition.(type) {
	case *value.SupportsOperation:
		op := c.Operator
		left, err := v.parenthesize(c.Left, &op)
		if err != nil {
			return "", err
		}
		right, err := v.parenthesize(c.Right, &op)
		if err != nil {
			return "", err
		}
		return left + " " + c.Operator.String() + " " + right, nil

	case *value.SupportsNegation:
		cond, err := v.parenthesize(c.Condition, nil)
		if err != nil {
			return "", err
		}
		return "not " + cond, nil

	case *value.SupportsInterpolation:
		return v.evaluateToCss(c.Expression, false)

	case *value.SupportsDeclaration:
		return v.withSupportsDeclaration(func() (string, error) {
			name, err := v.evaluateToCss(c.Name, true)
			if err != nil {
				return "", err
			}
			val, err := v.evaluateToCss(c.Value, true)
			if err != nil {
				return "", err
			}
			isCustomProp := false
			if se, ok := c.Name.(*value.StringExpression); ok && !se.HasQuotes {
				if se.Text != nil && strings.HasPrefix(se.Text.InitialPlain(), "--") {
					isCustomProp = true
				}
			}
			sep := " "
			if isCustomProp {
				sep = ""
			}
			return "(" + name + ":" + sep + val + ")", nil
		})

	case *value.SupportsFunction:
		name, err := v.performInterpolation(c.Name, nil)
		if err != nil {
			return "", err
		}
		args, err := v.performInterpolation(c.Arguments, nil)
		if err != nil {
			return "", err
		}
		return name + "(" + args + ")", nil

	case *value.SupportsAnything:
		contents, err := v.performInterpolation(c.Contents, nil)
		if err != nil {
			return "", err
		}
		return "(" + contents + ")", nil

	default:
		panic(fmt.Sprintf("Unknown supports condition type %T.", condition))
	}
}

// parenthesize renders condition with parentheses when needed: negations are
// always parenthesized, and operations are parenthesized unless they share the
// surrounding operator (or there is none).
func (v *EvaluateVisitor) parenthesize(condition value.SupportsCondition, op *value.BooleanOperator) (string, error) {
	_, isNeg := condition.(*value.SupportsNegation)
	so, isOp := condition.(*value.SupportsOperation)
	if isNeg || (isOp && (op == nil || *op != so.Operator)) {
		cond, err := v.visitSupportsCondition(condition)
		if err != nil {
			return "", err
		}
		return "(" + cond + ")", nil
	}
	return v.visitSupportsCondition(condition)
}

// withSupportsDeclaration runs callback with inSupportsDeclaration set, so
// calculations evaluated inside a supports declaration stay unsimplified.
func (v *EvaluateVisitor) withSupportsDeclaration(callback func() (string, error)) (string, error) {
	old := v.inSupportsDeclaration
	v.inSupportsDeclaration = true
	defer func() { v.inSupportsDeclaration = old }()
	return callback()
}

// evaluateToCss evaluates expression and returns its CSS text, wrapping
// serialization failures at the expression span. The expression node (not a
// bare span) is passed through so spans that take real work to manufacture are
// only computed when an error actually needs one.
func (v *EvaluateVisitor) evaluateToCss(expression value.Expression, quote bool) (string, error) {
	val, err := v.eval(expression)
	if err != nil {
		return "", err
	}
	return v.serialize(val, expression, quote)
}

// VisitSupportsExpression evaluates a supports() condition to an unquoted
// CSS-text string.
func (v *EvaluateVisitor) VisitSupportsExpression(expr *value.SupportsExpression) (value.Value, error) {
	text, err := v.visitSupportsCondition(expr.Condition)
	if err != nil {
		return nil, err
	}
	return &value.SassString{Text: text, HasQuotes: false}, nil
}

// invokeCallable evaluates invocation as applied to fn and returns the result.
// The call-site node becomes the current callable node so warnings inside the
// call point at the call. Built-in results pass through withoutSlash (so e.g.
// slash-separated defaults warn); user-defined functions run their body
// statements until the first @return; plain-CSS functions serialize inline.
// An unknown Go callable type panics, since the set of shapes is fixed.
func (v *EvaluateVisitor) invokeCallable(fn sasscallable.Callable, nodeWithSpan sasscommon.AstNode, arguments *value.ArgumentList) (value.Value, error) {
	old := v.ec.CallableNode()
	v.ec.SetCallableNode(nodeWithSpan)
	defer func() { v.ec.SetCallableNode(old) }()

	if b, ok := fn.(*functions.BuiltInCallable); ok {
		result, err := v.runBuiltInCallable(b, nodeWithSpan, arguments)
		if err != nil {
			return nil, err
		}
		return v.withoutSlash(result, nodeWithSpan)
	} else if u, ok := fn.(*functions.UserDefinedCallable); ok {
		return v.runUserDefinedCallable(u, arguments, nodeWithSpan, func() (value.Value, error) {
			for _, child := range u.Declaration().GetChildren() {
				result, err := v.visitStatement(child)
				if err != nil {
					return nil, err
				}
				if result != nil {
					return result, nil
				}
			}
			return nil, nil
		})
	} else if p, ok := fn.(*functions.PlainCssCallable); ok {
		return v.emitPlainCssCallable(p, nodeWithSpan, arguments)
	}
	panic(fmt.Sprintf("unknown callable type: %T", fn))
}

// emitPlainCssCallable serializes an unknown function call as plain CSS by
// evaluating each positional argument (plus the rest argument, if any) to CSS
// text and joining them with ", ". Keyword arguments are rejected; a failure
// ending in "isn't a valid CSS value." is re-reported as a MultiSpan error
// labeling the call as an unknown function treated as plain CSS, while any
// other failure propagates unchanged.
func (v *EvaluateVisitor) emitPlainCssCallable(c *functions.PlainCssCallable, nodeWithSpan sasscommon.AstNode, args *value.ArgumentList) (value.Value, error) {
	if args.Named.Len() > 0 || args.KeywordRest != nil {
		sp, err := nodeWithSpan.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Plain CSS functions don't support keyword arguments.", &sp)
	}

	var sb strings.Builder
	sb.WriteString(c.Name())
	sb.WriteString("(")

	// The serialization loop collects its first error and reports it after
	// the block, so a single failure path handles positionals and rest alike.
	var serializeErr error
	{
		first := true
		for _, expression := range args.Positional {
			if !first {
				sb.WriteString(", ")
			}
			first = false

			cssStr, err := v.evaluateToCss(expression, true)
			if err != nil {
				serializeErr = err
				break
			}
			sb.WriteString(cssStr)
		}

		if serializeErr == nil && args.Rest != nil {
			rest, err := v.eval(args.Rest)
			if err != nil {
				serializeErr = err
			} else {
				if !first {
					sb.WriteString(", ")
				}
				cssStr, err := v.serialize(rest, args.Rest, true)
				if err != nil {
					serializeErr = err
				} else {
					sb.WriteString(cssStr)
				}
			}
		}
	}

	if serializeErr != nil {
		// Only invalid-CSS-value failures are rewrapped; anything else was
		// already reported at the right span and propagates as-is.
		if sre, ok := errors.AsType[*sasscommon.SassRuntimeException](serializeErr); ok {
			if !strings.HasSuffix(sre.Message, "isn't a valid CSS value.") {
				return nil, serializeErr
			}
			fnSpan, err := nodeWithSpan.Span()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.MultiSpanSassRuntimeException{
				Message:      sre.Message,
				Span:         sre.Span,
				PrimaryLabel: "value",
				SecondarySpans: map[sasscommon.FileSpan]string{
					fnSpan: "unknown function treated as plain CSS",
				},
				Trace: sre.Trace,
				Cause: serializeErr,
			}
		}
		return nil, serializeErr
	}

	sb.WriteString(")")
	return &value.SassString{Text: sb.String(), HasQuotes: false}, nil
}
