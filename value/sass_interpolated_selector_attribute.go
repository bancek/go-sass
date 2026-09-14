// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/attribute.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedAttributeSelector is an attribute selector whose name, value,
// or modifier may still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// attribute selector once interpolation is evaluated.
type InterpolatedAttributeSelector struct {
	// Name is the attribute being selected for.
	Name *InterpolatedQualifiedName
	// Op defines how Value is matched. Nil exactly when Value is nil.
	Op *sasscommon.CssValue[AttributeOperator]
	// Value asserts a value for Name. Nil exactly when Op is nil.
	Value *Interpolation
	// Modifier tunes matching (for example case-insensitivity).
	// Nil whenever Op is nil.
	Modifier *Interpolation
	span     sasscommon.FileSpan
}

// NewInterpolatedAttributeSelector creates a bare [name] selector matching
// any element carrying that attribute.
func NewInterpolatedAttributeSelector(name *InterpolatedQualifiedName, span sasscommon.FileSpan) *InterpolatedAttributeSelector {
	return &InterpolatedAttributeSelector{Name: name, span: span}
}

// NewInterpolatedAttributeSelectorWithOperator creates a selector matching
// elements whose Name relates to value as Op dictates, with an optional
// case modifier.
func NewInterpolatedAttributeSelectorWithOperator(
	name *InterpolatedQualifiedName,
	op sasscommon.CssValue[AttributeOperator],
	value *Interpolation,
	span sasscommon.FileSpan,
	modifier *Interpolation,
) *InterpolatedAttributeSelector {
	return &InterpolatedAttributeSelector{
		Name:     name,
		Op:       &op,
		Value:    value,
		Modifier: modifier,
		span:     span,
	}
}

func (s *InterpolatedAttributeSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedAttributeSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedAttributeSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedAttributeSelector) IsAstNode()                         {}
func (s *InterpolatedAttributeSelector) IsSassNode()                        {}
func (s *InterpolatedAttributeSelector) String() (string, error) {
	nameStr, err := s.Name.String()
	if err != nil {
		return "", err
	}
	result := "[" + nameStr
	if s.Op != nil {
		valStr, err := s.Value.String()
		if err != nil {
			return "", err
		}
		result += s.Op.String() + valStr
		if s.Modifier != nil {
			modStr, err := s.Modifier.String()
			if err != nil {
				return "", err
			}
			result += " " + modStr
		}
	}
	return result + "]", nil
}
