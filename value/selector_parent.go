// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/parent.dart

// ParentSelector matches the parent in the Sass stylesheet.
//
// This is not a plain CSS selector—it should be removed before emitting a CSS
// document.
type ParentSelector struct {
	base SelectorBase
	// Suffix is text appended to the resolved parent selector, as in `&-suffix`.
	// It is assumed to be a valid identifier suffix, and nil when the parent
	// resolves unmodified.
	Suffix *string
}

// NewParentSelector creates a parent selector (`&`), with an optional suffix
// to append after resolution.
func NewParentSelector(span sasscommon.FileSpan, suffix *string) *ParentSelector {
	return &ParentSelector{
		base:   NewSelectorBase(span),
		Suffix: suffix,
	}
}

// Span returns the source span where this selector was written.
func (s *ParentSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *ParentSelector) IsAstNode()                                 {}
func (s *ParentSelector) IsSelector()                                {}
func (s *ParentSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *ParentSelector) IsBogus() bool                              { return false }
func (s *ParentSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *ParentSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// A parent selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *ParentSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the default simple-selector specificity of 1000. The
// resolved parent inherits its real specificity later; the placeholder value
// only matters before nesting resolution.
func (s *ParentSelector) Specificity() int { return s.base.Specificity() }

func (s *ParentSelector) IsSimpleSelector() {}

// ContainsParentSelector always reports true: this node is the marker that
// nesting resolution looks for.
func (s *ParentSelector) ContainsParentSelector() (bool, error) { return true, nil }

// AcceptVoid dispatches to VisitParentSelector on v.
func (s *ParentSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitParentSelector(s)
}

// AcceptBool dispatches to VisitParentSelector on v.
func (s *ParentSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitParentSelector(s)
}

// AcceptParentSelector dispatches to VisitParentSelector on v.
func (s *ParentSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitParentSelector(s)
}

func (s *ParentSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix always fails: the parent selector already carries its own Suffix
// field, so a further suffix is rejected as a script error.
func (s *ParentSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	return nil, &sasscommon.SassScriptException{Message: "parent selector cannot have a suffix"}
}

// String renders this selector in inspect mode.
func (s *ParentSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode hashes the optional suffix. A nil suffix hashes distinctly from
// any concrete suffix.
func (s *ParentSelector) HashCode() int {
	return hashPtr(s.Suffix)
}

// IsSuperselector delegates to the shared base-class check. Parent selectors
// never unify; see Unify in selector_extend_functions.go (V4a), which rejects
// them outright.
func (s *ParentSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	return simpleIsSuperselector(s, other)
}
