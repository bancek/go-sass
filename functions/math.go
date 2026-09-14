// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/math.dart

import (
	"fmt"
	"math"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmath"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalMathFunctions returns the deprecated global aliases of the sass:math
// functions. Each entry wraps the module function with a deprecation warning
// (the wrapper names the original module function — see docs/ref/functions.md
// for the ordering rule), except abs, which keeps its own body so that
// percentage arguments emit the abs-percent deprecation pointing at math.abs
// or an interpolated CSS abs(). The comparable and unitless names alias the
// compatible and is-unitless module functions.
// Matches Dart: global (math.dart).
func GlobalMathFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		absFunctionGlobal(),
		ceilFunction().WithDeprecationWarning("math", nil),
		floorFunction().WithDeprecationWarning("math", nil),
		maxFunction().WithDeprecationWarning("math", nil),
		minFunction().WithDeprecationWarning("math", nil),
		percentageFunction().WithDeprecationWarning("math", nil),
		randomFunction().WithDeprecationWarning("math", nil),
		roundFunction().WithDeprecationWarning("math", nil),
		unitFunction().WithDeprecationWarning("math", nil),
		comparableFunctionGlobal().WithDeprecationWarning("math", new("compatible")),
		unitlessFunctionGlobal().WithDeprecationWarning("math", new("is-unitless")),
	}
}

// MathModule returns the sass:math built-in module: the module-only callables
// (trig, log/pow/sqrt, div, clamp, hypot, compatible, is-unitless) together
// with the shared numeric functions, plus the module variables e, pi,
// epsilon, max-safe-integer, min-safe-integer, max-number, and min-number.
// min-number is the smallest positive subnormal (5e-324), not the smallest
// normal double.
// Matches Dart: module (math.dart).
func MathModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		absFunction(),
		acosFunction(),
		asinFunction(),
		atanFunction(),
		atan2Function(),
		ceilFunction(),
		clampFunction(),
		cosFunction(),
		compatibleFunction(),
		divFunction(),
		floorFunction(),
		hypotFunction(),
		isUnitlessModuleFunction(),
		logFunction(),
		maxFunction(),
		minFunction(),
		percentageFunction(),
		powFunction(),
		randomFunction(),
		roundFunction(),
		sinFunction(),
		sqrtFunction(),
		tanFunction(),
		unitFunction(),
	}
	return sassmodule.NewBuiltInModule("math", fns, nil, orderedmap.NewFromPairs(
		orderedmap.Pair[string, value.Value]{Key: "e", Val: value.NewUnitlessNumber(math.E)},
		orderedmap.Pair[string, value.Value]{Key: "pi", Val: value.NewUnitlessNumber(math.Pi)},
		orderedmap.Pair[string, value.Value]{Key: "epsilon", Val: value.NewUnitlessNumber(2.220446049250313e-16)},
		orderedmap.Pair[string, value.Value]{Key: "max-safe-integer", Val: value.NewUnitlessNumber(9007199254740991)},
		orderedmap.Pair[string, value.Value]{Key: "min-safe-integer", Val: value.NewUnitlessNumber(-9007199254740991)},
		orderedmap.Pair[string, value.Value]{Key: "max-number", Val: value.NewUnitlessNumber(math.MaxFloat64)},
		orderedmap.Pair[string, value.Value]{Key: "min-number", Val: value.NewUnitlessNumber(math.SmallestNonzeroFloat64)},
	))
}

// ---- Helpers ----

// numberFunction builds a math callable that maps a number's scalar value
// through transform and reattaches the original numerator/denominator units.
// Ports Dart's _numberFunction helper (math.dart).
func numberFunction(name string, transform func(float64) float64) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		return value.SassNumberWithUnits(
			transform(number.NumValue()),
			number.NumNumeratorUnits(),
			number.NumDenominatorUnits(),
		), nil
	})
}

// singleArgumentMathFunc builds a math callable that asserts a single $number
// argument and delegates unit checking and result construction to fn.
// Ports Dart's _singleArgumentMathFunc helper (math.dart).
func singleArgumentMathFunc(name string, fn func(value.SassNumber) (value.SassNumber, error)) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		return fn(number)
	})
}

