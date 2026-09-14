// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/qualified_name.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedQualifiedName is a possibly namespaced selector name whose
// parts may still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved along with
// its enclosing selector.
type InterpolatedQualifiedName struct {
	// Name is the interpolated identifier.
	Name *Interpolation
	span sasscommon.FileSpan
	// Namespace qualifies Name (elem|name), or nil when unqualified.
	Namespace *Interpolation
}

// NewInterpolatedQualifiedName creates a qualified name with an optional
// namespace.
func NewInterpolatedQualifiedName(name *Interpolation, span sasscommon.FileSpan, namespace *Interpolation) *InterpolatedQualifiedName {
	return &InterpolatedQualifiedName{Name: name, span: span, Namespace: namespace}
}

func (s *InterpolatedQualifiedName) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedQualifiedName) IsAstNode()                         {}
func (s *InterpolatedQualifiedName) IsSassNode()                        {}

func (s *InterpolatedQualifiedName) String() (string, error) {
	if s.Namespace != nil {
		nsStr, err := s.Namespace.String()
		if err != nil {
			return "", err
		}
		nameStr, err := s.Name.String()
		if err != nil {
			return "", err
		}
		return nsStr + "|" + nameStr, nil
	}
	return s.Name.String()
}
