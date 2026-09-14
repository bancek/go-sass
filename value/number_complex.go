// Copyright 2020 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/number/complex.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ComplexSassNumber is a SassNumber with multiple numerator or denominator units.
//
// This is one of the three SassNumber implementations Go keeps 1:1 with
// Dart (unitless, single-unit, complex), sharing storage and common behavior
// through the embedded sassNumberBase. It covers every shape the other two
// do not: several numerator units, or any denominator units. Complex numbers
// can never convert, so unit-sensitive operations either compose units
// (multiplication/division) or coerce the other side into this shape.
type ComplexSassNumber struct {
	base sassNumberBase
}

// newComplexNumber builds a complex number without validation, mirroring
// Dart's ComplexSassNumber.new constructor. Callers select this subclass
// exactly when there is more than one numerator unit or at least one
// denominator unit (Dart asserts that invariant on construction).
func newComplexNumber(v float64, numUnits, denUnits []string, asSlash *slashedPair) *ComplexSassNumber {
	return &ComplexSassNumber{
		base: sassNumberBase{
			value:            v,
			numeratorUnits:   numUnits,
			denominatorUnits: denUnits,
			asSlash:          asSlash,
		},
	}
}

// --- Value interface ---
//
// These realizations are identical across the three number subclasses and
// delegate to the shared number helpers: every number is truthy, never
// blank, and counts as a single-element unbracketed list.

func (n *ComplexSassNumber) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitNumber(n)
}
func (n *ComplexSassNumber) isValue() {}

// Equals reports Sass value equality, including unit conversion.
func (n *ComplexSassNumber) Equals(other Value) bool { return numEqual(n, other) }

// IsTruthy reports true: all numbers are truthy.
func (n *ComplexSassNumber) IsTruthy() bool { return numIsTruthy() }

// Separator reports Undecided: a number is a single-element list.
func (n *ComplexSassNumber) Separator() ListSeparator { return numSeparator() }

// HasBrackets reports false: numbers never render brackets.
func (n *ComplexSassNumber) HasBrackets() bool { return numHasBrackets() }

// AsList wraps the number in a single-element list.
func (n *ComplexSassNumber) AsList() ([]Value, error) { return []Value{n}, nil }

// LengthAsList reports 1.
func (n *ComplexSassNumber) LengthAsList() int { return numLengthAsList() }

// IsBlank reports false.
func (n *ComplexSassNumber) IsBlank() bool { return numIsBlank() }

// IsSpecialNumber reports false: plain numbers are not special CSS numbers.
func (n *ComplexSassNumber) IsSpecialNumber() bool { return numIsSpecialNumber() }

// IsSpecialVariable reports false: numbers are never var()/attr()/if()
// references.
func (n *ComplexSassNumber) IsSpecialVariable() bool { return numIsSpecialVariable() }

// TryMap returns nil: numbers are never maps.
func (n *ComplexSassNumber) TryMap() *SassMap { return numTryMap() }

// ToCssString serializes the number as CSS.
func (n *ComplexSassNumber) ToCssString(q bool) (string, error) { return SerializeValue(n, q) }

// String serializes the number in inspect form.
func (n *ComplexSassNumber) String() (string, error) { return SerializeValueInspect(n) }

// --- Operators ---
//
// Binary and unary Sass operators, dispatching on the operand's dynamic type
// through the shared helpers.

func (n *ComplexSassNumber) RealNull() Value { return DefaultRealNull(n) }

// SingleEquals implements the Sass == operator.
func (n *ComplexSassNumber) SingleEquals(o Value) (Value, error) { return numSingleEquals(n, o) }

// Plus implements the Sass + operator.
func (n *ComplexSassNumber) Plus(o Value) (Value, error) { return numPlus(n, o) }

// Minus implements the Sass - operator.
func (n *ComplexSassNumber) Minus(o Value) (Value, error) { return numMinus(n, o) }

// Times implements the Sass * operator.
func (n *ComplexSassNumber) Times(o Value) (Value, error) { return numTimes(n, o) }

// DividedBy implements the Sass / operator.
func (n *ComplexSassNumber) DividedBy(o Value) (Value, error) { return numDividedBy(n, o) }

// Modulo implements the Sass % operator with floored-division semantics.
func (n *ComplexSassNumber) Modulo(o Value) (Value, error) { return numModulo(n, o) }

// GreaterThan implements the Sass > operator.
func (n *ComplexSassNumber) GreaterThan(o Value) (Value, error) { return numGreaterThan(n, o) }

