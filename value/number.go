// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/number.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// slashedPair holds the two slash-separated numbers a SassNumber was written
// with (for example the 1 and 2 in `1/2` as a CSS slash separator). It is
// only a representation hint: arithmetic results drop it.
// Matches Dart: SassNumber.asSlash
type slashedPair struct {
	Numerator   SassNumber
	Denominator SassNumber
}

// SassNumber is the interface for SassScript numbers.
//
// Numbers can carry units. Although there is no literal syntax for it,
// numbers support numerator and denominator units (for example `miles/hour`)
// that are expected to be resolved before the value reaches CSS. The
// interface folds Dart's three subclasses (unitless, single-unit, complex):
// arithmetic dispatches at runtime on whether the operands carry units.
//
// Matches Dart: abstract SassNumber
type SassNumber interface {
	Value

	// IsInt reports whether the number is an integer under fuzzy equality.
	// Very large floats may report false even when mathematically integral,
	// where no platform holds an exact integer representation.
	// Matches Dart: SassNumber.isInt
	IsInt() bool
	// AsInt returns the value as an int64 when IsInt holds, and false
	// otherwise.
	// Matches Dart: SassNumber.asInt
	AsInt() (int64, bool)
	// AssertInt returns the value as an int64, or a SassScriptException when
	// it isn't an integer. Name is the originating argument name (without
	// the `$`) used for error reporting.
	// Matches Dart: SassNumber.assertInt
	AssertInt(name *string) (int64, error)
	// UnitString returns a human-readable rendering of the number's units,
	// or "" when unitless.
	// Matches Dart: SassNumber.unitString
	UnitString() string
	// HasUnits reports whether the number carries any units. Functions that
	// need unitless input should use AssertNoUnits; functions that need a
	// particular unit should use AssertUnit.
	// Matches Dart: SassNumber.hasUnits
	HasUnits() bool
	// HasComplexUnits reports whether the number has more than one numerator
	// unit or any denominator units: units that make it unrepresentable as a
	// CSS length.
	// Matches Dart: SassNumber.hasComplexUnits
	HasComplexUnits() bool
	// HasUnit reports whether unit is the number's only unit (as a
	// numerator).
	// Matches Dart: SassNumber.hasUnit
	HasUnit(unit string) bool
	// AssertUnit returns nil when unit is the number's only unit (as a
	// numerator), and a SassScriptException otherwise. Name is the
	// originating argument name (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.assertUnit
	AssertUnit(unit string, name *string) *sasscommon.SassScriptException
	// AssertNoUnits returns nil when the number is unitless, and a
	// SassScriptException otherwise. Name is the originating argument name
	// (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.assertNoUnits
	AssertNoUnits(name *string) *sasscommon.SassScriptException
	// ValueInRange returns the value when it lies between min and max,
	// clamped to the nearer bound under fuzzy equality, and a
	// SassScriptException otherwise. Name is the originating argument name
	// (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.valueInRange
	ValueInRange(min, max float64, name *string) (float64, *sasscommon.SassScriptException)
	// ValueInRangeWithUnit works like ValueInRange but renders the bounds
	// with the explicit unit, for call sites whose limits carry a unit the
	// number itself may not. Name is the originating argument name (without
	// the `$`) used for error reporting.
	// Matches Dart: SassNumber.valueInRangeWithUnit
	ValueInRangeWithUnit(min, max float64, name, unit string) (float64, *sasscommon.SassScriptException)
	// HasCompatibleUnits reports whether the number's units are compatible
	// with other's: same unit counts with comparable units. Unlike
	// IsComparableTo, a unitless number is only compatible with another
	// unitless number.
	// Matches Dart: SassNumber.hasCompatibleUnits
	HasCompatibleUnits(other SassNumber) bool
	// CompatibleWithUnit reports whether the number can be coerced to unit.
	// Unitless numbers are coercible to every unit.
	// Matches Dart: SassNumber.compatibleWithUnit
	CompatibleWithUnit(unit string) bool
	// HasPossiblyCompatibleUnits reports whether the number's units are
	// possibly compatible with other's under the Sass spec: unknown units
	// give the benefit of the doubt, while known units must share a
	// convertible family.
	// Matches Dart: SassNumber.hasPossiblyCompatibleUnits
	HasPossiblyCompatibleUnits(other SassNumber) bool
	// Convert returns a copy of the number in newNumerators/newDenominators
	// units, throwing when the units are incompatible or when exactly one
	// side is unitless. ConvertValue is cheaper when only the value is
	// needed. Name is the originating argument name (without the `$`) used
	// for error reporting.
	// Matches Dart: SassNumber.convert
	Convert(newNumerators, newDenominators []string, name *string) (SassNumber, error)
	// Coerce works like Convert but treats unitless numbers as convertible
	// to and from every unit without changing the value. CoerceValue is
	// cheaper when only the value is needed. Name is the originating
	// argument name (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.coerce
	Coerce(newNumerators, newDenominators []string, name *string) (SassNumber, error)
	// CoerceToMatch returns a copy of the number in other's units, with the
	// unitless-tolerant semantics of Coerce. Name and otherName are the
	// originating argument names (without the `$`) for this number and
	// other, used for error reporting.
	// Matches Dart: SassNumber.coerceToMatch
	CoerceToMatch(other SassNumber, name, otherName *string) (SassNumber, error)
	// ConvertToMatch returns a copy of the number in other's units, with the
	// strict semantics of Convert. Name and otherName are the originating
	// argument names (without the `$`) for this number and other, used for
	// error reporting.
	// Matches Dart: SassNumber.convertToMatch
	ConvertToMatch(other SassNumber, name, otherName *string) (SassNumber, error)
	// CoerceValueToMatch returns the value in other's units, with the
	// unitless-tolerant semantics of Coerce. Name and otherName are the
	// originating argument names (without the `$`) for this number and
	// other, used for error reporting.
	// Matches Dart: SassNumber.coerceValueToMatch
	CoerceValueToMatch(other SassNumber, name, otherName *string) (float64, error)
	// ConvertValueToMatch returns the value in other's units, with the
	// strict semantics of Convert. Name and otherName are the originating
	// argument names (without the `$`) for this number and other, used for
	// error reporting.
	// Matches Dart: SassNumber.convertValueToMatch
	ConvertValueToMatch(other SassNumber, name, otherName *string) (float64, error)
	// CoerceValue returns the value in newNumerators/newDenominators units,
	// with the unitless-tolerant semantics of Coerce. Name is the
	// originating argument name (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.coerceValue
	CoerceValue(newNumerators, newDenominators []string, name *string) (float64, error)
	// ConvertValue returns the value in newNumerators/newDenominators units,
	// with the strict semantics of Convert. Name is the originating argument
	// name (without the `$`) used for error reporting.
	// Matches Dart: SassNumber.convertValue
	ConvertValue(newNumerators, newDenominators []string, name *string) (float64, error)
	// ConvertValueToUnit is shorthand for ConvertValue with a single
	// numerator unit.
	// Matches Dart: SassNumber.convertValueToUnit
	ConvertValueToUnit(unit string, name *string) (float64, error)
	// CoerceValueToUnit is shorthand for CoerceValue with a single numerator
	// unit.
	// Matches Dart: SassNumber.coerceValueToUnit
	CoerceValueToUnit(unit string, name *string) (float64, error)
	// WithValue returns a number with the same units but value v.
	// Matches Dart: SassNumber.withValue
	WithValue(v float64) SassNumber
	// WithSlash returns a copy with the slash representation set to the
	// num/den pair.
	// Matches Dart: SassNumber.withSlash
	WithSlash(num, den SassNumber) SassNumber
	// WithUnits returns a copy with the given units carried over as-is (only
	// re-selecting the unitless/single/complex representation), without the
	// convertible-unit cancellation SassNumberWithUnits applies.
	WithUnits(numUnits, denUnits []string) SassNumber
	// HasSlash reports whether the number carries a slash-separated
	// representation.
	// Matches Dart: SassNumber.asSlash (presence check)
	HasSlash() bool
	// SlashPair returns the slash-separated numerator and denominator, or
	// nils when the number has no slash representation.
	// Matches Dart: SassNumber.asSlash
	SlashPair() (SassNumber, SassNumber)
	// WithoutSlash returns a copy without the slash representation set, or
	// itself when there is none.
	// Matches Dart: SassNumber.withoutSlash
	WithoutSlash() SassNumber
	// UnitSuggestion returns a Sass snippet converting a variable named name
	// holding this number into one with the given unit (or unitless when
	// unit is nil), for unit-deprecation warnings.
	// Matches Dart: SassNumber.unitSuggestion
	UnitSuggestion(name string, unit *string) string
	// NumValue returns the raw float64 value. Sass stores every number as a
	// float64 even when it represents an integer: use IsInt, AsInt, or
	// AssertInt to work with it as an integer.
	// Matches Dart: SassNumber.value
	NumValue() float64
	// NumNumeratorUnits returns the number's numerator units.
	// Matches Dart: SassNumber.numeratorUnits
	NumNumeratorUnits() []string
	// NumDenominatorUnits returns the number's denominator units.
	// Matches Dart: SassNumber.denominatorUnits
	NumDenominatorUnits() []string
	// SassNumberEquals reports Dart operator== semantics: equal unit counts,
	// canonicalized unit families, and fuzzy value equality after folding
	// canonical multipliers into the values.
	// Matches Dart: SassNumber.operator==
	SassNumberEquals(other SassNumber) bool
	// HashCode returns a hash consistent with SassNumberEquals, folding the
	// canonical unit multipliers into the fuzzy-hashed value.
	// Matches Dart: SassNumber.hashCode
	HashCode() int

	// Unexported virtuals preserving Dart's per-subclass override shape. Each
	// concrete number type implements them; the shared num* helpers below
	// dispatch through them so operator behavior stays identical across the
	// unitless/single-unit/complex representations.
	// Matches Dart: SassNumber protected members (_coerceUnits, multiplyUnits,
	// isComparableTo, and the operator overrides)
	plusNum(other SassNumber) (SassNumber, error)
	minusNum(other SassNumber) (SassNumber, error)
	timesNum(other SassNumber) (SassNumber, error)
	dividedByNum(other SassNumber) (SassNumber, error)
	moduloNum(other SassNumber) (SassNumber, error)
	unaryMinusNum() (SassNumber, error)
	greaterThanNum(other SassNumber) (*SassBoolean, error)
	greaterThanOrEqualNum(other SassNumber) (*SassBoolean, error)
	lessThanNum(other SassNumber) (*SassBoolean, error)
	lessThanOrEqualNum(other SassNumber) (*SassBoolean, error)
	// isComparableTo reports whether two numbers can be ordered: always true
	// when either side is unitless, otherwise decided by probing greaterThan.
	// Matches Dart: SassNumber.isComparableTo
	isComparableTo(other SassNumber) bool
	// multiplyUnits builds the product/quotient unit shape for times and
	// dividedBy, cancelling convertible numerator/denominator pairs.
	// Matches Dart: SassNumber.multiplyUnits
	multiplyUnits(value float64, otherNumerators, otherDenominators []string) SassNumber
	// coerceOrConvertValue is the shared engine behind Convert/Coerce and
	// their value/match variants; coerceUnitless selects the tolerant
	// reading. Name/otherName feed error reporting.
	// Matches Dart: SassNumber._coerceOrConvertValue
	coerceOrConvertValue(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) (float64, error)
	// compatibilityError builds the incompatible-units error for
	// coerceOrConvertValue, naming both operands when other is known.
	// Matches Dart: SassNumber._coerceOrConvertValue compatibilityException closure
	compatibilityError(newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) error
}

