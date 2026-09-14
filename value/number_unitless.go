// Copyright 2020 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/number/unitless.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// UnitlessSassNumber is a SassNumber with no units.
//
// This is one of the three SassNumber implementations Go keeps 1:1 with
// Dart (unitless, single-unit, complex), sharing storage and common behavior
// through the embedded sassNumberBase. Unitless numbers deliberately bypass
// unit coercion in arithmetic and coercion: the raw value composes directly
// with whatever units the other operand carries, mirroring Dart's
// UnitlessSassNumber overrides.
type UnitlessSassNumber struct {
	base sassNumberBase
}

// newUnitlessNumber builds a unitless number without validation, mirroring
// Dart's UnitlessSassNumber constructor. SassNumberWithUnits selects this
// subclass exactly when both unit lists are empty.
func newUnitlessNumber(v float64, asSlash *slashedPair) *UnitlessSassNumber {
	return &UnitlessSassNumber{
		base: sassNumberBase{
			value:   v,
			asSlash: asSlash,
		},
	}
}

// --- Value interface ---
//
// These realizations are identical across the three number subclasses and
// delegate to the shared number helpers: every number is truthy, never
// blank, and counts as a single-element unbracketed list.

func (n *UnitlessSassNumber) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitNumber(n)
}
func (n *UnitlessSassNumber) isValue() {}

// Equals reports Sass value equality (unitless only ever equals unitless).
func (n *UnitlessSassNumber) Equals(other Value) bool { return numEqual(n, other) }

// IsTruthy reports true: all numbers are truthy.
func (n *UnitlessSassNumber) IsTruthy() bool { return numIsTruthy() }

// Separator reports Undecided: a number is a single-element list.
func (n *UnitlessSassNumber) Separator() ListSeparator { return numSeparator() }

// HasBrackets reports false: numbers never render brackets.
func (n *UnitlessSassNumber) HasBrackets() bool { return numHasBrackets() }

// AsList wraps the number in a single-element list.
func (n *UnitlessSassNumber) AsList() ([]Value, error) { return []Value{n}, nil }

// LengthAsList reports 1.
func (n *UnitlessSassNumber) LengthAsList() int { return numLengthAsList() }

// IsBlank reports false.
func (n *UnitlessSassNumber) IsBlank() bool { return numIsBlank() }

// IsSpecialNumber reports false: plain numbers are not special CSS numbers.
func (n *UnitlessSassNumber) IsSpecialNumber() bool { return numIsSpecialNumber() }

// IsSpecialVariable reports false: numbers are never var()/attr()/if()
// references.
func (n *UnitlessSassNumber) IsSpecialVariable() bool { return numIsSpecialVariable() }

// TryMap returns nil: numbers are never maps.
func (n *UnitlessSassNumber) TryMap() *SassMap { return numTryMap() }

// ToCssString serializes the number as CSS.
func (n *UnitlessSassNumber) ToCssString(q bool) (string, error) { return SerializeValue(n, q) }

// String serializes the number in inspect form.
func (n *UnitlessSassNumber) String() (string, error) { return SerializeValueInspect(n) }

// --- Operators ---
//
// Binary and unary Sass operators, dispatching on the operand's dynamic type
// through the shared helpers.

func (n *UnitlessSassNumber) RealNull() Value { return DefaultRealNull(n) }

// SingleEquals implements the Sass == operator.
func (n *UnitlessSassNumber) SingleEquals(o Value) (Value, error) { return numSingleEquals(n, o) }

// Plus implements the Sass + operator.
func (n *UnitlessSassNumber) Plus(o Value) (Value, error) { return numPlus(n, o) }

// Minus implements the Sass - operator.
func (n *UnitlessSassNumber) Minus(o Value) (Value, error) { return numMinus(n, o) }

// Times implements the Sass * operator.
func (n *UnitlessSassNumber) Times(o Value) (Value, error) { return numTimes(n, o) }

// DividedBy implements the Sass / operator.
func (n *UnitlessSassNumber) DividedBy(o Value) (Value, error) { return numDividedBy(n, o) }

// Modulo implements the Sass % operator with floored-division semantics.
func (n *UnitlessSassNumber) Modulo(o Value) (Value, error) { return numModulo(n, o) }

// GreaterThan implements the Sass > operator.
func (n *UnitlessSassNumber) GreaterThan(o Value) (Value, error) { return numGreaterThan(n, o) }

// GreaterThanOrEquals implements the Sass >= operator.
func (n *UnitlessSassNumber) GreaterThanOrEquals(o Value) (Value, error) {
	return numGreaterThanOrEquals(n, o)
}

