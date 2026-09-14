// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (statements section: parse,
// single-production entries, statement dispatch, variable/declaration/
// style-rule disambiguation, style-rule children)

import (
	"errors"
	"fmt"
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// Parsing entry points

// Parse parses a full stylesheet: an optional byte-order mark, top-level
// statements, then end of input.
//
// A `@charset` rule is consumed and discarded here so the statement path
// always produces a real statement. The result carries the parser's
// plain-CSS flag, collected parse-time warnings, and the `!global`
// variables seen while parsing. Parse errors for non-stdin sources carry
// the source URL as a loaded URL, matching Stylesheet.parse.
func (p *StylesheetParser) Parse() (*Stylesheet, error) {
	var result *Stylesheet
	err := p.wrapSpanFormatException(func() error {
		start := p.scanner.State()
		p.scanner.ScanChar(0xFEFF)
		stmts, stmtsErr := p.statements(func() (Statement, error) {
			// Handle this specially so that the at-rule path below always
			// returns a non-nil statement.
			if p.scanner.Scan("@charset") {
				if err := p.whitespace(false); err != nil {
					return nil, err
				}
				if _, err := p.string(); err != nil {
					return nil, err
				}
				return nil, nil
			}
			stmt, err := p.statement(true)
			if err != nil {
				return nil, err
			}
			return stmt, nil
		})
		if stmtsErr != nil {
			return stmtsErr
		}
		if err := p.scanner.ExpectDone(); err != nil {
			return err
		}

		ws := make([]ParseTimeWarning, len(p.warnings))
		for i, w := range p.warnings {
			ws[i] = ParseTimeWarning{
				Deprecation: w.Deprecation,
				Message:     w.Message,
				Span:        w.Span,
			}
		}

		span, spErr := p.spanFrom(start)
		if spErr != nil {
			return spErr
		}
		result = NewStylesheetDetailed(stmts, span, ws, p.plainCss, p.globalVariables)
		return nil
	})
	if err != nil {
		// The stylesheet-level parse attaches the source URL as a loaded
		// URL (except for stdin parses with no real file to point at),
		// for both single- and multi-span format errors.
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			srcURL, urlErr := sfe.Span.SourceURL()
			if urlErr != nil {
				return nil, urlErr
			}
			if srcURL != nil && srcURL.String() != "stdin" {
				err = sasscommon.ThrowWithTrace(
					sfe.WithLoadedUrls([]*goUrl.URL{srcURL}),
					sfe,
				)
			}
		}
		if msfe, ok := errors.AsType[*sasscommon.MultiSpanSassFormatException](err); ok {
			srcURL, urlErr := msfe.Span.SourceURL()
			if urlErr != nil {
				return nil, urlErr
			}
			if srcURL != nil && srcURL.String() != "stdin" {
				err = sasscommon.ThrowWithTrace(
					msfe.WithLoadedUrls([]*goUrl.URL{srcURL}),
					msfe,
				)
			}
		}
		return nil, err
	}
	return result, nil
}

// parseParameterList parses a `@function name(params) {`-shaped parameter
// list as a single production. It exists for internal callers that parse
// generated signatures rather than stylesheet text.
func (p *StylesheetParser) parseParameterList() (*ParameterList, error) {
	return parseSingleProduction(p, func() (*ParameterList, error) {
		if err := p.expectChar('@'); err != nil {
			return nil, err
		}
		name, err := p.identifier(true, false)
		if err != nil {
			return nil, err
		}
		_ = name
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		name2, err := p.identifier(true, false)
		if err != nil {
			return nil, err
		}
		_ = name2
		parameters, err := p.parameterList()
		if err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if err := p.expectChar('{'); err != nil {
			return nil, err
		}
		return parameters, nil
	})
}

// parseExpression parses one expression as a single production, returning
// the warnings collected along the way.
func (p *StylesheetParser) parseExpression() (Expression, []ParseTimeWarning, error) {
	result, err := parseSingleProduction(p, func() (Expression, error) {
		expr, err := p._expression(expressionOpts{})
		if err != nil {
			return nil, err
		}
		return expr, nil
	})
	if err != nil {
		return nil, p.warnings, err
	}
	return result, p.warnings, nil
}

