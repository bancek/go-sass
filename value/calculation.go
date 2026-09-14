// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/calculation.dart

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmath"
	"github.com/bancek/go-sass/util"
)

// A SassScript calculation.
//
// Although calculations can in principle have any name or any number of
// arguments, this class only exposes the specific calculations that are
// supported by the Sass spec. This ensures that all calculations that the user
// works with are always fully simplified.
type SassCalculation struct {

	// The calculation's name, such as "calc".
	Name string

	// The calculation's arguments.
	//
	// Each argument is either a SassNumber, a *SassCalculation, an unquoted
	// *SassString, or a *CalculationOperation.
	Arguments []any

	cachedHash *int
}

// IsSpecialNumber returns whether this is a special CSS number.
func (c *SassCalculation) IsSpecialNumber() bool { return true }

func (c *SassCalculation) RealNull() Value { return DefaultRealNull(c) }

// HashCode returns name.hashCode ^ listHash(arguments).
// Matches Dart: SassCalculation.hashCode => name.hashCode ^ listHash(arguments).
func (c *SassCalculation) HashCode() int {
	if c.cachedHash != nil {
		return *c.cachedHash
	}
	argsHash := 0
	for _, arg := range c.Arguments {
		argsHash = hashCombine(argsHash, calcArgHashCode(arg))
	}
	h := stringHashCode(c.Name) ^ argsHash
	c.cachedHash = &h
	return h
}

// A binary operation that can appear in a SassCalculation.
//
// The operands mirror Dart's CalculationOperation getter contract: Dart uses
// getters (rather than fields) so the JS API implementation can override the
// logic, but the shapes are the same.
type CalculationOperation struct {
	// Operator is the binary operator applied to Left and Right.
	Operator CalculationOperator
	// Left is the left-hand operand: a SassNumber, *SassCalculation,
	// unquoted *SassString, or *CalculationOperation.
	Left any
	// Right is the right-hand operand, with the same accepted shapes as Left.
	Right      any
	cachedHash *int
}

// HashCode returns operator.hashCode ^ left.hashCode ^ right.hashCode.
// Matches Dart: CalculationOperation.hashCode.
func (op *CalculationOperation) HashCode() int {
	if op.cachedHash != nil {
		return *op.cachedHash
	}
	h := int(op.Operator) ^ calcArgHashCode(op.Left) ^ calcArgHashCode(op.Right)
	op.cachedHash = &h
	return h
}

// calcArgHashCode returns the hash code for a single calculation argument.
func calcArgHashCode(a any) int {
	if h, ok := a.(interface{ HashCode() int }); ok {
		return h.HashCode()
	}
	return 0
}

// Matches Dart: CalculationOperation ==
func (op *CalculationOperation) Equal(other *CalculationOperation) bool {
	return op.Operator == other.Operator && calcArgEquals(op.Left, other.Left) && calcArgEquals(op.Right, other.Right)
}

// Matches Dart: CalculationOperation.toString
func (op *CalculationOperation) String() string {
	// Serialize the operation as the sole argument of a throwaway calc()
	// in inspect mode, then strip the wrapper's parentheses to leave just
	// the parenthesized operation text.
	parenthesized, err := SerializeValueInspect(&SassCalculation{Name: "", Arguments: []any{op}})
	if err != nil {
		return ""
	}
	return parenthesized[1 : len(parenthesized)-1]
}

// An enumeration of possible operators for CalculationOperation.
type CalculationOperator int

const (
	// The addition operator.
	CalculationOperatorPlus CalculationOperator = iota
	// The subtraction operator.
	CalculationOperatorMinus
	// The multiplication operator.
	CalculationOperatorTimes
	// The division operator.
	CalculationOperatorDividedBy
)

// The English name of this operator.
func (op CalculationOperator) Name() string {
	switch op {
	case CalculationOperatorPlus:
		return "plus"
	case CalculationOperatorMinus:
		return "minus"
	case CalculationOperatorTimes:
		return "times"
	case CalculationOperatorDividedBy:
		return "divided by"
	default:
		return "?"
	}
}

// The CSS syntax for this operator.
func (op CalculationOperator) Operator() string {
	switch op {
	case CalculationOperatorPlus:
		return "+"
	case CalculationOperatorMinus:
		return "-"
	case CalculationOperatorTimes:
		return "*"
	case CalculationOperatorDividedBy:
		return "/"
	default:
		return "?"
	}
}

// The precedence of this operator.
//
// An operator with higher precedence binds tighter.
func (op CalculationOperator) Precedence() int {
	switch op {
	case CalculationOperatorPlus, CalculationOperatorMinus:
		return 1
	case CalculationOperatorTimes, CalculationOperatorDividedBy:
		return 2
	default:
		return 0
	}
}

func (op CalculationOperator) String() string { return op.Name() }

// A deprecated representation of a string injected into a SassCalculation
// using interpolation.
//
// This only exists for backwards-compatibility with an older version of Dart
// Sass. It's now equivalent to creating a SassString whose value is wrapped
// in parentheses.
//
// Matches Dart: CalculationInterpolation
type CalculationInterpolation struct {
	// Value is the interpolated string. Like Dart's value getter, this is a
	// plain field here because there is no JS API override to accommodate.
	Value string
}

func (c *CalculationInterpolation) Equal(other *CalculationInterpolation) bool {
	return c.Value == other.Value
}

// Matches Dart: CalculationInterpolation.toString
func (c *CalculationInterpolation) String() string { return c.Value }

// A function type for single-argument math functions used by singleArgument.
type calcMathFunc func(SassNumber) (SassNumber, error)

// Creates a new calculation with the given name and arguments that will not be
// simplified.
//
// This is the internal bypass around every NewXxx constructor: no validation
// and no simplification run, so callers must only use it for argument lists
// that are already known-good (Dart marks the underlying factory @internal).
func NewUnsimplified(name string, args ...any) *SassCalculation {
	ac := make([]any, len(args))
	copy(ac, args)
	return &SassCalculation{Name: name, Arguments: ac}
}

