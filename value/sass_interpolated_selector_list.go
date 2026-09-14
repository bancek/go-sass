// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/list.dart

import (
	"errors"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedSelectorList is a comma-separated selector list whose members
// may still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// selector list once interpolation is evaluated.
type InterpolatedSelectorList struct {
	// Components are the complex selectors joined by commas. Never empty.
	Components []*InterpolatedComplexSelector
	span       sasscommon.FileSpan
}

// NewInterpolatedSelectorList creates a selector list from components.
// It reports an error when components is empty.
func NewInterpolatedSelectorList(components []*InterpolatedComplexSelector, span sasscommon.FileSpan) (*InterpolatedSelectorList, error) {
	if len(components) == 0 {
		return nil, errors.New("components may not be empty.")
	}
	c := make([]*InterpolatedComplexSelector, len(components))
	copy(c, components)
	return &InterpolatedSelectorList{Components: c, span: span}, nil
}

func (t *InterpolatedSelectorList) Span() (sasscommon.FileSpan, error) { return t.span, nil }

func (s *InterpolatedSelectorList) IsInterpolatedSelector() {}
func (s *InterpolatedSelectorList) IsAstNode()              {}
func (s *InterpolatedSelectorList) IsSassNode()             {}

func (s *InterpolatedSelectorList) String() (string, error) {
	var parts []string
	for _, c := range s.Components {
		s, err := c.String()
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", "), nil
}