// ---- Global-only functions ----
//
// The constructors below port Dart's bare `_function(...)` closures, which
// carry no per-function docs; each note names the Sass signature and any
// behavior Dart documents at the call site. Every callable registers under
// the sass:math URL, porting Dart's _function URL helper.

// absFunctionGlobal implements the deprecated global abs($number). Percentage
// arguments warn with the abs-percent deprecation (future CSS abs() versus
// math.abs preservation); all other arguments emit the plain global-builtin
// warning before returning the absolute value with units preserved.
func absFunctionGlobal() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("abs", "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		if number.HasUnit("%") {
			numStr, err := number.String()
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				fmt.Sprintf("Passing percentage units to the global abs() function is deprecated.\n"+
					"In the future, this will emit a CSS abs() function to be resolved by the browser.\n"+
					"To preserve current behavior: math.abs(%s)\n"+
					"To emit a CSS abs() now: abs(#{%s})\n"+
					"More info: https://sass-lang.com/d/abs-percent", numStr, numStr),
				deprecation.AbsPercent,
			); err != nil {
				return nil, err
			}
		} else {
			if err := warnForGlobalBuiltIn(ec, "math", "abs"); err != nil {
				return nil, err
			}
		}
		return value.SassNumberWithUnits(
			sassmath.Abs(number.NumValue()),
			number.NumNumeratorUnits(),
			number.NumDenominatorUnits(),
		), nil
	})
}

// absFunction implements math.abs($number): the absolute value with units
// preserved.
func absFunction() *BuiltInCallable {
	return numberFunction("abs", sassmath.Abs)
}

// dartIntOp normalizes the result of ceil/floor/round: Dart's
// `num.ceil()/floor()/round()` return 64-bit ints, which have no negative
// zero, so the `.toDouble()` round-trip yields `+0.0`. Go's float64 ops can
// return `-0.0` instead (#2840).
func dartIntOp(v float64, op func(float64) float64) float64 {
	r := op(v)
	if r == 0 {
		return 0
	}
	return r
}

// Bounding functions (Dart section): ceil, clamp, floor, max, min, round.
func ceilFunction() *BuiltInCallable {
	return numberFunction("ceil", func(v float64) float64 { return dartIntOp(v, math.Ceil) })
}

// floorFunction implements math.floor($number) with units preserved.
func floorFunction() *BuiltInCallable {
	return numberFunction("floor", func(v float64) float64 { return dartIntOp(v, math.Floor) })
}

// roundFunction implements math.round($number) with units preserved.
func roundFunction() *BuiltInCallable {
	return numberFunction("round", func(v float64) float64 { return dartIntOp(v, math.Round) })
}

// maxFunction implements math.max($numbers...): the greatest of one or more
// comparable numbers, or an error when no arguments are passed.
func maxFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("max", "$numbers...", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		numbers, err := args[0].AsList()
		if err != nil {
			return nil, err
		}
		if len(numbers) == 0 {
			return nil, sasscommon.NewSassScriptException("At least one argument must be passed.", nil)
		}
		var max value.SassNumber
		for _, v := range numbers {
			number, err := value.AssertNumber(v, nil)
			if err != nil {
				return nil, err
			}
			if max == nil {
				max = number
			} else {
				less, err := max.LessThan(number)
				if err != nil {
					return nil, err
				}
				if less.IsTruthy() {
					max = number
				}
			}
		}
		return max, nil
	})
}

// minFunction implements math.min($numbers...): the least of one or more
// comparable numbers, or an error when no arguments are passed.
func minFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("min", "$numbers...", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		numbers, err := args[0].AsList()
		if err != nil {
			return nil, err
		}
		if len(numbers) == 0 {
			return nil, sasscommon.NewSassScriptException("At least one argument must be passed.", nil)
		}
		var min value.SassNumber
		for _, v := range numbers {
			number, err := value.AssertNumber(v, nil)
			if err != nil {
				return nil, err
			}
			if min == nil {
				min = number
			} else {
				gt, err := min.GreaterThan(number)
				if err != nil {
					return nil, err
				}
				if gt.IsTruthy() {
					min = number
				}
			}
		}
		return min, nil
	})
}

