// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (at-rule parsing: atRule
// dispatch, declaration/function children, individual @-rule productions,
// unknown/disallowed at-rules, parameter lists)

import (
	"fmt"
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/unvendor"
)

// At-rules.
//
// The dispatch in atRule is largely duplicated in the plain-CSS parser's
// at-rule handling; most changes here should be mirrored there.

// atRule consumes an at-rule allowed at any nesting level of the document.
// child consumes rules permitted specifically in the caller's context; when
// root is true, the root-only rules (@forward, @use) are accepted as well.
// The name may contain interpolation, in which case the rule falls through
// to unknownAtRule. isUseAllowed is cleared for every rule except @forward
// and @use: it is unconditionally reset and then restored from wasUseAllowed
// for those two, so the name only has to be compared once.
func (p *StylesheetParser) atRule(child func() (Statement, error), root bool) (Statement, error) {
	start := p.scanner.State()
	if err := p.expectChar('@'); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}

	wasUseAllowed := p.isUseAllowed
	p.isUseAllowed = false

	plain := name.AsPlain()
	if plain == nil {
		return p.unknownAtRule(start, name)
	}

	switch *plain {
	case "at-root":
		return p.atRootRule(start)
	case "content":
		return p.contentRule(start)
	case "debug":
		return p.debugRule(start)
	case "each":
		return p.eachRule(start, child)
	case "else":
		return p.disallowedAtRule(start)
	case "error":
		return p.errorRule(start)
	case "extend":
		return p.extendRule(start)
	case "for":
		return p.forRule(start, child)
	case "forward":
		p.isUseAllowed = wasUseAllowed
		if !root {
			return p.disallowedAtRule(start)
		}
		return p.forwardRule(start)
	case "function":
		return p.functionRule(start, name)
	case "if":
		return p.ifRule(start, child)
	case "import":
		return p.importRule(start)
	case "include":
		return p.includeRule(start)
	case "media":
		return p.mediaRule(start)
	case "mixin":
		return p.mixinRule(start)
	case "-moz-document":
		return p.mozDocumentRule(start, name)
	case "return":
		return p.disallowedAtRule(start)
	case "supports":
		return p.supportsRule(start)
	case "use":
		p.isUseAllowed = wasUseAllowed
		if !root {
			return p.disallowedAtRule(start)
		}
		return p.useRule(start)
	case "warn":
		return p.warnRule(start)
	case "while":
		return p.whileRule(start, child)
	default:
		return p.unknownAtRule(start, name)
	}
}

// declarationAtRule consumes an at-rule nested inside a property
// declaration. Only the control-flow and meta rules valid there are listed;
// the name is read without interpolation and anything else is rejected.
func (p *StylesheetParser) declarationAtRule() (Statement, error) {
	start := p.scanner.State()
	name, err := p.plainAtRuleName()
	if err != nil {
		return nil, err
	}
	switch name {
	case "content":
		return p.contentRule(start)
	case "debug":
		return p.debugRule(start)
	case "each":
		return p.eachRule(start, p.declarationChild)
	case "else":
		return p.disallowedAtRule(start)
	case "error":
		return p.errorRule(start)
	case "for":
		return p.forRule(start, p.declarationChild)
	case "if":
		return p.ifRule(start, p.declarationChild)
	case "include":
		return p.includeRule(start)
	case "warn":
		return p.warnRule(start)
	case "while":
		return p.whileRule(start, p.declarationChild)
	default:
		return p.disallowedAtRule(start)
	}
}

// functionChild consumes one statement allowed inside a @function body.
// Away from `@`, a failed namespaced variable declaration rewinds and is
// retried as a declaration or style rule, so that a misplaced style rule or
// property reports "@function rules may not contain ..." instead of the raw
// declaration error.
func (p *StylesheetParser) functionChild() (Statement, error) {
	if p.scanner.PeekChar(0) != '@' {
		state := p.scanner.State()
		stmt, varDeclErr := p.variableDeclarationWithNamespace()
		if varDeclErr == nil {
			return stmt, nil
		}

		p.scanner.SetState(state)

		statement, err := p.declarationOrStyleRule()
		if err != nil {
			return nil, varDeclErr
		}
		if statement == nil {
			return nil, varDeclErr
		}

		if _, ok := statement.(*StyleRule); ok {
			span, err := statement.Span()
			if err != nil {
				return nil, err
			}
			if err := p.error("@function rules may not contain style rules.", span, nil); err != nil {
				return nil, err
			}
		} else {
			span, err := statement.Span()
			if err != nil {
				return nil, err
			}
			if err := p.error("@function rules may not contain declarations.", span, nil); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	start := p.scanner.State()
	name, err := p.plainAtRuleName()
	if err != nil {
		return nil, err
	}
	switch name {
	case "debug":
		return p.debugRule(start)
	case "each":
		return p.eachRule(start, func() (Statement, error) {
			return p.functionChild()
		})
	case "else":
		return p.disallowedAtRule(start)
	case "error":
		return p.errorRule(start)
	case "for":
		return p.forRule(start, func() (Statement, error) {
			return p.functionChild()
		})
	case "if":
		return p.ifRule(start, func() (Statement, error) {
			return p.functionChild()
		})
	case "return":
		return p.returnRule(start)
	case "warn":
		return p.warnRule(start)
	case "while":
		return p.whileRule(start, func() (Statement, error) {
			return p.functionChild()
		})
	default:
		return p.disallowedAtRule(start)
	}
}

// plainAtRuleName consumes an at-rule's name with interpolation disallowed,
// used by the declaration- and function-child dispatches.
func (p *StylesheetParser) plainAtRuleName() (string, error) {
	if err := p.expectChar('@'); err != nil {
		return "", err
	}
	return p.identifier(true, false)
}

// atRootRule consumes an @at-root rule. start points before the `@`. A
// parenthesized query or a braced child block parses generic children;
// otherwise the rule wraps a single style rule.
func (p *StylesheetParser) atRootRule(start sasscommon.LineScannerState) (Statement, error) {
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	if p.scanner.PeekChar(0) == '(' {
		query, err := p.atRootQuery()
		if err != nil {
			return nil, err
		}
		return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (Statement, error) {
			return NewAtRootRule(children, span, query), nil
		})
	}
	hasChildren, err := p.lookingAtChildren()
	if err != nil {
		return nil, err
	}
	if hasChildren || (p.indented && p.atEndOfStatement()) {
		return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (Statement, error) {
			return NewAtRootRule(children, span, nil), nil
		})
	}
	child, err := p.styleRule(nil, nil)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewAtRootRule([]Statement{child}, span, nil), nil
}

