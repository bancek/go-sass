// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (almostAnyValue,
// _interpolatedDeclarationValue)

import (
	"fmt"

	"github.com/bancek/go-sass/util"
)

// almostAnyValue consumes tokens up to a statement boundary ("{", "}", ";",
// or "!") and returns them as interpolation for later re-parsing.
//
// String and comment boundaries are honored and interpolation is kept. When
// omitComments holds, comments are still consumed but left out of the result.
// Unlike interpolatedDeclarationValue below, this pass always stops at curly
// braces, copies backslashes literally instead of decoding escapes, and
// leaves adjacent whitespace uncompressed.
func (p *StylesheetParser) almostAnyValue(omitComments bool) (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}

	var brackets []int

loop:
	for {
		switch ch := p.scanner.PeekChar(0); {
		case ch == '\\':
			// Copy both bytes literally: the result is re-parsed later,
			// so escapes must survive this pass untouched.
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
			c, err = p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)

		case ch == '\'' || ch == '"':
			interp, err := p.interpolatedStringToken()
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(interp)

		case ch == '/':
			switch p.scanner.PeekChar(1) {
			case '*':
				if !omitComments {
					text, err := p.rawText(func() error { return p.loudComment() })
					if err != nil {
						return nil, err
					}
					buffer.Write(text)
				} else {
					if err := p.loudComment(); err != nil {
						return nil, err
					}
				}
			case '/':
				if !omitComments {
					text, err := p.rawText(func() error { _, err := p.silentComment(); return err })
					if err != nil {
						return nil, err
					}
					buffer.Write(text)
				} else {
					if _, err := p.silentComment(); err != nil {
						return nil, err
					}
				}
			default:
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
			}

		case ch == '#' && p.scanner.PeekChar(1) == '{':
			// Route through a full interpolated identifier so suffixes
			// like "#{...}--1" survive even though "--1" alone is not a
			// valid identifier.
			interp, err := p.interpolatedIdentifier()
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(interp)

		case ch == '\n' || ch == '\r' || ch == '\f':
			if p.indented && len(brackets) == 0 {
				break loop
			}
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)

		case ch == '!' || ch == ';' || ch == '{' || ch == '}':
			break loop

		case ch == 'u' || ch == 'U':
			beforeUrl := p.scanner.State()
			ident, err := p.identifier(false, false)
			if err != nil {
				return nil, err
			}
			// url-prefix is not standard CSS, but the old @document rule
			// accepted it, so it stays supported for backwards
			// compatibility. Anything else is plain text.
			if ident != "url" && ident != "url-prefix" {
				buffer.Write(ident)
				continue
			}
			// Fall back to a single char when the text is not really a
			// URL body, so the loop re-reads it as ordinary content.
			if contents, err := p.tryUrlContents(beforeUrl, ident, false); err != nil {
				return nil, err
			} else if contents != nil {
				buffer.AddInterpolation(contents)
			} else {
				p.scanner.SetState(beforeUrl)
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
			}

		case ch == '(' || ch == '[':
			bracket, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(bracket)
			opp, err := util.Opposite(bracket)
			if err != nil {
				return nil, p.scanner.Error(err.Error(), -1, 0)
			}
			brackets = append(brackets, opp)

		case ch == ')' || ch == ']':
			if len(brackets) == 0 {
				return nil, p.scanner.Error(fmt.Sprintf(`Unexpected "%c".`, ch), -1, 0)
			}
			expected := brackets[len(brackets)-1]
			brackets = brackets[:len(brackets)-1]
			if err := p.expectChar(expected); err != nil {
				return nil, err
			}
			buffer.WriteCharCode(expected)

		case ch < 0:
			break loop

		case p.lookingAtIdentifier(nil):
			ident, err := p.identifier(false, false)
			if err != nil {
				return nil, err
			}
			buffer.Write(ident)

		default:
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
		}
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	interp, err := buffer.Interpolation(span)
	if err != nil {
		return nil, err
	}
	return interp, nil
}

// declarationValueOpts mirrors the named parameters of Dart's
// _interpolatedDeclarationValue.
type declarationValueOpts struct {
	// allowEmpty fails with "Expected token." when the value holds nothing.
	allowEmpty bool
	// allowSemicolon keeps top-level semicolons in the output instead of
	// stopping before them.
	allowSemicolon bool
	// allowColon keeps top-level colons in the output; when false the scan
	// stops before them.
	allowColon bool
	// allowOpenBrace keeps "{" in the output; when false the scan stops
	// before opening braces.
	allowOpenBrace bool
	// endAfterOf stops after consuming a top-level "of" identifier (grid
	// syntax).
	endAfterOf bool
	// silentComments parses "//" as silent comments; when false the slashes
	// are preserved as text.
	silentComments bool
	// consumeNewlines lets the indented syntax fold newlines into whitespace.
	// Set it only where a statement cannot end.
	consumeNewlines bool
}

// declarationValueOptsDefaults returns the Dart-matching defaults for
// interpolatedDeclarationValue: colons, braces, and silent comments stay
// enabled unless the caller opts out.
func declarationValueOptsDefaults() declarationValueOpts {
	return declarationValueOpts{
		allowColon:     true,
		allowOpenBrace: true,
		silentComments: true,
	}
}