// GreaterThanOrEquals implements the Sass >= operator.
func (n *ComplexSassNumber) GreaterThanOrEquals(o Value) (Value, error) {
	return numGreaterThanOrEquals(n, o)
}

// LessThan implements the Sass < operator.
func (n *ComplexSassNumber) LessThan(o Value) (Value, error) { return numLessThan(n, o) }

// LessThanOrEquals implements the Sass <= operator.
func (n *ComplexSassNumber) LessThanOrEquals(o Value) (Value, error) {
	return numLessThanOrEquals(n, o)
}

// UnaryPlus returns the number unchanged.
func (n *ComplexSassNumber) UnaryPlus() (Value, error) { return numUnaryPlus(n) }

// UnaryMinus negates the number, preserving its units.
func (n *ComplexSassNumber) UnaryMinus() (Value, error) { return numUnaryMinus(n) }

// UnaryDivide implements the / prefix operator.
func (n *ComplexSassNumber) UnaryDivide() (Value, error) { return numUnaryDivide(n) }

// UnaryNot implements the `not` operator (always false for numbers).
func (n *ComplexSassNumber) UnaryNot() (Value, error) { return numUnaryNot(n) }

// --- SassNumber interface ---
//
// Plain accessors shared with the other subclasses through the embedded base.

func (n *ComplexSassNumber) IsInt() bool { return n.base.sharedIsInt() }

// AsInt returns the value as an int64 when it is fuzzy-equal to an integer.
func (n *ComplexSassNumber) AsInt() (int64, bool) { return n.base.sharedAsInt() }

// NumValue returns the raw numeric value.
func (n *ComplexSassNumber) NumValue() float64 { return n.base.sharedNumValue() }

// NumNumeratorUnits returns the numerator units.
func (n *ComplexSassNumber) NumNumeratorUnits() []string { return n.base.sharedNumNumeratorUnits() }

// NumDenominatorUnits returns the denominator units.
func (n *ComplexSassNumber) NumDenominatorUnits() []string { return n.base.sharedNumDenominatorUnits() }

// HasSlash reports whether the number was written with slash syntax.
func (n *ComplexSassNumber) HasSlash() bool { return n.base.sharedHasSlash() }

// SlashPair returns the slash-separated operands, if any.
func (n *ComplexSassNumber) SlashPair() (SassNumber, SassNumber) { return n.base.sharedSlashPair() }

// AssertInt asserts the value is fuzzy-integral, naming name on failure.
func (n *ComplexSassNumber) AssertInt(name *string) (int64, error) {
	return numAssertInt(n, n.base.value, name)
}

// ValueInRange checks the value against [min, max] with fuzzy equality,
// clamping onto an endpoint when fuzzy-equal to it.
func (n *ComplexSassNumber) ValueInRange(min, max float64, name *string) (float64, *sasscommon.SassScriptException) {
	return numValueInRange(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, min, max, name)
}

// ValueInRangeWithUnit checks the value against [min, max] expressed in unit.
func (n *ComplexSassNumber) ValueInRangeWithUnit(min, max float64, name, unit string) (float64, *sasscommon.SassScriptException) {
	return numValueInRangeWithUnit(n, n.base.value, min, max, name, unit)
}

// UnitString renders the composed unit suffix (e.g. px*mm/cm).
func (n *ComplexSassNumber) UnitString() string {
	return numUnitString(n.base.numeratorUnits, n.base.denominatorUnits)
}

// --- Complex: HasUnits/HasComplexUnits ---

// HasUnits always reports true: a complex number has units by construction.
func (n *ComplexSassNumber) HasUnits() bool { return true }

// HasComplexUnits always reports true: this is the complex subclass.
func (n *ComplexSassNumber) HasComplexUnits() bool { return true }

// HasUnit always reports false: no single unit describes a composed shape.
func (n *ComplexSassNumber) HasUnit(unit string) bool { return false }

// AssertUnit asserts the composed units match unit via the shared helper,
// which reports the full unit shape on failure.
func (n *ComplexSassNumber) AssertUnit(unit string, name *string) *sasscommon.SassScriptException {
	return numAssertUnit(n, n.base.numeratorUnits, n.base.denominatorUnits, unit, name)
}

// AssertNoUnits always raises a script error: a complex number has units.
func (n *ComplexSassNumber) AssertNoUnits(name *string) *sasscommon.SassScriptException {
	return numAssertNoUnits(n, n.HasUnits(), name)
}