// atRootQuery consumes a query of the form `(foo: bar)`, where either side
// may be a Sass expression injected into the interpolation buffer.
func (p *StylesheetParser) atRootQuery() (*Interpolation, error) {
	start := p.scanner.State()
	buffer := &InterpolationBuffer{}
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	buffer.WriteCharCode('(')
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	expr1, err := p._expression(expressionOpts{consumeNewlines: true})
	if err != nil {
		return nil, err
	}
	if err := p.addOrInject(buffer, expr1); err != nil {
		return nil, err
	}
	if p.scanner.ScanChar(':') {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		buffer.WriteCharCode(':')
		buffer.WriteCharCode(' ')
		expr2, err := p._expression(expressionOpts{consumeNewlines: true})
		if err != nil {
			return nil, err
		}
		if err := p.addOrInject(buffer, expr2); err != nil {
			return nil, err
		}
	}

	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	buffer.WriteCharCode(')')

	spanFromStart, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	interp, err := buffer.Interpolation(spanFromStart)
	if err != nil {
		spanFromStart, spanErr := p.spanFrom(start)
		if spanErr != nil {
			return nil, spanErr
		}
		if err := p.error(err.Error(), spanFromStart, nil); err != nil {
			return nil, err
		}
	}
	return interp, nil
}

