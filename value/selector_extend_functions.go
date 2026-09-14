// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// This file ports the per-selector helpers the extend algorithm builds on:
// each SimpleSelector's compound unification, the invisibility/bogus/useless
// predicates, and structural equality.
//
// dart-source: lib/src/ast/selector/simple.dart (base unify) +
// lib/src/ast/selector/{class,id,type,universal,pseudo,parent}.dart (unify
// overrides) + lib/src/ast/selector/placeholder.dart (isPrivate) +
// lib/src/ast/selector.dart (isInvisible/isBogus/isUseless visitors) —
// class/attribute/placeholder reuse the base unify and operator== lives on
// each selector type.

import "github.com/bancek/go-sass/sasscommon"

// simpleBaseUnify merges s into comps, returning the combined compound or
// nil when unification is impossible. A lone universal or :host-style pseudo
// on the other side takes over (it knows how to absorb s); an s already
// present keeps comps as-is; otherwise s slots in ahead of the first pseudo
// so pseudo selectors always close the compound. (Dart: SimpleSelector.unify
// in lib/src/ast/selector/simple.dart — internal there, hence plain comment.)
func simpleBaseUnify(s SimpleSelector, comps []SimpleSelector) ([]SimpleSelector, error) {
	// If compound is a single UniversalSelector or host/host-context PseudoSelector,
	// delegate to other.unify([this]).
	if len(comps) == 1 {
		other := comps[0]
		if _, ok := other.(*UniversalSelector); ok {
			return other.Unify([]SimpleSelector{s})
		}
		if ps, ok := other.(*PseudoSelector); ok && (ps.IsHost() || ps.IsHostContext()) {
			return other.Unify([]SimpleSelector{s})
		}
	}
	// If this selector already exists in the compound, return as-is.
	for _, comp := range comps {
		if comp.Equals(s) {
			return comps, nil
		}
	}
	// Insert before any PseudoSelector so pseudo selectors come last.
	result := make([]SimpleSelector, 0, len(comps)+1)
	addedThis := false
	for _, comp := range comps {
		if !addedThis {
			if _, ok := comp.(*PseudoSelector); ok {
				result = append(result, s)
				addedThis = true
			}
		}
		result = append(result, comp)
	}
	if !addedThis {
		result = append(result, s)
	}
	return result, nil
}

// ClassSelector reuses the base unification: classes never conflict, they
// just join the compound. (Dart: no unify override in class.dart.)
func (s *ClassSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	return simpleBaseUnify(s, comps)
}

// IDSelector refuses a compound holding a different ID — one element has one
// id attribute — then defers to the base merge. (Dart: id.dart unify.)
func (s *IDSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	for _, other := range comps {
		if o, ok := other.(*IDSelector); ok && !o.Equals(s) {
			return nil, nil
		}
	}
	return simpleBaseUnify(s, comps)
}

// TypeSelector folds into a leading universal or type selector via
// unifyUniversalAndElement (`div` + `*` = `div`); against anything else it
// takes the front, since a type selector always opens the compound.
// (Dart: type.dart unify.)
func (s *TypeSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	if len(comps) > 0 {
		switch comps[0].(type) {
		case *UniversalSelector, *TypeSelector:
			unified, err := unifyUniversalAndElement(s, comps[0])
			if err != nil {
				return nil, err
			}
			if unified == nil {
				return nil, nil
			}
			result := make([]SimpleSelector, 1, len(comps))
			result[0] = unified
			result = append(result, comps[1:]...)
			return result, nil
		}
	}
	// Dart else: return [this, ...compound] — type selector comes first
	result := make([]SimpleSelector, 1, len(comps)+1)
	result[0] = s
	result = append(result, comps...)
	return result, nil
}