// ParseNumber parses one number as a single production and returns it as
// a SassNumber value.
func (p *StylesheetParser) ParseNumber() (SassNumber, error) {
	expr, err := parseSingleProduction(p, func() (*NumberExpression, error) {
		n, err := p.number()
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return nil, err
	}
	if expr == nil {
		return nil, nil
	}
	if expr.Unit != nil {
		return NewSassNumber(expr.Value, expr.Unit), nil
	}
	return NewSassNumber(expr.Value, nil), nil
}

// parseVariableDeclaration parses one variable declaration (with or
// without a module namespace) as a single production, returning the
// warnings collected along the way.
func (p *StylesheetParser) parseVariableDeclaration() (*VariableDeclaration, []ParseTimeWarning, error) {
	result, err := parseSingleProduction(p, func() (*VariableDeclaration, error) {
		if p.lookingAtIdentifier(nil) {
			return p.variableDeclarationWithNamespace()
		}
		return p.variableDeclarationWithoutNamespace("", nil)
	})
	if err != nil {
		return nil, p.warnings, err
	}
	return result, p.warnings, nil
}

// parseUseRule parses one `@use` rule as a single production, returning
// the warnings collected along the way.
func (p *StylesheetParser) parseUseRule() (*UseRule, []ParseTimeWarning, error) {
	result, err := parseSingleProduction(p, func() (*UseRule, error) {
		start := p.scanner.State()
		if err := p.expectChar('@'); err != nil {
			return nil, err
		}
		if err := p.expectIdentifier("use", "", true); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		result, err := p.useRule(start)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
	if err != nil {
		return nil, p.warnings, err
	}
	return result, p.warnings, nil
}

// ParseSignature parses a function signature of the format allowed by Node Sass's
// functions option and returns its name and declaration.
//
// If requireParens is false, this allows parentheses to be omitted.
// Throws a SassFormatException if parsing fails.
//
// Matches Dart: parseSignature in lib/src/utils.dart
func ParseSignature(signature string, requireParens bool) (string, *ParameterList, error) {
	p := NewStylesheetParser([]byte(signature), nil, false, nil)
	name, params, err := p.parseSignature(requireParens)
	if err != nil {
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			return "", nil, sasscommon.ThrowWithTrace(
				&sasscommon.SassFormatException{
					Message: fmt.Sprintf(`Invalid signature "%s": %s`, signature, sfe.Message),
					Span:    sfe.Span,
				},
				sfe,
			)
		}
		return "", nil, &sasscommon.SassFormatException{
			Message: fmt.Sprintf(`Invalid signature "%s": %s`, signature, err),
			Span:    sasscommon.BogusSpan,
		}
	}
	return name, params, nil
}

// ParseSignature parses one function signature as a single production,
// allowing parentheses to be omitted when requireParens is false.
func (p *StylesheetParser) ParseSignature(requireParens bool) (string, *ParameterList, error) {
	return p.parseSignature(requireParens)
}

// parseSignature parses a bare `name(params)` signature with no leading
// `@function`, as a single production wrapped in span-format-exception
// handling. An empty parameter list results when parentheses are optional
// and absent.
func (p *StylesheetParser) parseSignature(requireParens bool) (string, *ParameterList, error) {
	var name string
	var parameters *ParameterList
	err := p.wrapSpanFormatException(func() error {
		var err error
		name, err = p.identifier(false, false)
		if err != nil {
			return err
		}
		if requireParens || p.scanner.PeekChar(0) == '(' {
			parameters, err = p.parameterList()
			if err != nil {
				return err
			}
		} else {
			parameters = NewParameterListEmpty(p.scanner.EmptySpan())
		}
		if err := p.scanner.ExpectDone(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	return name, parameters, nil
}

// parseSingleProduction parses production as the entire scanner contents:
// it runs the production, expects end of input, and wraps span-format
// errors — all in one place so the thin entry points above stay trivial.
func parseSingleProduction[T any](p *StylesheetParser, production func() (T, error)) (T, error) {
	var result T
	err := p.wrapSpanFormatException(func() error {
		var prodErr error
		result, prodErr = production()
		if prodErr != nil {
			return prodErr
		}
		if err := p.scanner.ExpectDone(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

// statement consumes one statement allowed at the top level of the
// stylesheet or within nested style and at-rules.
//
// When root is true, at-rules permitted only at the stylesheet root are
// parsed; otherwise they error. A leading `+`/`=` dispatches to the
// indented-syntax include/mixin rules only when followed by an identifier
// start (otherwise it is a style rule). A lone `}` is always an error.
// Anywhere else, the surrounding context decides: inside a style rule,
// unknown at-rule, mixin, or content block, declarations are possible so
// the declaration-or-style-rule path runs; at the top level only variable
// declarations compete with style rules.
func (p *StylesheetParser) statement(root bool) (Statement, error) {
	switch ch := p.scanner.PeekChar(0); {

	case ch == '@':
		if p.atRuleFn != nil {
			stmt, err := p.atRuleFn(func() (Statement, error) { return p.statement(false) }, root)
			if err != nil {
				return nil, err
			}
			return stmt, nil
		}
		stmt, err := p.atRule(func() (Statement, error) { return p.statement(false) }, root)
		if err != nil {
			return nil, err
		}
		return stmt, nil

	case ch == '+':
		// In the indented syntax `+name` is an include; in SCSS (or when
		// no identifier follows) the `+` opens a style rule instead.
		if !p.indented || !p.lookingAtIdentifier(new(1)) {
			stmt, err := p.styleRule(nil, nil)
			if err != nil {
				return nil, err
			}
			return stmt, nil
		}
		p.isUseAllowed = false
		start := p.scanner.State()
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		stmt, err := p.includeRule(start)
		if err != nil {
			return nil, err
		}
		return stmt, nil

	case ch == '=':
		// In the indented syntax `=name` is a mixin definition; in SCSS
		// the `=` opens a style rule instead.
		if !p.indented {
			stmt, err := p.styleRule(nil, nil)
			if err != nil {
				return nil, err
			}
			return stmt, nil
		}
		p.isUseAllowed = false
		start := p.scanner.State()
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		stmt, err := p.mixinRule(start)
		if err != nil {
			return nil, err
		}
		return stmt, nil

	case ch == '}':
		return nil, p.scanner.Error("unmatched \"}\".", -1, 1)

	default:
		if p.inStyleRule || p.inUnknownAtRule || p.inMixin || p.inContentBlock {
			return p.declarationOrStyleRule()
		}
		return p.variableDeclarationOrStyleRule()
	}
}

// variableDeclarationWithNamespace consumes a namespaced variable
// declaration of the form `namespace.$name: value`.
func (p *StylesheetParser) variableDeclarationWithNamespace() (*VariableDeclaration, error) {
	start := p.scanner.State()
	namespace, err := p.identifier(true, false)
	if err != nil {
		return nil, err
	}
	if err := p.expectChar('.'); err != nil {
		return nil, err
	}
	return p.variableDeclarationWithoutNamespace(namespace, &start)
}

// variableDeclarationWithoutNamespace consumes a variable declaration.
//
// It never consumes a namespace itself, but when namespace is passed the
// declaration is recorded under it (and the name must be public). Sass
// variables are rejected in plain CSS. Trailing `!default`/`!global` flags
// are folded in: a repeated flag warns about duplicate flags, and
// `!global` on a module-namespaced variable is an error. A `!global`
// assignment is recorded in the stylesheet's global-variable table at parse
// time — even before evaluation — so the generated module always exposes
// the same variable names. The silent comment seen just before the
// declaration attaches to it as documentation.
func (p *StylesheetParser) variableDeclarationWithoutNamespace(namespace string, start *sasscommon.LineScannerState) (*VariableDeclaration, error) {
	var precedingComment *SilentComment
	if p.lastSilentComment != nil {
		precedingComment = p.lastSilentComment
		p.lastSilentComment = nil
	}

	var st *sasscommon.LineScannerState
	if start != nil {
		st = start
	} else {
		s := p.scanner.State()
		st = &s
	}

	name, err := p.variableName()
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		stSpan, spErr := p.spanFrom(*st)
		if spErr != nil {
			return nil, spErr
		}
		if err := p.assertPublic(name, func() sasscommon.FileSpan { return stSpan }); err != nil {
			return nil, err
		}
	}

	if p.plainCss {
		errSpan, spErr := p.spanFrom(*st)
		if spErr != nil {
			return nil, spErr
		}
		if err := p.error("Sass variables aren't allowed in plain CSS.", errSpan, nil); err != nil {
			return nil, err
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

	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}

	var guarded, global bool
	flagStart := p.scanner.State()
	for p.scanner.ScanChar('!') {
		id, err := p.identifier(true, false)
		if err != nil {
			return nil, err
		}
		flagSpan, spErr := p.spanFrom(flagStart)
		if spErr != nil {
			return nil, spErr
		}
		switch id {
		case "default":
			if guarded {
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.DuplicateVarFlags,
					Message:     "!default should only be written once for each variable.\nThis will be an error in Dart Sass 2.0.0.",
					Span:        flagSpan,
				})
			}
			guarded = true
		case "global":
			if namespace != "" {
				if err := p.error("!global isn't allowed for variables in other modules.", flagSpan, nil); err != nil {
					return nil, err
				}
			} else if global {
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.DuplicateVarFlags,
					Message:     "!global should only be written once for each variable.\nThis will be an error in Dart Sass 2.0.0.",
					Span:        flagSpan,
				})
			}
			global = true
		default:
			if err := p.error("Invalid flag name.", flagSpan, nil); err != nil {
				return nil, err
			}
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		flagStart = p.scanner.State()
	}

	if err := p.expectStatementSeparator("variable declaration"); err != nil {
		return nil, err
	}
	var ns *string
	if namespace != "" {
		ns = &namespace
	}
	declSpan, spErr := p.spanFrom(*st)
	if spErr != nil {
		return nil, spErr
	}
	decl, err := NewVariableDeclaration(name, value, declSpan, ns, guarded, global, precedingComment)
	if err != nil {
		if cfgErr := p.error(err.Error(), declSpan, nil); cfgErr != nil {
			return nil, cfgErr
		}
		return nil, err
	}
	if global {
		if _, exists := p.globalVariables[name]; !exists {
			declSpan, spErr := decl.Span()
			if spErr != nil {
				return nil, spErr
			}
			p.globalVariables[name] = declSpan
		}
	}
	return decl, nil
}

// variableDeclarationOrStyleRule consumes a namespaced variable
// declaration or a style rule.
//
// Plain CSS has no variables, so this goes straight to a style rule. The
// indented syntax lets one leading backslash force a style rule (a legacy
// property-syntax escape hatch; old property syntax itself is unsupported
// but the backslash is cheap to honor). Otherwise, anything that does not
// start with an identifier must be a selector; the ambiguous remainder is
// probed as a variable-or-interpolation and, when it is not a declaration,
// its text seeds the style rule that follows.
func (p *StylesheetParser) variableDeclarationOrStyleRule() (Statement, error) {
	if p.plainCss {
		stmt, err := p.styleRule(nil, nil)
		if err != nil {
			return nil, err
		}
		return stmt, nil
	}

	if p.indented && p.scanner.ScanChar('\\') {
		stmt, err := p.styleRule(nil, nil)
		if err != nil {
			return nil, err
		}
		return stmt, nil
	}

	if !p.lookingAtIdentifier(nil) {
		stmt, err := p.styleRule(nil, nil)
		if err != nil {
			return nil, err
		}
		return stmt, nil
	}

	start := p.scanner.State()
	variableOrInterpolation, err := p.variableDeclarationOrInterpolation()
	if err != nil {
		return nil, err
	}
	if vd, ok := variableOrInterpolation.(*VariableDeclaration); ok {
		return vd, nil
	}
	interp := variableOrInterpolation.(*Interpolation)
	buf := &InterpolationBuffer{}
	buf.AddInterpolation(interp)
	stmt, err := p.styleRule(buf, &start)
	if err != nil {
		return nil, err
	}
	return stmt, nil
}

// declarationOrStyleRule consumes a variable declaration, a property
// declaration, or a style rule.
//
// Inside a style rule's children all three can start with a bare
// identifier, so the buffer-or-declaration probe below disambiguates and
// whatever text it leaves behind seeds the style rule. The criteria, from
// the Dart source: an identifier followed by `.$` is always a variable
// declaration (`.$` appears in nothing else); without an
// identifier-followed-by-colon shape it is a selector (plus minor cases
// for declaration hacks); a colon followed by another colon is a selector;
// otherwise a colon followed by something other than interpolation or an
// identifier start is a declaration; when interpolation or an identifier
// start follows, a declaration parse is attempted and, on failure, the
// input backtracks into selector parsing; and a declaration value followed
// by `{` is reparsed as a selector anyway, so `.foo:bar {` never becomes a
// property with nested properties beneath it.
func (p *StylesheetParser) declarationOrStyleRule() (Statement, error) {
	if p.indented && p.scanner.ScanChar('\\') {
		stmt, err := p.styleRule(nil, nil)
		if err != nil {
			return nil, err
		}
		return stmt, nil
	}

	start := p.scanner.State()
	declarationOrBuffer, err := p.declarationOrBuffer()
	if err != nil {
		return nil, err
	}
	if stmt, ok := declarationOrBuffer.(Statement); ok {
		return stmt, nil
	}
	if declarationOrBuffer == nil {
		return nil, nil
	}
	buf := declarationOrBuffer.(*InterpolationBuffer)
	stmt, err := p.styleRule(buf, &start)
	if err != nil {
		return nil, err
	}
	return stmt, nil
}

// declarationOrBuffer tries to parse a variable or property declaration,
// returning the value parsed so far when it fails.
//
// A returned InterpolationBuffer means no declaration was consumed and the
// caller should attempt selector parsing with the buffered text; a
// Declaration or VariableDeclaration means one was consumed.
func (p *StylesheetParser) declarationOrBuffer() (any, error) {
	start := p.scanner.State()
	nameBuffer := &InterpolationBuffer{}

	var startsWithPunctuation bool
	if p.lookingAtPotentialPropertyHack() {
		// A leading `:`, `*`, or similar hack character belongs to the
		// property name, not the selector, so buffer it (plus any
		// whitespace after it) before probing the identifier.
		startsWithPunctuation = true
		ch, err := p.readChar()
		if err != nil {
			return nil, err
		}
		nameBuffer.WriteCharCode(ch)
		text, err := p.rawText(func() error { return p.whitespace(false) })
		if err != nil {
			return nil, err
		}
		nameBuffer.Write(text)
	}

	if !p.lookingAtInterpolatedIdentifier() {
		return nameBuffer, nil
	}

	var variableOrInterpolation any
	if startsWithPunctuation {
		var err error
		variableOrInterpolation, err = p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		variableOrInterpolation, err = p.variableDeclarationOrInterpolation()
		if err != nil {
			return nil, err
		}
	}
	if vd, ok := variableOrInterpolation.(*VariableDeclaration); ok {
		return vd, nil
	}
	nameBuffer.AddInterpolation(variableOrInterpolation.(*Interpolation))

	p.isUseAllowed = false
	if p.scanner.PeekChar(0) == '/' && p.scanner.PeekChar(1) == '*' {
		text, err := p.rawText(func() error { return p.loudComment() })
		if err != nil {
			return nil, err
		}
		nameBuffer.Write(text)
	}

	midBuffer := &strings.Builder{}
	midText, err := p.rawText(func() error { return p.whitespace(false) })
	if err != nil {
		return nil, err
	}
	midBuffer.WriteString(midText)
	beforeColon := p.scanner.State()
	if !p.scanner.ScanChar(':') {
		// No colon: this is selector text. Preserve a separating space
		// when whitespace ran between the name and whatever follows.
		if midBuffer.Len() > 0 {
			nameBuffer.WriteCharCode(' ')
		}
		return nameBuffer, nil
	}
	midBuffer.WriteByte(':')

	interpSpan, spErr := p.spanFromTo(start, &beforeColon)
	if spErr != nil {
		return nil, spErr
	}
	name, nameErr := nameBuffer.Interpolation(interpSpan)
	if nameErr != nil {
		errSpan, spErr := p.spanFrom(start)
		if spErr != nil {
			return nil, spErr
		}
		if err := p.error(nameErr.Error(), errSpan, nil); err != nil {
			return nil, err
		}
		return nil, nil
	}
	isCustomProperty := strings.HasPrefix(name.InitialPlain(), "--")
	// Custom properties (`--*`) — and the plain-CSS `@function` `result`
	// property — parse as declarations no matter what follows, with raw
	// interpolated values rather than SassScript.
	if isCustomProperty || (p.inPlainCssFunction && name.AsPlain() != nil && strings.EqualFold(*name.AsPlain(), "result")) {
		var valueExpr *StringExpression
		if p.atEndOfStatement() {
			valueExpr = NewStringExpression(NewInterpolationPlain("", p.scanner.EmptySpan()), false)
		} else {
			val, valErr := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: false, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: false, consumeNewlines: false})
			if valErr != nil {
				return nil, valErr
			}
			valueExpr = NewStringExpression(val, false)
		}
		sepName := "custom property"
		if !isCustomProperty {
			sepName = "@function result"
		}
		if err := p.expectStatementSeparator(sepName); err != nil {
			return nil, err
		}
		retSpan, spErr := p.spanFrom(start)
		if spErr != nil {
			return nil, spErr
		}
		return NewDeclarationNotSassScript(name, valueExpr, retSpan), nil
	}

	if p.scanner.ScanChar(':') {
		// A second colon means a pseudo-element selector (`::before`),
		// not a declaration: hand the buffered text back for selector
		// parsing.
		nameBuffer.Write(midBuffer.String())
		nameBuffer.WriteCharCode(':')
		return nameBuffer, nil
	} else if p.indented && p.lookingAtInterpolatedIdentifier() {
		// In the indented syntax `foo:bar` is always a selector rather
		// than a property.
		nameBuffer.Write(midBuffer.String())
		return nameBuffer, nil
	}

	postColonWhitespace, err := p.rawText(func() error { return p.whitespace(false) })
	if err != nil {
		return nil, err
	}

	nested, err := p.tryDeclarationChildren(name, start, nil)
	if err != nil {
		return nil, err
	}
	if nested != nil {
		return nested, nil
	}

	midBuffer.WriteString(postColonWhitespace)
	couldBeSelector := postColonWhitespace == "" && p.lookingAtInterpolatedIdentifier()

	beforeDeclaration := p.scanner.State()
	value, valueErr := func() (Expression, error) {
		val, err := p._expression(expressionOpts{})
		if err != nil {
			return nil, err
		}

		hasChildren, err := p.lookingAtChildren()
		if err != nil {
			return nil, err
		}
		if hasChildren {
			// Ambiguous-with-selector properties cannot carry nested
			// children, so force an error here: it is caught below and
			// the text is reparsed as a selector.
			if couldBeSelector {
				if err := p.expectStatementSeparator(""); err != nil {
					return nil, err
				}
			}
		} else if !p.atEndOfStatement() {
			// Force an error when no valid end-of-property character
			// follows, without consuming it, so the text is reparsed
			// as a selector below.
			if err := p.expectStatementSeparator(""); err != nil {
				return nil, err
			}
		}
		return val, nil
	}()

	if valueErr != nil {
		if !couldBeSelector {
			return nil, valueErr
		}
		// The declaration parse failed but this could still be a
		// selector: rewind and slurp the remainder as selector text.
		// A value followed by a semicolon (outside the indented syntax)
		// is definitely a property, so re-raise in that case.
		p.scanner.SetState(beforeDeclaration)
		additional, err := p.almostAnyValue(false)
		if err != nil {
			return nil, err
		}
		if !p.indented && p.scanner.PeekChar(0) == ';' {
			return nil, nil
		}
		nameBuffer.Write(midBuffer.String())
		nameBuffer.AddInterpolation(additional)
		return nameBuffer, nil
	}

	nested, err = p.tryDeclarationChildren(name, start, value)
	if err != nil {
		return nil, err
	}
	if nested != nil {
		return nested, nil
	}
	if err := p.expectStatementSeparator(""); err != nil {
		return nil, err
	}
	retSpan, spErr := p.spanFrom(start)
	if spErr != nil {
		return nil, spErr
	}
	return NewDeclaration(name, value, retSpan), nil
}

// variableDeclarationOrInterpolation tries to parse a namespaced variable
// declaration, returning the value parsed so far when it fails.
//
// A returned Interpolation means no declaration was consumed and the
// caller should attempt property-declaration or selector parsing with it;
// a VariableDeclaration means one was consumed.
func (p *StylesheetParser) variableDeclarationOrInterpolation() (any, error) {
	if !p.lookingAtIdentifier(nil) {
		interp, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		return interp, nil
	}

	start := p.scanner.State()
	ident, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	if p.scanner.PeekChar(0) == '.' && p.scanner.PeekChar(1) == '$' {
		// `ident.$` starts a namespaced declaration (`.$` appears in
		// nothing else); anything else accumulates into an interpolation.
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		decl, err := p.variableDeclarationWithoutNamespace(ident, &start)
		if err != nil {
			return nil, err
		}
		return decl, nil
	}
	buf := &InterpolationBuffer{}
	buf.Write(ident)
	if p.lookingAtInterpolatedIdentifierBody() {
		// Consume the rest of an interpolated identifier here so callers
		// do not have to.
		interp, identErr := p.interpolatedIdentifier()
		if identErr != nil {
			return nil, identErr
		}
		buf.AddInterpolation(interp)
	}
	interpSpan, spErr := p.spanFrom(start)
	if spErr != nil {
		return nil, spErr
	}
	result, err := buf.Interpolation(interpSpan)
	if err != nil {
		errSpan, spErr := p.spanFrom(start)
		if spErr != nil {
			return nil, spErr
		}
		if cfgErr := p.error(err.Error(), errSpan, nil); cfgErr != nil {
			return nil, cfgErr
		}
		return nil, nil
	}
	return result, nil
}

// styleRule consumes a style rule, optionally seeded with a buffer holding
// text already parsed while disambiguating a declaration from a selector.
//
// Seeing a style rule closes the `@use`/`@forward` window. When parsed
// selectors are enabled the scanner rewinds to the rule start and the
// selector parses as a real selector list; otherwise the selector stays a
// raw interpolation (merged with the buffer when one was passed). An empty
// selector is an error.
func (p *StylesheetParser) styleRule(buffer *InterpolationBuffer, start_ *sasscommon.LineScannerState) (Statement, error) {
	p.isUseAllowed = false
	var start sasscommon.LineScannerState
	if start_ != nil {
		start = *start_
	} else {
		start = p.scanner.State()
	}

	if p.parseSelectors {
		if start_ != nil {
			p.scanner.SetState(start)
		}
		selectorList, err := p.selectorList()
		if err != nil {
			return nil, err
		}
		return p.withStyleRuleChildren(selectorList, start, func(children []Statement, span sasscommon.FileSpan) *StyleRule {
			return NewStyleRuleWithParsedSelector(selectorList, children, span)
		})
	}

	interpolation, err := p.styleRuleSelector()
	if err != nil {
		return nil, err
	}
	if interpolation == nil {
		return nil, nil
	}
	if buffer != nil {
		buffer.AddInterpolation(interpolation)
		var bufErr error
		interpSpan, spErr := p.spanFrom(start)
		if spErr != nil {
			return nil, spErr
		}
		interpolation, bufErr = buffer.Interpolation(interpSpan)
		if bufErr != nil {
			errSpan, spErr := p.spanFrom(start)
			if spErr != nil {
				return nil, spErr
			}
			if err := p.error(bufErr.Error(), errSpan, nil); err != nil {
				return nil, err
			}
			return nil, bufErr
		}
	}
	if len(interpolation.Contents) == 0 {
		return nil, p.scanner.Error("expected \"}\".", -1, 0)
	}

	return p.withStyleRuleChildren(interpolation, start, func(children []Statement, span sasscommon.FileSpan) *StyleRule {
		return NewStyleRule(interpolation, children, span)
	})
}

// withStyleRuleChildren consumes a style rule's children via withChildren
// and passes them, plus the span from start to the end of the child block,
// to create.
//
// The in-style-rule flag is set while children parse so nested statements
// take the declaration path. In the indented syntax a selector with no
// children warns that it has no properties and will not render.
func (p *StylesheetParser) withStyleRuleChildren(nodeWithSpan any, start sasscommon.LineScannerState, create func([]Statement, sasscommon.FileSpan) *StyleRule) (*StyleRule, error) {
	wasInStyleRule := p.inStyleRule
	// NOTE: Dart only restores _inStyleRule in the callback (not on exception).
	// We match that here for Dart parity.
	p.inStyleRule = true

	nodeSpan, spErr := nodeWithSpan.(sasscommon.AstNode).Span()
	if spErr != nil {
		return nil, spErr
	}

	return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*StyleRule, error) {
		if p.indented && len(children) == 0 {
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: nil,
				Message:     "This selector doesn't have any properties and won't be rendered.",
				Span:        nodeSpan,
			})
		}
		p.inStyleRule = wasInStyleRule
		return create(children, span), nil
	})
}