// LessThan implements the Sass < operator.
func (n *UnitlessSassNumber) LessThan(o Value) (Value, error) { return numLessThan(n, o) }

// LessThanOrEquals implements the Sass <= operator.
func (n *UnitlessSassNumber) LessThanOrEquals(o Value) (Value, error) {
	return numLessThanOrEquals(n, o)
}

// UnaryPlus returns the number unchanged.
func (n *UnitlessSassNumber) UnaryPlus() (Value, error) { return numUnaryPlus(n) }

// UnaryMinus negates the number.
func (n *UnitlessSassNumber) UnaryMinus() (Value, error) { return numUnaryMinus(n) }

// UnaryDivide implements the / prefix operator.
func (n *UnitlessSassNumber) UnaryDivide() (Value, error) { return numUnaryDivide(n) }

// UnaryNot implements the `not` operator (always false for numbers).
func (n *UnitlessSassNumber) UnaryNot() (Value, error) { return numUnaryNot(n) }

// --- SassNumber interface: basic accessors ---
//
// Plain accessors shared with the other subclasses through the embedded base.

func (n *UnitlessSassNumber) IsInt() bool { return n.base.sharedIsInt() }

// AsInt returns the value as an int64 when it is fuzzy-equal to an integer.
func (n *UnitlessSassNumber) AsInt() (int64, bool) { return n.base.sharedAsInt() }

// NumValue returns the raw numeric value.
func (n *UnitlessSassNumber) NumValue() float64 { return n.base.sharedNumValue() }

// NumNumeratorUnits returns no units.
func (n *UnitlessSassNumber) NumNumeratorUnits() []string { return n.base.sharedNumNumeratorUnits() }

// NumDenominatorUnits returns no units.
func (n *UnitlessSassNumber) NumDenominatorUnits() []string {
	return n.base.sharedNumDenominatorUnits()
}

// HasSlash reports whether the number was written with slash syntax.
func (n *UnitlessSassNumber) HasSlash() bool { return n.base.sharedHasSlash() }

// SlashPair returns the slash-separated operands, if any.
func (n *UnitlessSassNumber) SlashPair() (SassNumber, SassNumber) { return n.base.sharedSlashPair() }

// AssertInt asserts the value is fuzzy-integral, naming name on failure.
func (n *UnitlessSassNumber) AssertInt(name *string) (int64, error) {
	return numAssertInt(n, n.base.value, name)
}

// ValueInRange checks the value against [min, max] with fuzzy equality,
// clamping onto an endpoint when fuzzy-equal to it.
func (n *UnitlessSassNumber) ValueInRange(min, max float64, name *string) (float64, *sasscommon.SassScriptException) {
	return numValueInRange(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, min, max, name)
}

// ValueInRangeWithUnit checks the value against [min, max] expressed in unit.
func (n *UnitlessSassNumber) ValueInRangeWithUnit(min, max float64, name, unit string) (float64, *sasscommon.SassScriptException) {
	return numValueInRangeWithUnit(n, n.base.value, min, max, name, unit)
}

// UnitString renders the unit suffix (always empty for unitless numbers).
func (n *UnitlessSassNumber) UnitString() string {
	return numUnitString(n.base.numeratorUnits, n.base.denominatorUnits)
}

// --- Unitless overrides: HasUnits/HasComplexUnits hardcoded ---

// HasUnits always reports false: a unitless number has no units by construction.
func (n *UnitlessSassNumber) HasUnits() bool { return false }

// HasComplexUnits always reports false.
func (n *UnitlessSassNumber) HasComplexUnits() bool { return false }

// HasUnit always reports false: no unit can match a unitless number.
func (n *UnitlessSassNumber) HasUnit(unit string) bool { return false }

// AssertUnit always raises a script error: a unitless number carries no unit.
func (n *UnitlessSassNumber) AssertUnit(unit string, name *string) *sasscommon.SassScriptException {
	return numAssertUnit(n, n.base.numeratorUnits, n.base.denominatorUnits, unit, name)
}

// AssertNoUnits always succeeds: a unitless number trivially has no units.
func (n *UnitlessSassNumber) AssertNoUnits(name *string) *sasscommon.SassScriptException {
	return numAssertNoUnits(n, n.HasUnits(), name)
}

// --- Unitless overrides: compatible/coercion ---
//
// Coercion from unitless is infallible and value-preserving: the raw value
// is simply re-homed into the target's units. Conversion (as opposed to
// coercion) still rejects targets that carry units, routing through the
// shared implementation for the consistent error message.

