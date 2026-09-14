// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/parser.dart

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// Parser is the abstract base class for all Sass parsers.
//
// It provides utility methods and shared token parsing for the per-syntax
// parsers. Unless a method says otherwise, a parse method reports failure
// by returning a SassFormatException error.
type Parser struct {
	scanner          *sasscommon.SpanScanner
	interpolationMap *InterpolationMap // nil unless parsing interpolated source

	// Set by subclasses to override whitespaceWithoutComments behavior.
	whitespaceWithoutCommentsFn func(consumeNewlines bool)
	// Set by subclasses to override silentComment behavior.
	silentCommentFn func() (bool, error)
}

// NewParser creates a new parser for the given contents.
func NewParser(contents []byte, url *url.URL, interpolationMap *InterpolationMap) *Parser {
	return &Parser{
		scanner:          sasscommon.NewSpanScanner(contents, url),
		interpolationMap: interpolationMap,
	}
}

// --- Static helpers ---

// ParseIdentifier parses text as a CSS identifier and returns the result.
//
// It returns a SassFormatException error if parsing fails, including when
// trailing input remains after the identifier.
func ParseIdentifier(text string) (string, error) {
	p := NewParser([]byte(text), nil, nil)
	return p._parseIdentifier()
}

// IsIdentifier returns whether text is a valid CSS identifier.
func IsIdentifier(text string) bool {
	_, err := ParseIdentifier(text)
	return err == nil
}

// IsVariableDeclarationLike returns whether text starts like a variable declaration.
//
// Everything after the `:` is ignored.
func IsVariableDeclarationLike(text string) (bool, error) {
	p := NewParser([]byte(text), nil, nil)
	return p._isVariableDeclarationLike()
}

func (p *Parser) _isVariableDeclarationLike() (bool, error) {
	// A `$`, an identifier start, an identifier, optional whitespace, then
	// `:` — just a shape probe, so inner parse failures still mean "no".
	if !p.scanner.ScanChar('$') {
		return false, nil
	}
	if !p.lookingAtIdentifier(nil) {
		return false, nil
	}

	if _, err := p.identifier(false, false); err != nil {
		return false, nil
	}
	if err := p.whitespace(true); err != nil {
		return false, nil
	}
	return p.scanner.ScanChar(':'), nil
}

