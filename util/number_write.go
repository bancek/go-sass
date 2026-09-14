// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/visitor/serialize.dart (_writeNumber,
// _removeExponent, _writeRounded section; AsIntForSerialize backs the
// _asInt integer fast path)

import (
	"strconv"
	"strings"
)

// NumberWriter is satisfied by *strings.Builder and
// *sourcemapbuffer.DefaultSourceMapBuffer.
type NumberWriter interface {
	WriteString(string) (int, error)
	WriteByte(byte) error
}

// WriteNumberToString formats v as a CSS numeric string in the default
// spelling: full shortest round-trip without exponent notation, no
// inspect full-precision bypass, no compressed leading-zero strip.
//
// Matches Dart: _SerializeVisitor._writeNumber (default flags)
func WriteNumberToString(v float64) string {
	var buf strings.Builder
	WriteNumberTo(&buf, v, false, false)
	return buf.String()
}

// WriteNumberTo writes v without exponent notation and with at most Precision
// digits after the decimal point.
//
// Matches Dart: _SerializeVisitor._writeNumber
//
// Dart uses double.toString() (Dragon4 algorithm); Go uses strconv.FormatFloat
// (Ryu algorithm). Both produce shortest round-tripping representations, but
// differ in two ways: (1) exponent-notation thresholds — Go uses e-notation at
// |v| ≥ 1e6, Dart at |v| ≥ 1e21; (2) .0 suffix — Go omits, Dart includes.
// removeExponent handles (1) by converting e-notation back to plain decimal,
// and the FuzzyAsInt fast-path / writeRoundedTo handle (2) by stripping .0.
// The remaining theoretical difference — Ryu vs Dragon4 producing different
// digit sequences for the same float64 — would only affect very large/small
// values outside typical CSS ranges. No spec test covers such edge cases.
func WriteNumberTo(buf NumberWriter, v float64, inspect, compressed bool) {
	// Since dart-sass 1.104 (#2840), negative zero serializes as `-0`
	// (Dart `_writeNumber`). This precedes the integer fast-path: `-0.0`
	// is an exact integer, but must not print as `0`.
	if IsNegativeZero(v) {
		_, _ = buf.WriteString("-0")
		return
	}

	// Integer fast-path (Dart `_asInt`): fuzzy matching in inspect mode,
	// exact integers only otherwise.
	if integer, ok := AsIntForSerialize(v, inspect); ok {
		_, _ = buf.WriteString(removeExponent(strconv.FormatInt(integer, 10)))
		return
	}
	text := removeExponent(strconv.FormatFloat(v, 'g', -1, 64))
	if inspect {
		// Inspect mode shows the full precision of every number.
		_, _ = buf.WriteString(text)
		return
	}
	// Any value shorter than precision + 2 characters holds at most "0."
	// plus precision digits, so it is safe to emit directly with no
	// rounding or truncation.
	if len(text) < Precision+2 {
		if compressed && len(text) > 0 && text[0] == '0' {
			text = text[1:]
		}
		_, _ = buf.WriteString(text)
		return
	}
	writeRoundedTo(buf, text, compressed)
}

// WriteRoundedTo rounds [text] to [Precision] digits after the decimal and
// writes the result to [buf].
//
// [text] must be a number written without exponent notation.
//
// Matches Dart: _SerializeVisitor._writeRounded
func WriteRoundedTo(buf NumberWriter, text string, compressed bool) {
	writeRoundedTo(buf, text, compressed)
}

