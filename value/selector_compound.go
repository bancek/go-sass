// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/compound.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// A compound selector.
//
// A compound selector is composed of SimpleSelectors. It matches an element
// that matches all of the component simple selectors.
type CompoundSelector struct {
	base SelectorBase
	// Components are the simple selectors that must all match. Never empty.
	Components []SimpleSelector
}

// NewCompoundSelector creates a compound selector matching elements that
// match every component. It returns an error when components is empty.
func NewCompoundSelector(components []SimpleSelector, span sasscommon.FileSpan) (*CompoundSelector, error) {
	if len(components) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "components may not be empty"}
	}
	c := make([]SimpleSelector, len(components))
	copy(c, components)
	return &CompoundSelector{
		base:       NewSelectorBase(span),
		Components: c,
	}, nil
}

// Span returns the source span where this selector was written.
func (c *CompoundSelector) Span() (sasscommon.FileSpan, error) { return c.base.Span() }
func (c *CompoundSelector) IsAstNode()                         {}
func (c *CompoundSelector) IsSelector()                        {}

// Specificity returns the sum of the component specificities. Specificity is
// measured in base 1000, high enough that no single selector sequence
// plausibly reaches 1000 simple selectors.
func (c *CompoundSelector) Specificity() int {
	sum := 0
	for _, component := range c.Components {
		sum += component.Specificity()
	}
	return sum
}

// SingleSimple returns the sole simple selector when this compound holds
// exactly one, and nil otherwise. For internal use by superselector and
// unification checks.
func (c *CompoundSelector) SingleSimple() SimpleSelector {
	if len(c.Components) == 1 {
		return c.Components[0]
	}
	return nil
}

// HasComplicatedSuperselectorSemantics reports whether any component needs
// complex non-local reasoning for super- and sub-selector checks: that is,
// pseudo-elements and pseudos carrying nested selectors. For internal use by
// the superselector fast path.
func (c *CompoundSelector) HasComplicatedSuperselectorSemantics() bool {
	for _, component := range c.Components {
		if component.HasComplicatedSuperselectorSemantics() {
			return true
		}
	}
	return false
}

// AcceptVoid dispatches to VisitCompoundSelector on v.
func (c *CompoundSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitCompoundSelector(c)
}

// AcceptBool dispatches to VisitCompoundSelector on v.
func (c *CompoundSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitCompoundSelector(c)
}

// AcceptParentSelector dispatches to VisitCompoundSelector on v.
func (c *CompoundSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitCompoundSelector(c)
}

// IsSuperselector reports whether this compound matches every element that
// other matches, possibly plus more. The comparison itself lives in
// selector_extend.go (V4a), which handles pseudo-class strategies.
func (c *CompoundSelector) IsSuperselector(other *CompoundSelector) (bool, error) {
	return compoundIsSuperselector(c, other)
}

// HashCode folds the component hashes together. Spans never contribute to
// selector hashes.
func (c *CompoundSelector) HashCode() int {
	h := 0
	for _, comp := range c.Components {
		h = hashCombine(h, comp.HashCode())
	}
	return h
}

// ContainsParentSelector reports whether any component is or contains a
// parent selector.
func (c *CompoundSelector) ContainsParentSelector() (bool, error) {
	for _, s := range c.Components {
		found, err := s.ContainsParentSelector()
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

func (c *CompoundSelector) IsBogusOtherThanLeadingCombinator() bool {
	return c.IsBogus()
}

func (c *CompoundSelector) IsInvisibleOtherThanBogusCombinators() bool {
	for _, comp := range c.Components {
		if comp.IsInvisibleOtherThanBogusCombinators() {
			return true
		}
	}
	return false
}

// AssertNotBogus warns through warn when this selector is not valid CSS,
// forwarding to the shared helper.
func (c *CompoundSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(c, name, warn)
}

// String renders this selector in inspect mode.
func (c *CompoundSelector) String() (string, error) {
	return SerializeSelector(c, true)
}

// compoundIsSuperselector forwards the compound comparison to the
// extend-algorithm implementation in selector_extend.go (V4a).
func compoundIsSuperselector(a, b *CompoundSelector) (bool, error) {
	return CompoundIsSuperselector(a, b)
}