// HasCompatibleUnits reports whether other is also unitless: only unitless
// numbers convert freely into one another.
func (n *UnitlessSassNumber) HasCompatibleUnits(other SassNumber) bool {
	return numHasCompatibleUnits(n, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// CompatibleWithUnit always reports true: a unitless value can adopt any
// unit without conversion, mirroring Dart's UnitlessSassNumber override.
func (n *UnitlessSassNumber) CompatibleWithUnit(unit string) bool { return true }

// HasPossiblyCompatibleUnits reports whether other is also unitless.
func (n *UnitlessSassNumber) HasPossiblyCompatibleUnits(other SassNumber) bool {
	return numHasPossiblyCompatibleUnits(n, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// CoerceToMatch copies the raw value into other's units without conversion.
func (n *UnitlessSassNumber) CoerceToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	return other.WithValue(n.base.value), nil
}

// CoerceValueToMatch returns the raw value unchanged: no conversion factor
// applies when adopting the target's units.
func (n *UnitlessSassNumber) CoerceValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	return n.base.value, nil
}

// ConvertToMatch returns this number when other is unitless, and raises the
// shared conversion error when other carries units.
func (n *UnitlessSassNumber) ConvertToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	if other.HasUnits() {
		return numConvertToMatch(n, other, name, otherName)
	}
	return n, nil
}

// ConvertValueToMatch is the raw-value variant of ConvertToMatch.
func (n *UnitlessSassNumber) ConvertValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	if other.HasUnits() {
		return numConvertValueToMatch(n, other, name, otherName)
	}
	return n.base.value, nil
}

// CoerceValue returns the raw value: adopting any units needs no conversion.
func (n *UnitlessSassNumber) CoerceValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	return n.base.value, nil
}

// ConvertValue returns the raw value for a unitless target, and the shared
// conversion error for unit-bearing targets.
func (n *UnitlessSassNumber) ConvertValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	if len(newNumerators) > 0 || len(newDenominators) > 0 {
		return numConvertValue(n, newNumerators, newDenominators, name)
	}
	return n.base.value, nil
}

// ConvertValueToUnit converts the value into unit, rejecting unit-bearing
// targets through the shared error path.
func (n *UnitlessSassNumber) ConvertValueToUnit(unit string, name *string) (float64, error) {
	return n.ConvertValue([]string{unit}, nil, name)
}

// CoerceValueToUnit returns the raw value: any single unit can be adopted.
func (n *UnitlessSassNumber) CoerceValueToUnit(unit string, name *string) (float64, error) {
	return n.base.value, nil
}

// Coerce rebuilds the value with the requested units, no conversion needed.
func (n *UnitlessSassNumber) Coerce(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return SassNumberWithUnits(n.base.value, newNumerators, newDenominators), nil
}

// Convert converts into unitless targets and rejects unit-bearing ones via
// the shared error path.
func (n *UnitlessSassNumber) Convert(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return numConvert(n, newNumerators, newDenominators, name)
}

// --- WithValue/WithSlash/WithUnits ---
//
// Copy constructors that stay within the unitless subclass (Dart's
// withValue/withSlash); WithUnits re-selects whichever subclass fits the
// new shape.

// WithValue returns a unitless copy with value v.
func (n *UnitlessSassNumber) WithValue(v float64) SassNumber { return n.base.sharedWithValue(v) }

// WithSlash returns a unitless copy tagged with the slash-separated operands.
func (n *UnitlessSassNumber) WithSlash(num, den SassNumber) SassNumber {
	return n.base.sharedWithSlash(num, den)
}

// WithUnits returns a copy with replaced units, re-selecting whichever
// subclass fits the new shape.
func (n *UnitlessSassNumber) WithUnits(numUnits, denUnits []string) SassNumber {
	return n.base.sharedWithUnits(numUnits, denUnits)
}

// WithoutSlash returns a copy with the slash marker cleared.
func (n *UnitlessSassNumber) WithoutSlash() SassNumber { return n.base.sharedWithoutSlash(n) }