// contentRule consumes a @content rule. start points before the `@`. It is
// only valid inside a mixin body; the argument list is optional and defaults
// to an empty list positioned where the arguments would have started.
func (p *StylesheetParser) contentRule(start sasscommon.LineScannerState) (*ContentRule, error) {
	if !p.inMixin {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("@content is only allowed within mixin declarations.", span, nil); err != nil {
			return nil, err
		}
	}

	beforeWhitespace := p.scanner.State()
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	var arguments *ArgumentList
	if p.scanner.PeekChar(0) == '(' {
		var err error
		arguments, err = p.argumentInvocation(argumentInvocationOpts{mixin: true})
		if err != nil {
			return nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
	} else {
		arguments = NewArgumentListEmpty(p.scanner.SpanFromTo(beforeWhitespace.Position, beforeWhitespace.Position))
	}

	if err := p.expectStatementSeparator("@content rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewContentRule(arguments, span), nil
}

// debugRule consumes a @debug rule. start points before the `@`. The span
// ends at the end of the value expression rather than at the statement
// separator.
func (p *StylesheetParser) debugRule(start sasscommon.LineScannerState) (*DebugRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	expressionEnd := p.scanner.State()
	if err := p.expectStatementSeparator("@debug rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFromTo(start, &expressionEnd)
	if err != nil {
		return nil, err
	}
	return NewDebugRule(value, span), nil
}

// eachRule consumes an @each rule. start points before the `@`; child
// parses the block in the caller's context. The control-directive flag is
// set while parsing so nested declarations are rejected, and restored once
// the children are built.
func (p *StylesheetParser) eachRule(start sasscommon.LineScannerState, child func() (Statement, error)) (*EachRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	wasInControlDirective := p.inControlDirective
	p.inControlDirective = true

	variable, err := p.variableName()
	if err != nil {
		return nil, err
	}
	variables := []string{variable}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	for p.scanner.ScanChar(',') {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		variable, err = p.variableName()
		if err != nil {
			return nil, err
		}
		variables = append(variables, variable)
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if err := p.expectIdentifier("in", "", true); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	list, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}

	return withChildren(p, child, start, func(children []Statement, span sasscommon.FileSpan) (*EachRule, error) {
		p.inControlDirective = wasInControlDirective
		return NewEachRule(variables, list, children, span), nil
	})
}

// errorRule consumes an @error rule. start points before the `@`. Like
// debugRule, the span ends at the value expression.
func (p *StylesheetParser) errorRule(start sasscommon.LineScannerState) (*ErrorRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	expressionEnd := p.scanner.State()
	if err := p.expectStatementSeparator("@error rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFromTo(start, &expressionEnd)
	if err != nil {
		return nil, err
	}
	return NewErrorRule(value, span), nil
}

// extendRule consumes an @extend rule. start points before the `@`. It may
// only appear inside a style rule, mixin, or content block; the `!optional`
// flag is a separate scan after the almost-any-value selector.
func (p *StylesheetParser) extendRule(start sasscommon.LineScannerState) (*ExtendRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if !p.inStyleRule && !p.inMixin && !p.inContentBlock {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("@extend may only be used within style rules.", span, nil); err != nil {
			return nil, err
		}
	}

	value, err := p.almostAnyValue(false)
	if err != nil {
		return nil, err
	}
	optional := p.scanner.ScanChar('!')
	if optional {
		if err := p.expectIdentifier("optional", "", true); err != nil {
			return nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
	}
	if err := p.expectStatementSeparator("@extend rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewExtendRule(value, span, optional), nil
}

// functionRule consumes a @function declaration. start points before the
// `@`; atRuleName is the already-parsed name, so a `--`-prefixed custom
// property can be routed to the unknown-at-rule path without reparsing.
// The names `type`, `expression`, `url`, `and`/`or`/`not`, and unvendored
// `element` are hard errors (the middle group also warns as deprecated
// aliases), and declarations are rejected inside mixins, content blocks, and
// control directives. Children parse via functionChild and the rule keeps
// the preceding silent comment.
func (p *StylesheetParser) functionRule(start sasscommon.LineScannerState, atRuleName *Interpolation) (Statement, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	precedingComment := p.lastSilentComment
	p.lastSilentComment = nil
	beforeName := p.scanner.State()

	if p.scanner.PeekChar(0) == '-' && p.scanner.PeekChar(1) == '-' {
		return p.unknownAtRule(start, atRuleName)
	}

	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(name, "type") {
		span, err := p.spanFrom(beforeName)
		if err != nil {
			return nil, err
		}
		if err := p.error("This name is reserved for the plain-CSS function.", span, nil); err != nil {
			return nil, err
		}
	}

	switch name {
	case "expression", "url", "and", "or", "not":
		span, err := p.spanFrom(beforeName)
		if err != nil {
			return nil, err
		}
		if err := p.error("Invalid function name.", span, nil); err != nil {
			return nil, err
		}
	default:
		if unvendor.Unvendor(name) == "element" {
			span, err := p.spanFrom(beforeName)
			if err != nil {
				return nil, err
			}
			if err := p.error("Invalid function name.", span, nil); err != nil {
				return nil, err
			}
		}
	}

	lower := strings.ToLower(name)
	switch lower {
	case "expression", "url":
		span, err := p.spanFrom(beforeName)
		if err != nil {
			return nil, err
		}
		p.warnings = append(p.warnings, ParseTimeWarning{
			Deprecation: deprecation.FunctionName,
			Message:     "Custom functions with this name are deprecated and will be removed in a future\nrelease. Please choose a different name.\nMore info: https://sass-lang.com/d/function-name",
			Span:        span,
		})
	default:
		if unvendor.Unvendor(lower) == "element" {
			span, err := p.spanFrom(beforeName)
			if err != nil {
				return nil, err
			}
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: deprecation.FunctionName,
				Message:     "Custom functions with this name are deprecated and will be removed in a future\nrelease. Please choose a different name.\nMore info: https://sass-lang.com/d/function-name",
				Span:        span,
			})
		}
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	parameters, err := p.parameterList()
	if err != nil {
		return nil, err
	}

	if p.inMixin || p.inContentBlock {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Mixins may not contain function declarations.", span, nil); err != nil {
			return nil, err
		}
	} else if p.inControlDirective {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Functions may not be declared in control directives.", span, nil); err != nil {
			return nil, err
		}
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	return withChildren(p, p.functionChild, start, func(children []Statement, span sasscommon.FileSpan) (Statement, error) {
		return NewFunctionRule(name, parameters, children, span, precedingComment), nil
	})
}

// forRule consumes an @for rule. start points before the `@`; child parses
// the block in the caller's context. The lower bound is scanned with an
// until-callback that stops at `to` (exclusive) or `through` (inclusive),
// and a missing keyword is a hard error. The control-directive flag is
// restored once the children are built.
func (p *StylesheetParser) forRule(start sasscommon.LineScannerState, child func() (Statement, error)) (*ForRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	wasInControlDirective := p.inControlDirective
	p.inControlDirective = true
	variable, err := p.variableName()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	if err := p.expectIdentifier("from", "", true); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var exclusive *bool
	var scanErr error
	from, err := p._expression(
		expressionOpts{consumeNewlines: true,
			until: func() bool {
				if !p.lookingAtIdentifier(nil) {
					return false
				}
				ok, err := p.scanIdentifier("to", false)
				if err != nil {
					scanErr = err
					return true
				}
				if ok {
					exclusive = new(true)
					return true
				}
				ok, err = p.scanIdentifier("through", false)
				if err != nil {
					scanErr = err
					return true
				}
				if ok {
					exclusive = new(false)
					return true
				}
				return false
			}},
	)
	if err != nil {
		return nil, err
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if exclusive == nil {
		return nil, p.scanner.Error(`Expected "to" or "through".`, -1, 0)
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	to, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}

	return withChildren(p, child, start, func(children []Statement, span sasscommon.FileSpan) (*ForRule, error) {
		p.inControlDirective = wasInControlDirective
		return NewForRule(variable, from, to, children, span, *exclusive), nil
	})
}

// forwardRule consumes a @forward rule: a URL, an optional `as <prefix>*`
// clause, an optional `show`/`hide` member list (never both), and an
// optional `with` configuration that allows guarded variables. The rule must
// precede any other rule, checked against isUseAllowed after the span is
// taken. start points before the `@`.
func (p *StylesheetParser) forwardRule(start sasscommon.LineScannerState) (*ForwardRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	url, err := p.urlString()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	var prefix *string
	ok, err := p.scanIdentifier("as", false)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		s, err := p.identifier(true, false)
		if err != nil {
			return nil, err
		}
		prefix = &s
		if err := p.expectChar('*'); err != nil {
			return nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
	}

	var shownMixinsAndFunctions, shownVariables *orderedset.LinkedSet[string]
	var hiddenMixinsAndFunctions, hiddenVariables *orderedset.LinkedSet[string]
	ok, err = p.scanIdentifier("show", false)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		var memberErr error
		shownMixinsAndFunctions, shownVariables, memberErr = p.memberList()
		if memberErr != nil {
			return nil, memberErr
		}
	} else {
		ok, err = p.scanIdentifier("hide", false)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			var memberErr error
			hiddenMixinsAndFunctions, hiddenVariables, memberErr = p.memberList()
			if memberErr != nil {
				return nil, memberErr
			}
		}
	}

	configuration, err := p.configuration(true)
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	if err := p.expectStatementSeparator("@forward rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	if !p.isUseAllowed {
		if err := p.error("@forward rules must be written before any other rules.", span, nil); err != nil {
			return nil, err
		}
	}

	if shownMixinsAndFunctions != nil {
		return NewForwardRuleShow(url, shownMixinsAndFunctions, shownVariables, span, prefix, configuration), nil
	} else if hiddenMixinsAndFunctions != nil {
		return NewForwardRuleHide(url, hiddenMixinsAndFunctions, hiddenVariables, span, prefix, configuration), nil
	}
	return NewForwardRule(url, span, prefix, configuration), nil
}

// memberList consumes a `show`/`hide` member list of plain identifiers or
// `$variable` names, returning the plain names first and the variable names
// second. Duplicates collapse because the result is an ordered set.
func (p *StylesheetParser) memberList() (*orderedset.LinkedSet[string], *orderedset.LinkedSet[string], error) {
	var identOrder []string
	var varOrder []string
	identSet := map[string]struct{}{}
	varSet := map[string]struct{}{}
	for {
		if err := p.whitespace(true); err != nil {
			return nil, nil, err
		}
		if err := p.withErrorMessage("Expected variable, mixin, or function name", func() error {
			if p.scanner.PeekChar(0) == '$' {
				name, err := p.variableName()
				if err != nil {
					return err
				}
				if _, ok := varSet[name]; !ok {
					varOrder = append(varOrder, name)
					varSet[name] = struct{}{}
				}
			} else {
				name, err := p.identifier(true, false)
				if err != nil {
					return err
				}
				if _, ok := identSet[name]; !ok {
					identOrder = append(identOrder, name)
					identSet[name] = struct{}{}
				}
			}
			return nil
		}); err != nil {
			return nil, nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, nil, err
		}
		if !p.scanner.ScanChar(',') {
			break
		}
	}
	return orderedset.NewFromSlice(identOrder), orderedset.NewFromSlice(varOrder), nil
}

// ifRule consumes an @if rule with its `@else if`/`@else` chain. start
// points before the `@`; child parses each branch in the caller's context.
// Continuations are scanned at the original indentation so a dedented @else
// ends the rule. The control-directive flag is restored after the whole
// chain is built.
func (p *StylesheetParser) ifRule(start sasscommon.LineScannerState, child func() (Statement, error)) (*IfRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	ifIndentation := p.currentIndentation()
	wasInControlDirective := p.inControlDirective
	p.inControlDirective = true
	condition, exprErr := p._expression(expressionOpts{})
	if exprErr != nil {
		return nil, exprErr
	}
	children, err := p.children(child)
	if err != nil {
		return nil, err
	}
	if err := p.whitespaceWithoutComments(false); err != nil {
		return nil, err
	}

	var clauses []*IfClause
	clauses = append(clauses, NewIfClause(condition, children))
	var lastClause *ElseClause

	for {
		hasElse, err := p.scanElse(ifIndentation)
		if err != nil {
			return nil, err
		}
		if !hasElse {
			break
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		ok, err := p.scanIdentifier("if", false)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			elseIfExpr, err := p._expression(expressionOpts{})
			if err != nil {
				return nil, err
			}
			elseIfChildren, err := p.children(child)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, NewIfClause(elseIfExpr, elseIfChildren))
		} else {
			elseChildren, err := p.children(child)
			if err != nil {
				return nil, err
			}
			lastClause = NewElseClause(elseChildren)
			break
		}
	}
	p.inControlDirective = wasInControlDirective

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	if err := p.whitespaceWithoutComments(false); err != nil {
		return nil, err
	}
	return NewIfRule(clauses, span, lastClause), nil
}

// importRule consumes an @import rule: a comma-separated list of static and
// dynamic imports. Every dynamic import carries the legacy-@import
// deprecation warning, and dynamic imports are rejected inside control
// directives and mixins. start points before the `@`.
func (p *StylesheetParser) importRule(start sasscommon.LineScannerState) (*ImportRule, error) {
	var imports []Import
	for {
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		var argument Import
		var err error
		if p.importArgumentFn != nil {
			argument, err = p.importArgumentFn()
			if err != nil {
				return nil, err
			}
		} else {
			argument, err = p.importArgument()
			if err != nil {
				return nil, err
			}
		}
		if _, ok := argument.(*DynamicImport); ok {
			dynamicSpan, err := argument.Span()
			if err != nil {
				return nil, err
			}
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: deprecation.Import,
				Message:     "Sass @import rules are deprecated and will be removed in Dart Sass 3.0.0.\n\nMore info and automated migrator: https://sass-lang.com/d/import",
				Span:        dynamicSpan,
			})
			if p.inControlDirective || p.inMixin {
				_, err := p.disallowedAtRule(start)
				return nil, err
			}
		}
		if argument != nil {
			imports = append(imports, argument)
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		if !p.scanner.ScanChar(',') {
			break
		}
	}
	if err := p.expectStatementSeparator("@import rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewImportRule(imports, span), nil
}

// importArgument consumes one @import argument. A leading `u`/`U` parses a
// dynamic `url(...)` as a static import; otherwise a quoted string is read
// and stays static when it is a plain-CSS URL or has modifiers, else it
// becomes a dynamic import whose URL is validated by parseImportUrl.
func (p *StylesheetParser) importArgument() (Import, error) {
	start := p.scanner.State()
	ch := p.scanner.PeekChar(0)
	if ch == 'u' || ch == 'U' {
		urlExpr, dynErr := p.dynamicUrl()
		if dynErr != nil {
			return nil, dynErr
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		modifiers, modErr := p.tryImportModifiers()
		if modErr != nil {
			return nil, modErr
		}
		var interp *Interpolation
		if se, ok := urlExpr.(*StringExpression); ok {
			interp = se.Text
		} else {
			exprSpan, err := urlExpr.Span()
			if err != nil {
				return nil, err
			}
			interp, _ = NewInterpolation([]any{urlExpr}, []*sasscommon.FileSpan{&exprSpan}, exprSpan)
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewStaticImport(interp, span, modifiers), nil
	}

	urlStr, err := p.string()
	if err != nil {
		return nil, err
	}
	urlSpan, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	modifiers, err := p.tryImportModifiers()
	if err != nil {
		return nil, err
	}
	if p.isPlainImportUrl(urlStr) || modifiers != nil {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		urlSpanText, err := urlSpan.SpanText()
		if err != nil {
			return nil, err
		}
		return NewStaticImport(NewInterpolationPlain(urlSpanText, urlSpan), span, modifiers), nil
	}

	url := p.parseImportUrl(urlStr)
	// Matches Dart: DynamicImport(parseImportUrl(url), urlSpan) — the URL is
	// passed through unchanged; Uri.parse(url) only validates (throwing a
	// FormatException caught as "Invalid URL").
	if _, err := goUrl.Parse(url); err != nil {
		if pErr := p.error("Invalid URL: "+err.Error(), urlSpan, nil); pErr != nil {
			return nil, pErr
		}
	}
	return NewDynamicImport(url, urlSpan), nil
}

// parseImportUrl validates url as an import URL and returns it unchanged.
// Absolute Windows paths convert to file URIs for backwards compatibility;
// anything else must URL-parse, with failures reported as "Invalid URL" by
// the caller.
func (p *StylesheetParser) parseImportUrl(rawURL string) string {
	return ParseImportUrl(rawURL)
}

// isPlainImportUrl reports whether url forces an @import onto the plain-CSS
// path: a `.css` suffix, a protocol-relative `//` prefix, or an http(s) URL.
// Short strings can never match, so they are rejected up front.
func (p *StylesheetParser) isPlainImportUrl(url string) bool {
	if len(url) < 5 {
		return false
	}
	if strings.HasSuffix(url, ".css") {
		return true
	}
	switch url[0] {
	case '/':
		return len(url) > 1 && url[1] == '/'
	case 'h':
		return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
	}
	return false
}

// tryImportModifiers consumes the media/supports modifiers after an @import
// argument, returning nil when no modifier follows. It exits before
// allocating in the common modifier-less case. A `supports(...)` identifier
// parses its query specially (declarations stay unbracketed); other
// `name(...)` calls slurp a declaration value; a comma continues into a
// media query list, and a bare `(` starts one directly.
func (p *StylesheetParser) tryImportModifiers() (*Interpolation, error) {
	if !p.lookingAtInterpolatedIdentifier() && p.scanner.PeekChar(0) != '(' {
		return nil, nil
	}

	start := p.scanner.State()
	buffer := &InterpolationBuffer{}
	for {
		if p.lookingAtInterpolatedIdentifier() {
			if !buffer.IsEmpty() {
				buffer.WriteCharCode(' ')
			}

			identifier, err := p.interpolatedIdentifier()
			if err != nil {
				return nil, err
			}
			buffer.AddInterpolation(identifier)

			name := identifier.AsPlain()
			var lower string
			if name != nil {
				lower = strings.ToLower(*name)
			}
			if lower != "and" && p.scanner.ScanChar('(') {
				if lower == "supports" {
					query, err := p.importSupportsQuery()
					if err != nil {
						return nil, err
					}
					if _, ok := query.(*SupportsDeclaration); !ok {
						buffer.WriteCharCode('(')
					}
					querySpan, err := query.Span()
					if err != nil {
						return nil, err
					}
					buffer.Add(NewSupportsExpression(query), querySpan)
					if _, ok := query.(*SupportsDeclaration); !ok {
						buffer.WriteCharCode(')')
					}
				} else {
					buffer.WriteCharCode('(')
					idecl, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: true, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: true})
					if err != nil {
						return nil, err
					}
					buffer.AddInterpolation(idecl)
					buffer.WriteCharCode(')')
				}
				if err := p.expectChar(')'); err != nil {
					return nil, err
				}
				if err := p.whitespace(false); err != nil {
					return nil, err
				}
			} else {
				if err := p.whitespace(false); err != nil {
					return nil, err
				}
				if p.scanner.ScanChar(',') {
					buffer.Write(", ")
					mql, mqlErr := p.mediaQueryList()
					if mqlErr != nil {
						return nil, mqlErr
					}
					buffer.AddInterpolation(mql)
					spanFromStart, err := p.spanFrom(start)
					if err != nil {
						return nil, err
					}
					interp, err := buffer.Interpolation(spanFromStart)
					if err != nil {
						spanFromStart, spanErr := p.spanFrom(start)
						if spanErr != nil {
							return nil, spanErr
						}
						if err := p.error(err.Error(), spanFromStart, nil); err != nil {
							return nil, err
						}
					}
					return interp, nil
				}
			}
		} else if p.scanner.PeekChar(0) == '(' {
			if !buffer.IsEmpty() {
				buffer.WriteCharCode(' ')
			}
			mql, mqlErr := p.mediaQueryList()
			if mqlErr != nil {
				return nil, mqlErr
			}
			buffer.AddInterpolation(mql)
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			interp, err := buffer.Interpolation(span)
			if err != nil {
				if err := p.error(err.Error(), span, nil); err != nil {
					return nil, err
				}
			}
			return interp, nil
		} else {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			interp, err := buffer.Interpolation(span)
			if err != nil {
				if err := p.error(err.Error(), span, nil); err != nil {
					return nil, err
				}
			}
			return interp, nil
		}
	}
}

// includeRule consumes an @include rule: an optionally namespaced mixin
// name, an optional argument list, an optional `using` content-parameter
// list, and either a content block (parsed with inContentBlock set) or a
// bare statement separator. start points before the `@`; the span stretches
// from there over the content block when present, else over the arguments.
func (p *StylesheetParser) includeRule(start sasscommon.LineScannerState) (*IncludeRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	var namespace string
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	if p.scanner.ScanChar('.') {
		namespace = name
		name, err = p.publicIdentifier()
		if err != nil {
			return nil, err
		}
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	var arguments *ArgumentList
	if p.scanner.PeekChar(0) == '(' {
		var err error
		arguments, err = p.argumentInvocation(argumentInvocationOpts{mixin: true})
		if err != nil {
			return nil, err
		}
	} else {
		arguments = NewArgumentListEmpty(p.scanner.EmptySpan())
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	var contentParameters *ParameterList
	ok, err := p.scanIdentifier("using", false)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		contentParameters, err = p.parameterList()
		if err != nil {
			return nil, err
		}
		if err := p.whitespace(false); err != nil {
			return nil, err
		}
	}

	var content *ContentBlock
	hasContentChildren, err := p.lookingAtChildren()
	if err != nil {
		return nil, err
	}
	if contentParameters != nil || hasContentChildren {
		if contentParameters == nil {
			contentParameters = NewParameterListEmpty(p.scanner.EmptySpan())
		}
		wasInContentBlock := p.inContentBlock
		p.inContentBlock = true
		var contentErr error
		content, contentErr = withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*ContentBlock, error) {
			return NewContentBlock(contentParameters, children, span), nil
		})
		if contentErr != nil {
			return nil, contentErr
		}
		p.inContentBlock = wasInContentBlock
	} else {
		if err := p.expectStatementSeparator(""); err != nil {
			return nil, err
		}
	}

	var ns *string
	if namespace != "" {
		ns = &namespace
	}
	var span sasscommon.FileSpan
	if content != nil {
		contentSpan, err := content.Span()
		if err != nil {
			return nil, err
		}
		spanFrom, err := p.spanFromTo(start, &start)
		if err != nil {
			return nil, err
		}
		span, err = spanFrom.Expand(contentSpan)
		if err != nil {
			return nil, err
		}
	} else {
		argSpan, err := arguments.Span()
		if err != nil {
			return nil, err
		}
		spanFrom, err := p.spanFromTo(start, &start)
		if err != nil {
			return nil, err
		}
		span, err = spanFrom.Expand(argSpan)
		if err != nil {
			return nil, err
		}
	}
	return NewIncludeRule(name, arguments, span, ns, content), nil
}

// mediaRule consumes a @media rule: a media query list followed by generic
// statement children. start points before the `@`.
func (p *StylesheetParser) mediaRule(start sasscommon.LineScannerState) (*MediaRule, error) {
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	query, err := p.mediaQueryList()
	if err != nil {
		return nil, err
	}
	return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*MediaRule, error) {
		return NewMediaRule(query, children, span), nil
	})
}