// Creates a calc() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewCalc(argument any) (Value, error) {
	// A lone number or nested calculation collapses straight through
	// simplify; anything else is wrapped so the calc() shell survives to CSS.
	v := simplify(argument)
	if err, ok := v.(error); ok {
		return nil, err
	}
	if num, ok := v.(SassNumber); ok {
		return num, nil
	}
	if calc, ok := v.(*SassCalculation); ok {
		return calc, nil
	}
	return &SassCalculation{Name: "calc", Arguments: []any{v}}, nil
}

// Creates a min() calculation with the given arguments.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation. It must be passed at least one
// argument.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewMin(args ...any) (Value, error) {
	simplified, err := simplifyArguments(args)
	if err != nil {
		return nil, err
	}
	if len(simplified) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "min() must have at least one argument."}
	}

	var minimum SassNumber
	// Fold to a single number only while every argument is a number that is
	// mutually comparable with the running minimum. The first non-number
	// argument, or the first pair with incompatible units, abandons the fold
	// so the call survives below as an unsimplified min() node (after the
	// compatibility check reports any definitely-invalid combinations).
	for _, arg := range simplified {
		num, ok := arg.(SassNumber)
		if !ok {
			minimum = nil
			break
		}
		if minimum != nil && !minimum.isComparableTo(num) {
			minimum = nil
			break
		}
		if minimum == nil {
			minimum = num
		} else {
			gt, err := minimum.greaterThanNum(num)
			if err != nil {
				return nil, err
			}
			if gt.IsTruthy() {
				minimum = num
			}
		}
	}
	if minimum != nil {
		return minimum, nil
	}

	if err := verifyCompatibleNumbers(simplified); err != nil {
		return nil, err
	}
	return &SassCalculation{Name: "min", Arguments: simplified}, nil
}

// Creates a max() calculation with the given arguments.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation. It must be passed at least one
// argument.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewMax(args ...any) (Value, error) {
	simplified, err := simplifyArguments(args)
	if err != nil {
		return nil, err
	}
	if len(simplified) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "max() must have at least one argument."}
	}

	var maximum SassNumber
	// Same fold discipline as NewMin, tracking the greatest comparable
	// number instead: any non-number or incomparable pair keeps the call
	// symbolic rather than resolving it here.
	for _, arg := range simplified {
		num, ok := arg.(SassNumber)
		if !ok {
			maximum = nil
			break
		}
		if maximum != nil && !maximum.isComparableTo(num) {
			maximum = nil
			break
		}
		if maximum == nil {
			maximum = num
		} else {
			lt, err := maximum.lessThanNum(num)
			if err != nil {
				return nil, err
			}
			if lt.IsTruthy() {
				maximum = num
			}
		}
	}
	if maximum != nil {
		return maximum, nil
	}

	if err := verifyCompatibleNumbers(simplified); err != nil {
		return nil, err
	}
	return &SassCalculation{Name: "max", Arguments: simplified}, nil
}

// Creates a hypot() calculation with the given arguments.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation. It must be passed at least one
// argument.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewHypot(args ...any) (Value, error) {
	simplified, err := simplifyArguments(args)
	if err != nil {
		return nil, err
	}
	if len(simplified) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "hypot() must have at least one argument."}
	}
	if err := verifyCompatibleNumbers(simplified); err != nil {
		return nil, err
	}

	var subtotal float64
	first, ok := simplified[0].(SassNumber)
	// Hypot only resolves when the leading argument pins down a concrete,
	// non-percentage unit: percentages resolve against an unknown layout
	// context, so they (like any non-number or incompatible argument below)
	// leave the call symbolic.
	if !ok || first.HasUnit("%") {
		return &SassCalculation{Name: "hypot", Arguments: simplified}, nil
	}
	for i, arg := range simplified {
		num, ok := arg.(SassNumber)
		if !ok || !num.HasCompatibleUnits(first) {
			return &SassCalculation{Name: "hypot", Arguments: simplified}, nil
		}
		// Convert every operand into the first operand's units before
		// accumulating the sum of squares, so the final square root lands in
		// those same units.
		name := fmt.Sprintf("numbers[%d]", i+1)
		otherName := "numbers[1]"
		val, err := num.ConvertValueToMatch(first, &name, &otherName)
		if err != nil {
			return nil, err
		}
		subtotal += val * val
	}
	return SassNumberWithUnits(sassmath.Sqrt(subtotal), first.NumNumeratorUnits(), first.NumDenominatorUnits()), nil
}

// Creates a sqrt() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewSqrt(argument any) (Value, error) {
	return singleArgument("sqrt", argument, calcSqrt, true)
}

// Creates a sin() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewSin(argument any) (Value, error) {
	return singleArgument("sin", argument, calcSin, false)
}

// Creates a cos() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewCos(argument any) (Value, error) {
	return singleArgument("cos", argument, calcCos, false)
}

// Creates a tan() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewTan(argument any) (Value, error) {
	return singleArgument("tan", argument, calcTan, false)
}

// Creates an atan() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewAtan(argument any) (Value, error) {
	return singleArgument("atan", argument, calcAtan, true)
}

// Creates an asin() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewAsin(argument any) (Value, error) {
	return singleArgument("asin", argument, calcAsin, true)
}

// Creates an acos() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewAcos(argument any) (Value, error) {
	return singleArgument("acos", argument, calcAcos, true)
}

// Creates an abs() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewAbs(argument any) (Value, error) {
	return NewAbsInternal(argument, nil)
}

// Like NewAbs, but with an internal-only warn parameter.
//
// The warn callback is used to surface deprecation warnings.
func NewAbsInternal(argument any, warn WarnCallback) (Value, error) {
	arg := simplify(argument)
	num, ok := arg.(SassNumber)
	if !ok {
		return &SassCalculation{Name: "abs", Arguments: []any{arg}}, nil
	}
	// Percentage operands resolve against an unknown layout context, so a
	// future CSS abs() would be correct; resolving now is deprecated and
	// warns through the caller's callback (nil in pure calculation
	// contexts, where the warning is surfaced elsewhere).
	if num.HasUnit("%") {
		if warn != nil {
			s, _ := num.String()
			if err := warn(
				"Passing percentage units to the global abs() function is deprecated.\n"+
					"In the future, this will emit a CSS abs() function to be resolved by the browser.\n"+
					"To preserve current behavior: math.abs("+s+")"+
					"\n"+
					"To emit a CSS abs() now: abs(#{"+s+"})\n"+
					"More info: https://sass-lang.com/d/abs-percent",
				deprecation.AbsPercent,
			); err != nil {
				return nil, err
			}
		}
	}
	return NewUnitlessNumber(sassmath.Abs(num.NumValue())).CoerceToMatch(num, nil, nil)
}

