// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/complex.dart

import (
	"net/url"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sasslogger"
)

// ComplexSelectorEquals compares complex selectors by structural equality,
// ignoring spans. It is used as the comparator for ordered maps keyed by
// complex selectors.
var ComplexSelectorEquals = func(a, b *ComplexSelector) bool { return a.Equals(b) }

// A complex selector.
//
// A complex selector is composed of CompoundSelectors separated by
// Combinators. It selects elements based on their ancestors and siblings.
type ComplexSelector struct {
	base SelectorBase
	// LeadingCombinators are combinators before the first component. Empty
	// means no leading combinator; more than one is invalid CSS but still
	// supported for backwards compatibility.
	LeadingCombinators []sasscommon.CssValue[Combinator]
	// Components are the compound-plus-trailing-combinator pairs. Adjacent
	// compounds imply a descendant combinator. Empty only when
	// LeadingCombinators is non-empty.
	Components []*ComplexSelectorComponent
	// LineBreak records whether a line break is emitted before this selector.
	// For internal use by the serializer.
	LineBreak bool
}

// NewComplexSelector creates a complex selector from leading combinators and
// components. It returns an error when both are empty.
func NewComplexSelector(
	leadingCombinators []sasscommon.CssValue[Combinator],
	components []*ComplexSelectorComponent,
	span sasscommon.FileSpan,
	lineBreak bool,
) (*ComplexSelector, error) {
	if len(leadingCombinators) == 0 && len(components) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "leadingCombinators and components may not both be empty."}
	}
	lc := make([]sasscommon.CssValue[Combinator], len(leadingCombinators))
	copy(lc, leadingCombinators)
	c := make([]*ComplexSelectorComponent, len(components))
	copy(c, components)
	return &ComplexSelector{
		base:               NewSelectorBase(span),
		LeadingCombinators: lc,
		Components:         c,
		LineBreak:          lineBreak,
	}, nil
}

// Span returns the source span where this selector was written.
func (c *ComplexSelector) Span() (sasscommon.FileSpan, error) { return c.base.Span() }
func (c *ComplexSelector) IsAstNode()                         {}
func (c *ComplexSelector) IsSelector()                        {}

// ComplexSelectorParse parses a complex selector from contents. If url is
// non-nil it names the file contents came from; allowParent controls whether
// a parent selector is accepted; logger receives deprecation warnings and may
// be nil to use the default logger. It returns an error when parsing fails.
func ComplexSelectorParse(contents string, url *url.URL, allowParent bool, logger sasslogger.Logger) (*ComplexSelector, error) {
	parser := NewSelectorParser(contents, url, nil, &SelectorParserOptions{
		AllowParent: &allowParent,
		Logger:      logger,
	})
	return parser.ParseComplexSelector()
}

