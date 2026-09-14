// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/complex.dart

import (
	"errors"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedComplexSelector is a complex selector whose components may
// still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// complex selector once interpolation is evaluated.
type InterpolatedComplexSelector struct {
	// LeadingCombinator prefixes the selector, or nil for none.
	// Components may only be empty when a leading combinator is present.
	LeadingCombinator *sasscommon.CssValue[Combinator]
	// Components are the compound parts separated by combinators.
	Components []*InterpolatedComplexSelectorComponent
	span       sasscommon.FileSpan
}

// NewInterpolatedComplexSelector creates a complex selector from components.
// It reports an error when components is empty without a leading combinator.
func NewInterpolatedComplexSelector(
	components []*InterpolatedComplexSelectorComponent,
	span sasscommon.FileSpan,
	leadingCombinator *sasscommon.CssValue[Combinator],
) (*InterpolatedComplexSelector, error) {
	c := make([]*InterpolatedComplexSelectorComponent, len(components))
	copy(c, components)
	if leadingCombinator == nil && len(c) == 0 {
		return nil, errors.New("components may not be empty if leadingCombinator is null")
	}
	return &InterpolatedComplexSelector{
		LeadingCombinator: leadingCombinator,
		Components:        c,
		span:              span,
	}, nil
}

func (s *InterpolatedComplexSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedComplexSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedComplexSelector) IsAstNode()                         {}
func (s *InterpolatedComplexSelector) IsSassNode()                        {}

func (s *InterpolatedComplexSelector) String() (string, error) {
	var parts []string
	for _, c := range s.Components {
		s, err := c.String()
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " "), nil
}
