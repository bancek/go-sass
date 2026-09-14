// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/selector.dart

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/unvendor"
	"github.com/bancek/go-sass/util"
)

// Pseudo-class selectors that take unadorned selectors as arguments.
//
// dart-source: lib/src/parse/selector.dart (selectorPseudoClasses).
// Internal in Dart (@internal): resolution-level list shared with the
// stylesheet-level interpolated selector parser, which applies the same
// pseudo-argument rules.
var selectorPseudoClasses = map[string]bool{
	"not": true, "is": true, "matches": true, "where": true,
	"current": true, "any": true, "has": true, "host": true,
	"host-context": true,
}

// Pseudo-element selectors that take unadorned selectors as arguments.
//
// dart-source: lib/src/parse/selector.dart (selectorPseudoElements).
// Internal in Dart (@internal); only `slotted` parses its argument as a
// selector list, everything else falls back to a declaration value.
var selectorPseudoElements = map[string]bool{
	"slotted": true,
}

// SelectorParser parses resolution-level selectors: plain selectors with no
// interpolation.
//
// Use this to re-parse selector text produced by interpolation and by
// built-in functions. This is largely duplicated with the stylesheet-level
// interpolated selector parsing in parse_stylesheet_selector.go; most changes
// here should be mirrored there and vice versa.
type SelectorParser struct {
	Parser
	allowParent       bool
	plainCss          bool
	logger            sasslogger.Logger
	warnDeprecationFn func(string, *deprecation.Deprecation) error
}

// SelectorParserOptions contains optional configuration for SelectorParser.
//
// If nil or a zero-value struct is passed, defaults are used:
//   - AllowParent defaults to true
//   - PlainCss defaults to false
//   - Logger defaults to the default stderr logger
//   - WarnDeprecationFn, if set, is called for deprecation warnings instead of
//     Logger.WarnDeprecation. It should generate a proper stack trace and use
//     the correct evaluation context span (not the parser-internal span).
type SelectorParserOptions struct {
	AllowParent       *bool
	PlainCss          *bool
	Logger            sasslogger.Logger
	WarnDeprecationFn func(string, *deprecation.Deprecation) error
}

// NewSelectorParser creates a new parser that parses CSS selectors.
//
// If AllowParent is false, parsing reports a SassFormatException when the
// selector contains the parent selector `&`. If PlainCss is true, the input
// is parsed as a plain CSS selector rather than a Sass selector, so Sass-only
// constructs (placeholders, parent suffixes, trailing combinators) are
// rejected. Logger reports deprecation warnings and defaults to the default
// stderr logger; WarnDeprecationFn, when set, replaces Logger.WarnDeprecation
// so callers can attach a proper stack trace and evaluation-context span
// instead of the parser-internal span.
//
// Matches Dart: SelectorParser constructor
func NewSelectorParser(contents string, url *url.URL, interpolationMap *InterpolationMap, opts *SelectorParserOptions) *SelectorParser {
	allowParent := true
	plainCss := false
	var logger sasslogger.Logger
	var warnDepFn func(string, *deprecation.Deprecation) error
	if opts != nil {
		if opts.AllowParent != nil {
			allowParent = *opts.AllowParent
		}
		if opts.PlainCss != nil {
			plainCss = *opts.PlainCss
		}
		if opts.Logger != nil {
			logger = opts.Logger
		}
		warnDepFn = opts.WarnDeprecationFn
	}
	if logger == nil {
		logger = sasslogger.NewDefaultLogger(true)
	}
	return &SelectorParser{
		Parser:            *NewParser([]byte(contents), url, interpolationMap),
		allowParent:       allowParent,
		plainCss:          plainCss,
		logger:            logger,
		warnDeprecationFn: warnDepFn,
	}
}