// percentageFunction implements math.percentage($number): the unitless number
// scaled by 100 and given a % unit.
func percentageFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("percentage", "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		if err := number.AssertNoUnits(new("number")); err != nil {
			return nil, err
		}
		return value.NewSingleUnitNumber(number.NumValue()*100, "%"), nil
	})
}

// ---- Module functions (Dart sections: bounding, distance, exponential,
// trigonometric, unit, other) ----

// acosFunction implements math.acos($number) via value.AcosNumber, which owns
// unit checking.
func acosFunction() sasscallable.Callable {
	return singleArgumentMathFunc("acos", value.AcosNumber)
}

// asinFunction implements math.asin($number) via value.AsinNumber.
func asinFunction() sasscallable.Callable {
	return singleArgumentMathFunc("asin", value.AsinNumber)
}

// atanFunction implements math.atan($number) via value.AtanNumber.
func atanFunction() sasscallable.Callable {
	return singleArgumentMathFunc("atan", value.AtanNumber)
}

// atan2Function implements math.atan2($y, $x) via value.Atan2Number.
func atan2Function() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("atan2", "$y, $x", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		y, err := value.AssertNumber(args[0], new("y"))
		if err != nil {
			return nil, err
		}
		x, err := value.AssertNumber(args[1], new("x"))
		if err != nil {
			return nil, err
		}
		return value.Atan2Number(y, x)
	})
}

// clampFunction implements math.clamp($min, $number, $max): $min when $min
// orders at or above $max or $number, $max when $number orders at or above
// $max, else $number.
func clampFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("clamp", "$min, $number, $max", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		min, err := value.AssertNumber(args[0], new("min"))
		if err != nil {
			return nil, err
		}
		number, err := value.AssertNumber(args[1], new("number"))
		if err != nil {
			return nil, err
		}
		max, err := value.AssertNumber(args[2], new("max"))
		if err != nil {
			return nil, err
		}
		// Even though the converted values are discarded, ConvertValueToMatch
		// reports unit mismatches with the parameter names attached, which reads
		// better than the bare comparison errors would.
		if _, err := number.ConvertValueToMatch(min, new("number"), new("min")); err != nil {
			return nil, err
		}
		if _, err := max.ConvertValueToMatch(min, new("max"), new("min")); err != nil {
			return nil, err
		}
		gte, err := min.GreaterThanOrEquals(max)
		if err != nil {
			return nil, err
		}
		if gte.IsTruthy() {
			return min, nil
		}
		gte, err = min.GreaterThanOrEquals(number)
		if err != nil {
			return nil, err
		}
		if gte.IsTruthy() {
			return min, nil
		}
		gte, err = number.GreaterThanOrEquals(max)
		if err != nil {
			return nil, err
		}
		if gte.IsTruthy() {
			return max, nil
		}
		return number, nil
	})
}

// cosFunction implements math.cos($number) via value.CosNumber.
func cosFunction() sasscallable.Callable {
	return singleArgumentMathFunc("cos", value.CosNumber)
}

// divFunction implements math.div($number1, $number2). Non-number arguments
// still divide, but warn that div will only accept numbers in a future
// release and point callers at list.slash for slash separators.
func divFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("div", "$number1, $number2", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		_, ok1 := args[0].(value.SassNumber)
		_, ok2 := args[1].(value.SassNumber)
		if !ok1 || !ok2 {
			if err := ec.WarnWithDeprecation("math.div() will only support number arguments in a future release.\nUse list.slash() instead for a slash separator.", false); err != nil {
				return nil, err
			}
		}
		return args[0].DividedBy(args[1])
	})
}