// sassNumberBase holds the fields shared by all three number
// representations: the float64 value, the unit lists, an optional
// slash-separated form, and a lazily computed hash.
type sassNumberBase struct {
	value            float64
	numeratorUnits   []string
	denominatorUnits []string
	asSlash          *slashedPair
	hashCache        *int
}

// --- Shared implementations (not on interface, called by concrete types) ---
// One copy of each behavior lives here and runs against the caller's base,
// so the three representations can't drift apart. Concrete types forward
// their Value and SassNumber methods to these with their own base.

func (n *sassNumberBase) sharedIsInt() bool                   { return util.FuzzyIsInt(n.value) }
func (n *sassNumberBase) sharedAsInt() (int64, bool)          { return util.FuzzyAsInt(n.value) }
func (n *sassNumberBase) sharedNumValue() float64             { return n.value }
func (n *sassNumberBase) sharedNumNumeratorUnits() []string   { return n.numeratorUnits }
func (n *sassNumberBase) sharedNumDenominatorUnits() []string { return n.denominatorUnits }
func (n *sassNumberBase) sharedHasSlash() bool                { return n.asSlash != nil }

func (n *sassNumberBase) sharedSlashPair() (SassNumber, SassNumber) {
	if n.asSlash == nil {
		return nil, nil
	}
	return n.asSlash.Numerator, n.asSlash.Denominator
}

