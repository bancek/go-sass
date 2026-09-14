// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/number.dart (conversion tables, canonicalization, unitString) + lib/src/value/number/single_unit.dart (_knownCompatibilities) + lib/src/util/number.dart (abs section)

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sassmath"
)

// A nested map containing unit conversion rates.
// `1unit1 * _conversions[unit2][unit1] = 1unit2`.
// Inner maps are ordered to match Dart's LinkedHashMap iteration order.
//
// This ports Dart's _conversions table from value/number.dart verbatim in
// content: five convertible families (length, angle, time, frequency, pixel
// density) plus identity entries so same-unit lookups succeed uniformly.
// Units outside every family (em, %, ... ) have no row and are never
// convertible — only possibly-compatible via the table below.
var conversionFactors = map[string]*orderedmap.LinkedMap[string, float64]{
	"in":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "cm", Val: 1 / 2.54}, orderedmap.Pair[string, float64]{Key: "pc", Val: 1 / 6.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 1 / 25.4}, orderedmap.Pair[string, float64]{Key: "q", Val: 1 / 101.6}, orderedmap.Pair[string, float64]{Key: "pt", Val: 1 / 72.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 1 / 96.0}),
	"cm":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 2.54}, orderedmap.Pair[string, float64]{Key: "cm", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "pc", Val: 2.54 / 6.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 1 / 10.0}, orderedmap.Pair[string, float64]{Key: "q", Val: 1 / 40.0}, orderedmap.Pair[string, float64]{Key: "pt", Val: 2.54 / 72.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 2.54 / 96.0}),
	"pc":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 6.0}, orderedmap.Pair[string, float64]{Key: "cm", Val: 6 / 2.54}, orderedmap.Pair[string, float64]{Key: "pc", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 6 / 25.4}, orderedmap.Pair[string, float64]{Key: "q", Val: 6 / 101.6}, orderedmap.Pair[string, float64]{Key: "pt", Val: 1 / 12.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 1 / 16.0}),
	"mm":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 25.4}, orderedmap.Pair[string, float64]{Key: "cm", Val: 10.0}, orderedmap.Pair[string, float64]{Key: "pc", Val: 25.4 / 6.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "q", Val: 1 / 4.0}, orderedmap.Pair[string, float64]{Key: "pt", Val: 25.4 / 72.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 25.4 / 96.0}),
	"q":    orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 101.6}, orderedmap.Pair[string, float64]{Key: "cm", Val: 40.0}, orderedmap.Pair[string, float64]{Key: "pc", Val: 101.6 / 6.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 4.0}, orderedmap.Pair[string, float64]{Key: "q", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "pt", Val: 101.6 / 72.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 101.6 / 96.0}),
	"pt":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 72.0}, orderedmap.Pair[string, float64]{Key: "cm", Val: 72 / 2.54}, orderedmap.Pair[string, float64]{Key: "pc", Val: 12.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 72 / 25.4}, orderedmap.Pair[string, float64]{Key: "q", Val: 72 / 101.6}, orderedmap.Pair[string, float64]{Key: "pt", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 3 / 4.0}),
	"px":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "in", Val: 96.0}, orderedmap.Pair[string, float64]{Key: "cm", Val: 96 / 2.54}, orderedmap.Pair[string, float64]{Key: "pc", Val: 16.0}, orderedmap.Pair[string, float64]{Key: "mm", Val: 96 / 25.4}, orderedmap.Pair[string, float64]{Key: "q", Val: 96 / 101.6}, orderedmap.Pair[string, float64]{Key: "pt", Val: 4 / 3.0}, orderedmap.Pair[string, float64]{Key: "px", Val: 1.0}),
	"deg":  orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "deg", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "grad", Val: 9 / 10.0}, orderedmap.Pair[string, float64]{Key: "rad", Val: 180 / math.Pi}, orderedmap.Pair[string, float64]{Key: "turn", Val: 360.0}),
	"grad": orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "deg", Val: 10 / 9.0}, orderedmap.Pair[string, float64]{Key: "grad", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "rad", Val: 200 / math.Pi}, orderedmap.Pair[string, float64]{Key: "turn", Val: 400.0}),
	"rad":  orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "deg", Val: math.Pi / 180}, orderedmap.Pair[string, float64]{Key: "grad", Val: math.Pi / 200}, orderedmap.Pair[string, float64]{Key: "rad", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "turn", Val: 2 * math.Pi}),
	"turn": orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "deg", Val: 1 / 360.0}, orderedmap.Pair[string, float64]{Key: "grad", Val: 1 / 400.0}, orderedmap.Pair[string, float64]{Key: "rad", Val: 1 / (2 * math.Pi)}, orderedmap.Pair[string, float64]{Key: "turn", Val: 1.0}),
	"s":    orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "s", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "ms", Val: 1 / 1000.0}),
	"ms":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "s", Val: 1000.0}, orderedmap.Pair[string, float64]{Key: "ms", Val: 1.0}),
	"Hz":   orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "Hz", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "kHz", Val: 1000.0}),
	"kHz":  orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "Hz", Val: 1 / 1000.0}, orderedmap.Pair[string, float64]{Key: "kHz", Val: 1.0}),
	"dpi":  orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "dpi", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "dpcm", Val: 2.54}, orderedmap.Pair[string, float64]{Key: "dppx", Val: 96.0}),
	"dpcm": orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "dpi", Val: 1 / 2.54}, orderedmap.Pair[string, float64]{Key: "dpcm", Val: 1.0}, orderedmap.Pair[string, float64]{Key: "dppx", Val: 96 / 2.54}),
	"dppx": orderedmap.NewFromPairs(orderedmap.Pair[string, float64]{Key: "dpi", Val: 1 / 96.0}, orderedmap.Pair[string, float64]{Key: "dpcm", Val: 2.54 / 96.0}, orderedmap.Pair[string, float64]{Key: "dppx", Val: 1.0}),
}