// mixinRule consumes a @mixin declaration. start points before the `@`.
// `--`-prefixed names are rejected for forward compatibility with plain-CSS
// mixins, and nested mixin declarations (inside mixins, content blocks, or
// control directives) are rejected. The inMixin flag guards the body parse
// and is cleared once the children are built; the rule keeps the preceding
// silent comment.
func (p *StylesheetParser) mixinRule(start sasscommon.LineScannerState) (*MixinRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	precedingComment := p.lastSilentComment
	p.lastSilentComment = nil
	beforeName := p.scanner.State()
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(name, "--") {
		span, err := p.spanFrom(beforeName)
		if err != nil {
			return nil, err
		}
		if err := p.error(
			"Sass @mixin names beginning with -- are forbidden for forward-compatibility with plain CSS mixins.\n"+
				"\n"+
				"For details, see https://sass-lang.com/d/css-function-mixin",
			span,
			nil,
		); err != nil {
			return nil, err
		}
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	var parameters *ParameterList
	if p.scanner.PeekChar(0) == '(' {
		var err error
		parameters, err = p.parameterList()
		if err != nil {
			return nil, err
		}
	} else {
		parameters = NewParameterListEmpty(p.scanner.EmptySpan())
	}

	if p.inMixin || p.inContentBlock {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Mixins may not contain mixin declarations.", span, nil); err != nil {
			return nil, err
		}
	} else if p.inControlDirective {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error("Mixins may not be declared in control directives.", span, nil); err != nil {
			return nil, err
		}
	}

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	p.inMixin = true

	return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*MixinRule, error) {
		p.inMixin = false
		return NewMixinRule(name, parameters, children, span, precedingComment), nil
	})
}

