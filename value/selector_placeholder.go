// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/placeholder.dart

// PlaceholderSelector doesn't match any elements. It's intended to be extended
// using @extend. It's not a plain CSS selector—it should be removed before
// emitting a CSS document.
type PlaceholderSelector struct {
	base SelectorBase
	// Name is the placeholder name, without the leading percent sign.
	Name string
}

// NewPlaceholderSelector creates a placeholder selector for Name.
func NewPlaceholderSelector(name string, span sasscommon.FileSpan) *PlaceholderSelector {
	return &PlaceholderSelector{
		base: NewSelectorBase(span),
		Name: name,
	}
}

// Span returns the source span where this selector was written.
func (s *PlaceholderSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *PlaceholderSelector) IsAstNode()                                 {}
func (s *PlaceholderSelector) IsSelector()                                {}
func (s *PlaceholderSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *PlaceholderSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *PlaceholderSelector) IsBogus() bool                              { return false }
func (s *PlaceholderSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *PlaceholderSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// A placeholder selector is never bogus, so this only forwards to the
// shared helper for uniformity.
func (s *PlaceholderSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the default simple-selector specificity of 1000.
func (s *PlaceholderSelector) Specificity() int { return s.base.Specificity() }

func (s *PlaceholderSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitPlaceholderSelector on v.
func (s *PlaceholderSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitPlaceholderSelector(s)
}

// AcceptBool dispatches to VisitPlaceholderSelector on v.
func (s *PlaceholderSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitPlaceholderSelector(s)
}

// AcceptParentSelector dispatches to VisitPlaceholderSelector on v.
func (s *PlaceholderSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitPlaceholderSelector(s)
}

func (s *PlaceholderSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix returns a copy of this selector as though suffix had been written
// at the end of the placeholder name. The suffix is assumed to be a valid
// identifier suffix.
func (s *PlaceholderSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewPlaceholderSelector(s.Name+suffix, span), nil
}

// String renders this selector in inspect mode.
func (s *PlaceholderSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode hashes the placeholder name. Spans never contribute to selector
// hashes.
func (s *PlaceholderSelector) HashCode() int {
	return stringHashCode(s.Name)
}

// IsSuperselector reports whether this placeholder is a superselector of
// other: same-named placeholders match each other, and the shared
// base-class check covers subselector pseudos wrapping this placeholder.
func (s *PlaceholderSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	if o, ok := other.(*PlaceholderSelector); ok {
		return s.Name == o.Name, nil
	}
	return simpleIsSuperselector(s, other)
}
