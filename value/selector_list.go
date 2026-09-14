// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/list.dart

import (
	"errors"
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// A selector list.
//
// A selector list is composed of ComplexSelectors. It matches any element
// that matches any of the component selectors.
type SelectorList struct {
	base SelectorBase
	// Components are the complex selectors of this list. Never empty.
	Components []*ComplexSelector
}

// NewSelectorList creates a selector list from components, copying the slice
// to keep the list immutable. It returns an error when components is empty.
func NewSelectorList(components []*ComplexSelector, span sasscommon.FileSpan) (*SelectorList, error) {
	if len(components) == 0 {
		return nil, &sasscommon.ArgumentError{Message: "components may not be empty"}
	}
	c := make([]*ComplexSelector, len(components))
	copy(c, components)
	return &SelectorList{
		base:       NewSelectorBase(span),
		Components: c,
	}, nil
}

// Span returns the source span where this selector was written.
func (s *SelectorList) Span() (sasscommon.FileSpan, error) { return s.base.Span() }
func (s *SelectorList) IsAstNode()                         {}
func (s *SelectorList) IsSelector()                        {}

// HashCode folds the component hashes together. Spans never contribute to
// selector hashes.
func (s *SelectorList) HashCode() int {
	h := 0
	for _, comp := range s.Components {
		h = hashCombine(h, comp.HashCode())
	}
	return h
}

// AcceptVoid dispatches to VisitSelectorList on v.
func (s *SelectorList) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitSelectorList(s)
}

// AcceptBool dispatches to VisitSelectorList on v.
func (s *SelectorList) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitSelectorList(s)
}

// AcceptParentSelector dispatches to VisitSelectorList on v.
func (s *SelectorList) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitSelectorList(s)
}