// UniversalSelector merges with a leading universal or type selector
// (`*|*` + `svg|a` = `svg|a`); a lone :host-style pseudo cannot absorb a
// universal, so that fails; an empty compound is just the universal itself;
// otherwise a namespaced universal (`svg|*`) takes the front while a bare
// `*` adds nothing and vanishes. (Dart: universal.dart unify.)
func (s *UniversalSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	// case [UniversalSelector() || TypeSelector(), ...var rest]:
	if len(comps) > 0 {
		switch comps[0].(type) {
		case *UniversalSelector, *TypeSelector:
			unified, err := unifyUniversalAndElement(s, comps[0])
			if err != nil {
				return nil, err
			}
			if unified == nil {
				return nil, nil
			}
			result := make([]SimpleSelector, 1, len(comps))
			result[0] = unified
			result = append(result, comps[1:]...)
			return result, nil
		}
	}
	// case [PseudoSelector first] when first.isHost || first.isHostContext:
	if len(comps) == 1 {
		if ps, ok := comps[0].(*PseudoSelector); ok && (ps.IsHost() || ps.IsHostContext()) {
			return nil, nil
		}
	}
	// case []:
	if len(comps) == 0 {
		return []SimpleSelector{s}, nil
	}
	// case _:
	if s.Namespace == nil || *s.Namespace == "*" {
		r := make([]SimpleSelector, len(comps))
		copy(r, comps)
		return r, nil
	}
	result := make([]SimpleSelector, 1, len(comps)+1)
	result[0] = s
	result = append(result, comps...)
	return result, nil
}

// PseudoSelector keeps pseudo-classes ahead of pseudo-elements: a second
// pseudo-element fails the compound (an element has one ::before), while a
// pseudo-class slots in before any element. :host and :host-context only
// merge with other host-capable pseudos or selector-bearing pseudos —
// anything else (a class, an ID) cannot live inside a shadow host. A lone
// universal or host pseudo on the other side takes over, as in the base
// merge. (Dart: pseudo.dart unify.)
func (s *PseudoSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	if s.Name == "host" || s.Name == "host-context" {
		for _, simple := range comps {
			ps, ok := simple.(*PseudoSelector)
			if !ok || (!ps.IsHost() && ps.Selector == nil) {
				return nil, nil
			}
		}
	} else if len(comps) == 1 {
		other := comps[0]
		if _, ok := other.(*UniversalSelector); ok {
			return other.Unify([]SimpleSelector{s})
		}
		if ps, ok := other.(*PseudoSelector); ok && (ps.IsHost() || ps.IsHostContext()) {
			return other.Unify([]SimpleSelector{s})
		}
	}

	for _, simple := range comps {
		if simple.Equals(s) {
			return comps, nil
		}
	}

	result := make([]SimpleSelector, 0, len(comps)+1)
	addedThis := false
	for _, simple := range comps {
		if ps, ok := simple.(*PseudoSelector); ok && ps.IsElement() {
			if s.IsElement() {
				return nil, nil
			}
			result = append(result, s)
			addedThis = true
		}
		result = append(result, simple)
	}
	if !addedThis {
		result = append(result, s)
	}
	return result, nil
}

// PlaceholderSelector reuses the base unification: placeholders never
// conflict, they just join the compound. (Dart: no unify override in
// placeholder.dart.)
func (s *PlaceholderSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	return simpleBaseUnify(s, comps)
}

// AttributeSelector reuses the base unification: attribute selectors never
// conflict, they just join the compound. (Dart: no unify override in
// attribute.dart.)
func (s *AttributeSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	return simpleBaseUnify(s, comps)
}

// ParentSelector cannot unify: `&` resolves against the stylesheet parent,
// not against sibling selectors, so merging it into a compound is
// meaningless and always errors. (Dart: parent.dart unify.)
func (s *ParentSelector) Unify(comps []SimpleSelector) ([]SimpleSelector, error) {
	return nil, &sasscommon.UnsupportedError{Message: "& doesn't support unification."}
}

// Invisibility: whether a selector (and any complex containing it) must not
// be emitted. Only placeholders are invisible on their own — %foo never
// matches an element — while :not(%foo) stays visible (it means "not
// nothing", i.e. *). Internal in Dart (selector.dart _IsInvisibleVisitor),
// hence plain comments here.
func (s *PlaceholderSelector) IsInvisible() bool { return true }
func (s *ClassSelector) IsInvisible() bool       { return false }
func (s *IDSelector) IsInvisible() bool          { return false }
func (s *TypeSelector) IsInvisible() bool        { return false }
func (s *UniversalSelector) IsInvisible() bool   { return false }

func (s *AttributeSelector) IsInvisible() bool { return false }
func (s *ParentSelector) IsInvisible() bool    { return false }

func (c *CompoundSelector) IsInvisible() bool {
	for _, comp := range c.Components {
		if comp.IsInvisible() {
			return true
		}
	}
	return false
}