// propertyOrVariableDeclaration consumes either a property declaration or
// a namespaced variable declaration.
//
// It runs only nested beneath other declarations; the general case goes
// through declarationOrStyleRule instead. Property-hack prefixes are
// honored here too. Custom-property names (`--*`) may not nest, and plain
// CSS parses the name as interpolation only (no variable declarations).
func (p *StylesheetParser) propertyOrVariableDeclaration() (Statement, error) {
	start := p.scanner.State()

	var name *Interpolation
	if p.lookingAtPotentialPropertyHack() {
		nameBuffer := &InterpolationBuffer{}
		ch, err := p.readChar()
		if err != nil {
			return nil, err
		}
		nameBuffer.WriteCharCode(ch)
		text, err := p.rawText(func() error { return p.whitespace(false) })
		if err != nil {
			return nil, err
		}
		nameBuffer.Write(text)
		interp, interpErr := p.interpolatedIdentifier()
		if interpErr != nil {
			return nil, interpErr
		}
		nameBuffer.AddInterpolation(interp)
		var nameErr error
		interpSpan, spErr := p.spanFrom(start)
		if spErr != nil {
			return nil, spErr
		}
		name, nameErr = nameBuffer.Interpolation(interpSpan)
		if nameErr != nil {
			errSpan, spErr := p.spanFrom(start)
			if spErr != nil {
				return nil, spErr
			}
			if cfgErr := p.error(nameErr.Error(), errSpan, nil); cfgErr != nil {
				return nil, cfgErr
			}
			return nil, nameErr
		}
	} else if !p.plainCss {
		variableOrInterpolation, err := p.variableDeclarationOrInterpolation()
		if err != nil {
			return nil, err
		}
		if vd, ok := variableOrInterpolation.(*VariableDeclaration); ok {
			return vd, nil
		}
		name = variableOrInterpolation.(*Interpolation)
	} else {
		interp, interpErr := p.interpolatedIdentifier()
		if interpErr != nil {
			return nil, interpErr
		}
		name = interp
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	if err := p.expectChar(':'); err != nil {
		return nil, err
	}

	if strings.HasPrefix(name.InitialPlain(), "--") {
		nameSpan, spErr := name.Span()
		if spErr != nil {
			return nil, spErr
		}
		if err := p.error("Declarations whose names begin with \"--\" may not be nested.", nameSpan, nil); err != nil {
			return nil, err
		}
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	nested, err := p.tryDeclarationChildren(name, start, nil)
	if err != nil {
		return nil, err
	}
	if nested != nil {
		return nested, nil
	}

	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	nested, err = p.tryDeclarationChildren(name, start, value)
	if err != nil {
		return nil, err
	}
	if nested != nil {
		return nested, nil
	}
	if err := p.expectStatementSeparator(""); err != nil {
		return nil, err
	}
	retSpan, spErr := p.spanFrom(start)
	if spErr != nil {
		return nil, spErr
	}
	return NewDeclaration(name, value, retSpan), nil
}

// tryDeclarationChildren tries parsing nested children beneath the
// already-parsed declaration name, returning nil when none follow.
//
// Nested declarations are rejected in plain CSS. When children exist,
// value becomes the declaration's own value (nil for a value-less parent).
func (p *StylesheetParser) tryDeclarationChildren(name *Interpolation, start sasscommon.LineScannerState, value Expression) (*Declaration, error) {
	hasChildren, err := p.lookingAtChildren()
	if err != nil {
		return nil, err
	}
	if !hasChildren {
		return nil, nil
	}
	if p.plainCss {
		return nil, p.scanner.Error("Nested declarations aren't allowed in plain CSS.", -1, 0)
	}
	return withChildren(p, func() (Statement, error) { return p.declarationChild() }, start, func(children []Statement, span sasscommon.FileSpan) (*Declaration, error) {
		return NewDeclarationNested(name, children, span, value), nil
	})
}

// declarationChild consumes one statement allowed within a declaration:
// `@`-rules go to the declaration at-rule path, everything else parses as
// a nested property or variable declaration.
func (p *StylesheetParser) declarationChild() (Statement, error) {
	if p.scanner.PeekChar(0) == '@' {
		return p.declarationAtRule()
	}
	stmt, err := p.propertyOrVariableDeclaration()
	if err != nil {
		return nil, err
	}
	return stmt, nil
}
