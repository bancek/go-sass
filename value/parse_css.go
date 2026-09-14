// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/css.dart

import (
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

// CssParser parses plain CSS with Sass interpolation, matching Dart's
// CssParser in css.dart.
//
// It extends the SCSS parser and narrows it: Sass-only at-rules and
// interpolation are rejected, @import disallows interpolation, and function
// calls are checked against the disallowed-name set. Because Go has no
// virtual dispatch, the narrowing is wired through the base parser's func
// fields (identifierLikeFn, parenthesesFn, atRuleFn) plus the plainCss flag.
type CssParser struct {
	ScssParser
	// disallowedFunctionNames holds every global Sass function name except
	// the plain-CSS-safe subset (color constructors, min/max, and friends).
	// Calls to a listed name fail at parse time. Matches Dart's
	// _disallowedFunctionNames.
	disallowedFunctionNames map[string]bool
}

// NewCssParser creates a CssParser over contents, resolving relative URLs
// against url when provided. When parseSelectors is set, style rule
// selectors parse as interpolated selectors; disallowedFunctionNames lists
// the Sass function names rejected in plain CSS (nil keeps the field unset).
// Use of @use-family rules stays allowed, matching the SCSS base.
func NewCssParser(contents []byte, url *goUrl.URL, parseSelectors bool, disallowedFunctionNames map[string]bool) *CssParser {
	p := &CssParser{}
	if disallowedFunctionNames != nil {
		p.disallowedFunctionNames = disallowedFunctionNames
	}
	p.Parser = *NewParser(contents, url, nil)
	p.Parser.silentCommentFn = func() (bool, error) {
		return p.silentComment()
	}
	p.ScssParser.StylesheetParser.identifierLikeFn = func() (Expression, error) {
		return p.identifierLike()
	}
	p.ScssParser.StylesheetParser.parenthesesFn = func() (Expression, error) {
		return p.parentheses()
	}
	p.ScssParser.StylesheetParser.parseSelectors = parseSelectors
	p.ScssParser.StylesheetParser.isUseAllowed = true
	p.ScssParser.StylesheetParser.globalVariables = make(map[string]sasscommon.FileSpan)
	p.ScssParser.StylesheetParser.indented = false
	p.ScssParser.StylesheetParser.plainCss = true
	p.ScssParser.StylesheetParser.currentIndentation = func() int { return 0 }
	p.ScssParser.StylesheetParser.styleRuleSelector = func() (*Interpolation, error) {
		val, err := p.almostAnyValue(false)
		if err != nil {
			return nil, err
		}
		return val, nil
	}
	p.ScssParser.StylesheetParser.expectStatementSeparator = func(name string) error {
		if err := p.whitespaceWithoutComments(true); err != nil {
			return err
		}
		if p.scanner.IsDone() {
			return nil
		}
		next := p.scanner.PeekChar(0)
		if next == ';' || next == '}' {
			return nil
		}
		return p.expectChar(';')
	}
	p.ScssParser.StylesheetParser.atEndOfStatement = func() bool {
		next := p.scanner.PeekChar(0)
		return next < 0 || next == ';' || next == '}' || next == '{'
	}
	p.ScssParser.StylesheetParser.lookingAtChildren = func() (bool, error) {
		return p.scanner.PeekChar(0) == '{', nil
	}
	p.ScssParser.StylesheetParser.scanElse = func(ifIndentation int) (bool, error) {
		start := p.scanner.State()
		if err := p.whitespace(true); err != nil {
			return false, err
		}
		beforeAt := p.scanner.State()
		if p.scanner.ScanChar('@') {
			ok, err := p.scanIdentifier("else", true)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
			ok, err = p.scanIdentifier("elseif", true)
			if err != nil {
				return false, err
			}
			if ok {
				sp, err := p.spanFrom(beforeAt)
				if err != nil {
					return false, err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.Elseif,
					Message: "@elseif is deprecated and will not be supported in future Sass " +
						"versions.\n\nRecommendation: @else if",
					Span: sp,
				})
				if err := p.setPosition(p.scanner.Position() - 2); err != nil {
					return false, err
				}
				return true, nil
			}
		}
		p.scanner.SetState(start)
		return false, nil
	}
	p.ScssParser.StylesheetParser.children = func(child func() (Statement, error)) ([]Statement, error) {
		if err := p.expectChar('{'); err != nil {
			return nil, err
		}
		if err := p.whitespaceWithoutComments(true); err != nil {
			return nil, err
		}
		var children = make([]Statement, 0)
		for {
			switch ch := p.scanner.PeekChar(0); ch {
			case '$':
				st, err := p.variableDeclarationWithoutNamespace("", nil)
				if err != nil {
					return nil, err
				}
				children = append(children, st)
			case '/':
				switch p.scanner.PeekChar(1) {
				case '/':
					if _, err := p.scssSilentComment(); err != nil {
						return nil, err
					}
				case '*':
					st, err := p.scssLoudComment()
					if err != nil {
						return nil, err
					}
					children = append(children, st)
				default:
					st, err := child()
					if err != nil {
						return nil, err
					}
					children = append(children, st)
				}
			case ';':
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
			case '}':
				if err := p.expectChar('}'); err != nil {
					return nil, err
				}
				return children, nil
			case -1:
				return nil, p.scanner.Error("expected \"}\".", -1, 0)
			default:
				st, err := child()
				if err != nil {
					return nil, err
				}
				children = append(children, st)
			}
			if err := p.whitespaceWithoutComments(true); err != nil {
				return nil, err
			}
		}
	}
	p.ScssParser.StylesheetParser.statements = func(statement func() (Statement, error)) ([]Statement, error) {
		var stmts []Statement
		if err := p.whitespaceWithoutComments(true); err != nil {
			return nil, err
		}
		for !p.scanner.IsDone() {
			switch ch := p.scanner.PeekChar(0); ch {
			case '$':
				st, err := p.variableDeclarationWithoutNamespace("", nil)
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, st)
			case '/':
				switch p.scanner.PeekChar(1) {
				case '/':
					if _, err := p.scssSilentComment(); err != nil {
						return nil, err
					}
				case '*':
					st, err := p.scssLoudComment()
					if err != nil {
						return nil, err
					}
					stmts = append(stmts, st)
				default:
					st, err := statement()
					if err != nil {
						return nil, err
					}
					if st != nil {
						stmts = append(stmts, st)
					}
				}
			case ';':
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
			case -1:
				return stmts, nil
			default:
				st, err := statement()
				if err != nil {
					return nil, err
				}
				if st != nil {
					stmts = append(stmts, st)
				}
			}
			if err := p.whitespaceWithoutComments(true); err != nil {
				return nil, err
			}
		}
		return stmts, nil
	}
	p.ScssParser.StylesheetParser.atRuleFn = p.cssAtRule
	return p
}