func (n *sassNumberBase) sharedHashCode() int {
	if n.hashCache != nil {
		return *n.hashCache
	}
	multiplier := 1.0
	for _, unit := range n.numeratorUnits {
		multiplier *= canonicalMultiplierForUnit(unit)
	}
	for _, unit := range n.denominatorUnits {
		multiplier /= canonicalMultiplierForUnit(unit)
	}
	h := util.FuzzyHashCode(n.value * multiplier)
	n.hashCache = &h
	return h
}

func (n *sassNumberBase) sharedWithValue(v float64) SassNumber {
	return newSassNumberPriv(v, n.numeratorUnits, n.denominatorUnits, nil)
}

func (n *sassNumberBase) sharedWithSlash(num, den SassNumber) SassNumber {
	return newSassNumberPriv(n.value, n.numeratorUnits, n.denominatorUnits, &slashedPair{num, den})
}

func (n *sassNumberBase) sharedWithUnits(numUnits, denUnits []string) SassNumber {
	return newSassNumberPriv(n.value, numUnits, denUnits, n.asSlash)
}

func (n *sassNumberBase) sharedWithoutSlash(self SassNumber) SassNumber {
	if n.asSlash == nil {
		return self
	}
	return self.WithValue(n.value)
}

// --- UnitString helpers ---
// numUnitString renders the unit shape ("no units" lives in the shared
// unitString helper): empty for unitless, joined numerators, and
// denominator suffixes that parenthesize only when needed.

