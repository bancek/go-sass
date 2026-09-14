// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (expression sections: _expression family)

import (
	"fmt"
	"math"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/unvendor"
	"github.com/bancek/go-sass/util"
)

// Expressions.
//
// This file holds the `_expression` family of stylesheet.dart: a Pratt
// operator-precedence parser that turns source text into Expression nodes.
// Dart implements the precedence loop with nested closures sharing nullable
// locals; Go keeps that shape with local slices plus function-valued locals
// (parseSingleExpression, resolveOneOperation, resolveOperations,
// addSingleExpression, addOperator, resolveSpaceExpressions) over the same
// shared state. Comma-separated lists, space-separated lists, pending
// operators with their left-hand operands, and the slash-division allowance
// all accumulate in those locals until the tail of _expression folds them
// into ListExpression nodes.

// argumentInvocationOpts carries Dart's named parameters of
// _argumentInvocation. When mixin is true the invocation is parsed as a mixin
// include, which forbids the Microsoft-style `=` operator at the top level;
// function invocations allow it. When allowEmptySecondArg is true a missing
// second argument (as in `var()` with one argument) is filled with an
// unquoted empty string.
type argumentInvocationOpts struct {
	mixin               bool
	allowEmptySecondArg bool
}

// argumentInvocation parses a parenthesized argument list. Each element is
// read with expressionUntilComma using singleEquals inverted from mixin, so
// `=` stays available everywhere except mixin includes. A `$name: value`
// pair after a bare variable becomes a named argument (duplicates are an
// error, with the span expanded from name to value); a trailing `...` marks
// the rest argument and a second `...` the keyword rest, which ends the list.
// Positional arguments after named ones are an error, and arguments placed
// after a rest argument emit the misplaced-rest deprecation warning once.
func (p *StylesheetParser) argumentInvocation(opts argumentInvocationOpts) (*ArgumentList, error) {
	start := p.scanner.State()
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var positional []Expression
	named := orderedmap.New[string, Expression]()
	namedSpans := map[string]sasscommon.FileSpan{}
	var rest, keywordRest Expression
	emittedRestDeprecation := false

	for p.lookingAtExpression() {
		expression, exprErr := p.expressionUntilComma(!opts.mixin)
		if exprErr != nil {
			return nil, exprErr
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}

		if ve, ok := expression.(*VariableExpression); ok && p.scanner.ScanChar(':') {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if named.Has(ve.Name()) {
				veSpan, err := ve.Span()
				if err != nil {
					return nil, err
				}
				if err := p.error("Duplicate argument.", veSpan, nil); err != nil {
					return nil, err
				}
			}
			value, valErr := p.expressionUntilComma(!opts.mixin)
			if valErr != nil {
				return nil, valErr
			}
			named.Put(ve.Name(), value)
			exprSpan, err := expression.Span()
			if err != nil {
				return nil, err
			}
			valueSpan, err := value.Span()
			if err != nil {
				return nil, err
			}
			expandedSpan, err := exprSpan.Expand(valueSpan)
			if err != nil {
				return nil, err
			}
			namedSpans[ve.Name()] = expandedSpan

			if rest != nil && !emittedRestDeprecation {
				emittedRestDeprecation = true
				exprSpan2, err := expression.Span()
				if err != nil {
					return nil, err
				}
				valueSpan2, err := value.Span()
				if err != nil {
					return nil, err
				}
				deprecSpan, err := exprSpan2.Expand(valueSpan2)
				if err != nil {
					return nil, err
				}
				restSpan, err := rest.Span()
				if err != nil {
					return nil, err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.MisplacedRest,
					Message:     "Named arguments must come before rest arguments.\nThis will be an error in Dart Sass 2.0.0.",
					Span: sasscommon.NewMultiSpanFileSpan(
						deprecSpan,
						"named argument",
						map[sasscommon.FileSpan]string{restSpan: "rest argument"},
					),
				})
			}
		} else if p.scanner.ScanChar('.') {
			if err := p.expectChar('.'); err != nil {
				return nil, err
			}
			if err := p.expectChar('.'); err != nil {
				return nil, err
			}
			if rest == nil {
				rest = expression
			} else {
				keywordRest = expression
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
				if p.scanner.ScanChar(',') {
					if err := p.whitespace(true); err != nil {
						return nil, err
					}
				}
				break
			}
		} else if named.Len() > 0 {
			exprSpan, err := expression.Span()
			if err != nil {
				return nil, err
			}
			if err := p.error("Positional arguments must come before keyword arguments.", exprSpan, nil); err != nil {
				return nil, err
			}
		} else {
			positional = append(positional, expression)
			if rest != nil && !emittedRestDeprecation {
				emittedRestDeprecation = true
				exprSpan, err := expression.Span()
				if err != nil {
					return nil, err
				}
				restSpan, err := rest.Span()
				if err != nil {
					return nil, err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.MisplacedRest,
					Message:     "Positional arguments must come before rest arguments.\nThis will be an error in Dart Sass 2.0.0.",
					Span: sasscommon.NewMultiSpanFileSpan(
						exprSpan,
						"positional argument",
						map[sasscommon.FileSpan]string{restSpan: "rest argument"},
					),
				})
			}
		}

		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if !p.scanner.ScanChar(',') {
			break
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}

		if opts.allowEmptySecondArg && len(positional) == 1 && named.Len() == 0 && rest == nil && p.scanner.PeekChar(0) == ')' {
			positional = append(positional, NewStringExpressionPlain("", p.scanner.EmptySpan(), false))
			break
		}
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewArgumentList(positional, named, namedSpans, span, rest, keywordRest), nil
}

// expressionOpts carries Dart's named parameters of _expression. When
// bracketList is true the expression is parsed as the contents of a `[`...
// `]` list. When singleEquals is true the Microsoft-style `=` operator is
// accepted at the top level. When consumeNewlines is true the indented syntax
// treats newlines as whitespace, which callers only set where a statement
// cannot end. When until is set it is consulted each time the expression
// could end, and a true result ends the expression early.
// Matches Dart: StylesheetParser._expression (parse/stylesheet.dart).
type expressionOpts struct {
	bracketList     bool
	singleEquals    bool
	consumeNewlines bool
	until           func() bool
}

