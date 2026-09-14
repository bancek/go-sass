// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/pseudo.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/unvendor"
)

// PseudoSelector is a pseudo-class or pseudo-element selector.
//
// Which behavior a pseudo has depends on its name: some take arguments,
// including nested selectors, and Sass encodes per-name logic for those so
// that extension and other selector operations treat them correctly.
type PseudoSelector struct {
	base SelectorBase
	// Name is the pseudo name, without colons and with vendor prefixes intact.
	Name string
	// NormalizedName is Name with any vendor prefix stripped, used for
	// per-name logic such as specificity and superselector checks. For
	// internal use by selector operations.
	NormalizedName string
	// IsClass reports whether this is a pseudo-class. It is true exactly
	// when IsElement is false.
	IsClass bool
	// IsSyntacticClass records whether the pseudo was written with single
	// colon syntax. It matches IsClass except for pseudo-elements that allow
	// pseudo-class spelling (`before`, `after`, `first-line`, `first-letter`).
	IsSyntacticClass bool
	// Argument is the non-selector argument (such as an `An+B` expression).
	// Nil when there is no argument; when both Argument and Selector are set,
	// the selector follows the argument.
	Argument *string
	// Selector is the nested selector argument. Nil when the pseudo takes no
	// selector; when both Argument and Selector are set, the selector follows
	// the argument.
	Selector Selector
}

// NewPseudoSelector creates a pseudo-class or pseudo-element named name.
// element selects pseudo-element spelling; argument and selector carry the
// optional non-selector and selector arguments. Whether the pseudo counts as
// a class is derived from element and the fake-pseudo-element names below.
func NewPseudoSelector(
	name string,
	span sasscommon.FileSpan,
	element bool,
	argument *string,
	selector Selector,
) *PseudoSelector {
	isClass := !element && !isFakePseudoElement(name)
	return &PseudoSelector{
		base:             NewSelectorBase(span),
		Name:             name,
		NormalizedName:   unvendor.Unvendor(name),
		IsClass:          isClass,
		IsSyntacticClass: !element,
		Argument:         argument,
		Selector:         selector,
	}
}

// Span returns the source span where this selector was written.
func (s *PseudoSelector) Span() (sasscommon.FileSpan, error)      { return s.base.Span() }
func (s *PseudoSelector) IsAstNode()                              {}
func (s *PseudoSelector) IsSelector()                             {}
func (s *PseudoSelector) ContainsParentSelector() (bool, error)   { return false, nil }
func (s *PseudoSelector) IsBogusOtherThanLeadingCombinator() bool { return s.IsBogus() }