func numUnitString(numUnits, denUnits []string) string {
	if len(numUnits) == 0 && len(denUnits) == 0 {
		return ""
	}
	return unitString(numUnits, denUnits)
}

func numHasUnit(numUnits, denUnits []string, unit string) bool {
	return len(numUnits) == 1 && len(denUnits) == 0 && numUnits[0] == unit
}

// --- Assert helpers ---
// Each helper renders the offending number once for the message, then throws
// a SassScriptException against the optional argument name.

func numAssertInt(n SassNumber, value float64, name *string) (int64, error) {
	if i, ok := util.FuzzyAsInt(value); ok {
		return i, nil
	}
	s, err := n.String()
	if err != nil {
		return 0, err
	}
	return 0, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not an int.", s), name)
}

func numValueInRange(n SassNumber, value float64, numUnits, denUnits []string, min, max float64, name *string) (float64, *sasscommon.SassScriptException) {
	if r, ok := util.FuzzyCheckRange(value, min, max); ok {
		return r, nil
	}
	unitStrStr := numUnitString(numUnits, denUnits)
	s, _ := n.String()
	return 0, sasscommon.NewSassScriptException(
		fmt.Sprintf("Expected %s to be within %s%s and %s%s.", s, fmt.Sprint(min), unitStrStr, fmt.Sprint(max), unitStrStr),
		name,
	)
}

func numValueInRangeWithUnit(n SassNumber, value float64, min, max float64, name, unit string) (float64, *sasscommon.SassScriptException) {
	if r, ok := util.FuzzyCheckRange(value, min, max); ok {
		return r, nil
	}
	s, _ := n.String()
	return 0, sasscommon.NewSassScriptException(
		fmt.Sprintf("Expected %s to be within %s%s and %s%s.", s, fmt.Sprint(min), unit, fmt.Sprint(max), unit),
		new(name),
	)
}

func numAssertUnit(n SassNumber, numUnits, denUnits []string, unit string, name *string) *sasscommon.SassScriptException {
	if numHasUnit(numUnits, denUnits, unit) {
		return nil
	}
	s, _ := n.String()
	return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have unit \"%s\".", s, unit), name)
}

func numAssertNoUnits(n SassNumber, hasUnits bool, name *string) *sasscommon.SassScriptException {
	if !hasUnits {
		return nil
	}
	s, _ := n.String()
	return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have no units.", s), name)
}

// --- Comparable/compatible ---
// Compatibility is structural (unit counts plus comparability) except for
// the possibly-compatible check, which gives unknown units the benefit of
// the doubt per the Sass spec: a unit outside every known family is assumed
// convertible, while two known units must share a family.

func numHasCompatibleUnits(n SassNumber, numUnits, denUnits []string, other SassNumber) bool {
	if len(numUnits) != len(other.NumNumeratorUnits()) {
		return false
	}
	if len(denUnits) != len(other.NumDenominatorUnits()) {
		return false
	}
	return n.isComparableTo(other)
}

func numCompatibleWithUnit(n SassNumber, numUnits, denUnits []string, unit string) bool {
	if !n.HasUnits() {
		return true
	}
	if n.HasComplexUnits() {
		return false
	}
	_, ok := conversionFactor(unit, numUnits[0])
	return ok
}

func numHasPossiblyCompatibleUnits(n SassNumber, numUnits, denUnits []string, other SassNumber) bool {
	if !n.HasUnits() {
		return !other.HasUnits()
	}
	if !other.HasUnits() {
		return false
	}
	if n.HasComplexUnits() || other.HasComplexUnits() {
		return n.isComparableTo(other)
	}
	myUnit := strings.ToLower(numUnits[0])
	otherUnit := strings.ToLower(other.NumNumeratorUnits()[0])
	mySet, ok := knownCompatibilitiesByUnit[myUnit]
	if !ok {
		return true
	}
	_, otherKnown := knownCompatibilitiesByUnit[otherUnit]
	_, inSet := mySet[otherUnit]
	return inSet || !otherKnown
}

