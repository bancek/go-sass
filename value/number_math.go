// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/util/number.dart (sqrt/sin/cos/tan/atan/asin/acos/abs/log/pow/atan2 section)

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmath"
)

// SqrtNumber returns the square root of number.
//
// Ports Dart's sqrt: the operand must be unitless (units under a root are
// meaningless in CSS), and the result is likewise unitless.
func SqrtNumber(number SassNumber) (SassNumber, error) {
	if err := number.AssertNoUnits(new("number")); err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Sqrt(number.NumValue())), nil
}

// SinNumber returns the sine of number.
//
// Ports Dart's sin: the operand is coerced to radians first, so callers may
// pass any angle unit (deg, grad, turn, ...), and the result is unitless.
func SinNumber(number SassNumber) (SassNumber, error) {
	val, err := number.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Sin(val)), nil
}

// CosNumber returns the cosine of number.
//
// Ports Dart's cos, with the same coerce-to-radians-then-unitless contract
// as SinNumber.
func CosNumber(number SassNumber) (SassNumber, error) {
	val, err := number.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Cos(val)), nil
}

// TanNumber returns the tangent of number.
//
// Ports Dart's tan, with the same coerce-to-radians-then-unitless contract
// as SinNumber.
func TanNumber(number SassNumber) (SassNumber, error) {
	val, err := number.CoerceValueToUnit("rad", new("number"))
	if err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Tan(val)), nil
}

// AtanNumber returns the arctangent of number in degrees.
//
// Ports Dart's atan: the operand must be unitless (it is a ratio, not an
// angle), and the result carries the deg unit via radiansToDegrees.
func AtanNumber(number SassNumber) (SassNumber, error) {
	if err := number.AssertNoUnits(new("number")); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Atan(number.NumValue())), nil
}

// AsinNumber returns the arcsine of number in degrees.
//
// Ports Dart's asin, with the same unitless-in/deg-out contract as
// AtanNumber.
func AsinNumber(number SassNumber) (SassNumber, error) {
	if err := number.AssertNoUnits(new("number")); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Asin(number.NumValue())), nil
}

// AcosNumber returns the arccosine of number in degrees.
//
// Ports Dart's acos, with the same unitless-in/deg-out contract as
// AtanNumber.
func AcosNumber(number SassNumber) (SassNumber, error) {
	if err := number.AssertNoUnits(new("number")); err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Acos(number.NumValue())), nil
}

// AbsNumber returns the absolute value of number, preserving units.
//
// This rebuilds the result directly in the operand's units, so unlike Abs
// in number_util.go (which routes through the unitless-abs-then-coerce
// spelling of Dart's abs) it cannot fail on complex units.
func AbsNumber(number SassNumber) (SassNumber, error) {
	return SassNumberWithUnits(sassmath.Abs(number.NumValue()), number.NumNumeratorUnits(), number.NumDenominatorUnits()), nil
}

// LogNumber returns the logarithm of number with respect to base. If base
// is nil, the natural logarithm is returned.
//
// Ports Dart's log: both operands must be unitless (checked inline here so
// the error can name the offending argument), and a nil base selects the
// natural logarithm rather than erroring.
func LogNumber(number, base SassNumber) (SassNumber, error) {
	if number.HasUnits() {
		s, err := number.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have no units.", s), new("number"))
	}
	if base != nil {
		if base.HasUnits() {
			s, err := base.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have no units.", s), new("base"))
		}
		return NewUnitlessNumber(sassmath.Log(number.NumValue()) / sassmath.Log(base.NumValue())), nil
	}
	return NewUnitlessNumber(sassmath.Log(number.NumValue())), nil
}

// PowNumber returns base raised to the power of exponent.
//
// Ports Dart's pow: raising a unit-bearing number to a power would produce
// units the CSS layer cannot interpret, so both operands must be unitless
// and the result is unitless.
func PowNumber(base, exponent SassNumber) (SassNumber, error) {
	if err := base.AssertNoUnits(new("base")); err != nil {
		return nil, err
	}
	if err := exponent.AssertNoUnits(new("exponent")); err != nil {
		return nil, err
	}
	return NewUnitlessNumber(sassmath.Pow(base.NumValue(), exponent.NumValue())), nil
}

// Atan2Number returns the arctangent of y/x in degrees.
//
// Ports Dart's atan2: x is converted into y's units first (so mixed angle
// units reconcile before the division), and the result carries deg like the
// other inverse trig functions.
func Atan2Number(y, x SassNumber) (SassNumber, error) {
	xVal, err := x.ConvertValueToMatch(y, new("x"), new("y"))
	if err != nil {
		return nil, err
	}
	return radiansToDegrees(sassmath.Atan2(y.NumValue(), xVal)), nil
}