// mozDocumentRule consumes a `@-moz-document` rule. Gecko diverges from the
// spec by letting `url-prefix` and `domain` omit quotation marks, so each
// comma-separated item parses either an interpolation, a known function
// (`url`, `url-prefix`, `domain` with unquoted contents when possible,
// `regexp`), or a hard error. start points before the `@`. The rule warns
// as deprecated unless every item was a bare or empty-string `url-prefix()`.
func (p *StylesheetParser) mozDocumentRule(start sasscommon.LineScannerState, name *Interpolation) (*AtRule, error) {
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	valueStart := p.scanner.State()
	buffer := &InterpolationBuffer{}
	needsDeprecationWarning := false

	for {
		if p.scanner.PeekChar(0) == '#' {
			expr, span, err := p.singleInterpolation()
			if err != nil {
				return nil, err
			}
			buffer.Add(expr, span)
			needsDeprecationWarning = true
		} else {
			identifierStart := p.scanner.State()
			identifier, err := p.identifier(true, false)
			if err != nil {
				return nil, err
			}
			switch identifier {
			case "url", "url-prefix", "domain":
				contents, err := p.tryUrlContents(identifierStart, identifier, false)
				if err != nil {
					return nil, err
				}
				if contents != nil {
					buffer.AddInterpolation(contents)
				} else {
					if err := p.expectChar('('); err != nil {
						return nil, err
					}
					if err := p.whitespace(false); err != nil {
						return nil, err
					}
					argument, err := p.interpolatedStringToken()
					if err != nil {
						return nil, err
					}
					if err := p.expectChar(')'); err != nil {
						return nil, err
					}
					buffer.Write(identifier)
					buffer.WriteCharCode('(')
					buffer.AddInterpolation(argument)
					buffer.WriteCharCode(')')
				}

				trailing := buffer.TrailingString()
				if !strings.HasSuffix(trailing, "url-prefix()") &&
					!strings.HasSuffix(trailing, "url-prefix('')") &&
					!strings.HasSuffix(trailing, `url-prefix("")`) {
					needsDeprecationWarning = true
				}

			case "regexp":
				buffer.Write("regexp(")
				if err := p.expectChar('('); err != nil {
					return nil, err
				}
				ist, istErr := p.interpolatedStringToken()
				if istErr != nil {
					return nil, istErr
				}
				buffer.AddInterpolation(ist)
				if err := p.expectChar(')'); err != nil {
					return nil, err
				}
				buffer.WriteCharCode(')')
				needsDeprecationWarning = true

			default:
				span, err := p.spanFrom(identifierStart)
				if err != nil {
					return nil, err
				}
				if err := p.error("Invalid function name.", span, nil); err != nil {
					return nil, err
				}
			}
		}

		if err := p.whitespace(false); err != nil {
			return nil, err
		}
		if !p.scanner.ScanChar(',') {
			break
		}
		buffer.WriteCharCode(',')
		text, err := p.rawText(func() error { return p.whitespace(false) })
		if err != nil {
			return nil, err
		}
		buffer.Write(text)
	}

	span, err := p.spanFrom(valueStart)
	if err != nil {
		return nil, err
	}
	value, valueErr := buffer.Interpolation(span)
	if valueErr != nil {
		span2, err2 := p.spanFrom(valueStart)
		if err2 != nil {
			return nil, err2
		}
		if err := p.error(valueErr.Error(), span2, nil); err != nil {
			return nil, err
		}
	}
	return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*AtRule, error) {
		if needsDeprecationWarning {
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: deprecation.MozDocument,
				Message:     "@-moz-document is deprecated and support will be removed in Dart Sass 2.0.0.\n\nFor details, see https://sass-lang.com/d/moz-document.",
				Span:        span,
			})
		}
		return NewAtRule(name, span, value, children), nil
	})
}

