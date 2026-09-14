// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/universal.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedUniversalSelector is a universal selector with an optional
// interpolated namespace.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// universal selector once interpolation is evaluated.
type InterpolatedUniversalSelector struct {
	// Namespace restricts the universal match, or nil for every namespace.
	Namespace *Interpolation
	span      sasscommon.FileSpan
}

// NewInterpolatedUniversalSelector creates a universal selector with an
// optional namespace.
func NewInterpolatedUniversalSelector(span sasscommon.FileSpan, namespace *Interpolation) *InterpolatedUniversalSelector {
	return &InterpolatedUniversalSelector{Namespace: namespace, span: span}
}

func (s *InterpolatedUniversalSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedUniversalSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedUniversalSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedUniversalSelector) IsAstNode()                         {}
func (s *InterpolatedUniversalSelector) IsSassNode()                        {}

func (s *InterpolatedUniversalSelector) String() (string, error) {
	if s.Namespace != nil {
		nsStr, err := s.Namespace.String()
		if err != nil {
			return "", err
		}
		return nsStr + "|*", nil
	}
	return "*", nil
}
