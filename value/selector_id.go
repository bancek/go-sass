// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/id.dart

// IDSelector selects elements whose id attribute exactly matches the given name.
type IDSelector struct {
	base SelectorBase
	// Name is the ID being selected for, without the leading hash.
	Name string
}

// NewIDSelector creates an ID selector matching Name.
func NewIDSelector(name string, span sasscommon.FileSpan) *IDSelector {
	return &IDSelector{
		base: NewSelectorBase(span),
		Name: name,
	}
}

// Span returns the source span where this selector was written.
func (s *IDSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *IDSelector) IsAstNode()                                 {}
func (s *IDSelector) IsSelector()                                {}
func (s *IDSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *IDSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *IDSelector) IsBogus() bool                              { return false }
func (s *IDSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *IDSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// An ID selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *IDSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the ID specificity: the default simple specificity of
// 1000 squared, or 1,000,000.
func (s *IDSelector) Specificity() int {
	base := s.base.Specificity()
	return base * base
}

func (s *IDSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitIDSelector on v.
func (s *IDSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitIDSelector(s)
}

// AcceptBool dispatches to VisitIDSelector on v.
func (s *IDSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) { return v.VisitIDSelector(s) }

// AcceptParentSelector dispatches to VisitIDSelector on v.
func (s *IDSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitIDSelector(s)
}

func (s *IDSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix returns a copy of this selector as though suffix had been written
// at the end of the ID. The suffix is assumed to be a valid identifier
// suffix.
func (s *IDSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewIDSelector(s.Name+suffix, span), nil
}

// String renders this selector in inspect mode.
func (s *IDSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode hashes the ID name. Spans never contribute to selector hashes.
func (s *IDSelector) HashCode() int {
	return stringHashCode(s.Name)
}

// IsSuperselector reports whether this ID is a superselector of other:
// same-named IDs match each other, and the shared base-class check covers
// subselector pseudos wrapping this ID. Unification against a compound with
// a second ID fails; see Unify in selector_extend_functions.go (V4a).
func (s *IDSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	if o, ok := other.(*IDSelector); ok {
		return s.Name == o.Name, nil
	}
	return simpleIsSuperselector(s, other)
}