// AsSassList returns a SassScript list representing this selector, in the
// same format as a list returned by `selector-parse()`: a comma-separated
// list with one space-separated entry per complex, where each entry holds the
// leading combinators, compound texts, and trailing combinators as unquoted
// strings.
func (s *SelectorList) AsSassList() (*SassList, error) {
	complexes := make([]Value, len(s.Components))
	for i, complex := range s.Components {
		var parts []Value
		for _, lc := range complex.LeadingCombinators {
			parts = append(parts, &SassString{Text: lc.String(), HasQuotes: false})
		}
		for _, comp := range complex.Components {
			text, err := comp.Selector.String()
			if err != nil {
				return nil, err
			}
			parts = append(parts, &SassString{Text: text, HasQuotes: false})
			for _, comb := range comp.Combinators {
				parts = append(parts, &SassString{Text: comb.String(), HasQuotes: false})
			}
		}
		list, err := NewSassList(parts, ListSeparatorSpace, false)
		if err != nil {
			return nil, err
		}
		complexes[i] = list
	}
	result, err := NewSassList(complexes, ListSeparatorComma, false)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Unify returns a SelectorList matching only elements matched by both this
// list and other, or nil when no such list exists. Every complex in this
// list is unified pairwise against every complex in other; the per-pair work
// lives in UnifyComplex in selector_extend.go (V4a).
func (s *SelectorList) Unify(other *SelectorList) (*SelectorList, error) {
	var contents []*ComplexSelector
	for _, complex1 := range s.Components {
		for _, complex2 := range other.Components {
			complex1Span, err := complex1.Span()
			if err != nil {
				return nil, err
			}
			unified, err := UnifyComplex([]*ComplexSelector{complex1, complex2}, complex1Span)
			if err != nil {
				return nil, err
			}
			contents = append(contents, unified...)
		}
	}
	if len(contents) == 0 {
		return nil, nil
	}
	sSpan, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewSelectorList(contents, sSpan)
}

// NestWithin returns a new selector list representing this list nested within
// parent, resolving parent selectors (`&`) against parent.
//
// By default, parent selectors in this list are replaced with parent. When
// preserveParentSelectors is set they are kept as-is instead. When
// implicitParent is set, complexes without an explicit parent selector are
// prepended with parent; otherwise they pass through unchanged.
//
// A nil parent means there is no enclosing style rule: the list returns
// unchanged unless it holds a suffixed parent selector (such as `&-suffix`),
// which cannot resolve without a parent and fails with a script error.
func (s *SelectorList) NestWithin(
	parent *SelectorList,
	implicitParent bool,
	preserveParentSelectors bool,
) (*SelectorList, error) {
	if parent == nil {
		if preserveParentSelectors {
			return s, nil
		}
		psv := &parentSelectorWithSuffixVisitor{}
		found, err := s.AcceptBool(psv)
		if err != nil {
			return nil, err
		}
		if found {
			ps := psv.result
			psSpan, err := ps.Span()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.SassException{
				Message: "A top-level selector may not contain a parent selector with a suffix.",
				Span:    psSpan,
			}
		}
		return s, nil
	}
	nested := make([][]*ComplexSelector, len(s.Components))
	for i, complex := range s.Components {
		var err error
		nested[i], err = s.nestWithinComplex(complex, parent, implicitParent, preserveParentSelectors)
		if err != nil {
			return nil, err
		}
	}
	flattened := flattenVertically(nested)
	sSpan, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewSelectorList(flattened, sSpan)
}

// nestWithinComplex resolves nesting for one complex selector. Complexes
// without a parent selector either pass through or concatenate onto each
// parent complex for implicit parents. Complexes with parent selectors
// resolve component by component: leading components without a parent
// accumulate on every in-progress result, while components holding a parent
// fan out against the already-accumulated prefixes.
func (s *SelectorList) nestWithinComplex(
	complex *ComplexSelector,
	parent *SelectorList,
	implicitParent bool,
	preserveParentSelectors bool,
) ([]*ComplexSelector, error) {
	hasParent := false
	if !preserveParentSelectors {
		cpv := &containsParentVisitor{}
		var err error
		hasParent, err = complex.AcceptBool(cpv)
		if err != nil {
			return nil, err
		}
	}
	if preserveParentSelectors || !hasParent {
		if !implicitParent {
			return []*ComplexSelector{complex}, nil
		}
		complexSpan, err := complex.Span()
		if err != nil {
			return nil, err
		}
		result := make([]*ComplexSelector, 0, len(parent.Components))
		for _, parentComplex := range parent.Components {
			result = append(result, parentComplex.Concatenate(complex, complexSpan, false))
		}
		return result, nil
	}

	var newComplexes []*ComplexSelector
	for _, component := range complex.Components {
		resolved, err := s.nestWithinCompound(component, parent)
		if err != nil {
			return nil, err
		}
		if resolved == nil {
			if len(newComplexes) == 0 {
				complexSpan, err := complex.Span()
				if err != nil {
					return nil, err
				}
				cs, err := NewComplexSelector(
					complex.LeadingCombinators,
					[]*ComplexSelectorComponent{component},
					complexSpan,
					false,
				)
				if err != nil {
					return nil, err
				}
				newComplexes = append(newComplexes, cs)
			} else {
				complexSpan, err := complex.Span()
				if err != nil {
					return nil, err
				}
				for i := range newComplexes {
					newComplexes[i] = newComplexes[i].WithAdditionalComponent(component, complexSpan, false)
				}
			}
		} else if len(newComplexes) == 0 {
			if len(complex.LeadingCombinators) == 0 {
				newComplexes = append(newComplexes, resolved...)
			} else {
				complexSpan, err := complex.Span()
				if err != nil {
					return nil, err
				}
				for _, resolvedComplex := range resolved {
					newLC := make([]sasscommon.CssValue[Combinator], 0, len(complex.LeadingCombinators)+len(resolvedComplex.LeadingCombinators))
					newLC = append(newLC, complex.LeadingCombinators...)
					if len(resolvedComplex.LeadingCombinators) > 0 {
						newLC = append(newLC, resolvedComplex.LeadingCombinators...)
					}
					cs, err := NewComplexSelector(
						newLC,
						resolvedComplex.Components,
						complexSpan,
						resolvedComplex.LineBreak,
					)
					if err != nil {
						return nil, err
					}
					newComplexes = append(newComplexes, cs)
				}
			}
		} else {
			var updatedComplexes []*ComplexSelector
			for _, newComplex := range newComplexes {
				newComplexSpan, err := newComplex.Span()
				if err != nil {
					return nil, err
				}
				for _, resolvedComplex := range resolved {
					updatedComplexes = append(updatedComplexes, newComplex.Concatenate(resolvedComplex, newComplexSpan, false))
				}
			}
			newComplexes = updatedComplexes
		}
	}
	return newComplexes, nil
}

// containsParentVisitor reports whether a selector tree holds a parent
// selector anywhere, descending into pseudo-selector arguments. It plays the
// role of Dart's _containsParentSelector helper: Go's bool-visitor port has
// no nullable search, so the check is spelled as a dedicated visitor
// returning a found flag.
type containsParentVisitor struct{}

func (v *containsParentVisitor) VisitParentSelector(*ParentSelector) (bool, error) {
	return true, nil
}

func (v *containsParentVisitor) VisitAttributeSelector(*AttributeSelector) (bool, error) {
	return false, nil
}
func (v *containsParentVisitor) VisitClassSelector(*ClassSelector) (bool, error) { return false, nil }
func (v *containsParentVisitor) VisitIDSelector(*IDSelector) (bool, error)       { return false, nil }
func (v *containsParentVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (bool, error) {
	return false, nil
}
func (v *containsParentVisitor) VisitTypeSelector(*TypeSelector) (bool, error) { return false, nil }
func (v *containsParentVisitor) VisitUniversalSelector(*UniversalSelector) (bool, error) {
	return false, nil
}

func (v *containsParentVisitor) VisitComplexSelector(complex *ComplexSelector) (bool, error) {
	for _, component := range complex.Components {
		result, err := v.VisitCompoundSelector(component.Selector)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *containsParentVisitor) VisitCompoundSelector(compound *CompoundSelector) (bool, error) {
	for _, simple := range compound.Components {
		result, err := simple.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *containsParentVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (bool, error) {
	if pseudo.Selector != nil {
		return pseudo.Selector.AcceptBool(v)
	}
	return false, nil
}

func (v *containsParentVisitor) VisitSelectorList(list *SelectorList) (bool, error) {
	for _, complex := range list.Components {
		result, err := v.VisitComplexSelector(complex)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

// parentSelectorWithSuffixVisitor finds the first parent selector carrying a
// suffix and captures it in result, returning true once found. NestWithin
// with no parent uses it to reject top-level suffixed parents, which cannot
// resolve without an enclosing rule.
type parentSelectorWithSuffixVisitor struct {
	result *ParentSelector
}

func (v *parentSelectorWithSuffixVisitor) VisitParentSelector(s *ParentSelector) (bool, error) {
	if s.Suffix != nil {
		v.result = s
		return true, nil
	}
	return false, nil
}

func (v *parentSelectorWithSuffixVisitor) VisitAttributeSelector(*AttributeSelector) (bool, error) {
	return false, nil
}
func (v *parentSelectorWithSuffixVisitor) VisitClassSelector(*ClassSelector) (bool, error) {
	return false, nil
}
func (v *parentSelectorWithSuffixVisitor) VisitIDSelector(*IDSelector) (bool, error) {
	return false, nil
}
func (v *parentSelectorWithSuffixVisitor) VisitPlaceholderSelector(*PlaceholderSelector) (bool, error) {
	return false, nil
}
func (v *parentSelectorWithSuffixVisitor) VisitTypeSelector(*TypeSelector) (bool, error) {
	return false, nil
}
func (v *parentSelectorWithSuffixVisitor) VisitUniversalSelector(*UniversalSelector) (bool, error) {
	return false, nil
}

func (v *parentSelectorWithSuffixVisitor) VisitComplexSelector(complex *ComplexSelector) (bool, error) {
	for _, component := range complex.Components {
		result, err := v.VisitCompoundSelector(component.Selector)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *parentSelectorWithSuffixVisitor) VisitCompoundSelector(compound *CompoundSelector) (bool, error) {
	for _, simple := range compound.Components {
		result, err := simple.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (v *parentSelectorWithSuffixVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (bool, error) {
	if pseudo.Selector != nil {
		return pseudo.Selector.AcceptBool(v)
	}
	return false, nil
}

func (v *parentSelectorWithSuffixVisitor) VisitSelectorList(list *SelectorList) (bool, error) {
	for _, complex := range list.Components {
		result, err := v.VisitComplexSelector(complex)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

// nestWithinCompound returns selector complexes for component with every
// parent selector replaced by parent.
//
// It returns nil when component holds no parent selector: only a leading
// parent, or a pseudo-selector whose nested argument contains one, counts.
// Pseudo arguments holding parents resolve first by nesting them with no
// implicit parent, so `:&hover` style nesting cannot accidentally prepend the
// parent twice. A non-parent first simple selector rebuilds the component
// unchanged; a lone suffix-less parent expands to the parent's own complexes;
// otherwise each parent complex merges with the component's remainder,
// appending suffixes onto the parent's last simple selector. Failures are
// rethrown with an additional "parent selector" span on the offending `&`.
func (s *SelectorList) nestWithinCompound(
	component *ComplexSelectorComponent,
	parent *SelectorList,
) ([]*ComplexSelector, error) {
	simples := component.Selector.Components
	containsSelectorPseudo := false
	for _, simple := range simples {
		if pseudo, ok := simple.(*PseudoSelector); ok {
			if selector := pseudo.Selector; selector != nil {
				cpv := &containsParentVisitor{}
				hasParent, err := selector.AcceptBool(cpv)
				if err != nil {
					return nil, err
				}
				if hasParent {
					containsSelectorPseudo = true
					break
				}
			}
		}
	}
	firstSimple := simples[0]
	if !containsSelectorPseudo && !IsParentSelector(firstSimple) {
		return nil, nil
	}

	// Resolve pseudo-selector arguments that contain parent selectors before
	// touching the leading `&`, so nested arguments resolve against the parent
	// without an implicit prepend of their own.
	var resolvedSimples []SimpleSelector
	if containsSelectorPseudo {
		resolvedSimples = make([]SimpleSelector, len(simples))
		for i, simple := range simples {
			if ps, ok := simple.(*PseudoSelector); ok && ps.Selector != nil {
				cpv := &containsParentVisitor{}
				hasParent, err := ps.Selector.AcceptBool(cpv)
				if err != nil {
					firstSimpleSpan, spanErr := firstSimple.Span()
					if spanErr != nil {
						return nil, spanErr
					}
					if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
						return nil, sasscommon.ThrowWithTrace(
							mse.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
							mse,
						)
					}
					if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
						return nil, sasscommon.ThrowWithTrace(
							se.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
							se,
						)
					}
					return nil, &sasscommon.MultiSpanSassException{
						Message:      err.Error(),
						Span:         firstSimpleSpan,
						PrimaryLabel: "",
						Secondary:    map[sasscommon.FileSpan]string{firstSimpleSpan: "parent selector"},
					}
				}
				if hasParent {
					firstSimpleSpan, err := firstSimple.Span()
					if err != nil {
						if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								mse.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								mse,
							)
						}
						if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								se.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								se,
							)
						}
						return nil, &sasscommon.MultiSpanSassException{
							Message:      err.Error(),
							Span:         firstSimpleSpan,
							PrimaryLabel: "",
							Secondary:    map[sasscommon.FileSpan]string{firstSimpleSpan: "parent selector"},
						}
					}
					nested, err := ps.Selector.(*SelectorList).NestWithin(parent, false, false)
					if err != nil {
						if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								mse.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								mse,
							)
						}
						if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								se.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								se,
							)
						}
						return nil, &sasscommon.MultiSpanSassException{
							Message:      err.Error(),
							Span:         firstSimpleSpan,
							PrimaryLabel: "",
							Secondary:    map[sasscommon.FileSpan]string{firstSimpleSpan: "parent selector"},
						}
					}
					psel, err := ps.WithSelector(nested)
					if err != nil {
						if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								mse.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								mse,
							)
						}
						if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
							return nil, sasscommon.ThrowWithTrace(
								se.WithAdditionalSpan(firstSimpleSpan, "parent selector"),
								se,
							)
						}
						return nil, &sasscommon.MultiSpanSassException{
							Message:      err.Error(),
							Span:         firstSimpleSpan,
							PrimaryLabel: "",
							Secondary:    map[sasscommon.FileSpan]string{firstSimpleSpan: "parent selector"},
						}
					}
					resolvedSimples[i] = psel
				} else {
					resolvedSimples[i] = simple
				}
			} else {
				resolvedSimples[i] = simple
			}
		}
	} else {
		resolvedSimples = simples
	}

	parentSelectorSpan, err := firstSimple.Span()
	if err != nil {
		return nil, err
	}
	parentSelector := firstSimple
	ps, ok := parentSelector.(*ParentSelector)
	if !ok {
		componentSelectorSpan, err := component.Selector.Span()
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		compound, err := NewCompoundSelector(resolvedSimples, componentSelectorSpan)
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		cs, err := NewComplexSelector(
			nil,
			[]*ComplexSelectorComponent{
				NewComplexSelectorComponent(
					compound,
					component.Combinators,
					component.Span,
				),
			},
			component.Span,
			false,
		)
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		return []*ComplexSelector{cs}, nil
	}

	// Fast path: a lone `&` with no suffix expands directly to the parent's
	// complexes, carrying over this component's trailing combinators.
	if len(simples) == 1 && ps.Suffix == nil {
		withCombinators, err := parent.WithAdditionalCombinators(component.Combinators)
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		return withCombinators.Components, nil
	}

	// General case: splice each parent complex's last compound together with
	// the `&` remainder. A parent ending in a combinator cannot merge into a
	// compound and fails; a suffix rewrites the parent's last simple while a
	// bare `&` just appends the remaining simples.
	result := make([]*ComplexSelector, 0, len(parent.Components))
	for _, complex := range parent.Components {
		lastComponent := complex.Components[len(complex.Components)-1]
		if len(lastComponent.Combinators) > 0 {
			trimmed, err := lastComponent.Span.TrimRight()
			if err != nil {
				return nil, err
			}
			complexStr, err := complex.String()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      fmt.Sprintf("Selector \"%s\" can't be used as a parent in a compound selector.", complexStr),
				Span:         trimmed,
				PrimaryLabel: "outer selector",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}

		suffix := ps.Suffix
		lastSimples := lastComponent.Selector.Components
		var last *CompoundSelector
		componentSelectorSpan, err := component.Selector.Span()
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		if suffix == nil {
			combined := make([]SimpleSelector, 0, len(lastSimples)+len(resolvedSimples)-1)
			combined = append(combined, lastSimples...)
			combined = append(combined, resolvedSimples[1:]...)
			last, err = NewCompoundSelector(combined, componentSelectorSpan)
			if err != nil {
				if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						mse,
					)
				}
				if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						se,
					)
				}
				return nil, &sasscommon.MultiSpanSassException{
					Message:      err.Error(),
					Span:         parentSelectorSpan,
					PrimaryLabel: "",
					Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
				}
			}
		} else {
			combined := make([]SimpleSelector, 0, len(lastSimples)+len(resolvedSimples)-1)
			combined = append(combined, lastSimples[:len(lastSimples)-1]...)
			added, err := lastSimples[len(lastSimples)-1].AddSuffix(*suffix)
			if err != nil {
				if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						mse,
					)
				}
				if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						se,
					)
				}
				return nil, &sasscommon.MultiSpanSassException{
					Message:      err.Error(),
					Span:         parentSelectorSpan,
					PrimaryLabel: "",
					Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
				}
			}
			combined = append(combined, added)
			combined = append(combined, resolvedSimples[1:]...)
			last, err = NewCompoundSelector(combined, componentSelectorSpan)
			if err != nil {
				if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						mse,
					)
				}
				if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
					return nil, sasscommon.ThrowWithTrace(
						se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
						se,
					)
				}
				return nil, &sasscommon.MultiSpanSassException{
					Message:      err.Error(),
					Span:         parentSelectorSpan,
					PrimaryLabel: "",
					Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
				}
			}
		}

		comps := make([]*ComplexSelectorComponent, 0, len(complex.Components))
		comps = append(comps, complex.Components[:len(complex.Components)-1]...)
		comps = append(comps, NewComplexSelectorComponent(last, component.Combinators, component.Span))

		cs, err := NewComplexSelector(
			complex.LeadingCombinators,
			comps,
			component.Span,
			complex.LineBreak,
		)
		if err != nil {
			if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					mse.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					mse,
				)
			}
			if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				return nil, sasscommon.ThrowWithTrace(
					se.WithAdditionalSpan(parentSelectorSpan, "parent selector"),
					se,
				)
			}
			return nil, &sasscommon.MultiSpanSassException{
				Message:      err.Error(),
				Span:         parentSelectorSpan,
				PrimaryLabel: "",
				Secondary:    map[sasscommon.FileSpan]string{parentSelectorSpan: "parent selector"},
			}
		}
		result = append(result, cs)
	}
	return result, nil
}