// Creates an exp() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewExp(argument any) (Value, error) {
	arg := simplify(argument)
	num, ok := arg.(SassNumber)
	if !ok {
		return &SassCalculation{Name: "exp", Arguments: []any{arg}}, nil
	}
	if err := num.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	// exp(x) is e^x: Dart spells it as pow(SassNumber(math.E), argument).
	e := NewUnitlessNumber(math.E)
	return calcPow(e, num)
}

// Creates a sign() calculation with the given argument.
//
// The argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewSign(argument any) (Value, error) {
	arg := simplify(argument)
	switch n := arg.(type) {
	case SassNumber:
		// NaN and zero (including negative zero) pass through untouched so
		// their payload, sign bit, and units survive; only other
		// non-percentage numbers collapse to their unit-carrying sign.
		if math.IsNaN(n.NumValue()) || n.NumValue() == 0 {
			return n, nil
		}
		if !n.HasUnit("%") {
			return NewUnitlessNumber(signIncludingZero(n.NumValue())).CoerceToMatch(n, nil, nil)
		}
	}
	return &SassCalculation{Name: "sign", Arguments: []any{arg}}, nil
}

// Creates a clamp() calculation with the given min, value, and max.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than three arguments, but only if one of the
// arguments is an unquoted var() string.
func NewClamp(min, value, max any) (Value, error) {
	if value == nil && max != nil {
		return nil, &sasscommon.ArgumentError{Message: "If value is null, max must also be null."}
	}

	min = simplify(min)
	if value != nil {
		value = simplify(value)
	}
	if max != nil {
		max = simplify(max)
	}

	if minNum, ok := min.(SassNumber); ok && value != nil && max != nil {
		if valueNum, ok := value.(SassNumber); ok {
			if maxNum, ok := max.(SassNumber); ok {
				// Fast path: three mutually compatible numbers resolve
				// immediately by clamping value into [min, max].
				if minNum.HasCompatibleUnits(valueNum) && minNum.HasCompatibleUnits(maxNum) {
					lte, err := valueNum.lessThanOrEqualNum(minNum)
					if err != nil {
						return nil, err
					}
					if lte.IsTruthy() {
						return minNum, nil
					}
					gte, err := valueNum.greaterThanOrEqualNum(maxNum)
					if err != nil {
						return nil, err
					}
					if gte.IsTruthy() {
						return maxNum, nil
					}
					return valueNum, nil
				}
			}
		}
	}

	var args []any
	args = append(args, min)
	if value != nil {
		args = append(args, value)
	}
	if max != nil {
		args = append(args, max)
	}
	// Compatibility is verified before arity on purpose: a definitely-invalid
	// unit combination reports the incompatibility even when a var() standing
	// in for a missing argument would also waive the length check below.
	if err := verifyCompatibleNumbers(args); err != nil {
		return nil, err
	}
	if err := verifyLength(args, 3); err != nil {
		return nil, err
	}
	return &SassCalculation{Name: "clamp", Arguments: args}, nil
}

// Creates a pow() calculation with the given base and exponent.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than two arguments, but only if one of the
// arguments is an unquoted var() string.
func NewPow(base, exponent any) (Value, error) {
	args := []any{base}
	if exponent != nil {
		args = append(args, exponent)
	}
	// The arity check runs against the raw arguments so that a var()
	// standing in for a missing operand waives the two-argument requirement;
	// only the surviving (simplified) values below decide whether the call
	// resolves numerically.
	if err := verifyLength(args, 2); err != nil {
		return nil, err
	}
	base = simplify(base)
	var exp any
	if exponent != nil {
		exp = simplify(exponent)
	}
	baseNum, baseOK := base.(SassNumber)
	expNum, expOK := exp.(SassNumber)
	if !baseOK || !expOK {
		// The symbolic node keeps the pre-simplification arguments so
		// var()-style placeholders survive verbatim for the browser.
		return &SassCalculation{Name: "pow", Arguments: args}, nil
	}
	if err := baseNum.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	if err := expNum.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return calcPow(baseNum, expNum)
}

// Creates a log() calculation with the given number and base.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// If arguments contains exactly a single argument, the base is set to math.E.
func NewLog(number, base any) (Value, error) {
	number = simplify(number)
	var b any
	if base != nil {
		b = simplify(base)
	}
	numberNum, numberOK := number.(SassNumber)
	_, baseOK := b.(SassNumber)
	if !numberOK || (b != nil && !baseOK) {
		var args []any
		args = append(args, number)
		if b != nil {
			args = append(args, b)
		}
		return &SassCalculation{Name: "log", Arguments: args}, nil
	}
	if err := numberNum.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	if baseNum, ok := b.(SassNumber); ok {
		if err := baseNum.AssertNoUnits(nil); err != nil {
			return nil, err
		}
		return calcLog(numberNum, baseNum)
	}
	// A lone number means the natural logarithm: Dart defaults the base to
	// math.E, which calcLog expresses with a nil base.
	return calcLog(numberNum, nil)
}

// Creates an atan2() calculation for y and x.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than two arguments, but only if one of the
// arguments is an unquoted var() string.
func NewAtan2(y, x any) (Value, error) {
	y = simplify(y)
	var xv any
	if x != nil {
		xv = simplify(x)
	}
	args := []any{y}
	if xv != nil {
		args = append(args, xv)
	}
	if err := verifyLength(args, 2); err != nil {
		return nil, err
	}
	if err := verifyCompatibleNumbers(args); err != nil {
		return nil, err
	}
	yNum, yOK := y.(SassNumber)
	xNum, xOK := xv.(SassNumber)
	if !yOK || !xOK {
		return &SassCalculation{Name: "atan2", Arguments: args}, nil
	}
	// Percentages have no absolute meaning until layout, and mismatched
	// units cannot be reconciled here, so both cases stay symbolic even
	// though the pair already passed the known-incompatibility screen above.
	if yNum.HasUnit("%") || xNum.HasUnit("%") || !yNum.HasCompatibleUnits(xNum) {
		return &SassCalculation{Name: "atan2", Arguments: args}, nil
	}
	return calcAtan2(yNum, xNum)
}