// --- Compatible ---

// HasCompatibleUnits delegates to the shared helper: complex numbers only
// admit operands whose canonicalized unit shape matches.
func (n *ComplexSassNumber) HasCompatibleUnits(other SassNumber) bool {
	return numHasCompatibleUnits(n, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// CompatibleWithUnit always reports false: no single unit converts into a
// composed shape.
func (n *ComplexSassNumber) CompatibleWithUnit(unit string) bool {
	return numCompatibleWithUnit(n, n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// HasPossiblyCompatibleUnits panics, mirroring Dart's UnimplementedError:
// the check is well-defined in principle but fairly complex and nothing yet
// needs it, so it is deliberately left unimplemented rather than guessed.
func (n *ComplexSassNumber) HasPossiblyCompatibleUnits(other SassNumber) bool {
	panic("BUG: ComplexSassNumber.hasPossiblyCompatibleUnits is not implemented")
}

// --- Coercion/Conversion ---
//
// Complex numbers convert only into identical unit shapes; everything runs
// through the shared base implementation, which reconciles canonicalized
// unit lists or raises the consistent conversion error.

// Convert converts the value into newNumerators/newDenominators.
func (n *ComplexSassNumber) Convert(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return numConvert(n, newNumerators, newDenominators, name)
}

// Coerce converts the value, also accepting unitless sources that
// conversion would reject.
func (n *ComplexSassNumber) Coerce(newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	return numCoerce(n, newNumerators, newDenominators, name)
}

// CoerceToMatch coerces the value into other's units.
func (n *ComplexSassNumber) CoerceToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	return numCoerceToMatch(n, other, name, otherName)
}

// ConvertToMatch converts the value into other's units.
func (n *ComplexSassNumber) ConvertToMatch(other SassNumber, name, otherName *string) (SassNumber, error) {
	return numConvertToMatch(n, other, name, otherName)
}

// CoerceValueToMatch returns the raw value expressed in other's units.
func (n *ComplexSassNumber) CoerceValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	return numCoerceValueToMatch(n, other, name, otherName)
}

// ConvertValueToMatch is the conversion-only variant of CoerceValueToMatch.
func (n *ComplexSassNumber) ConvertValueToMatch(other SassNumber, name, otherName *string) (float64, error) {
	return numConvertValueToMatch(n, other, name, otherName)
}

// CoerceValue is the raw-value variant of Coerce.
func (n *ComplexSassNumber) CoerceValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	return numCoerceValue(n, newNumerators, newDenominators, name)
}

// ConvertValue is the raw-value variant of Convert.
func (n *ComplexSassNumber) ConvertValue(newNumerators, newDenominators []string, name *string) (float64, error) {
	return numConvertValue(n, newNumerators, newDenominators, name)
}

// ConvertValueToUnit converts the value into a single unit.
func (n *ComplexSassNumber) ConvertValueToUnit(unit string, name *string) (float64, error) {
	return numConvertValueToUnit(n, unit, name)
}

// CoerceValueToUnit coerces the value into a single unit.
func (n *ComplexSassNumber) CoerceValueToUnit(unit string, name *string) (float64, error) {
	return numCoerceValueToUnit(n, unit, name)
}

// --- WithValue/WithSlash ---
//
// Copy constructors that preserve the composed units (Dart's withValue/
// withSlash), re-selecting the subclass through the shared base.

// WithValue returns a copy with value v, keeping this number's units.
func (n *ComplexSassNumber) WithValue(v float64) SassNumber { return n.base.sharedWithValue(v) }

// WithSlash returns a copy tagged with the slash-separated operands.
func (n *ComplexSassNumber) WithSlash(num, den SassNumber) SassNumber {
	return n.base.sharedWithSlash(num, den)
}

// WithUnits returns a copy with replaced units, re-selecting whichever
// subclass fits the new shape.
func (n *ComplexSassNumber) WithUnits(numUnits, denUnits []string) SassNumber {
	return n.base.sharedWithUnits(numUnits, denUnits)
}

// WithoutSlash returns a copy with the slash marker cleared.
func (n *ComplexSassNumber) WithoutSlash() SassNumber { return n.base.sharedWithoutSlash(n) }

// UnitSuggestion renders the "expected a <type> unit" hint for error messages.
func (n *ComplexSassNumber) UnitSuggestion(name string, unit *string) string {
	return numUnitSuggestion(name, n.base.numeratorUnits, n.base.denominatorUnits, unit)
}

// --- Equals / HashCode ---

// SassNumberEquals reports value equality over canonicalized unit shapes
// with fuzzy comparison.
func (n *ComplexSassNumber) SassNumberEquals(other SassNumber) bool {
	return numEquals(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, other)
}

// HashCode hashes the value rescaled by the composed canonical multipliers,
// so equal complex numbers hash equal.
func (n *ComplexSassNumber) HashCode() int { return n.base.sharedHashCode() }

// --- Arithmetic ---
//
// Addition, subtraction, modulo, and comparisons coerce the operand into
// this shape first (shared numCoerceUnits). Multiplication and division by a
// unitless scalar preserve the shape; unit-bearing operands compose units
// through multiplyUnits.

// plusNum adds other (coerced to these units) and keeps these units.
func (n *ComplexSassNumber) plusNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, func(a, b float64) float64 { return a + b })
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// minusNum subtracts other (coerced to these units) and keeps these units.
func (n *ComplexSassNumber) minusNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, func(a, b float64) float64 { return a - b })
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// timesNum scales by a unitless operand, or composes units with a
// unit-bearing one via multiplyUnits.
func (n *ComplexSassNumber) timesNum(other SassNumber) (SassNumber, error) {
	if !other.HasUnits() {
		return n.WithValue(n.base.value * other.NumValue()), nil
	}
	return n.multiplyUnits(n.base.value*other.NumValue(), other.NumNumeratorUnits(), other.NumDenominatorUnits()), nil
}