// numUnitSuggestion builds the `$name * 1den / 1num ...` fix-it snippet,
// wrapped in calc() whenever numerators are present so the suggestion parses
// as a single expression.
func numUnitSuggestion(name string, numUnits, denUnits []string, unit *string) string {
	var result strings.Builder
	result.WriteString("$")
	result.WriteString(name)
	for _, den := range denUnits {
		result.WriteString(" * 1")
		result.WriteString(den)
	}
	for _, num := range numUnits {
		result.WriteString(" / 1")
		result.WriteString(num)
	}
	if unit != nil && *unit != "" {
		result.WriteString(" * 1")
		result.WriteString(*unit)
	}
	if len(numUnits) > 0 {
		return "calc(" + result.String() + ")"
	}
	return result.String()
}

// --- Coerce/Convert delegators ---
// The number-returning forms are thin shells: convert the value, then wrap
// it back up in the target units via SassNumberWithUnits.

func numConvert(n SassNumber, newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	val, err := n.ConvertValue(newNumerators, newDenominators, name)
	if err != nil {
		return nil, err
	}
	return SassNumberWithUnits(val, newNumerators, newDenominators), nil
}

func numCoerce(n SassNumber, newNumerators, newDenominators []string, name *string) (SassNumber, error) {
	val, err := n.CoerceValue(newNumerators, newDenominators, name)
	if err != nil {
		return nil, err
	}
	return SassNumberWithUnits(val, newNumerators, newDenominators), nil
}

func numCoerceToMatch(n SassNumber, other SassNumber, name, otherName *string) (SassNumber, error) {
	var n2, oName string
	if name != nil {
		n2 = *name
	}
	if otherName != nil {
		oName = *otherName
	}
	val, err := n.coerceOrConvertValue(other.NumNumeratorUnits(), other.NumDenominatorUnits(), true, n2, other, oName)
	if err != nil {
		return nil, err
	}
	return SassNumberWithUnits(val, other.NumNumeratorUnits(), other.NumDenominatorUnits()), nil
}

func numConvertToMatch(n SassNumber, other SassNumber, name, otherName *string) (SassNumber, error) {
	var n2, oName string
	if name != nil {
		n2 = *name
	}
	if otherName != nil {
		oName = *otherName
	}
	val, err := n.coerceOrConvertValue(other.NumNumeratorUnits(), other.NumDenominatorUnits(), false, n2, other, oName)
	if err != nil {
		return nil, err
	}
	return SassNumberWithUnits(val, other.NumNumeratorUnits(), other.NumDenominatorUnits()), nil
}

func numCoerceValueToMatch(n SassNumber, other SassNumber, name, otherName *string) (float64, error) {
	var n2, oName string
	if name != nil {
		n2 = *name
	}
	if otherName != nil {
		oName = *otherName
	}
	return n.coerceOrConvertValue(other.NumNumeratorUnits(), other.NumDenominatorUnits(), true, n2, other, oName)
}

func numConvertValueToMatch(n SassNumber, other SassNumber, name, otherName *string) (float64, error) {
	var n2, oName string
	if name != nil {
		n2 = *name
	}
	if otherName != nil {
		oName = *otherName
	}
	return n.coerceOrConvertValue(other.NumNumeratorUnits(), other.NumDenominatorUnits(), false, n2, other, oName)
}

func numCoerceValue(n SassNumber, newNumerators, newDenominators []string, name *string) (float64, error) {
	var n2 string
	if name != nil {
		n2 = *name
	}
	return n.coerceOrConvertValue(newNumerators, newDenominators, true, n2, nil, "")
}

func numConvertValue(n SassNumber, newNumerators, newDenominators []string, name *string) (float64, error) {
	var n2 string
	if name != nil {
		n2 = *name
	}
	return n.coerceOrConvertValue(newNumerators, newDenominators, false, n2, nil, "")
}

func numConvertValueToUnit(n SassNumber, unit string, name *string) (float64, error) {
	return n.ConvertValue([]string{unit}, nil, name)
}

func numCoerceValueToUnit(n SassNumber, unit string, name *string) (float64, error) {
	return n.CoerceValue([]string{unit}, nil, name)
}

// --- Core coerceOrConvertValue algorithm (shared) ---
// Conversion matches each requested unit against one stored unit, folding
// the conversion factor into the value as it goes. Anything left unmatched
// on either side means the units are incompatible.