// Creates a rem() calculation with the given dividend and modulus.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than two arguments, but only if one of the
// arguments is an unquoted var() string.
func NewRem(dividend, modulus any) (Value, error) {
	dividend = simplify(dividend)
	var mod any
	if modulus != nil {
		mod = simplify(modulus)
	}
	args := []any{dividend}
	if mod != nil {
		args = append(args, mod)
	}
	if err := verifyLength(args, 2); err != nil {
		return nil, err
	}
	if err := verifyCompatibleNumbers(args); err != nil {
		return nil, err
	}
	divNum, divOK := dividend.(SassNumber)
	modNum, modOK := mod.(SassNumber)
	if !divOK || !modOK || !divNum.HasCompatibleUnits(modNum) {
		return &SassCalculation{Name: "rem", Arguments: args}, nil
	}
	result, err := divNum.moduloNum(modNum)
	if err != nil {
		return nil, err
	}
	// Sass's % is floored division, which takes the divisor's sign, but CSS
	// rem() takes the dividend's sign. When the signs disagree, shift the
	// result by one modulus to correct it; an infinite modulus leaves the
	// dividend untouched and an exact zero just flips its sign bit.
	modSign := signIncludingZero(modNum.NumValue())
	divSign := signIncludingZero(divNum.NumValue())
	if modSign != divSign {
		if math.IsInf(modNum.NumValue(), 0) {
			return divNum, nil
		}
		if result.NumValue() == 0 {
			return result.unaryMinusNum()
		}
		return result.minusNum(modNum)
	}
	return result, nil
}

// Creates a mod() calculation with the given dividend and modulus.
//
// Each argument must be either a SassNumber, a *SassCalculation, an unquoted
// *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than two arguments, but only if one of the
// arguments is an unquoted var() string.
func NewMod(dividend, modulus any) (Value, error) {
	dividend = simplify(dividend)
	var mod any
	if modulus != nil {
		mod = simplify(modulus)
	}
	args := []any{dividend}
	if mod != nil {
		args = append(args, mod)
	}
	if err := verifyLength(args, 2); err != nil {
		return nil, err
	}
	if err := verifyCompatibleNumbers(args); err != nil {
		return nil, err
	}
	divNum, divOK := dividend.(SassNumber)
	modNum, modOK := mod.(SassNumber)
	if !divOK || !modOK || !divNum.HasCompatibleUnits(modNum) {
		return &SassCalculation{Name: "mod", Arguments: args}, nil
	}
	return divNum.moduloNum(modNum)
}

// Creates a round() calculation with the given strategyOrNumber, numberOrStep,
// and step. Strategy must be either nearest, up, down or to-zero.
//
// Number and step must be either a SassNumber, a *SassCalculation, an
// unquoted *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation, so it may return a
// SassNumber rather than a *SassCalculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
//
// This may be passed fewer than two arguments, but only if one of the
// arguments is an unquoted var() string.
func NewRound(strategyOrNumber, numberOrStep, step any) (Value, error) {
	return NewRoundInternal(strategyOrNumber, numberOrStep, step, nil, nil)
}

// Like NewRound, but with the internal-only inLegacySassFunction and warn
// parameters.
//
// If inLegacySassFunction isn't nil, this allows unitless numbers to be
// added and subtracted with numbers with units, for backwards-compatibility
// with the old global min() and max() functions. This emits a deprecation
// warning using the string as the function's name.
//
// The warn callback is used to surface deprecation warnings.
func NewRoundInternal(strategyOrNumber, numberOrStep, step any,
	inLegacySassFunction *string,
	warn WarnCallback,
) (Value, error) {
	strategy := simplify(strategyOrNumber)
	var number any
	if numberOrStep != nil {
		number = simplify(numberOrStep)
	}
	var stepVal any
	if step != nil {
		stepVal = simplify(step)
	}

	// Case table, in order: a lone unitless number rounds directly; a lone
	// unit-bearing number only resolves inside the legacy global round() with
	// a deprecation warning; two numbers resolve via roundWithStep once their
	// units check out (otherwise they survive as a node so the browser can
	// reject or resolve them); a strategy plus two compatible numbers does
	// the same; a strategy plus an opaque string stays symbolic; a strategy
	// without its step, or a strategy with neither operand, is a script
	// error; anything else either survives as a partially-var() node or
	// fails on the unknown strategy name.
	switch {
	case isUnitlessSassNumber(strategy) && number == nil && stepVal == nil:
		return NewUnitlessNumber(math.Round(strategy.(SassNumber).NumValue())), nil

	case isSassNumber(strategy) && number == nil && stepVal == nil && inLegacySassFunction != nil:
		if warn != nil {
			if err := warn("In future versions of Sass, round() will be interpreted as a CSS round() calculation. This requires an explicit modulus when rounding numbers with units. If you want to use the Sass function, call math.round() instead.\n\nSee https://sass-lang.com/d/import", deprecation.GlobalBuiltin); err != nil {
				return nil, err
			}
		}
		return matchUnits(math.Round(strategy.(SassNumber).NumValue()), strategy.(SassNumber)), nil

	case isSassNumber(strategy) && isSassNumber(number) && stepVal == nil:
		stratNum := strategy.(SassNumber)
		num := number.(SassNumber)
		if !stratNum.HasCompatibleUnits(num) {
			if err := verifyCompatibleNumbers([]any{stratNum, num}); err != nil {
				return nil, err
			}
			return &SassCalculation{Name: "round", Arguments: []any{stratNum, num}}, nil
		}
		if err := verifyCompatibleNumbers([]any{stratNum, num}); err != nil {
			return nil, err
		}
		return roundWithStep("nearest", stratNum, num)

	case isRoundingStrategy(strategy) && isSassNumber(number) && isSassNumber(stepVal):
		strat := strategy.(*SassString)
		num := number.(SassNumber)
		stp := stepVal.(SassNumber)
		if !num.HasCompatibleUnits(stp) {
			if err := verifyCompatibleNumbers([]any{num, stp}); err != nil {
				return nil, err
			}
			return &SassCalculation{Name: "round", Arguments: []any{strat, num, stp}}, nil
		}
		if err := verifyCompatibleNumbers([]any{num, stp}); err != nil {
			return nil, err
		}
		return roundWithStep(strat.Text, num, stp)

	case isRoundingStrategy(strategy) && isSassString(number) && stepVal == nil:
		return &SassCalculation{Name: "round", Arguments: []any{strategy, number}}, nil

	case isRoundingStrategy(strategy) && number != nil && stepVal == nil:
		return nil, sasscommon.NewSassScriptException("If strategy is not null, step is required.", nil)

	case isRoundingStrategy(strategy) && number == nil && stepVal == nil:
		return nil, sasscommon.NewSassScriptException("Number to round and step arguments are required.", nil)

	case !isRoundingStrategy(strategy) && number == nil && stepVal == nil:
		return &SassCalculation{Name: "round", Arguments: []any{strategy}}, nil

	case !isRoundingStrategy(strategy) && number != nil && stepVal == nil:
		return &SassCalculation{Name: "round", Arguments: []any{strategy, number}}, nil

	case (isRoundingStrategy(strategy) || isSpecialVariableSassString(strategy)) && number != nil && stepVal != nil:
		return &SassCalculation{Name: "round", Arguments: []any{strategy, number, stepVal}}, nil

	case number != nil && stepVal != nil:
		s, err := SprintAny(strategyOrNumber)
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("%s must be either nearest, up, down or to-zero.", s), nil)

	case stepVal != nil && numberOrStep == nil:
		fallthrough
	default:
		return nil, sasscommon.NewSassScriptException("Invalid parameters.", nil)
	}
}

