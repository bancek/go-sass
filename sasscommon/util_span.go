// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/util/span.dart

import (
	"fmt"
	"net/url"
)

// BogusSpan is a span that points nowhere: an empty span in an empty,
// URL-less file.
//
// Fake AST nodes that will never be presented to the user carry this instead
// of a real location, as do embedded compilation failures that have no
// associated spans, so formatting paths never need a nil check for them.
//
// Matches Dart: bogusSpan
var BogusSpan = SimpleFileSpan{}

// Before returns the span covering the text from the start of this span to
// the start of sub. Both spans must live in the same file and sub must sit
// fully inside this span; otherwise it reports an ArgumentError naming both
// spans, matching Dart's containment throw.
//
// Matches Dart: SpanExtensions.before
func (s SimpleFileSpan) Before(sub FileSpan) (FileSpan, error) {
	subURL, err := sub.SourceURL()
	if err != nil {
		return nil, err
	}
	sURL, err := s.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(sURL, subURL) {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		subStr, err := sub.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s and %s are in different files.", sStr, subStr)}
	}
	subStart, err := sub.StartLocation()
	if err != nil {
		return nil, err
	}
	subEnd, err := sub.EndLocation()
	if err != nil {
		return nil, err
	}
	if subStart.Offset < s.start || subEnd.Offset > s.end {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		subStr, err := sub.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s isn't inside %s.", subStr, sStr)}
	}
	return SimpleFileSpan{file: s.file, start: s.start, end: subStart.Offset}, nil
}

// After returns the span covering the text from the end of sub to the end
// of this span. Both spans must live in the same file and sub must sit
// fully inside this span; otherwise it reports an ArgumentError naming both
// spans, matching Dart's containment throw.
//
// Matches Dart: SpanExtensions.after
func (s SimpleFileSpan) After(sub FileSpan) (FileSpan, error) {
	subURL, err := sub.SourceURL()
	if err != nil {
		return nil, err
	}
	sURL, err := s.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(sURL, subURL) {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		subStr, err := sub.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s and %s are in different files.", sStr, subStr)}
	}
	subStart, err := sub.StartLocation()
	if err != nil {
		return nil, err
	}
	subEnd, err := sub.EndLocation()
	if err != nil {
		return nil, err
	}
	if subStart.Offset < s.start || subEnd.Offset > s.end {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		subStr, err := sub.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s isn't inside %s.", subStr, sStr)}
	}
	return SimpleFileSpan{file: s.file, start: subEnd.Offset, end: s.end}, nil
}

// Between returns the span covering the text after this span ends and
// before other begins. Both spans must live in the same file and other must
// start on or after this span's end; otherwise it reports an ArgumentError,
// matching Dart's ordering throw.
//
// Matches Dart: SpanExtensions.between
func (s SimpleFileSpan) Between(other FileSpan) (FileSpan, error) {
	otherURL, err := other.SourceURL()
	if err != nil {
		return nil, err
	}
	sURL, err := s.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(sURL, otherURL) {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		otherStr, err := other.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s and %s are in different files.", sStr, otherStr)}
	}
	otherStart, err := other.StartLocation()
	if err != nil {
		return nil, err
	}
	if s.end > otherStart.Offset {
		sStr, err := s.String()
		if err != nil {
			return nil, err
		}
		otherStr, err := other.String()
		if err != nil {
			return nil, err
		}
		return nil, &ArgumentError{Message: fmt.Sprintf("%s isn't before %s.", sStr, otherStr)}
	}
	return SimpleFileSpan{file: s.file, start: s.end, end: otherStart.Offset}, nil
}

// Contains reports whether target sits fully inside this span's inclusive
// [start, end] range. Spans from different files never contain one another,
// so a URL mismatch reports false rather than an error; offsets compare by
// character position within the shared file.
//
// Matches Dart: SpanExtensions.contains
func (s SimpleFileSpan) Contains(target FileSpan) (bool, error) {
	targetURL, err := target.SourceURL()
	if err != nil {
		return false, err
	}
	var sourceURL *url.URL
	if s.file != nil {
		sourceURL = s.file.URL()
	}
	if !sameURL(sourceURL, targetURL) {
		return false, nil
	}
	targetStart, err := target.StartLocation()
	if err != nil {
		return false, err
	}
	if s.start > targetStart.Offset {
		return false, nil
	}
	targetEnd, err := target.EndLocation()
	if err != nil {
		return false, err
	}
	return s.end >= targetEnd.Offset, nil
}