// dividedByNum scales by a unitless divisor, or composes inverted units with
// a unit-bearing one (the divisor's numerators become denominators).
func (n *ComplexSassNumber) dividedByNum(other SassNumber) (SassNumber, error) {
	if !other.HasUnits() {
		return n.WithValue(n.base.value / other.NumValue()), nil
	}
	return n.multiplyUnits(n.base.value/other.NumValue(), other.NumDenominatorUnits(), other.NumNumeratorUnits()), nil
}

// unaryMinusNum negates the value, preserving the units.
func (n *ComplexSassNumber) unaryMinusNum() (SassNumber, error) {
	return n.WithValue(-n.base.value), nil
}

// moduloNum applies floored-division modulo in these units.
func (n *ComplexSassNumber) moduloNum(other SassNumber) (SassNumber, error) {
	val, err := numCoerceUnits(n, other, util.ModuloLikeSass)
	if err != nil {
		return nil, err
	}
	return n.WithValue(val), nil
}

// Comparisons coerce the operand into these units and use fuzzy comparison
// so values equal up to the 11th decimal compare as ordered, not unequal.
func (n *ComplexSassNumber) greaterThanNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyGreaterThan)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *ComplexSassNumber) greaterThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyGreaterThanOrEquals)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *ComplexSassNumber) lessThanNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyLessThan)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

func (n *ComplexSassNumber) lessThanOrEqualNum(other SassNumber) (*SassBoolean, error) {
	result, err := numCoerceUnits(n, other, util.FuzzyLessThanOrEquals)
	if err != nil {
		return nil, err
	}
	if result {
		return SassTrue, nil
	}
	return SassFalse, nil
}

// isComparableTo reports whether comparisons against other are meaningful,
// probing the shared greaterThan path (unitless sources always qualify).
func (n *ComplexSassNumber) isComparableTo(other SassNumber) bool {
	return numIsComparableTo(n, n.HasUnits(), other)
}

// multiplyUnits composes other's units onto this shape, cancelling
// convertible numerator/denominator pairs through the shared construction.
func (n *ComplexSassNumber) multiplyUnits(value float64, otherNumerators, otherDenominators []string) SassNumber {
	return numMultiplyUnits(value, n.base.numeratorUnits, n.base.denominatorUnits, otherNumerators, otherDenominators)
}

// coerceOrConvertValue is the shared conversion workhorse behind the
// Coerce*/Convert* family above.
func (n *ComplexSassNumber) coerceOrConvertValue(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) (float64, error) {
	return numCoerceOrConvertValue(n, n.base.value, n.base.numeratorUnits, n.base.denominatorUnits, newNumerators, newDenominators, coerceUnitless, name, other, otherName)
}

// compatibilityError builds the "incompatible units" script error with the
// unit-type suggestion text.
func (n *ComplexSassNumber) compatibilityError(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) error {
	return numCompatibilityError(n, newNumerators, newDenominators, name, other, otherName)
}