// A complex is invisible when any compound in it is invisible (a placeholder
// anywhere in the ancestor chain matches nothing), or when it carries bogus
// combinators outside the leading position; an empty complex matches nothing
// and is trivially invisible.
func (c *ComplexSelector) IsInvisible() bool {
	if len(c.Components) == 0 {
		return true
	}
	for _, comp := range c.Components {
		if comp.Selector.IsInvisible() {
			return true
		}
	}
	return c.IsBogusOtherThanLeadingCombinator()
}

// A list is invisible only when every complex in it is — one visible branch
// still emits.
func (s *SelectorList) IsInvisible() bool {
	for _, comp := range s.Components {
		if !comp.IsInvisible() {
			return false
		}
	}
	return true
}

// Uselessness: whether a selector is bogus *and* beyond rescue — neither
// @extend nor nesting can turn it into valid CSS (for example doubled-up
// combinators, or a compound @extend can never fix). UnifyComplex rejects
// useless inputs outright. Internal in Dart (selector.dart _IsUselessVisitor).
func (s *ComplexSelector) IsUseless() bool {
	if len(s.LeadingCombinators) > 1 {
		return true
	}
	for _, comp := range s.Components {
		if len(comp.Combinators) > 1 {
			return true
		}
		if comp.Selector.IsUseless() {
			return true
		}
	}
	return false
}

// A list is useless when any complex in it is — @extend still processes the
// salvageable branches.
func (s *SelectorList) IsUseless() bool {
	for _, comp := range s.Components {
		if comp.IsUseless() {
			return true
		}
	}
	return false
}

// IsBogus reports whether the complex is not valid CSS — a build-time-only
// shape (a leading `>` awaiting nesting) or invalid-but-tolerated syntax
// (doubled-up combinators like `.foo + ~ .bar`). An empty component list with
// leading combinators, any leading combinator, a trailing combinator on the
// last component, doubled-up combinators, or a bogus compound anywhere inside
// all qualify. (Dart: Selector.isBogus via _IsBogusVisitor in selector.dart.)
func (s *ComplexSelector) IsBogus() bool {
	if len(s.Components) == 0 {
		return len(s.LeadingCombinators) > 0
	}
	if len(s.LeadingCombinators) > 0 {
		return true
	}
	if len(s.Components[len(s.Components)-1].Combinators) > 0 {
		return true
	}
	for _, comp := range s.Components {
		if len(comp.Combinators) > 1 {
			return true
		}
		if comp.Selector.IsBogus() {
			return true
		}
	}
	return false
}

// IsBogus reports whether any complex in the list is not valid CSS. One
// bogus branch poisons serialization of the whole list.
func (s *SelectorList) IsBogus() bool {
	for _, comp := range s.Components {
		if comp.IsBogus() {
			return true
		}
	}
	return false
}

// IsBogus reports whether any simple selector in the compound is not valid
// CSS, delegating to each member (pseudos apply the :has leading-combinator
// exemption themselves).
func (c *CompoundSelector) IsBogus() bool {
	for _, comp := range c.Components {
		if comp.IsBogus() {
			return true
		}
	}
	return false
}

// A compound is useless when any member is — @extend cannot rescue a branch
// whose combinators are already invalid.
func (c *CompoundSelector) IsUseless() bool {
	for _, comp := range c.Components {
		if comp.IsUseless() {
			return true
		}
	}
	return false
}

// IsPrivate reports whether the placeholder is private — its name begins
// with `-` or `_`. Private placeholders stay within their module: @extend
// across modules cannot reach them.
func (s *PlaceholderSelector) IsPrivate() bool {
	return len(s.Name) > 0 && (s.Name[0] == '-' || s.Name[0] == '_')
}

// Structural equality (Dart: operator== on each selector type — spans never
// participate). Every Equals below compares only the selecting state: names,
// namespaces, operators, and nested selector arguments.

// Equals reports whether other is a class selector for the same class.
func (s *ClassSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*ClassSelector)
	return ok && s.Name == o.Name
}

// Equals reports whether other is an ID selector for the same ID.
func (s *IDSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*IDSelector)
	return ok && s.Name == o.Name
}

// Equals reports whether other is a type selector with an equal qualified
// name (namespace included: `svg|a` differs from `a`).
func (s *TypeSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*TypeSelector)
	return ok && s.Name.Equal(o.Name)
}