// returnRule consumes a @return rule: a value expression up to the
// statement separator. start points before the `@`.
func (p *StylesheetParser) returnRule(start sasscommon.LineScannerState) (*ReturnRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	if err := p.expectStatementSeparator("@return rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewReturnRule(value, span), nil
}

// supportsRule consumes a @supports rule: a supports condition followed by
// generic statement children. start points before the `@`.
func (p *StylesheetParser) supportsRule(start sasscommon.LineScannerState) (*SupportsRule, error) {
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	condition, condErr := p.supportsCondition(false)
	if condErr != nil {
		return nil, condErr
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	return withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*SupportsRule, error) {
		return NewSupportsRule(condition, children, span), nil
	})
}

// useRule consumes a @use rule: a URL, its namespace, and an optional `with`
// configuration. The rule must precede any other rule, checked against
// isUseAllowed after the span is taken and before the separator. start
// points before the `@`.
func (p *StylesheetParser) useRule(start sasscommon.LineScannerState) (*UseRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	url, err := p.urlString()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	namespace, err := p.useNamespace(url, start)
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	configuration, err := p.configuration(false)
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	if !p.isUseAllowed {
		if err := p.error("@use rules must be written before any other rules.", span, nil); err != nil {
			return nil, err
		}
	}
	if err := p.expectStatementSeparator("@use rule"); err != nil {
		return nil, err
	}

	result, err := NewUseRule(url, namespace, span, configuration)
	if err != nil {
		if pErr := p.error(err.Error(), span, nil); pErr != nil {
			return nil, pErr
		}
	}
	return result, nil
}

