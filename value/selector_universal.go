// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// dart-source: lib/src/ast/selector/universal.dart

// UniversalSelector matches any element in the given namespace.
type UniversalSelector struct {
	base SelectorBase
	// Namespace selects which namespace to match. Nil matches the default
	// namespace, "" matches elements in no namespace, and "*" matches
	// elements in any namespace.
	Namespace *string
}

// NewUniversalSelector creates a universal selector (`*`) for the given
// namespace.
func NewUniversalSelector(span sasscommon.FileSpan, namespace *string) *UniversalSelector {
	return &UniversalSelector{
		base:      NewSelectorBase(span),
		Namespace: namespace,
	}
}

// Span returns the source span where this selector was written.
func (s *UniversalSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *UniversalSelector) IsAstNode()                                 {}
func (s *UniversalSelector) IsSelector()                                {}
func (s *UniversalSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *UniversalSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *UniversalSelector) IsBogus() bool                              { return false }
func (s *UniversalSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *UniversalSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// A universal selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *UniversalSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the universal-selector specificity of 0: it adds no
// weight to its compound.
func (s *UniversalSelector) Specificity() int { return 0 }

func (s *UniversalSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitUniversalSelector on v.
func (s *UniversalSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitUniversalSelector(s)
}

// AcceptBool dispatches to VisitUniversalSelector on v.
func (s *UniversalSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitUniversalSelector(s)
}

// AcceptParentSelector dispatches to VisitUniversalSelector on v.
func (s *UniversalSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitUniversalSelector(s)
}

func (s *UniversalSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix always fails: `*` cannot take an identifier suffix, so nesting
// resolution reports the outer selector's span.
func (s *UniversalSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	selStr, err := s.String()
	if err != nil {
		return nil, err
	}
	return nil, &sasscommon.MultiSpanSassException{
		Message:      fmt.Sprintf("Selector %q can't have a suffix", selStr),
		Span:         span,
		PrimaryLabel: "outer selector",
	}
}

// String renders this selector in inspect mode.
func (s *UniversalSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode hashes the namespace. Spans never contribute to selector hashes.
func (s *UniversalSelector) HashCode() int {
	return hashPtr(s.Namespace)
}

// IsSuperselector reports whether this universal is a superselector of other.
// `*` in any namespace matches everything; otherwise a universal matches type
// and universal selectors in the same namespace, and a default-namespace
// universal matches anything the shared base-class check accepts.
func (s *UniversalSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	if s.Namespace != nil && *s.Namespace == "*" {
		return true, nil
	}
	if o, ok := other.(*TypeSelector); ok {
		return ptrEq(s.Namespace, o.Name.Namespace), nil
	}
	if o, ok := other.(*UniversalSelector); ok {
		return ptrEq(s.Namespace, o.Namespace), nil
	}
	if s.Namespace == nil {
		return true, nil
	}
	return simpleIsSuperselector(s, other)
}