// Creates an calc-size() calculation with the given basis and value.
//
// The basis and value must be either a SassNumber, a *SassCalculation, an
// unquoted *SassString, or a *CalculationOperation.
//
// This automatically simplifies the calculation. It returns an error if it can
// determine that the calculation will definitely produce invalid CSS.
func NewCalcSize(basis, value any) (*SassCalculation, error) {
	args := []any{basis}
	if value != nil {
		args = append(args, value)
	}
	if err := verifyLength(args, 2); err != nil {
		return nil, err
	}
	basis = simplify(basis)
	var v any
	if value != nil {
		v = simplify(value)
	}
	// calc-size() never resolves numerically: unlike the math functions
	// above, even fully-known operands stay wrapped so the browser applies
	// the size-resolution semantics at layout time.
	result := &SassCalculation{Name: "calc-size", Arguments: []any{basis}}
	if v != nil {
		result.Arguments = append(result.Arguments, v)
	}
	return result, nil
}

// Creates and simplifies a CalculationOperation with the given operator,
// left, and right.
//
// This automatically simplifies the operation, so it may return a
// SassNumber rather than a *CalculationOperation.
//
// Each of left and right must be either a SassNumber, a *SassCalculation, an
// unquoted *SassString, or a *CalculationOperation.
func Operate(operator CalculationOperator, left, right any) (any, error) {
	return OperateInternal(operator, left, right, nil, true, nil)
}

// Like Operate, but with the internal-only inLegacySassFunction and warn
// parameters.
//
// If inLegacySassFunction isn't nil, this allows unitless numbers to be
// added and subtracted with numbers with units, for backwards-compatibility
// with the old global min() and max() functions. This emits a deprecation
// warning using the string as the function's name.
//
// If simplify is false, no simplification will be done.
//
// The warn callback is used to surface deprecation warnings.
func OperateInternal(operator CalculationOperator, left, right any,
	inLegacySassFunction *string,
	doSimplify bool,
	warn WarnCallback,
) (any, error) {
	if !doSimplify {
		return &CalculationOperation{Operator: operator, Left: left, Right: right}, nil
	}
	left = simplify(left)
	right = simplify(right)

	// Addition and subtraction fold only when both sides are numbers with
	// mutually convertible units. The legacy global min()/max() path relaxes
	// this for merely comparable pairs (unitless mixed with unit-bearing)
	// with a deprecation warning, because the old Sass functions allowed it.
	if operator == CalculationOperatorPlus || operator == CalculationOperatorMinus {
		if leftNum, ok := left.(SassNumber); ok {
			if rightNum, ok := right.(SassNumber); ok {
				compatible := leftNum.HasCompatibleUnits(rightNum)
				if !compatible && inLegacySassFunction != nil && leftNum.isComparableTo(rightNum) {
					if warn != nil {
						if err := warn(fmt.Sprintf("In future versions of Sass, %s() will be interpreted as the CSS %s() calculation. This doesn't allow unitless numbers to be mixed with numbers with units. If you want to use the Sass function, call math.%s() instead.\n\nSee https://sass-lang.com/d/import", *inLegacySassFunction, *inLegacySassFunction, *inLegacySassFunction), deprecation.GlobalBuiltin); err != nil {
							return nil, err
						}
					}
					compatible = true
				}
				if compatible {
					if operator == CalculationOperatorPlus {
						return leftNum.plusNum(rightNum)
					}
					return leftNum.minusNum(rightNum)
				}
			}
		}

		if err := verifyCompatibleNumbers([]any{left, right}); err != nil {
			return nil, err
		}

		// Normalize a negative right-hand side into the operator: a + (-b)
		// becomes a - b (and symmetrically), keeping the tree canonical.
		if rightNum, ok := right.(SassNumber); ok && util.FuzzyLessThan(rightNum.NumValue(), 0) {
			neg, err := rightNum.timesNum(NewUnitlessNumber(-1))
			if err != nil {
				return nil, err
			}
			right = neg
			if operator == CalculationOperatorPlus {
				operator = CalculationOperatorMinus
			} else {
				operator = CalculationOperatorPlus
			}
		}

		return &CalculationOperation{Operator: operator, Left: left, Right: right}, nil
	} else if leftNum, ok := left.(SassNumber); ok {
		// Multiplication and division always fold when both sides are
		// numbers: unlike addition, they never need compatible units because
		// the units compose instead of merging.
		if rightNum, ok := right.(SassNumber); ok {
			if operator == CalculationOperatorTimes {
				return leftNum.timesNum(rightNum)
			}
			return leftNum.dividedByNum(rightNum)
		}
	}

	return &CalculationOperation{Operator: operator, Left: left, Right: right}, nil
}