// Trim returns this span with whitespace stripped from both sides,
// implemented as a left trim followed by a right trim.
//
// Matches Dart: SpanExtensions.trim
func (s SimpleFileSpan) Trim() (FileSpan, error) {
	result, err := s.TrimLeft()
	if err != nil {
		return nil, err
	}
	return result.TrimRight()
}

// TrimLeft returns this span with leading whitespace stripped. A span with
// no leading whitespace returns itself unchanged.
//
// Matches Dart: SpanExtensions.trimLeft
func (s SimpleFileSpan) TrimLeft() (FileSpan, error) {
	return trimLeftSpan(s)
}

// TrimRight returns this span with trailing whitespace stripped. The end
// offset backs up over ASCII whitespace one byte at a time; a span with no
// trailing whitespace returns itself unchanged.
//
// Matches Dart: SpanExtensions.trimRight
func (s SimpleFileSpan) TrimRight() (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	end := len(text)
	for end > 0 && isWhitespaceByte(text[end-1]) {
		end--
	}
	if end == len(text) {
		return s, nil
	}
	return SimpleFileSpan{file: s.file, start: s.start, end: s.start + end}, nil
}

// WithoutInitialAtRule returns the subspan after a leading at-rule
// (@ followed by an identifier) plus any whitespace following it. A span
// that does not start with "@" is a scan failure, matching Dart's
// expectChar behavior.
//
// Matches Dart: SpanExtensions.withoutInitialAtRule
func (s SimpleFileSpan) WithoutInitialAtRule() (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) == 0 || text[0] != '@' {
		return nil, &ScanError{Message: "Expected @.", Span: s}
	}
	pos, err := scanIdent(text, 1, s)
	if err != nil {
		return nil, err
	}
	if pos >= len(text) {
		return s, nil
	}
	result, err := s.Subspan(pos, len(text))
	if err != nil {
		return nil, err
	}
	result2, err := trimLeftSpan(result)
	if err != nil {
		return nil, err
	}
	return result2, nil
}

// scanIdent advances pos past one CSS identifier, folding backslash escapes
// into the identifier the way Dart's string scanner does: a backslash
// consumes its escape sequence, name characters are taken literally, and
// anything else ends the identifier.
//
// Matches Dart: _scanIdentifier (in lib/src/util/span.dart)
func scanIdent(text string, pos int, span FileSpan) (int, error) {
	for pos < len(text) {
		ch := text[pos]
		switch {
		case ch == '\\':
			var err error
			pos, err = consumeEscaped(text, pos, span)
			if err != nil {
				return pos, err
			}
		case isName(text[pos]):
			pos++
		default:
			return pos, nil
		}
	}
	return pos, nil
}

// consumeEscaped advances pos past the escape sequence opening at the
// backslash: a newline after the backslash is a scan failure (a continuation
// cannot hide there), up to six hex digits plus one optional whitespace form
// a code-point escape, and anything else is a single escaped character.
//
// Matches Dart: consumeEscapedCharacter (in lib/src/utils.dart)
func consumeEscaped(text string, pos int, span FileSpan) (int, error) {
	if pos >= len(text) || text[pos] != '\\' {
		return pos, nil
	}
	pos++ // skip \
	if pos >= len(text) {
		return pos, nil
	}
	ch := text[pos]
	switch {
	case ch == '\n' || ch == '\r' || ch == '\f':
		return pos, &ScanError{Message: "Expected escape sequence.", Span: span}
	case isHex(ch):
		for i := 0; i < 6 && pos < len(text) && isHex(text[pos]); i++ {
			pos++
		}
		if pos < len(text) && isWhitespaceByte(text[pos]) {
			pos++
		}
	default:
		pos++
	}
	return pos, nil
}

