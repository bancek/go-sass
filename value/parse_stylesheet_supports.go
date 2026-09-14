// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (supports condition sections:
// _importSupportsQuery, _tryImportSupportsFunction, _supportsCondition,
// _supportsConditionInParens, _supportsDeclarationValue, _trySupportsOperation)

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// Supports conditions after `@import` and in `@supports` rules.
//
// Operators (`and`/`or`/`not`) match case-insensitively per the CSS spec.
// The parenthesized form is ambiguous (`Expression ":" Expression` versus an
// interpolated identifier plus an optional value), so the parser tries the
// declaration shape first and backtracks to the operation/anything fallbacks
// on failure.

// importSupportsQuery consumes the contents of a supports() function after an
// @import rule, not including the function name or its parentheses.
func (p *StylesheetParser) importSupportsQuery() (SupportsCondition, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if ok, err := p.scanIdentifier("not", false); err != nil {
		return nil, err
	} else if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		start := p.scanner.State()
		cond, err := p.supportsConditionInParens()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewSupportsNegation(cond, span), nil
	} else if p.scanner.PeekChar(0) == '(' {
		return p.supportsCondition(true)
	}
	fn, err := p.tryImportSupportsFunction()
	if err != nil {
		return nil, err
	}
	if fn != nil {
		return fn, nil
	}
	start := p.scanner.State()
	name, exprErr := p._expression(expressionOpts{consumeNewlines: true})
	if exprErr != nil {
		return nil, exprErr
	}
	if err := p.expectChar(':'); err != nil {
		return nil, err
	}
	value, err := p.supportsDeclarationValue(name)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewSupportsDeclaration(name, value, span), nil
}

// tryImportSupportsFunction consumes a function call inside a supports()
// function after an @import when one is present, and reports nil without
// consuming input otherwise. The scanner position is rewound when the
// identifier is not followed by an opening paren.
func (p *StylesheetParser) tryImportSupportsFunction() (SupportsCondition, error) {
	if !p.lookingAtInterpolatedIdentifier() {
		return nil, nil
	}
	start := p.scanner.State()
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}

	if !p.scanner.ScanChar('(') {
		p.scanner.SetState(start)
		return nil, nil
	}

	value, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
	if err != nil {
		return nil, err
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewSupportsFunction(name, value, span), nil
}

// supportsCondition consumes a @supports condition. When inParentheses holds,
// the indented syntax treats newlines as whitespace in spots where a statement
// would otherwise terminate.
func (p *StylesheetParser) supportsCondition(inParentheses bool) (SupportsCondition, error) {
	start := p.scanner.State()
	if ok, err := p.scanIdentifier("not", false); err != nil {
		return nil, err
	} else if ok {
		if err := p.whitespace(inParentheses); err != nil {
			return nil, err
		}
		cond, err := p.supportsConditionInParens()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewSupportsNegation(cond, span), nil
	}

	condition, err := p.supportsConditionInParens()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(inParentheses); err != nil {
		return nil, err
	}
	var operator *BooleanOperator
	// The first connective fixes the operator for the whole chain; later
	// terms must repeat that same keyword, so once set it is expected
	// literally instead of scanning for either alternative.
	for p.lookingAtIdentifier(nil) {
		if operator != nil {
			if err := p.expectIdentifier(operator.String(), "", false); err != nil {
				return nil, err
			}
		} else if ok, err := p.scanIdentifier("or", false); err != nil {
			return nil, err
		} else if ok {
			o := BooleanOperatorOr
			operator = &o
		} else {
			if err := p.expectIdentifier("and", "", false); err != nil {
				return nil, err
			}
			o := BooleanOperatorAnd
			operator = &o
		}

		if err := p.whitespace(inParentheses); err != nil {
			return nil, err
		}
		right, err := p.supportsConditionInParens()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		condition = NewSupportsOperation(condition, right, *operator, span)
		if err := p.whitespace(inParentheses); err != nil {
			return nil, err
		}
	}
	return condition, nil
}