func (c *SassCalculation) Equals(other Value) bool {
	// Structural equality: same function name, same arity, and pairwise
	// argument equality across the number/string/calculation/operation
	// shapes (numbers compare by value with unit conversion, strings by
	// text — quotes can never reach here because simplify rejects them).
	if oc, ok := other.(*SassCalculation); ok {
		if c.Name != oc.Name || len(c.Arguments) != len(oc.Arguments) {
			return false
		}
		for i := range c.Arguments {
			if !calcArgEquals(c.Arguments[i], oc.Arguments[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func calcArgEquals(a, b any) bool {
	switch va := a.(type) {
	case SassNumber:
		if vb, ok := b.(SassNumber); ok {
			return va.Equals(vb)
		}
		return false
	case *SassString:
		if vb, ok := b.(*SassString); ok {
			return va.Text == vb.Text
		}
		return false
	case *SassCalculation:
		if vb, ok := b.(*SassCalculation); ok {
			return va.Equals(vb)
		}
		return false
	case *CalculationOperation:
		if vb, ok := b.(*CalculationOperation); ok {
			return va.Operator == vb.Operator && calcArgEquals(va.Left, vb.Left) && calcArgEquals(va.Right, vb.Right)
		}
		return false
	}
	return false
}

// --- Value interface methods ---

func (c *SassCalculation) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitCalculation(c)
}
func (c *SassCalculation) isValue()                               {}
func (c *SassCalculation) ToCssString(quote bool) (string, error) { return SerializeValue(c, quote) }
func (c *SassCalculation) String() (string, error)                { return SerializeValueInspect(c) }
func (c *SassCalculation) IsTruthy() bool                         { return true }
func (c *SassCalculation) Separator() ListSeparator               { return ListSeparatorUndecided }
func (c *SassCalculation) HasBrackets() bool                      { return false }
func (c *SassCalculation) AsList() ([]Value, error)               { return []Value{c}, nil }
func (c *SassCalculation) LengthAsList() int                      { return 1 }
func (c *SassCalculation) IsBlank() bool                          { return false }
func (c *SassCalculation) IsSpecialVariable() bool                { return false }
func (c *SassCalculation) TryMap() *SassMap                       { return nil }

// AssertCalculation returns this calculation.
func (c *SassCalculation) AssertCalculation(name *string) *SassCalculation { return c }

// Plus implements the + operator for calculations.
//
// Dart marks these operator overrides @nodoc @internal: string operands
// concatenate through the base Value implementation, while every other
// operand type is an "Undefined operation" script error.
func (c *SassCalculation) Plus(other Value) (Value, error) {
	if os, ok := other.(*SassString); ok {
		s, err := c.ToCssString(true)
		if err != nil {
			return nil, err
		}
		return &SassString{Text: s + os.Text, HasQuotes: os.HasQuotes}, nil
	}
	s, err := c.ToCssString(true)
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s + %s\".", s, otherStr), nil)
}

// Minus implements the - operator for calculations.
func (c *SassCalculation) Minus(other Value) (Value, error) {
	cStr, err := c.ToCssString(true)
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s - %s\".", cStr, otherStr), nil)
}

// UnaryPlus implements the + prefix operator for calculations.
func (c *SassCalculation) UnaryPlus() (Value, error) {
	cStr, err := c.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"+%s\".", cStr), nil)
}

// UnaryMinus implements the - prefix operator for calculations.
func (c *SassCalculation) UnaryMinus() (Value, error) {
	cStr, err := c.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"-%s\".", cStr), nil)
}

// --- Value interface methods (default implementations) ---

func (c *SassCalculation) SingleEquals(other Value) (Value, error) {
	return DefaultSingleEquals(c, other)
}
func (c *SassCalculation) DividedBy(other Value) (Value, error) { return DefaultDividedBy(c, other) }
func (c *SassCalculation) Times(other Value) (Value, error)     { return DefaultTimes(c, other) }
func (c *SassCalculation) Modulo(other Value) (Value, error)    { return DefaultModulo(c, other) }
func (c *SassCalculation) GreaterThan(other Value) (Value, error) {
	return DefaultGreaterThan(c, other)
}
func (c *SassCalculation) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(c, other)
}
func (c *SassCalculation) LessThan(other Value) (Value, error) { return DefaultLessThan(c, other) }
func (c *SassCalculation) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(c, other)
}
func (c *SassCalculation) UnaryDivide() (Value, error) { return DefaultUnaryDivide(c) }
func (c *SassCalculation) UnaryNot() (Value, error)    { return DefaultUnaryNot(c) }

// --- Private helpers ---

// Returns value coerced to number's units.
//
// No conversion runs here: every caller has already established that the
// units match, so this just re-homes a freshly computed float into the
// operand's unit shape (Dart's _matchUnits).
func matchUnits(value float64, number SassNumber) SassNumber {
	return SassNumberWithUnits(value, number.NumNumeratorUnits(), number.NumDenominatorUnits())
}

