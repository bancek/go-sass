// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/interpolated_selector/pseudo.dart

// InterpolatedPseudoSelector is a pseudo-class or pseudo-element selector
// whose name or arguments may still contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// pseudo selector once interpolation is evaluated.
type InterpolatedPseudoSelector struct {
	// Name is the interpolated pseudo name with any vendor prefix.
	Name *Interpolation
	// IsSyntacticClass selects single-colon (true) versus double-colon
	// pseudo-element (false) syntax.
	IsSyntacticClass bool
	// Argument is the non-selector argument, or nil when absent.
	// When both Argument and Selector are set, the selector follows it.
	Argument *Interpolation
	// Selector is the nested selector argument, or nil when absent.
	// When both Argument and Selector are set, the selector follows it.
	Selector *InterpolatedSelectorList
	span     sasscommon.FileSpan
}

// NewInterpolatedPseudoSelector creates a pseudo selector for name.
// Element selects double-colon pseudo-element syntax over single-colon
// pseudo-class syntax; argument and selector hold the optional parenthesized
// payload.
func NewInterpolatedPseudoSelector(
	name *Interpolation,
	span sasscommon.FileSpan,
	element bool,
	argument *Interpolation,
	selector *InterpolatedSelectorList,
) *InterpolatedPseudoSelector {
	return &InterpolatedPseudoSelector{
		Name:             name,
		span:             span,
		IsSyntacticClass: !element,
		Argument:         argument,
		Selector:         selector,
	}
}

func (s *InterpolatedPseudoSelector) Span() (sasscommon.FileSpan, error) { return s.span, nil }
func (s *InterpolatedPseudoSelector) IsInterpolatedSelector()            {}
func (s *InterpolatedPseudoSelector) IsInterpolatedSimpleSelector()      {}
func (s *InterpolatedPseudoSelector) IsAstNode()                         {}
func (s *InterpolatedPseudoSelector) IsSassNode()                        {}

// IsSyntacticElement returns true if this is syntactically a pseudo-element.
//
// Matches Dart: InterpolatedPseudoSelector.isSyntacticElement
func (s *InterpolatedPseudoSelector) IsSyntacticElement() bool {
	return !s.IsSyntacticClass
}

func (s *InterpolatedPseudoSelector) String() (string, error) {
	prefix := ":"
	if !s.IsSyntacticClass {
		prefix = "::"
	}
	nameStr, err := s.Name.String()
	if err != nil {
		return "", err
	}
	result := prefix + nameStr
	if s.Argument != nil || s.Selector != nil {
		result += "("
		if s.Argument != nil {
			argStr, err := s.Argument.String()
			if err != nil {
				return "", err
			}
			result += argStr
			if s.Selector != nil {
				result += " "
			}
		}
		if s.Selector != nil {
			selStr, err := s.Selector.String()
			if err != nil {
				return "", err
			}
			result += selStr
		}
		result += ")"
	}
	return result, nil
}