func numCoerceOrConvertValue(n SassNumber, val float64, numUnits, denUnits []string, newNumerators, newDenominators []string, coerceUnitless bool, name string, other SassNumber, otherName string) (float64, error) {
	// Identical unit shapes convert to themselves with no arithmetic.
	if stringSliceEqual(numUnits, newNumerators) && stringSliceEqual(denUnits, newDenominators) {
		return val, nil
	}
	otherHasUnits := len(newNumerators) > 0 || len(newDenominators) > 0
	// Coercion (unlike conversion) lets unitless pass through untouched in
	// either direction.
	if coerceUnitless && (!n.HasUnits() || !otherHasUnits) {
		return val, nil
	}
	compatErr := numCompatibilityError(n, newNumerators, newDenominators, name, other, otherName)
	v := val
	oldNumerators := copySlice(numUnits)
	for _, newNum := range newNumerators {
		var found bool
		for i, oldNum := range oldNumerators {
			factor, ok := conversionFactor(newNum, oldNum)
			if !ok {
				continue
			}
			v *= factor
			oldNumerators = append(oldNumerators[:i], oldNumerators[i+1:]...)
			found = true
			break
		}
		if !found {
			return 0, compatErr
		}
	}
	oldDenominators := copySlice(denUnits)
	for _, newDen := range newDenominators {
		var found bool
		for i, oldDen := range oldDenominators {
			factor, ok := conversionFactor(newDen, oldDen)
			if !ok {
				continue
			}
			v /= factor
			oldDenominators = append(oldDenominators[:i], oldDenominators[i+1:]...)
			found = true
			break
		}
		if !found {
			return 0, compatErr
		}
	}
	if len(oldNumerators) > 0 || len(oldDenominators) > 0 {
		return 0, compatErr
	}
	return v, nil
}

func numCompatibilityError(n SassNumber, newNumerators, newDenominators []string, name string, other SassNumber, otherName string) error {
	// Built lazily inside the conversion above so the message renders the
	// operands only on the failure path.
	var p *string
	if name != "" {
		p = &name
	}
	otherHasUnits := len(newNumerators) > 0 || len(newDenominators) > 0
	if other != nil {
		nStr, err := n.String()
		if err != nil {
			return err
		}
		otherStr, err := other.String()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("%s and", nStr)
		if otherName != "" {
			msg += fmt.Sprintf(" $%s:", otherName)
		}
		msg += fmt.Sprintf(" %s have incompatible units", otherStr)
		if !n.HasUnits() || !otherHasUnits {
			msg += " (one has units and the other doesn't)"
		}
		return sasscommon.NewSassScriptException(fmt.Sprintf("%s.", msg), p)
	}
	if !otherHasUnits {
		nStr, err := n.String()
		if err != nil {
			return err
		}
		return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have no units.", nStr), p)
	}
	if len(newNumerators) == 1 && len(newDenominators) == 0 {
		nStr, err := n.String()
		if err != nil {
			return err
		}
		typ := typesByUnit[newNumerators[0]]
		if typ != "" {
			// Converting to a unit of a named family names the family and
			// lists every unit it converts with, so the user sees the full
			// target set (for example all length units for px).
			return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have %s unit (%s).", nStr, util.A(typ), strings.Join(unitsByType[typ], ", ")), p)
		}
	}
	nStr, err := n.String()
	if err != nil {
		return err
	}
	return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have %s %s.", nStr, util.Pluralize("unit", len(newNumerators)+len(newDenominators), nil), unitString(newNumerators, newDenominators)), p)
}

// --- multiplyUnits (shared) ---
// Unit arithmetic for times/dividedBy: pair each numerator off against a
// convertible denominator (folding the factor into the value) and keep the
// survivors. The first two branches short-circuit without allocating new
// unit lists when one side is unitless or nothing can cancel.

func numMultiplyUnits(n float64, numUnits, denUnits []string, otherNumerators, otherDenominators []string) SassNumber {
	if len(otherNumerators) == 0 && len(otherDenominators) == 0 {
		return SassNumberWithUnits(n, copySlice(numUnits), copySlice(denUnits))
	}
	if len(numUnits) == 0 && len(denUnits) == 0 {
		return SassNumberWithUnits(n, copySlice(otherNumerators), copySlice(otherDenominators))
	}
	if len(numUnits) == 0 && len(otherDenominators) == 0 && !unitsAreConvertible(denUnits, otherNumerators) {
		return SassNumberWithUnits(n, copySlice(otherNumerators), copySlice(denUnits))
	}
	if len(denUnits) == 0 && len(otherNumerators) == 0 && !unitsAreConvertible(numUnits, otherDenominators) {
		return SassNumberWithUnits(n, copySlice(numUnits), copySlice(otherDenominators))
	}
	val := n
	mutableOtherDen := copySlice(otherDenominators)
	var newNumerators []string
	for _, num := range numUnits {
		found := false
		for i, den := range mutableOtherDen {
			factor, ok := conversionFactor(num, den)
			if !ok {
				continue
			}
			val /= factor
			mutableOtherDen = append(mutableOtherDen[:i], mutableOtherDen[i+1:]...)
			found = true
			break
		}
		if !found {
			newNumerators = append(newNumerators, num)
		}
	}
	mutableDen := copySlice(denUnits)
	for _, num := range otherNumerators {
		found := false
		for i, den := range mutableDen {
			factor, ok := conversionFactor(num, den)
			if !ok {
				continue
			}
			val /= factor
			mutableDen = append(mutableDen[:i], mutableDen[i+1:]...)
			found = true
			break
		}
		if !found {
			newNumerators = append(newNumerators, num)
		}
	}
	return SassNumberWithUnits(val, newNumerators, append(mutableDen, mutableOtherDen...))
}