// useNamespace resolves the namespace of a @use rule: an explicit `as`
// clause (`as *` yields nil for no namespace), else the default derived
// from the URL's last path segment with a leading `_` stripped and the file
// extension dropped. start points before the `@` for the error span when the
// default is not a valid identifier. A nil return means `@use ... as *`.
func (p *StylesheetParser) useNamespace(u *goUrl.URL, start sasscommon.LineScannerState) (*string, error) {
	ok, err := p.scanIdentifier("as", false)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if p.scanner.ScanChar('*') {
			return nil, nil
		}
		s, err := p.identifier(true, false)
		if err != nil {
			return nil, err
		}
		return &s, nil
	}

	basename := u.Path
	if idx := strings.LastIndex(u.Path, "/"); idx >= 0 {
		basename = u.Path[idx+1:]
	}
	basename = strings.TrimPrefix(basename, "_")
	if dot := strings.Index(basename, "."); dot >= 0 {
		basename = basename[:dot]
	}

	result := basename
	if _, err := ParseIdentifier(result); err != nil {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		if err := p.error(
			fmt.Sprintf(
				"The default namespace %q is not a valid Sass identifier.\n\n"+
					"Recommendation: add an \"as\" clause to define an explicit namespace.",
				result,
			),
			span,
			nil,
		); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

// configuration consumes the `with` clause of a @use or @forward rule,
// returning nil when there is none. When allowGuarded is true (only
// @forward), entries accept a `!default` flag. Configuring `-private`
// variables warns as deprecated, and configuring the same variable twice is
// a hard error. A trailing comma ends the list when no expression follows.
func (p *StylesheetParser) configuration(allowGuarded bool) ([]*ConfiguredVariable, error) {
	ok, err := p.scanIdentifier("with", false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	variableNames := map[string]struct{}{}
	var configuration []*ConfiguredVariable
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if err := p.expectChar('('); err != nil {
		return nil, err
	}

	for {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}

		variableStart := p.scanner.State()
		name, err := p.variableName()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(name, "-") {
			span, err := p.spanFrom(variableStart)
			if err != nil {
				return nil, err
			}
			p.warnings = append(p.warnings, ParseTimeWarning{
				Deprecation: deprecation.WithPrivate,
				Message:     "Configuring private variables is deprecated.\nThis will be an error in Dart Sass 2.0.0.",
				Span:        span,
			})
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

		expression, exprErr := p.expressionUntilComma(false)
		if exprErr != nil {
			return nil, exprErr
		}

		var guarded bool
		flagStart := p.scanner.State()
		if allowGuarded && p.scanner.ScanChar('!') {
			if name, err := p.identifier(true, false); err != nil {
				return nil, err
			} else if name == "default" {
				guarded = true
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
			} else {
				flagSpan, err := p.spanFrom(flagStart)
				if err != nil {
					return nil, err
				}
				if err := p.error("Invalid flag name.", flagSpan, nil); err != nil {
					return nil, err
				}
			}
		}

		span, err := p.spanFrom(variableStart)
		if err != nil {
			return nil, err
		}
		if _, exists := variableNames[name]; exists {
			if err := p.error("The same variable may only be configured once.", span, nil); err != nil {
				return nil, err
			}
		}
		variableNames[name] = struct{}{}
		configuration = append(configuration, NewConfiguredVariable(name, expression, span, guarded))

		if !p.scanner.ScanChar(',') {
			break
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if !p.lookingAtExpression() {
			break
		}
	}

	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	return configuration, nil
}

// warnRule consumes a @warn rule. start points before the `@`. Like
// debugRule, the span ends at the value expression.
func (p *StylesheetParser) warnRule(start sasscommon.LineScannerState) (*WarnRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	value, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	expressionEnd := p.scanner.State()
	if err := p.expectStatementSeparator("@warn rule"); err != nil {
		return nil, err
	}
	span, err := p.spanFromTo(start, &expressionEnd)
	if err != nil {
		return nil, err
	}
	return NewWarnRule(value, span), nil
}

// whileRule consumes a @while rule. start points before the `@`; child
// parses the block in the caller's context. The control-directive flag is
// set for the condition and body, and restored once the children are built.
func (p *StylesheetParser) whileRule(start sasscommon.LineScannerState, child func() (Statement, error)) (*WhileRule, error) {
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	wasInControlDirective := p.inControlDirective
	p.inControlDirective = true
	condition, err := p._expression(expressionOpts{})
	if err != nil {
		return nil, err
	}
	return withChildren(p, child, start, func(children []Statement, span sasscommon.FileSpan) (*WhileRule, error) {
		p.inControlDirective = wasInControlDirective
		return NewWhileRule(condition, children, span), nil
	})
}

// unknownAtRule consumes an at-rule Sass does not know: start points before
// the `@` and name is the already-parsed (possibly interpolated) name. The
// inUnknownAtRule flag is set so nested statements parse as declarations or
// style rules, and a rule literally named `function` additionally sets
// inPlainCssFunction so its body parses `result` as a plain custom
// property. The value is skipped when the rule ends immediately or starts
// with `!`; a braced block parses generic children, otherwise the rule ends
// at the statement separator. Both flags are restored on either path.
func (p *StylesheetParser) unknownAtRule(start sasscommon.LineScannerState, name *Interpolation) (*AtRule, error) {
	wasInUnknownAtRule := p.inUnknownAtRule
	p.inUnknownAtRule = true

	if err := p.whitespace(false); err != nil {
		return nil, err
	}

	var value *Interpolation
	if p.scanner.PeekChar(0) != '!' && !p.atEndOfStatement() {
		var err error
		value, err = p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: false, allowSemicolon: false, allowColon: true, allowOpenBrace: false, endAfterOf: false, silentComments: true, consumeNewlines: false})
		if err != nil {
			return nil, err
		}
	}

	wasInPlainCssFunction := p.inPlainCssFunction
	if name.AsPlain() != nil && strings.EqualFold(*name.AsPlain(), "function") {
		p.inPlainCssFunction = true
	}

	hasChildren, err := p.lookingAtChildren()
	if err != nil {
		return nil, err
	}
	if hasChildren {
		result, err := withChildren(p, func() (Statement, error) { return p.statement(false) }, start, func(children []Statement, span sasscommon.FileSpan) (*AtRule, error) {
			return NewAtRule(name, span, value, children), nil
		})
		if err != nil {
			return nil, err
		}
		p.inUnknownAtRule = wasInUnknownAtRule
		p.inPlainCssFunction = wasInPlainCssFunction
		return result, nil
	}
	if err := p.expectStatementSeparator(""); err != nil {
		return nil, err
	}
	p.inUnknownAtRule = wasInUnknownAtRule
	p.inPlainCssFunction = wasInPlainCssFunction
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewAtRule(name, span, value, nil), nil
}

// disallowedAtRule reports an at-rule that is not permitted in the current
// context. start points before the `@`. The rule's value is swallowed first
// so the error span covers the whole rule. It declares a Statement return
// type purely so dispatch arms can return its result directly; it always
// returns a nil statement with an error.
func (p *StylesheetParser) disallowedAtRule(start sasscommon.LineScannerState) (Statement, error) {
	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	if _, err := p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: false, endAfterOf: false, silentComments: true, consumeNewlines: false}); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return nil, p.error("This at-rule is not allowed here.", span, nil)
}