// _parseIdentifier runs identifier parsing plus an end-of-input check
// inside span-format-exception wrapping, so callers get SassFormatException
// errors with adjusted spans.
func (p *Parser) _parseIdentifier() (string, error) {
	var result string
	err := p.wrapSpanFormatException(func() error {
		var err error
		result, err = p.identifier(false, false)
		if err != nil {
			return err
		}
		return p.scanner.ExpectDone()
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// --- Scanner helper wrappers ---
//
// These are thin delegates to the span scanner, kept so parser code reads
// at one level of abstraction and span bookkeeping stays in one place.

func (p *Parser) readChar() (int, error) {
	return p.scanner.ReadChar()
}

func (p *Parser) setPosition(pos int) error {
	return p.scanner.SetPosition(pos)
}

func (p *Parser) expectChar(ch int) error {
	return p.scanner.ExpectChar(ch)
}

// expectCharName expects a specific character, using [name] in the error
// message instead of the default character representation.
func (p *Parser) expectCharName(ch int, name string) error {
	if p.scanner.ScanChar(ch) {
		return nil
	}
	return p.scanner.Error(fmt.Sprintf("expected %s.", name), -1, 0)
}

func (p *Parser) expect(str string) error {
	return p.scanner.Expect(str)
}

// --- Whitespace ---

// whitespace consumes whitespace, including any comments.
//
// When consumeNewlines is true the indented syntax also consumes newlines
// as whitespace. Only set it where a statement cannot end.
func (p *Parser) whitespace(consumeNewlines bool) error {
	for {
		if err := p.whitespaceWithoutComments(consumeNewlines); err != nil {
			return err
		}
		hasComment, err := p.scanComment()
		if err != nil {
			return err
		}
		if !hasComment {
			break
		}
	}
	return nil
}

// Consumes whitespace, but not comments.
//
// If [consumeNewlines] is `true`, the indented syntax will consume newlines
// as whitespace. It should only be set to `true` in positions when a
// statement can't end.
//
// The parameter is ignored by this base implementation; only the indented
// syntax override respects it.
func (p *Parser) whitespaceWithoutComments(consumeNewlines bool) error {
	if p.whitespaceWithoutCommentsFn != nil {
		p.whitespaceWithoutCommentsFn(consumeNewlines)
		return nil
	}
	for !p.scanner.IsDone() && util.IsWhitespace(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	return nil
}

// spaces consumes spaces and tabs, but not newlines or comments.
func (p *Parser) spaces() error {
	for !p.scanner.IsDone() && util.IsSpaceOrTab(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	return nil
}

// scanComment consumes and ignores a comment if one is ahead.
//
// It returns whether a comment was consumed. `//` goes through the
// subclass silent-comment hook (the plain-CSS parser rejects it there);
// `/*` is always consumed as a loud comment.
func (p *Parser) scanComment() (bool, error) {
	if p.scanner.PeekChar(0) != '/' {
		return false, nil
	}
	switch p.scanner.PeekChar(1) {
	case '/':
		if p.silentCommentFn != nil {
			return p.silentCommentFn()
		}
		return p.silentComment()
	case '*':
		if err := p.loudComment(); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}

// silentComment consumes and ignores a single silent (Sass-style) comment,
// not including the trailing newline.
//
// It returns whether the comment was consumed.
func (p *Parser) silentComment() (bool, error) {
	if err := p.expect("//"); err != nil {
		return false, err
	}
	for !p.scanner.IsDone() && !util.IsNewline(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return false, err
		}
	}
	return true, nil
}

// loudComment consumes and ignores a loud (CSS-style) comment.
//
// The inner star-skipping loop tolerates runs of `*` so sequences like
// `**/` close the comment exactly once.
func (p *Parser) loudComment() error {
	if err := p.expect("/*"); err != nil {
		return err
	}
	for {
		next, err := p.readChar()
		if err != nil {
			return err
		}
		if next != '*' {
			continue
		}
		for {
			next, err = p.readChar()
			if err != nil {
				return err
			}
			if next != '*' {
				break
			}
		}
		if next == '/' {
			break
		}
	}
	return nil
}

// expectWhitespace works like whitespace, but reports an error when no
// whitespace (or comment) is consumed.
//
// When consumeNewlines is true the indented syntax also accepts newlines.
// Only set it where a statement cannot end.
func (p *Parser) expectWhitespace(consumeNewlines bool) error {
	if p.scanner.IsDone() {
		return p.scanner.Error("Expected whitespace.", -1, 0)
	}
	if !util.IsWhitespace(p.scanner.PeekChar(0)) {
		hasComment, err := p.scanComment()
		if err != nil {
			return err
		}
		if !hasComment {
			return p.scanner.Error("Expected whitespace.", -1, 0)
		}
	}
	return p.whitespace(consumeNewlines)
}

// --- Identifiers ---

// identifier consumes a plain CSS identifier.
//
// When normalize is true underscores convert to hyphens. When unit is true
// a `-` followed by a digit or dot is left unconsumed, so `1px-2px` reads
// as a subtraction rather than the unit `px-2px`.
//
// NOTE: this logic is largely duplicated in interpolatedIdentifier on the
// stylesheet parser. Most changes here should be mirrored there.
func (p *Parser) identifier(normalize bool, unit bool) (string, error) {
	var sb strings.Builder

	if p.scanner.ScanChar('-') {
		sb.WriteByte('-')
		if p.scanner.ScanChar('-') {
			sb.WriteByte('-')
			if err := p._identifierBody(&sb, normalize, unit); err != nil {
				return "", err
			}
			return sb.String(), nil
		}
	}

	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return "", p.scanner.Error("Expected identifier.", -1, 0)
	case ch == '_' && normalize:
		_, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteByte('-')
	case util.IsNameStart(ch):
		c, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteRune(rune(c))
	case ch == '\\':
		s, err := p.escape(true)
		if err != nil {
			return "", err
		}
		sb.WriteString(s)
	default:
		return "", p.scanner.Error("Expected identifier.", -1, 0)
	}

	if err := p._identifierBody(&sb, normalize, unit); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// identifierBody consumes a chunk of a plain CSS identifier after the
// name start, and reports an error when nothing was consumed.
func (p *Parser) identifierBody() (string, error) {
	var sb strings.Builder
	if err := p._identifierBody(&sb, false, false); err != nil {
		return "", err
	}
	if sb.Len() == 0 {
		return "", p.scanner.Error("Expected identifier body.", -1, 0)
	}
	return sb.String(), nil
}

func (p *Parser) _identifierBody(sb *strings.Builder, normalize bool, unit bool) error {
	for {
		switch ch := p.scanner.PeekChar(0); {
		case ch < 0:
			return nil
		case ch == '-' && unit:
			// Disallow `-` followed by a dot or a digit in units.
			next := p.scanner.PeekChar(1)
			if next == '.' || util.IsDigit(next) {
				return nil
			}
			_, err := p.readChar()
			if err != nil {
				return err
			}
			sb.WriteByte('-')
		case ch == '_' && normalize:
			_, err := p.readChar()
			if err != nil {
				return err
			}
			sb.WriteByte('-')
		case util.IsName(ch):
			c, err := p.readChar()
			if err != nil {
				return err
			}
			sb.WriteRune(rune(c))
		case ch == '\\':
			s, err := p.escape(false)
			if err != nil {
				return err
			}
			sb.WriteString(s)
		default:
			return nil
		}
	}
}

// --- String ---

// string consumes a plain CSS string and returns its parsed contents —
// without quotes and with escapes resolved.
//
// NOTE: this logic is largely duplicated in the stylesheet parser's
// interpolated-string helpers. Most changes here should be mirrored there.
func (p *Parser) string() (string, error) {
	quote, err := p.readChar()
	if err != nil {
		return "", err
	}
	if quote != '\'' && quote != '"' {
		return "", p.scanner.Error("Expected string.", p.scanner.Position()-1, 0)
	}
	var sb strings.Builder
	for {
		switch next := p.scanner.PeekChar(0); {
		case next == quote:
			_, err := p.readChar()
			if err != nil {
				return "", err
			}
			return sb.String(), nil
		case next < 0 || util.IsNewline(next):
			return "", p.scanner.Error(fmt.Sprintf("Expected %c.", rune(quote)), -1, 0)
		case next == '\\':
			// A backslash followed by a newline is a line continuation
			// (both characters dropped); any other escape resolves to
			// the character it represents.
			if util.IsNewline(p.scanner.PeekChar(1)) {
				_, err := p.readChar()
				if err != nil {
					return "", err
				}
				_, err = p.readChar()
				if err != nil {
					return "", err
				}
			} else {
				ch, err := p.escapeCharacter()
				if err != nil {
					return "", err
				}
				sb.WriteRune(rune(ch))
			}
		default:
			c, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteRune(rune(c))
		}
	}
}

// --- Numbers ---

// naturalNumber consumes and returns a natural number (a non-negative
// integer) as a float64.
//
// Scientific notation is not supported.
func (p *Parser) naturalNumber() (float64, error) {
	first, err := p.readChar()
	if err != nil {
		return 0, err
	}
	if !util.IsDigit(first) {
		return 0, p.scanner.Error("Expected digit.", p.scanner.Position()-1, 0)
	}
	number := float64(util.AsDecimal(first))
	for util.IsDigit(p.scanner.PeekChar(0)) {
		number *= 10
		c, err := p.readChar()
		if err != nil {
			return 0, err
		}
		number += float64(util.AsDecimal(c))
	}
	return number, nil
}

// --- Declaration value ---

// declarationValue consumes tokens until a top-level `;`, `)`, `]` or `}`
// and returns their contents as a string.
//
// Bracket depth is tracked so separators inside brackets are kept, and an
// unclosed bracket is reported at the end. When allowEmpty is false (the
// default) at least one token is required.
//
// NOTE: this logic is largely duplicated in the stylesheet parser's
// interpolated-declaration-value helper. Most changes here should be
// mirrored there.
func (p *Parser) declarationValue(allowEmpty bool) (string, error) {
	var sb strings.Builder
	brackets := make([]int, 0)
	wroteNewline := false
	for {
		next := p.scanner.PeekChar(0)
		switch {
		case next < 0:
			goto done

		case next == '\\':
			s, err := p.escape(true)
			if err != nil {
				return sb.String(), err
			}
			sb.WriteString(s)
			wroteNewline = false

		case next == '"' || next == '\'':
			text, err := p.rawText(func() error { _, err := p.string(); return err })
			if err != nil {
				return sb.String(), err
			}
			sb.WriteString(text)
			wroteNewline = false

		case next == '/':
			// A loud comment is preserved verbatim as declaration text;
			// a lone `/` is an ordinary token character.
			if p.scanner.PeekChar(1) == '*' {
				text, err := p.rawText(func() error { return p.loudComment() })
				if err != nil {
					return sb.String(), err
				}
				sb.WriteString(text)
			} else {
				c, err := p.readChar()
				if err != nil {
					return sb.String(), err
				}
				sb.WriteRune(rune(c))
			}
			wroteNewline = false

		case next == ' ' || next == '\t':
			// Runs of spaces/tabs collapse to one space, except a run
			// that follows a newline is dropped (the newline already
			// marks the break).
			if wroteNewline || !util.IsWhitespace(p.scanner.PeekChar(1)) {
				sb.WriteByte(' ')
			}
			_, err := p.readChar()
			if err != nil {
				return sb.String(), err
			}

		case next == '\n' || next == '\r' || next == '\f':
			// A newline run folds to a single `\n`; consecutive newline
			// characters do not each emit output.
			if !util.IsNewline(p.scanner.PeekChar(-1)) {
				sb.WriteByte('\n')
			}
			_, err := p.readChar()
			if err != nil {
				return sb.String(), err
			}
			wroteNewline = true

		case next == '(' || next == '{' || next == '[':
			sb.WriteRune(rune(next))
			c, err := p.readChar()
			if err != nil {
				return sb.String(), err
			}
			opp, err := util.Opposite(c)
			if err != nil {
				return sb.String(), err
			}
			brackets = append(brackets, opp)
			wroteNewline = false

		case next == ')' || next == '}' || next == ']':
			// A closer with no open bracket ends the value; otherwise
			// it must match the innermost opener.
			if len(brackets) == 0 {
				goto done
			}
			sb.WriteRune(rune(next))
			if err := p.expectChar(brackets[len(brackets)-1]); err != nil {
				return sb.String(), err
			}
			brackets = brackets[:len(brackets)-1]
			wroteNewline = false

		case next == ';':
			// A top-level semicolon ends the value; nested ones are kept.
			if len(brackets) == 0 {
				goto done
			}
			c, err := p.readChar()
			if err != nil {
				return sb.String(), err
			}
			sb.WriteRune(rune(c))

		case next == 'u' || next == 'U':
			url, err := p.tryUrl()
			if err != nil {
				return sb.String(), err
			}
			if url != "" {
				sb.WriteString(url)
			} else {
				c, err := p.readChar()
				if err != nil {
					return sb.String(), err
				}
				sb.WriteRune(rune(c))
			}
			wroteNewline = false

		default:
			if p.lookingAtIdentifier(nil) {
				s, err := p.identifier(false, false)
				if err != nil {
					return sb.String(), err
				}
				sb.WriteString(s)
			} else {
				c, err := p.readChar()
				if err != nil {
					return sb.String(), err
				}
				sb.WriteRune(rune(c))
			}
			wroteNewline = false
		}
	}
done:
	if len(brackets) > 0 {
		if err := p.expectChar(brackets[len(brackets)-1]); err != nil {
			return sb.String(), err
		}
	}
	if !allowEmpty && sb.Len() == 0 {
		return sb.String(), p.scanner.Error("Expected token.", -1, 0)
	}
	return sb.String(), nil
}

// --- URL ---

// tryUrl consumes a `url()` token when one is ahead and returns its text,
// or an empty string when the input is not a raw URL.
//
// On any shape mismatch the scanner rewinds to the start so the caller can
// re-parse the input as a function expression (matching Ruby Sass). A
// quoted or otherwise non-raw argument is not a raw URL.
//
// NOTE: this logic is largely duplicated in the SCSS parser's URL-contents
// helper. Most changes here should be mirrored there.
func (p *Parser) tryUrl() (string, error) {
	start := p.scanner.State()
	ok, err := p.scanIdentifier("url", false)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	if !p.scanner.ScanChar('(') {
		p.scanner.SetState(start)
		return "", nil
	}
	if err := p.whitespace(true); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("url(")
	for {
		switch ch := p.scanner.PeekChar(0); {
		case ch < 0:
			goto fail

		case ch == '\\':
			s, err := p.escape(false)
			if err != nil {
				return "", err
			}
			sb.WriteString(s)

		case ch == '%' || ch == '&' || ch == '#' || (ch >= '*' && ch <= '~') || ch >= 0x0080:
			c, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteRune(rune(c))

		case util.IsWhitespace(ch):
			// Trailing whitespace inside `url( ... )` is allowed only
			// directly before the closer; anything else backtracks.
			if err := p.whitespace(true); err != nil {
				return "", err
			}
			if p.scanner.PeekChar(0) != ')' {
				goto fail
			}

		case ch == ')':
			_, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteByte(')')
			return sb.String(), nil

		default:
			goto fail
		}
	}

fail:
	p.scanner.SetState(start)
	return "", nil
}

// --- Variable name ---

// variableName consumes a Sass variable name and returns it without the
// leading dollar sign. Underscores normalize to hyphens.
func (p *Parser) variableName() (string, error) {
	if err := p.expectChar('$'); err != nil {
		return "", err
	}
	return p.identifier(true, false)
}

// --- Escape sequences ---

// escape consumes an escape sequence and returns the text that defines it.
//
// When identifierStart is true the result normalizes as though at the
// beginning of an identifier (leading digits re-encode as hex escapes; a
// valid name character decodes to itself). This follows the CSS Syntax
// consume-escaped-code-point algorithm.
func (p *Parser) escape(identifierStart bool) (string, error) {
	start := p.scanner.Position()
	if err := p.expectChar('\\'); err != nil {
		return "", err
	}
	value := 0
	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return "", p.scanner.Error("Expected escape sequence.", -1, 0)
	case util.IsNewline(ch):
		return "", p.scanner.Error("Expected escape sequence.", -1, 0)
	// Up to 6 hex digits, then one optional whitespace character is
	// swallowed as the escape terminator.
	case util.IsHex(ch):
		for range 6 {
			next := p.scanner.PeekChar(0)
			if next < 0 || !util.IsHex(next) {
				break
			}
			value <<= 4
			hex, err := p.readChar()
			if err != nil {
				return "", err
			}
			value += util.AsHex(hex)
		}
		_, err := p.scanCharIf(util.IsWhitespace)
		if err != nil {
			return "", err
		}
	default:
		v, err := p.readChar()
		if err != nil {
			return "", err
		}
		value = v
	}

	if identifierStart && util.IsNameStart(value) || !identifierStart && util.IsName(value) {
		// A valid name character decodes to itself; surrogates and
		// out-of-range code points are an error here (the string-level
		// escape helper substitutes U+FFFD instead).
		if value < 0 || value >= 0xD800 && value <= 0xDFFF || value > unicode.MaxRune {
			return "", p.scanner.Error("Invalid Unicode code point.", start, p.scanner.Position()-start)
		}
		return string(rune(value)), nil
	} else if value <= 0x1F || value == 0x7F || (identifierStart && util.IsDigit(value)) {
		// Control characters, DEL, and (at an identifier start) digits
		// cannot appear literally, so re-encode them as hex escapes.
		var sb strings.Builder
		sb.WriteByte('\\')
		if value > 0xF {
			sb.WriteByte(byte(hexCharFor(value >> 4)))
		}
		sb.WriteByte(byte(hexCharFor(value & 0xF)))
		sb.WriteByte(' ')
		return sb.String(), nil
	} else {
		return string(rune('\\')) + string(rune(value)), nil
	}
}

// escapeCharacter consumes an escape sequence and returns the character
// it represents.
func (p *Parser) escapeCharacter() (int, error) {
	return consumeEscapedCharacter(p.scanner)
}

// --- Character matching ---

// scanCharIf consumes the next character when it matches condition, and
// reports whether anything was consumed.
func (p *Parser) scanCharIf(condition func(int) bool) (bool, error) {
	next := p.scanner.PeekChar(0)
	if !condition(next) {
		return false, nil
	}
	_, err := p.readChar()
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanIdentChar consumes the next character or escape sequence when it
// matches ch, and reports whether anything was consumed.
//
// Matching is case-insensitive unless caseSensitive is true. A failed
// escape match rewinds so the backslash remains unconsumed.
func (p *Parser) scanIdentChar(ch int, caseSensitive bool) (bool, error) {
	matches := func(actual int) bool {
		if caseSensitive {
			return actual == ch
		}
		return util.CharacterEqualsIgnoreCase(ch, actual)
	}

	switch next := p.scanner.PeekChar(0); {
	case next >= 0 && matches(next):
		_, err := p.readChar()
		if err != nil {
			return false, err
		}
		return true, nil
	case next == '\\':
		start := p.scanner.State()
		ch, err := p.escapeCharacter()
		if err != nil {
			p.scanner.SetState(start)
			return false, err
		}
		if matches(ch) {
			return true, nil
		}
		p.scanner.SetState(start)
	}
	return false, nil
}

// expectIdentChar consumes the next character or escape sequence and
// asserts that it matches letter.
//
// Matching is case-insensitive unless caseSensitive is true.
func (p *Parser) expectIdentChar(letter int, caseSensitive bool) error {
	ok, err := p.scanIdentChar(letter, caseSensitive)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return p.scanner.Error(fmt.Sprintf("Expected %q.", string(rune(letter))), p.scanner.Position(), 0)
}

// --- Lookahead ---

// lookingAtNumber returns whether the scanner is immediately before a
// number, following the CSS starts-with-a-number algorithm.
func (p *Parser) lookingAtNumber() bool {
	switch ch := p.scanner.PeekChar(0); {
	case util.IsDigit(ch):
		return true
	case ch == '.':
		next := p.scanner.PeekChar(1)
		return next >= 0 && util.IsDigit(next)
	case ch == '+' || ch == '-':
		next := p.scanner.PeekChar(1)
		if util.IsDigit(next) {
			return true
		}
		if next == '.' {
			next2 := p.scanner.PeekChar(2)
			return next2 >= 0 && util.IsDigit(next2)
		}
		return false
	default:
		return false
	}
}

// lookingAtIdentifier returns whether the scanner is immediately before a
// plain CSS identifier.
//
// When forward is non-nil it looks that many characters ahead instead.
// This follows the CSS would-start-an-identifier check, except every
// backslash is assumed to start an escape.
func (p *Parser) lookingAtIdentifier(forward *int) bool {
	f := 0
	if forward != nil {
		f = *forward
	}
	switch ch := p.scanner.PeekChar(f); {
	case util.IsNameStart(ch) || ch == '\\':
		return true
	case ch == '-':
		next := p.scanner.PeekChar(f + 1)
		return util.IsNameStart(next) || next == '\\' || next == '-'
	default:
		return false
	}
}

// lookingAtIdentifierBody returns whether the scanner is immediately
// before characters that could continue a plain CSS identifier body.
func (p *Parser) lookingAtIdentifierBody() bool {
	next := p.scanner.PeekChar(0)
	return next >= 0 && (util.IsName(next) || next == '\\')
}

// --- Identifier scanning ---

// scanIdentifier consumes an identifier when its name exactly matches text,
// and reports whether it did. A partial match or trailing identifier text
// rewinds to the start.
func (p *Parser) scanIdentifier(text string, caseSensitive bool) (bool, error) {
	if !p.lookingAtIdentifier(nil) {
		return false, nil
	}

	start := p.scanner.State()
	if p._consumeIdentifier(text, caseSensitive) && !p.lookingAtIdentifierBody() {
		return true, nil
	}
	p.scanner.SetState(start)
	return false, nil
}

// matchesIdentifier returns whether an identifier whose name exactly
// matches text sits at the current position, without moving the scan
// pointer.
func (p *Parser) matchesIdentifier(text string, caseSensitive bool) bool {
	if !p.lookingAtIdentifier(nil) {
		return false
	}

	start := p.scanner.State()
	result := p._consumeIdentifier(text, caseSensitive) && !p.lookingAtIdentifierBody()
	p.scanner.SetState(start)
	return result
}

// _consumeIdentifier consumes text as an identifier without checking for
// trailing identifier text. It reports whether the full text was consumed
// and leaves the scan pointer where it stopped.
func (p *Parser) _consumeIdentifier(text string, caseSensitive bool) bool {
	for _, letter := range text {
		ok, err := p.scanIdentChar(int(letter), caseSensitive)
		if err != nil || !ok {
			return false
		}
	}
	return true
}

// expectIdentifier consumes an identifier and asserts that its name exactly
// matches text. When name is empty the error quotes text; otherwise it
// names the expected construct. Trailing identifier text is also an error.
func (p *Parser) expectIdentifier(text string, name string, caseSensitive bool) error {
	if name == "" {
		name = fmt.Sprintf("%q", text)
	}
	start := p.scanner.Position()
	for _, letter := range text {
		ok, err := p.scanIdentChar(int(letter), caseSensitive)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		return p.scanner.Error(fmt.Sprintf("Expected %s.", name), start, 0)
	}
	if !p.lookingAtIdentifierBody() {
		return nil
	}
	return p.scanner.Error(fmt.Sprintf("Expected %s", name), start, 0)
}

// --- Utilities ---

// rawText runs consumer and returns the source text it consumed.
func (p *Parser) rawText(consumer func() error) (string, error) {
	start := p.scanner.Position()
	if err := consumer(); err != nil {
		return "", err
	}
	return p.scanner.Substring(start, nil), nil
}

// spanFrom works like the scanner's SpanFrom, but routes the span through
// the interpolation map when parsing generated (e.g. interpolated) source
// so it points back at the original file.
func (p *Parser) spanFrom(startState sasscommon.LineScannerState) (sasscommon.FileSpan, error) {
	return p.spanFromTo(startState, nil)
}

// spanFromTo works like the scanner's SpanFromTo, but routes the span
// through the interpolation map when one is present. A nil end means the
// span runs to the current position.
func (p *Parser) spanFromTo(startState sasscommon.LineScannerState, end *sasscommon.LineScannerState) (sasscommon.FileSpan, error) {
	var span sasscommon.FileSpan
	if end != nil {
		span = p.scanner.SpanFromTo(startState.Position, end.Position)
	} else {
		span = p.scanner.SpanFrom(startState)
	}
	if p.interpolationMap == nil {
		return span, nil
	}
	return sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
		mapped, err := p.interpolationMap.MapSpan(span)
		if err != nil {
			return nil, err
		}
		if fs, ok := mapped.(sasscommon.FileSpan); ok {
			return fs, nil
		}
		return span, nil
	}), nil
}

// spanFromPosition works like the scanner's SpanFromPosition, but routes
// the span through the interpolation map when one is present. A nil end
// means the span runs to the current position.
func (p *Parser) spanFromPosition(startPos int, end *int) (sasscommon.FileSpan, error) {
	var span sasscommon.FileSpan
	if end != nil {
		span = p.scanner.SpanFromTo(startPos, *end)
	} else {
		span = p.scanner.SpanFromPosition(startPos)
	}
	if p.interpolationMap == nil {
		return span, nil
	}
	return sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
		mapped, err := p.interpolationMap.MapSpan(span)
		if err != nil {
			return nil, err
		}
		if fs, ok := mapped.(sasscommon.FileSpan); ok {
			return fs, nil
		}
		return span, nil
	}), nil
}

// error builds a format error for span. When trace is non-nil it is
// attached as the error's stack trace.
func (p *Parser) error(msg string, span sasscommon.FileSpan, trace error) error {
	err := &sasscommon.SassFormatException{Message: msg, Span: span}
	if trace != nil {
		return sasscommon.ThrowWithTrace(err, trace)
	}
	return err
}

// multiSpanError builds a multi-span format error with a primary label and
// secondary spans.
//
// It ports the Dart parser's MultiSpanSassFormatException throw.
func (p *Parser) multiSpanError(msg string, span sasscommon.FileSpan, primaryLabel string, secondary map[sasscommon.FileSpan]string) error {
	return &sasscommon.MultiSpanSassFormatException{
		Message:      msg,
		Span:         span,
		PrimaryLabel: primaryLabel,
		Secondary:    secondary,
	}
}

// withErrorMessage runs callback and, when it fails with a spanned format
// error, re-reports it under msg while keeping the original span and error
// chain.
func (p *Parser) withErrorMessage(msg string, callback func() error) error {
	err := callback()
	if err != nil {
		if ssfErr, ok := errors.AsType[*sasscommon.SourceSpanFormatException](err); ok {
			return sasscommon.ThrowWithTrace(
				&sasscommon.SourceSpanFormatException{
					Message: msg,
					Span:    ssfErr.Span,
					Source:  ssfErr.Source,
				},
				ssfErr,
			)
		}
		if scanErr, ok := errors.AsType[*sasscommon.ScanError](err); ok {
			return sasscommon.ThrowWithTrace(
				&sasscommon.SourceSpanFormatException{
					Message: msg,
					Span:    scanErr.Span,
				},
				scanErr,
			)
		}
		return err
	}
	return nil
}

// wrapSpanFormatException runs callback and converts scanner-level errors
// into SassFormatException errors.
//
// It ports Parser.wrapSpanFormatException (parser.dart): first remap spans
// through the interpolation map when one is present, then convert both
// single- and multi-span errors, adjusting zero-length "expected ..." spans
// to point at the preceding newline. Anything else passes through untouched.
func (p *Parser) wrapSpanFormatException(callback func() error) error {
	err := callback()
	if err == nil {
		return nil
	}

	// Inner handling: map through interpolation map if available
	if p.interpolationMap != nil {
		if ssfErr, ok := errors.AsType[*sasscommon.SourceSpanFormatException](err); ok {
			mapped := p.interpolationMap.MapException(ssfErr)
			if mapped != ssfErr {
				err = mapped
			}
		} else if scanErr, ok := errors.AsType[*sasscommon.ScanError](err); ok {
			ssfErr := &sasscommon.SourceSpanFormatException{
				Message: scanErr.Message,
				Span:    scanErr.Span,
			}
			mapped := p.interpolationMap.MapException(ssfErr)
			if mapped != ssfErr {
				err = mapped
			}
		}
	}

	// Outer handling: convert to appropriate SassFormatException
	if multiErr, ok := errors.AsType[*sasscommon.MultiSourceSpanFormatError](err); ok {
		span := multiErr.Span
		secondarySpans := multiErr.Secondary
		if hasPrefixIgnoreCase(multiErr.Message, "expected") {
			span, err = p._adjustExceptionSpan(span)
			if err != nil {
				return err
			}
			adjustedSecondary := make(map[sasscommon.FileSpan]string, len(secondarySpans))
			for s, desc := range secondarySpans {
				adjusted, err := p._adjustExceptionSpan(s)
				if err != nil {
					return err
				}
				adjustedSecondary[adjusted] = desc
			}
			secondarySpans = adjustedSecondary
		}
		return sasscommon.ThrowWithTrace(
			&sasscommon.MultiSpanSassFormatException{
				Message:      multiErr.Message,
				Span:         span,
				PrimaryLabel: multiErr.PrimaryLabel,
				Secondary:    secondarySpans,
			},
			multiErr,
		)
	}

	if ssfErr, ok := errors.AsType[*sasscommon.SourceSpanFormatException](err); ok {
		span := ssfErr.Span
		if hasPrefixIgnoreCase(ssfErr.Message, "expected") {
			span, err = p._adjustExceptionSpan(span)
			if err != nil {
				return err
			}
		}
		return sasscommon.ThrowWithTrace(
			&sasscommon.SassFormatException{
				Message: ssfErr.Message,
				Span:    span,
			},
			ssfErr,
		)
	}

	if scanErr, ok := errors.AsType[*sasscommon.ScanError](err); ok {
		span := scanErr.Span
		if hasPrefixIgnoreCase(scanErr.Message, "expected") {
			span, err = p._adjustExceptionSpan(span)
			if err != nil {
				return err
			}
		}
		return sasscommon.ThrowWithTrace(
			&sasscommon.SassFormatException{
				Message: scanErr.Message,
				Span:    span,
			},
			scanErr,
		)
	}

	return err
}

// hasPrefixIgnoreCase returns whether s starts with prefix, comparing
// ASCII case-insensitively (matching Dart's startsWithIgnoreCase, which
// uses ASCII-only character comparison rather than Unicode folding).
func hasPrefixIgnoreCase(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if !asciiCharEqualsIgnoreCase(s[i], prefix[i]) {
			return false
		}
	}
	return true
}