// writeRoundedTo rounds [text] to [Precision] digits after the decimal
// and writes the result to [buf].
//
// [text] must be a number written without exponent notation.
//
// Matches Dart: _SerializeVisitor._writeRounded
func writeRoundedTo(buf NumberWriter, text string, compressed bool) {
	// Dart prints every double with a trailing ".0", even integer values,
	// so such spellings never need precision handling: drop the suffix.
	if strings.HasSuffix(text, ".0") {
		_, _ = buf.WriteString(text[:len(text)-2])
		return
	}
	neg := len(text) > 0 && text[0] == '-'

	textIdx := 0
	if neg {
		textIdx++
	}

	maxDigits := len(text) + 1
	digits := make([]int, maxDigits)
	// Start at index 1 to leave room for rounding up into a higher decimal
	// place than the input represents.
	digitsIdx := 1

	for {
		if textIdx >= len(text) {
			// No decimal point at all: nothing to round, write as-is.
			_, _ = buf.WriteString(text)
			return
		}
		ch := text[textIdx]
		textIdx++
		if ch == '.' {
			break
		}
		digits[digitsIdx] = int(ch - '0')
		digitsIdx++
	}

	firstFractionalDigit := digitsIdx
	indexAfterPrecision := textIdx + Precision
	if indexAfterPrecision >= len(text) {
		// Fewer than precision digits past the point: no rounding or
		// truncation needed, write as-is.
		_, _ = buf.WriteString(text)
		return
	}

	for textIdx < indexAfterPrecision {
		digits[digitsIdx] = int(text[textIdx] - '0')
		digitsIdx++
		textIdx++
	}

	if int(text[textIdx]-'0') >= 5 {
		// Round up from the last kept digit, rippling the carry leftwards
		// through the integer part when every trailing digit is a 9. The
		// leading spare slot guarantees the index stays positive even when
		// the whole value rounds up.
		for {
			digits[digitsIdx-1]++
			if digits[digitsIdx-1] != 10 {
				break
			}
			digitsIdx--
		}
	}

	for ; digitsIdx < firstFractionalDigit; digitsIdx++ {
		digits[digitsIdx] = 0
	}
	// Only one of the two adjustments runs: either the carry rippled past
	// the decimal point (integer digits above become zeros) or leftover
	// fractional zeros are trimmed. Either way the end index stays at or
	// past the first fractional slot.
	for digitsIdx > firstFractionalDigit && digits[digitsIdx-1] == 0 {
		digitsIdx--
	}

	if digitsIdx == 2 && digits[0] == 0 && digits[1] == 0 {
		// Rounding landed exactly on zero: drop any minus sign and write
		// an explicit 0 so compressed mode neither keeps a stray minus nor
		// omits the number with its leading zero.
		_ = buf.WriteByte('0')
		return
	}

	if neg {
		_ = buf.WriteByte('-')
	}

	writtenIdx := 0
	if digits[0] == 0 {
		// Skip the spare leading slot, and in compressed mode also the
		// zero before the decimal point (0.5 becomes .5, keeping the sign
		// so -0.5 stays -0.5).
		writtenIdx++
		if compressed && digits[1] == 0 {
			writtenIdx++
		}
	}
	for ; writtenIdx < firstFractionalDigit; writtenIdx++ {
		buf.WriteByte(byte('0' + digits[writtenIdx]))
	}

	if digitsIdx > firstFractionalDigit {
		buf.WriteByte('.')
		for ; writtenIdx < digitsIdx; writtenIdx++ {
			buf.WriteByte(byte('0' + digits[writtenIdx]))
		}
	}
}

// removeExponent returns a representation of [text] without exponent notation.
//
// If [text] doesn't use exponent notation, it's returned as-is. Positive
// exponents append zeros (or splice in a decimal point when the exponent
// lands mid-digits); negative exponents become 0.xxx with one zero per
// missing place.
//
// Matches Dart: _SerializeVisitor._removeExponent
func removeExponent(text string) string {
	eIdx := -1
	for i := 0; i < len(text); i++ {
		if text[i] == 'e' || text[i] == 'E' {
			eIdx = i
			break
		}
	}
	if eIdx == -1 {
		return text
	}

	negative := len(text) > 0 && text[0] == '-'
	var buf strings.Builder
	buf.WriteByte(text[0])
	if negative {
		if eIdx > 1 {
			buf.WriteByte(text[1])
		}
		if eIdx > 3 {
			_, _ = buf.WriteString(text[3:eIdx])
		}
	} else {
		if eIdx > 2 {
			_, _ = buf.WriteString(text[2:eIdx])
		}
	}

	exp, _ := strconv.Atoi(text[eIdx+1:])
	if exp > 0 {
		additionalZeroes := exp - (buf.Len() - 1)
		if negative {
			additionalZeroes = exp - (buf.Len() - 2)
		}
		if additionalZeroes < 0 {
			s := buf.String()
			insertPos := exp + 1
			if negative {
				insertPos = exp + 2
			}
			return s[:insertPos] + "." + s[insertPos:]
		}
		for i := 0; i < additionalZeroes; i++ {
			buf.WriteByte('0')
		}
		return buf.String()
	}

	var result strings.Builder
	if negative {
		result.WriteByte('-')
	}
	result.WriteString("0.")
	for i := -1; i > exp; i-- {
		result.WriteByte('0')
	}
	s := buf.String()
	if negative && len(s) > 1 {
		result.WriteString(s[1:])
	} else if !negative {
		result.WriteString(s)
	} else {
		result.WriteString(s)
	}
	return result.String()
}
