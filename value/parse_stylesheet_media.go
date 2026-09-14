// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (media query sections:
// _mediaQueryList, _mediaQuery, _mediaLogicSequence, _mediaOrInterp,
// _mediaInParens, _expressionUntilComparison)

import (
	"strings"
)

// Media queries at stylesheet level.
//
// Unlike the resolved-query parser over plain CSS text, these routines build
// an interpolation that may still hold Sass expressions such as
// `(width: $n)` or range comparisons like `(100px < width < 200px)`. The
// shape here partly mirrors that parser, so changes to the query grammar
// should be kept in sync there.

// mediaQueryList consumes a comma-separated list of media queries.
func (p *StylesheetParser) mediaQueryList() (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}
	for {
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		if err := p.mediaQuery(buffer); err != nil {
			return nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		if !p.scanner.ScanChar(',') {
			break
		}
		buffer.WriteCharCode(',')
		buffer.WriteCharCode(' ')
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

// mediaQuery consumes one media query into the buffer: a parenthesized
// condition with an optional and/or tail, or a type/modifier prefix form such
// as "@media screen", "@media only screen and ...", or "@media not (...)".
func (p *StylesheetParser) mediaQuery(buffer *InterpolationBuffer) error {
	if p.scanner.PeekChar(0) == '(' {
		if err := p.mediaInParens(buffer); err != nil {
			return err
		}
		if err := p.whitespace(false); err != nil {
			return err
		}
		if ok, err := p.scanIdentifier("and", false); err != nil {
			return err
		} else if ok {
			buffer.Write(" and ")
			if err := p.expectWhitespace(false); err != nil {
				return err
			}
			return p.mediaLogicSequence(buffer, "and")
		} else if ok, err := p.scanIdentifier("or", false); err != nil {
			return err
		} else if ok {
			buffer.Write(" or ")
			if err := p.expectWhitespace(false); err != nil {
				return err
			}
			return p.mediaLogicSequence(buffer, "or")
		}
		return nil
	}

	identifier1, err := p.interpolatedIdentifier()
	if err != nil {
		return err
	}
	// A leading "not" followed by something other than an identifier starts
	// a negated condition, as in "@media not (...)".
	if identifier1.AsPlain() != nil && strings.EqualFold(*identifier1.AsPlain(), "not") {
		if err := p.expectWhitespace(false); err != nil {
			return err
		}
		if !p.lookingAtInterpolatedIdentifier() {
			buffer.Write("not ")
			return p.mediaOrInterp(buffer)
		}
	}

	if err := p.whitespace(false); err != nil {
		return err
	}
	buffer.AddInterpolation(identifier1)
	if !p.lookingAtInterpolatedIdentifier() {
		// Bare type with no modifier, as in "@media screen".
		return nil
	}

	buffer.WriteCharCode(' ')
	identifier2, err := p.interpolatedIdentifier()
	if err != nil {
		return err
	}

	if identifier2.AsPlain() != nil && strings.EqualFold(*identifier2.AsPlain(), "and") {
		// "@media screen and ...": the second word was the connective.
		if err := p.expectWhitespace(false); err != nil {
			return err
		}
		buffer.Write(" and ")
	} else {
		if err := p.whitespace(false); err != nil {
			return err
		}
		buffer.AddInterpolation(identifier2)
		if ok, err := p.scanIdentifier("and", false); err != nil {
			return err
		} else if ok {
			// "@media only screen and ...": two modifiers, then the tail.
			if err := p.expectWhitespace(false); err != nil {
				return err
			}
			buffer.Write(" and ")
		} else {
			// "@media only screen": the query ends after the second word.
			return nil
		}
	}

	// Either `IDENTIFIER "and"` or `IDENTIFIER IDENTIFIER "and"` was consumed
	// above; what follows is the condition tail.

	if ok, err := p.scanIdentifier("not", false); err != nil {
		return err
	} else if ok {
		// "@media screen and not (...)".
		if err := p.expectWhitespace(false); err != nil {
			return err
		}
		buffer.Write("not ")
		return p.mediaOrInterp(buffer)
	}

	return p.mediaLogicSequence(buffer, "and")
}

// mediaLogicSequence consumes one or more media-or-interpolation terms joined
// by the given operator and writes them to the buffer.
func (p *StylesheetParser) mediaLogicSequence(buffer *InterpolationBuffer, operator string) error {
	for {
		if err := p.mediaOrInterp(buffer); err != nil {
			return err
		}
		if err := p.whitespace(false); err != nil {
			return err
		}
		if ok, err := p.scanIdentifier(operator, false); err != nil {
			return err
		} else if !ok {
			return nil
		}
		if err := p.expectWhitespace(false); err != nil {
			return err
		}
		buffer.WriteCharCode(' ')
		buffer.Write(operator)
		buffer.WriteCharCode(' ')
	}
}

// mediaOrInterp consumes one media-or-interpolation term: a raw #{...}
// interpolation when the scanner sits on "#", otherwise a parenthesized
// media condition.
func (p *StylesheetParser) mediaOrInterp(buffer *InterpolationBuffer) error {
	if p.scanner.PeekChar(0) == '#' {
		expr, span, err := p.singleInterpolation()
		if err != nil {
			return err
		}
		buffer.Add(expr, span)
	} else {
		if err := p.mediaInParens(buffer); err != nil {
			return err
		}
	}
	return nil
}

// mediaInParens consumes a parenthesized media condition into the buffer:
// nested conditions with an and/or tail, "not" plus a term, a "feature:
// value" declaration, or a range comparison (including the two-sided form).
// Newlines fold as whitespace inside the parens; only plain whitespace may
// follow the closing paren.
func (p *StylesheetParser) mediaInParens(buffer *InterpolationBuffer) error {
	if err := p.expectCharName('(', "media condition in parentheses"); err != nil {
		return err
	}
	buffer.WriteCharCode('(')
	if err := p.whitespace(true); err != nil {
		return err
	}

	if p.scanner.PeekChar(0) == '(' {
		if err := p.mediaInParens(buffer); err != nil {
			return err
		}
		if err := p.whitespace(true); err != nil {
			return err
		}
		if ok, err := p.scanIdentifier("and", false); err != nil {
			return err
		} else if ok {
			buffer.Write(" and ")
			if err := p.expectWhitespace(true); err != nil {
				return err
			}
			if err := p.mediaLogicSequence(buffer, "and"); err != nil {
				return err
			}
		} else if ok, err := p.scanIdentifier("or", false); err != nil {
			return err
		} else if ok {
			buffer.Write(" or ")
			if err := p.expectWhitespace(true); err != nil {
				return err
			}
			if err := p.mediaLogicSequence(buffer, "or"); err != nil {
				return err
			}
		}
	} else if ok, err := p.scanIdentifier("not", false); err != nil {
		return err
	} else if ok {
		buffer.Write("not ")
		if err := p.expectWhitespace(true); err != nil {
			return err
		}
		if err := p.mediaOrInterp(buffer); err != nil {
			return err
		}
	} else {
		expressionBefore, exprErr := p.expressionUntilComparison()
		if exprErr != nil {
			return exprErr
		}
		span, err := expressionBefore.Span()
		if err != nil {
			return err
		}
		buffer.Add(expressionBefore, span)
		if p.scanner.ScanChar(':') {
			if err := p.whitespace(true); err != nil {
				return err
			}
			buffer.WriteCharCode(':')
			buffer.WriteCharCode(' ')
			expressionAfter, exprErr := p._expression(expressionOpts{consumeNewlines: true})
			if exprErr != nil {
				return exprErr
			}
			span, err := expressionAfter.Span()
			if err != nil {
				return err
			}
			buffer.Add(expressionAfter, span)
		} else {
			next := p.scanner.PeekChar(0)
			if next == '<' || next == '>' || next == '=' {
				buffer.WriteCharCode(' ')
				ch, err := p.readChar()
				if err != nil {
					return err
				}
				buffer.WriteCharCode(ch)
				if (next == '<' || next == '>') && p.scanner.ScanChar('=') {
					buffer.WriteCharCode('=')
				}
				buffer.WriteCharCode(' ')
				if err := p.whitespace(true); err != nil {
					return err
				}
				expressionMiddle, exprMidErr := p.expressionUntilComparison()
				if exprMidErr != nil {
					return exprMidErr
				}
				span, err := expressionMiddle.Span()
				if err != nil {
					return err
				}
				buffer.Add(expressionMiddle, span)

				// A second comparison with the same bracket makes the range
				// two-sided, as in "(100px < width < 200px)".
				if (next == '<' || next == '>') && p.scanner.PeekChar(0) == next {
					buffer.WriteCharCode(' ')
					ch, err := p.readChar()
					if err != nil {
						return err
					}
					buffer.WriteCharCode(ch)
					if p.scanner.ScanChar('=') {
						buffer.WriteCharCode('=')
					}
					buffer.WriteCharCode(' ')
					if err := p.whitespace(true); err != nil {
						return err
					}
					expressionAfter, exprAfterErr := p.expressionUntilComparison()
					if exprAfterErr != nil {
						return exprAfterErr
					}
					span, err := expressionAfter.Span()
					if err != nil {
						return err
					}
					buffer.Add(expressionAfter, span)
				}
			}
		}
	}

	if err := p.expectChar(')'); err != nil {
		return err
	}
	if err := p.whitespace(false); err != nil {
		return err
	}
	buffer.WriteCharCode(')')
	return nil
}

// expressionUntilComparison consumes an expression stopping before a
// top-level "<", ">", or a "=" that is not part of "==".
func (p *StylesheetParser) expressionUntilComparison() (Expression, error) {
	return p._expression(expressionOpts{consumeNewlines: true, until: func() bool {
		ch := p.scanner.PeekChar(0)
		if ch == '=' {
			return p.scanner.PeekChar(1) != '='
		}
		return ch == '<' || ch == '>'
	}})
}
