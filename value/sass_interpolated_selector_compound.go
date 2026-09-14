// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/compound.dart

import (
	"errors"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedCompoundSelector is a compound selector whose simple parts may
// still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// compound selector once interpolation is evaluated.
type InterpolatedCompoundSelector struct {
	// Components are the simple selectors joined without combinators.
	// Never empty.
	Components []InterpolatedSimpleSelector
}

// NewInterpolatedCompoundSelector creates a compound selector from components.
// It reports an error when components is empty.
func NewInterpolatedCompoundSelector(components []InterpolatedSimpleSelector) (*InterpolatedCompoundSelector, error) {
	if len(components) == 0 {
		return nil, errors.New("components may not be empty")
	}
	c := make([]InterpolatedSimpleSelector, len(components))
	copy(c, components)
	return &InterpolatedCompoundSelector{Components: c}, nil
}

func (s *InterpolatedCompoundSelector) Span() (sasscommon.FileSpan, error) {
	if len(s.Components) == 1 {
		return s.Components[0].Span()
	}
	first, err := s.Components[0].Span()
	if err != nil {
		return nil, err
	}
	last, err := s.Components[len(s.Components)-1].Span()
	if err != nil {
		return nil, err
	}
	firstFile, err := first.File()
	if err != nil {
		return nil, err
	}
	firstLoc, err := first.StartLocation()
	if err != nil {
		return nil, err
	}
	lastLoc, err := last.EndLocation()
	if err != nil {
		return nil, err
	}
	return sasscommon.NewSimpleFileSpan(firstFile, firstLoc.Offset, lastLoc.Offset), nil
}

func (s *InterpolatedCompoundSelector) IsInterpolatedSelector() {}
func (s *InterpolatedCompoundSelector) IsAstNode()              {}
func (s *InterpolatedCompoundSelector) IsSassNode()             {}

func (s *InterpolatedCompoundSelector) String() (string, error) {
	var parts []string
	for _, c := range s.Components {
		s, err := c.String()
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ""), nil
}