// asciiCharEqualsIgnoreCase compares two bytes ASCII case-insensitively:
// identical bytes match, and letters match when they differ only in the
// case bit.
func asciiCharEqualsIgnoreCase(a, b byte) bool {
	if a == b {
		return true
	}
	if a^b != 0x20 {
		return false
	}
	c := a &^ 0x20
	return c >= 'A' && c <= 'Z'
}

// _adjustExceptionSpan moves a zero-length span back to the preceding
// newline's span when the error position is separated from prior text by
// newlines. Non-empty spans pass through unchanged.
//
// It ports Parser._adjustExceptionSpan (parser.dart).
func (p *Parser) _adjustExceptionSpan(span sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	endLoc, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	if endLoc.Offset > startLoc.Offset {
		return span, nil
	}
	start, err := p._firstNewlineBefore(span, startLoc)
	if err != nil {
		return nil, err
	}
	startLoc2, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	if start.Offset == startLoc2.Offset {
		return span, nil
	}
	file, err := span.File()
	if err != nil {
		return nil, err
	}
	return sasscommon.NewSimpleFileSpan(
		file,
		start.Offset,
		start.Offset,
	), nil
}

// _firstNewlineBefore returns the location of the last newline separating
// location from the preceding non-whitespace character, or location itself
// when there is no such newline.
//
// Pointing "expected" errors at that newline keeps a missing token from
// being blamed on the next closing bracket on a later line. When only
// whitespace precedes location, location is returned unchanged.
//
// It ports Parser._firstNewlineBefore (parser.dart).
func (p *Parser) _firstNewlineBefore(span sasscommon.FileSpan, location sasscommon.SourceLocation) (sasscommon.SourceLocation, error) {
	file, err := span.File()
	if err != nil {
		return location, err
	}
	sourceText := ""
	if file != nil {
		sourceText = file.Text()
	}
	index := location.Offset - 1
	if index < 0 {
		return location, nil
	}
	if index >= len(sourceText) {
		index = len(sourceText) - 1
	}
	lastNewline := -1
	for index >= 0 {
		ch := int(sourceText[index])
		if !util.IsWhitespace(ch) {
			if lastNewline == -1 {
				return location, nil
			}
			return sasscommon.SourceLocation{Offset: lastNewline}, nil
		}
		if util.IsNewline(ch) {
			lastNewline = index
		}
		index--
	}
	return location, nil
}

