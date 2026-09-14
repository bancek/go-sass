// Copyright 2020 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/number/single_unit.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// SingleUnitSassNumber is a SassNumber with a single numerator unit.
//
// This is one of the three SassNumber implementations Go keeps 1:1 with
// Dart (unitless, single-unit, complex), sharing storage and common behavior
// through the embedded sassNumberBase. Only the unit-shape-specific behavior
// — compatibility, coercion fast paths, arithmetic, equality — lives here,
// mirroring Dart's SingleUnitSassNumber overrides.
type SingleUnitSassNumber struct {
	base sassNumberBase
}

// newSingleUnitNumber builds a single-unit number without validation, mirroring
// Dart's positional SingleUnitSassNumber constructor. Callers select this
// subclass exactly when there is one numerator unit and no denominator units.
func newSingleUnitNumber(v float64, unit string, asSlash *slashedPair) *SingleUnitSassNumber {
	return &SingleUnitSassNumber{
		base: sassNumberBase{
			value:          v,
			numeratorUnits: []string{unit},
			asSlash:        asSlash,
		},
	}
}

// --- Value interface ---
//
// These realizations are identical across the three number subclasses and
// delegate to the shared number helpers: every number is truthy, never
// blank, and counts as a single-element unbracketed list.

func (n *SingleUnitSassNumber) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitNumber(n)
}
func (n *SingleUnitSassNumber) isValue() {}

// Equals reports Sass value equality, including unit conversion.
func (n *SingleUnitSassNumber) Equals(other Value) bool { return numEqual(n, other) }

// IsTruthy reports true: all numbers are truthy.
func (n *SingleUnitSassNumber) IsTruthy() bool { return numIsTruthy() }

// Separator reports Undecided: a number is a single-element list.
func (n *SingleUnitSassNumber) Separator() ListSeparator { return numSeparator() }

// HasBrackets reports false: numbers never render brackets.
func (n *SingleUnitSassNumber) HasBrackets() bool { return numHasBrackets() }

// AsList wraps the number in a single-element list.
func (n *SingleUnitSassNumber) AsList() ([]Value, error) { return []Value{n}, nil }

// LengthAsList reports 1.
func (n *SingleUnitSassNumber) LengthAsList() int { return numLengthAsList() }

// IsBlank reports false.
func (n *SingleUnitSassNumber) IsBlank() bool { return numIsBlank() }

// IsSpecialNumber reports false: plain numbers are not special CSS numbers.
func (n *SingleUnitSassNumber) IsSpecialNumber() bool { return numIsSpecialNumber() }

// IsSpecialVariable reports false: numbers are never var()/attr()/if()
// references.
func (n *SingleUnitSassNumber) IsSpecialVariable() bool { return numIsSpecialVariable() }

// TryMap returns nil: numbers are never maps.
func (n *SingleUnitSassNumber) TryMap() *SassMap { return numTryMap() }

// ToCssString serializes the number as CSS.
func (n *SingleUnitSassNumber) ToCssString(q bool) (string, error) { return SerializeValue(n, q) }

// String serializes the number in inspect form.
func (n *SingleUnitSassNumber) String() (string, error) { return SerializeValueInspect(n) }

// --- Operators ---
//
// Binary and unary Sass operators, dispatching on the operand's dynamic type
// through the shared helpers.

func (n *SingleUnitSassNumber) RealNull() Value { return DefaultRealNull(n) }

// SingleEquals implements the Sass == operator.
func (n *SingleUnitSassNumber) SingleEquals(o Value) (Value, error) { return numSingleEquals(n, o) }

// Plus implements the Sass + operator.
func (n *SingleUnitSassNumber) Plus(o Value) (Value, error) { return numPlus(n, o) }

// Minus implements the Sass - operator.
func (n *SingleUnitSassNumber) Minus(o Value) (Value, error) { return numMinus(n, o) }

// Times implements the Sass * operator.
func (n *SingleUnitSassNumber) Times(o Value) (Value, error) { return numTimes(n, o) }

// DividedBy implements the Sass / operator.
func (n *SingleUnitSassNumber) DividedBy(o Value) (Value, error) { return numDividedBy(n, o) }

// Modulo implements the Sass % operator with floored-division semantics.
func (n *SingleUnitSassNumber) Modulo(o Value) (Value, error) { return numModulo(n, o) }

// GreaterThan implements the Sass > operator.
func (n *SingleUnitSassNumber) GreaterThan(o Value) (Value, error) { return numGreaterThan(n, o) }

