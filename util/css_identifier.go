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
// If body is true, characters that are valid in an identifier body but not at
// its start are left unescaped.
//
// Returns an error if text cannot be represented as a CSS identifier: the
// empty string, U+0000, or a lone surrogate.
//
// Matches Dart: StringExtension.toCssIdentifier
func ToCssIdentifier(text string) (string, error) {
	return toCssIdentifier(text, false)
}

// ToCssIdentifierBody is ToCssIdentifier with body mode enabled.
//
// Matches Dart: StringExtension.toCssIdentifier(body: true)
func ToCssIdentifierBody(text string) (string, error) {
	return toCssIdentifier(text, true)
}

// TryCssIdentifier returns the escaped identifier for text, or false when
// text cannot be represented as one.
//
// Matches Dart: StringExtension.tryCssIdentifier
func TryCssIdentifier(text string) (string, bool) {
	result, err := ToCssIdentifier(text)
	return result, err == nil
}

// IsSimpleIdentifier reports whether text is a valid CSS identifier whose
// semantic content is the text itself.
//
// This is a quick optimization check: it may return false negatives, but not
// false positives.
//
// Matches Dart: StringExtension._isSimpleIdentifier
func IsSimpleIdentifier(text string) bool {
	first := true
	for _, ch := range text {
		if first {
			if !IsNameStart(int(ch)) || IsPrivateUseBMP(int(ch)) || ch > 0xFFFF {
				return false
			}
			first = false
			continue
		}
		if !IsName(int(ch)) || IsPrivateUseBMP(int(ch)) || ch > 0xFFFF {
			return false
		}
	}
	return !first
}

func isPrivateUseSupplementary(ch int) bool {
	return ch >= 0xF0000 && ch <= 0x10FFFF
}

func toCssIdentifier(text string, body bool) (string, error) {
	if IsSimpleIdentifier(text) {
		return text, nil
	}

	scanner := sasscommon.NewSpanScanner([]byte(text), nil)

	// Writes ch as a lowercase-hex backslash escape. Dart always writes a
	// trailing space so the escape can't swallow a following hex digit.
	writeHexEscape := func(buf *strings.Builder, ch int) {
		buf.WriteByte('\\')
		buf.WriteString(strconv.FormatInt(int64(ch), 16))
		buf.WriteByte(' ')
	}

	// Consumes a surrogate pair: a lone high surrogate is an error, a pair
	// over a private-use plane escapes as its combined code point, and any
	// other pair passes through raw.
	consumeSurrogatePair := func(buf *strings.Builder) error {
		high, err := scanner.ReadChar()
		if err != nil {
			return err
		}
		next := scanner.PeekChar(0)
		if next < 0 || !IsLowSurrogate(next) {
			return scanner.Error(
				"An individual surrogates can't be represented as a CSS identifier.",
				scanner.Position(), 1,
			)
		}
		low, err := scanner.ReadChar()
		if err != nil {
			return err
		}
		if IsPrivateUseHighSurrogate(int(high)) {
			writeHexEscape(buf, CombineSurrogates(int(high), int(low)))
		} else {
			buf.WriteRune(rune(high))
			buf.WriteRune(rune(low))
		}
		return nil
	}

	var buf strings.Builder
	doubleDash := false
	if !body && scanner.ScanChar('-') {
		if scanner.IsDone() {
			return "\\2d ", nil
		}
		buf.WriteByte('-')
		if scanner.ScanChar('-') {
			buf.WriteByte('-')
			doubleDash = true
		}
	}

	if !doubleDash {
		switch ch := scanner.PeekChar(0); {
		case ch < 0:
			return "", scanner.Error(
				"The empty string can't be represented as a CSS identifier.", -1, 0)
		case ch == 0:
			return "", scanner.Error(
				"U+0000 can't be represented as a CSS identifier.", -1, 0)
		case IsHighSurrogate(ch):
			if err := consumeSurrogatePair(&buf); err != nil {
				return "", err
			}
		case IsLowSurrogate(ch):
			scanner.ReadChar()
			return "", scanner.Error(
				"An individual surrogate can't be represented as a CSS identifier.",
				scanner.Position()-1, 1)
		case isPrivateUseSupplementary(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			writeHexEscape(&buf, c)
		case body && IsName(ch) && !IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteRune(rune(c))
		case !body && IsNameStart(ch) && !IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteRune(rune(c))
		case ch <= 0x1F || ch == 0x7F || IsDigit(ch) || IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			writeHexEscape(&buf, c)
		default:
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteByte('\\')
			buf.WriteRune(rune(c))
		}
	}

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
		case isPrivateUseSupplementary(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			writeHexEscape(&buf, c)
		case IsName(ch) && !IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteRune(rune(c))
		case ch <= 0x1F || ch == 0x7F || IsPrivateUseBMP(ch):
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			writeHexEscape(&buf, c)
		default:
			c, err := scanner.ReadChar()
			if err != nil {
				return "", err
			}
			buf.WriteByte('\\')
			buf.WriteRune(rune(c))
		}
	}
}
