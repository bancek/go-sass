// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (lookahead predicates,
// _withChildren, _urlString, _publicIdentifier, _assertPublic, _addOrInject)

import (
	goUrl "net/url"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassurl"
	"github.com/bancek/go-sass/util"
)

// Scanner-only lookahead predicates plus small stylesheet-parser helpers.

// lookingAtInterpolatedIdentifier reports whether the scanner sits before an
// identifier that may carry interpolation. It follows the CSS
// identifier-start algorithm, except every backslash opens an escape and
// #{...} counts as identifier text.
func (p *StylesheetParser) lookingAtInterpolatedIdentifier() bool {
	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return false
	case util.IsNameStart(ch) || ch == '\\':
		return true
	case ch == '#':
		return p.scanner.PeekChar(1) == '{'
	case ch == '-':
		next := p.scanner.PeekChar(1)
		if next < 0 {
			return false
		}
		if next == '#' && p.scanner.PeekChar(2) == '{' {
			return true
		}
		return util.IsNameStart(next) || next == '\\' || next == '-'
	}
	return false
}

// lookingAtPotentialPropertyHack reports whether the scanner sits before a
// character that could open a legacy property hack: "*prop: val",
// ":prop: val", "#prop: val", or ".prop: val". A "#" only counts when it
// does not open interpolation.
func (p *StylesheetParser) lookingAtPotentialPropertyHack() bool {
	switch ch := p.scanner.PeekChar(0); {
	case ch == ':' || ch == '*' || ch == '.':
		return true
	case ch == '#':
		return p.scanner.PeekChar(1) != '{'
	}
	return false
}

// lookingAtInterpolatedIdentifierBody reports whether the scanner sits
// before characters that could continue a CSS identifier body, including an
// interpolation opener.
func (p *StylesheetParser) lookingAtInterpolatedIdentifierBody() bool {
	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return false
	case util.IsName(ch) || ch == '\\':
		return true
	case ch == '#':
		return p.scanner.PeekChar(1) == '{'
	}
	return false
}

// lookingAtExpression reports whether the scanner sits before something that
// can start a SassScript expression. A "." only counts when it does not open
// "..", and "!" only when it opens "!important"-style text, whitespace, or
// the end of input.
func (p *StylesheetParser) lookingAtExpression() bool {
	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return false
	case ch == '.':
		return p.scanner.PeekChar(1) != '.'
	case ch == '!':
		next := p.scanner.PeekChar(1)
		return next < 0 || next == 'i' || next == 'I' || util.IsWhitespace(next)
	case ch == '(' || ch == '/' || ch == '[' || ch == '\'' || ch == '"' || ch == '#' || ch == '+' || ch == '-' || ch == '\\' || ch == '$' || ch == '&' || ch == '%':
		return true
	case util.IsNameStart(ch) || util.IsDigit(ch):
		return true
	}
	return false
}

// Utilities

// withChildren consumes a block of child statements via child and hands the
// children plus the span from start through the end of the block to create.
// Trailing plain whitespace (comments excluded) is consumed after the block.
func withChildren[T any](p *StylesheetParser, child func() (Statement, error), start sasscommon.LineScannerState, create func([]Statement, sasscommon.FileSpan) (T, error)) (T, error) {
	children, err := p.children(child)
	if err != nil {
		var zero T
		return zero, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		var zero T
		return zero, err
	}
	result, err := create(children, span)
	if err != nil {
		var zero T
		return zero, err
	}
	if err := p.whitespaceWithoutComments(false); err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

// urlString consumes a quoted string holding a valid URL. A parse failure
// reports "Invalid URL: ..." over the span from the opening quote.
func (p *StylesheetParser) urlString() (*goUrl.URL, error) {
	start := p.scanner.State()
	urlStr, err := p.string()
	if err != nil {
		return nil, err
	}
	parsed, err := sassurl.Parse(urlStr)
	if err != nil {
		span, spanErr := p.spanFrom(start)
		if spanErr != nil {
			return nil, spanErr
		}
		return nil, p.error("Invalid URL: "+err.Error(), span, nil)
	}
	return parsed, nil
}

// publicIdentifier consumes an identifier like identifier does but rejects
// names that start a private member.
func (p *StylesheetParser) publicIdentifier() (string, error) {
	start := p.scanner.State()
	result, err := p.identifier(true, false)
	if err != nil {
		return "", err
	}
	span, spanErr := p.spanFrom(start)
	if spanErr != nil {
		return "", spanErr
	}
	if err := p.assertPublic(result, func() sasscommon.FileSpan { return span }); err != nil {
		return "", err
	}
	return result, nil
}

// assertPublic fails when identifier is private (a leading "_" or "-").
// The span callback runs lazily so the success path builds no span.
func (p *StylesheetParser) assertPublic(identifier string, span func() sasscommon.FileSpan) error {
	if util.IsPrivate(identifier) {
		return p.error("Private members can't be accessed from outside their modules.", span(), nil)
	}
	return nil
}

// addOrInject appends expression to the buffer, unwrapping unquoted strings
// to their inner interpolation instead of nesting the expression node.
func (p *StylesheetParser) addOrInject(buffer *InterpolationBuffer, expression Expression) error {
	if se, ok := expression.(*StringExpression); ok && !se.HasQuotes {
		buffer.AddInterpolation(se.Text)
	} else {
		span, err := expression.Span()
		if err != nil {
			return err
		}
		buffer.Add(expression, span)
	}
	return nil
}
