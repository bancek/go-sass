// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/id.dart

// InterpolatedIDSelector is an ID selector whose name may still contain
// interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// ID selector once interpolation is evaluated.
type InterpolatedIDSelector struct {
	// Name is the interpolated ID without the leading hash.
	Name *Interpolation
	span sasscommon.FileSpan
}

// NewInterpolatedIDSelector creates an ID selector for name, widening the
// span by one character to cover the leading hash.
func NewInterpolatedIDSelector(name *Interpolation) (*InterpolatedIDSelector, error) {
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
	return &InterpolatedIDSelector{Name: name, span: sasscommon.NewSimpleFileSpan(file, startLoc.Offset, endLoc.Offset)}, nil
}

func (s *InterpolatedIDSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedIDSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedIDSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedIDSelector) IsAstNode()                         {}
func (s *InterpolatedIDSelector) IsSassNode()                        {}
func (s *InterpolatedIDSelector) String() (string, error) {
	nameStr, err := s.Name.String()
	if err != nil {
		return "", err
	}
	return "#" + nameStr, nil
}