// Parse parses a selector list and requires the whole input to be consumed.
//
// Matches Dart: SelectorParser.parse
func (p *SelectorParser) Parse() (*SelectorList, error) {
	var result *SelectorList
	err := p.wrapSpanFormatException(func() error {
		var err error
		result, err = p._selectorList()
		if err != nil {
			return err
		}
		if !p.scanner.IsDone() {
			return p.scanner.Error("expected selector.", -1, 0)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseComplexSelector parses a single complex selector and requires the
// whole input to be consumed.
//
// Matches Dart: SelectorParser.parseComplexSelector
func (p *SelectorParser) ParseComplexSelector() (*ComplexSelector, error) {
	var result *ComplexSelector
	err := p.wrapSpanFormatException(func() error {
		var err error
		result, err = p._complexSelector(false)
		if err != nil {
			return err
		}
		if !p.scanner.IsDone() {
			return p.scanner.Error("expected selector.", -1, 0)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseCompoundSelector parses a single compound selector and requires the
// whole input to be consumed.
//
// Matches Dart: SelectorParser.parseCompoundSelector
func (p *SelectorParser) ParseCompoundSelector() (*CompoundSelector, error) {
	var result *CompoundSelector
	err := p.wrapSpanFormatException(func() error {
		var err error
		result, err = p._compoundSelector()
		if err != nil {
			return err
		}
		if !p.scanner.IsDone() {
			return p.scanner.Error("expected selector.", -1, 0)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseSimpleSelector parses a single simple selector and requires the whole
// input to be consumed. Whether `&` is accepted is controlled by the
// AllowParent option.
//
// Matches Dart: SelectorParser.parseSimpleSelector
func (p *SelectorParser) ParseSimpleSelector() (SimpleSelector, error) {
	var result SimpleSelector
	err := p.wrapSpanFormatException(func() error {
		var err error
		result, err = p._simpleSelector(p.allowParent)
		if err != nil {
			return err
		}
		if !p.scanner.IsDone() {
			return p.scanner.Error("unexpected token.", -1, 0)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// _selectorList consumes a comma-separated selector list.
//
// Doubled commas are skipped and a trailing comma ends the list. A line break
// before each subsequent complex selector is recorded so downstream
// serialization can preserve it.
func (p *SelectorParser) _selectorList() (*SelectorList, error) {
	start := p.scanner.State()
	previousLine := p.scanner.Line()
	cs, err := p._complexSelector(false)
	if err != nil {
		return nil, err
	}
	components := []*ComplexSelector{cs}

	if err := p._whitespace(); err != nil {
		return nil, err
	}
	for p.scanner.ScanChar(',') {
		if err := p._whitespace(); err != nil {
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
		cs, err := p._complexSelector(lineBreak)
		if err != nil {
			return nil, err
		}
		components = append(components, cs)
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	result, err := NewSelectorList(components, span)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// _complexSelector consumes a complex selector.
//
// When lineBreak is set, a line break preceded this selector. Leading
// combinators are kept as the complex selector's initial combinators; a
// trailing combinator with no compound after it is an "expected selector."
// error in plain CSS, and otherwise becomes the trailing initial combinator.
// Two compounds with no combinator or whitespace between them keep parsing
// but emit the adjacent-compounds deprecation, and `&` anywhere but the start
// of a compound is an error.
func (p *SelectorParser) _complexSelector(lineBreak bool) (*ComplexSelector, error) {
	start := p.scanner.State()

	componentStart := p.scanner.State()
	var lastCompound *CompoundSelector
	var combinators []sasscommon.CssValue[Combinator]

	var initialCombinators []sasscommon.CssValue[Combinator]
	var components []*ComplexSelectorComponent

	for {
		beforeWhitespace := p.scanner.Position()
		if err := p._whitespace(); err != nil {
			return nil, err
		}
		consumedWhitespace := p.scanner.Position() != beforeWhitespace

		ch := p.scanner.PeekChar(0)
		switch {
		case ch == '+':
			combinatorStart := p.scanner.State()
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			combinatorSpan, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			combinators = append(combinators, sasscommon.NewCssValue(CombinatorNextSibling, combinatorSpan))

		case ch == '>':
			combinatorStart := p.scanner.State()
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			combinatorSpan, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			combinators = append(combinators, sasscommon.NewCssValue(CombinatorChild, combinatorSpan))

		case ch == '~':
			combinatorStart := p.scanner.State()
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			combinatorSpan, err := p.spanFrom(combinatorStart)
			if err != nil {
				return nil, err
			}
			combinators = append(combinators, sasscommon.NewCssValue(CombinatorFollowingSibling, combinatorSpan))

		case ch < 0:
			goto done

		default:
			isSimpleStart := ch == '[' || ch == '.' || ch == '#' || ch == '%' || ch == ':' || ch == '&' || ch == '*' || ch == '|'
			if !isSimpleStart && !p.lookingAtIdentifier(nil) {
				goto done
			}

			if lastCompound != nil {
				componentSpan, err := p.spanFrom(componentStart)
				if err != nil {
					return nil, err
				}
				components = append(components, NewComplexSelectorComponent(lastCompound, combinators, componentSpan))
			} else if len(combinators) > 0 {
				initialCombinators = combinators
				componentStart = p.scanner.State()
			}

			nextCompound, err := p._compoundSelector()
			if err != nil {
				return nil, err
			}

			if lastCompound != nil && len(combinators) == 0 && !consumedWhitespace {
				lastSpan, err := lastCompound.Span()
				if err != nil {
					return nil, err
				}
				nextSpan, err := nextCompound.Span()
				if err != nil {
					return nil, err
				}
				warnSpan, err := lastSpan.Expand(nextSpan)
				if err != nil {
					return nil, err
				}
				lastText, err := lastSpan.SpanText()
				if err != nil {
					return nil, err
				}
				nextText, err := nextSpan.SpanText()
				if err != nil {
					return nil, err
				}
				msg := fmt.Sprintf(
					"Adjacent compound selectors must be separated by whitespace. "+
						"This will be an error in Dart Sass 2.0.0. Suggestion:\n"+
						"\n"+
						"%s %s\n"+
						"\n"+
						"More info: https://sass-lang.com/d/adjacent-compounds",
					lastText, nextText,
				)
				if p.warnDeprecationFn != nil {
					if err := p.warnDeprecationFn(msg, deprecation.AdjacentCompounds); err != nil {
						return nil, err
					}
				} else if err := p.logger.WarnDeprecation(
					msg,
					&warnSpan,
					deprecation.AdjacentCompounds,
					nil,
				); err != nil {
					return nil, err
				}
			}

			lastCompound = nextCompound
			combinators = nil
			if p.scanner.PeekChar(0) == '&' {
				return nil, p.scanner.Error("\"&\" may only used at the beginning of a compound selector.", -1, 0)
			}
		}
	}
done:

	if len(combinators) > 0 && p.plainCss {
		return nil, p.scanner.Error("expected selector.", -1, 0)
	} else if lastCompound != nil {
		componentSpan, err := p.spanFrom(componentStart)
		if err != nil {
			return nil, err
		}
		components = append(components, NewComplexSelectorComponent(lastCompound, combinators, componentSpan))
	} else if len(combinators) > 0 {
		initialCombinators = combinators
	} else {
		return nil, p.scanner.Error("expected selector.", -1, 0)
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	cs, err := NewComplexSelector(initialCombinators, components, span, lineBreak)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// _compoundSelector consumes a compound selector: one simple selector
// followed by any further simples that can continue the compound.
func (p *SelectorParser) _compoundSelector() (*CompoundSelector, error) {
	start := p.scanner.State()
	sel, err := p._simpleSelector(p.allowParent)
	if err != nil {
		return nil, err
	}
	components := []SimpleSelector{sel}

	for p._isSimpleSelectorStart(p.scanner.PeekChar(0)) {
		sel, err := p._simpleSelector(p.plainCss)
		if err != nil {
			return nil, err
		}
		components = append(components, sel)
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	result, err := NewCompoundSelector(components, span)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// _simpleSelector consumes a simple selector.
//
// When allowParent is passed it controls whether `&` is accepted here;
// otherwise the parser-wide allowParent setting applies. Placeholder
// selectors are rejected up front in plain CSS mode.
func (p *SelectorParser) _simpleSelector(allowParent bool) (SimpleSelector, error) {
	start := p.scanner.State()

	switch ch := p.scanner.PeekChar(0); {
	case ch == '[':
		return p._attributeSelector()
	case ch == '.':
		return p._classSelector()
	case ch == '#':
		return p._idSelector()
	case ch == '%':
		sel, err := p._placeholderSelector()
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
		return p._pseudoSelector()
	case ch == '&':
		sel, err := p._parentSelector()
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
		return p._typeOrUniversalSelector()
	}
}

// _attributeSelector consumes an attribute selector, including the optional
// operator, value (quoted string or bare identifier), and case modifier.
func (p *SelectorParser) _attributeSelector() (*AttributeSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('['); err != nil {
		return nil, err
	}
	if err := p._whitespace(); err != nil {
		return nil, err
	}

	name, err := p._attributeName()
	if err != nil {
		return nil, err
	}

	if err := p._whitespace(); err != nil {
		return nil, err
	}
	if p.scanner.ScanChar(']') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewAttributeSelector(name, span), nil
	}

	op, err := p._attributeOperator()
	if err != nil {
		return nil, err
	}
	if err := p._whitespace(); err != nil {
		return nil, err
	}

	next := p.scanner.PeekChar(0)
	var value string
	var valErr error
	if next == '\'' || next == '"' {
		value, valErr = p.string()
	} else {
		value, valErr = p.identifier(false, false)
	}
	if valErr != nil {
		return nil, valErr
	}
	if err := p._whitespace(); err != nil {
		return nil, err
	}

	next = p.scanner.PeekChar(0)
	var modifier *string
	if next >= 0 && util.IsAlphabetic(next) {
		ch, err := p.readChar()
		if err != nil {
			return nil, err
		}
		s := string(rune(ch))
		modifier = &s
	}

	if err := p.expectChar(']'); err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewAttributeSelectorWithOperator(name, op, value, span, modifier), nil
}

// _attributeName consumes a qualified name inside an attribute selector.
//
// A `|` that is followed by `=` belongs to the operator, not the namespace,
// so `name|=value` still parses as name + `|=` operator.
func (p *SelectorParser) _attributeName() (QualifiedName, error) {
	if p.scanner.ScanChar('*') {
		if err := p.expectChar('|'); err != nil {
			return QualifiedName{}, err
		}
		wildcard := "*"
		name, err := p.identifier(false, false)
		if err != nil {
			return QualifiedName{}, err
		}
		return NewQualifiedNameWithNamespace(name, &wildcard), nil
	}

	if p.scanner.ScanChar('|') {
		empty := ""
		name, err := p.identifier(false, false)
		if err != nil {
			return QualifiedName{}, err
		}
		return NewQualifiedNameWithNamespace(name, &empty), nil
	}

	nameOrNamespace, err := p.identifier(false, false)
	if err != nil {
		return QualifiedName{}, err
	}
	if p.scanner.PeekChar(0) != '|' || p.scanner.PeekChar(1) == '=' {
		return NewQualifiedName(nameOrNamespace), nil
	}

	if _, err := p.readChar(); err != nil {
		return QualifiedName{}, err
	}
	name, err := p.identifier(false, false)
	if err != nil {
		return QualifiedName{}, err
	}
	return NewQualifiedNameWithNamespace(name, &nameOrNamespace), nil
}

// _attributeOperator consumes an attribute selector's operator (`=`, `~=`,
// `|=`, `^=`, `$=`, `*=`).
func (p *SelectorParser) _attributeOperator() (AttributeOperator, error) {
	start := p.scanner.Position()
	ch, err := p.readChar()
	if err != nil {
		return 0, err
	}
	switch ch {
	case '=':
		return AttributeOperatorEqual, nil
	case '~':
		if err := p.expectChar('='); err != nil {
			return 0, err
		}
		return AttributeOperatorInclude, nil
	case '|':
		if err := p.expectChar('='); err != nil {
			return 0, err
		}
		return AttributeOperatorDash, nil
	case '^':
		if err := p.expectChar('='); err != nil {
			return 0, err
		}
		return AttributeOperatorPrefix, nil
	case '$':
		if err := p.expectChar('='); err != nil {
			return 0, err
		}
		return AttributeOperatorSuffix, nil
	case '*':
		if err := p.expectChar('='); err != nil {
			return 0, err
		}
		return AttributeOperatorSubstring, nil
	default:
		return 0, p.scanner.Error("Expected \"]\".", start, 1)
	}
}

// _classSelector consumes a class selector (`.` plus identifier).
func (p *SelectorParser) _classSelector() (*ClassSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('.'); err != nil {
		return nil, err
	}
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewClassSelector(name, span), nil
}

// _idSelector consumes an ID selector (`#` plus identifier).
func (p *SelectorParser) _idSelector() (*IDSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('#'); err != nil {
		return nil, err
	}
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewIDSelector(name, span), nil
}

// _placeholderSelector consumes a placeholder selector (`%` plus identifier).
// The plain-CSS rejection lives in _simpleSelector, not here.
func (p *SelectorParser) _placeholderSelector() (*PlaceholderSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('%'); err != nil {
		return nil, err
	}
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewPlaceholderSelector(name, span), nil
}

// _parentSelector consumes a parent selector (`&` plus an optional suffix
// such as `-suffix`). Suffixes are rejected in plain CSS mode.
func (p *SelectorParser) _parentSelector() (*ParentSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar('&'); err != nil {
		return nil, err
	}
	var suffix *string
	if p.lookingAtIdentifierBody() {
		s, err := p.identifierBody()
		if err != nil {
			return nil, err
		}
		suffix = &s
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
	return NewParentSelector(span, suffix), nil
}

// _pseudoSelector consumes a pseudo selector.
//
// Functional pseudos whose names are vendor-prefix-independent take either a
// nested selector list (`:not()`, `:is()`, `:slotted()`, …) or an `An+B`
// microsyntax argument (`:nth-child()` with an optional `of <selector>`
// clause); anything else takes a raw declaration value, trimmed of trailing
// whitespace.
func (p *SelectorParser) _pseudoSelector() (*PseudoSelector, error) {
	start := p.scanner.State()
	if err := p.expectChar(':'); err != nil {
		return nil, err
	}
	element := p.scanner.ScanChar(':')
	name, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}

	if !p.scanner.ScanChar('(') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewPseudoSelector(name, span, element, nil, nil), nil
	}
	if err := p._whitespace(); err != nil {
		return nil, err
	}

	unvendored := unvendor.Unvendor(name)
	var argument *string
	var sel Selector
	if element {
		if selectorPseudoElements[unvendored] {
			sel, err = p._selectorList()
			if err != nil {
				return nil, err
			}
		} else {
			arg, err := p.declarationValue(true)
			if err != nil {
				return nil, err
			}
			argument = &arg
		}
	} else if selectorPseudoClasses[unvendored] {
		sel, err = p._selectorList()
		if err != nil {
			return nil, err
		}
	} else if unvendored == "nth-child" || unvendored == "nth-last-child" {
		arg, err := p._aNPlusB()
		if err != nil {
			return nil, err
		}
		if err := p._whitespace(); err != nil {
			return nil, err
		}
		if p.scanner.PeekChar(-1) >= 0 && util.IsWhitespace(p.scanner.PeekChar(-1)) && p.scanner.PeekChar(0) != ')' {
			if err := p.expectIdentifier("of", "", false); err != nil {
				return nil, err
			}
			arg += " of"
			if err := p._whitespace(); err != nil {
				return nil, err
			}
			sel, err = p._selectorList()
			if err != nil {
				return nil, err
			}
		}
		argument = &arg
	} else {
		val, err := p.declarationValue(true)
		if err != nil {
			return nil, err
		}
		arg := strings.TrimRight(val, " \t\r\n")
		argument = &arg
	}
	if err := p.expectChar(')'); err != nil {
		return nil, err
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}
	return NewPseudoSelector(name, span, element, argument, sel), nil
}

// _aNPlusB consumes an `An+B` microsyntax production (CSS Syntax 3) and
// returns its source text, covering `even`, `odd`, bare `n`, and signed
// `n ± B` forms. Whitespace around the `n` and sign is allowed; a bare number
// with no `n` is returned as-is.
func (p *SelectorParser) _aNPlusB() (string, error) {
	var sb strings.Builder
	switch ch := p.scanner.PeekChar(0); {
	case ch == 'e' || ch == 'E':
		if err := p.expectIdentifier("even", "", false); err != nil {
			return "", err
		}
		return "even", nil
	case ch == 'o' || ch == 'O':
		if err := p.expectIdentifier("odd", "", false); err != nil {
			return "", err
		}
		return "odd", nil
	case ch == '+' || ch == '-':
		b, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte(b))
	}

	if util.IsDigit(p.scanner.PeekChar(0)) {
		for util.IsDigit(p.scanner.PeekChar(0)) {
			b, err := p.readChar()
			if err != nil {
				return "", err
			}
			sb.WriteByte(byte(b))
		}
		if err := p._whitespace(); err != nil {
			return "", err
		}
		ok, err := p.scanIdentChar('n', false)
		if err != nil {
			return "", err
		}
		if !ok {
			return sb.String(), nil
		}
	} else {
		if err := p.expectIdentChar('n', false); err != nil {
			return "", err
		}
	}
	sb.WriteByte('n')
	if err := p._whitespace(); err != nil {
		return "", err
	}

	next := p.scanner.PeekChar(0)
	if next != '+' && next != '-' {
		return sb.String(), nil
	}
	b, err := p.readChar()
	if err != nil {
		return "", err
	}
	sb.WriteByte(byte(b))
	if err := p._whitespace(); err != nil {
		return "", err
	}

	if !util.IsDigit(p.scanner.PeekChar(0)) {
		return "", p.scanner.Error("Expected a number.", -1, 0)
	}
	for util.IsDigit(p.scanner.PeekChar(0)) {
		b, err := p.readChar()
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte(b))
	}
	return sb.String(), nil
}

// _typeOrUniversalSelector consumes a type selector or a universal selector.
//
// The two are combined because either one can start with `*`: `*` alone or
// `*|*` is universal, while `*|name` is a type selector, and likewise for the
// empty-namespace `|` prefix.
func (p *SelectorParser) _typeOrUniversalSelector() (SimpleSelector, error) {
	start := p.scanner.State()
	if p.scanner.ScanChar('*') {
		if !p.scanner.ScanChar('|') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewUniversalSelector(span, nil), nil
		}
		wildcard := "*"
		if p.scanner.ScanChar('*') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewUniversalSelector(span, &wildcard), nil
		}
		name, err := p.identifier(false, false)
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewTypeSelector(
			NewQualifiedNameWithNamespace(name, &wildcard),
			span), nil
	} else if p.scanner.ScanChar('|') {
		empty := ""
		if p.scanner.ScanChar('*') {
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			return NewUniversalSelector(span, &empty), nil
		}
		name, err := p.identifier(false, false)
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewTypeSelector(
			NewQualifiedNameWithNamespace(name, &empty),
			span), nil
	}

	nameOrNamespace, err := p.identifier(false, false)
	if err != nil {
		return nil, err
	}
	if !p.scanner.ScanChar('|') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewTypeSelector(NewQualifiedName(nameOrNamespace), span), nil
	} else if p.scanner.ScanChar('*') {
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewUniversalSelector(span, &nameOrNamespace), nil
	} else {
		name, err := p.identifier(false, false)
		if err != nil {
			return nil, err
		}
		span, err := p.spanFrom(start)
		if err != nil {
			return nil, err
		}
		return NewTypeSelector(
			NewQualifiedNameWithNamespace(name, &nameOrNamespace),
			span), nil
	}
}

// _isSimpleSelectorStart reports whether ch can start a simple selector in
// the middle of a compound selector. `&` only continues a compound in plain
// CSS mode, where suffixes are forbidden and nesting never applies.
func (p *SelectorParser) _isSimpleSelectorStart(ch int) bool {
	switch ch {
	case '*', '[', '.', '#', '%', ':':
		return true
	case '&':
		return p.plainCss
	default:
		return false
	}
}

// _whitespace consumes selector whitespace. The consume-newlines flag is
// fixed to true here; selector parsing always spans lines.
func (p *SelectorParser) _whitespace() error {
	return p.whitespace(true)
}
