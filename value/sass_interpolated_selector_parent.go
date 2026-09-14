// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/parent.dart

// InterpolatedParentSelector is a parent (&) selector with an optional
// interpolated suffix.
//
// It is parsed during the initial stylesheet pass and resolved once the
// parent selector and any suffix interpolation are known.
type InterpolatedParentSelector struct {
	// Suffix is appended to the resolved parent selector, or nil for a bare &.
	Suffix *Interpolation
	span   sasscommon.FileSpan
}

// NewInterpolatedParentSelector creates a parent selector with an optional
// suffix.
func NewInterpolatedParentSelector(span sasscommon.FileSpan, suffix *Interpolation) *InterpolatedParentSelector {
	return &InterpolatedParentSelector{Suffix: suffix, span: span}
}

func (s *InterpolatedParentSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedParentSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedParentSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedParentSelector) IsAstNode()                         {}
func (s *InterpolatedParentSelector) IsSassNode()                        {}
func (s *InterpolatedParentSelector) String() (string, error) {
	if s.Suffix != nil {
		suffixStr, err := s.Suffix.String()
		if err != nil {
			return "", err
		}
		return "&" + suffixStr, nil
	}
	return "&", nil
}