// _expression consumes an expression with Pratt operator precedence.
//
// Dart marks this @protected for subclass parsers; here it is an unexported
// method on StylesheetParser shared the same way by the per-syntax parsers.
// An until hook that is already true fails with "Expected expression." A
// bracketList call first consumes `[`, returns an empty bracketed list for an
// immediate `]`, and otherwise parses the contents with newlines treated as
// whitespace. The entry single expression is parsed up front; a nil result
// with until set is also "Expected expression."
//
// The loop below mirrors Dart's closure set: resolveOneOperation folds the
// top pending operator (a `/` of two slash-compatible operands outside
// parentheses stays a slash-separated number, otherwise slash is closed off;
// `+`/`-` with the operator glued to the right operand but space after the
// left emits the strict-unary warning); resolveOperations drains the pending
// stack; addSingleExpression discovers space-separated lists and, inside
// parentheses, resets the scanner to the expression start and reparses once a
// space appears so `(1/2 1)` never divides; addOperator rejects non-arithmentic
// operators in plain CSS, keeps slash allowed only across `/`, resolves
// higher-or-equal precedence operators first, and turns a trailing `%` with
// no expression after it into a literal percent string; resolveSpaceExpressions
// folds adjacent expressions into a space-separated list with a span expanded
// from first to last element.
func (p *StylesheetParser) _expression(opts expressionOpts) (Expression, error) {
	// An already-true terminator means the caller asked for an expression
	// where none can start.
	if opts.until != nil && opts.until() {
		return nil, p.scanner.Error("Expected expression.", -1, 0)
	}

	var beforeBracket sasscommon.LineScannerState
	hasBrackets := false
	if opts.bracketList {
		// Bracket contents parse with newlines as whitespace. An immediate
		// `]` is an empty bracketed list spanning both brackets.
		hasBrackets = true
		beforeBracket = p.scanner.State()
		if err := p.expectChar('['); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}

		if p.scanner.ScanChar(']') {
			bracketSpan, err := p.spanFrom(beforeBracket)
			if err != nil {
				return nil, err
			}
			return NewListExpression(nil, ListSeparatorUndecided, bracketSpan, true), nil
		}
	}

	start := p.scanner.State()
	wasInExpression := p.inExpression
	wasInParentheses := p.inParentheses
	// NOTE: Dart doesn't restore state on exception (parsing stops entirely).
	// We match that here — state is only restored before successful returns.
	p.inExpression = true

	var commaExpressions []Expression
	var spaceExpressions []Expression
	var operators []BinaryOperator
	var operands []Expression
	allowSlash := true

	// Dispatches one whitespace-free operand. New cases must also be reflected
	// in lookingAtExpression and in the main loop below, matching Dart's
	// note on _singleExpression.
	parseSingleExpression := func() (Expression, error) {
		switch ch := p.scanner.PeekChar(0); {
		case ch < 0:
			return nil, p.scanner.Error("Expected expression.", -1, 0)
		case ch == '(':
			return p.parentheses()
		case ch == '/':
			u, err := p.unaryOperation()
			if err != nil {
				return nil, err
			}
			return u, nil
		case ch == '.':
			n, err := p.number()
			if err != nil {
				return nil, err
			}
			return n, nil
		case ch == '[':
			return p._expression(expressionOpts{bracketList: true})
		case ch == '$':
			v, err := p.variable()
			if err != nil {
				return nil, err
			}
			return v, nil
		case ch == '&':
			s, err := p.selectorExpr()
			if err != nil {
				return nil, err
			}
			return s, nil
		case ch == '\'' || ch == '"':
			s, err := p.interpolatedString()
			if err != nil {
				return nil, err
			}
			return s, nil
		case ch == '#':
			return p.hashExpression()
		case ch == '+':
			return p.plusExpression()
		case ch == '-':
			return p.minusExpression()
		case ch == '!':
			return p.importantExpression()
		case ch == '%':
			return p.percentExpression()
		case ch == 'u' || ch == 'U':
			if p.scanner.PeekChar(1) == '+' {
				u, err := p.unicodeRange()
				if err != nil {
					return nil, err
				}
				return u, nil
			}
			return p.identifierLike()
		case util.IsDigit(ch):
			n, err := p.number()
			if err != nil {
				return nil, err
			}
			return n, nil
		case util.IsNameStart(ch) || ch == '\\' || ch >= 0x80:
			return p.identifierLike()
		default:
			return nil, p.scanner.Error("Expected expression.", -1, 0)
		}
	}

	singleExpression, err := parseSingleExpression()
	if err != nil {
		return nil, err
	}

	// Folds the top pending operator: the popped operator's left side comes
	// off operands while the current single expression is the right side.
	// A missing right side reports "Expected expression." at the operator.
	resolveOneOperation := func() error {
		op := operators[len(operators)-1]
		operators = operators[:len(operators)-1]
		left := operands[len(operands)-1]
		operands = operands[:len(operands)-1]
		right := singleExpression
		if right == nil {
			return p.scanner.Error("Expected expression.", p.scanner.Position()-len(op.OperatorSyntax()), len(op.OperatorSyntax()))
		}

		if allowSlash && !p.inParentheses && op == BinaryOperatorDividedBy {
			// Slash-separated numbers survive only when both operands qualify
			// (see isSlashOperand); any other division closes slash parsing off
			// for the rest of this expression level.
			if p.isSlashOperand(left) && p.isSlashOperand(right) {
				singleExpression = NewBinaryOperationExpressionSlash(left, right)
			} else {
				singleExpression = NewBinaryOperationExpression(op, left, right)
				allowSlash = false
			}
		} else {
			singleExpression = NewBinaryOperationExpression(op, left, right)
			allowSlash = false

			rightSpan, err := right.Span()
			if err != nil {
				return err
			}
			leftSpan, err := left.Span()
			if err != nil {
				return err
			}
			rightLoc, err := rightSpan.StartLocation()
			if err != nil {
				return err
			}
			leftLoc, err := leftSpan.EndLocation()
			if err != nil {
				return err
			}
			// Warns when `a +b`/`a -b` adjacency suggests a unary operand was
			// meant: the operator touches the right operand's start while
			// whitespace follows the left operand's end.
			if (op == BinaryOperatorPlus || op == BinaryOperatorMinus) &&
				rightLoc.Offset > 0 &&
				p.scanner.Substring(rightLoc.Offset-1, &rightLoc.Offset) == op.OperatorSyntax() &&
				util.IsWhitespace(int(p.scanner.Text()[leftLoc.Offset])) {
				singleExpressionSpan, err := singleExpression.Span()
				if err != nil {
					return err
				}
				leftStr, err := left.String()
				if err != nil {
					return err
				}
				rightStr, err := right.String()
				if err != nil {
					return err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.StrictUnary,
					Message: fmt.Sprintf("This operation is parsed as:\n"+
						"\n"+
						"    %s %s %s\n"+
						"\n"+
						"but you may have intended it to mean:\n"+
						"\n"+
						"    %s (%s%s)\n"+
						"\n"+
						"Add a space after %s to clarify that it's meant to be a binary operation, "+
						"or wrap\n"+
						"it in parentheses to make it a unary operation. This will be an error in "+
						"future\n"+
						"versions of Sass.\n"+
						"\n"+
						"More info and automated migrator: "+
						"https://sass-lang.com/d/strict-unary",
						leftStr, op.OperatorSyntax(), rightStr,
						leftStr, op.OperatorSyntax(), rightStr,
						op.OperatorSyntax()),
					Span: singleExpressionSpan,
				})
			}
		}
		return nil
	}

	resolveOperations := func() error {
		for len(operators) > 0 {
			if err := resolveOneOperation(); err != nil {
				return err
			}
		}
		return nil
	}

	addSingleExpression := func(expression Expression) error {
		if singleExpression != nil {
			// A space inside parentheses means the first element is a list
			// item, not a division: drop the paren context, rewind to the
			// expression start, and reparse so `(1/2 1)` never divides.
			if p.inParentheses {
				p.inParentheses = false
				if allowSlash {
					// When in parentheses and we discover space-separated list,
					// reset to reparse without division context.
					commaExpressions = nil
					spaceExpressions = nil
					operators = nil
					operands = nil
					p.scanner.SetState(start)
					allowSlash = true
					var err2 error
					singleExpression, err2 = parseSingleExpression()
					if err2 != nil {
						return err2
					}
					return nil
				}
			}

			if err := resolveOperations(); err != nil {
				return err
			}
			spaceExpressions = append(spaceExpressions, singleExpression)
			allowSlash = true
		}
		singleExpression = expression
		return nil
	}

	addOperator := func(op BinaryOperator) error {
		// Plain CSS allows only `=`, `+`, `-`, `*`, `/` here; the arithmetic
		// survivors are rechecked as calculations at evaluation time.
		if p.plainCss && op != BinaryOperatorSingleEquals && op != BinaryOperatorPlus &&
			op != BinaryOperatorMinus && op != BinaryOperatorTimes && op != BinaryOperatorDividedBy {
			return p.scanner.Error("Operators aren't allowed in plain CSS.", p.scanner.Position()-len(op.OperatorSyntax()), len(op.OperatorSyntax()))
		}

		allowSlash = allowSlash && op == BinaryOperatorDividedBy

		// Lower precedence finishes every pending higher-or-equal-precedence
		// operator first, so the pending stack stays lowest-to-highest.
		for len(operators) > 0 && operators[len(operators)-1].Precedence() >= op.Precedence() {
			if err := resolveOneOperation(); err != nil {
				return err
			}
		}

		if singleExpression == nil {
			return p.scanner.Error("Expected expression.", p.scanner.Position()-len(op.OperatorSyntax()), len(op.OperatorSyntax()))
		}
		operatorEnd := p.scanner.Position()
		if err := p.whitespace(true); err != nil {
			return err
		}

		// A `%` not followed by an expression (as in `50% red`) is a literal
		// percent string, not a modulo operator.
		if op == BinaryOperatorModulo && !p.lookingAtExpression() {
			modSpan := p.scanner.SpanFromTo(operatorEnd-1, operatorEnd)
			if err := addSingleExpression(NewStringExpressionPlain("%", modSpan, false)); err != nil {
				return err
			}
		} else {
			operators = append(operators, op)
			operands = append(operands, singleExpression)
			var err2 error
			singleExpression, err2 = parseSingleExpression()
			if err2 != nil {
				return err2
			}
		}
		return nil
	}

	resolveSpaceExpressions := func() error {
		// Folds pending operators, then appends the current expression to the
		// space list and replaces it with the space-separated list, whose span
		// expands from the first element to the last.
		if err := resolveOperations(); err != nil {
			return err
		}
		if spaceExpressions == nil {
			return nil
		}
		if singleExpression == nil {
			return p.scanner.Error("Expected expression.", -1, 0)
		}
		spaceExpressions = append(spaceExpressions, singleExpression)
		firstSpan, err := spaceExpressions[0].Span()
		if err != nil {
			return err
		}
		singleSpan, err := singleExpression.Span()
		if err != nil {
			return err
		}
		span, err := firstSpan.Expand(singleSpan)
		if err != nil {
			return err
		}
		singleExpression = NewListExpression(spaceExpressions, ListSeparatorSpace, span, false)
		spaceExpressions = nil
		return nil
	}

	// Main loop