// Sets of units that are known to be compatible with one another in the browser.
//
// These units are likewise known to be *incompatible* with units in other sets.
//
// Ported from Dart's _knownCompatibilities in value/number/single_unit.dart:
// the first set deliberately groups the convertible absolute lengths together
// with the font/viewport/container-relative units, because a browser may
// resolve those against each other at layout time even though Sass cannot
// convert them. Frequency and resolution entries are lowercased here because
// lookups normalize case before consulting this table.
//
// Matches Dart: _knownCompatibilities
var knownCompatibilities = []map[string]struct{}{
	{"em": {}, "rem": {}, "ex": {}, "rex": {}, "cap": {}, "rcap": {}, "ch": {}, "rch": {}, "ic": {}, "ric": {}, "lh": {},
		"rlh": {}, "vw": {}, "lvw": {}, "svw": {}, "dvw": {}, "vh": {}, "lvh": {}, "svh": {}, "dvh": {}, "vi": {}, "lvi": {},
		"svi": {}, "dvi": {}, "vb": {}, "lvb": {}, "svb": {}, "dvb": {}, "vmin": {}, "lvmin": {}, "svmin": {},
		"dvmin": {}, "vmax": {}, "lvmax": {}, "svmax": {}, "dvmax": {}, "cqw": {}, "cqh": {}, "cqi": {}, "cqb": {},
		"cqmin": {}, "cqmax": {}, "cm": {}, "mm": {}, "q": {}, "in": {}, "pt": {}, "pc": {}, "px": {},
	},
	{"deg": {}, "grad": {}, "rad": {}, "turn": {}},
	{"s": {}, "ms": {}},
	{"hz": {}, "khz": {}},
	{"dpi": {}, "dpcm": {}, "dppx": {}},
}

// A map from units to the set of other units they're known to be compatible with.
//
// Inverted index over knownCompatibilities so single-unit
// possibly-compatible checks are a single lookup. Units absent from every
// set map to nothing, which callers treat as "unknown, therefore possibly
// compatible with anything".
//
// Matches Dart: _knownCompatibilitiesByUnit
var knownCompatibilitiesByUnit = func() map[string]map[string]struct{} {
	m := make(map[string]map[string]struct{})
	for _, set := range knownCompatibilities {
		for unit := range set {
			m[unit] = set
		}
	}
	return m
}()

// A map from human-readable names of unit types to the convertible units that
// fall into those types.
//
// Ports Dart's _unitsByType: the first entry of each list is the canonical
// representative that canonicalizeUnitList normalizes its whole family to.
var unitsByType = map[string][]string{
	"length":        {"in", "cm", "pc", "mm", "q", "pt", "px"},
	"angle":         {"deg", "grad", "rad", "turn"},
	"time":          {"s", "ms"},
	"frequency":     {"Hz", "kHz"},
	"pixel density": {"dpi", "dpcm", "dppx"},
}

// A map from units to the human-readable names of those unit types.
//
// Inverse of unitsByType; drives both canonicalization and the "did you
// mean a <type> unit?" suggestion text in conversion error messages. Units
// with no entry (%, em, unknown) have no type and are left untouched by
// canonicalization.
var typesByUnit = func() map[string]string {
	m := make(map[string]string)
	for typ, units := range unitsByType {
		for _, unit := range units {
			m[unit] = typ
		}
	}
	return m
}()

