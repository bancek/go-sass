// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/util/number.dart

import (
	"fmt"
	"math"

	"github.com/bancek/go-sass/sasscommon"
)

// Precision is the power of ten to which Sass numbers round to determine
// whether they are fuzzy-equal to one another: the number of distinct
// digits emitted when converting a number to CSS.
//
// Declared on Dart's SassNumber; consumed here. See the fuzzy-equality
// table in the Sass number spec.
//
// Matches Dart: SassNumber.precision (value/number.dart)
const Precision = 10

// 10^(-Precision-1): the minimum distance such that a-b > epsilon implies
// a isn't fuzzy-equal to b. The converse need not hold: 5.1e-11 and 4.4e-11
// differ by less than epsilon yet round to distinct 11th-digit buckets.
//
// Matches Dart: _epsilon (util/number.dart)
var epsilon = math.Pow10(-Precision - 1)

// 1/epsilon, cached since Pow10 may not constant-fold.
//
// Matches Dart: _inverseEpsilon (util/number.dart)
var inverseEpsilon = math.Pow10(Precision + 1)

// FuzzyEquals returns whether a and b are equal up to the 11th decimal
// digit: their difference fits within epsilon and they round to the same
// 11th-digit bucket.
//
// Matches Dart: fuzzyEquals (util/number.dart)
func FuzzyEquals(a, b float64) bool {
	if a == b {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= epsilon && math.Round(a*inverseEpsilon) == math.Round(b*inverseEpsilon)
}

// FuzzyEqualsNullable behaves like FuzzyEquals but threads presence as
// separate missing flags: two missing values are equal, while a missing
// value never equals a present one. Dart spells the same contract with
// nullable numbers.
//
// Matches Dart: fuzzyEqualsNullable (util/number.dart)
func FuzzyEqualsNullable(v1, v2 float64, m1, m2 bool) bool {
	if m1 && m2 {
		return true
	}
	if m1 || m2 {
		return false
	}
	return FuzzyEquals(v1, v2)
}

// FuzzyHashCode returns a hash code for n that matches FuzzyEquals: the
// rounded 11th-digit bucket.
//
// Non-finite inputs hash to 0. Dart returns their own hash codes; Go has
// no matching float hashCode, so a fixed bucket is used deliberately.
//
// Matches Dart: fuzzyHashCode (util/number.dart)
func FuzzyHashCode(n float64) int {
	if !math.IsInf(n, 0) && !math.IsNaN(n) {
		return int(math.Round(n * inverseEpsilon))
	}
	return 0
}

// FuzzyLessThan returns whether a is less than b, and not FuzzyEquals to
// it.
//
// Matches Dart: fuzzyLessThan (util/number.dart)
func FuzzyLessThan(a, b float64) bool {
	return a < b && !FuzzyEquals(a, b)
}

// FuzzyLessThanOrEquals returns whether a is less than b, or FuzzyEquals
// to it.
//
// Matches Dart: fuzzyLessThanOrEquals (util/number.dart)
func FuzzyLessThanOrEquals(a, b float64) bool {
	return a < b || FuzzyEquals(a, b)
}

// FuzzyGreaterThan returns whether a is greater than b, and not
// FuzzyEquals to it.
//
// Matches Dart: fuzzyGreaterThan (util/number.dart)
func FuzzyGreaterThan(a, b float64) bool {
	return a > b && !FuzzyEquals(a, b)
}

// FuzzyGreaterThanOrEquals returns whether a is greater than b, or
// FuzzyEquals to it.
//
// Matches Dart: fuzzyGreaterThanOrEquals (util/number.dart)
func FuzzyGreaterThanOrEquals(a, b float64) bool {
	return a > b || FuzzyEquals(a, b)
}

// FuzzyInRange returns whether v is within min and max inclusive, using
// fuzzy equality at the bounds.
//
// Matches Dart: fuzzyInRange (util/number.dart)
func FuzzyInRange(v, min, max float64) bool {
	return FuzzyGreaterThanOrEquals(v, min) && FuzzyLessThanOrEquals(v, max)
}

// FuzzyIsInt returns whether n is FuzzyEquals to an integer. Infinite and
// NaN values are never integers.
//
// Matches Dart: fuzzyIsInt (util/number.dart)
func FuzzyIsInt(n float64) bool {
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return false
	}
	return FuzzyEquals(n, math.Round(n))
}

// FuzzyAsInt returns n as an integer if it is one according to FuzzyIsInt,
// reporting false otherwise. Values outside the int64 range also report
// false, mirroring Dart's 64-bit round bounds.
//
// Matches Dart: fuzzyAsInt (util/number.dart)
func FuzzyAsInt(n float64) (int64, bool) {
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, false
	}
	r := math.Round(n)
	if FuzzyEquals(n, r) {
		if r >= 1<<63 || r < -(1<<63) {
			return 0, false
		}
		return int64(r), true
	}
	return 0, false
}

// AsIntForSerialize returns n as an integer if it's "close enough".
//
// Normally "close enough" includes fuzzy matching, but in inspect mode we want
// to show the full precision of every number, so outside inspect mode "close
// enough" requires literally being an integer.
//
// Matches Dart: _SerializeVisitor._asInt (serialize.dart).
func AsIntForSerialize(n float64, inspect bool) (int64, bool) {
	if inspect {
		return FuzzyAsInt(n)
	}
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, false
	}
	r := math.Round(n)
	if r != n {
		return 0, false
	}
	if r >= 1<<63 || r < -(1<<63) {
		return 0, false
	}
	return int64(r), true
}