// NOTE: this logic is largely duplicated in StylesheetParser.atRule. Most changes
// here should be mirrored there.
//
// cssAtRule consumes an at-rule from the restricted plain-CSS set: Sass-only
// names (plus any interpolated name) are rejected, import/function get
// CSS-specific handling, media/supports/-moz-document delegate to the shared
// rules, and anything else falls through to the unknown-at-rule parser.
// Matches Dart's CssParser.atRule.
func (p *CssParser) cssAtRule(child func() (Statement, error), root bool) (Statement, error) {
	start := p.scanner.State()
	if err := p.expectChar('@'); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	plain := name.AsPlain()
	if plain == nil {
		result, err := p.cssForbiddenAtRule(start)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	switch *plain {
	case "at-root", "content", "debug", "each", "error", "extend",
		"for", "if", "include", "mixin", "return", "warn", "while":
		result, err := p.cssForbiddenAtRule(start)
		if err != nil {
			return nil, err
		}
		return result, nil
	case "import":
		result, err := p.cssImportRule(start)
		if err != nil {
			return nil, err
		}
		return result, nil
	case "function":
		result, err := p.cssFunctionRule(start, name)
		if err != nil {
			return nil, err
		}
		return result, nil
	case "media":
		return p.mediaRule(start)
	case "-moz-document":
		return p.mozDocumentRule(start, name)
	case "supports":
		return p.supportsRule(start)
	default:
		return p.unknownAtRule(start, name)
	}
}

// cssForbiddenAtRule throws an error for a forbidden at-rule, consuming the
// remainder first so the scanner advances past it. Matches Dart's
// CssParser._forbiddenAtRule.
func (p *CssParser) cssForbiddenAtRule(start sasscommon.LineScannerState) (Statement, error) {
	if _, err := p.almostAnyValue(false); err != nil {
		return nil, err
	}
	sp, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return nil, p.error("This at-rule isn't allowed in plain CSS.", sp, nil)
}

// cssImportRule consumes a plain-CSS @import rule that disallows
// interpolation.
//
// [start] should point before the @. A u/U-leading URL goes through the
// dynamic-URL path (single-string interpolated-function forms are re-wrapped
// as interpolation; anything else reports "Unsupported plain CSS import.");
// otherwise the URL is a static interpolated string. Matches Dart's
// CssParser._cssImportRule.
func (p *CssParser) cssImportRule(start sasscommon.LineScannerState) (Statement, error) {
	urlStart := p.scanner.State()
	var urlText *Interpolation
	var urlErr error
	switch ch := p.scanner.PeekChar(0); {
	case ch == 'u' || ch == 'U':
		dynExpr, dynErr := p.dynamicUrl()
		if dynErr != nil {
			return nil, dynErr
		}
		switch e := dynExpr.(type) {
		case *StringExpression:
			urlText = e.Text
		case *InterpolatedFunctionExpression:
			al := e.Arguments()
			if len(al.Positional) == 1 &&
				al.Named.Len() == 0 &&
				al.Rest == nil &&
				al.KeywordRest == nil {
				if se, ok := al.Positional[0].(*StringExpression); ok {
					buf := &InterpolationBuffer{}
					buf.AddInterpolation(e.Name)
					buf.WriteCharCode('(')
					interp, err := se.AsInterpolation(false, nil)
					if err != nil {
						return nil, err
					}
					buf.AddInterpolation(interp)
					buf.WriteCharCode(')')
					sp, err := e.Span()
					if err != nil {
						return nil, err
					}
					urlText, urlErr = buf.Interpolation(sp)
				} else {
					sp, err := e.Span()
					if err != nil {
						return nil, err
					}
					return nil, p.error("Unsupported plain CSS import.", sp, nil)
				}
			} else {
				sp, err := e.Span()
				if err != nil {
					return nil, err
				}
				return nil, p.error("Unsupported plain CSS import.", sp, nil)
			}
		default:
			sp, err := e.Span()
			if err != nil {
				return nil, err
			}
			return nil, p.error("Unsupported plain CSS import.", sp, nil)
		}
	default:
		strExpr, strErr := p.interpolatedString()
		if strErr != nil {
			return nil, strErr
		}
		// Wraps the interpolated string as a static interpolation, matching
		// Dart's StringExpression(interpolatedString().asInterpolation(static:
		// true)).text.
		urlText, urlErr = strExpr.AsInterpolation(true, nil)
	}

	if urlErr != nil {
		sp, err := p.spanFrom(urlStart)
		if err != nil {
			return nil, err
		}
		return nil, p.error(urlErr.Error(), sp, nil)
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	modifiers, err := p.tryImportModifiers()
	if err != nil {
		return nil, err
	}
	if err := p.expectStatementSeparator("@import rule"); err != nil {
		return nil, err
	}
	importSpan, err := p.spanFrom(urlStart)
	if err != nil {
		return nil, err
	}
	ruleSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewImportRule([]Import{
		NewStaticImport(urlText, importSpan, modifiers),
	}, ruleSpan), nil
}

// cssFunctionRule consumes a plain CSS function declaration. [start] should
// point before the @. A `--`-prefixed name delegates to the unknown-at-rule
// parser; anything else is consumed and rejected as forbidden in plain CSS.
// Matches Dart's CssParser._cssFunctionRule.
func (p *CssParser) cssFunctionRule(start sasscommon.LineScannerState, atRuleName *Interpolation) (Statement, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if p.scanner.PeekChar(0) != '-' || p.scanner.PeekChar(1) != '-' {
		if _, err := p.almostAnyValue(false); err != nil {
			return nil, err
		}
		sp, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return nil, p.error("This at-rule isn't allowed in plain CSS.", sp, nil)
	}
	return p.unknownAtRule(start, atRuleName)
}

// silentComment rejects // comments in plain CSS, unless the parser is
// inside an expression. It consumes the comment first so the error span
// covers it. Matches Dart's CssParser.silentComment.
func (p *CssParser) silentComment() (bool, error) {
	if p.inExpression {
		return false, nil
	}
	start := p.scanner.State()
	if _, err := p.Parser.silentComment(); err != nil {
		return false, err
	}
	sp, err := p.spanFrom(start)
	if err != nil {
		return false, err
	}
	return true, p.error("Silent comments aren't allowed in plain CSS.", sp, nil)
}

// parentheses parses a parenthesized expression holding a single
// comma-free expression. Matches Dart's CssParser.parentheses.
//
// Expressions are only allowed within calculations, but we verify this at
// evaluation time.
func (p *CssParser) parentheses() (Expression, error) {
	start := p.scanner.State()
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	expression, err := p.expressionUntilComma(false)
	if err != nil {
		return nil, err
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	exprSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewParenthesizedExpression(expression, exprSpan), nil
}

// identifierLike parses an identifier-like expression in plain CSS. The
// identifier must be plain: interpolation fails. Special functions take
// priority; a dotted name parses only to throw the clearer module-namespace
// error; if(...) parses as a CSS if(); otherwise a bare identifier becomes
// an unquoted string and name(args) a function call with comma-separated
// arguments (var() allows an empty second argument). Names in the
// disallowed set fail. Matches Dart's CssParser.identifierLike.
func (p *CssParser) identifierLike() (Expression, error) {
	start := p.scanner.State()
	identifier, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	if identifier == nil {
		return nil, nil
	}
	plain := identifier.AsPlain() // CSS doesn't allow non-plain identifiers

	if plain == nil {
		sp, err := identifier.Span()
		if err != nil {
			return nil, err
		}
		return nil, p.error("Interpolation isn't allowed in CSS identifiers.", sp, nil)
	}

	lower := strings.ToLower(*plain)

	specialFn, err := p.trySpecialFunction(lower, start)
	if err != nil {
		return nil, err
	}
	if specialFn != nil {
		return specialFn, nil
	}

	beforeArguments := p.scanner.State()
	// namespacedExpression is just here to throw a clearer error.
	if p.scanner.ScanChar('.') {
		expr, err := p.namespacedExpression(*plain, start)
		if err != nil {
			return nil, err
		}
		return expr, nil
	}

	if lower == "if" && p.scanner.PeekChar(0) == '(' {
		ifExpr, err := p.ifExpression(start)
		if err != nil {
			return nil, err
		}
		return ifExpr, nil
	}

	if !p.scanner.ScanChar('(') {
		return NewStringExpression(identifier, false), nil
	}

	allowEmptySecondArg := lower == "var"
	var arguments []Expression
	if !p.scanner.ScanChar(')') {
		for {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if allowEmptySecondArg && len(arguments) == 1 && p.scanner.PeekChar(0) == ')' {
				arguments = append(arguments, NewStringExpressionPlain("", p.scanner.EmptySpan(), false))
				break
			}
			expr, err := p.expressionUntilComma(true)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, expr)
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if !p.scanner.ScanChar(',') {
				break
			}
		}
		if err := p.expectChar(')'); err != nil {
			return nil, err
		}
	}

	if p.disallowedFunctionNames[*plain] {
		sp, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return nil, p.error("This function isn't allowed in plain CSS.", sp, nil)
	}

	argsSpan, err := p.spanFrom(beforeArguments)
	if err != nil {
		return nil, err
	}
	funcSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewFunctionExpression(
		*plain,
		NewArgumentList(arguments, orderedmap.New[string, Expression](), map[string]sasscommon.FileSpan{}, argsSpan, nil, nil),
		funcSpan,
		nil,
	), nil
}

// namespacedExpression parses a dotted name only to throw the clearer
// "Module namespaces aren't allowed in plain CSS." error on its span.
// Matches Dart's CssParser.namespacedExpression.
func (p *CssParser) namespacedExpression(namespace string, start sasscommon.LineScannerState) (Expression, error) {
	expression, err := p.StylesheetParser.namespacedExpression(namespace, start)
	if err != nil {
		return nil, err
	}
	exprSpan, err := expression.Span()
	if err != nil {
		return nil, err
	}
	if err := p.error("Module namespaces aren't allowed in plain CSS.", exprSpan, nil); err != nil {
		return nil, err
	}
	return expression, nil
}
