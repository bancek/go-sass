// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/sass.dart

import (
	"fmt"
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// SassParser parses the indented Sass syntax: newline- and
// indentation-delimited statements rather than SCSS braces and semicolons.
//
// It embeds StylesheetParser with constructor-assigned hooks (statement
// separators, child detection, selector parsing) so the shared stylesheet
// grammar runs under indented rules. Indentation state (the current level,
// the peeked next level, and whether the document uses spaces or tabs) lives
// on this struct; see sassPeekIndentation for the lookahead cache.
//
// Matches Dart: SassParser in lib/src/parse/sass.dart
type SassParser struct {
	StylesheetParser
	// currentIndentation is the indentation level of the statement being
	// parsed. It only advances via sassReadIndentation, which consumes the
	// peeked next-line indentation.
	currentIndentation int
	// nextIndentation caches the indentation level of the next source line
	// after the scanner position, or is nil when it has not been computed
	// yet. A source line is any line that is not entirely whitespace.
	nextIndentation *int
	// nextIndentationEnd caches the scanner position at the start of that
	// next source line, so sassReadIndentation can seek to it.
	nextIndentationEnd *sasscommon.LineScannerState
	// indentSpaces records whether the document indents with spaces (true)
	// or tabs (false), or nil when no indented line has been seen yet.
	// Mixed indentation is an error once this is settled.
	indentSpaces *bool
}

// NewSassParser creates a parser for the indented Sass syntax.
//
// contents is the source text, url the source URL for spans, and
// parseSelectors controls whether style-rule selectors are resolved into
// selector ASTs (false leaves them as raw interpolation for keyframe-style
// contexts). The constructor wires the StylesheetParser hooks for indented
// rules: whitespace consumption that only takes newlines when asked, child
// and statement separation by indentation level, multi-line comma-continued
// rule selectors, indented `@import` arguments, and `@else` matching by
// indentation.
//
// Matches Dart: SassParser constructor in lib/src/parse/sass.dart
func NewSassParser(contents []byte, url *goUrl.URL, parseSelectors bool) *SassParser {
	p := &SassParser{}
	p.Parser = *NewParser(contents, url, nil)
	p.StylesheetParser.parseSelectors = parseSelectors
	p.StylesheetParser.isUseAllowed = true
	p.StylesheetParser.globalVariables = make(map[string]sasscommon.FileSpan)
	p.StylesheetParser.indented = true
	p.StylesheetParser.plainCss = false

	p.Parser.whitespaceWithoutCommentsFn = func(consumeNewlines bool) {
		// This overrides whitespace consumption to only consume newlines when
		// `consumeNewlines` is true.
		for !p.scanner.IsDone() {
			next := p.scanner.PeekChar(0)
			if consumeNewlines {
				if !util.IsWhitespace(next) {
					break
				}
			} else {
				if !util.IsSpaceOrTab(next) {
					break
				}
			}
			if _, err := p.readChar(); err != nil {
				return
			}
		}
	}

	p.StylesheetParser.currentIndentation = func() int {
		return p.currentIndentation
	}

	// styleRuleSelector parses an indented-syntax rule selector: values joined
	// across lines while the accumulated text ends with a comma, so a trailing
	// comma continues the selector on the next line.
	p.StylesheetParser.styleRuleSelector = func() (*Interpolation, error) {
		start := p.scanner.State()
		buffer := &InterpolationBuffer{}
		for {
			val, err := p.almostAnyValue(true)
			if err != nil {
				return nil, err
			}
			if val == nil {
				break
			}
			buffer.AddInterpolation(val)
			buffer.WriteCharCode('\n')
			trailing := strings.TrimRight(buffer.TrailingString(), " \t\r\n")
			if !strings.HasSuffix(trailing, ",") {
				break
			}
			ok, err := p.scanCharIf(func(ch int) bool { return util.IsNewline(ch) })
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
		}
		if buffer.IsEmpty() {
			return nil, nil
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		interp, err := buffer.Interpolation(span)
		if err != nil {
			return nil, p.error(err.Error(), span, nil)
		}
		return interp, nil
	}

	// expectStatementSeparator enforces one statement per line: after an
	// optional trailing semicolon the statement must end at a newline, and
	// anything indented deeper than the current level with no owning
	// statement is an error.
	p.StylesheetParser.expectStatementSeparator = func(name string) error {
		trailingSemicolon, err := p.trySassTrailingSemicolon()
		if err != nil {
			return err
		}
		if !p.StylesheetParser.atEndOfStatement() {
			if err := p.sassExpectNewline(trailingSemicolon); err != nil {
				return err
			}
		}
		peekIndent, err := p.sassPeekIndentation()
		if err != nil {
			return err
		}
		if peekIndent <= p.currentIndentation {
			return nil
		}
		msg := "here."
		if name != "" {
			msg = fmt.Sprintf("beneath a %s.", name)
		}
		return p.scanner.Error("Nothing may be indented "+msg,
			p.nextIndentationEnd.Position, 0)
	}

	// atEndOfStatement reports whether the scanner is at a newline or EOF;
	// indented syntax has no `;` or `{...}` terminators.
	p.StylesheetParser.atEndOfStatement = func() bool {
		next := p.scanner.PeekChar(0)
		return next < 0 || util.IsNewline(next)
	}

	// lookingAtChildren reports whether an indented child block follows:
	// the current statement must end here and the next source line must be
	// indented deeper.
	p.StylesheetParser.lookingAtChildren = func() (bool, error) {
		if !p.StylesheetParser.atEndOfStatement() {
			return false, nil
		}
		peek, err := p.sassPeekIndentation()
		if err != nil {
			return false, err
		}
		return peek > p.currentIndentation, nil
	}

	// scanElse matches `@else` at the same indentation as its `@if`: it peeks
	// the next line's indentation first and rewinds (scanner plus indentation
	// cache) when the line is not an `@else` at ifIndentation.
	p.StylesheetParser.scanElse = func(ifIndentation int) (bool, error) {
		peek, err := p.sassPeekIndentation()
		if err != nil {
			return false, err
		}
		if peek != ifIndentation {
			return false, nil
		}
		start := p.scanner.State()
		startIndentation := p.currentIndentation
		startNextIndentation := p.nextIndentation
		startNextIndentationEnd := p.nextIndentationEnd

		if _, err := p.sassReadIndentation(); err != nil {
			return false, err
		}
		if p.scanner.ScanChar('@') {
			ok, err := p.scanIdentifier("else", true)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}

		p.scanner.SetState(start)
		p.currentIndentation = startIndentation
		p.nextIndentation = startNextIndentation
		p.nextIndentationEnd = startNextIndentationEnd
		return false, nil
	}

	// children consumes the indented block beneath the current statement,
	// requiring consistent indentation across the block.
	p.StylesheetParser.children = func(child func() (Statement, error)) ([]Statement, error) {
		var children = make([]Statement, 0)
		if err := p.sassWhileIndentedLower(func() error {
			parsed, err := p.sassChild(child)
			if err != nil {
				return err
			}
			if parsed != nil {
				children = append(children, parsed)
			}
			return nil
		}); err != nil {
			return children, err
		}
		return children, nil
	}

	// statements consumes the whole document: leading indentation is illegal
	// and every top-level statement must return to indentation zero.
	p.StylesheetParser.statements = func(statement func() (Statement, error)) ([]Statement, error) {
		next := p.scanner.PeekChar(0)
		if next == '\t' || next == ' ' {
			return nil, p.scanner.Error("Indenting at the beginning of the document is illegal.", 0, p.scanner.Position())
		}

		var stmts []Statement
		for !p.scanner.IsDone() {
			child, err := p.sassChild(statement)
			if err != nil {
				return stmts, err
			}
			if child != nil {
				stmts = append(stmts, child)
			}
			if indentation, err := p.sassReadIndentation(); err != nil {
				return stmts, err
			} else if indentation != 0 {
				return stmts, fmt.Errorf("internal error: expected indentation 0, got %d", indentation)
			}
		}
		return stmts, nil
	}

	// importArgumentFn parses indented-syntax `@import` arguments, which run
	// to the end of the line rather than stopping at SCSS delimiters.
	p.StylesheetParser.importArgumentFn = func() (Import, error) {
		imp, err := p.sassImportArgument()
		if err != nil {
			return nil, err
		}
		return imp, nil
	}

	return p
}

// sassChild consumes a child of the current statement.
//
// This handles children allowed at every document level; child is called for
// anything specifically allowed in the caller's context. Empty lines produce
// no statement, `$` starts a variable declaration without namespace, and `/`
// dispatches to silent (`//`) or loud (`/*`) comments.
//
// Matches Dart: SassParser._child
func (p *SassParser) sassChild(child func() (Statement, error)) (Statement, error) {
	switch ch := p.scanner.PeekChar(0); {
	case ch == '\r' || ch == '\n' || ch == '\f':
		return nil, nil
	case ch == '$':
		return p.variableDeclarationWithoutNamespace("", nil)
	case ch == '/':
		switch p.scanner.PeekChar(1) {
		case '/':
			return p.sassSilentComment()
		case '*':
			return p.sassLoudComment()
		default:
			return child()
		}
	default:
		return child()
	}
}

// sassSilentComment consumes an indented-style silent comment.
//
// Consecutive `//` lines at the same indentation merge into one SilentComment
// (a `///` prefix is preserved per line), and deeper-indented lines fold in
// with their relative offset kept. The merged text is remembered as
// lastSilentComment for `//` documentation lookups, and silent comments never
// reach the CSS output.
//
// Matches Dart: SassParser._silentComment
func (p *SassParser) sassSilentComment() (Statement, error) {
	start := p.scanner.State()
	if err := p.expect("//"); err != nil {
		return nil, err
	}
	var sb strings.Builder
	parentIndentation := p.currentIndentation

outer:
	for {
		commentPrefix := "//"
		if p.scanner.ScanChar('/') {
			commentPrefix = "///"
		}

		for {
			sb.WriteString(commentPrefix)

			for i := len(commentPrefix); i < p.currentIndentation-parentIndentation; i++ {
				sb.WriteByte(' ')
			}

			for !p.scanner.IsDone() && !util.IsNewline(p.scanner.PeekChar(0)) {
				ch, err := p.readChar()
				if err != nil {
					return nil, err
				}
				sb.WriteRune(rune(ch))
			}
			sb.WriteByte('\n')

			peek, err := p.sassPeekIndentation()
			if err != nil {
				return nil, err
			}
			if peek < parentIndentation {
				break outer
			}

			if peek == parentIndentation {
				if p.scanner.PeekChar(1+parentIndentation) == '/' &&
					p.scanner.PeekChar(2+parentIndentation) == '/' {
					if _, err := p.sassReadIndentation(); err != nil {
						return nil, err
					}
				}
				break
			}
			if _, err := p.sassReadIndentation(); err != nil {
				return nil, err
			}
		}

		if !(p.scanner.PeekChar(0) == '/' && p.scanner.PeekChar(1) == '/') {
			break
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	last := NewSilentComment(sb.String(), span)
	p.lastSilentComment = last
	return last, nil
}

// sassLoudComment consumes an indented-style loud comment.
//
// Continuation lines are re-prefixed with ` * `, blank lines inside the
// comment are preserved, `#{}` runs interpolation, and an empty first line is
// ignored. After the closing `*/`, further indented comments are consumed for
// backwards compatibility, but any other trailing text on the line is a
// multi-span "extra text" error pointing back at the comment.
//
// Matches Dart: SassParser._loudComment
func (p *SassParser) sassLoudComment() (Statement, error) {
	start := p.scanner.State()
	if err := p.expect("/*"); err != nil {
		return nil, err
	}

	first := true
	buffer := &InterpolationBuffer{}
	buffer.Write("/*")
	parentIndentation := p.currentIndentation

	for {
		if first {
			beginningOfComment := p.scanner.Position()
			if err := p.spaces(); err != nil {
				return nil, err
			}
			if util.IsNewline(p.scanner.PeekChar(0)) {
				if _, err := p.sassReadIndentation(); err != nil {
					return nil, err
				}
				buffer.WriteCharCode(' ')
			} else {
				buffer.Write(p.scanner.Substring(beginningOfComment, nil))
			}
		} else {
			buffer.Writeln("")
			buffer.Write(" * ")
		}
		first = false

		for i := 3; i < p.currentIndentation-parentIndentation; i++ {
			buffer.WriteCharCode(' ')
		}

		for !p.scanner.IsDone() {
			switch ch := p.scanner.PeekChar(0); {
			case ch == '\n' || ch == '\r' || ch == '\f':
				goto endInner
			case ch == '#':
				if p.scanner.PeekChar(1) == '{' {
					expr, span, err := p.singleInterpolation()
					if err != nil {
						return nil, err
					}
					buffer.Add(expr, span)
				} else {
					c, err := p.readChar()
					if err != nil {
						return nil, err
					}
					buffer.WriteCharCode(c)
				}
			case ch == '*':
				if p.scanner.PeekChar(1) == '/' {
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
					span, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					if err := p.whitespace(false); err != nil {
						return nil, err
					}

					for util.IsNewline(p.scanner.PeekChar(0)) {
						peek, err := p.sassPeekIndentation()
						if err != nil {
							return nil, err
						}
						if peek <= parentIndentation {
							break
						}
						for {
							dbl, err := p.sassIsDoubleNewline()
							if err != nil {
								return nil, err
							}
							if !dbl {
								break
							}
							if err := p.sassExpectNewline(false); err != nil {
								return nil, err
							}
						}
						if _, err := p.sassReadIndentation(); err != nil {
							return nil, err
						}
						if err := p.whitespace(false); err != nil {
							return nil, err
						}
					}
					if !p.scanner.IsDone() && !util.IsNewline(p.scanner.PeekChar(0)) {
						errorStart := p.scanner.State()
						for !p.scanner.IsDone() && !util.IsNewline(p.scanner.PeekChar(0)) {
							if _, err := p.readChar(); err != nil {
								return nil, err
							}
						}
						errorSpan, err := p.spanFrom(errorStart)
						if err != nil {
							return nil, err
						}
						return nil, p.multiSpanError("Unexpected text after end of comment",
							errorSpan, "extra text",
							map[sasscommon.FileSpan]string{span: "comment"})
					}
					comment, commentErr := buffer.Interpolation(span)
					if commentErr != nil {
						return nil, p.error(commentErr.Error(), span, nil)
					}
					return NewLoudComment(comment), nil
				}
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
			default:
				c, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(c)
			}
		}
	endInner:

		peek, err := p.sassPeekIndentation()
		if err != nil {
			return nil, err
		}
		if peek <= parentIndentation {
			break
		}

		for {
			dbl, err := p.sassIsDoubleNewline()
			if err != nil {
				return nil, err
			}
			if !dbl {
				break
			}
			if err := p.sassExpectNewline(false); err != nil {
				return nil, err
			}
			buffer.Writeln("")
			buffer.Write(" *")
		}

		if _, err := p.sassReadIndentation(); err != nil {
			return nil, err
		}
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	comment, commentErr := buffer.Interpolation(span)
	if commentErr != nil {
		return nil, p.error(commentErr.Error(), span, nil)
	}
	return NewLoudComment(comment), nil
}

// sassImportArgument parses an indented-syntax `@import` argument.
//
// A `url(...)` call or quoted string delegates to the shared SCSS import
// parsing; otherwise the argument runs bare to the end of the line (comma,
// semicolon, or newline). Plain import URLs become quoted static imports,
// anything else is URL-parsed into a dynamic import.
//
// Matches Dart: SassParser.importArgument
func (p *SassParser) sassImportArgument() (Import, error) {
	ch := p.scanner.PeekChar(0)
	if ch == 'u' || ch == 'U' {
		start := p.scanner.State()
		ok, err := p.scanIdentifier("url", true)
		if err != nil {
			return nil, err
		}
		if ok {
			if p.scanner.ScanChar('(') {
				p.scanner.SetState(start)
				imp, err := p.importArgument()
				if err != nil {
					return nil, err
				}
				return imp, nil
			}
			p.scanner.SetState(start)
		}
	}

	if ch == '\'' || ch == '"' {
		imp, err := p.importArgument()
		if err != nil {
			return nil, err
		}
		return imp, nil
	}

	start := p.scanner.State()
	for {
		next := p.scanner.PeekChar(0)
		if next < 0 || next == ',' || next == ';' || util.IsNewline(next) {
			break
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
	}
	url := p.scanner.Substring(start.Position, nil)
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}

	if p.isPlainImportUrl(url) {
		quoted, err := (&SassString{Text: url, HasQuotes: true}).String()
		if err != nil {
			return nil, err
		}
		return NewStaticImport(NewInterpolationPlain(quoted, span), span, nil), nil
	}

	parsedURL := p.parseImportUrl(url)
	_, parseErr := goUrl.Parse(parsedURL)
	if parseErr != nil {
		if err := p.error("Invalid URL: "+parseErr.Error(), span, nil); err != nil {
			return nil, err
		}
	}
	return NewDynamicImport(parsedURL, span), nil
}

// sassExpectNewline expects and consumes a single newline (`\r\n`, `\n`, or
// form feed).
//
// When trailingSemicolon is set the statement ended with `;`, so the failure
// reports that multiple statements on one line are unsupported in the
// indented syntax instead of the generic "expected newline."
//
// Matches Dart: SassParser._expectNewline
func (p *SassParser) sassExpectNewline(trailingSemicolon bool) error {
	ch := p.scanner.PeekChar(0)
	switch ch {
	case '\r':
		if _, err := p.readChar(); err != nil {
			return err
		}
		if p.scanner.PeekChar(0) == '\n' {
			if _, err := p.readChar(); err != nil {
				return err
			}
		}
	case '\n', '\f':
		if _, err := p.readChar(); err != nil {
			return err
		}
	default:
		msg := "expected newline."
		if trailingSemicolon {
			msg = "multiple statements on one line are not supported in the indented syntax."
		}
		return p.scanner.Error(msg, -1, 0)
	}
	return nil
}

// sassIsDoubleNewline reports whether the scanner is immediately before two
// newlines, covering `\r\n`, lone `\r`, `\n`, and form-feed pairs. Used to
// preserve blank lines inside loud comments.
//
// Matches Dart: SassParser._lookingAtDoubleNewline
func (p *SassParser) sassIsDoubleNewline() (bool, error) {
	ch := p.scanner.PeekChar(0)
	switch ch {
	case '\r':
		next := p.scanner.PeekChar(1)
		if next == '\n' {
			return util.IsNewline(p.scanner.PeekChar(2)), nil
		}
		return next == '\r' || next == '\f', nil
	case '\n', '\f':
		return util.IsNewline(p.scanner.PeekChar(1)), nil
	}
	return false, nil
}

// sassWhileIndentedLower runs body for each statement indented beneath the
// starting line. The first child's indentation fixes the block level;
// siblings at any other deeper level are an "inconsistent indentation" error.
//
// Matches Dart: SassParser._whileIndentedLower
func (p *SassParser) sassWhileIndentedLower(body func() error) error {
	parentIndentation := p.currentIndentation
	var childIndentation *int
	for {
		peek, err := p.sassPeekIndentation()
		if err != nil {
			return err
		}
		if peek <= parentIndentation {
			break
		}
		indentation, err := p.sassReadIndentation()
		if err != nil {
			return err
		}
		if childIndentation == nil {
			childIndentation = &indentation
		}
		if *childIndentation != indentation {
			return p.scanner.Error(
				fmt.Sprintf("Inconsistent indentation, expected %d spaces.", *childIndentation),
				p.scanner.Position()-p.scanner.Column(),
				p.scanner.Column(),
			)
		}
		if err := body(); err != nil {
			return err
		}
	}
	return nil
}

// sassReadIndentation consumes the pending indentation and returns the new
// current level. It computes the peek cache first when empty, then seeks the
// scanner to the cached next-line start and clears the cache.
//
// Matches Dart: SassParser._readIndentation
func (p *SassParser) sassReadIndentation() (int, error) {
	if p.nextIndentation == nil {
		if _, err := p.sassPeekIndentation(); err != nil {
			return 0, err
		}
	}
	p.currentIndentation = *p.nextIndentation
	if p.nextIndentationEnd != nil {
		p.scanner.SetState(*p.nextIndentationEnd)
	}
	p.nextIndentation = nil
	p.nextIndentationEnd = nil
	return p.currentIndentation, nil
}

// sassPeekIndentation returns the indentation level of the next source line
// without consuming input.
//
// The result is cached in nextIndentation/nextIndentationEnd: a missing
// newline where one is required is an error, blank lines are skipped, EOF
// counts as indentation zero, and the first observed indent character settles
// the document's spaces-vs-tabs choice. The scanner is always rewound to the
// entry position.
//
// Matches Dart: SassParser._peekIndentation
func (p *SassParser) sassPeekIndentation() (int, error) {
	if p.nextIndentation != nil {
		return *p.nextIndentation, nil
	}

	if p.scanner.IsDone() {
		zero := 0
		p.nextIndentation = &zero
		state := p.scanner.State()
		p.nextIndentationEnd = &state
		return 0, nil
	}

	start := p.scanner.State()
	ok, err := p.scanCharIf(util.IsNewline)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, p.scanner.Error("Expected newline.", -1, 0)
	}

	var containsTab, containsSpace bool
	var nextIndentation int

	for {
		containsTab = false
		containsSpace = false
		nextIndentation = 0

		for {
			ch := p.scanner.PeekChar(0)
			if ch != ' ' && ch != '\t' {
				break
			}
			if ch == ' ' {
				containsSpace = true
			} else {
				containsTab = true
			}
			nextIndentation++
			if _, err := p.readChar(); err != nil {
				return 0, err
			}
		}

		if p.scanner.IsDone() {
			zero := 0
			p.nextIndentation = &zero
			state := p.scanner.State()
			p.nextIndentationEnd = &state
			p.scanner.SetState(start)
			return 0, nil
		}

		ok, err := p.scanCharIf(util.IsNewline)
		if err != nil {
			return 0, err
		}
		if !ok {
			break
		}
	}

	if err := p.sassCheckIndentationConsistency(containsTab, containsSpace); err != nil {
		return 0, err
	}

	p.nextIndentation = &nextIndentation
	if nextIndentation > 0 && p.indentSpaces == nil {
		b := containsSpace
		p.indentSpaces = &b
	}
	state := p.scanner.State()
	p.nextIndentationEnd = &state
	p.scanner.SetState(start)
	return nextIndentation, nil
}

// sassCheckIndentationConsistency rejects mixed or switched indentation for
// one parsed indent run: tabs and spaces may not mix on a line, and once the
// document has settled on one style the other is an error.
//
// Matches Dart: SassParser._checkIndentationConsistency
func (p *SassParser) sassCheckIndentationConsistency(containsTab, containsSpace bool) error {
	if containsTab {
		if containsSpace {
			return p.scanner.Error(
				"Tabs and spaces may not be mixed.",
				p.scanner.Position()-p.scanner.Column(),
				p.scanner.Column(),
			)
		} else if p.indentSpaces != nil && *p.indentSpaces {
			return p.scanner.Error(
				"Expected spaces, was tabs.",
				p.scanner.Position()-p.scanner.Column(),
				p.scanner.Column(),
			)
		}
	} else if containsSpace && p.indentSpaces != nil && !*p.indentSpaces {
		return p.scanner.Error(
			"Expected tabs, was spaces.",
			p.scanner.Position()-p.scanner.Column(),
			p.scanner.Column(),
		)
	}
	return nil
}

// trySassTrailingSemicolon consumes an optional trailing `;` plus following
// same-line whitespace (comments included) and reports whether one was
// present. The indented syntax tolerates the semicolon but still requires a
// newline after it.
//
// Matches Dart: SassParser._tryTrailingSemicolon
func (p *SassParser) trySassTrailingSemicolon() (bool, error) {
	ok, err := p.scanCharIf(func(ch int) bool { return ch == ';' })
	if err != nil {
		return false, err
	}
	if ok {
		if err := p.whitespace(false); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