// parameterList consumes a parenthesized parameter list: `$name` entries
// with optional `: default` values, plus an optional trailing `$rest...`.
// A repeated name is a hard error, and a comma may follow the rest
// parameter before the closing paren.
func (p *StylesheetParser) parameterList() (*ParameterList, error) {
	start := p.scanner.State()
	if err := p.expectChar('('); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	var parameters []*Parameter
	named := map[string]struct{}{}
	var restParameter string
	for p.scanner.PeekChar(0) == '$' {
		variableStart := p.scanner.State()
		name, err := p.variableName()
		if err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}

		var defaultValue Expression
		if p.scanner.ScanChar(':') {
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			var exprErr error
			defaultValue, exprErr = p.expressionUntilComma(false)
			if exprErr != nil {
				return nil, exprErr
			}
		} else if p.scanner.ScanChar('.') {
			if err := p.expectChar('.'); err != nil {
				return nil, err
			}
			if err := p.expectChar('.'); err != nil {
				return nil, err
			}
			if err := p.whitespace(true); err != nil {
				return nil, err
			}
			if p.scanner.ScanChar(',') {
				if err := p.whitespace(true); err != nil {
					return nil, err
				}
			}
			restParameter = name
			break
		}

		paramSpan, err := p.spanFrom(variableStart)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, NewParameter(name, paramSpan, defaultValue))
		if _, exists := named[name]; exists {
			dupSpan, err := parameters[len(parameters)-1].Span()
			if err != nil {
				return nil, err
			}
			if err := p.error("Duplicate parameter.", dupSpan, nil); err != nil {
				return nil, err
			}
		}
		named[name] = struct{}{}

		if !p.scanner.ScanChar(',') {
			break
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}
	var rest *string
	if restParameter != "" {
		rest = &restParameter
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewParameterList(parameters, span, rest), nil
}
