// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/type.dart

// InterpolatedTypeSelector is a type selector whose qualified name may still
// contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// type selector once interpolation is evaluated.
type InterpolatedTypeSelector struct {
	// Name is the element name being selected for.
	Name *InterpolatedQualifiedName
}

// NewInterpolatedTypeSelector creates a type selector for name.
func NewInterpolatedTypeSelector(name *InterpolatedQualifiedName) *InterpolatedTypeSelector {
	return &InterpolatedTypeSelector{Name: name}
}

func (s *InterpolatedTypeSelector) Span() (sasscommon.FileSpan, error) { return s.Name.Span() }
func (s *InterpolatedTypeSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedTypeSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedTypeSelector) IsAstNode()                         {}
func (s *InterpolatedTypeSelector) IsSassNode()                        {}
func (s *InterpolatedTypeSelector) String() (string, error) {
	return s.Name.String()
}