// IsSuperselector reports whether this list matches every element that other
// matches, possibly plus more. The list-level comparison itself lives in
// selector_extend.go (V4a).
func (s *SelectorList) IsSuperselector(other *SelectorList) (bool, error) {
	return listIsSuperselector(s.Components, other.Components)
}

// WithAdditionalCombinators returns a copy of this list with combinators
// appended to every complex. For internal use by nesting resolution; an
// empty addition returns the list unchanged.
func (s *SelectorList) WithAdditionalCombinators(
	combinators []sasscommon.CssValue[Combinator],
) (*SelectorList, error) {
	if len(combinators) == 0 {
		return s, nil
	}
	newComponents := make([]*ComplexSelector, len(s.Components))
	for i, comp := range s.Components {
		newComponents[i] = comp.WithAdditionalCombinators(combinators, false)
	}
	sSpan, err := s.Span()
	if err != nil {
		return nil, err
	}
	return NewSelectorList(newComponents, sSpan)
}

// listIsSuperselector forwards the list-level comparison to the
// extend-algorithm implementation in selector_extend.go (V4a).
func listIsSuperselector(a, b []*ComplexSelector) (bool, error) {
	return ListIsSuperselector(a, b)
}

// ContainsParentSelector reports whether any complex in this list contains a
// parent selector.
func (s *SelectorList) ContainsParentSelector() (bool, error) {
	v := &containsParentVisitor{}
	return s.AcceptBool(v)
}

