// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/placeholder.dart

// InterpolatedPlaceholderSelector is a placeholder selector whose name may
// still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// placeholder selector once interpolation is evaluated.
type InterpolatedPlaceholderSelector struct {
	// Name is the interpolated placeholder name without the leading percent.
	Name *Interpolation
	span sasscommon.FileSpan
}

// NewInterpolatedPlaceholderSelector creates a placeholder selector for name,
// widening the span by one character to cover the leading percent sign.
func NewInterpolatedPlaceholderSelector(name *Interpolation) (*InterpolatedPlaceholderSelector, error) {
	s, err := name.Span()
	if err != nil {
		return nil, err
	}
	startLoc, err := s.StartLocation()
	if err != nil {
		return nil, err
	}
	startLoc.Offset--
	endLoc, err := s.EndLocation()
	if err != nil {
		return nil, err
	}
	file, err := s.File()
	if err != nil {
		return nil, err
	}
	return &InterpolatedPlaceholderSelector{Name: name, span: sasscommon.NewSimpleFileSpan(file, startLoc.Offset, endLoc.Offset)}, nil
}

func (s *InterpolatedPlaceholderSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedPlaceholderSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedPlaceholderSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedPlaceholderSelector) IsAstNode()                         {}
func (s *InterpolatedPlaceholderSelector) IsSassNode()                        {}
func (s *InterpolatedPlaceholderSelector) String() (string, error) {
	nameStr, err := s.Name.String()
	if err != nil {
		return "", err
	}
	return "%" + nameStr, nil
}
