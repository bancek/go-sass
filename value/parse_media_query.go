// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/media_query.dart

import (
	"strings"
)

// CssMediaQueryParser parses resolved @media queries from plain CSS text,
// matching Dart's MediaQueryParser in media_query.dart.
//
// Each query is either a parenthesized condition with an optional and/or
// tail, or a not/only/<type> prefix form. The query shape here is somewhat
// duplicated in the stylesheet parser's interpolated _mediaQuery, which
// writes into an interpolation buffer instead; changes to the shape should
// be mirrored there and vice versa. Newline handling is irrelevant here:
// newlines are always consumed.
type CssMediaQueryParser struct {
	Parser
}

// Parse consumes a comma-separated list of media queries, failing on
// trailing input once the queries no longer match (a stray tail such as
// "and (...)" after "not (...)" belongs to the condition, not to a new
// query). Matches Dart's MediaQueryParser.parse.
func (p *CssMediaQueryParser) Parse() ([]*CssMediaQuery, error) {
	var queries []*CssMediaQuery
	err := p.wrapSpanFormatException(func() error {
		for {
			if err := p._whitespace(); err != nil {
				return err
			}
			q, err := p._mediaQuery()
			if err != nil {
				return err
			}
			queries = append(queries, q)
			if err := p._whitespace(); err != nil {
				return err
			}
			if !p.scanner.ScanChar(',') {
				break
			}
		}
		return p.scanner.ExpectDone()
	})
	if err != nil {
		return nil, err
	}
	return queries, nil
}

// _mediaQuery consumes a single media query: a parenthesized condition with
// an optional and/or tail, or a not/only/<type> prefix form such as
// "@media screen and ..." or "@media only screen and ...". A "not" followed
// by an identifier is a modifier ("not screen"); followed by "(" it wraps
// the condition ("(not (color))"). Matches Dart's
// MediaQueryParser._mediaQuery.
func (p *CssMediaQueryParser) _mediaQuery() (*CssMediaQuery, error) {
	if p.scanner.PeekChar(0) == '(' {
		cond, err := p._mediaInParens()
		if err != nil {
			return nil, err
		}
		conditions := []string{cond}
		if err := p._whitespace(); err != nil {
			return nil, err
		}

		conjunction := true
		ok, err := p.scanIdentifier("and", false)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := p.expectWhitespace(false); err != nil {
				return nil, err
			}
			seq, err := p._mediaLogicSequence("and")
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, seq...)
		} else {
			ok, err := p.scanIdentifier("or", false)
			if err != nil {
				return nil, err
			}
			if ok {
				if err := p.expectWhitespace(false); err != nil {
					return nil, err
				}
				conjunction = false
				seq, err := p._mediaLogicSequence("or")
				if err != nil {
					return nil, err
				}
				conditions = append(conditions, seq...)
			}
		}

		result, err := NewCssMediaQueryCondition(conditions, &conjunction)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	var modifier, typ *string
	identifier1, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(identifier1, "not") {
		if err := p.expectWhitespace(false); err != nil {
			return nil, err
		}
		if !p.lookingAtIdentifier(nil) {
			// For example, "@media not (...) {"
			cond, err := p._mediaInParens()
			if err != nil {
				return nil, err
			}
			condFull := "(not " + cond + ")"
			result, err := NewCssMediaQueryCondition([]string{condFull}, nil)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	if err := p._whitespace(); err != nil {
		return nil, err
	}
	if !p.lookingAtIdentifier(nil) {
		// For example, "@media screen {"
		return NewCssMediaQueryType(&identifier1, nil, nil), nil
	}

	identifier2, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(identifier2, "and") {
		if err := p.expectWhitespace(false); err != nil {
			return nil, err
		}
		// For example, "@media screen and ..."
		typ = &identifier1
	} else {
		if err := p._whitespace(); err != nil {
			return nil, err
		}
		modifier = &identifier1
		typ = &identifier2
		ok, err := p.scanIdentifier("and", false)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := p.expectWhitespace(false); err != nil {
				return nil, err
			}
			// For example, "@media only screen and ..."
		} else {
			// For example, "@media only screen {"
			return NewCssMediaQueryType(typ, modifier, nil), nil
		}
	}

	ok, err := p.scanIdentifier("not", false)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := p.expectWhitespace(false); err != nil {
			return nil, err
		}
		// For example, "@media screen and not (...) {"
		cond, err := p._mediaInParens()
		if err != nil {
			return nil, err
		}
		condFull := "(not " + cond + ")"
		return NewCssMediaQueryType(typ, modifier, []string{condFull}), nil
	}

	seq, err := p._mediaLogicSequence("and")
	if err != nil {
		return nil, err
	}
	return NewCssMediaQueryType(typ, modifier, seq), nil
}

// _mediaLogicSequence consumes one or more <media-in-parens> expressions
// separated by operator and returns them. Matches Dart's
// MediaQueryParser._mediaLogicSequence.
func (p *CssMediaQueryParser) _mediaLogicSequence(operator string) ([]string, error) {
	var result []string
	for {
		cond, err := p._mediaInParens()
		if err != nil {
			return nil, err
		}
		result = append(result, cond)
		if err := p._whitespace(); err != nil {
			return nil, err
		}
		ok, err := p.scanIdentifier(operator, false)
		if err != nil {
			return nil, err
		}
		if !ok {
			return result, nil
		}
		if err := p.expectWhitespace(false); err != nil {
			return nil, err
		}
	}
}

// _mediaInParens consumes a <media-in-parens> expression and returns it,
// parentheses included. The inner value parses as an unrestricted
// declaration value, so Sass expressions survive verbatim for later
// evaluation. Matches Dart's MediaQueryParser._mediaInParens.
func (p *CssMediaQueryParser) _mediaInParens() (string, error) {
	if err := p.expectCharName('(', "media condition in parentheses"); err != nil {
		return "", err
	}
	value, err := p.declarationValue(false)
	if err != nil {
		return "", err
	}
	r := "(" + value + ")"
	if err := p.expectChar(')'); err != nil {
		return "", err
	}
	return r, nil
}

// _whitespace consumes whitespace; the consumeNewlines value is not
// relevant for this parser (always true).
func (p *CssMediaQueryParser) _whitespace() error {
	return p.whitespace(true)
}