// supportsConditionInParens consumes a parenthesized supports condition or an
// interpolation.
//
// A leading interpolated identifier resolves to a name(args) function, a lone
// expression interpolation, or an error ("not" is rejected outright; anything
// else that fits neither shape reports "Expected @supports condition."). A
// "(" lead is a negation, a nested condition (re-spanned to the outer parens),
// or a declaration/operation/anything condition parsed with backtracking (see
// below).
func (p *StylesheetParser) supportsConditionInParens() (SupportsCondition, error) {
	start := p.scanner.State()

	if p.lookingAtInterpolatedIdentifier() {
		identifier, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		if identifier.AsPlain() != nil && strings.ToLower(*identifier.AsPlain()) == "not" {
			idSpan, err := identifier.Span()
			if err != nil {
				return nil, err
			}
			if err := p.error("\"not\" is not a valid identifier here.", idSpan, nil); err != nil {
				return nil, err
			}
		}

		if p.scanner.ScanChar('(') {
			arguments, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
			if err != nil {
				return nil, err
			}
			if err := p.expectChar(')'); err != nil {
				return nil, err
			}
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewSupportsFunction(identifier, arguments, span), nil
		} else if len(identifier.Contents) == 1 {
			if e, ok := identifier.Contents[0].(Expression); ok {
				span, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				return NewSupportsInterpolation(e, span), nil
			}
		}
		idSpan, err := identifier.Span()
		if err != nil {
			return nil, err
		}
		if err := p.error("Expected @supports condition.", idSpan, nil); err != nil {
			return nil, err
		}
	}

	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if ok, err := p.scanIdentifier("not", false); err != nil {
		return nil, err
	} else if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		condition, err := p.supportsConditionInParens()
		if err != nil {
			return nil, err
		}
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewSupportsNegation(condition, span), nil
	} else if p.scanner.PeekChar(0) == '(' {
		condition, err := p.supportsCondition(true)
		if err != nil {
			return nil, err
		}
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return condition.WithSpan(span), nil
	}

	var name Expression
	nameStart := p.scanner.State()
	wasInParentheses := p.inParentheses

	// The grammar here is ambiguous: `Expression ":" Expression` versus an
	// interpolated identifier plus an optional anything-value. A top-level
	// colon cannot appear in the anything-value, but the parser still has to
	// read the full expression to see whether a colon follows. The fast path
	// tries the declaration shape first (the common case in practice) and
	// backtracks below; a lookahead for a colon outside balanced brackets
	// would avoid the re-parse at the cost of extra complexity.
	nameExpr, nameErr := p._expression(expressionOpts{consumeNewlines: true})
	if nameErr == nil {
		name = nameExpr
		colErr := p.expectChar(':')
		if colErr == nil {
			value, err := p.supportsDeclarationValue(name)
			if err != nil {
				return nil, err
			}
			if err := p.expectChar(')'); err != nil {
				return nil, err
			}
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewSupportsDeclaration(name, value, span), nil
		}
		// Colon not found — reset and fall through to unified fallback
		nameErr = colErr
	}

	// Declaration parse failed: rewind (including the parentheses flag) and
	// retry as an operation or an unknown-value condition. This matches Dart's
	// FormatException fallback.
	p.scanner.SetState(nameStart)
	p.inParentheses = wasInParentheses

	identifier, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	op, opErr := p.trySupportsOperation(identifier, nameStart)
	if opErr != nil {
		return nil, opErr
	}
	if op != nil {
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		opSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return op.WithSpan(opSpan), nil
	}

	contents := &InterpolationBuffer{}
	contents.AddInterpolation(identifier)
	// The fallback value forbids a top-level colon so a genuine declaration
	// stays distinguishable from unknown syntax.
	decValue, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: false, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
	if err != nil {
		return nil, err
	}
	contents.AddInterpolation(decValue)

	// A colon here means the input was really meant as a declaration, so
	// surface the original declaration-parsing error instead.
	if p.scanner.PeekChar(0) == ':' {
		return nil, nameErr
	}

	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	contentsSpan, err := p.spanFrom(nameStart)
	if err != nil {
		return nil, err
	}
	contentsInterp, err := contents.Interpolation(contentsSpan)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewSupportsAnything(contentsInterp, span), nil
}

// trySupportsOperation parses interpolation as a supports operation when an
// "and" or "or" follows it, and otherwise returns nil leaving the scanner
// where it started.
func (p *StylesheetParser) trySupportsOperation(interpolation *Interpolation, start sasscommon.LineScannerState) (*SupportsOperation, error) {
	// Only a lone expression can serve as the left side; anything else
	// cannot start an operation.
	if len(interpolation.Contents) != 1 {
		return nil, nil
	}
	expr, ok := interpolation.Contents[0].(Expression)
	if !ok {
		return nil, nil
	}

	beforeWhitespace := p.scanner.State()
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var operation *SupportsOperation
	var operator *BooleanOperator
	// As in supportsCondition, the first keyword locks the operator and later
	// terms must repeat it.
	for p.lookingAtIdentifier(nil) {
		if operator != nil {
			if err := p.expectIdentifier(operator.String(), "", false); err != nil {
				return nil, err
			}
		} else if ok, err := p.scanIdentifier("and", false); err != nil {
			return nil, err
		} else if ok {
			o := BooleanOperatorAnd
			operator = &o
		} else if ok, err := p.scanIdentifier("or", false); err != nil {
			return nil, err
		} else if ok {
			o := BooleanOperatorOr
			operator = &o
		} else {
			// No connective follows, so this was never an operation:
			// rewind past the skipped whitespace and let the caller try
			// the anything-value fallback.
			p.scanner.SetState(beforeWhitespace)
			return nil, nil
		}

		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		right, err := p.supportsConditionInParens()
		if err != nil {
			return nil, err
		}
		interpSpan, err := interpolation.Span()
		if err != nil {
			return nil, err
		}
		opSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		// The left side starts as a bare interpolation and folds
		// left-associatively as further terms arrive.
		if operation == nil {
			operation = NewSupportsOperation(
				NewSupportsInterpolation(expr, interpSpan),
				right,
				*operator,
				opSpan,
			)
		} else {
			opSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			operation = NewSupportsOperation(
				operation,
				right,
				*operator,
				opSpan,
			)
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}

	return operation, nil
}

// supportsDeclarationValue parses the right side of a declaration in a
// supports query. Custom-property names keep their raw interpolated text;
// anything else parses as a full expression.
func (p *StylesheetParser) supportsDeclarationValue(name Expression) (Expression, error) {
	if se, ok := name.(*StringExpression); ok && !se.HasQuotes && strings.HasPrefix(se.Text.InitialPlain(), "--") {
		interp, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: false, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
		if err != nil {
			return nil, err
		}
		return NewStringExpression(interp, false), nil
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	return p._expression(expressionOpts{consumeNewlines: true})
}