// --- isComparableTo (shared) ---
// Comparability is probed, not predicted: ordering succeeds exactly when the
// units coerce, so greaterThan doubles as the compatibility test.

func numIsComparableTo(n SassNumber, hasUnits bool, other SassNumber) bool {
	if !hasUnits || !other.HasUnits() {
		return true
	}
	_, err := n.greaterThanNum(other)
	return err == nil
}

// --- Equals (shared) ---
// Equality canonicalizes both unit lists (each known family collapses to its
// first unit, multi-unit lists sort) and compares the values after folding
// the canonical multipliers in, all under fuzzy equality.

func numEquals(n SassNumber, val float64, numUnits, denUnits []string, other SassNumber) bool {
	if len(numUnits) != len(other.NumNumeratorUnits()) || len(denUnits) != len(other.NumDenominatorUnits()) {
		return false
	}
	if !n.HasUnits() {
		return util.FuzzyEquals(val, other.NumValue())
	}
	canonNum := canonicalizeUnitList(numUnits)
	otherCanonNum := canonicalizeUnitList(other.NumNumeratorUnits())
	canonDen := canonicalizeUnitList(denUnits)
	otherCanonDen := canonicalizeUnitList(other.NumDenominatorUnits())
	if !stringSliceEqual(canonNum, otherCanonNum) || !stringSliceEqual(canonDen, otherCanonDen) {
		return false
	}
	return util.FuzzyEquals(
		val*canonicalMultiplierForList(numUnits)/canonicalMultiplierForList(denUnits),
		other.NumValue()*canonicalMultiplierForList(other.NumNumeratorUnits())/canonicalMultiplierForList(other.NumDenominatorUnits()),
	)
}

// --- Value interface operator methods (shared, dispatched to unexported virtual methods) ---
// Each operator takes the fast path when the operand is a number (coercing
// units first) and otherwise mirrors Dart: number/color arithmetic is an
// Undefined operation error, while non-number operands fall back to the
// string-concatenation defaults.

func numEqual(n SassNumber, other Value) bool {
	if on, ok := other.(SassNumber); ok {
		return n.SassNumberEquals(on)
	}
	return false
}

func numSingleEquals(n SassNumber, other Value) (Value, error) {
	s, err := n.ToCssString(true)
	if err != nil {
		return nil, err
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: s + "=" + right, HasQuotes: false}, nil
}

func numPlus(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.plusNum(on)
	}
	if _, ok := other.(*SassColor); ok {
		s, _ := n.String()
		otherStr, _ := other.String()
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s + %s\".", s, otherStr), nil)
	}
	return DefaultPlus(n, other)
}

func numMinus(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.minusNum(on)
	}
	if _, ok := other.(*SassColor); ok {
		s, _ := n.String()
		otherStr, _ := other.String()
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s - %s\".", s, otherStr), nil)
	}
	return DefaultMinus(n, other)
}

func numTimes(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.timesNum(on)
	}
	return DefaultTimes(n, other)
}

func numDividedBy(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.dividedByNum(on)
	}
	return DefaultDividedBy(n, other)
}

func numModulo(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.moduloNum(on)
	}
	return DefaultModulo(n, other)
}

func numGreaterThan(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.greaterThanNum(on)
	}
	return DefaultGreaterThan(n, other)
}

func numGreaterThanOrEquals(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.greaterThanOrEqualNum(on)
	}
	return DefaultGreaterThanOrEquals(n, other)
}

func numLessThan(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.lessThanNum(on)
	}
	return DefaultLessThan(n, other)
}

func numLessThanOrEquals(n SassNumber, other Value) (Value, error) {
	if on, ok := other.(SassNumber); ok {
		return n.lessThanOrEqualNum(on)
	}
	return DefaultLessThanOrEquals(n, other)
}