// conversionFactor returns the number of from units per to unit.
// Equivalently, 1to * conversionFactor(from, to) = 1from.
//
// A same-unit pair short-circuits to 1 without consulting the table; anything
// involving an unconvertible unit reports false so callers fall through to
// the compatibility-error path.
func conversionFactor(from, to string) (float64, bool) {
	if from == to {
		return 1, true
	}
	m, ok := conversionFactors[from]
	if !ok {
		return 0, false
	}
	return m.Get(to)
}

// canonicalMultiplierForUnit returns a multiplier that encapsulates unit
// equivalence. That is, if X unit1 == Y unit2, X * canonicalMultiplierForUnit(unit1)
// == Y * canonicalMultiplierForUnit(unit2).
//
// The multiplier is the inverse of the unit's first table row entry, which
// rescales every member of a convertible family onto a shared axis. Unknown
// units map to 1 (identity). Equality hashes multiply the raw value by this
// so that numbers which compare equal under conversion also hash equal.
func canonicalMultiplierForUnit(unit string) float64 {
	m, ok := conversionFactors[unit]
	if !ok {
		return 1
	}
	for _, val := range m.Entries() {
		return 1 / val
	}
	return 1
}

// canonicalMultiplierForList returns a multiplier that encapsulates unit
// equivalence for a list of units.
//
// Numerator and denominator multipliers compose multiplicatively, mirroring
// how multiplyUnits accumulates converted values across composed unit lists.
func canonicalMultiplierForList(units []string) float64 {
	m := 1.0
	for _, u := range units {
		m *= canonicalMultiplierForUnit(u)
	}
	return m
}

// canonicalizeUnitList converts a unit list into an equivalent list in a
// canonical form, to make it easier to check whether two numbers have
// compatible units.
//
// Each convertible unit is replaced by its family's first representative
// (in → in, cm → in, deg → deg, ...) and the result is sorted, so
// differently-ordered spellings of the same unit shape compare equal.
// Unknown units pass through unchanged and therefore only match themselves.
func canonicalizeUnitList(units []string) []string {
	if len(units) == 0 {
		return units
	}
	result := make([]string, len(units))
	for i, u := range units {
		typ := typesByUnit[u]
		if typ != "" {
			result[i] = unitsByType[typ][0]
		} else {
			result[i] = u
		}
	}
	sort.Strings(result)
	return result
}

// unitsAreConvertible reports whether any unit in units1 converts into any
// unit in units2 via the conversion table.
//
// Table-less units only match an identical unit on the other side; this is
// the lenient pre-check behind coercion of composed unit lists, where a
// single shared dimension is enough to attempt the full conversion.
func unitsAreConvertible(units1, units2 []string) bool {
	for _, u1 := range units1 {
		m, ok := conversionFactors[u1]
		if !ok {
			if slices.Contains(units2, u1) {
				return true
			}
			continue
		}
		for _, u2 := range units2 {
			if _, ok := m.Get(u2); ok {
				return true
			}
		}
	}
	return false
}

// stringSliceEqual reports whether two unit lists hold the same units in the
// same order. Callers canonicalize (and thereby sort) both sides first, so
// order-sensitivity here is intentional: it only runs on normalized input.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// copySlice duplicates a unit list, preserving nil so callers can
// distinguish "no units" from "empty units" downstream.
func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	r := make([]string, len(s))
	copy(r, s)
	return r
}

// unitString renders numerator/denominator unit lists in Dart's unitString
// shape: bare units joined by *, a lone denominator inverted with ^-1,
// multiple denominators parenthesized before inversion, and mixed units
// joined with a slash (parenthesizing multiple denominators).
func unitString(numerators, denominators []string) string {
	switch {
	case len(numerators) == 0 && len(denominators) == 0:
		return ""
	case len(numerators) == 0 && len(denominators) == 1:
		return denominators[0] + "^-1"
	case len(numerators) == 0:
		return "(" + strings.Join(denominators, "*") + ")^-1"
	case len(denominators) == 0:
		return strings.Join(numerators, "*")
	case len(denominators) == 1:
		return strings.Join(numerators, "*") + "/" + denominators[0]
	default:
		return strings.Join(numerators, "*") + "/(" + strings.Join(denominators, "*") + ")"
	}
}

// Abs returns the absolute value of number.
//
// This is the exact spelling of Dart's util/number.dart abs: take the
// unitless absolute value, then coerce it back into the operand's units.
// Prefer AbsNumber in number_math.go when the result must preserve complex
// units without consulting the coercion machinery.
func Abs(number SassNumber) (SassNumber, error) {
	return NewUnitlessNumber(sassmath.Abs(number.NumValue())).CoerceToMatch(number, nil, nil)
}