// isName reports whether b can appear in a CSS identifier: letters, digits,
// "_", "-", or any non-ASCII byte (the high range is accepted byte-wise
// because identifier scanning works on raw bytes here).
//
// Matches Dart: CharacterExtension.isName
func isName(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-' ||
		b >= 0x80
}

// isHex reports whether b is an ASCII hexadecimal digit.
//
// Matches Dart: hex-digit check in consumeEscapedCharacter
func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// isWhitespaceByte reports whether b is ASCII whitespace (space, tab,
// newline, carriage return, or form feed), matching the set Dart's
// isWhitespace recognizes for span trimming.
//
// Matches Dart: NullableCharacterExtension.isWhitespace
func isWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// trimLeftSpan strips leading whitespace shared-helper style: it counts the
// whitespace prefix, returns the span unchanged when there is none, and
// otherwise slices from the first non-whitespace byte to the span end.
//
// Matches Dart: SpanExtensions.trimLeft
func trimLeftSpan(s FileSpan) (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	start := 0
	for start < len(text) && isWhitespaceByte(text[start]) {
		start++
	}
	if start == 0 {
		return s, nil
	}
	length, err := s.Length()
	if err != nil {
		return nil, err
	}
	return s.Subspan(start, length)
}

// InitialQuoted returns the span of the quoted string at the start of this
// span, including both quotes. The span must open with a double or single
// quote; the scan then walks to the matching close quote, stepping over backslash-escaped
// pairs so an escaped quote cannot end the string early. An unterminated
// string is a scan failure.
//
// Matches Dart: SpanExtensions.initialQuoted
func (s SimpleFileSpan) InitialQuoted() (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) == 0 || (text[0] != '"' && text[0] != '\'') {
		return nil, &ScanError{Message: "Expected quote.", Span: s}
	}
	quote := text[0]
	end := 1
	for end < len(text) {
		if text[end] == quote {
			end++
			result, err := s.Subspan(0, end)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		if text[end] == '\\' {
			end++
			if end < len(text) {
				end++
			}
			continue
		}
		end++
	}
	return nil, &ScanError{Message: fmt.Sprintf("Expected %s.", string(quote)), Span: s}
}

// InitialIdentifier returns the span of the identifier at the start of this
// span. When includeLeading is positive, that many leading characters (for
// example a "-" prefix the caller already consumed) are folded in before the
// identifier scan begins.
//
// Matches Dart: SpanExtensions.initialIdentifier
func (s SimpleFileSpan) InitialIdentifier(includeLeading int) (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	pos := 0
	for i := 0; i < includeLeading; i++ {
		if pos >= len(text) {
			result, err := s.Subspan(0, pos)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		pos++
	}
	pos, err = scanIdent(text, pos, s)
	if err != nil {
		return nil, err
	}
	result, err := s.Subspan(0, pos)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// WithoutInitialIdentifier returns the subspan starting where the leading
// identifier ends, so callers can peel a namespace or name off the front.
//
// Matches Dart: SpanExtensions.withoutInitialIdentifier
func (s SimpleFileSpan) WithoutInitialIdentifier() (FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	pos, err := scanIdent(text, 0, s)
	if err != nil {
		return nil, err
	}
	result, err := s.Subspan(pos, len(text))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// WithoutNamespace returns the subspan after a leading "namespace." prefix:
// it peels the identifier, then one more byte for the "." separator. When
// no dot follows, the span after the identifier is returned as-is, so plain
// names pass through unchanged.
//
// Matches Dart: SpanExtensions.withoutNamespace
func (s SimpleFileSpan) WithoutNamespace() (FileSpan, error) {
	afterIdent, err := s.WithoutInitialIdentifier()
	if err != nil {
		return nil, err
	}
	text, err := afterIdent.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) == 0 || text[0] != '.' {
		return afterIdent, nil
	}
	result, err := afterIdent.Subspan(1, len(text))
	if err != nil {
		return nil, err
	}
	return result, nil
}