loop:
	for {
		// Newlines count as whitespace inside brackets or when the caller is
		// in a position where a statement cannot end; the until hook ends the
		// expression wherever it could still be valid.
		if err := p.whitespace(opts.consumeNewlines || hasBrackets); err != nil {
			return nil, err
		}
		if opts.until != nil && opts.until() {
			break
		}

		ch := p.scanner.PeekChar(0)
		switch {
		case ch < 0:
			break loop

		case ch == '(':
			expr, _err := p.parentheses()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(expr); err != nil {
				return nil, err
			}

		case ch == '[':
			expr, _err := p._expression(expressionOpts{bracketList: true})
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(expr); err != nil {
				return nil, err
			}

		case ch == '$':
			v, _err := p.variable()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(v); err != nil {
				return nil, err
			}

		case ch == '&':
			s, _err := p.selectorExpr()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(s); err != nil {
				return nil, err
			}

		case ch == '\'' || ch == '"':
			s, _err := p.interpolatedString()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(s); err != nil {
				return nil, err
			}

		case ch == '#':
			expr, _err := p.hashExpression()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(expr); err != nil {
				return nil, err
			}

		case ch == '=':
			// A lone `=` is the Microsoft-style operator only when the caller
			// opted in and it is not followed by a second `=`; otherwise both
			// characters form `==`.
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if opts.singleEquals && p.scanner.PeekChar(0) != '=' {
				if err := addOperator(BinaryOperatorSingleEquals); err != nil {
					return nil, err
				}
			} else {
				if _err := p.expectChar('='); _err != nil {
					return nil, _err
				}
				if err := addOperator(BinaryOperatorEquals); err != nil {
					return nil, err
				}
			}

		case ch == '!':
			// `!=` is an operator; `!important` (with any whitespace gap, or
			// EOF/`i` right after `!`) is an operand; anything else ends the
			// expression.
			switch p.scanner.PeekChar(1) {
			case '=':
				if _, _err := p.readChar(); _err != nil {
					return nil, _err
				}
				if _, _err := p.readChar(); _err != nil {
					return nil, _err
				}
				if err := addOperator(BinaryOperatorNotEquals); err != nil {
					return nil, err
				}
			case -1, 'i', 'I':
				expr, _err := p.importantExpression()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(expr); err != nil {
					return nil, err
				}
			default:
				if util.IsWhitespace(p.scanner.PeekChar(1)) {
					expr, _err := p.importantExpression()
					if _err != nil {
						return nil, _err
					}
					if err := addSingleExpression(expr); err != nil {
						return nil, err
					}
				} else {
					break loop
				}
			}

		case ch == '<':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if p.scanner.ScanChar('=') {
				if err := addOperator(BinaryOperatorLessThanOrEquals); err != nil {
					return nil, err
				}
			} else {
				if err := addOperator(BinaryOperatorLessThan); err != nil {
					return nil, err
				}
			}

		case ch == '>':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if p.scanner.ScanChar('=') {
				if err := addOperator(BinaryOperatorGreaterThanOrEquals); err != nil {
					return nil, err
				}
			} else {
				if err := addOperator(BinaryOperatorGreaterThan); err != nil {
					return nil, err
				}
			}

		case ch == '*':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if err := addOperator(BinaryOperatorTimes); err != nil {
				return nil, err
			}

		case ch == '+' && singleExpression == nil:
			u, _err := p.unaryOperation()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(u); err != nil {
				return nil, err
			}

		case ch == '+':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if err := addOperator(BinaryOperatorPlus); err != nil {
				return nil, err
			}

		case ch == '-':
			// `1-2` splits as `1 - 2`, not `1 (-2)`: a leading digit or dot
			// binds to the number only at the expression start or after
			// whitespace. An interpolated identifier (as in `-moz-...`) wins
			// over both unary and binary readings.
			if (util.IsDigit(p.scanner.PeekChar(1)) || p.scanner.PeekChar(1) == '.') &&
				(singleExpression == nil || util.IsWhitespace(p.scanner.PeekChar(-1))) {
				n, _err := p.number()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(n); err != nil {
					return nil, err
				}
			} else if p.lookingAtInterpolatedIdentifier() {
				expr, _err := p.identifierLike()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(expr); err != nil {
					return nil, err
				}
			} else if singleExpression == nil {
				u, _err := p.unaryOperation()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(u); err != nil {
					return nil, err
				}
			} else {
				if _, _err := p.readChar(); _err != nil {
					return nil, _err
				}
				if err := addOperator(BinaryOperatorMinus); err != nil {
					return nil, err
				}
			}

		case ch == '/' && singleExpression == nil:
			u, _err := p.unaryOperation()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(u); err != nil {
				return nil, err
			}

		case ch == '/':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if err := addOperator(BinaryOperatorDividedBy); err != nil {
				return nil, err
			}

		case ch == '%':
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			if err := addOperator(BinaryOperatorModulo); err != nil {
				return nil, err
			}

		case util.IsDigit(ch):
			n, _err := p.number()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(n); err != nil {
				return nil, err
			}

		case ch == '.' && p.scanner.PeekChar(1) == '.':
			// A `..` sequence starts a rest argument outside this parser, so
			// the expression ends here.
			break loop

		case ch == '.':
			n, _err := p.number()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(n); err != nil {
				return nil, err
			}

		case ch == 'a' && !p.plainCss:
			// `and`/`or` are operators outside plain CSS; anything else
			// starting with those letters falls back to identifier parsing.
			isAnd, _err := p.scanIdentifier("and", false)
			if _err != nil {
				return nil, _err
			}
			if isAnd {
				if err := addOperator(BinaryOperatorAnd); err != nil {
					return nil, err
				}
			}
			// If not "and", let it fall through to identifier handling via identifierLike
			if !isAnd {
				u, _err := p.identifierLike()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(u); err != nil {
					return nil, err
				}
			}

		case ch == 'o' && !p.plainCss:
			// See the `and` case above.
			isOr, _err := p.scanIdentifier("or", false)
			if _err != nil {
				return nil, _err
			}
			if isOr {
				if err := addOperator(BinaryOperatorOr); err != nil {
					return nil, err
				}
			}
			// If not "or", let it fall through to identifier handling via identifierLike
			if !isOr {
				u, _err := p.identifierLike()
				if _err != nil {
					return nil, _err
				}
				if err := addSingleExpression(u); err != nil {
					return nil, err
				}
			}

		case (ch == 'u' || ch == 'U') && p.scanner.PeekChar(1) == '+':
			u, _err := p.unicodeRange()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(u); err != nil {
				return nil, err
			}

		case util.IsNameStart(ch) || ch == '\\' || ch >= 0x80:
			expr, _err := p.identifierLike()
			if _err != nil {
				return nil, _err
			}
			if err := addSingleExpression(expr); err != nil {
				return nil, err
			}

		case ch == ',':
			// A comma inside parentheses also abandons slash parsing: rewind
			// and reparse from the expression start so `(1/2, 1)` never
			// divides its first element.
			if p.inParentheses {
				p.inParentheses = false
				if allowSlash {
					commaExpressions = nil
					spaceExpressions = nil
					operators = nil
					operands = nil
					p.scanner.SetState(start)
					allowSlash = true
					var err2 error
					singleExpression, err2 = parseSingleExpression()
					if err2 != nil {
						return nil, err2
					}
					continue
				}
			}

			if singleExpression == nil {
				return nil, p.scanner.Error("Expected expression.", -1, 0)
			}
			if err := resolveSpaceExpressions(); err != nil {
				return nil, err
			}
			commaExpressions = append(commaExpressions, singleExpression)
			if _, _err := p.readChar(); _err != nil {
				return nil, _err
			}
			allowSlash = true
			singleExpression = nil

		default:
			break loop
		}
	}

	if hasBrackets {
		if _err := p.expectChar(']'); _err != nil {
			return nil, _err
		}
	}

	if commaExpressions != nil {
		// Comma lists fold trailing space expressions first; the span covers
		// the whole list, starting at the `[` when bracketed.
		if err := resolveSpaceExpressions(); err != nil {
			return nil, err
		}
		p.inParentheses = wasInParentheses
		if singleExpression != nil {
			commaExpressions = append(commaExpressions, singleExpression)
		}
		p.inExpression = wasInExpression
		commaSpanStart := start
		if hasBrackets {
			commaSpanStart = beforeBracket
		}
		commaSpan, err := p.spanFrom(commaSpanStart)
		if err != nil {
			return nil, err
		}
		return NewListExpression(commaExpressions, ListSeparatorComma, commaSpan, hasBrackets), nil
	}

	if hasBrackets && spaceExpressions != nil {
		// A bracketed space list resolves pending operators, appends the
		// current expression, and spans from the opening `[`.
		if err := resolveOperations(); err != nil {
			return nil, err
		}
		p.inExpression = wasInExpression
		spaceExpressions = append(spaceExpressions, singleExpression)
		bracketSpan, err := p.spanFrom(beforeBracket)
		if err != nil {
			return nil, err
		}
		return NewListExpression(spaceExpressions, ListSeparatorSpace, bracketSpan, true), nil
	}

	if err := resolveSpaceExpressions(); err != nil {
		return nil, err
	}
	if hasBrackets {
		// A lone bracketed operand becomes an undecided single-element list
		// spanning from the opening `[`.
		bracketSpan2, err := p.spanFrom(beforeBracket)
		if err != nil {
			return nil, err
		}
		singleExpression = NewListExpression([]Expression{singleExpression}, ListSeparatorUndecided, bracketSpan2, true)
	}
	p.inExpression = wasInExpression
	return singleExpression, nil
}

// isSlashOperand reports whether expression may act as an operand of a `/`
// that yields a potentially slash-separated number: number and function
// expressions, plus binary operations that still allow a slash.
func (p *StylesheetParser) isSlashOperand(expression Expression) bool {
	switch expression := expression.(type) {
	case *NumberExpression, *FunctionExpression:
		return true
	case *BinaryOperationExpression:
		return expression.AllowsSlash()
	}
	return false
}

// expressionUntilComma consumes an expression, including newlines, until it
// reaches a top-level comma. When singleEquals is true the Microsoft-style
// `=` operator is allowed at the top level; callers pass the negation of the
// mixin flag so only function invocations get it.
func (p *StylesheetParser) expressionUntilComma(singleEquals bool) (Expression, error) {
	return p._expression(expressionOpts{singleEquals: singleEquals, consumeNewlines: true, until: func() bool {
		return p.scanner.PeekChar(0) == ','
	}})
}

