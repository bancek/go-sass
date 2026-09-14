// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/string.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// StringExpression is a string literal.
//
// Matches Dart: StringExpression
type StringExpression struct {
	// Text is the interpolation that, when evaluated, produces the contents
	// of this string. For quoted strings escapes are resolved and quotes
	// excluded here (unlike AsInterpolation); for unquoted strings escapes
	// are left unresolved.
	Text *Interpolation
	// HasQuotes reports whether this string has quotes.
	HasQuotes bool
}

// NewStringExpression creates a string literal from an evaluated-contents
// interpolation.
//
// Matches Dart: StringExpression.new
func NewStringExpression(text *Interpolation, hasQuotes bool) *StringExpression {
	return &StringExpression{Text: text, HasQuotes: hasQuotes}
}

// NewStringExpressionPlain creates a string expression with no
// interpolation.
//
// Matches Dart: StringExpression.plain
func NewStringExpressionPlain(text string, span sasscommon.FileSpan, hasQuotes bool) *StringExpression {
	return &StringExpression{
		Text:      NewInterpolationPlain(text, span),
		HasQuotes: hasQuotes,
	}
}

// QuoteText returns Sass source for a quoted string that, when evaluated,
// will have text as its contents. The quote character is chosen to minimize
// escaping.
//
// Matches Dart: StringExpression.quoteText
func QuoteText(text string) string {
	quote := bestQuote([]string{text})
	var sb strings.Builder
	sb.WriteByte(byte(quote))
	quoteInnerText(text, quote, &sb, true)
	sb.WriteByte(byte(quote))
	return sb.String()
}

// AsInterpolation returns an interpolation that, when evaluated, produces
// the syntax of this string. Unlike Text, this doesn't resolve escapes and
// does include quotes for quoted strings.
//
// If static is true, any `#{` sequences are escaped. If quote is passed, it
// selects the quote mark; otherwise the quote minimizing escapes is used.
//
// Matches Dart: StringExpression.asInterpolation
func (e *StringExpression) AsInterpolation(static bool, quote *int) (*Interpolation, error) {
	if !e.HasQuotes {
		return e.Text, nil
	}

	var strings_ []string
	for _, c := range e.Text.Contents {
		if s, ok := c.(string); ok {
			strings_ = append(strings_, s)
		}
	}
	q := bestQuote(strings_)
	if quote != nil {
		q = *quote
	}
	buf := &InterpolationBuffer{}
	buf.WriteCharCode(q)
	for i, content := range e.Text.Contents {
		switch v := content.(type) {
		case Expression:
			sp, err := e.Text.SpanForElement(i)
			if err != nil {
				return nil, err
			}
			buf.Add(v, sp)
		case string:
			var sb strings.Builder
			quoteInnerText(v, q, &sb, static)
			buf.Write(sb.String())
		}
	}
	buf.WriteCharCode(q)

	span, err := e.Text.Span()
	if err != nil {
		return nil, err
	}
	result, err := buf.Interpolation(span)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Span returns the span of the string's source text.
//
// Matches Dart: StringExpression.span (delegates to text.span)
func (e *StringExpression) Span() (sasscommon.FileSpan, error) {
	return e.Text.Span()
}

func (e *StringExpression) IsExpression()                       {}
func (e *StringExpression) IsSassNode()                         {}
func (e *StringExpression) IsAstNode()                          {}
func (e *StringExpression) SourceInterpolation() *Interpolation { return e.Text }

// String renders the string's syntax, including quotes for quoted strings,
// via AsInterpolation.
//
// Matches Dart: StringExpression.toString
func (e *StringExpression) String() (string, error) {
	interp, err := e.AsInterpolation(false, nil)
	if err != nil {
		return "", err
	}
	return interp.String()
}

// bestQuote returns the code unit of the quote minimizing escapes for
// contents: double quotes unless the text holds a single quote (then double
// is forced) or holds double quotes only (then single).
func bestQuote(contents []string) int {
	containsDoubleQuote := false
	for _, s := range contents {
		for _, r := range s {
			if r == '\'' {
				return '"'
			}
			if r == '"' {
				containsDoubleQuote = true
			}
		}
	}
	if containsDoubleQuote {
		return '\''
	}
	return '"'
}

// quoteInnerText writes the contents of a string without quotes so Sass
// parsing reproduces text: it always escapes the quote character and
// backslashes, renders newlines as the `a` escape (with a disambiguating
// space when the next character is whitespace or hex), and, when static is
// true, escapes `#{` so interpolation stays inert.
func quoteInnerText(text string, quote int, sb *strings.Builder, static bool) {
	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch {
		case ch == '\n' || ch == '\r' || ch == '\f':
			sb.WriteByte('\\')
			sb.WriteByte('a')
			if i != len(text)-1 {
				next := text[i+1]
				if next == ' ' || next == '\t' || next == '\n' || next == '\r' || next == '\f' ||
					(next >= '0' && next <= '9') ||
					(next >= 'a' && next <= 'f') ||
					(next >= 'A' && next <= 'F') {
					sb.WriteByte(' ')
				}
			}
		case ch == '\\':
			sb.WriteByte('\\')
			sb.WriteByte(ch)
		case int(ch) == quote:
			sb.WriteByte('\\')
			sb.WriteByte(ch)
		case ch == '#' && static && i < len(text)-1 && text[i+1] == '{':
			sb.WriteByte('\\')
			sb.WriteByte(ch)
		default:
			sb.WriteByte(ch)
		}
	}
}