// UnitSuggestion renders the "expected a <type> unit" hint for error messages.
func (n *UnitlessSassNumber) UnitSuggestion(name string, unit *string) string {
	return numUnitSuggestion(name, n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// --- Equals / HashCode ---

// SassNumberEquals reports value equality with fuzzy comparison; unitless
// numbers only ever equal other unitless numbers.
func (n *UnitlessSassNumber) SassNumberEquals(other SassNumber) bool {
	return numEquals(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// HashCode hashes the raw value with fuzzy granularity, matching SassNumberEquals.
func (n *UnitlessSassNumber) HashCode() int { return n.base.sharedHashCode() }

// --- Arithmetic: unitless overrides ---
//
// These bypass unit coercion entirely, matching Dart's UnitlessSassNumber
// operator overrides: the raw values combine directly and the result adopts
// the other operand's units (or stays unitless when both sides are bare).

// plusNum adds the raw values, adopting other's units when it has any.
func (n *UnitlessSassNumber) plusNum(other SassNumber) (SassNumber, error) {
	if other.HasUnits() {
		return other.WithValue(n.base.value + other.NumValue()), nil
	}
	return NewUnitlessNumber(n.base.value + other.NumValue()), nil
}

// minusNum subtracts the raw values, adopting other's units when it has any.
func (n *UnitlessSassNumber) minusNum(other SassNumber) (SassNumber, error) {
	if other.HasUnits() {
		return other.WithValue(n.base.value - other.NumValue()), nil
	}
	return NewUnitlessNumber(n.base.value - other.NumValue()), nil
}

// timesNum multiplies the raw values, adopting other's units when it has any.
func (n *UnitlessSassNumber) timesNum(other SassNumber) (SassNumber, error) {
	if other.HasUnits() {
		return other.WithValue(n.base.value * other.NumValue()), nil
	}
	return NewUnitlessNumber(n.base.value * other.NumValue()), nil
}

// dividedByNum divides the raw values. Dividing by a unit-bearing number
// inverts its units (numerators become denominators), matching Dart's
// UnitlessSassNumber.dividedBy inverse-unit construction.
func (n *UnitlessSassNumber) dividedByNum(other SassNumber) (SassNumber, error) {
	if other.HasUnits() {
		return SassNumberWithUnits(n.base.value/other.NumValue(), other.NumDenominatorUnits(), other.NumNumeratorUnits()), nil
	}
	return NewUnitlessNumber(n.base.value / other.NumValue()), nil
}

// unaryMinusNum negates the raw value, staying unitless.
func (n *UnitlessSassNumber) unaryMinusNum() (SassNumber, error) {
	return NewUnitlessNumber(-n.base.value), nil
}

// moduloNum applies floored-division modulo to the raw values, adopting
// other's units when it has any.
func (n *UnitlessSassNumber) moduloNum(other SassNumber) (SassNumber, error) {
	if other.HasUnits() {
		return other.WithValue(util.ModuloLikeSass(n.base.value, other.NumValue())), nil
	}
	return NewUnitlessNumber(util.ModuloLikeSass(n.base.value, other.NumValue())), nil
}

// Comparisons use the raw values with fuzzy granularity and need no unit
// reconciliation, since both sides are effectively bare magnitudes.
func (n *UnitlessSassNumber) greaterThanNum(other SassNumber) (*SassBoolean, error) {
	result := util.FuzzyGreaterThan(n.base.value, other.NumValue())
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *UnitlessSassNumber) greaterThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result := util.FuzzyGreaterThanOrEquals(n.base.value, other.NumValue())
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *UnitlessSassNumber) lessThanNum(other SassNumber) (*SassBoolean, error) {
	result := util.FuzzyLessThan(n.base.value, other.NumValue())
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *UnitlessSassNumber) lessThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result := util.FuzzyLessThanOrEquals(n.base.value, other.NumValue())
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

// isComparableTo delegates to the shared helper: a unitless number is
// comparable exactly when the other side admits unitless comparison.
func (n *UnitlessSassNumber) isComparableTo(other SassNumber) bool {
	return numIsComparableTo(n, n.HasUnits(), other)
}

// --- multiplyUnits (inherits base) ---

// multiplyUnits folds other's units onto this (empty) unit shape through the
// shared composed-units construction.
func (n *UnitlessSassNumber) multiplyUnits(value float64, otherNumerators, otherDenominators []string) SassNumber {
	return numMultiplyUnits(value, n.base.numeratorUnits, n.base.denominatorUnits, otherNumerators, otherDenominators)
}

// --- coerceOrConvertValue ---

// coerceOrConvertValue is the shared conversion workhorse behind the
// Coerce*/Convert* family above.
func (n *UnitlessSassNumber) coerceOrConvertValue(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) (float64, error) {
	return numCoerceOrConvertValue(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, newNumerators, newDenominators, coerceUnitless, name, other, otherName)
}

// compatibilityError builds the "incompatible units" script error with the
// unit-type suggestion text.
func (n *UnitlessSassNumber) compatibilityError(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) error {
	return numCompatibilityError(n, newNumerators, newDenominators, name, other, otherName)
}