// Returns a rounded number based on a selected rounding strategy,
// to the nearest integer multiple of step.
//
// NaN operands and a zero step always yield NaN; an infinite number stays
// itself; an infinite step collapses per strategy (nearest/to-zero snap to
// signed zero, up/down snap to the matching infinity or signed zero). A
// negative step inverts the sense of up/down because rounding follows the
// step's direction. The result is re-homed into number's units.
func roundWithStep(strategy string, number, step SassNumber) (SassNumber, error) {
	if strategy != "nearest" && strategy != "up" && strategy != "down" && strategy != "to-zero" {
		return nil, &sasscommon.ArgumentError{Message: fmt.Sprintf("%s must be either nearest, up, down or to-zero.", strategy)}
	}

	if (math.IsInf(number.NumValue(), 0) && math.IsInf(step.NumValue(), 0)) ||
		step.NumValue() == 0 ||
		math.IsNaN(number.NumValue()) ||
		math.IsNaN(step.NumValue()) {
		return SassNumberWithUnits(math.NaN(), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
	}
	if math.IsInf(number.NumValue(), 0) {
		return number, nil
	}

	if math.IsInf(step.NumValue(), 0) {
		switch {
		case number.NumValue() == 0:
			return number, nil
		case strategy == "nearest" || strategy == "to-zero":
			if number.NumValue() > 0 {
				return SassNumberWithUnits(0.0, number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
			}
			return SassNumberWithUnits(math.Copysign(0.0, -1), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
		case strategy == "up":
			if number.NumValue() > 0 {
				return SassNumberWithUnits(math.Inf(1), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
			}
			return SassNumberWithUnits(math.Copysign(0.0, -1), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
		case strategy == "down":
			if number.NumValue() < 0 {
				return SassNumberWithUnits(math.Inf(-1), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
			}
			return SassNumberWithUnits(0.0, number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
		}
	}

	stepVal, err := step.ConvertValueToMatch(number, nil, nil)
	if err != nil {
		return nil, err
	}

	var result float64
	switch strategy {
	case "nearest":
		result = math.Round(number.NumValue()/stepVal) * stepVal
	case "up":
		if step.NumValue() < 0 {
			result = math.Floor(number.NumValue()/stepVal) * stepVal
		} else {
			result = math.Ceil(number.NumValue()/stepVal) * stepVal
		}
	case "down":
		if step.NumValue() < 0 {
			result = math.Ceil(number.NumValue()/stepVal) * stepVal
		} else {
			result = math.Floor(number.NumValue()/stepVal) * stepVal
		}
	case "to-zero":
		if number.NumValue() < 0 {
			result = math.Ceil(number.NumValue()/stepVal) * stepVal
		} else {
			result = math.Floor(number.NumValue()/stepVal) * stepVal
		}
	default:
		result = math.NaN()
	}
	return SassNumberWithUnits(result, number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
}

// Returns an unmodifiable list of args, with each argument simplified.
//
// A simplification failure is threaded back as an error return (rather than
// panicking) so constructors like NewMin/NewMax can surface quoted-string
// and invalid-value rejections with their call-site context.
func simplifyArguments(args []any) ([]any, error) {
	result := make([]any, len(args))
	for i, arg := range args {
		s := simplify(arg)
		if err, ok := s.(error); ok {
			return nil, err
		}
		result[i] = s
	}
	return result, nil
}

// Simplifies a calculation argument.
//
// Numbers and operations are already canonical and pass through. Legacy
// interpolations are wrapped in parentheses as unquoted strings. Quoted
// strings and non-calculation values (colors, lists, maps, ...) are rejected
// because they can never appear in valid CSS math. A single-argument calc()
// unwraps to its contents — parenthesizing payloads that would otherwise
// reparse differently (whitespace, / or * operators, var() prefixes) — while
// every other calculation survives as-is.
//
// Errors here are unspanned script errors: the evaluator wraps them with the
// operation's span when the calculation surfaces through a BinaryOperation
// or a multi-argument call.
//
// nolint: cyclop
func simplify(arg any) any {
	switch a := arg.(type) {
	case SassNumber, *CalculationOperation:
		return a
	case *CalculationInterpolation:
		return &SassString{Text: "(" + a.Value + ")", HasQuotes: false}
	case *SassString:
		if !a.HasQuotes {
			return a
		}
		// Quoted strings can't be used in calculations
		s, err := a.String()
		if err != nil {
			return err
		}
		return sasscommon.NewSassScriptException(fmt.Sprintf("Quoted string %s can't be used in a calculation.", s), nil)
	case *SassCalculation:
		if a.Name == "calc" && len(a.Arguments) == 1 {
			if str, ok := a.Arguments[0].(*SassString); ok && !str.HasQuotes && needsParentheses(str.Text) {
				return &SassString{Text: "(" + str.Text + ")", HasQuotes: false}
			}
			return a.Arguments[0]
		}
		return a
	default:
		if aValue, ok := a.(Value); ok {
			s, err := aValue.String()
			if err != nil {
				return err
			}
			return sasscommon.NewSassScriptException(fmt.Sprintf("Value %s can't be used in a calculation.", s), nil)
		}
		s, err := SprintAny(a)
		if err != nil {
			return err
		}
		return &sasscommon.ArgumentError{Message: fmt.Sprintf("Unexpected calculation argument %s.", s)}
	}
}

// Returns whether text needs parentheses if it's the contents of a calc()
// being embedded in another calculation.
//
// This scans every rune for intrinsic paren-triggers (whitespace, /, *) and
// additionally treats a var( prefix as opaque-function syntax that must be
// protected. Dart checks the first four code units explicitly before looping,
// but the accepted language is identical: any trigger rune anywhere, or the
// case-insensitive var( lead.
func needsParentheses(text string) bool {
	runes := []rune(text)
	if slices.ContainsFunc(runes, charNeedsParentheses) {
		return true
	}

	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "var(") && len(runes) >= 4 {
		return true
	}
	return false
}

// Returns whether character intrinsically needs parentheses if it appears
// in the unquoted string argument of a calc() being embedded in another
// calculation.
func charNeedsParentheses(ch rune) bool {
	return unicode.IsSpace(ch) || ch == '/' || ch == '*'
}

// Verifies that all the numbers in args aren't known to be incompatible
// with one another, and that they don't have units that are too complex for
// calculations.
//
// Note: this logic is largely duplicated in the evaluator's own
// compatibility check, and most changes here should also be reflected there
// (mirroring Dart's note that _verifyCompatibleNumbers is duplicated in
// _EvaluateVisitor and the two must be kept in sync).
//
// Like simplify, this raises unspanned script errors: incompatible pairs are
// reported as "<a> and <b> are incompatible.", and complex-unit numbers as
// "<n> isn't compatible with CSS calculations." Span attachment is the
// evaluating caller's job.
func verifyCompatibleNumbers(args []any) error {
	for _, arg := range args {
		if num, ok := arg.(SassNumber); ok && num.HasComplexUnits() {
			numStr, err := num.String()
			if err != nil {
				return err
			}
			return sasscommon.NewSassScriptException(fmt.Sprintf("Number %s isn't compatible with CSS calculations.", numStr), nil)
		}
	}

	for i := 0; i < len(args)-1; i++ {
		num1, ok := args[i].(SassNumber)
		if !ok {
			continue
		}
		for j := i + 1; j < len(args); j++ {
			num2, ok := args[j].(SassNumber)
			if !ok {
				continue
			}
			if num1.HasPossiblyCompatibleUnits(num2) {
				continue
			}
			s1, err := num1.String()
			if err != nil {
				return err
			}
			s2, err := num2.String()
			if err != nil {
				return err
			}
			return sasscommon.NewSassScriptException(fmt.Sprintf("%s and %s are incompatible.", s1, s2), nil)
		}
	}
	return nil
}

// Throws an error if args isn't expectedLength and doesn't contain a
// *SassString (which indicates a var() that could "skip" arguments).
//
// Any unquoted string waives the arity check, not just an actual var():
// because var() (or attr()/if()) substitution happens in the browser, a
// string operand may expand to the missing arguments at runtime, so the
// calculation must survive as a symbolic node rather than fail here.
func verifyLength(args []any, expectedLength int) error {
	if len(args) == expectedLength {
		return nil
	}
	for _, arg := range args {
		if _, ok := arg.(*SassString); ok {
			return nil
		}
	}
	return sasscommon.NewSassScriptException(fmt.Sprintf("%d arguments required, but only %d %s passed.", expectedLength, len(args), util.Pluralize("was", len(args), new("were"))), nil)
}

// Returns a Value by calling a single-argument math function.
//
// Non-number arguments survive as an unsimplified name() node. When
// forbidUnits holds (sqrt, atan, asin, acos), unit-bearing numbers are
// rejected before the math callback runs; sin/cos/tan instead coerce their
// operand to radians inside their own implementations.
func singleArgument(name string, argument any, fn calcMathFunc, forbidUnits bool) (Value, error) {
	arg := simplify(argument)
	num, ok := arg.(SassNumber)
	if !ok {
		return &SassCalculation{Name: name, Arguments: []any{arg}}, nil
	}
	if forbidUnits {
		if err := num.AssertNoUnits(nil); err != nil {
			return nil, err
		}
	}
	return fn(num)
}

// Returns whether v is a SassNumber.
func isSassNumber(v any) bool {
	_, ok := v.(SassNumber)
	return ok
}

// Returns whether v is a unitless SassNumber.
func isUnitlessSassNumber(v any) bool {
	n, ok := v.(SassNumber)
	return ok && !n.HasUnits()
}

// Returns whether v is a *SassString.
func isSassString(v any) bool {
	_, ok := v.(*SassString)
	return ok
}

// Returns whether v is a *SassString whose text is a rounding strategy.
func isRoundingStrategy(v any) bool {
	s, ok := v.(*SassString)
	if !ok || s.HasQuotes {
		return false
	}
	return s.Text == "nearest" || s.Text == "up" || s.Text == "down" || s.Text == "to-zero"
}

// Returns whether v is a *SassString that is a special variable
// (var(), attr(), or if()). Matches Dart's SassString.isSpecialVariable.
//
// The match is case-insensitive on the function-name prefix and excludes
// quoted strings, which are opaque data rather than browser-resolvable
// references. Round() treats these as arity wildcards that keep the call
// symbolic.
func isSpecialVariableSassString(v any) bool {
	s, ok := v.(*SassString)
	if !ok || s.HasQuotes {
		return false
	}
	lower := strings.ToLower(s.Text)
	return strings.HasPrefix(lower, "var(") ||
		strings.HasPrefix(lower, "attr(") ||
		strings.HasPrefix(lower, "if(")
}

// Math function implementations for singleArgument.
//
// Unit contracts mirror Dart's util/number.dart exactly: sqrt/asin/acos/atan
// assert unitless operands (re-checked here even though singleArgument
// already enforced forbidUnits, because these are also reachable directly),
// sin/cos/tan coerce to radians and return unitless results, the inverse
// functions return degrees, and pow/log assert unitless operands with log
// defaulting a nil base to the natural logarithm.

func calcSqrt(n SassNumber) (SassNumber, error) {
	if err := n.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Sqrt(n.NumValue())), nil
}

func calcSin(n SassNumber) (SassNumber, error) {
	val, err := n.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Sin(val)), nil
}

func calcCos(n SassNumber) (SassNumber, error) {
	val, err := n.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Cos(val)), nil
}

func calcTan(n SassNumber) (SassNumber, error) {
	val, err := n.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Tan(val)), nil
}

func calcAtan(n SassNumber) (SassNumber, error) {
	if err := n.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Atan(n.NumValue())), nil
}

func calcAsin(n SassNumber) (SassNumber, error) {
	if err := n.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Asin(n.NumValue())), nil
}

func calcAcos(n SassNumber) (SassNumber, error) {
	if err := n.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Acos(n.NumValue())), nil
}