// FuzzyCheckRange returns n if it is within min and max, reporting false
// if it is not. Values FuzzyEquals to either bound clamp to that bound.
//
// Matches Dart: fuzzyCheckRange (util/number.dart)
func FuzzyCheckRange(n, min, max float64) (float64, bool) {
	if FuzzyEquals(n, min) {
		return min, true
	}
	if FuzzyEquals(n, max) {
		return max, true
	}
	if n > min && n < max {
		return n, true
	}
	return 0, false
}

// FuzzyAssertRange returns n if it is within min and max, or an error if
// not. Values FuzzyEquals to either bound clamp to that bound; name is
// used in error reporting.
//
// Matches Dart: fuzzyAssertRange (util/number.dart)
func FuzzyAssertRange(number float64, min, max int, name *string) (float64, error) {
	result, ok := FuzzyCheckRange(number, float64(min), float64(max))
	if ok {
		return result, nil
	}
	nameStr := ""
	if name != nil && *name != "" {
		nameStr = *name
	}
	return 0, &sasscommon.RangeError{Name: nameStr, Message: fmt.Sprintf("must be between %d and %d: %v", min, max, number)}
}

// FuzzyRound rounds n to the nearest integer. Numbers FuzzyEquals to X.5
// round away from the half below: up for positives, and for negatives the
// half itself floors while anything above it ceils.
//
// Matches Dart: fuzzyRound (util/number.dart)
func FuzzyRound(n float64) (int64, error) {
	if n < math.MinInt64 || n > math.MaxInt64 {
		return 0, fmt.Errorf("cannot round %v to int64: out of range", n)
	}
	if n > 0 {
		if FuzzyLessThan(n-math.Floor(n), 0.5) {
			return int64(math.Floor(n)), nil
		}
		return int64(math.Ceil(n)), nil
	}
	mod := n - math.Floor(n)
	if FuzzyLessThanOrEquals(mod, 0.5) {
		return int64(math.Floor(n)), nil
	}
	return int64(math.Ceil(n)), nil
}

// ModuloLikeSass returns a modulo b using Sass's floored division modulo
// semantics, which it inherited from Ruby and which differ from Go's.
// Infinite dividends and zero divisors produce NaN; an infinite divisor
// returns the dividend when its sign matches, NaN otherwise.
//
// Matches Dart: moduloLikeSass (util/number.dart)
func ModuloLikeSass(a, b float64) float64 {
	if math.IsInf(a, 0) {
		return math.NaN()
	}
	if math.IsInf(b, 0) {
		if SignIncludingZero(a) == math.Copysign(1, b) {
			return a
		}
		return math.NaN()
	}
	if b > 0 {
		result := math.Mod(a, b)
		if result < 0 {
			result += b
		}
		if result == 0 {
			return 0
		}
		return result
	}
	if b == 0 {
		return math.NaN()
	}
	// Dart's % operator returns an always-non-negative Euclidean modulo, but
	// Go's math.Mod returns a truncated result with the same sign as the
	// dividend. When the divisor is negative, we first convert from Go's
	// truncated result to Dart's Euclidean result (non-negative), then apply
	// the same correction as Dart's moduloLikeSass.
	result := math.Mod(a, b)
	if result < 0 {
		result += -b
	}
	if result == 0 {
		return 0
	}
	return result + b
}

// IsNegativeZero returns whether f is the special value negative zero.
//
// Matches Dart: DoubleWithSignedZero.isNegativeZero (util/number.dart), which
// compares with crossPlatformIdentical. Go targets one platform, so a bit
// comparison suffices for the same observable behavior.
func IsNegativeZero(f float64) bool {
	return math.Float64bits(f) == math.Float64bits(math.Copysign(0, -1))
}

// NormalizeLinear maps NaN and negative zero to 0.0, passing everything else
// through.
//
// Matches Dart: SassColor._normalizeLinear (value/color.dart).
func NormalizeLinear(v float64) float64 {
	if v == 0 || math.IsNaN(v) {
		return 0.0
	}
	return v
}

// SignIncludingZero returns the sign of f, treating +0 and -0 as +1 and -1
// respectively.
//
// Matches Dart: DoubleWithSignedZero.signIncludingZero
func SignIncludingZero(f float64) float64 {
	if IsNegativeZero(f) {
		return -1.0
	}
	if f == 0 {
		return 1.0
	}
	return math.Copysign(1, f)
}

// ClampLikeCss returns v clamped between lower and upper, with NaN preferring
// the lower bound (unlike Go's math functions for which NaN prefers the upper).
//
// Like Dart's `num.clamp`, negative zero clamps to the lower bound when the
// lower bound is zero (Dart's `compareTo` orders -0.0 below +0.0), so
// `ClampLikeCss(-0.0, 0, 1)` yields `+0.0` (#2840).
//
// Matches Dart: clampLikeCss
func ClampLikeCss(v, lower, upper float64) float64 {
	if math.IsNaN(v) {
		return lower
	}
	if v < lower || (IsNegativeZero(v) && lower == 0) {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}