func numUnaryPlus(n SassNumber) (Value, error)   { return n, nil }
func numUnaryMinus(n SassNumber) (Value, error)  { return n.unaryMinusNum() }
func numUnaryDivide(n SassNumber) (Value, error) { return DefaultUnaryDivide(n) }
func numUnaryNot(n SassNumber) (Value, error)    { return DefaultUnaryNot(n) }

func numIsTruthy() bool           { return true }
func numSeparator() ListSeparator { return ListSeparatorUndecided }
func numHasBrackets() bool        { return false }
func numLengthAsList() int        { return 1 }
func numIsBlank() bool            { return false }
func numIsSpecialNumber() bool    { return false }
func numIsSpecialVariable() bool  { return false }
func numTryMap() *SassMap         { return nil }

// --- _coerceUnits ---
// numCoerceUnits converts other's value into this number's units and applies
// the operation. When that fails it re-runs the conversion in the opposite
// direction purely for its error message, which prints this number before
// the operand and reads better; the first error is the one reported.

func numCoerceUnits[T any](n, other SassNumber, operation func(float64, float64) T) (T, error) {
	coerced, err := other.CoerceValueToMatch(n, nil, nil)
	if err == nil {
		return operation(n.NumValue(), coerced), nil
	}
	_, err2 := n.CoerceValueToMatch(other, nil, nil)
	if err2 != nil {
		var zero T
		return zero, err2
	}
	var zero T
	return zero, err
}

// --- Constructors ---
// newSassNumberPriv selects the representation by unit shape: no units is
// unitless, one numerator unit is single-unit, anything else is complex.
// Matches Dart: SassNumber factory dispatch (Unitless/SingleUnit/Complex)

func newSassNumberPriv(v float64, numUnits, denUnits []string, asSlash *slashedPair) SassNumber {
	if len(numUnits) == 0 && len(denUnits) == 0 {
		return newUnitlessNumber(v, asSlash)
	}
	if len(numUnits) == 1 && len(denUnits) == 0 {
		return newSingleUnitNumber(v, numUnits[0], asSlash)
	}
	return newComplexNumber(v, numUnits, denUnits, asSlash)
}

// NewSassNumber creates a number with an optional single numerator unit,
// matching the numbers writable as literals: a nil unit builds a unitless
// number. For numerator/denominator combinations use SassNumberWithUnits.
// Matches Dart: SassNumber factory (value, [unit])
func NewSassNumber(v float64, unit *string) SassNumber {
	if unit != nil {
		return NewSingleUnitNumber(v, *unit)
	}
	return NewUnitlessNumber(v)
}

// NewUnitlessNumber creates a number with no units.
// Matches Dart: UnitlessSassNumber constructor
func NewUnitlessNumber(v float64) SassNumber {
	return newUnitlessNumber(v, nil)
}

// NewSingleUnitNumber creates a number with one numerator unit.
// Matches Dart: SingleUnitSassNumber constructor
func NewSingleUnitNumber(v float64, unit string) SassNumber {
	return newSingleUnitNumber(v, unit, nil)
}

// NewComplexNumber creates a number with the given numerator and denominator
// units, carried over as-is.
// Matches Dart: ComplexSassNumber constructor
func NewComplexNumber(v float64, numUnits, denUnits []string) SassNumber {
	return newComplexNumber(v, numUnits, denUnits, nil)
}

// SassNumberWithUnits creates a number with full numerator and denominator
// units, simplifying away any denominator convertible to a numerator: the
// conversion factor folds into the value and fully cancelled results collapse
// to unitless or single-unit.
// Matches Dart: SassNumber.withUnits factory
func SassNumberWithUnits(value float64, numeratorUnits, denominatorUnits []string) SassNumber {
	numUnits := copySlice(numeratorUnits)
	denUnits := copySlice(denominatorUnits)
	val := value
	remainingDen := copySlice(denUnits)
	denUnits = nil
	for _, den := range remainingDen {
		simplified := false
		for i := 0; i < len(numUnits); i++ {
			factor, ok := conversionFactor(den, numUnits[i])
			if !ok {
				continue
			}
			val *= factor
			numUnits = append(numUnits[:i], numUnits[i+1:]...)
			simplified = true
			break
		}
		if !simplified {
			denUnits = append(denUnits, den)
		}
	}
	if len(numUnits) == 0 && len(denUnits) == 0 {
		return newUnitlessNumber(val, nil)
	}
	if len(numUnits) == 1 && len(denUnits) == 0 {
		return newSingleUnitNumber(val, numUnits[0], nil)
	}
	return newComplexNumber(val, numUnits, denUnits, nil)
}
