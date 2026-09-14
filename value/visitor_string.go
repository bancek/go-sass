// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitString section:
// _visitQuotedString, _visitUnquotedString, _tryPrivateUseCharacter,
// _writeEscape)

import (
	"strconv"
	"strings"

	"github.com/bancek/go-sass/util"
)

// visitQuotedString writes s surrounded by quotes, picking the quote
// character from the contents (see visitQuotedStringForce).
//
// Matches Dart: _SerializeVisitor._visitQuotedString (default call)
func (sv *SerializeVisitor) visitQuotedString(s string) {
	sv.visitQuotedStringForce(s, false)
}

// visitQuotedStringForce writes s as a quoted string. With forceDouble
// unset, the contents accumulate in a scratch buffer first so the quote
// can be chosen afterwards: double quotes unless the text holds a double
// quote, in which case single quotes win — unless both quote kinds appear,
// which restarts the whole write in forced-double mode with inner doubles
// escaped and inner singles kept bare. Newlines and unprintable ASCII
// become hex escapes, backslashes double, and private-use characters take
// the expanded-mode escape path.
//
// Matches Dart: _SerializeVisitor._visitQuotedString (forceDoubleQuote
// parameter)
func (sv *SerializeVisitor) visitQuotedStringForce(s string, forceDouble bool) {
	includesSingleQuote := false
	includesDoubleQuote := false

	// In the Dart code, when not forceDouble, a temporary StringBuffer is used.
	// The content is accumulated there, then wrapped in quotes and written to _buffer.
	// When forceDouble, we write directly to _buffer with double quotes.
	var contentBuf strings.Builder
	var buf *strings.Builder
	if forceDouble {
		buf = &strings.Builder{}
		buf.WriteByte('"')
	} else {
		buf = &contentBuf
	}

	writeRuneToBuf := func(r rune) {
		if buf != nil {
			buf.WriteRune(r)
		} else {
			_, _ = sv.sb.WriteRune(r)
		}
	}

	writeByteToBuf := func(b byte) {
		if buf != nil {
			buf.WriteByte(b)
		} else {
			_ = sv.sb.WriteByte(b)
		}
	}

	for i := 0; i < len(s); {
		r, size := utf8Decode(s[i:])

		switch {
		case r == '\'' && forceDouble:
			writeByteToBuf('\'')
			i += size

		case r == '\'' && includesDoubleQuote:
			// Both quote types encountered, restart with forceDouble
			sv.visitQuotedStringForce(s, true)
			return

		case r == '\'':
			includesSingleQuote = true
			writeByteToBuf('\'')
			i += size

		case r == '"' && forceDouble:
			writeByteToBuf('\\')
			writeByteToBuf('"')
			i += size

		case r == '"' && includesSingleQuote:
			sv.visitQuotedStringForce(s, true)
			return

		case r == '"':
			includesDoubleQuote = true
			writeByteToBuf('"')
			i += size

		case r == '\\':
			writeByteToBuf('\\')
			writeByteToBuf('\\')
			i += size

		case r != '\t' && (r <= 0x1F || r == 0x7F):
			sv.writeEscape(buf, int(r), s, i)
			i += size

		default:
			if newIdx := sv.tryPrivateUse(buf, int(r), s, i); newIdx >= 0 {
				i = newIdx + size
			} else {
				writeRuneToBuf(r)
				i += size
			}
		}
	}

	if forceDouble {
		buf.WriteByte('"')
		_, _ = sv.sb.WriteString(buf.String())
	} else {
		quote := byte('"')
		if includesDoubleQuote {
			quote = '\''
		}
		_ = sv.sb.WriteByte(quote)
		_, _ = sv.sb.WriteString(contentBuf.String())
		_ = sv.sb.WriteByte(quote)
	}
}

// visitUnquotedString writes s without quotes, folding each newline into
// a single space and collapsing the spaces after a newline so line breaks
// never introduce doubled whitespace. Private-use characters still take
// the expanded-mode escape path.
//
// Matches Dart: _SerializeVisitor._visitUnquotedString
func (sv *SerializeVisitor) visitUnquotedString(s string) {
	afterNewline := false
	for i := 0; i < len(s); {
		r, size := utf8Decode(s[i:])
		ch := int(r)

		switch ch {
		case '\n':
			_ = sv.sb.WriteByte(' ')
			afterNewline = true
			i += size

		case ' ':
			if !afterNewline {
				_ = sv.sb.WriteByte(' ')
			}
			i += size

		default:
			afterNewline = false
			if newIdx := sv.tryPrivateUse(nil, ch, s, i); newIdx >= 0 {
				i = newIdx + size
			} else {
				_, _ = sv.sb.WriteString(string(r))
				i += size
			}
		}
	}
}

// tryPrivateUse writes the private-use character at byte offset i as an
// escape and returns i (or the second unit's offset for a surrogate pair)
// when it does. It returns -1 when there is nothing to escape: always in
// compressed mode, where private-use characters pass through visibly.
// Expanded mode escapes them all since glyph-font code points are easier
// to tell apart as escapes than as rendered glyphs.
//
// Matches Dart: _SerializeVisitor._tryPrivateUseCharacter
func (sv *SerializeVisitor) tryPrivateUse(buf *strings.Builder, codeUnit int, s string, i int) int {
	if sv.isCompressed() {
		return -1
	}
	if isPrivateBMP(codeUnit) || isSupplementaryPrivateUse(codeUnit) {
		sv.writeEscape(buf, codeUnit, s, i)
		return i
	}
	if isPrivateHighSurrogate(codeUnit) && i+1 < len(s) {
		next := 0
		r, _ := utf8Decode(s[i+1:])
		next = int(r)
		combined := util.CombineSurrogates(codeUnit, next)
		sv.writeEscape(buf, combined, s, i+1)
		return i + 1
	}
	return -1
}

// writeEscape writes character as a backslash-plus-lowercase-hex escape,
// appending a disambiguating space when the next character is a hex digit,
// space, or tab that would otherwise merge into the escape. buf selects
// the destination: the scratch quote buffer when non-nil, the output
// buffer otherwise.
//
// Matches Dart: _SerializeVisitor._writeEscape
func (sv *SerializeVisitor) writeEscape(buf *strings.Builder, character int, s string, i int) {
	if buf != nil {
		buf.WriteByte('\\')
		buf.WriteString(strconv.FormatInt(int64(character), 16))
	} else {
		_ = sv.sb.WriteByte('\\')
		_, _ = sv.sb.WriteString(strconv.FormatInt(int64(character), 16))
	}

	if i+1 >= len(s) {
		return
	}
	next := 0
	r, _ := utf8Decode(s[i+1:])
	next = int(r)
	if isHexDig(next) || next == ' ' || next == '\t' {
		if buf != nil {
			buf.WriteByte(' ')
		} else {
			_ = sv.sb.WriteByte(' ')
		}
	}
}