// ContainsParentSelector reports whether any compound in this complex
// contains a parent selector.
func (c *ComplexSelector) ContainsParentSelector() (bool, error) {
	for _, comp := range c.Components {
		found, err := comp.Selector.ContainsParentSelector()
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// IsBogusOtherThanLeadingCombinator reports whether this complex is invalid
// CSS for reasons beyond a leading combinator: doubled leading or trailing
// combinators, or multiple adjacent combinators anywhere. Such selectors are
// kept for backwards compatibility and only warned about from custom
// functions.
func (c *ComplexSelector) IsBogusOtherThanLeadingCombinator() bool {
	if len(c.Components) == 0 {
		return len(c.LeadingCombinators) > 0
	}
	if len(c.LeadingCombinators) > 1 {
		return true
	}
	if len(c.Components[len(c.Components)-1].Combinators) > 0 {
		return true
	}
	for _, comp := range c.Components {
		if len(comp.Combinators) > 1 {
			return true
		}
		if comp.Selector.IsBogus() {
			return true
		}
	}
	return false
}

// IsInvisibleOtherThanBogusCombinators reports whether any compound in this
// complex keeps it from being emitted, ignoring bogus-combinator handling.
func (c *ComplexSelector) IsInvisibleOtherThanBogusCombinators() bool {
	for _, comp := range c.Components {
		if comp.Selector.IsInvisibleOtherThanBogusCombinators() {
			return true
		}
	}
	return false
}

// AssertNotBogus warns through warn when this selector is not valid CSS,
// forwarding to the shared helper.
func (c *ComplexSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(c, name, warn)
}

// Specificity returns the sum of the component compound specificities.
// Specificity is measured in base 1000, high enough that no single selector
// sequence plausibly reaches 1000 simple selectors.
func (c *ComplexSelector) Specificity() int {
	sum := 0
	for _, component := range c.Components {
		sum += component.Selector.Specificity()
	}
	return sum
}

// SingleCompound returns the sole compound selector when this complex holds
// exactly one compound with no combinators, and nil otherwise. For internal
// use by superselector and unification checks.
func (c *ComplexSelector) SingleCompound() *CompoundSelector {
	if len(c.LeadingCombinators) > 0 {
		return nil
	}
	if len(c.Components) == 1 {
		comp := c.Components[0]
		if len(comp.Combinators) == 0 {
			return comp.Selector
		}
	}
	return nil
}

// AcceptVoid dispatches to VisitComplexSelector on v.
func (c *ComplexSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitComplexSelector(c)
}

// AcceptBool dispatches to VisitComplexSelector on v.
func (c *ComplexSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitComplexSelector(c)
}

// AcceptParentSelector dispatches to VisitComplexSelector on v.
func (c *ComplexSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitComplexSelector(c)
}

// IsSuperselector reports whether this complex matches every element that
// other matches, possibly plus more. Complexes with leading combinators
// never qualify; the component comparison itself lives in selector_extend.go
// (V4a).
func (c *ComplexSelector) IsSuperselector(other *ComplexSelector) (bool, error) {
	if len(c.LeadingCombinators) != 0 || len(other.LeadingCombinators) != 0 {
		return false, nil
	}
	return complexIsSuperselector(c.Components, other.Components)
}

// WithAdditionalCombinators returns a copy of this complex with combinators
// appended to its final component, or to the leading combinators when there
// are no components. When forceLineBreak is set the copy is marked for a
// line break. An empty addition returns the complex unchanged. For internal
// use by nesting resolution.
func (c *ComplexSelector) WithAdditionalCombinators(
	combinators []sasscommon.CssValue[Combinator],
	forceLineBreak bool,
) *ComplexSelector {
	if len(combinators) == 0 {
		return c
	}
	if len(c.Components) > 0 {
		last := c.Components[len(c.Components)-1]
		newLast := last.WithAdditionalCombinators(combinators)
		newComponents := make([]*ComplexSelectorComponent, len(c.Components))
		copy(newComponents, c.Components)
		newComponents[len(newComponents)-1] = newLast
		return &ComplexSelector{
			base:               c.base,
			LeadingCombinators: c.LeadingCombinators,
			Components:         newComponents,
			LineBreak:          c.LineBreak || forceLineBreak,
		}
	}
	newLC := make([]sasscommon.CssValue[Combinator], 0, len(c.LeadingCombinators)+len(combinators))
	newLC = append(newLC, c.LeadingCombinators...)
	newLC = append(newLC, combinators...)
	return &ComplexSelector{
		base:               c.base,
		LeadingCombinators: newLC,
		Components:         []*ComplexSelectorComponent{},
		LineBreak:          c.LineBreak || forceLineBreak,
	}
}

// WithAdditionalComponent returns a copy of this complex with component
// appended, re-spanned to span. When forceLineBreak is set the copy is
// marked for a line break. For internal use while resolving nesting.
func (c *ComplexSelector) WithAdditionalComponent(
	component *ComplexSelectorComponent,
	span sasscommon.FileSpan,
	forceLineBreak bool,
) *ComplexSelector {
	newComponents := make([]*ComplexSelectorComponent, len(c.Components)+1)
	copy(newComponents, c.Components)
	newComponents[len(newComponents)-1] = component
	return &ComplexSelector{
		base:               NewSelectorBase(span),
		LeadingCombinators: c.LeadingCombinators,
		Components:         newComponents,
		LineBreak:          c.LineBreak || forceLineBreak,
	}
}

// Concatenate returns a copy of this complex with child's combinators and
// components appended. A child's leading combinators fold into this
// complex's last component; this never resolves parent selectors. The copy
// is re-spanned to span, and forceLineBreak marks it for a line break. For
// internal use by implicit-parent nesting.
func (c *ComplexSelector) Concatenate(
	child *ComplexSelector,
	span sasscommon.FileSpan,
	forceLineBreak bool,
) *ComplexSelector {
	if len(child.LeadingCombinators) == 0 {
		newComponents := make([]*ComplexSelectorComponent, len(c.Components)+len(child.Components))
		copy(newComponents, c.Components)
		copy(newComponents[len(c.Components):], child.Components)
		return &ComplexSelector{
			base:               NewSelectorBase(span),
			LeadingCombinators: c.LeadingCombinators,
			Components:         newComponents,
			LineBreak:          c.LineBreak || child.LineBreak || forceLineBreak,
		}
	}
	if len(c.Components) > 0 {
		last := c.Components[len(c.Components)-1]
		newLast := last.WithAdditionalCombinators(child.LeadingCombinators)
		newComponents := make([]*ComplexSelectorComponent, len(c.Components)+len(child.Components))
		copy(newComponents, c.Components)
		copy(newComponents[len(c.Components):], child.Components)
		newComponents[len(newComponents)-len(child.Components)-1] = newLast
		return &ComplexSelector{
			base:               NewSelectorBase(span),
			LeadingCombinators: c.LeadingCombinators,
			Components:         newComponents,
			LineBreak:          c.LineBreak || child.LineBreak || forceLineBreak,
		}
	}
	newLC := make([]sasscommon.CssValue[Combinator], len(c.LeadingCombinators)+len(child.LeadingCombinators))
	copy(newLC, c.LeadingCombinators)
	copy(newLC[len(c.LeadingCombinators):], child.LeadingCombinators)
	newComponents := make([]*ComplexSelectorComponent, len(child.Components))
	copy(newComponents, child.Components)
	return &ComplexSelector{
		base:               NewSelectorBase(span),
		LeadingCombinators: newLC,
		Components:         newComponents,
		LineBreak:          c.LineBreak || child.LineBreak || forceLineBreak,
	}
}

// HashCode folds the leading combinators and components together. Spans
// never contribute to selector hashes.
func (c *ComplexSelector) HashCode() int {
	h := 0
	for _, lc := range c.LeadingCombinators {
		h = hashCombine(h, int(lc.Value)*31)
	}
	for _, comp := range c.Components {
		h = hashCombine(h, comp.HashCode())
	}
	return h
}

// Equals reports whether two complex selectors are structurally equal:
// same leading combinators and same components, with spans ignored.
func (c *ComplexSelector) Equals(other any) bool {
	o, ok := other.(*ComplexSelector)
	if !ok {
		return false
	}
	if c == o {
		return true
	}
	if c == nil || o == nil {
		return false
	}
	if len(c.LeadingCombinators) != len(o.LeadingCombinators) {
		return false
	}
	for i := range c.LeadingCombinators {
		if !c.LeadingCombinators[i].Equal(o.LeadingCombinators[i]) {
			return false
		}
	}
	if len(c.Components) != len(o.Components) {
		return false
	}
	for i := range c.Components {
		if !c.Components[i].Equal(o.Components[i]) {
			return false
		}
	}
	return true
}

// String renders this selector in inspect mode.
func (c *ComplexSelector) String() (string, error) {
	return SerializeSelector(c, true)
}

// complexIsSuperselector forwards the component-level comparison to the
// extend-algorithm implementation in selector_extend.go (V4a).
func complexIsSuperselector(a, b []*ComplexSelectorComponent) (bool, error) {
	return ComplexIsSuperselector(a, b)
}
