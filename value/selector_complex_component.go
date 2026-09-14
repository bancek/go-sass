// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/complex_component.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// A component of a ComplexSelector.
//
// This is a CompoundSelector with one or more trailing Combinators.
type ComplexSelectorComponent struct {
	// Selector is this component's compound selector.
	Selector *CompoundSelector
	// Combinators are the combinators following this component. Empty means
	// an implicit descendant combinator; more than one is invalid CSS but
	// still supported for backwards compatibility.
	Combinators []sasscommon.CssValue[Combinator]
	// Span is the source span covering this component.
	Span sasscommon.FileSpan
}

// NewComplexSelectorComponent creates a component pairing selector with its
// trailing combinators. The combinators are copied to keep the component
// immutable.
func NewComplexSelectorComponent(
	selector *CompoundSelector,
	combinators []sasscommon.CssValue[Combinator],
	span sasscommon.FileSpan,
) *ComplexSelectorComponent {
	// Make a copy to ensure immutability
	c := make([]sasscommon.CssValue[Combinator], len(combinators))
	copy(c, combinators)
	return &ComplexSelectorComponent{
		Selector:    selector,
		Combinators: c,
		Span:        span,
	}
}

// HashCode folds the compound selector and trailing combinators together.
// Spans never contribute to selector hashes.
func (c *ComplexSelectorComponent) HashCode() int {
	h := c.Selector.HashCode()
	for _, comb := range c.Combinators {
		h = hashCombine(h, int(comb.Value)*31)
	}
	return h
}

// Equal reports whether this component matches other: combinators compared
// by value, compound selectors by structural equality, spans ignored.
func (c *ComplexSelectorComponent) Equal(other *ComplexSelectorComponent) bool {
	if c == other {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if len(c.Combinators) != len(other.Combinators) {
		return false
	}
	for i := range c.Combinators {
		if !c.Combinators[i].Equal(other.Combinators[i]) {
			return false
		}
	}
	return EqualSelectors(c.Selector, other.Selector)
}

// WithAdditionalCombinators returns a copy of this component with combinators
// appended to its trailing combinators. For internal use by nesting and
// extension; an empty addition returns the component unchanged.
func (c *ComplexSelectorComponent) WithAdditionalCombinators(
	combinators []sasscommon.CssValue[Combinator],
) *ComplexSelectorComponent {
	if len(combinators) == 0 {
		return c
	}
	newCombinators := make([]sasscommon.CssValue[Combinator], 0, len(c.Combinators)+len(combinators))
	newCombinators = append(newCombinators, c.Combinators...)
	newCombinators = append(newCombinators, combinators...)
	return &ComplexSelectorComponent{
		Selector:    c.Selector,
		Combinators: newCombinators,
		Span:        c.Span,
	}
}
