// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/keyframe_selector.dart

import (
	"strings"

	"github.com/bancek/go-sass/util"
)

// KeyframeSelectorParser parses @keyframes block selectors (from, to, and
// percentages such as 50%, including +/decimal/exponent forms), matching
// Dart's KeyframeSelectorParser in keyframe_selector.dart.
//
// Newline handling is irrelevant here: newlines are always consumed.
type KeyframeSelectorParser struct {
	Parser
}

// Parse consumes a comma-separated list of keyframe selectors. An
// identifier parses as "from", else as "to" (with the '"to" or "from"'
// expectation message); anything else parses as a percentage. Matches
// Dart's KeyframeSelectorParser.parse.
func (p *KeyframeSelectorParser) Parse() ([]string, error) {
	var selectors []string
	err := p.wrapSpanFormatException(func() error {
		for {
			if err := p._whitespace(); err != nil {
				return err
			}
			if p.lookingAtIdentifier(nil) {
				ok, err := p.scanIdentifier("from", false)
				if err != nil {
					return err
				}
				if ok {
					selectors = append(selectors, "from")
				} else {
					if err := p.expectIdentifier("to", `"to" or "from"`, false); err != nil {
						return err
					}
					selectors = append(selectors, "to")
				}
			} else {
				s, err := p._percentage()
				if err != nil {
					return err
				}
				selectors = append(selectors, s)
			}
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
	return selectors, nil
}

// _percentage consumes one percentage keyframe selector (50%, with an
// optional + prefix, decimal part, and e/E exponent) and returns its text,
// % included. A missing mantissa reports "Expected number."; a dangling
// exponent reports "Expected digit.". Matches Dart's
// KeyframeSelectorParser._percentage.
func (p *KeyframeSelectorParser) _percentage() (string, error) {
	var sb strings.Builder
	if p.scanner.ScanChar('+') {
		sb.WriteByte('+')
	}

	second := p.scanner.PeekChar(0)
	if !util.IsDigit(second) && second != '.' {
		return "", p.scanner.Error("Expected number.", p.scanner.Position(), 0)
	}

	for util.IsDigit(p.scanner.PeekChar(0)) {
		ch, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte(ch))
	}

	if p.scanner.PeekChar(0) == '.' {
		ch, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte(ch))
		for util.IsDigit(p.scanner.PeekChar(0)) {
			ch, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteByte(byte(ch))
		}
	}

	ok, err := p.scanIdentChar('e', true)
	if err != nil {
		return "", err
	}
	if !ok {
		ok, err = p.scanIdentChar('E', true)
		if err != nil {
			return "", err
		}
	}
	if ok {
		sb.WriteByte('e')
		if ch := p.scanner.PeekChar(0); ch == '+' || ch == '-' {
			c, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteByte(byte(c))
		}
		if !util.IsDigit(p.scanner.PeekChar(0)) {
			return "", p.scanner.Error("Expected digit.", p.scanner.Position(), 0)
		}
		for {
			c, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteByte(byte(c))
			if !util.IsDigit(p.scanner.PeekChar(0)) {
				break
			}
		}
	}

	if err := p.expectChar('%'); err != nil {
		return "", err
	}
	sb.WriteByte('%')
	return sb.String(), nil
}

// _whitespace consumes whitespace; the consumeNewlines value is not
// relevant for this parser (always true).
func (p *KeyframeSelectorParser) _whitespace() error {
	return p.whitespace(true)
}
