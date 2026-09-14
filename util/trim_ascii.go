// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/utils.dart

// TrimAscii strips ASCII whitespace characters from s, like String.trim
// but touching only ASCII whitespace.
//
// If excludeEscape is true, whitespace included in a CSS escape is left in
// place.
//
// Matches Dart: trimAscii (utils.dart)
func TrimAscii(s string, excludeEscape bool) string {
	start := firstNonWhitespace(s)
	if start == nil {
		return ""
	}
	end := lastNonWhitespace(s, excludeEscape)
	return s[*start : *end+1]
}

// TrimAsciiLeft strips leading ASCII whitespace characters from s, like
// String.trimLeft but touching only ASCII whitespace.
//
// Matches Dart: trimAsciiLeft (utils.dart)
func TrimAsciiLeft(s string) string {
	start := firstNonWhitespace(s)
	if start == nil {
		return ""
	}
	return s[*start:]
}

// TrimAsciiRight strips trailing ASCII whitespace characters from s, like
// String.trimRight but touching only ASCII whitespace.
//
// If excludeEscape is true, whitespace included in a CSS escape is left in
// place.
//
// Matches Dart: trimAsciiRight (utils.dart)
func TrimAsciiRight(s string, excludeEscape bool) string {
	end := lastNonWhitespace(s, excludeEscape)
	if end == nil {
		return ""
	}
	return s[:*end+1]
}

// firstNonWhitespace returns the index of the first character in s that
// is not ASCII whitespace, or nil if s is entirely whitespace.
//
// Matches Dart: _firstNonWhitespace (utils.dart)
func firstNonWhitespace(s string) *int {
	for i := 0; i < len(s); i++ {
		if !isASCIIWhitespace(s[i]) {
			return &i
		}
	}
	return nil
}

// lastNonWhitespace returns the index of the last character in s that is
// not ASCII whitespace, or nil if s is entirely whitespace.
//
// If excludeEscape is true, a trailing backslash counts as content (it and
// the whitespace it escapes are part of a CSS escape), so the index just
// past it is returned instead of moving further left.
//
// Matches Dart: _lastNonWhitespace (utils.dart)
func lastNonWhitespace(s string, excludeEscape bool) *int {
	for i := len(s) - 1; i >= 0; i-- {
		if !isASCIIWhitespace(s[i]) {
			if excludeEscape && i != 0 && i != len(s)-1 && s[i] == '\\' {
				v := i + 1
				return &v
			}
			v := i
			return &v
		}
	}
	return nil
}

// isASCIIWhitespace returns whether b is an ASCII whitespace character:
// space, tab, line feed, carriage return, or form feed. Only these five
// are trimmed, so non-ASCII whitespace is never touched.
//
// Matches Dart: NullableCharacterExtension.isWhitespace (util/character.dart)
func isASCIIWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// A returns "an word" when word starts with a vowel and "a word"
// otherwise.
//
// Matches Dart: a (utils.dart)
func A(word string) string {
	switch word[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an " + word
	default:
		return "a " + word
	}
}