// IsBogusOtherThanLeadingCombinator reports whether any complex in this list
// is invalid CSS beyond a leading combinator.
func (s *SelectorList) IsBogusOtherThanLeadingCombinator() bool {
	for _, comp := range s.Components {
		if comp.IsBogusOtherThanLeadingCombinator() {
			return true
		}
	}
	return false
}

// IsInvisibleOtherThanBogusCombinators reports whether every complex in this
// list is invisible for reasons other than bogus combinators. An empty match
// set hides the whole rule.
func (s *SelectorList) IsInvisibleOtherThanBogusCombinators() bool {
	for _, comp := range s.Components {
		if !comp.IsInvisibleOtherThanBogusCombinators() {
			return false
		}
	}
	return true
}

// String renders this selector in inspect mode.
func (s *SelectorList) String() (string, error) {
	return SerializeSelector(s, true)
}

// AssertNotBogus warns through warn when this selector is not valid CSS,
// forwarding to the shared helper.
func (s *SelectorList) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// flattenVertically interleaves nested complex slices round-robin: the first
// element of each inner slice comes before the second element of any slice,
// preserving source order across the fanned-out nesting results.
// The result is ordered first by index in the nested slice, then by the index
// of that slice. For example:
//
//	flattenVertically([["1a","1b"], ["2a","2b"]]) returns ["1a","2a","1b","2b"]
func flattenVertically(components [][]*ComplexSelector) []*ComplexSelector {
	if len(components) == 1 {
		return components[0]
	}
	queues := make([][]*ComplexSelector, len(components))
	for i, inner := range components {
		q := make([]*ComplexSelector, len(inner))
		copy(q, inner)
		queues[i] = q
	}
	var result []*ComplexSelector
	for len(queues) > 0 {
		i := 0
		for i < len(queues) {
			result = append(result, queues[i][0])
			queues[i] = queues[i][1:]
			if len(queues[i]) == 0 {
				queues = append(queues[:i], queues[i+1:]...)
			} else {
				i++
			}
		}
	}
	return result
}
