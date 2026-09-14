// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (selector parsing: selector
// lists, complex/compound/simple selectors, attribute operators, pseudo and
// type/universal selectors)
//
// What follows is largely duplicated in the standalone selector parser; most
// changes here should be mirrored there and vice versa.

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/unvendor"
)

// selectorList consumes a comma-separated selector list. Doubled commas are
// skipped, a trailing comma at end of input ends the list, and each
// component after the first records whether a line break preceded it.
func (p *StylesheetParser) selectorList() (*InterpolatedSelectorList, error) {
	start := p.scanner.State()
	var previousLine int
	first, err := p.complexSelector(false, true, true)
	if err != nil {
		return nil, err
	}
	components := []*InterpolatedComplexSelector{first}
	previousLine = p.scanner.Line()

	if err := p.whitespace(false); err != nil {
		return nil, err
	}
	for p.scanner.ScanChar(',') {
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		if p.scanner.PeekChar(0) == ',' {
			continue
		}
		if p.scanner.IsDone() {
			break
		}

		lineBreak := p.scanner.Line() != previousLine
		if lineBreak {
			previousLine = p.scanner.Line()
		}
		sel, err := p.complexSelector(lineBreak, true, true)
		if err != nil {
			return nil, err
		}
		components = append(components, sel)
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	result, err := NewInterpolatedSelectorList(components, span)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// complexSelector consumes a complex selector. When lineBreak is true, a line
// break preceded this selector. Combinators are only recognized before any
// compound has set a pending combinator and, for leading position, only when
// allowLeadingCombinator is set. A finished compound is flushed as a
// component carrying the pending combinator; a combinator with no compound
// yet becomes the leading combinator. A `&` after a compound is rejected
// (it may only open one). A trailing combinator is an error in plain CSS or
// when allowTrailingCombinator is false; a lone leading combinator is kept,
// and anything else fails with "expected selector."
func (p *StylesheetParser) complexSelector(lineBreak bool, allowLeadingCombinator, allowTrailingCombinator bool) (*InterpolatedComplexSelector, error) {
	start := p.scanner.State()
	var err error

	componentStart := p.scanner.State()
	var lastCompound *InterpolatedCompoundSelector
	var combinator *sasscommon.CssValue[Combinator]
	var leadingCombinator *sasscommon.CssValue[Combinator]
	var components []*InterpolatedComplexSelectorComponent

loop:
	for {
		if err := p.whitespace(false); err != nil {
			return nil, err
		}

		allowCombinator := combinator == nil && (allowLeadingCombinator || lastCompound != nil)

		switch ch := p.scanner.PeekChar(0); {
		case ch == '+' && allowCombinator:
			combinatorStart := p.scanner.State()
			if _, err = p.readChar(); err != nil {
				return nil, err
			}
			span, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			c := sasscommon.NewCssValue(CombinatorNextSibling, span)
			combinator = &c

		case ch == '>' && allowCombinator:
			combinatorStart := p.scanner.State()
			if _, err = p.readChar(); err != nil {
				return nil, err
			}
			span, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			c := sasscommon.NewCssValue(CombinatorChild, span)
			combinator = &c

		case ch == '~' && allowCombinator:
			combinatorStart := p.scanner.State()
			if _, err = p.readChar(); err != nil {
				return nil, err
			}
			span, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			c := sasscommon.NewCssValue(CombinatorFollowingSibling, span)
			combinator = &c

		case ch < 0:
			break loop

		case ch == '[' || ch == '.' || ch == '#' || ch == '%' || ch == ':' || ch == '&' || ch == '*' || ch == '|':
			if lastCompound != nil {
				span, err := p.spanFrom(componentStart)
				if err != nil {
					return nil, err
				}
				comp := NewInterpolatedComplexSelectorComponent(lastCompound, span, combinator)
				components = append(components, comp)
			} else if combinator != nil {
				leadingCombinator = combinator
				componentStart = p.scanner.State()
			}

			lastCompound, err = p.compoundSelector()
			combinator = nil
			if err != nil {
				return nil, err
			}
			if p.scanner.PeekChar(0) == '&' {
				return nil, p.scanner.Error("\"&\" may only used at the beginning of a compound selector.", -1, 0)
			}

		case p.lookingAtInterpolatedIdentifier():
			if lastCompound != nil {
				span, err := p.spanFrom(componentStart)
				if err != nil {
					return nil, err
				}
				comp := NewInterpolatedComplexSelectorComponent(lastCompound, span, combinator)
				components = append(components, comp)
			} else if combinator != nil {
				leadingCombinator = combinator
				componentStart = p.scanner.State()
			}

			lastCompound, err = p.compoundSelector()
			combinator = nil
			if err != nil {
				return nil, err
			}
			if p.scanner.PeekChar(0) == '&' {
				return nil, p.scanner.Error("\"&\" may only used at the beginning of a compound selector.", -1, 0)
			}

		default:
			break loop
		}
	}

	if combinator != nil && (p.plainCss || !allowTrailingCombinator) {
		return nil, p.scanner.Error("expected selector.", -1, 0)
	} else if lastCompound != nil {
		span, err := p.spanFrom(componentStart)
		if err != nil {
			return nil, err
		}
		components = append(components,
			NewInterpolatedComplexSelectorComponent(lastCompound, span, combinator))
	} else if combinator != nil {
		leadingCombinator = combinator
	} else {
		return nil, p.scanner.Error("expected selector.", -1, 0)
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	result, err := NewInterpolatedComplexSelector(components, span, leadingCombinator)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// compoundSelector consumes a compound selector: one simple selector
// followed by more while they start. Continuation selectors may only use
// `&` under the plain-CSS rule, matching the stylesheet's plainCss flag.
func (p *StylesheetParser) compoundSelector() (*InterpolatedCompoundSelector, error) {
	first, err := p.simpleSelector(true)
	if err != nil {
		return nil, err
	}
	components := []InterpolatedSimpleSelector{first}

	for p.isSimpleSelectorStart(p.scanner.PeekChar(0)) {
		sel, err := p.simpleSelector(p.plainCss)
		if err != nil {
			return nil, err
		}
		components = append(components, sel)
	}

	result, err := NewInterpolatedCompoundSelector(components)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// isSimpleSelectorStart reports whether ch can continue a compound
// selector. `&` only continues one in plain CSS, where parent selectors
// parse as plain selectors rather than Sass references.
func (p *StylesheetParser) isSimpleSelectorStart(ch int) bool {
	switch ch {
	case '*', '[', '.', '#', '%', ':':
		return true
	case '&':
		return p.plainCss
	default:
		return false
	}
}

// simpleSelector consumes one simple selector, dispatched on the next
// character. `#` followed by `#{` is interpolation rather than an ID and
// falls through to the type/universal path. Placeholders are rejected in
// plain CSS, and parent selectors are rejected unless allowParent permits
// them here.
func (p *StylesheetParser) simpleSelector(allowParent bool) (InterpolatedSimpleSelector, error) {
	start := p.scanner.State()

	switch ch := p.scanner.PeekChar(0); {
	case ch == '[':
		return p.attributeSelector()
	case ch == '.':
		return p.classSelector()
	case ch == '#':
		if p.scanner.PeekChar(1) != '{' {
			return p.idSelector()
		}
		return p.typeOrUniversalSelector()
	case ch == '%':
		sel, err := p.placeholderSelector()
		if err != nil {
			return nil, err
		}
		if p.plainCss {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			if err := p.error("Placeholder selectors aren't allowed in plain CSS.", span, nil); err != nil {
				return nil, err
			}
		}
		return sel, nil
	case ch == ':':
		return p.pseudoSelector()
	case ch == '&':
		sel, err := p.parentSelector()
		if err != nil {
			return nil, err
		}
		if !allowParent {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			if err := p.error("Parent selectors aren't allowed here.", span, nil); err != nil {
				return nil, err
			}
		}
		return sel, nil
	default:
		return p.typeOrUniversalSelector()
	}
}

// attributeSelector consumes an attribute selector. A bare `[name]` returns
// immediately; otherwise an operator follows, then a value that parses as a
// quoted string token when quoted and as an identifier when not, plus an
// optional trailing modifier identifier.
func (p *StylesheetParser) attributeSelector() (*InterpolatedAttributeSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('['); err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	name, err := p.attributeName()
	if err != nil {
		return nil, err
	}

	if err := p.whitespace(true); err != nil {
		return nil, err
	}
	if p.scanner.ScanChar(']') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedAttributeSelector(name, span), nil
	}

	op, err := p.attributeOperator()
	if err != nil {
		return nil, err
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	next := p.scanner.PeekChar(0)
	var value *Interpolation
	if next == '\'' || next == '"' {
		var istErr error
		value, istErr = p.interpolatedStringToken()
		if istErr != nil {
			return nil, istErr
		}
	} else {
		value, err = p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var modifier *Interpolation
	if p.lookingAtInterpolatedIdentifier() {
		modifier, err = p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	if err := p.expectChar(']'); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewInterpolatedAttributeSelectorWithOperator(name, op, value, span, modifier), nil
}

// attributeName consumes a qualified name inside an attribute selector:
// `*|name`, `|name` (empty namespace), `ns|name`, or a bare name. A `|`
// followed by `=` is the `[a|=b]` operator rather than a namespace
// separator, so the name stands alone in that case.
func (p *StylesheetParser) attributeName() (*InterpolatedQualifiedName, error) {
	start := p.scanner.State()
	if p.scanner.ScanChar('*') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		namespace := NewInterpolationPlain("*", span)
		if err := p.expectChar('|'); err != nil {
			return nil, err
		}
		name, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		span, err = p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedQualifiedName(name, span, namespace), nil
	}

	if p.scanner.ScanChar('|') {
		spanTo, err := p.spanFromTo(start, &start)
		if err != nil {
			return nil, err
		}
		namespace := NewInterpolationPlain("", spanTo)
		name, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedQualifiedName(name, span, namespace), nil
	}

	nameOrNamespace, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	if p.scanner.PeekChar(0) != '|' || p.scanner.PeekChar(1) == '=' {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedQualifiedName(nameOrNamespace, span, nil), nil
	}

	if _, err := p.readChar(); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewInterpolatedQualifiedName(name, span, nameOrNamespace), nil
}

// attributeOperator consumes an attribute selector's operator: `=`, `~=`,
// `|=`, `^=`, `$=`, or `*=`. Anything else fails with `Expected "]".`
// positioned at the operator start.
func (p *StylesheetParser) attributeOperator() (sasscommon.CssValue[AttributeOperator], error) {
	start := p.scanner.State()
	var op AttributeOperator
	ch, err := p.readChar()
	if err != nil {
		return sasscommon.CssValue[AttributeOperator]{}, err
	}
	switch ch {
	case '=':
		op = AttributeOperatorEqual
	case '~':
		if err := p.expectChar('='); err != nil {
			return sasscommon.CssValue[AttributeOperator]{}, err
		}
		op = AttributeOperatorInclude
	case '|':
		if err := p.expectChar('='); err != nil {
			return sasscommon.CssValue[AttributeOperator]{}, err
		}
		op = AttributeOperatorDash
	case '^':
		if err := p.expectChar('='); err != nil {
			return sasscommon.CssValue[AttributeOperator]{}, err
		}
		op = AttributeOperatorPrefix
	case '$':
		if err := p.expectChar('='); err != nil {
			return sasscommon.CssValue[AttributeOperator]{}, err
		}
		op = AttributeOperatorSuffix
	case '*':
		if err := p.expectChar('='); err != nil {
			return sasscommon.CssValue[AttributeOperator]{}, err
		}
		op = AttributeOperatorSubstring
	default:
		return sasscommon.CssValue[AttributeOperator]{}, p.scanner.Error("Expected \"]\".", start.Position, 1)
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return sasscommon.CssValue[AttributeOperator]{}, err
	}
	return sasscommon.NewCssValue(op, span), nil
}

// classSelector consumes a class selector: `.` followed by an interpolated
// identifier.
func (p *StylesheetParser) classSelector() (*InterpolatedClassSelector, error) {
	if err := p.expectChar('.'); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	sel, err := NewInterpolatedClassSelector(name)
	if err != nil {
		return nil, err
	}
	return sel, nil
}

// idSelector consumes an ID selector: `#` followed by an interpolated
// identifier (callers route `#` followed by `#{` elsewhere).
func (p *StylesheetParser) idSelector() (*InterpolatedIDSelector, error) {
	if err := p.expectChar('#'); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	sel, err := NewInterpolatedIDSelector(name)
	if err != nil {
		return nil, err
	}
	return sel, nil
}

// placeholderSelector consumes a placeholder selector: `%` followed by an
// interpolated identifier.
func (p *StylesheetParser) placeholderSelector() (*InterpolatedPlaceholderSelector, error) {
	if err := p.expectChar('%'); err != nil {
		return nil, err
	}
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	sel, err := NewInterpolatedPlaceholderSelector(name)
	if err != nil {
		return nil, err
	}
	return sel, nil
}

// parentSelector consumes a parent selector: `&` with an optional
// identifier-body suffix (`&-suffix`). Suffixed parents are rejected in
// plain CSS.
func (p *StylesheetParser) parentSelector() (*InterpolatedParentSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('&'); err != nil {
		return nil, err
	}
	var suffix *Interpolation
	if p.lookingAtInterpolatedIdentifierBody() {
		var err error
		suffix, err = p.interpolatedIdentifierBody()
		if err != nil {
			return nil, err
		}
	}
	if p.plainCss && suffix != nil {
		return nil, p.scanner.Error(
			"Parent selectors can't have suffixes in plain CSS.",
			start.Position,
			p.scanner.Position()-start.Position,
		)
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewInterpolatedParentSelector(span, suffix), nil
}

// pseudoSelector consumes a pseudo class or element, dispatched on the
// unvendored name once a parenthesized argument opens. Selector-bearing
// pseudo elements and pseudo classes parse a nested selector list;
// `nth-child`/`nth-last-child` parse a declaration value that may continue
// into a selector list (`of <selector>`); anything else takes a plain
// declaration-value argument. Without parens the pseudo has no argument.
func (p *StylesheetParser) pseudoSelector() (*InterpolatedPseudoSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar(':'); err != nil {
		return nil, err
	}
	element := p.scanner.ScanChar(':')
	name, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}

	if !p.scanner.ScanChar('(') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedPseudoSelector(name, span, element, nil, nil), nil
	}
	if err := p.whitespace(true); err != nil {
		return nil, err
	}

	var unvendored string
	if plain := name.AsPlain(); plain != nil {
		unvendored = unvendor.Unvendor(*plain)
	}
	var argument *Interpolation
	var selectorList *InterpolatedSelectorList
	if element {
		if selectorPseudoElements[unvendored] {
			selectorList, err = p.selectorList()
			if err != nil {
				return nil, err
			}
		} else {
			argument, err = p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
			if err != nil {
				return nil, err
			}
		}
	} else if selectorPseudoClasses[unvendored] {
		selectorList, err = p.selectorList()
		if err != nil {
			return nil, err
		}
	} else if unvendored == "nth-child" || unvendored == "nth-last-child" {
		argument, err = p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: false, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: true, silentComments: true, consumeNewlines: true})
		if err != nil {
			return nil, err
		}
		if p.scanner.PeekChar(0) != ')' {
			selectorList, err = p.selectorList()
			if err != nil {
				return nil, err
			}
		}
	} else {
		argument, err = p.interpolatedDeclarationValue(declarationValueOpts{allowEmpty: true, allowSemicolon: false, allowColon: true, allowOpenBrace: true, endAfterOf: false, silentComments: true, consumeNewlines: false})
		if err != nil {
			return nil, err
		}
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewInterpolatedPseudoSelector(name, span, element, argument, selectorList), nil
}

// typeOrUniversalSelector consumes a type selector or a universal selector.
// The two share one production because either can open with `*`: a bare `*`
// is universal, `*|...` carries an any-namespace, `|...` an empty namespace,
// and `ns|*` vs `ns|name` split on the name after the bar.
func (p *StylesheetParser) typeOrUniversalSelector() (InterpolatedSimpleSelector, error) {
	start := p.scanner.State()
	if p.scanner.ScanChar('*') {
		afterAsterisk := p.scanner.State()
		if !p.scanner.ScanChar('|') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewInterpolatedUniversalSelector(span, nil), nil
		}
		spanTo, err := p.spanFromTo(start, &afterAsterisk)
		if err != nil {
			return nil, err
		}
		namespace := NewInterpolationPlain("*", spanTo)
		if p.scanner.ScanChar('*') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewInterpolatedUniversalSelector(span, namespace), nil
		}
		name, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedTypeSelector(
			NewInterpolatedQualifiedName(name, span, namespace)), nil
	} else if p.scanner.ScanChar('|') {
		spanTo, err := p.spanFromTo(start, &start)
		if err != nil {
			return nil, err
		}
		namespace := NewInterpolationPlain("", spanTo)
		if p.scanner.ScanChar('*') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewInterpolatedUniversalSelector(span, namespace), nil
		}
		name, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedTypeSelector(
			NewInterpolatedQualifiedName(name, span, namespace)), nil
	}

	nameOrNamespace, err := p.interpolatedIdentifier()
	if err != nil {
		return nil, err
	}
	if !p.scanner.ScanChar('|') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedTypeSelector(
			NewInterpolatedQualifiedName(nameOrNamespace, span, nil)), nil
	} else if p.scanner.ScanChar('*') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedUniversalSelector(span, nameOrNamespace), nil
	} else {
		name, err := p.interpolatedIdentifier()
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewInterpolatedTypeSelector(
			NewInterpolatedQualifiedName(name, span, nameOrNamespace)), nil
	}
}