// GreaterThanOrEquals implements the Sass >= operator.
func (n *SingleUnitSassNumber) GreaterThanOrEquals(o Value) (Value, error) {
	return numGreaterThanOrEquals(n, o)
}

// LessThan implements the Sass < operator.
func (n *SingleUnitSassNumber) LessThan(o Value) (Value, error) { return numLessThan(n, o) }

// LessThanOrEquals implements the Sass <= operator.
func (n *SingleUnitSassNumber) LessThanOrEquals(o Value) (Value, error) {
	return numLessThanOrEquals(n, o)
}

// UnaryPlus returns the number unchanged.
func (n *SingleUnitSassNumber) UnaryPlus() (Value, error) { return numUnaryPlus(n) }

// UnaryMinus negates the number, preserving its unit.
func (n *SingleUnitSassNumber) UnaryMinus() (Value, error) { return numUnaryMinus(n) }

// UnaryDivide implements the / prefix operator.
func (n *SingleUnitSassNumber) UnaryDivide() (Value, error) { return numUnaryDivide(n) }

// UnaryNot implements the `not` operator (always false for numbers).
func (n *SingleUnitSassNumber) UnaryNot() (Value, error) { return numUnaryNot(n) }

// --- SassNumber interface ---
//
// Plain accessors shared with the other subclasses through the embedded base.

func (n *SingleUnitSassNumber) IsInt() bool { return n.base.sharedIsInt() }

// AsInt returns the value as an int64 when it is fuzzy-equal to an integer.
func (n *SingleUnitSassNumber) AsInt() (int64, bool) { return n.base.sharedAsInt() }

// NumValue returns the raw numeric value.
func (n *SingleUnitSassNumber) NumValue() float64 { return n.base.sharedNumValue() }

// NumNumeratorUnits returns the single numerator unit.
func (n *SingleUnitSassNumber) NumNumeratorUnits() []string { return n.base.sharedNumNumeratorUnits() }

// NumDenominatorUnits returns no units: single-unit numbers never carry
// denominators.
func (n *SingleUnitSassNumber) NumDenominatorUnits() []string {
	return n.base.sharedNumDenominatorUnits()
}

// HasSlash reports whether the number was written with slash syntax.
func (n *SingleUnitSassNumber) HasSlash() bool { return n.base.sharedHasSlash() }

// SlashPair returns the slash-separated operands, if any.
func (n *SingleUnitSassNumber) SlashPair() (SassNumber, SassNumber) { return n.base.sharedSlashPair() }

// AssertInt asserts the value is fuzzy-integral, naming name on failure.
func (n *SingleUnitSassNumber) AssertInt(name *string) (int64, error) {
	return numAssertInt(n, n.base.value, name)
}

// ValueInRange checks the value against [min, max] with fuzzy equality,
// clamping onto an endpoint when fuzzy-equal to it.
func (n *SingleUnitSassNumber) ValueInRange(min, max float64, name *string) (float64, *sasscommon.SassScriptException) {
	return numValueInRange(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, min, max, name)
}

// ValueInRangeWithUnit checks the value against [min, max] expressed in unit.
func (n *SingleUnitSassNumber) ValueInRangeWithUnit(min, max float64, name, unit string) (float64, *sasscommon.SassScriptException) {
	return numValueInRangeWithUnit(n, n.base.value, min, max, name, unit)
}

// UnitString renders the unit suffix (here, the single unit itself).
func (n *SingleUnitSassNumber) UnitString() string {
	return numUnitString(n.base.numeratorUnits, n.base.denominatorUnits)
}

// --- SingleUnit: HasUnits/HasComplexUnits ---

// HasUnits always reports true: a single-unit number has units by construction.
func (n *SingleUnitSassNumber) HasUnits() bool { return true }

// HasComplexUnits always reports false: one numerator unit is never complex.
func (n *SingleUnitSassNumber) HasComplexUnits() bool { return false }