// AssertNotBogus warns through warn when this selector is not valid CSS,
// forwarding to the shared helper.
func (s *PseudoSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// IsElement reports whether this is a pseudo-element (exactly when IsClass
// is false).
func (s *PseudoSelector) IsElement() bool { return !s.IsClass }

// IsHost reports whether this is a valid `:host` selector: a pseudo-class
// literally named "host".
func (s *PseudoSelector) IsHost() bool {
	return s.IsClass && s.Name == "host"
}

// IsHostContext reports whether this is a valid `:host-context` selector: a
// pseudo-class named "host-context" carrying a nested selector.
func (s *PseudoSelector) IsHostContext() bool {
	return s.IsClass && s.Name == "host-context" && s.Selector != nil
}

// isFakePseudoElement reports whether name is a pseudo-element that may be
// spelled with pseudo-class syntax: `before`, `after`, `first-line`, or
// `first-letter`, matched case-insensitively on the first letter.
func isFakePseudoElement(name string) bool {
	if len(name) == 0 {
		return false
	}
	switch name[0] {
	case 'a', 'A':
		return strings.EqualFold(name, "after")
	case 'b', 'B':
		return strings.EqualFold(name, "before")
	case 'f', 'F':
		return strings.EqualFold(name, "first-line") ||
			strings.EqualFold(name, "first-letter")
	default:
		return false
	}
}

// Specificity follows https://drafts.csswg.org/selectors/#specificity-rules:
// pseudo-elements count 1; `:where` counts 0; `:is`, `:not`, `:has`, and
// `:matches` count the maximum of their arguments; `:nth-child` and
// `:nth-last-child` add that maximum to the default simple specificity; every
// other pseudo counts the default simple specificity of 1000.
func (s *PseudoSelector) Specificity() int {
	if s.IsElement() {
		return 1
	}
	sel := s.Selector
	if sel == nil {
		return s.base.Specificity()
	}
	selList, ok := sel.(*SelectorList)
	if !ok {
		return s.base.Specificity()
	}
	switch s.NormalizedName {
	case "where":
		return 0
	case "is", "not", "has", "matches":
		maxSp := 0
		for _, comp := range selList.Components {
			if sp := comp.Specificity(); sp > maxSp {
				maxSp = sp
			}
		}
		return maxSp
	case "nth-child", "nth-last-child":
		maxSp := 0
		for _, comp := range selList.Components {
			if sp := comp.Specificity(); sp > maxSp {
				maxSp = sp
			}
		}
		return s.base.Specificity() + maxSp
	default:
		return s.base.Specificity()
	}
}

func (s *PseudoSelector) IsSimpleSelector() {}

// AddSuffix returns a copy of this selector as though suffix had been written
// at the end of the pseudo name. Pseudos carrying an argument or a nested
// selector reject suffixes with a script error naming the outer selector;
// the suffix is assumed to be a valid identifier suffix otherwise.
func (s *PseudoSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	if s.Argument != nil || s.Selector != nil {
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
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewPseudoSelector(s.Name+suffix, span, s.IsElement(), nil, nil), nil
}

// IsInvisible reports whether this pseudo keeps its compound from being
// emitted. Only pseudos with nested selectors can be invisible; `:not` with
// a bogus argument is treated as matching everything (like `*`), while other
// pseudos are invisible exactly when their argument is.
func (s *PseudoSelector) IsInvisible() bool {
	if s.Selector == nil {
		return false
	}
	if s.Name == "not" {
		return s.Selector.IsBogus()
	}
	return s.Selector.IsInvisible()
}

// IsInvisibleOtherThanBogusCombinators reports whether this pseudo is
// invisible for reasons other than bogus combinators. `:not` never counts
// here (its bogus-argument exception from IsInvisible depends on bogus
// handling); other pseudos follow their nested selector.
func (s *PseudoSelector) IsInvisibleOtherThanBogusCombinators() bool {
	if s.Selector == nil {
		return false
	}
	if s.Name == "not" {
		return false
	}
	return s.Selector.IsInvisibleOtherThanBogusCombinators()
}

// IsUseless reports whether this selector is bogus and cannot be rescued by
// extension or nesting. For pseudos this is exactly bogusness.
func (s *PseudoSelector) IsUseless() bool {
	return s.IsBogus()
}

// HasComplicatedSuperselectorSemantics reports whether superselector checks
// against this pseudo need non-local reasoning: true for pseudo-elements and
// for pseudos carrying nested selectors.
func (s *PseudoSelector) HasComplicatedSuperselectorSemantics() bool {
	return s.IsElement() || s.Selector != nil
}

// IsBogus reports whether this pseudo is not valid CSS. Only pseudos with
// nested selectors can be bogus; `:has` tolerates leading combinators in its
// argument (which the CSS spec allows) while other pseudos propagate the
// argument's bogusness as-is.
func (s *PseudoSelector) IsBogus() bool {
	if s.Selector == nil {
		return false
	}
	if s.Name == "has" {
		return s.Selector.IsBogusOtherThanLeadingCombinator()
	}
	return s.Selector.IsBogus()
}

// String renders this selector in inspect mode.
func (s *PseudoSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// IsSuperselector reports whether this pseudo is a superselector of other.
// Argument-less pseudos only match structurally equal selectors; `::slotted`
// compares nested selector lists; other pseudos with arguments defer to the
// compound-level comparison in selector_extend.go (V4a), which knows how to
// compare selector pseudo-classes against plain selectors.
func (s *PseudoSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	// First apply the shared base-class check (equality plus subselector
	// pseudos wrapping this pseudo).
	if ok, err := simpleIsSuperselector(s, other); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	// An argument-less pseudo matches nothing beyond itself.
	if s.Selector == nil {
		return s.Equals(other), nil
	}

	// `::slotted` arguments compare as nested selector lists.
	if p, ok := other.(*PseudoSelector); ok && s.IsElement() && p.IsElement() &&
		s.NormalizedName == "slotted" && p.Name == s.Name {
		if p.Selector != nil {
			if selList, ok := s.Selector.(*SelectorList); ok {
				if otherList, ok := p.Selector.(*SelectorList); ok {
					result, err := selList.IsSuperselector(otherList)
					if err != nil {
						return false, err
					}
					return result, nil
				}
			}
		}
		return false, nil
	}

	// Fall back to CompoundSelector.isSuperselector, which knows how to
	// compare selector pseudo-classes against raw selectors.
	span, err := s.Span()
	if err != nil {
		return false, err
	}
	compound1, err := NewCompoundSelector([]SimpleSelector{s}, span)
	if err != nil {
		return false, err
	}
	compound2, err := NewCompoundSelector([]SimpleSelector{other}, span)
	if err != nil {
		return false, err
	}
	result, err := compound1.IsSuperselector(compound2)
	if err != nil {
		return false, err
	}
	return result, nil
}

// WithSelector returns a copy of this pseudo with its nested selector
// replaced by sel, preserving the name, spelling, and argument.
func (s *PseudoSelector) WithSelector(sel *SelectorList) (*PseudoSelector, error) {
	span, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewPseudoSelector(s.Name, span, s.IsElement(), s.Argument, sel), nil
}

// HashCode folds the name, element/class bit, argument, and nested selector
// together. Spans never contribute to selector hashes.
func (s *PseudoSelector) HashCode() int {
	h := hashCombine(stringHashCode(s.Name), boolHashCode(s.IsElement()))
	if s.Argument != nil {
		h = hashCombine(h, stringHashCode(*s.Argument))
	}
	if s.Selector != nil {
		h = hashCombine(h, s.Selector.HashCode())
	}
	return h
}

// AcceptVoid dispatches to VisitPseudoSelector on v.
func (s *PseudoSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitPseudoSelector(s)
}

// AcceptBool dispatches to VisitPseudoSelector on v.
func (s *PseudoSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitPseudoSelector(s)
}

// AcceptParentSelector dispatches to VisitPseudoSelector on v.
func (s *PseudoSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitPseudoSelector(s)
}
