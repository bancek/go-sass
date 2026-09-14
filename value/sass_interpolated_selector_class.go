// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/class.dart

// InterpolatedClassSelector is a class selector whose name may still contain
// interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// class selector once interpolation is evaluated.
type InterpolatedClassSelector struct {
	// Name is the interpolated class name without the leading dot.
	Name *Interpolation
	span sasscommon.FileSpan
}

// NewInterpolatedClassSelector creates a class selector for name, widening
// the span by one character to cover the leading dot.
func NewInterpolatedClassSelector(name *Interpolation) (*InterpolatedClassSelector, error) {
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
	return &InterpolatedClassSelector{Name: name, span: sasscommon.NewSimpleFileSpan(file, startLoc.Offset, endLoc.Offset)}, nil
}

func (s *InterpolatedClassSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedClassSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedClassSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedClassSelector) IsAstNode()                         {}
func (s *InterpolatedClassSelector) IsSassNode()                        {}
func (s *InterpolatedClassSelector) String() (string, error) {
	nameStr, err := s.Name.String()
	if err != nil {
		return "", err
	}
	return "." + nameStr, nil
}