// hypotFunction implements math.hypot($numbers...): the length of the
// hypotenuse. Every argument converts to match the first, so the result
// carries the first number's units; an empty argument list is an error.
func hypotFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("hypot", "$numbers...", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		numbers, err := args[0].AsList()
		if err != nil {
			return nil, err
		}
		if len(numbers) == 0 {
			return nil, sasscommon.NewSassScriptException("At least one argument must be passed.", nil)
		}
		parsed := make([]value.SassNumber, len(numbers))
		for i, v := range numbers {
			n, err := value.AssertNumber(v, nil)
			if err != nil {
				return nil, err
			}
			parsed[i] = n
		}
		var subtotal float64
		for i, n := range parsed {
			name := fmt.Sprintf("numbers[%d]", i+1)
			otherName := "numbers[1]"
			val, err := n.ConvertValueToMatch(parsed[0], &name, &otherName)
			if err != nil {
				return nil, err
			}
			subtotal += sassmath.Pow(val, 2)
		}
		return value.SassNumberWithUnits(
			sassmath.Sqrt(subtotal),
			parsed[0].NumNumeratorUnits(),
			parsed[0].NumDenominatorUnits(),
		), nil
	})
}

// isUnitlessFunction implements math.is-unitless($number), the module
// spelling; the global unitless alias below shares unitlessImpl.
func isUnitlessFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("is-unitless", "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		if !number.HasUnits() {
			return value.SassTrue, nil
		}
		return value.SassFalse, nil
	})
}

// logFunction implements math.log($number, $base: null): the natural log for
// a null base, else the log of $number in the given base. Unit checks live in
// value.LogNumber.
func logFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("log", "$number, $base: null", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		var base value.SassNumber
		if args[1] != value.Null {
			base, err = value.AssertNumber(args[1], new("base"))
			if err != nil {
				return nil, err
			}
		}
		return value.LogNumber(number, base)
	})
}

// powFunction implements math.pow($base, $exponent) via value.PowNumber.
func powFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("pow", "$base, $exponent", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		base, err := value.AssertNumber(args[0], new("base"))
		if err != nil {
			return nil, err
		}
		exponent, err := value.AssertNumber(args[1], new("exponent"))
		if err != nil {
			return nil, err
		}
		return value.PowNumber(base, exponent)
	})
}

// sinFunction implements math.sin($number) via value.SinNumber.
func sinFunction() sasscallable.Callable {
	return singleArgumentMathFunc("sin", value.SinNumber)
}

// sqrtFunction implements math.sqrt($number) via value.SqrtNumber.
func sqrtFunction() sasscallable.Callable {
	return singleArgumentMathFunc("sqrt", value.SqrtNumber)
}

// tanFunction implements math.tan($number) via value.TanNumber.
func tanFunction() sasscallable.Callable {
	return singleArgumentMathFunc("tan", value.TanNumber)
}

// unitFunction implements math.unit($number): the number's unit string as a
// quoted string, rendered by unitString below.
func unitFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("unit", "$number", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		number, err := value.AssertNumber(args[0], new("number"))
		if err != nil {
			return nil, err
		}
		return &value.SassString{Text: unitString(number), HasQuotes: true}, nil
	})
}

// unitString renders a number's units the way Dart's unitString getter does:
// empty for unitless, a single denominator unit with a ^-1 suffix, and
// parenthesized products when several denominator units are present.
func unitString(n value.SassNumber) string {
	numUnits := n.NumNumeratorUnits()
	denUnits := n.NumDenominatorUnits()
	switch {
	case len(numUnits) == 0 && len(denUnits) == 0:
		return ""
	case len(numUnits) == 0:
		if len(denUnits) == 1 {
			return denUnits[0] + "^-1"
		}
		return "(" + joinStrings(denUnits, "*") + ")^-1"
	case len(denUnits) == 0:
		return joinStrings(numUnits, "*")
	case len(denUnits) == 1:
		return joinStrings(numUnits, "*") + "/" + denUnits[0]
	default:
		return joinStrings(numUnits, "*") + "/(" + joinStrings(denUnits, "*") + ")"
	}
}

// joinStrings is Go-only glue for the unit rendering above (Dart relies on
// string joining from its number formatting).
func joinStrings(strs []string, sep string) string {
	var result strings.Builder
	for i, s := range strs {
		if i > 0 {
			result.WriteString(sep)
		}
		result.WriteString(s)
	}
	return result.String()
}

