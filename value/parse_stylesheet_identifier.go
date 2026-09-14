// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (interpolatedIdentifier,
// _interpolatedIdentifierBody, singleInterpolation)

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// Interpolated identifiers: plain names plus #{...} splices and escapes.

// interpolatedIdentifier consumes an identifier that may carry interpolation.
func (p *StylesheetParser) interpolatedIdentifier() (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}

	// A "--" lead takes the fast path: custom-property bodies skip the
	// name-start check and go straight to the body loop.
	if p.scanner.ScanChar('-') {
		buffer.WriteCharCode('-')
		if p.scanner.ScanChar('-') {
			buffer.WriteCharCode('-')
			if err := p.interpolatedIdentifierBodyHelper(buffer); err != nil {
				return nil, err
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
	}

	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return nil, p.scanner.Error("Expected identifier.", -1, 0)
	case util.IsNameStart(ch):
		rc, err := p.readChar()
		if err != nil {
			return nil, err
		}
		buffer.WriteCharCode(rc)
	case ch == '\\':
		// Leading escapes may start the name, unlike later ones.
		esc, err := p.escape(true)
		if err != nil {
			return nil, err
		}
		buffer.Write(esc)
	case ch == '#' && p.scanner.PeekChar(1) == '{':
		// Interpolation may open the name, which covers cases like
		// "#{...}--1" where the remainder alone would be invalid.
		expr, span, err := p.singleInterpolation()
		if err != nil {
			return nil, err
		}
		buffer.Add(expr, span)
	default:
		return nil, p.scanner.Error("Expected identifier.", -1, 0)
	}

	if err := p.interpolatedIdentifierBodyHelper(buffer); err != nil {
		return nil, err
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

// interpolatedIdentifierBody consumes the tail of a possibly-interpolated
// CSS identifier once the name start is gone. It fails with "Expected
// identifier body." when nothing follows, so callers must have consumed the
// leading character first.
func (p *StylesheetParser) interpolatedIdentifierBody() (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}
	if err := p.interpolatedIdentifierBodyHelper(buffer); err != nil {
		return nil, err
	}
	if buffer.IsEmpty() {
		return nil, p.scanner.Error("Expected identifier body.", -1, 0)
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

// interpolatedIdentifierBodyHelper parses an identifier body into the given
// buffer: name characters, escapes, and #{...} splices until anything else
// appears.
func (p *StylesheetParser) interpolatedIdentifierBodyHelper(buffer *InterpolationBuffer) error {
	for {
		switch ch := p.scanner.PeekChar(0); {
		case ch < 0:
			return nil
		case ch == '_' || ch == '-' || util.IsAlphanumeric(ch) || ch >= 0x80:
			rc, err := p.readChar()
			if err != nil {
				return err
			}
			buffer.WriteCharCode(rc)
		case ch == '\\':
			esc, err := p.escape(false)
			if err != nil {
				return err
			}
			buffer.Write(esc)
		case ch == '#' && p.scanner.PeekChar(1) == '{':
			expr, span, err := p.singleInterpolation()
			if err != nil {
				return err
			}
			buffer.Add(expr, span)
		default:
			return nil
		}
	}
}

// singleInterpolation consumes one #{...} splice and returns its expression
// plus the span covering the whole "#{}" marker. Plain-CSS stylesheets
// reject interpolation outright.
func (p *StylesheetParser) singleInterpolation() (Expression, sasscommon.FileSpan, error) {
	start := p.scanner.State()
	if err := p.expect("#{"); err != nil {
		span, _ := p.spanFrom(start)
		return nil, span, err
	}
	if err := p.whitespace(true); err != nil {
		span, _ := p.spanFrom(start)
		return nil, span, err
	}
	contents, exprErr := p._expression(expressionOpts{consumeNewlines: true})
	if exprErr != nil {
		span, _ := p.spanFrom(start)
		return nil, span, exprErr
	}
	if err := p.expectChar('}'); err != nil {
		span, _ := p.spanFrom(start)
		return nil, span, err
	}

	if p.plainCss {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, span, err
		}
		if err := p.error("Interpolation isn't allowed in plain CSS.", span, nil); err != nil {
			return nil, span, err
		}
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, span, err
	}
	return contents, span, nil
}