// --- Free functions ---

// consumeEscapedCharacter consumes an escape sequence from scanner and
// returns the character it represents.
//
// A trailing single whitespace character terminates a hex escape and is
// swallowed. Null, surrogate, and out-of-range values substitute U+FFFD,
// while a newline or end of input after the backslash is an error.
func consumeEscapedCharacter(scanner *sasscommon.SpanScanner) (int, error) {
	if err := scanner.ExpectChar('\\'); err != nil {
		return 0, err
	}
	switch ch := scanner.PeekChar(0); {
	case ch < 0:
		return 0, scanner.Error("Expected escape sequence.", scanner.Position(), 0)
	case util.IsNewline(ch):
		return 0, scanner.Error("Expected escape sequence.", scanner.Position(), 0)
	case util.IsHex(ch):
		value := 0
		for range 6 {
			next := scanner.PeekChar(0)
			if next < 0 || !util.IsHex(next) {
				break
			}
			hex, err := scanner.ReadChar()
			if err != nil {
				return 0, err
			}
			value = (value << 4) + util.AsHex(hex)
		}
		if util.IsWhitespace(scanner.PeekChar(0)) {
			if _, err := scanner.ReadChar(); err != nil {
				return 0, err
			}
		}
		switch {
		case value == 0, value >= 0xD800 && value <= 0xDFFF, value >= util.MaxAllowedCharacter:
			return 0xFFFD, nil
		default:
			return value, nil
		}
	default:
		return scanner.ReadChar()
	}
}