// Equals reports whether other is a universal selector in the same namespace.
// A nil (default) namespace differs from `""` (none) and `"*"` (any).
func (s *UniversalSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*UniversalSelector)
	return ok && ptrEq(s.Namespace, o.Namespace)
}

// Equals reports whether other is the same pseudo: equal names, equal
// class/element sense, equal raw arguments, and structurally equal selector
// arguments. Nested selector lists compare by value — never by pointer, since
// separately parsed `:is(.a)` arguments must still match.
func (s *PseudoSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*PseudoSelector)
	// Do NOT change this to a pointer equality (==) check.
	return ok &&
		s.Name == o.Name &&
		s.IsClass == o.IsClass &&
		ptrEqStr(s.Argument, o.Argument) &&
		EqualSelectors(s.Selector, o.Selector)
}

// Equals reports whether other is a placeholder with the same name.
func (s *PlaceholderSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*PlaceholderSelector)
	return ok && s.Name == o.Name
}

// Equals reports whether other is an attribute selector with equal name,
// operator, value, and modifier.
func (s *AttributeSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*AttributeSelector)
	return ok && s.Name.Equal(o.Name) && ptrEqOp(s.Op, o.Op) &&
		ptrEqStr(s.Value, o.Value) && ptrEqStr(s.Modifier, o.Modifier)
}

// Equals reports whether other is a parent selector with an equal suffix.
func (s *ParentSelector) Equals(other SimpleSelector) bool {
	o, ok := other.(*ParentSelector)
	return ok && ptrEqStr(s.Suffix, o.Suffix)
}

// Equals reports whether other holds the same components in order. Identity
// short-circuits; nil never equals non-nil.
func (c *CompoundSelector) Equals(other *CompoundSelector) bool {
	if c == other {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if len(c.Components) != len(other.Components) {
		return false
	}
	for i := range c.Components {
		if !c.Components[i].Equals(other.Components[i]) {
			return false
		}
	}
	return true
}

// Equals reports whether other holds the same complexes in order — the list
// equality the extension store relies on when matching extenders. (Dart's
// SelectorList.== is structural; the store's identity map is a separate
// allocation-identity mechanism.)
func (s *SelectorList) Equals(other *SelectorList) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	if len(s.Components) != len(other.Components) {
		return false
	}
	for i := range s.Components {
		if !s.Components[i].Equals(other.Components[i]) {
			return false
		}
	}
	return true
}

// EqualSelectors reports whether two Selector values are structurally equal,
// dispatching to the concrete Equals for the dynamic type. Mismatched dynamic
// types never match; two nils match. (Dart has no direct counterpart — its
// operator== dispatches natively; this switch recovers that over the Go
// Selector interface.)
func EqualSelectors(a, b Selector) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a == b {
		return true
	}
	switch a := a.(type) {
	case *SelectorList:
		b, ok := b.(*SelectorList)
		return ok && a.Equals(b)
	case *CompoundSelector:
		b, ok := b.(*CompoundSelector)
		return ok && a.Equals(b)
	case *ComplexSelector:
		b, ok := b.(*ComplexSelector)
		return ok && a.Equals(b)
	case SimpleSelector:
		b, ok := b.(SimpleSelector)
		return ok && a.Equals(b)
	}
	return false
}

// ptrEqStr compares optional strings by value (nil equals nil), for pseudo
// arguments, attribute values, modifiers, and parent suffixes.
func ptrEqStr(a, b *string) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ptrEqOp compares optional attribute operators by value.
func ptrEqOp(a, b *AttributeOperator) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Compile-time proof that every SimpleSelector carries its Unify method: the
// extend algorithm calls Unify through the interface, so a missing override
// must fail the build, not a weaving run.
var _ = (SimpleSelector)((*ClassSelector)(nil)).Unify
var _ = (SimpleSelector)((*IDSelector)(nil)).Unify
var _ = (SimpleSelector)((*TypeSelector)(nil)).Unify
var _ = (SimpleSelector)((*UniversalSelector)(nil)).Unify
var _ = (SimpleSelector)((*PseudoSelector)(nil)).Unify
var _ = (SimpleSelector)((*PlaceholderSelector)(nil)).Unify
var _ = (SimpleSelector)((*AttributeSelector)(nil)).Unify
var _ = (SimpleSelector)((*ParentSelector)(nil)).Unify
