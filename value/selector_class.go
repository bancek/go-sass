// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/class.dart

// ClassSelector selects elements whose class attribute contains an identifier
// with the given name.
type ClassSelector struct {
	base SelectorBase
	// Name is the class name being selected for, without the leading dot.
	Name string
}

// NewClassSelector creates a class selector matching Name.
func NewClassSelector(name string, span sasscommon.FileSpan) *ClassSelector {
	return &ClassSelector{
		base: NewSelectorBase(span),
		Name: name,
	}
}

// Span returns the source span where this selector was written.
func (s *ClassSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *ClassSelector) IsAstNode()                                 {}
func (s *ClassSelector) IsSelector()                                {}
func (s *ClassSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *ClassSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *ClassSelector) IsBogus() bool                              { return false }
func (s *ClassSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *ClassSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// A class selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *ClassSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the default simple-selector specificity of 1000.
func (s *ClassSelector) Specificity() int { return s.base.Specificity() }

func (s *ClassSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitClassSelector on v.
func (s *ClassSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitClassSelector(s)
}

// AcceptBool dispatches to VisitClassSelector on v.
func (s *ClassSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitClassSelector(s)
}

// AcceptParentSelector dispatches to VisitClassSelector on v.
func (s *ClassSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitClassSelector(s)
}

func (s *ClassSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix returns a copy of this selector as though suffix had been written
// at the end of the class name. The suffix is assumed to be a valid
// identifier suffix.
func (s *ClassSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewClassSelector(s.Name+suffix, span), nil
}

// String renders this selector in inspect mode.
func (s *ClassSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode hashes the class name. Spans never contribute to selector hashes.
func (s *ClassSelector) HashCode() int {
	return stringHashCode(s.Name)
}

// IsSuperselector reports whether this class is a superselector of other:
// same-named classes match each other, and the shared base-class check
// covers subselector pseudos wrapping this class.
func (s *ClassSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	if o, ok := other.(*ClassSelector); ok {
		return s.Name == o.Name, nil
	}
	return simpleIsSuperselector(s, other)
}