// HasUnit reports whether unit is exactly this number's unit.
func (n *SingleUnitSassNumber) HasUnit(unit string) bool {
	return numHasUnit(n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// AssertUnit asserts the number carries unit, raising a script error otherwise.
func (n *SingleUnitSassNumber) AssertUnit(unit string, name *string) *sasscommon.SassScriptException {
	return numAssertUnit(n, n.base.numeratorUnits, n.base.denominatorUnits, unit, name)
}

// AssertNoUnits raises a script error: a single-unit number always has units.
func (n *SingleUnitSassNumber) AssertNoUnits(name *string) *sasscommon.SassScriptException {
	return numAssertNoUnits(n, n.HasUnits(), name)
}

// --- Compatible ---
//
// Compatibility against other numbers and bare unit names. The
// possibly-compatible check encodes the browser-knowledge rule: a unit with
// no entry in the known-compatibilities table is possibly compatible with
// anything, while a known unit only admits same-set members and unknown
// units.

// HasCompatibleUnits reports whether other converts into this number's unit.
func (n *SingleUnitSassNumber) HasCompatibleUnits(other SassNumber) bool {
	return numHasCompatibleUnits(n, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// CompatibleWithUnit reports whether unit converts into this number's unit.
func (n *SingleUnitSassNumber) CompatibleWithUnit(unit string) bool {
	return numCompatibleWithUnit(n, n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// HasPossiblyCompatibleUnits applies the browser-knowledge rule above,
// consulting the table hoisted into number_util.go.
func (n *SingleUnitSassNumber) HasPossiblyCompatibleUnits(other SassNumber) bool {
	return numHasPossiblyCompatibleUnits(n, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// --- Coercion/Conversion ---
//
// Single-to-single conversions take the factor fast path; anything else
// falls through to the shared base implementation, which raises the
// consistent conversion error (Dart's `?? super.coerce...` shape).

func (n *SingleUnitSassNumber) Convert(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return numConvert(n, newNumerators, newDenominators, name)
}

// Coerce converts the value into newNumerators/newDenominators, also
// accepting unitless sources that conversion would reject.
func (n *SingleUnitSassNumber) Coerce(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return numCoerce(n, newNumerators, newDenominators, name)
}

// CoerceToMatch coerces the value into other's units.
func (n *SingleUnitSassNumber) CoerceToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	return numCoerceToMatch(n, other, name, otherName)
}

// ConvertToMatch converts the value into other's units, rejecting unitless
// sources with units on the other side.
func (n *SingleUnitSassNumber) ConvertToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	return numConvertToMatch(n, other, name, otherName)
}

// CoerceValueToMatch returns the raw value expressed in other's units.
func (n *SingleUnitSassNumber) CoerceValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	return numCoerceValueToMatch(n, other, name, otherName)
}

// ConvertValueToMatch is the conversion-only variant of CoerceValueToMatch.
func (n *SingleUnitSassNumber) ConvertValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	return numConvertValueToMatch(n, other, name, otherName)
}

// CoerceValue is the raw-value variant of Coerce.
func (n *SingleUnitSassNumber) CoerceValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	return numCoerceValue(n, newNumerators, newDenominators, name)
}

// ConvertValue is the raw-value variant of Convert.
func (n *SingleUnitSassNumber) ConvertValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	return numConvertValue(n, newNumerators, newDenominators, name)
}

// ConvertValueToUnit converts the value into a single unit.
func (n *SingleUnitSassNumber) ConvertValueToUnit(unit string, name *string) (float64, error) {
	return numConvertValueToUnit(n, unit, name)
}

// CoerceValueToUnit coerces the value into a single unit.
func (n *SingleUnitSassNumber) CoerceValueToUnit(unit string, name *string) (float64, error) {
	return numCoerceValueToUnit(n, unit, name)
}

// --- WithValue/WithSlash ---
//
// Copy constructors that preserve the single unit (Dart's withValue/
// withSlash), re-selecting the subclass through the shared base.

// WithValue returns a copy with value v, keeping this number's unit.
func (n *SingleUnitSassNumber) WithValue(v float64) SassNumber { return n.base.sharedWithValue(v) }

// WithSlash returns a copy tagged with the slash-separated operands.
func (n *SingleUnitSassNumber) WithSlash(num, den SassNumber) SassNumber {
	return n.base.sharedWithSlash(num, den)
}

// WithUnits returns a copy with replaced units, re-selecting whichever
// subclass fits the new shape.
func (n *SingleUnitSassNumber) WithUnits(numUnits, denUnits []string) SassNumber {
	return n.base.sharedWithUnits(numUnits, denUnits)
}

// WithoutSlash returns a copy with the slash marker cleared.
func (n *SingleUnitSassNumber) WithoutSlash() SassNumber { return n.base.sharedWithoutSlash(n) }

// UnitSuggestion renders the "expected a <type> unit" hint for error messages.
func (n *SingleUnitSassNumber) UnitSuggestion(name string, unit *string) string {
	return numUnitSuggestion(name, n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// --- Equals / HashCode ---

// SassNumberEquals reports value equality, converting other into this
// number's unit and comparing with fuzzy equality.
func (n *SingleUnitSassNumber) SassNumberEquals(other SassNumber) bool {
	return numEquals(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// HashCode hashes the value rescaled by the unit's canonical multiplier, so
// numbers that compare equal across convertible units hash equal too.
func (n *SingleUnitSassNumber) HashCode() int { return n.base.sharedHashCode() }

// --- Arithmetic ---
//
// Addition, subtraction, modulo, and comparisons coerce the operand into
// this number's units first (shared numCoerceUnits). Multiplication and
// division by a unitless scalar preserve the unit; unit-bearing operands
// compose units through multiplyUnits below.

// plusNum adds other (coerced to this unit) and keeps this unit.
func (n *SingleUnitSassNumber) plusNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, func(a, b float64) float64 { return a + b })
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// minusNum subtracts other (coerced to this unit) and keeps this unit.
func (n *SingleUnitSassNumber) minusNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, func(a, b float64) float64 { return a - b })
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// moduloNum applies floored-division modulo in this unit.
func (n *SingleUnitSassNumber) moduloNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, util.ModuloLikeSass)
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// timesNum scales by a unitless operand, or composes units with a
// unit-bearing one via multiplyUnits.
func (n *SingleUnitSassNumber) timesNum(other SassNumber) (SassNumber, error) {
	if !other.HasUnits() {
		return n.WithValue(n.base.value * other.NumValue()), nil
	}
	return n.multiplyUnits(n.base.value*other.NumValue(), other.NumNumeratorUnits(), other.NumDenominatorUnits()), nil
}

// dividedByNum scales by a unitless divisor, or composes inverted units with
// a unit-bearing one (the divisor's numerators become denominators).
func (n *SingleUnitSassNumber) dividedByNum(other SassNumber) (SassNumber, error) {
	if !other.HasUnits() {
		return n.WithValue(n.base.value / other.NumValue()), nil
	}
	return n.multiplyUnits(n.base.value/other.NumValue(), other.NumDenominatorUnits(), other.NumNumeratorUnits()), nil
}

// unaryMinusNum negates the value, preserving the unit.
func (n *SingleUnitSassNumber) unaryMinusNum() (SassNumber, error) {
	return n.WithValue(-n.base.value), nil
}

// Comparisons coerce the operand into this unit and use fuzzy comparison so
// values equal up to the 11th decimal compare as ordered, not unequal.
func (n *SingleUnitSassNumber) greaterThanNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyGreaterThan)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *SingleUnitSassNumber) greaterThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyGreaterThanOrEquals)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *SingleUnitSassNumber) lessThanNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyLessThan)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *SingleUnitSassNumber) lessThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyLessThanOrEquals)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