// interpolatedDeclarationValue consumes tokens until a top-level ";", ")",
// "]", or "}" and returns them as interpolation.
//
// Unlike a plain declaration value this pass keeps interpolation, and unlike
// almostAnyValue it decodes escapes, tracks all three bracket kinds,
// collapses whitespace and newlines, and honors its stop-condition options.
// Most edits here should be mirrored in the base-parser declarationValue
// pass, which shares the logic.
func (p *StylesheetParser) interpolatedDeclarationValue(opts declarationValueOpts) (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}

	var brackets []int
	wroteNewline := false

loop:
	for {
		ch := p.scanner.PeekChar(0)
		switch {
		case ch == '\\':
			s, err := p.escape(true)
			if err != nil {
				return nil, err
			}
			buffer.Write(s)
			wroteNewline = false

		case ch == '\'' || ch == '"':
			interp, err := p.interpolatedStringToken()
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(interp)
			wroteNewline = false

		case ch == '/':
			switch p.scanner.PeekChar(1) {
			case '*':
				text, err := p.rawText(func() error { return p.loudComment() })
				if err != nil {
					return nil, err
				}
				buffer.Write(text)
				wroteNewline = false
			case '/':
				if opts.silentComments {
					if _, err := p.silentComment(); err != nil {
						return nil, err
					}
					wroteNewline = false
				} else {
					c, err := p.readChar()
					if err != nil {
						return nil, err
					}
					buffer.WriteCharCode(c)
					wroteNewline = false
				}
			default:
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
				wroteNewline = false
			}

		case ch == '#' && p.scanner.PeekChar(1) == '{':
			// Full interpolated identifier again: "#{...}--1" needs the
			// identifier path since "--1" is not valid on its own.
			interp, err := p.interpolatedIdentifier()
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(interp)
			wroteNewline = false

		case (ch == ' ' || ch == '\t') && !wroteNewline && util.IsWhitespace(p.scanner.PeekChar(1)):
			// Fold runs of blank characters into one, unless they follow a
			// newline where the blanks are indentation.
			if _, err := p.readChar(); err != nil {
				return nil, err
			}

		case ch == ' ' || ch == '\t':
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)

		case (ch == '\n' || ch == '\r' || ch == '\f') && p.indented && !opts.consumeNewlines && len(brackets) == 0:
			break loop

		case ch == '\n' || ch == '\r' || ch == '\f':
			// Collapse a run of newlines into a single line break.
			if !util.IsNewline(p.scanner.PeekChar(-1)) {
				buffer.Writeln("")
			}
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			wroteNewline = true

		case ch == '{' && !opts.allowOpenBrace:
			break loop

		case ch == '(' || ch == '{' || ch == '[':
			bracket, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(bracket)
			opp, err := util.Opposite(bracket)
			if err != nil {
				return nil, p.scanner.Error(err.Error(), -1, 0)
			}
			brackets = append(brackets, opp)
			wroteNewline = false

		case ch == ')' || ch == '}' || ch == ']':
			if len(brackets) == 0 {
				break loop
			}
			expected := brackets[len(brackets)-1]
			brackets = brackets[:len(brackets)-1]
			if err := p.expectChar(expected); err != nil {
				return nil, err
			}
			buffer.WriteCharCode(expected)
			wroteNewline = false

		case ch == ';':
			if !opts.allowSemicolon && len(brackets) == 0 {
				break loop
			}
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
			wroteNewline = false

		case ch == ':':
			if !opts.allowColon && len(brackets) == 0 {
				break loop
			}
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
			wroteNewline = false

		case ch == 'u' || ch == 'U':
			beforeUrl := p.scanner.State()
			ident, err := p.identifier(false, false)
			if err != nil {
				return nil, err
			}
			if ident != "url" && ident != "url-prefix" {
				buffer.Write(ident)
				wroteNewline = false
				continue
			}
			if contents, err := p.tryUrlContents(beforeUrl, ident, false); err != nil {
				return nil, err
			} else if contents != nil {
				buffer.AddInterpolation(contents)
			} else {
				p.scanner.SetState(beforeUrl)
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
			}
			wroteNewline = false

		case ch == 'o' || ch == 'O':
			// Grid syntax ends the value right after a top-level "of".
			if opts.endAfterOf && len(brackets) == 0 {
				of, err := p.rawText(func() error { _, err := p.scanIdentifier("of", false); return err })
				if err != nil {
					return nil, err
				}
				if of != "" {
					buffer.Write(of)
					break loop
				}
			}
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
			wroteNewline = false

		case ch < 0:
			break loop

		case p.lookingAtIdentifier(nil):
			ident, err := p.identifier(false, false)
			if err != nil {
				return nil, err
			}
			buffer.Write(ident)
			wroteNewline = false

		default:
			c, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(c)
			wroteNewline = false
		}
	}

	// A dangling opener reports the missing closer at the end of input.
	if len(brackets) > 0 {
		if err := p.expectChar(brackets[len(brackets)-1]); err != nil {
			return nil, err
		}
	}
	if !opts.allowEmpty && buffer.IsEmpty() {
		return nil, p.scanner.Error("Expected token.", -1, 0)
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	interp, err := buffer.Interpolation(span)
	if err != nil {
		return nil, err
	}
	return interp, nil
}
