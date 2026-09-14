// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/type.dart

// TypeSelector selects elements whose name equals the given name.
type TypeSelector struct {
	base SelectorBase
	// Name is the element name being selected, with its namespace.
	Name QualifiedName
}

// NewTypeSelector creates a type selector for the given qualified name.
func NewTypeSelector(name QualifiedName, span sasscommon.FileSpan) *TypeSelector {
	return &TypeSelector{
		base: NewSelectorBase(span),
		Name: name,
	}
}

// Span returns the source span where this selector was written.
func (s *TypeSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *TypeSelector) IsAstNode()                                 {}
func (s *TypeSelector) IsSelector()                                {}
func (s *TypeSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *TypeSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *TypeSelector) IsBogus() bool                              { return false }
func (s *TypeSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *TypeSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// A type selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *TypeSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the type-selector specificity of 1.
func (s *TypeSelector) Specificity() int { return 1 }

func (s *TypeSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitTypeSelector on v.
func (s *TypeSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitTypeSelector(s)
}

// AcceptBool dispatches to VisitTypeSelector on v.
func (s *TypeSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitTypeSelector(s)
}

// AcceptParentSelector dispatches to VisitTypeSelector on v.
func (s *TypeSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitTypeSelector(s)
}

func (s *TypeSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix returns a copy of this selector as though suffix had been written
// at the end of the element name, preserving the namespace. The suffix is
// assumed to be a valid identifier suffix.
func (s *TypeSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewTypeSelector(QualifiedName{Name: s.Name.Name + suffix, Namespace: s.Name.Namespace}, span), nil
}

// String renders this selector in inspect mode.
func (s *TypeSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode folds the element name and namespace together. Spans never
// contribute to selector hashes.
func (s *TypeSelector) HashCode() int {
	return hashCombine(stringHashCode(s.Name.Name), hashPtr(s.Name.Namespace))
}

// IsSuperselector reports whether this type is a superselector of other.
// Beyond the shared base-class check, a type matches another type with the
// same element name when this namespace is `*` (any namespace) or equals the
// other's namespace. Unification against a leading universal or type folds
// the two together; see Unify in selector_extend_functions.go (V4a).
func (s *TypeSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	if ok, err := simpleIsSuperselector(s, other); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	o, ok := other.(*TypeSelector)
	if !ok {
		return false, nil
	}
	return s.Name.Name == o.Name.Name &&
		((s.Name.Namespace != nil && *s.Name.Namespace == "*") || ptrEqStr(s.Name.Namespace, o.Name.Namespace)), nil
}