// parentheses consumes a parenthesized expression. Dart marks this @protected
// for subclass parsers; here it is shared the same way as an unexported
// method. Empty parens yield an empty undecided list; a `:` after the first
// element switches to map parsing; further commas build a comma list whose
// span starts inside the parens, and the whole is wrapped as a parenthesized
// expression. The paren flag is restored on return, so callers observe the
// entry context. A parser hook overrides this in tests.
// Matches Dart: StylesheetParser.parentheses (parse/stylesheet.dart).
func (p *StylesheetParser) parentheses() (Expression, error) {
	if p.parenthesesFn != nil {
		return p.parenthesesFn()
	}

	wasInParentheses := p.inParentheses
	p.inParentheses = true
	defer func() { p.inParentheses = wasInParentheses }()

	start := p.scanner.State()
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	inside := p.scanner.State()
	if !p.lookingAtExpression() {
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		listSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewListExpression(nil, ListSeparatorUndecided, listSpan, false), nil
	}

	first, err := p.expressionUntilComma(false)
	if err != nil {
		return nil, err
	}
	if p.scanner.ScanChar(':') {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		return p.mapExpr(first, start)
	}

	if !p.scanner.ScanChar(',') {
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		parenSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewParenthesizedExpression(first, parenSpan), nil
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var expressions []Expression
	expressions = append(expressions, first)
	for {
		if !p.lookingAtExpression() {
			break
		}
		expr, listErr := p.expressionUntilComma(false)
		if listErr != nil {
			return nil, listErr
		}
		expressions = append(expressions, expr)
		if !p.scanner.ScanChar(',') {
			break
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}

	insideSpan, err := p.spanFrom(inside)
	if err != nil {
		return nil, err
	}
	list := NewListExpression(expressions, ListSeparatorComma, insideSpan, false)
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	closeParenSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewParenthesizedExpression(list, closeParenSpan), nil
}

// mapExpr consumes a map expression. It runs after the first colon of the
// map, with first holding the expression before the colon and start marking
// the point before the opening parenthesis. Further `key: value` pairs follow
// comma-separated; a trailing comma with no expression after it simply ends
// the pair list.
// Matches Dart: StylesheetParser._map (parse/stylesheet.dart).
func (p *StylesheetParser) mapExpr(first Expression, start sasscommon.LineScannerState) (*MapExpression, error) {
	var pairs []struct {
		Key   Expression
		Value Expression
	}
	value, err := p.expressionUntilComma(false)
	if err != nil {
		return nil, err
	}
	pairs = append(pairs, struct {
		Key   Expression
		Value Expression
	}{first, value})

	for p.scanner.ScanChar(',') {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if !p.lookingAtExpression() {
			break
		}
		key, keyErr := p.expressionUntilComma(false)
		if keyErr != nil {
			return nil, keyErr
		}
		if err := p.expectChar(':'); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		val, valErr := p.expressionUntilComma(false)
		if valErr != nil {
			return nil, valErr
		}
		pairs = append(pairs, struct {
			Key   Expression
			Value Expression
		}{key, val})
	}

	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	mapSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewMapExpression(pairs, mapSpan), nil
}

// hashExpression consumes an expression starting with `#`. A `#{` opens
// interpolation and falls through to identifier parsing. A digit right after
// `#` starts a hex color; otherwise the identifier is read and, when its
// plain text has a hex-color length (3, 4, 6, or 8 hex digits), the scanner
// rewinds past the identifier and re-reads it as a color. Anything else
// becomes a `#`-prefixed unquoted string.
// Matches Dart: StylesheetParser._hashExpression (parse/stylesheet.dart).
func (p *StylesheetParser) hashExpression() (Expression, error) {
	if p.scanner.PeekChar(1) == '{' {
		return p.identifierLike()
	}

	start := p.scanner.State()
	if err := p.expectChar('#'); err != nil {
		return nil, err
	}

	if util.IsDigit(p.scanner.PeekChar(0)) {
		color, colorErr := p.hexColorContents(start)
		if colorErr != nil {
			errSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			if err := p.error(colorErr.Error(), errSpan, nil); err != nil {
				return nil, err
			}
			return nil, colorErr
		}
		colorSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewColorExpression(color, colorSpan), nil
	}

	afterHash := p.scanner.State()
	identifier, identErr := p.interpolatedIdentifier()
	if identErr != nil {
		return nil, identErr
	}
	if isHexColor(identifier) {
		p.scanner.SetState(afterHash)
		color, colorErr := p.hexColorContents(start)
		if colorErr != nil {
			errSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			if err := p.error(colorErr.Error(), errSpan, nil); err != nil {
				return nil, err
			}
			return nil, colorErr
		}
		colorSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewColorExpression(color, colorSpan), nil
	}

	buffer := &InterpolationBuffer{}
	buffer.WriteCharCode('#')
	buffer.AddInterpolation(identifier)
	hashSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	interp, interpErr := buffer.Interpolation(hashSpan)
	if interpErr != nil {
		hashSpan2, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error(interpErr.Error(), hashSpan2, nil); err != nil {
			return nil, err
		}
		return nil, interpErr
	}
	return NewStringExpression(interp, false), nil
}

// hexColorContents consumes the digits of a hex color after the `#`, with
// start marking the `#`. Three digits expand to `#aabbcc` form; a fourth adds
// alpha; six digits are literal channels; two more add alpha. Only colors
// without alpha keep a span-backed hex format, since four- and eight-digit
// hex output is not yet well-supported in browsers.
// Matches Dart: StylesheetParser._hexColorContents (parse/stylesheet.dart).
func (p *StylesheetParser) hexColorContents(start sasscommon.LineScannerState) (*SassColor, error) {
	digit1, err := p.hexDigit()
	if err != nil {
		return nil, err
	}
	digit2, err := p.hexDigit()
	if err != nil {
		return nil, err
	}
	digit3, err := p.hexDigit()
	if err != nil {
		return nil, err
	}

	var red, green, blue int
	var alpha *float64

	if !util.IsHex(p.scanner.PeekChar(0)) {
		red = (digit1 << 4) + digit1
		green = (digit2 << 4) + digit2
		blue = (digit3 << 4) + digit3
	} else {
		digit4, digit4Err := p.hexDigit()
		if digit4Err != nil {
			return nil, digit4Err
		}
		if !util.IsHex(p.scanner.PeekChar(0)) {
			red = (digit1 << 4) + digit1
			green = (digit2 << 4) + digit2
			blue = (digit3 << 4) + digit3
			a := float64((digit4<<4)+digit4) / 0xff
			alpha = &a
		} else {
			red = (digit1 << 4) + digit2
			green = (digit3 << 4) + digit4
			d5, d5err := p.hexDigit()
			if d5err != nil {
				return nil, d5err
			}
			d6, d6err := p.hexDigit()
			if d6err != nil {
				return nil, d6err
			}
			blue = (d5 << 4) + d6

			if util.IsHex(p.scanner.PeekChar(0)) {
				d7, d7err := p.hexDigit()
				if d7err != nil {
					return nil, d7err
				}
				d8, d8err := p.hexDigit()
				if d8err != nil {
					return nil, d8err
				}
				a := float64((d7<<4)+d8) / 0xff
				alpha = &a
			}
		}
	}

	alphaVal := 1.0
	if alpha != nil {
		alphaVal = *alpha
	}

	var format *ColorFormat
	if alpha == nil {
		spanColorFormatSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		sf := ColorFormat(&SpanColorFormat{Span: spanColorFormatSpan})
		format = &sf
	}
	return NewColorRGBInternal(new(float64(red)), new(float64(green)), new(float64(blue)), new(alphaVal), format)
}

// hexDigit consumes one hexadecimal digit, reporting "Expected hex digit."
// when the next character is not one.
func (p *StylesheetParser) hexDigit() (int, error) {
	if util.IsHex(p.scanner.PeekChar(0)) {
		ch, err := p.readChar()
		if err != nil {
			return 0, err
		}
		return util.AsHex(ch), nil
	}
	return 0, p.scanner.Error("Expected hex digit.", -1, 0)
}

// isHexColor reports whether interpolation is a plain string shaped like a
// hex color: 3, 4, 6, or 8 characters, all hexadecimal digits.
func isHexColor(interpolation *Interpolation) bool {
	plain := interpolation.AsPlain()
	if plain == nil {
		return false
	}
	l := len(*plain)
	if l != 3 && l != 4 && l != 6 && l != 8 {
		return false
	}
	for _, ch := range *plain {
		if !util.IsHex(int(ch)) {
			return false
		}
	}
	return true
}

// plusExpression consumes an expression starting with `+`: a signed number
// when a digit or dot follows, otherwise a unary operation.
func (p *StylesheetParser) plusExpression() (Expression, error) {
	next := p.scanner.PeekChar(1)
	if util.IsDigit(next) || next == '.' {
		n, err := p.number()
		if err != nil {
			return nil, err
		}
		return n, nil
	}
	u, err := p.unaryOperation()
	if err != nil {
		return nil, err
	}
	return u, nil
}

// minusExpression consumes an expression starting with `-`: a signed number
// when a digit or dot follows, an identifier-like expression when looking at
// an interpolated identifier (vendor prefixes such as `-moz-...`), otherwise
// a unary operation.
func (p *StylesheetParser) minusExpression() (Expression, error) {
	if util.IsDigit(p.scanner.PeekChar(1)) || p.scanner.PeekChar(1) == '.' {
		n, err := p.number()
		if err != nil {
			return nil, err
		}
		return n, nil
	}
	if p.lookingAtInterpolatedIdentifier() {
		return p.identifierLike()
	}
	u, err := p.unaryOperation()
	if err != nil {
		return nil, err
	}
	return u, nil
}

// importantExpression consumes an `!important` expression: `!`, optional
// whitespace including newlines, then the `important` identifier, returned as
// a plain unquoted string.
func (p *StylesheetParser) importantExpression() (Expression, error) {
	start := p.scanner.State()
	if _, err := p.readChar(); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if err := p.expectIdentifier("important", "", true); err != nil {
		return nil, err
	}
	importantSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewStringExpressionPlain("!important", importantSpan, false), nil
}

// percentExpression consumes a lone `%` expression as a plain unquoted
// string. It runs only when the main loop has ruled out the modulo operator.
func (p *StylesheetParser) percentExpression() (Expression, error) {
	start := p.scanner.State()
	if _, err := p.readChar(); err != nil {
		return nil, err
	}
	percentSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewStringExpressionPlain("%", percentSpan, false), nil
}

// unaryOperation consumes a unary operation: the operator character, an
// "Expected unary operator." failure for anything else, a plain-CSS rejection
// for every operator except `/`, then the whitespace-free operand.
func (p *StylesheetParser) unaryOperation() (*UnaryOperationExpression, error) {
	start := p.scanner.State()
	ch, err := p.readChar()
	if err != nil {
		return nil, err
	}
	operator := p.unaryOperatorFor(ch)
	if operator == nil {
		return nil, p.scanner.Error("Expected unary operator.", p.scanner.Position()-1, 1)
	} else if p.plainCss && *operator != UnaryOperatorDivide {
		return nil, p.scanner.Error("Operators aren't allowed in plain CSS.", p.scanner.Position()-1, 1)
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	operand, opErr := p.singleExpression()
	if opErr != nil {
		return nil, opErr
	}
	unarySpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewUnaryOperationExpression(*operator, operand, unarySpan), nil
}

// unaryOperatorFor maps a character to its unary operator (`+`, `-`, `/`),
// returning nil for anything that is not a unary operator. Go returns a
// pointer because Dart's nullable return becomes a nilable result here.
func (p *StylesheetParser) unaryOperatorFor(ch int) *UnaryOperator {
	switch ch {
	case '+':
		op := UnaryOperatorPlus
		return &op
	case '-':
		op := UnaryOperatorMinus
		return &op
	case '/':
		op := UnaryOperatorDivide
		return &op
	}
	return nil
}

// singleExpression consumes one expression with no top-level whitespace:
// parenthesized, unary, numeric, bracketed, variable, parent-selector,
// quoted, hash, sign, important, percent, unicode-range, or identifier-like,
// selected by the next character. New operand kinds must also be reflected in
// lookingAtExpression and in the _expression main loop.
func (p *StylesheetParser) singleExpression() (Expression, error) {
	switch ch := p.scanner.PeekChar(0); {
	case ch < 0:
		return nil, p.scanner.Error("Expected expression.", -1, 0)
	case ch == '(':
		return p.parentheses()
	case ch == '/':
		u, err := p.unaryOperation()
		if err != nil {
			return nil, err
		}
		return u, nil
	case ch == '.':
		n, err := p.number()
		if err != nil {
			return nil, err
		}
		return n, nil
	case ch == '[':
		return p._expression(expressionOpts{bracketList: true})
	case ch == '$':
		v, err := p.variable()
		if err != nil {
			return nil, err
		}
		return v, nil
	case ch == '&':
		s, err := p.selectorExpr()
		if err != nil {
			return nil, err
		}
		return s, nil
	case ch == '\'' || ch == '"':
		s, err := p.interpolatedString()
		if err != nil {
			return nil, err
		}
		return s, nil
	case ch == '#':
		return p.hashExpression()
	case ch == '+':
		return p.plusExpression()
	case ch == '-':
		return p.minusExpression()
	case ch == '!':
		return p.importantExpression()
	case ch == '%':
		return p.percentExpression()
	case (ch == 'u' || ch == 'U') && p.scanner.PeekChar(1) == '+':
		u, err := p.unicodeRange()
		if err != nil {
			return nil, err
		}
		return u, nil
	case util.IsDigit(ch):
		n, err := p.number()
		if err != nil {
			return nil, err
		}
		return n, nil
	case util.IsNameStart(ch) || ch == '\\' || ch >= 0x80:
		return p.identifierLike()
	default:
		return nil, p.scanner.Error("Expected expression.", -1, 0)
	}
}

// number consumes a number expression with an optional unit. A leading sign
// is taken first; the integer part is skipped when the number starts with a
// dot. The text parses with float scanning so long digit runs do not
// accumulate extra floating-point error (overflowing text becomes infinity,
// as in the Go port). A trailing `%` is the unit; otherwise an identifier
// that does not start with `--` becomes the unit.
// Matches Dart: StylesheetParser._number (parse/stylesheet.dart).
func (p *StylesheetParser) number() (*NumberExpression, error) {
	start := p.scanner.State()
	first := p.scanner.PeekChar(0)
	if first == '+' || first == '-' {
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
	}

	if p.scanner.PeekChar(0) != '.' {
		if err := p.consumeNaturalNumber(); err != nil {
			return nil, err
		}
	}

	if err := p.tryDecimal(start.Position != p.scanner.Position() && first != '+' && first != '-'); err != nil {
		return nil, err
	}
	if err := p.tryExponent(); err != nil {
		return nil, err
	}

	numberText := p.scanner.Substring(start.Position, nil)
	var number float64
	if n, scanErr := fmt.Sscanf(numberText, "%f", &number); scanErr != nil || n != 1 {
		if strings.HasPrefix(numberText, "-") {
			number = math.Inf(-1)
		} else {
			number = math.Inf(1)
		}
	}

	var unit *string
	if p.scanner.ScanChar('%') {
		s := "%"
		unit = &s
	} else if p.lookingAtIdentifier(nil) &&
		!(p.scanner.PeekChar(0) == '-' && p.scanner.PeekChar(1) == '-') {
		u, err := p.identifier(false, true)
		if err != nil {
			return nil, err
		}
		unit = &u
	}

	numSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewNumberExpression(number, numSpan, unit), nil
}

// consumeNaturalNumber consumes a non-negative integer with no scientific
// notation: one required digit followed by further digits.
func (p *StylesheetParser) consumeNaturalNumber() error {
	ch, err := p.readChar()
	if err != nil {
		return err
	}
	if !util.IsDigit(ch) {
		return p.scanner.Error("Expected digit.", p.scanner.Position()-1, 0)
	}
	for util.IsDigit(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	return nil
}

// tryDecimal consumes the fractional component of a number when present.
// When allowTrailingDot is false a dot with no digits after it is an
// "Expected digit." error; when true the dot is left unconsumed. Callers
// allow the trailing dot only for numbers that do not start with a dot, so a
// bare `.` still fails while `1.` stays available for rest arguments like
// `1...`. Go reports the error at the digit position with its result error.
func (p *StylesheetParser) tryDecimal(allowTrailingDot bool) error {
	if p.scanner.PeekChar(0) != '.' {
		return nil
	}
	if !util.IsDigit(p.scanner.PeekChar(1)) {
		if allowTrailingDot {
			return nil
		}
		if _, err := p.readChar(); err != nil {
			return err
		}
		pos := p.scanner.Position()
		return p.scanner.Error("Expected digit.", pos, 1)
	}
	if _, err := p.readChar(); err != nil {
		return err
	}
	for util.IsDigit(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	return nil
}

// tryExponent consumes the exponent component (`e`/`E`, optional sign,
// required digits) of a number when present, and returns nil untouched when
// the next characters cannot form one.
func (p *StylesheetParser) tryExponent() error {
	first := p.scanner.PeekChar(0)
	if first != 'e' && first != 'E' {
		return nil
	}
	next := p.scanner.PeekChar(1)
	if !util.IsDigit(next) && next != '-' && next != '+' {
		return nil
	}
	if _, err := p.readChar(); err != nil {
		return err
	}
	if next == '+' || next == '-' {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	if !util.IsDigit(p.scanner.PeekChar(0)) {
		return p.scanner.Error("Expected digit.", -1, 0)
	}
	for util.IsDigit(p.scanner.PeekChar(0)) {
		if _, err := p.readChar(); err != nil {
			return err
		}
	}
	return nil
}

// unicodeRange consumes a `U+...` range: `u`, `+`, up to six hex digits with
// `?` wildcards allowed, plus an optional `-` second range. A wildcarded
// first range returns immediately; overlong ranges and a trailing identifier
// body are errors. The result is a plain unquoted string of the source text.
func (p *StylesheetParser) unicodeRange() (*StringExpression, error) {
	start := p.scanner.State()
	if err := p.expectIdentChar('u', false); err != nil {
		return nil, err
	}
	if err := p.expectChar('+'); err != nil {
		return nil, err
	}

	firstRangeLength := 0
	for ch := p.scanner.PeekChar(0); ch >= 0 && util.IsHex(ch); ch = p.scanner.PeekChar(0) {
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		firstRangeLength++
	}

	hasQuestionMark := false
	for p.scanner.ScanChar('?') {
		hasQuestionMark = true
		firstRangeLength++
	}

	if firstRangeLength == 0 {
		return nil, p.scanner.Error(`Expected hex digit or "?".`, -1, 0)
	} else if firstRangeLength > 6 {
		unicodeErrSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Expected at most 6 digits.", unicodeErrSpan, nil); err != nil {
			return nil, err
		}
	} else if hasQuestionMark {
		unicodeQSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewStringExpressionPlain(p.scanner.Substring(start.Position, nil), unicodeQSpan, false), nil
	}

	if p.scanner.ScanChar('-') {
		secondRangeStart := p.scanner.State()
		secondRangeLength := 0
		for ch := p.scanner.PeekChar(0); ch >= 0 && util.IsHex(ch); ch = p.scanner.PeekChar(0) {
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			secondRangeLength++
		}

		if secondRangeLength == 0 {
			return nil, p.scanner.Error("Expected hex digit.", -1, 0)
		} else if secondRangeLength > 6 {
			secondRangeSpan, err := p.spanFrom(secondRangeStart)
			if err != nil {
				return nil, err
			}
			if err := p.error("Expected at most 6 digits.", secondRangeSpan, nil); err != nil {
				return nil, err
			}
		}
	}

	if p.lookingAtInterpolatedIdentifierBody() {
		return nil, p.scanner.Error("Expected end of identifier.", -1, 0)
	}

	unicodeSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewStringExpressionPlain(p.scanner.Substring(start.Position, nil), unicodeSpan, false), nil
}

// variable consumes a `$name` expression. Plain CSS rejects Sass variables
// with a spanned error before the variable node is built.
func (p *StylesheetParser) variable() (*VariableExpression, error) {
	start := p.scanner.State()
	name, err := p.variableName()
	if err != nil {
		return nil, err
	}

	if p.plainCss {
		varErrSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Sass variables aren't allowed in plain CSS.", varErrSpan, nil); err != nil {
			return nil, err
		}
	}

	varSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewVariableExpression(name, varSpan, nil), nil
}

// selectorExpr consumes a `&` parent-selector expression. Plain CSS rejects
// it outright. A doubled `&&` warns that Sass reads it as two copies of the
// parent selector (suggesting `and` instead) and rewinds one character so the
// second `&` parses as its own expression.
func (p *StylesheetParser) selectorExpr() (*SelectorExpression, error) {
	if p.plainCss {
		return nil, p.scanner.Error("The parent selector isn't allowed in plain CSS.", -1, 1)
	}

	start := p.scanner.State()
	if err := p.expectChar('&'); err != nil {
		return nil, err
	}

	if p.scanner.ScanChar('&') {
		selSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		p.warnings = append(p.warnings, ParseTimeWarning{
			Deprecation: nil,
			Message:     `In Sass, "&&" means two copies of the parent selector. You probably want to use "and" instead.`,
			Span:        selSpan,
		})
		p.setPosition(p.scanner.Position() - 1)
	}

	selExprSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewSelectorExpression(selExprSpan), nil
}

// interpolatedString consumes a quoted string expression with interpolation.
// A newline or EOF before the closing quote fails with "Expected <quote>.".
// A backslash-newline (including `\r\n`) is a line continuation; other
// escapes decode through escapeCharacter; `#{` runs one interpolation. This
// logic largely duplicates the token form below and the base string parser,
// so most changes here should be mirrored there.
// Matches Dart: StylesheetParser.interpolatedString (parse/stylesheet.dart).
func (p *StylesheetParser) interpolatedString() (*StringExpression, error) {
	start := p.scanner.State()
	quote, err := p.readChar()
	if err != nil {
		return nil, err
	}

	if quote != '\'' && quote != '"' {
		return nil, p.scanner.Error("Expected string.", start.Position, 0)
	}

	buffer := &InterpolationBuffer{}
	for {
		next := p.scanner.PeekChar(0)
		switch {
		case next == quote:
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			strSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			interp, interpErr := buffer.Interpolation(strSpan)
			if interpErr != nil {
				strSpan2, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				if err := p.error(interpErr.Error(), strSpan2, nil); err != nil {
					return nil, err
				}
				return nil, interpErr
			}
			return NewStringExpression(interp, true), nil

		case next < 0 || util.IsNewline(next):
			return nil, p.scanner.Error(fmt.Sprintf("Expected %c.", rune(quote)), -1, 0)

		case next == '\\':
			second := p.scanner.PeekChar(1)
			if util.IsNewline(second) {
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
				if second == '\r' {
					p.scanner.ScanChar('\n')
				}
			} else {
				ch, err := p.escapeCharacter()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(ch)
			}

		case next == '#' && p.scanner.PeekChar(1) == '{':
			expr, span, siErr := p.singleInterpolation()
			if siErr != nil {
				return nil, siErr
			}
			buffer.Add(expr, span)

		default:
			ch, rdErr := p.readChar()
			if rdErr != nil {
				return nil, rdErr
			}
			buffer.WriteCharCode(ch)
		}
	}
}

// interpolatedStringToken consumes a quoted string as raw source text,
// keeping the quotes and escapes verbatim instead of interpreting them. Line
// continuations are preserved character-for-character here (unlike the
// semantic form above, which drops them). Mirrors the same duplication note:
// most changes here should be mirrored in interpolatedString and the base
// string parser.
// Matches Dart: StylesheetParser.interpolatedStringToken (parse/stylesheet.dart).
func (p *StylesheetParser) interpolatedStringToken() (*Interpolation, error) {
	start := p.scanner.State()
	quote, err := p.readChar()
	if err != nil {
		return nil, err
	}

	if quote != '\'' && quote != '"' {
		return nil, p.scanner.Error("Expected string.", start.Position, 0)
	}

	buffer := &InterpolationBuffer{}
	buffer.WriteCharCode(quote)
	for {
		next := p.scanner.PeekChar(0)
		switch {
		case next == quote:
			ch, rdErr := p.readChar()
			if rdErr != nil {
				return nil, rdErr
			}
			buffer.WriteCharCode(ch)
			tokSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			interp, interpErr := buffer.Interpolation(tokSpan)
			if interpErr != nil {
				tokSpan2, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				if err := p.error(interpErr.Error(), tokSpan2, nil); err != nil {
					return nil, err
				}
				return nil, interpErr
			}
			return interp, nil

		case next < 0 || util.IsNewline(next):
			return nil, p.scanner.Error(fmt.Sprintf("Expected %c.", rune(quote)), -1, 0)

		case next == '\\':
			second := p.scanner.PeekChar(1)
			if util.IsNewline(second) {
				ch1, rdErr1 := p.readChar()
				if rdErr1 != nil {
					return nil, rdErr1
				}
				buffer.WriteCharCode(ch1)
				ch2, rdErr2 := p.readChar()
				if rdErr2 != nil {
					return nil, rdErr2
				}
				buffer.WriteCharCode(ch2)
				if second == '\r' {
					if p.scanner.ScanChar('\n') {
						buffer.WriteCharCode('\n')
					}
				}
			} else {
				text, err := p.rawText(func() error { _, err := p.escapeCharacter(); return err })
				if err != nil {
					return nil, err
				}
				buffer.Write(text)
			}

		case next == '#' && p.scanner.PeekChar(1) == '{':
			expr, span, siErr := p.singleInterpolation()
			if siErr != nil {
				return nil, siErr
			}
			buffer.Add(expr, span)

		default:
			ch, rdErr := p.readChar()
			if rdErr != nil {
				return nil, rdErr
			}
			buffer.WriteCharCode(ch)
		}
	}
}

// identifierLike consumes an expression that starts like an identifier.
// Dart marks this @protected for subclass parsers; here it is shared the
// same way as an unexported method, with a test hook override. A lowercase
// `if(` is tried as the legacy `if()` function first, falling back to the
// modern `if()` on failure since the two need arbitrary lookahead to tell
// apart; any-case `if(` is modern directly. `not` negates the following
// single expression with the span expanded across both. Without a following
// `(`, `true`/`false`/`null` and named colors become literals; special
// functions (calc, url, vendor forms) win next. Finally `.` either continues
// a rest argument (`..`, returned as a string) or a namespace, while `(`
// builds a plain or interpolated function call by whether the name is plain.
// Matches Dart: StylesheetParser.identifierLike (parse/stylesheet.dart).
func (p *StylesheetParser) identifierLike() (Expression, error) {
	if p.identifierLikeFn != nil {
		return p.identifierLikeFn()
	}

	start := p.scanner.State()
	identifier, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	if identifier == nil {
		return nil, nil
	}
	plain := identifier.AsPlain()
	var lower string
	if plain != nil {
		if *plain == "if" && p.scanner.PeekChar(0) == '(' {
			beforeParen := p.scanner.State()
			invocation, invErr := p.argumentInvocation(argumentInvocationOpts{})
			if invErr != nil {
				p.scanner.SetState(beforeParen)
				return p.ifExpression(start)
			}

			identifierSpan, err := identifier.Span()
			if err != nil {
				return nil, err
			}
			invocationSpan, err := invocation.Span()
			if err != nil {
				return nil, err
			}
			legacySpan, err := identifierSpan.Expand(invocationSpan)
			if err != nil {
				return nil, err
			}
			expression := NewLegacyIfExpression(invocation, legacySpan)
			message := "The Sass if() syntax is deprecated in favor of the modern CSS syntax.\n\n"
			suggestion, err := expression.ModernSuggestion()
			if err != nil {
				return nil, err
			}
			if suggestion != nil {
				message += "Suggestion: " + *suggestion + "\n\n"
			}
			message += "More info: https://sass-lang.com/d/if-function"
			exprSpan, err := expression.Span()
			if err != nil {
				return nil, err
			}
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: deprecation.IfFunction,
				Message:     message,
				Span:        exprSpan,
			})
			return expression, nil
		} else if strings.ToLower(*plain) == "if" && p.scanner.PeekChar(0) == '(' {
			return p.ifExpression(start)
		} else if *plain == "not" {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			expression, exprErr := p.singleExpression()
			if exprErr != nil {
				return nil, exprErr
			}
			notSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewUnaryOperationExpression(UnaryOperatorNot, expression, notSpan), nil
		}

		lower = strings.ToLower(*plain)
		if p.scanner.PeekChar(0) != '(' {
			switch *plain {
			case "false":
				identSpan, err := identifier.Span()
				if err != nil {
					return nil, err
				}
				return NewBooleanExpression(false, identSpan), nil
			case "null":
				identSpan, err := identifier.Span()
				if err != nil {
					return nil, err
				}
				return NewNullExpression(identSpan), nil
			case "true":
				identSpan, err := identifier.Span()
				if err != nil {
					return nil, err
				}
				return NewBooleanExpression(true, identSpan), nil
			}

			if c, ok := colorsByName.Get(lower); ok {
				identSpan, err := identifier.Span()
				if err != nil {
					return nil, err
				}
				sf := ColorFormat(&SpanColorFormat{Span: identSpan})
				color, colorErr := NewColorRGBInternal(new(float64(c.r)), new(float64(c.g)), new(float64(c.b)), new(c.a), &sf)
				if colorErr != nil {
					if err := p.error(colorErr.Error(), identSpan, nil); err != nil {
						return nil, err
					}
					return nil, colorErr
				}
				return NewColorExpression(color, identSpan), nil
			}
		}

		if specialFn, fnErr := p.trySpecialFunction(lower, start); fnErr != nil {
			return nil, fnErr
		} else if specialFn != nil {
			return specialFn, nil
		}
	}

	switch ch := p.scanner.PeekChar(0); {
	case ch == '.' && p.scanner.PeekChar(1) == '.':
		return NewStringExpression(identifier, false), nil

	case ch == '.':
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		if plain != nil {
			expr, exprErr := p.namespacedExpression(*plain, start)
			if exprErr != nil {
				return nil, exprErr
			}
			return expr, nil
		}
		identSpan, err := identifier.Span()
		if err != nil {
			return nil, err
		}
		if err := p.error("Interpolation isn't allowed in namespaces.", identSpan, nil); err != nil {
			return nil, err
		}
		return nil, nil

	case ch == '(' && plain != nil:
		invocation, invErr := p.argumentInvocation(argumentInvocationOpts{allowEmptySecondArg: lower == "var"})
		if invErr != nil {
			return nil, invErr
		}
		funcSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewFunctionExpression(*plain, invocation, funcSpan, nil), nil

	case ch == '(':
		invocation, invErr := p.argumentInvocation(argumentInvocationOpts{})
		if invErr != nil {
			return nil, invErr
		}
		interpFuncSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedFunctionExpression(identifier, invocation, interpFuncSpan), nil

	default:
		return NewStringExpression(identifier, false), nil
	}
}

// ifExpression consumes a modern CSS-style `if()` expression, starting after
// the name with start marking the name. Each branch is an optional condition
// (`else` gives a nil condition) followed by `:` and a newline-tolerant
// expression; branches separate with `;`. Dart marks this @protected; here it
// is shared the same way as an unexported method.
// Matches Dart: StylesheetParser.ifExpression (parse/stylesheet.dart).
func (p *StylesheetParser) ifExpression(start sasscommon.LineScannerState) (*IfExpression, error) {
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	var branches []IfBranch
	for p.scanner.PeekChar(0) != ')' {
		var condition IfConditionExpression
		isElse, _err := p.scanIdentifier("else", false)
		if _err != nil {
			return nil, _err
		}
		if !isElse {
			var condErr error
			condition, condErr = p.ifConditionExpression()
			if condErr != nil {
				return nil, condErr
			}
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if err := p.expectChar(':'); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		expr, exprErr := p._expression(expressionOpts{consumeNewlines: true})
		if exprErr != nil {
			return nil, exprErr
		}
		branches = append(branches, IfBranch{Condition: condition, Expression: expr})
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if !p.scanner.ScanChar(';') {
			break
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	result, err := NewIfExpression(branches, p.scanner.SpanFrom(start))
	if err != nil {
		if cfgErr := p.error(err.Error(), p.scanner.SpanFrom(start), nil); cfgErr != nil {
			return nil, cfgErr
		}
		return nil, err
	}
	return result, nil
}

// ifConditionExpression consumes one modern `if()` condition: a leading
// `not` (which requires whitespace before any `(`), then `and`/`or`-joined
// groups that never mix both operators. When a group or an arbitrary
// substitution (`var()`/`attr()`/`if()`/`#{}`/custom property) cannot stay a
// structured operation, parsing restarts that tail as raw condition text via
// ifConditionRaw. A single group returns unwrapped.
// Matches Dart: StylesheetParser._ifConditionExpression (parse/stylesheet.dart).
func (p *StylesheetParser) ifConditionExpression() (IfConditionExpression, error) {
	start := p.scanner.State()
	isNot, _err := p.scanIdentifier("not", false)
	if _err != nil {
		return nil, _err
	}
	if isNot {
		if p.scanner.PeekChar(0) == '(' {
			return nil, p.scanner.Error(`Whitespace is required between "not" and "("`, -1, 0)
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		group, grpErr := p.ifGroup()
		if grpErr != nil {
			return nil, grpErr
		}
		return NewIfConditionNegation(group, p.scanner.SpanFrom(start)), nil
	}

	first, firstErr := p.ifGroup()
	if firstErr != nil {
		return nil, firstErr
	}
	groups := []IfConditionExpression{first}
	var op *BooleanOperator

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	for {
		isAnd := false
		if op == nil || *op == BooleanOperatorAnd {
			var scanErr error
			isAnd, scanErr = p.scanIdentifier("and", false)
			if scanErr != nil {
				return nil, scanErr
			}
		}
		if isAnd {
			if p.scanner.PeekChar(0) == '(' {
				return nil, p.scanner.Error(`Whitespace is required between "and" and "("`, -1, 0)
			}
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if op == nil {
				o := BooleanOperatorAnd
				op = &o
			}
			g, gErr := p.ifGroup()
			if gErr != nil {
				return nil, gErr
			}
			groups = append(groups, g)
		} else {
			isOr := false
			if op == nil || *op == BooleanOperatorOr {
				var scanErr error
				isOr, scanErr = p.scanIdentifier("or", false)
				if scanErr != nil {
					return nil, scanErr
				}
			}
			if isOr {
				if p.scanner.PeekChar(0) == '(' {
					return nil, p.scanner.Error(`Whitespace is required between "and" and "("`, -1, 0)
				}
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
				if op == nil {
					o := BooleanOperatorOr
					op = &o
				}
				g, gErr := p.ifGroup()
				if gErr != nil {
					return nil, gErr
				}
				groups = append(groups, g)
			} else if ch := p.scanner.PeekChar(0); ch != ')' && ch != ':' && ch >= 0 && groups[len(groups)-1].IsArbitrarySubstitution() {
				var preceding IfConditionExpression
				if len(groups) == 1 {
					preceding = groups[0]
				} else {
					var err error
					preceding, err = NewIfConditionOperation(groups, *op)
					if err != nil {
						if cfgErr := p.error(err.Error(), p.scanner.SpanFrom(start), nil); cfgErr != nil {
							return nil, cfgErr
						}
						return nil, err
					}
				}
				next, nextErr := p.ifGroup()
				if nextErr != nil {
					return nil, nextErr
				}
				result, err := p.ifConditionRaw(preceding, next)
				if err != nil {
					return nil, err
				}
				return result, nil
			} else if substitution, subErr := p.tryArbitrarySubstitution(); subErr != nil {
				return nil, subErr
			} else if substitution != nil {
				var preceding IfConditionExpression
				if len(groups) == 1 {
					preceding = groups[0]
				} else {
					var err error
					preceding, err = NewIfConditionOperation(groups, *op)
					if err != nil {
						if cfgErr := p.error(err.Error(), p.scanner.SpanFrom(start), nil); cfgErr != nil {
							return nil, cfgErr
						}
						return nil, err
					}
				}
				result, err := p.ifConditionRaw(preceding, substitution)
				if err != nil {
					return nil, err
				}
				return result, nil
			} else {
				break
			}
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}

	if len(groups) == 1 {
		return groups[0], nil
	}
	result, err := NewIfConditionOperation(groups, *op)
	if err != nil {
		if cfgErr := p.error(err.Error(), p.scanner.SpanFrom(start), nil); cfgErr != nil {
			return nil, cfgErr
		}
		return nil, err
	}
	return result, nil
}

// ifGroup consumes one grouped expression in an `if()` condition:
// parenthesized recursion, case-sensitive `sass(...)` holding a real Sass
// expression (rejected in plain CSS), or a function-style condition whose
// arguments are declaration-value text. An identifier carrying an embedded
// expression without a following `(` is raw text, while `and`/`or`/`not`
// glued to `(` fail for the missing whitespace.
// Matches Dart: StylesheetParser._ifGroup (parse/stylesheet.dart).
func (p *StylesheetParser) ifGroup() (IfConditionExpression, error) {
	start := p.scanner.State()
	switch p.scanner.PeekChar(0) {
	case '(':
		if err := p.expectChar('('); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		expression, exprErr := p.ifConditionExpression()
		if exprErr != nil {
			return nil, exprErr
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		return NewIfConditionParenthesized(expression, p.scanner.SpanFrom(start)), nil

	default:
		isSass, _err := p.scanIdentifier("sass", true)
		if _err != nil {
			return nil, _err
		}
		if isSass {
			if err := p.expectChar('('); err != nil {
				return nil, err
			}
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			expression, exprErr := p._expression(expressionOpts{})
			if exprErr != nil {
				return nil, exprErr
			}
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if err := p.expectChar(')'); err != nil {
				return nil, err
			}
			if p.plainCss {
				if err := p.error("sass() conditions aren't allowed in plain CSS", p.scanner.SpanFrom(start), nil); err != nil {
					return nil, err
				}
				return nil, nil
			}
			return NewIfConditionSass(expression, p.scanner.SpanFrom(start)), nil
		}
		identifier, identErr := p.interpolatedIdentifier()
		if identErr != nil {
			return nil, identErr
		}
		plain := identifier.AsPlain()

		if len(identifier.Contents) == 1 {
			if _, ok := identifier.Contents[0].(Expression); ok && p.scanner.PeekChar(0) != '(' {
				return NewIfConditionRaw(identifier), nil
			}
		}

		if plain != nil {
			lower := strings.ToLower(*plain)
			if (lower == "and" || lower == "or" || lower == "not") && p.scanner.PeekChar(0) == '(' {
				return nil, p.scanner.Error(fmt.Sprintf("Whitespace is required between %q and \"(\"", *plain), -1, 0)
			}
		}

		if err := p.expectChar('('); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		expression, exprErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
		if exprErr != nil {
			return nil, exprErr
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
		return NewIfConditionFunction(identifier, expression, p.scanner.SpanFrom(start)), nil
	}
}

// tryArbitrarySubstitution consumes an arbitrary-substitution expression
// (`#{...}`, `if(...)`, `var(...)`, `attr(...)`, or a custom property call),
// or returns a nil condition with no error when the input is not one. A
// recognized name without a following `(` rewinds to the start and also
// yields nil, leaving the text for the caller. The arguments parse as
// declaration-value text through the closing paren.
// Matches Dart: StylesheetParser._tryArbitrarySubstitution (parse/stylesheet.dart).
func (p *StylesheetParser) tryArbitrarySubstitution() (IfConditionExpression, error) {
	if p.scanner.PeekChar(0) == '#' && p.scanner.PeekChar(1) == '{' {
		expression, span, siErr := p.singleInterpolation()
		if siErr != nil {
			return nil, siErr
		}
		buf := &InterpolationBuffer{}
		buf.Add(expression, span)
		raw, rawErr := buf.Interpolation(span)
		if rawErr != nil {
			if err := p.error(rawErr.Error(), span, nil); err != nil {
				return nil, err
			}
			return nil, rawErr
		}
		return NewIfConditionRaw(raw), nil
	}

	start := p.scanner.State()
	var name *Interpolation

	isIf, ifErr := p.scanIdentifier("if", false)
	if ifErr != nil {
		return nil, ifErr
	}
	if isIf {
		name = NewInterpolationPlain("if", p.scanner.SpanFrom(start))
	} else {
		isVar, varErr := p.scanIdentifier("var", false)
		if varErr != nil {
			return nil, varErr
		}
		if isVar {
			name = NewInterpolationPlain("var", p.scanner.SpanFrom(start))
		} else {
			isAttr, attrErr := p.scanIdentifier("attr", false)
			if attrErr != nil {
				return nil, attrErr
			}
			if isAttr {
				name = NewInterpolationPlain("attr", p.scanner.SpanFrom(start))
			} else if p.scanner.PeekChar(0) == '-' && p.scanner.PeekChar(1) == '-' {
				var identErr error
				name, identErr = p.interpolatedIdentifier()
				if identErr != nil {
					return nil, identErr
				}
			}
		}
	}

	if name == nil {
		p.scanner.SetState(start)
		return nil, nil
	}

	if !p.scanner.ScanChar('(') {
		p.scanner.SetState(start)
		return nil, nil
	}

	arguments, argErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
	if argErr != nil {
		return nil, argErr
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}

	return NewIfConditionFunction(name, arguments, p.scanner.SpanFrom(start)), nil
}

// ifConditionRaw consumes the remainder of what would have been an
// if-condition operation as raw text. It requires that preceding ends with
// (or is) an arbitrary substitution, or that next is one, and panics
// otherwise — matching Dart's argument error, since callers only invoke it on
// that path. Both sides render through the substitution's interpolation, then
// further `and`/`or`/juxtaposed groups and substitutions append with their
// separators, keeping a single `and`/`or` operator throughout. The span runs
// from the preceding expression's start to the current position.
// Matches Dart: StylesheetParser._ifConditionRaw (parse/stylesheet.dart).
func (p *StylesheetParser) ifConditionRaw(preceding, next IfConditionExpression) (*IfConditionRaw, error) {
	var substitution IfConditionExpression
	if preceding.IsArbitrarySubstitution() {
		substitution = preceding
	} else if op, ok := preceding.(*IfConditionOperation); ok && len(op.Expressions) > 0 && op.Expressions[len(op.Expressions)-1].IsArbitrarySubstitution() {
		substitution = op.Expressions[len(op.Expressions)-1]
	} else if next.IsArbitrarySubstitution() {
		substitution = next
	} else {
		panic("ifConditionRaw: either preceding must end with an arbitrary substitution or next must be one")
	}

	buffer := &InterpolationBuffer{}
	precedingInterp, err := preceding.ToInterpolation(substitution)
	if err != nil {
		return nil, err
	}
	buffer.AddInterpolation(precedingInterp)
	buffer.WriteCharCode(' ')
	nextInterp, err := next.ToInterpolation(substitution)
	if err != nil {
		return nil, err
	}
	buffer.AddInterpolation(nextInterp)

	lastGroup := next
	var op *BooleanOperator
	if operation, ok := preceding.(*IfConditionOperation); ok {
		op = &operation.Op
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	for {
		isAnd := false
		if op == nil || *op != BooleanOperatorOr {
			var scanErr error
			isAnd, scanErr = p.scanIdentifier("and", false)
			if scanErr != nil {
				return nil, scanErr
			}
		}
		if isAnd {
			if p.scanner.PeekChar(0) == '(' {
				return nil, p.scanner.Error(`Whitespace is required between "and" and "("`, -1, 0)
			}
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if op == nil {
				o := BooleanOperatorAnd
				op = &o
			}
			lg, lgErr := p.ifGroup()
			if lgErr != nil {
				return nil, lgErr
			}
			buffer.Write(" and ")
			lastInterp, err := lg.ToInterpolation(substitution)
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(lastInterp)
		} else {
			isOr := false
			if op == nil || *op != BooleanOperatorAnd {
				var scanErr error
				isOr, scanErr = p.scanIdentifier("or", false)
				if scanErr != nil {
					return nil, scanErr
				}
			}
			if isOr {
				if p.scanner.PeekChar(0) == '(' {
					return nil, p.scanner.Error(`Whitespace is required between "or" and "("`, -1, 0)
				}
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
				if op == nil {
					o := BooleanOperatorOr
					op = &o
				}
				lg, lgErr := p.ifGroup()
				if lgErr != nil {
					return nil, lgErr
				}
				lastGroup = lg
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
				buffer.Write(" or ")
				lastInterp, err := lastGroup.ToInterpolation(substitution)
				if err != nil {
					return nil, err
				}
				buffer.AddInterpolation(lastInterp)
			} else if ch := p.scanner.PeekChar(0); ch != ')' && ch != ':' && ch >= 0 && lastGroup.IsArbitrarySubstitution() {
				lg, lgErr := p.ifGroup()
				if lgErr != nil {
					return nil, lgErr
				}
				lastGroup = lg
				buffer.WriteCharCode(' ')
				lastInterp, err := lastGroup.ToInterpolation(substitution)
				if err != nil {
					return nil, err
				}
				buffer.AddInterpolation(lastInterp)
			} else if next, nextErr := p.tryArbitrarySubstitution(); nextErr != nil {
				return nil, nextErr
			} else if next != nil {
				lastGroup = next
				buffer.WriteCharCode(' ')
				lastInterp, err := lastGroup.ToInterpolation(substitution)
				if err != nil {
					return nil, err
				}
				buffer.AddInterpolation(lastInterp)
			} else {
				break
			}
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}

	precedingSpan, err := preceding.Span()
	if err != nil {
		return nil, err
	}
	precedingSpanStartLocation, err := precedingSpan.StartLocation()
	if err != nil {
		return nil, err
	}
	startPos := precedingSpanStartLocation.Offset
	endPos := p.scanner.Position()
	span := p.scanner.SpanFromTo(startPos, endPos)
	raw, rawErr := buffer.Interpolation(span)
	if rawErr != nil {
		return nil, p.error(rawErr.Error(), span, nil)
	}
	return NewIfConditionRaw(raw), nil
}

// namespacedExpression consumes an expression after a namespace dot. The
// scanner sits just past the `.` and start marks the namespace beginning. A
// `$name` becomes a namespaced variable (asserted public); otherwise a public
// identifier plus an argument invocation becomes a namespaced function.
// Dart marks this @protected; here it is shared the same way as an unexported
// method.
// Matches Dart: StylesheetParser.namespacedExpression (parse/stylesheet.dart).
func (p *StylesheetParser) namespacedExpression(namespace string, start sasscommon.LineScannerState) (Expression, error) {
	if p.scanner.PeekChar(0) == '$' {
		name, err := p.variableName()
		if err != nil {
			return nil, err
		}
		nsVarSpan, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.assertPublic(name, func() sasscommon.FileSpan { return nsVarSpan }); err != nil {
			return nil, err
		}
		return NewVariableExpression(name, nsVarSpan, &namespace), nil
	}

	ident, identErr := p.publicIdentifier()
	if identErr != nil {
		return nil, identErr
	}
	invocation, invErr := p.argumentInvocation(argumentInvocationOpts{})
	if invErr != nil {
		return nil, invErr
	}
	nsFuncSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewFunctionExpression(ident, invocation, nsFuncSpan, &namespace), nil
}

// trySpecialFunction consumes a function with special syntax and returns its
// expression, or nil with no error when name is ordinary. Start marks the
// position before the name. The `type(` prefix, vendor-prefixed
// `expression(...)` (which trial-parses SassScript first and warns when the
// argument is invalid or non-CSS, so interpolation preserves it), the
// calc-like family (`calc` when vendored, unprefixed `expression`, `element`),
// `progid:...(...)` (warned when vendored), and `url` (delegated to
// tryUrlContents) each accumulate raw interpolation text through the closing
// paren instead of parsing arguments as SassScript. Dart marks this @protected;
// here it is shared the same way as an unexported method.
// Matches Dart: StylesheetParser.trySpecialFunction (parse/stylesheet.dart).
func (p *StylesheetParser) trySpecialFunction(name string, start sasscommon.LineScannerState) (Expression, error) {
	var buffer *InterpolationBuffer
	normalized := unvendor.Unvendor(name)
	vendored := normalized != name

	if name == "type" && p.scanner.ScanChar('(') {
		buffer = &InterpolationBuffer{}
		buffer.Write(name)
		buffer.WriteCharCode('(')
	} else {
		switch normalized {
		case "expression":
			if vendored && p.scanner.ScanChar('(') {
				buffer = &InterpolationBuffer{}
				buffer.Write(name)
				buffer.WriteCharCode('(')

				beforeArg := p.scanner.State()
				invalidSassScript := false
				nonCssSassScript := false
				// Dart skips the probe entirely for empty `()` (no warning at
				// all); a missing `)` after an argument counts as invalid
				// SassScript rather than propagating (#2148, via the "prefer
				// interpolation" refactor 548e6604).
				if err := p.whitespace(true); err != nil {
					invalidSassScript = true
				} else if !p.scanner.ScanChar(')') {
					argument, argErr := p._expression(expressionOpts{})
					if argErr != nil {
						invalidSassScript = true
					} else if err := p.expectChar(')'); err != nil {
						invalidSassScript = true
					} else {
						plainCss, plainCssErr := argument.IsPlainCss(true)
						nonCssSassScript = plainCssErr != nil || !plainCss
					}
				}
				p.scanner.SetState(beforeArg)

				value, valErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
				if valErr != nil {
					return nil, valErr
				}
				buffer.AddInterpolation(value)
				if err := p.expectChar(')'); err != nil {
					return nil, err
				}
				buffer.WriteCharCode(')')

				if invalidSassScript || nonCssSassScript {
					suggestExpr := NewStringExpression(value, true)
					suggestInterp, err := suggestExpr.AsInterpolation(false, nil)
					if err != nil {
						return nil, err
					}
					suggestion, err := suggestInterp.String()
					if err != nil {
						return nil, err
					}
					message := fmt.Sprintf("Vendor-prefixed %s() functions will no longer have special parsing in a future release of Dart Sass. Once that happens, this argument will ", normalized)
					if invalidSassScript {
						message += "no longer be valid syntax. "
					} else {
						message += "be parsed as SassScript. "
					}
					message += fmt.Sprintf("To preserve current behavior:\n\n%s(#{%s})\n\nMore info: https://sass-lang.com/d/function-name", name, suggestion)
					warnSpan, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					p.warnings = append(p.warnings, ParseTimeWarning{
						Deprecation: deprecation.FunctionName,
						Message:     message,
						Span:        warnSpan,
					})
				}

				exprSpan, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				interp, interpErr := buffer.Interpolation(exprSpan)
				if interpErr != nil {
					exprSpan2, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					if err := p.error(interpErr.Error(), exprSpan2, nil); err != nil {
						return nil, err
					}
					return nil, interpErr
				}
				return NewStringExpression(interp, false), nil
			}

			if !vendored && p.scanner.ScanChar('(') {
				goto calcLike
			}
			return nil, nil

		case "calc":
			if vendored && p.scanner.ScanChar('(') {
				goto calcLike
			}
			return nil, nil

		case "element":
			if p.scanner.ScanChar('(') {
				goto calcLike
			}
			return nil, nil

		case "progid":
			if p.scanner.ScanChar(':') {
				buffer = &InterpolationBuffer{}
				buffer.Write(name)
				buffer.WriteCharCode(':')
				next := p.scanner.PeekChar(0)
				for next >= 0 && (util.IsAlphabetic(next) || next == '.') {
					ch, rdErr := p.readChar()
					if rdErr != nil {
						return nil, rdErr
					}
					buffer.WriteCharCode(ch)
					next = p.scanner.PeekChar(0)
				}
				if err := p.expectChar('('); err != nil {
					return nil, err
				}
				buffer.WriteCharCode('(')

				val, valErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
				if valErr != nil {
					return nil, valErr
				}
				buffer.AddInterpolation(val)
				if err := p.expectChar(')'); err != nil {
					return nil, err
				}
				buffer.WriteCharCode(')')

				if vendored {
					vendorSpan, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					interp, interpErr := buffer.Interpolation(vendorSpan)
					if interpErr != nil {
						vendorSpan2, err := p.spanFrom(start)
						if err != nil {
							return nil, err
						}
						if err := p.error(interpErr.Error(), vendorSpan2, nil); err != nil {
							return nil, err
						}
						return nil, interpErr
					}
					suggestExpr := NewStringExpression(interp, true)
					suggestInterp, err := suggestExpr.AsInterpolation(false, nil)
					if err != nil {
						return nil, err
					}
					suggestion, err := suggestInterp.String()
					if err != nil {
						return nil, err
					}
					p.warnings = append(p.warnings, ParseTimeWarning{
						Deprecation: deprecation.FunctionName,
						Message:     fmt.Sprintf("Vendor-prefixed progid:...() functions will no longer be supported in a future release of Dart Sass. To preserve current behavior:\n\n#{%s}\n\nMore info: https://sass-lang.com/d/function-name", suggestion),
						Span:        vendorSpan,
					})
				}

				progidSpan, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				interp, interpErr := buffer.Interpolation(progidSpan)
				if interpErr != nil {
					progidSpan2, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					if err := p.error(interpErr.Error(), progidSpan2, nil); err != nil {
						return nil, err
					}
					return nil, interpErr
				}
				return NewStringExpression(interp, false), nil
			}
			return nil, nil

		case "url":
			contents, urlErr := p.tryUrlContents(start, "", vendored)
			if urlErr != nil {
				return nil, urlErr
			}
			if contents != nil {
				return NewStringExpression(contents, false), nil
			}
			return nil, nil

		default:
			return nil, nil
		}
	}

calcLike:
	if buffer == nil {
		buffer = &InterpolationBuffer{}
		buffer.Write(name)
		buffer.WriteCharCode('(')
	}
	val, valErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
	if valErr != nil {
		return nil, valErr
	}
	buffer.AddInterpolation(val)
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	buffer.WriteCharCode(')')

	calcSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	interp, interpErr := buffer.Interpolation(calcSpan)
	if interpErr != nil {
		calcSpan2, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error(interpErr.Error(), calcSpan2, nil); err != nil {
			return nil, err
		}
		return nil, interpErr
	}
	return NewStringExpression(interp, false), nil
}

// tryUrlContents parses a `url(...)` body as raw URL text, returning nil with
// no error when the contents do not parse so the caller falls back to a real
// function expression. Start marks the position before the name; name defaults
// to `"url"` when empty; vendored trial-parses SassScript first to decide the
// deprecation warning, then rewinds. Like Ruby Sass, this parses a raw URL
// when possible and otherwise rewinds to the contents start for the fallback.
// Whitespace may only precede the closing paren, and the escapable/raw/inter-
// polation character set mirrors the base URL parser — most changes here
// should be mirrored there.
// Matches Dart: StylesheetParser._tryUrlContents (parse/stylesheet.dart).
func (p *StylesheetParser) tryUrlContents(start sasscommon.LineScannerState, name string, vendored bool) (*Interpolation, error) {
	beginningOfContents := p.scanner.State()
	if !p.scanner.ScanChar('(') {
		return nil, nil
	}

	var invalidSassScript bool
	if vendored {
		beforeArg := p.scanner.State()
		_, exprErr := p._expression(expressionOpts{})
		if exprErr != nil {
			invalidSassScript = true
		}
		p.scanner.SetState(beforeArg)
	}

	if err := p.whitespaceWithoutComments(true); err != nil {
		return nil, err
	}

	buffer := &InterpolationBuffer{}
	n := name
	if n == "" {
		n = "url"
	}
	buffer.Write(n)
	buffer.WriteCharCode('(')

	for {
		ch := p.scanner.PeekChar(0)
		switch {
		case ch < 0:
			p.scanner.SetState(beginningOfContents)
			return nil, nil

		case ch == '\\':
			s, err := p.escape(false)
			if err != nil {
				return nil, err
			}
			buffer.Write(s)

		case ch == '#' && p.scanner.PeekChar(1) == '{':
			expr, span, siErr := p.singleInterpolation()
			if siErr != nil {
				return nil, siErr
			}
			buffer.Add(expr, span)

		case ch == '!' || ch == '%' || ch == '&' || ch == '#' || (ch >= '*' && ch <= '~') || ch >= 0x80:
			c, rdErr := p.readChar()
			if rdErr != nil {
				return nil, rdErr
			}
			buffer.WriteCharCode(c)

		case util.IsWhitespace(ch):
			if err := p.whitespaceWithoutComments(true); err != nil {
				return nil, err
			}
			if p.scanner.PeekChar(0) != ')' {
				p.scanner.SetState(beginningOfContents)
				return nil, nil
			}

		case ch == ')':
			c, rdErr := p.readChar()
			if rdErr != nil {
				return nil, rdErr
			}
			buffer.WriteCharCode(c)
			urlSpan, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			result, urlErr := buffer.Interpolation(urlSpan)
			if urlErr != nil {
				urlSpan2, err := p.spanFrom(start)
				if err != nil {
					return nil, err
				}
				if err := p.error(urlErr.Error(), urlSpan2, nil); err != nil {
					return nil, err
				}
				return nil, urlErr
			}

			if vendored && invalidSassScript {
				// Dart uses "$name(#{$suggestion})" with name defaulting to null,
				// which produces "null" in string interpolation (identical to
				// hardcoded "null" here). The Dart caller also omits name:,
				// matching the "" name passed by Go's caller at trySpecialFunction.
				suggestExpr := NewStringExpression(result, true)
				suggestInterp, err := suggestExpr.AsInterpolation(false, nil)
				if err != nil {
					return nil, err
				}
				suggestion, err := suggestInterp.String()
				if err != nil {
					return nil, err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.FunctionName,
					Message:     fmt.Sprintf("Vendor-prefixed url() functions will no longer have special parsing in a future release of Dart Sass. Once that happens, this argument will be parsed as SassScript. To preserve current behavior:\n\nnull(#{%s})\n\nMore info: https://sass-lang.com/d/function-name", suggestion),
					Span:        urlSpan,
				})
			}

			return result, nil

		default:
			p.scanner.SetState(beginningOfContents)
			return nil, nil
		}
	}
}

// dynamicUrl consumes a `url` token that may hold SassScript: raw URL text
// when tryUrlContents accepts it, otherwise an interpolated `url(...)`
// function with a real argument list. Dart marks this @protected; here it is
// shared the same way as an unexported method.
// Matches Dart: StylesheetParser.dynamicUrl (parse/stylesheet.dart).
func (p *StylesheetParser) dynamicUrl() (Expression, error) {
	start := p.scanner.State()
	if err := p.expectIdentifier("url", "", true); err != nil {
		return nil, err
	}
	contents, urlErr := p.tryUrlContents(start, "url", false)
	if urlErr != nil {
		return nil, urlErr
	}
	if contents != nil {
		return NewStringExpression(contents, false), nil
	}

	invocation, invErr := p.argumentInvocation(argumentInvocationOpts{})
	if invErr != nil {
		return nil, invErr
	}
	dynUrlSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewInterpolatedFunctionExpression(
		NewInterpolationPlain("url", dynUrlSpan),
		invocation,
		dynUrlSpan,
	), nil
}