// comparableFunctionGlobal implements the deprecated global comparable($number1,
// $number2), the renamed alias of the compatible module function below.
func comparableFunctionGlobal() *BuiltInCallable {
	// Dart: _compatible.withDeprecationWarning('math').withName("comparable")
	return MustNewBuiltInCallableFunction("comparable", "$number1, $number2", "sass:math", compatibleImpl)
}

// compatibleFunction implements math.compatible($number1, $number2): whether
// the two numbers share convertible units. It keeps Dart's internal
// "compatible" name for the module spelling.
func compatibleFunction() sasscallable.Callable {
	// Dart: module uses internal name "compatible"
	return MustNewBuiltInCallableFunction("compatible", "$number1, $number2", "sass:math", compatibleImpl)
}

// compatibleImpl answers the compatibility query shared by both spellings: a
// fast compatible-units check first, then a trial comparison whose success
// also counts as compatible.
func compatibleImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	number1, err := value.AssertNumber(args[0], new("number1"))
	if err != nil {
		return nil, err
	}
	number2, err := value.AssertNumber(args[1], new("number2"))
	if err != nil {
		return nil, err
	}
	if number1.HasCompatibleUnits(number2) {
		return value.SassTrue, nil
	}
	_, err = number1.GreaterThan(number2)
	if err == nil {
		return value.SassTrue, nil
	}
	return value.SassFalse, nil
}

// unitlessFunctionGlobal implements the deprecated global unitless($number),
// the renamed alias of the is-unitless module function below.
func unitlessFunctionGlobal() *BuiltInCallable {
	// Dart: _isUnitless.withDeprecationWarning('math').withName("unitless")
	return MustNewBuiltInCallableFunction("unitless", "$number", "sass:math", unitlessImpl)
}

// isUnitlessModuleFunction implements math.is-unitless($number) under Dart's
// internal "is-unitless" module name.
func isUnitlessModuleFunction() sasscallable.Callable {
	// Dart: module uses internal name "is-unitless"
	return MustNewBuiltInCallableFunction("is-unitless", "$number", "sass:math", unitlessImpl)
}

// unitlessImpl answers the unitlessness query shared by both spellings.
func unitlessImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	number, err := value.AssertNumber(args[0], new("number"))
	if err != nil {
		return nil, err
	}
	if !number.HasUnits() {
		return value.SassTrue, nil
	}
	return value.SassFalse, nil
}

// randomFunction implements math.random($limit: null): a fractional value in
// [0, 1) for a null limit, else an integer in [1, $limit]. Unit-carrying
// limits keep Dart's unit-ignoring behavior behind a function-units
// deprecation suggesting math.div normalization; limits below 1 are an error.
func randomFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("random", "$limit: null", "sass:math", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if args[0] == value.Null {
			return value.NewUnitlessNumber(randomGen.Float64()), nil
		}
		limit, err := value.AssertNumber(args[0], new("limit"))
		if err != nil {
			return nil, err
		}
		if limit.HasUnits() {
			limStr, err := limit.String()
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				fmt.Sprintf(
					"math.random() will no longer ignore $limit units (%s) in a future release.\n\n"+
						"Recommendation: math.random(math.div($limit, 1%s)) * 1%s\n\n"+
						"To preserve current behavior: math.random(math.div($limit, 1%s))\n\n"+
						"More info: https://sass-lang.com/d/function-units",
					limStr, unitString(limit), unitString(limit), unitString(limit),
				),
				deprecation.FunctionUnits,
			); err != nil {
				return nil, err
			}
		}
		limitScalar, err := limit.AssertInt(new("limit"))
		if err != nil {
			return nil, err
		}
		if limitScalar < 1 {
			limStr, err := limit.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Must be greater than 0, was %s.", limStr), new("limit"))
		}
		return value.NewUnitlessNumber(float64(randomGen.Int63n(limitScalar) + 1)), nil
	})
}
