// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/util/span.dart (trim, trimLeft, trimRight,
//   withoutInitialAtRule), lib/src/parse/parser.dart
//   (indexAfterWhitespaceAndComments), lib/src/parse/stylesheet.dart
//   (lastIndexOfWhitespaceAndComments)

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// indexAfterWhitespaceAndComments returns the leftmost index in text after any
// initial whitespace or comments (either loud or silent).
//
// Matches Dart: Parser.indexAfterWhitespaceAndComments
func indexAfterWhitespaceAndComments(text string) (int, error) {
	p := NewParser([]byte(text), nil, nil)
	if err := p.whitespace(true); err != nil {
		return 0, err
	}
	return p.scanner.Position(), nil
}

// lastIndexOfWhitespaceAndComments returns the index in text of the beginning
// of the final contiguous sequence of whitespace and/or comments (either loud
// or silent).
//
// If text doesn't end with whitespace or comments, it returns len(text). If
// text is entirely whitespace or comments, it returns 0. It assumes text is
// syntactically valid Sass source code.
//
// Matches Dart: StylesheetParser.lastIndexOfWhitespaceAndComments
func lastIndexOfWhitespaceAndComments(text string) (int, error) {
	if index, ok := lastIndexOfWhitespaceAndCommentsQuick(text); ok {
		return index, nil
	}
	p := NewScssParser([]byte(text), nil, true)
	return p.lastIndexOfWhitespaceAndComments()
}

// lastIndexOfWhitespaceAndCommentsQuick is a partial implementation of
// lastIndexOfWhitespaceAndComments that scans the string from right to left
// but reports false if it encounters anything that could be a comment.
func lastIndexOfWhitespaceAndCommentsQuick(text string) (int, bool) {
	lastNonWhitespace := -1
	for i := len(text) - 1; i >= 0; i-- {
		ch := int(text[i])
		switch {
		case ch == '/' && i > 0:
			if text[i-1] == '/' || text[i-1] == '*' {
				return 0, false
			}
			if lastNonWhitespace < 0 {
				lastNonWhitespace = i
			}

		// If we hit a newline after we've seen a non-whitespace character and
		// we haven't seen anything that could be a single-line comment, that's
		// definitely our stopping point.
		case util.IsNewline(ch) && lastNonWhitespace >= 0:
			return lastNonWhitespace + 1, true

		case util.IsWhitespace(ch):

		default:
			if lastNonWhitespace < 0 {
				lastNonWhitespace = i
			}
		}
	}

	if lastNonWhitespace >= 0 {
		return lastNonWhitespace + 1, true
	}
	return 0, true
}

// lastIndexOfWhitespaceAndComments is a helper method for
// lastIndexOfWhitespaceAndComments.
//
// It's not feasible to determine this in all cases by iterating backwards
// from the end of the string. For example, `... //"` may or may not be a
// single-line comment depending on whether the beginning of the line
// contains a `"`, and `... // */` may or may not be a multi-line comment
// depending on whether `/*` appears anywhere earlier in the string, whether
// *that's* commented out, and so on. In order to handle this problem
// generally, we have to essentially tokenize the string from the beginning
// and record where the most recent whitespace character appeared.
//
// Matches Dart: StylesheetParser._lastIndexOfWhitespaceAndComments
func (p *StylesheetParser) lastIndexOfWhitespaceAndComments() (int, error) {
	index := 0
	hasIndex := false

	for {
		ch, escaped, err := p.peekCharOrEscape(0)
		if err != nil {
			return 0, err
		}
		switch {
		case !escaped && (ch == '"' || ch == '\''):
			hasIndex = false
			if _, err := p.interpolatedStringToken(); err != nil {
				return 0, err
			}

		case !escaped && ch == '/':
			switch p.scanner.PeekChar(1) {
			case '*', '/':
				index = p.scanner.Position()
				hasIndex = true
				if err := p.whitespace(true); err != nil {
					return 0, err
				}
			default:
				hasIndex = false
				if _, err := p.readChar(); err != nil {
					return 0, err
				}
			}

		// Add a full interpolated identifier to handle cases like "#{...}--1",
		// since "--1" isn't a valid identifier on its own.
		case !escaped && ch == '#' && p.scanner.PeekChar(1) == '{':
			hasIndex = false
			if _, err := p.interpolatedIdentifier(false); err != nil {
				return 0, err
			}

		case !escaped && util.IsWhitespace(ch):
			index = p.scanner.Position()
			hasIndex = true
			if err := p.whitespace(true); err != nil {
				return 0, err
			}

		case ch == 'u' || ch == 'U':
			hasIndex = false
			beforeURL := p.scanner.State()
			identifier, err := p.interpolatedIdentifier(false)
			if err != nil {
				return 0, err
			}
			plain, err := identifier.AsPlainIdentifier()
			if err != nil {
				return 0, err
			}
			if plain == nil || (*plain != "url" && *plain != "url-prefix") {
				continue
			}
			if _, err := p.tryUrlContents(beforeURL, identifier, false); err != nil {
				return 0, err
			}

		case ch < 0:
			if hasIndex {
				return index, nil
			}
			return p.scanner.Position(), nil

		case escaped || p.lookingAtIdentifier(nil):
			hasIndex = false
			if _, err := p.interpolatedIdentifier(false); err != nil {
				return 0, err
			}

		default:
			hasIndex = false
			if _, err := p.readChar(); err != nil {
				return 0, err
			}
		}
	}
}

// trimSpan returns s with all whitespace and comments trimmed from both
// sides.
//
// This operation is lazy—it only actually loads or changes the span once its
// data is accessed.
//
// Matches Dart: SpanExtensions.trim
func trimSpan(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	return sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
		left, err := trimLeft(s)
		if err != nil {
			return nil, err
		}
		return trimRight(left)
	}), nil
}

// trimLeftSpan returns s with all leading whitespace and comments trimmed.
//
// This operation is lazy—it only actually loads or changes the span once its
// data is accessed.
//
// Matches Dart: SpanExtensions.trimLeft
func trimLeftSpan(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	return sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
		return trimLeft(s)
	}), nil
}

// trimLeft returns s with all leading whitespace and comments trimmed.
//
// Matches Dart: SpanExtensions._trimLeft
func trimLeft(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	start, err := indexAfterWhitespaceAndComments(text)
	if err != nil {
		return nil, err
	}
	return s.Subspan(start, len(text))
}

// trimRightSpan returns s with all trailing whitespace and comments trimmed.
//
// This operation is lazy—it only actually loads or changes the span once its
// data is accessed.
//
// Matches Dart: SpanExtensions.trimRight
func trimRightSpan(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	return sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
		return trimRight(s)
	}), nil
}

// trimRight returns s with all trailing whitespace and comments trimmed.
//
// Matches Dart: SpanExtensions._trimRight
func trimRight(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	end, err := lastIndexOfWhitespaceAndComments(text)
	if err != nil {
		return nil, err
	}
	return s.Subspan(0, end)
}

// withoutInitialAtRule returns a subspan excluding an initial at-rule and any
// whitespace and comments after it.
//
// Matches Dart: SpanExtensions.withoutInitialAtRule
func withoutInitialAtRule(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) == 0 || text[0] != '@' {
		return nil, &sasscommon.ScanError{Message: "Expected @.", Span: s}
	}
	pos, err := scanIdentName(text, 1, s)
	if err != nil {
		return nil, err
	}
	after, err := s.Subspan(pos, len(text))
	if err != nil {
		return nil, err
	}
	return trimLeftSpan(after)
}