// isComparableTo reports whether comparisons against other are meaningful:
// another single-unit number is comparable whenever coercion could reconcile
// the pair (unitless sources always qualify through the shared helper).
func (n *SingleUnitSassNumber) isComparableTo(other SassNumber) bool {
	return numIsComparableTo(n, n.HasUnits(), other)
}

// multiplyUnits folds other's units into this number's single unit,
// cancelling the first denominator that converts into it (absorbing the
// factor into the value) and otherwise prepending this unit to the
// numerators. This ports Dart's SingleUnitSassNumber.multiplyUnits loop,
// including its first-match-wins cancellation order.
func (n *SingleUnitSassNumber) multiplyUnits(value float64, otherNumerators, otherDenominators []string) SassNumber {
	return numMultiplyUnits(value, n.base.numeratorUnits, n.base.denominatorUnits, otherNumerators, otherDenominators)
}

// coerceOrConvertValue is the shared conversion workhorse behind the
// Coerce*/Convert* family above.
func (n *SingleUnitSassNumber) coerceOrConvertValue(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) (float64, error) {
	return numCoerceOrConvertValue(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, newNumerators, newDenominators, coerceUnitless, name, other, otherName)
}

// compatibilityError builds the "incompatible units" script error with the
// unit-type suggestion text.
func (n *SingleUnitSassNumber) compatibilityError(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) error {
	return numCompatibilityError(n, newNumerators, newDenominators, name, other, otherName)
}