func calcPow(base, exp SassNumber) (SassNumber, error) {
	if err := base.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	if err := exp.AssertNoUnits(nil); err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Pow(base.NumValue(), exp.NumValue())), nil
}

func calcLog(number, base SassNumber) (SassNumber, error) {
	if base != nil {
		return NewUnitlessNumber(sassmath.Log(number.NumValue()) / sassmath.Log(base.NumValue())), nil
	}
	return NewUnitlessNumber(sassmath.Log(number.NumValue())), nil
}

func calcAtan2(y, x SassNumber) (SassNumber, error) {
	// Reconcile x into y's units first so the arctangent sees a single unit
	// system, then report the angle in degrees like the other inverse trig
	// functions.
	xVal, err := x.ConvertValueToMatch(y, new("x"), new("y"))
	if err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Atan2(y.NumValue(), xVal)), nil
}

// Returns radians as a SassNumber with unit deg.
//
// Every inverse trig entry point funnels through here so callers never have
// to remember the radians-to-degrees factor themselves.
func radiansToDegrees(radians float64) SassNumber {
	return NewSingleUnitNumber(radians*(180/math.Pi), "deg")
}

// Returns the sign of f, including positive and negative zero.
//
// Plain comparisons cannot tell +0 from -0, so the zero arm inspects the
// sign bit explicitly: negative zero reports -1, positive zero reports +1,
// and every other value reports its mathematical sign. This is what lets
// rem() and sign() honor signed zeros per the CSS math spec.
func signIncludingZero(f float64) float64 {
	if f == 0 {
		if math.Signbit(f) {
			return -1.0
		}
		return 1.0
	}
	return math.Copysign(1, f)
}
