// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitBoolean, visitNull,
// visitString, visitNumber, visitColor dispatch, visitCalculation,
// visitList, visitMap, visitFunction, visitMixin section)

import (
	"fmt"
	"math"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ---- ValueVisitor implementations ----

// VisitBoolean writes the bare true/false literal.
//
// Matches Dart: _SerializeVisitor.visitBoolean
func (sv *SerializeVisitor) VisitBoolean(b *SassBoolean) (struct{}, error) {
	if b.Value {
		_, _ = sv.sb.WriteString("true")
	} else {
		_, _ = sv.sb.WriteString("false")
	}
	return struct{}{}, nil
}

// VisitNull writes the null literal in inspect mode and nothing in CSS
// mode, where nulls vanish from the output.
//
// Matches Dart: _SerializeVisitor.visitNull
func (sv *SerializeVisitor) VisitNull() (struct{}, error) {
	if sv.inspect {
		_, _ = sv.sb.WriteString("null")
	}
	return struct{}{}, nil
}

// VisitString writes a string quoted or unquoted: quotes survive only
// when both the visitor's quote flag and the string itself call for them.
//
// Matches Dart: _SerializeVisitor.visitString
func (sv *SerializeVisitor) VisitString(s *SassString) (struct{}, error) {
	if sv.quote && s.HasQuotes {
		sv.visitQuotedString(s.Text)
	} else {
		sv.visitUnquotedString(s.Text)
	}
	return struct{}{}, nil
}

// VisitNumber writes a number with its single numerator unit, if any.
// Slash-separated numbers recurse with a literal slash between the sides.
// Non-finite values and complex units cannot appear as plain CSS numbers,
// so they detour through an unsimplified calc() calculation (which also
// covers numbers carrying units on infinities).
//
// Matches Dart: _SerializeVisitor.visitNumber
func (sv *SerializeVisitor) VisitNumber(value SassNumber) (struct{}, error) {
	if value.HasSlash() {
		before, after := value.SlashPair()
		if _, err := sv.VisitNumber(before); err != nil {
			return struct{}{}, err
		}
		_ = sv.sb.WriteByte('/')
		if _, err := sv.VisitNumber(after); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}

	if math.IsInf(value.NumValue(), 0) || math.IsNaN(value.NumValue()) {
		return sv.VisitCalculation(NewUnsimplified("calc", value))
	}

	if value.HasComplexUnits() {
		return sv.VisitCalculation(NewUnsimplified("calc", value))
	} else {
		sv.writeNumber(value.NumValue())
		if len(value.NumNumeratorUnits()) == 1 {
			_, _ = sv.sb.WriteString(value.NumNumeratorUnits()[0])
		}
	}
	return struct{}{}, nil
}

// VisitColor dispatches on the color's space and completeness. Fully
// specified legacy colors take the legacy decision tree (names, hex,
// rgb/hsl functions); legacy colors with missing channels render in the
// modern space-separated syntax; out-of-gamut lab-family colors in normal
// mode go through color-mix; in-gamut lab-family colors keep their own
// function; every other space uses the generic color() function. See
// visitor_color.go for each arm.
//
// Matches Dart: _SerializeVisitor.visitColor (dispatch over space)
func (sv *SerializeVisitor) VisitColor(c *SassColor) (struct{}, error) {
	if c.Space().IsLegacy() &&
		!c.IsChannel0Missing() && !c.IsChannel1Missing() && !c.IsChannel2Missing() && !c.IsAlphaMissing() {
		return struct{}{}, sv.writeLegacyColor(c)
	}

	if c.Space().IsLegacy() {
		sv.writeColorWithMissing(c)
		return struct{}{}, nil
	}

	if !sv.inspect && needsColorMix(c) {
		if err := sv.writeColorMix(c); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}

	if isLabLchOklabOklch(c.Space()) {
		sv.writeLabLchColor(c)
		return struct{}{}, nil
	}

	sv.writeColorFunction(c)
	return struct{}{}, nil
}

// needsColorMix reports whether c needs the color-mix() fallback: a
// lab/lch color outside [0, 100] lightness, an oklab/oklch color outside
// [0, 1] lightness, or a polar lch/oklch color with negative chroma, in
// each case with the channels the fallback requires present. It factors
// out the guarded case patterns heading Dart's visitColor gamut arms.
//
// Matches Dart: _SerializeVisitor.visitColor (color-mix guard clauses)
func needsColorMix(c *SassColor) bool {
	switch c.Space() {
	case LabColorSpace, LchColorSpace:
		return !util.FuzzyInRange(c.Channel0(), 0, 100) && !c.IsChannel1Missing() && !c.IsChannel2Missing()
	case OklabColorSpace, OklchColorSpace:
		return !util.FuzzyInRange(c.Channel0(), 0, 1) && !c.IsChannel1Missing() && !c.IsChannel2Missing()
	}
	if (c.Space() == LchColorSpace || c.Space() == OklchColorSpace) &&
		util.FuzzyLessThan(c.Channel1(), 0) && !c.IsChannel0Missing() && !c.IsChannel1Missing() {
		return true
	}
	return false
}

// isLabLchOklabOklch reports whether space is one of the four
// lightness-based modern spaces sharing the lab-function rendering arm.
// It is Go-only glue for the combined case pattern in Dart's visitColor.
func isLabLchOklabOklch(space ColorSpace) bool {
	return space == LabColorSpace || space == LchColorSpace ||
		space == OklabColorSpace || space == OklchColorSpace
}

// VisitCalculation writes a name(arguments) calculation with its
// arguments separated by the style's comma separator.
//
// Matches Dart: _SerializeVisitor.visitCalculation
func (sv *SerializeVisitor) VisitCalculation(c *SassCalculation) (struct{}, error) {
	_, _ = sv.sb.WriteString(c.Name)
	_ = sv.sb.WriteByte('(')
	if err := writeBetween(sv.sb, c.Arguments, sv.commaSep(), sv.writeCalculationValue); err != nil {
		return struct{}{}, err
	}
	_ = sv.sb.WriteByte(')')
	return struct{}{}, nil
}

// VisitList writes a list with its separator spelling, wrapped in
// brackets when bracketed. An empty unbracketed list is only valid in
// inspect mode (rendered as ()); in CSS mode it raises a script exception.
// CSS mode also drops blank elements. In inspect mode, a lone comma- or
// slash-separated element gains a trailing separator (and parentheses when
// unbracketed) so the output stays an unambiguous single-element list,
// and nested lists that would read ambiguously are parenthesized.
//
// Matches Dart: _SerializeVisitor.visitList
func (sv *SerializeVisitor) VisitList(l ListValue) (struct{}, error) {
	if l.HasBrackets() {
		_ = sv.sb.WriteByte('[')
	} else if l.LengthAsList() == 0 {
		if !sv.inspect {
			return struct{}{}, sasscommon.NewSassScriptException("() isn't a valid CSS value.", nil)
		}
		_, _ = sv.sb.WriteString("()")
		return struct{}{}, nil
	}

	singleton := sv.inspect &&
		l.LengthAsList() == 1 &&
		(l.Separator() == ListSeparatorComma || l.Separator() == ListSeparatorSlash)
	if singleton && !l.HasBrackets() {
		_ = sv.sb.WriteByte('(')
	}

	elems, err := l.AsList()
	if err != nil {
		return struct{}{}, err
	}
	if !sv.inspect {
		var filtered []Value
		for _, elem := range elems {
			if !elem.IsBlank() {
				filtered = append(filtered, elem)
			}
		}
		elems = filtered
	}

	sep := sv.listSepStr(l.Separator())
	var writeElem func(Value) error
	if sv.inspect {
		writeElem = func(elem Value) error {
			if elementNeedsParens(l.Separator(), elem) {
				_ = sv.sb.WriteByte('(')
			}
			if _, err := elem.AcceptVoid(sv); err != nil {
				return err
			}
			if elementNeedsParens(l.Separator(), elem) {
				_ = sv.sb.WriteByte(')')
			}
			return nil
		}
	} else {
		writeElem = func(elem Value) error {
			_, err := elem.AcceptVoid(sv)
			return err
		}
	}
	if err := writeBetween(sv.sb, elems, sep, writeElem); err != nil {
		return struct{}{}, err
	}

	if singleton {
		switch l.Separator() {
		case ListSeparatorComma:
			_, _ = sv.sb.WriteString(",")
		case ListSeparatorSlash:
			_, _ = sv.sb.WriteString("/")
		}
		if !l.HasBrackets() {
			_ = sv.sb.WriteByte(')')
		}
	}

	if l.HasBrackets() {
		_ = sv.sb.WriteByte(']')
	}
	return struct{}{}, nil
}

// VisitMap writes a map as a parenthesized key: value sequence. Maps are
// inspect-only; in CSS mode it raises a script exception naming the map.
//
// Matches Dart: _SerializeVisitor.visitMap
func (sv *SerializeVisitor) VisitMap(m *SassMap) (struct{}, error) {
	if !sv.inspect {
		mStr, err := m.String()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, sasscommon.NewSassScriptException(fmt.Sprintf("%s isn't a valid CSS value.", mStr), nil)
	}
	_ = sv.sb.WriteByte('(')
	first := true
	for key, value := range m.Entries() {
		if first {
			first = false
		} else {
			_, _ = sv.sb.WriteString(", ")
		}
		if err := sv.writeMapElement(key); err != nil {
			return struct{}{}, err
		}
		_, _ = sv.sb.WriteString(": ")
		if err := sv.writeMapElement(value); err != nil {
			return struct{}{}, err
		}
	}
	_ = sv.sb.WriteByte(')')
	return struct{}{}, nil
}

// VisitFunction writes a function value as a get-function("name") call.
// Functions are inspect-only; in CSS mode it raises a script exception
// naming the function.
//
// Matches Dart: _SerializeVisitor.visitFunction
func (sv *SerializeVisitor) VisitFunction(f *SassFunction) (struct{}, error) {
	if !sv.inspect {
		fStr, err := f.String()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, sasscommon.NewSassScriptException(fmt.Sprintf("%s isn't a valid CSS value.", fStr), nil)
	}
	_, _ = sv.sb.WriteString("get-function(")
	sv.visitQuotedString(f.Name())
	_ = sv.sb.WriteByte(')')
	return struct{}{}, nil
}

// VisitMixin writes a mixin value as a get-mixin("name") call. Mixins are
// inspect-only; in CSS mode it raises a script exception naming the mixin.
//
// Matches Dart: _SerializeVisitor.visitMixin
func (sv *SerializeVisitor) VisitMixin(m *SassMixin) (struct{}, error) {
	if !sv.inspect {
		mStr, err := m.String()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, sasscommon.NewSassScriptException(fmt.Sprintf("%s isn't a valid CSS value.", mStr), nil)
	}
	_, _ = sv.sb.WriteString("get-mixin(")
	sv.visitQuotedString(m.Name())
	_ = sv.sb.WriteByte(')')
	return struct{}{}, nil
}
