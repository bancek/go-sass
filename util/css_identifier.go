// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/util/string.dart

import (
	"strconv"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ToCssIdentifier returns a minimally-escaped CSS identifier whose contents
// evaluates to text.
//
// Only characters that need escaping are escaped; everything else passes
// through raw. Returns an error if text cannot be represented as a CSS
// identifier: the empty string, U+0000, or a lone surrogate.
//
// Matches Dart: StringExtension.toCssIdentifier (util/string.dart)
//
// # Divergence from Dart
//
// Dart stores strings as UTF-16 code units, so surrogates are real and
// must be handled. Go strings are UTF-8 and cannot contain individual
// surrogates. Non-BMP code points (> U+FFFF) are represented as single
// runes. The surrogate handling is kept as a safety check but should
// never trigger on valid UTF-8 input.
func ToCssIdentifier(text string) (string, error) {
	scanner := sasscommon.NewSpanScanner([]byte(text), nil)

	// Writes ch as a lowercase-hex backslash escape, with a disambiguating
	// trailing space when the next character is itself hex (otherwise the
	// escape would swallow it).
	//
	// Matches Dart: writeEscape in toCssIdentifier
	writeEscape := func(buf *strings.Builder, ch rune) {
		buf.WriteRune('\\')
		buf.WriteString(strconv.FormatInt(int64(ch), 16))
		if next := scanner.PeekChar(0); next >= 0 && IsHex(next) {
			buf.WriteRune(' ')
		}
	}

	// Consumes a surrogate pair: a lone high surrogate is an error, a pair
	// over a private-use plane escapes as its combined code point, and any
	// other pair passes through raw.
	//
	// Matches Dart: consumeSurrogatePair in toCssIdentifier
	consumeSurrogatePair := func(buf *strings.Builder) error {
		high, err := scanner.ReadChar()
		if err != nil {
			return err
		}
		next := scanner.PeekChar(0)
		if next < 0 || !IsLowSurrogate(next) {
			return scanner.Error(
				"An individual surrogate can't be represented as a CSS identifier.",
				scanner.Position(), 1,
			)
		}
		low, err := scanner.ReadChar()
		if err != nil {
			return err
		}
		if IsPrivateUseHighSurrogate(int(high)) {
			writeEscape(buf, rune(CombineSurrogates(int(high), int(low))))
		} else {
			buf.WriteRune(rune(high))
			buf.WriteRune(rune(low))
		}
		return nil
	}

	var buf strings.Builder
	// A lone "-" is not a valid identifier start, so it escapes as its own
	// hex form; a "--" prefix passes through since custom properties are
	// always representable.
	if scanner.ScanChar('-') {
		if scanner.IsDone() {
			return "\\2d", nil
		}
		buf.WriteRune('-')
		if scanner.ScanChar('-') {
			buf.WriteRune('-')
			goto body
		}
	}

	// The first character follows name-start rules (name-start or escaped)
	// while the body below follows the looser name rules; private-use BMP
	// characters always escape so they never surface raw.
	switch ch := scanner.PeekChar(0); {
	case ch < 0:
		return "", scanner.Error(
			"The empty string can't be represented as a CSS identifier.", -1, 0)
	case ch == 0:
		return "", scanner.Error(
			"The U+0000 can't be represented as a CSS identifier.", -1, 0)
	case IsHighSurrogate(ch):
		if err := consumeSurrogatePair(&buf); err != nil {
			return "", err
		}
	case IsLowSurrogate(ch):
		scanner.ReadChar()
		return "", scanner.Error(
			"An individual surrogate can't be represented as a CSS identifier.",
			scanner.Position()-1, 1)
	case IsNameStart(ch) && !IsPrivateUseBMP(ch):
		c, err := scanner.ReadChar()
		if err != nil {
			return "", err
		}
		buf.WriteRune(rune(c))
	default:
		c, err := scanner.ReadChar()
		if err != nil {
			return "", err
		}
		writeEscape(&buf, rune(c))
	}

body:
	// The identifier body: name characters pass through raw (except
	// private-use BMP, which escapes); anything else escapes. U+0000 and
	// lone surrogates are errors.
	for {
		switch ch := scanner.PeekChar(0); {
		case ch < 0:
			return buf.String(), nil
		case ch == 0:
			return "", scanner.Error(
				"The U+0000 can't be represented as a CSS identifier.", -1, 0)
		case IsHighSurrogate(ch):
			if err := consumeSurrogatePair(&buf); err != nil {
				return "", err
			}
		case IsLowSurrogate(ch):
			scanner.ReadChar()
			return "", scanner.Error(
				"An individual surrogate can't be represented as a CSS identifier.",
				scanner.Position()-1, 1)
		case IsName(ch) && !IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteRune(rune(c))
		default:
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			writeEscape(&buf, rune(c))
		}
	}
}
