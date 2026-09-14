// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/complex_component.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedComplexSelectorComponent is one compound step of an
// InterpolatedComplexSelector, still carrying any interpolation.
//
// It is parsed during the initial stylesheet pass and resolved along with
// its parent complex selector.
type InterpolatedComplexSelectorComponent struct {
	// Selector is this step's compound selector.
	Selector *InterpolatedCompoundSelector
	// Combinator separates this step from the next.
	// Nil means the implicit descendant combinator.
	Combinator *sasscommon.CssValue[Combinator]
	span       sasscommon.FileSpan
}

// NewInterpolatedComplexSelectorComponent creates a complex-selector step
// pairing sel with its trailing combinator.
func NewInterpolatedComplexSelectorComponent(
	sel *InterpolatedCompoundSelector,
	span sasscommon.FileSpan,
	combinator *sasscommon.CssValue[Combinator],
) *InterpolatedComplexSelectorComponent {
	return &InterpolatedComplexSelectorComponent{
		Selector:   sel,
		span:       span,
		Combinator: combinator,
	}
}

func (s *InterpolatedComplexSelectorComponent) Span() (sasscommon.FileSpan, error) {
	return s.span, nil
}
func (s *InterpolatedComplexSelectorComponent) IsAstNode()  {}
func (s *InterpolatedComplexSelectorComponent) IsSassNode() {}

func (s *InterpolatedComplexSelectorComponent) String() (string, error) {
	selStr, err := s.Selector.String()
	if err != nil {
		return "", err
	}
	if s.Combinator != nil {
		return selStr + " " + s.Combinator.String(), nil
	}
	return selStr, nil
}
