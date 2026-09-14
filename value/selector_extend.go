// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// This file ports the selector-extension algorithm: complex unification,
// parent-selector weaving, and the superselector predicates that drive
// @extend. Simple-selector Unify methods and the invisibility/bogus/useless
// predicates live in selector_extend_functions.go.
//
// dart-source: lib/src/extend/functions.dart (unifyComplex, weave, and
// superselector sections) + lib/src/utils.dart (longestCommonSubsequence,
// ported as lcsFn) + lib/src/util/iterable.dart (paths, ported as Paths)

import (
	"fmt"
	"slices"

	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/sasscommon"
)

// Rootish pseudo-classes that may only meaningfully appear in the first
// component of a complex selector. Weave hoists and unifies them up front so
// a :root buried mid-selector can never leak into the output.
var rootishPseudoClasses = map[string]bool{
	"root": true, "scope": true, "host": true, "host-context": true,
}

// UnifyComplex returns the contents of a SelectorList matching only elements
// matched by every complex selector in complexes, or nil when no such list
// exists. span marks any complex selectors built along the way.
//
// Worked example: unifying `.foo .bar` with `.baz .bar` first unifies the
// shared base (`.bar` with `.bar`), then weaves the leftover parents
// (`.foo`, `.baz`) back in front, yielding `.foo .baz .bar` and
// `.baz .foo .bar` (plus merged forms such as `.foo.baz .bar` where the
// parents unify). A nil result means the selectors are disjoint — for
// example conflicting IDs (`#a` vs `#b`) or mismatched leading combinators.
func UnifyComplex(complexes []*ComplexSelector, span sasscommon.FileSpan) ([]*ComplexSelector, error) {
	if len(complexes) == 1 {
		return complexes, nil
	}

	var unifiedBase *CompoundSelector
	var leadingCombinator *sasscommon.CssValue[Combinator]
	var trailingCombinator *sasscommon.CssValue[Combinator]
	// A useless complex (for example one with a doubled-up combinator) can
	// never match, so the intersection is empty.
	for _, complex := range complexes {
		if complex.IsUseless() {
			return nil, nil
		}
		// Single-component complexes contribute a leading combinator (as in
		// `:has(> .foo)`); every input must agree on it or unification fails.
		if len(complex.LeadingCombinators) == 1 && len(complex.Components) == 1 {
			lc := complex.LeadingCombinators[0]
			if leadingCombinator == nil {
				leadingCombinator = &lc
			} else if !leadingCombinator.Equal(lc) {
				return nil, nil
			}
		}
		base := complex.Components[len(complex.Components)-1]
		// Likewise every input must agree on the trailing combinator of its
		// base component; `a > b` unified with `a + b` has no single answer.
		if len(base.Combinators) == 1 {
			tc := base.Combinators[0]
			if trailingCombinator != nil && !trailingCombinator.Equal(tc) {
				return nil, nil
			}
			trailingCombinator = &tc
		}
		// Fold each base compound into the running intersection. A nil
		// unifiedBase (ID clash, conflicting type selectors) ends the search.
		if unifiedBase == nil {
			unifiedBase = base.Selector
		} else {
			var err error
			unifiedBase, err = unifyCompound(unifiedBase, base.Selector)
			if err != nil {
				return nil, err
			}
			if unifiedBase == nil {
				return nil, nil
			}
		}
	}

	var withoutBases []*ComplexSelector
	for _, complex := range complexes {
		if len(complex.Components) > 1 {
			span, err := complex.Span()
			if err != nil {
				return nil, err
			}
			cs, err := NewComplexSelector(
				complex.LeadingCombinators,
				complex.Components[:len(complex.Components)-1],
				span,
				complex.LineBreak,
			)
			if err != nil {
				return nil, err
			}
			withoutBases = append(withoutBases, cs)
		}
	}

	baseLC := leadingCombinator
	var baseLCs []sasscommon.CssValue[Combinator]
	if baseLC != nil {
		baseLCs = []sasscommon.CssValue[Combinator]{*baseLC}
	}
	var baseTCs []sasscommon.CssValue[Combinator]
	if trailingCombinator != nil {
		baseTCs = []sasscommon.CssValue[Combinator]{*trailingCombinator}
	}
	base, err := NewComplexSelector(
		baseLCs,
		[]*ComplexSelectorComponent{NewComplexSelectorComponent(unifiedBase, baseTCs, span)},
		span, false,
	)
	if err != nil {
		return nil, err
	}
	// A line break anywhere propagates to the unified base so the emitted
	// selector keeps its multi-line shape.
	for _, c := range complexes {
		if c.LineBreak {
			base.LineBreak = true
			break
		}
	}

	// With no parents left the base stands alone; otherwise the last parent
	// chain absorbs the base (concatenate) and weave interleaves the rest.
	if len(withoutBases) == 0 {
		return Weave([]*ComplexSelector{base}, span, nil)
	}
	last := withoutBases[len(withoutBases)-1].Concatenate(base, span, false)
	combined := append(withoutBases[:len(withoutBases)-1], last)
	return Weave(combined, span, nil)
}

// unifyCompound returns a compound matching only elements matched by both
// inputs, or nil when they conflict. compound1's span marks the result.
//
// Pseudo ordering is preserved per input: pseudo-classes seen before a
// pseudo-element stay before it, while ones seen after are unified
// separately (pseudoResult) so their relative order with the element survives.
// For example unifying `::foo:bar` with `:baz` yields `:baz::foo:bar` — :baz
// moves before ::foo, but :bar keeps its place after it.
func unifyCompound(compound1, compound2 *CompoundSelector) (*CompoundSelector, error) {
	result := compound1.Components
	var pseudoResult []SimpleSelector
	pseudoElementFound := false

	for _, simple := range compound2.Components {
		if pseudoElementFound {
			// Everything past the first pseudo-element unifies against the
			// trailing pseudoResult list, keeping pre- and post-element
			// pseudo-classes in their original lanes.
			if ps, ok := simple.(*PseudoSelector); ok {
				unified, err := ps.Unify(pseudoResult)
				if err != nil {
					return nil, err
				}
				if unified == nil {
					return nil, nil
				}
				pseudoResult = unified
				continue
			}
		}
		if ps, ok := simple.(*PseudoSelector); ok && ps.IsElement() {
			pseudoElementFound = true
		}
		unified, err := simple.Unify(result)
		if err != nil {
			return nil, err
		}
		if unified == nil {
			return nil, nil
		}
		result = unified
	}

	all := make([]SimpleSelector, 0, len(result)+len(pseudoResult))
	all = append(all, result...)
	all = append(all, pseudoResult...)
	span, err := compound1.Span()
	if err != nil {
		return nil, err
	}
	compound, err := NewCompoundSelector(all, span)
	if err != nil {
		return nil, err
	}
	return compound, nil
}

// unifyUniversalAndElement returns a simple selector matching only elements
// matched by both inputs, which must each be a universal or type selector, or
// nil when namespaces or names conflict. A wildcard (`*`) or absent name on
// either side defers to the other: `*` with `svg|a` is `svg|a`, `*|*` with
// `div` is `div`, and two concrete mismatches fail.
func unifyUniversalAndElement(s1, s2 SimpleSelector) (SimpleSelector, error) {
	ns1, n1, err := namespaceAndName(s1)
	if err != nil {
		return nil, err
	}
	ns2, n2, err := namespaceAndName(s2)
	if err != nil {
		return nil, err
	}

	var ns *string
	if ptrEq(ns1, ns2) || (ns2 != nil && *ns2 == "*") {
		ns = ns1
	} else if ns1 != nil && *ns1 == "*" {
		ns = ns2
	} else {
		return nil, nil
	}

	var n *string
	if ptrEq(n1, n2) || n2 == nil {
		n = n1
	} else if n1 == nil || (n1 != nil && *n1 == "*") {
		n = n2
	} else {
		return nil, nil
	}

	span, err := s1.Span()
	if err != nil {
		return nil, err
	}
	if n == nil {
		return NewUniversalSelector(span, ns), nil
	}
	return NewTypeSelector(QualifiedName{Name: *n, Namespace: ns}, span), nil
}

// namespaceAndName splits s into its namespace and name parts. A universal
// selector contributes only a namespace (nil name); anything else is a caller
// bug and reports an error naming the offending selector kind.
func namespaceAndName(s SimpleSelector) (ns, name *string, err error) {
	switch v := s.(type) {
	case *UniversalSelector:
		return v.Namespace, nil, nil
	case *TypeSelector:
		return v.Name.Namespace, &v.Name.Name, nil
	default:
		return nil, nil, fmt.Errorf("must be UniversalSelector or TypeSelector")
	}
}

// Weave expands "parenthesized selectors" in complexes. Given
// `.A .B {@extend .C}` and `.D .C {...}`, extension conceptually needs
// `.D (.A .B)` — represented here as the list `[.D, .A .B]` — and Weave
// translates that into `.D .A .B, .A .D .B`. Fully merged forms like
// `.A.D .B` are deliberately skipped: they would blow output up
// exponentially for very little gain. span marks any combined selectors, and
// forceLineBreak (nil = false) stamps line breaks onto every result.
func Weave(complexes []*ComplexSelector, span sasscommon.FileSpan, forceLineBreak *bool) ([]*ComplexSelector, error) {
	fb := false
	if forceLineBreak != nil {
		fb = *forceLineBreak
	}
	if len(complexes) == 1 {
		if !fb || complexes[0].LineBreak {
			return complexes, nil
		}
		span, err := complexes[0].Span()
		if err != nil {
			return nil, err
		}
		cs, err := NewComplexSelector(complexes[0].LeadingCombinators, complexes[0].Components, span, true)
		if err != nil {
			return nil, err
		}
		return []*ComplexSelector{cs}, nil
	}

	prefixes := []*ComplexSelector{complexes[0]}
	for _, c := range complexes[1:] {
		if len(c.Components) == 1 {
			// A lone target appends to every accumulated prefix — no
			// interleaving is possible with a single component.
			for i := range prefixes {
				prefixes[i] = prefixes[i].Concatenate(c, span, fb)
			}
			continue
		}
		// Otherwise interleave the parents via weaveParents, then reattach
		// this complex's own target to each interleaving.
		var np []*ComplexSelector
		for _, p := range prefixes {
			pp, err := weaveParents(p, c, span)
			if err != nil {
				return nil, err
			}
			if pp == nil {
				continue
			}
			for _, ppc := range pp {
				np = append(np, ppc.WithAdditionalComponent(c.Components[len(c.Components)-1], span, fb))
			}
		}
		prefixes = np
		if prefixes == nil {
			return nil, nil
		}
	}
	return prefixes, nil
}

// deque is a minimal double-ended queue over a slice, standing in for Dart's
// QueueList. Weave consumes parent components from the front (LCS grouping)
// and trailing combinators from the back (mergeTrailingCombinators), so both
// ends need cheap removal.
type deque[T any] struct {
	items []T
}

func newDeque[T any](items []T) *deque[T] {
	c := make([]T, len(items))
	copy(c, items)
	return &deque[T]{items: c}
}

func (q *deque[T]) AddFirst(v T) { q.items = append([]T{v}, q.items...) }
func (q *deque[T]) RemoveFirst() T {
	v := q.items[0]
	q.items = q.items[1:]
	return v
}
func (q *deque[T]) RemoveLast() T {
	v := q.items[len(q.items)-1]
	q.items = q.items[:len(q.items)-1]
	return v
}
func (q *deque[T]) First() T      { return q.items[0] }
func (q *deque[T]) Last() T       { return q.items[len(q.items)-1] }
func (q *deque[T]) Len() int      { return len(q.items) }
func (q *deque[T]) IsEmpty() bool { return len(q.items) == 0 }
func (q *deque[T]) Slice() []T    { return q.items }

// weaveParents interweaves prefix's components with base's components other
// than the last (its target, which the caller reattaches).
//
// It returns every ordering of the parents — including unified mergings —
// that keeps each input's relative order. For example `.foo .bar` with
// `.baz .bang div` yields `.foo .bar .baz .bang div`, `.foo .bar.baz ...`,
// `.foo .baz .bar ...`, and so on through `.baz .bang .foo .bar div`.
// Semantically, for parents P and target-context C, the union of the results
// matches exactly the intersection of C with descendants of P; some orderings
// are elided to bound output size. A nil result means the intersection is
// empty. span marks any combined selectors.
func weaveParents(prefix, base *ComplexSelector, span sasscommon.FileSpan) ([]*ComplexSelector, error) {
	leading := mergeLeadingCombinators(prefix.LeadingCombinators, base.LeadingCombinators)
	if leading == nil {
		return nil, nil
	}

	// Queue only the parents: the prefix is all parents, but base's target
	// must not be woven in.
	q1 := newDeque(prefix.Components)
	q2 := newDeque(base.Components[:len(base.Components)-1])

	// Peel off the trailing combinators first (they constrain the join), then
	// re-append each surviving choice at the end via trailingResult.
	trailingResultIface := mergeTrailingCombinators(q1, q2, span)
	if trailingResultIface == nil {
		return nil, nil
	}
	trailingResult := trailingResultIface.([][][]*ComplexSelectorComponent)

	// Rootish selectors (:root and kin) must open the result. Two of them
	// unify; a lone one is pinned to the front of both queues so every
	// interleaving starts with it.
	switch r1, r2 := firstIfRootish(q1), firstIfRootish(q2); {
	case r1 != nil && r2 != nil:
		rootish, err := unifyCompound(r1.Selector, r2.Selector)
		if err != nil {
			return nil, err
		}
		if rootish == nil {
			return nil, nil
		}
		q1.AddFirst(NewComplexSelectorComponent(rootish, r1.Combinators, r1.Span))
		q2.AddFirst(NewComplexSelectorComponent(rootish, r2.Combinators, r1.Span))
	case r1 != nil:
		q1.AddFirst(r1)
		q2.AddFirst(r1)
	case r2 != nil:
		q1.AddFirst(r2)
		q2.AddFirst(r2)
	}

	g1 := groupSelectors(q1.Slice())
	g2 := groupSelectors(q2.Slice())
	// The longest common subsequence anchors the interleave: equal groups
	// match outright, parent-superselectors collapse to the narrower group,
	// and groups sharing a unique selector (ID, pseudo-element) unify — but
	// only then, since unification is what makes the output exponential.
	lcs := lcsFn(g2.Slice(), g1.Slice(), func(a, b []*ComplexSelectorComponent) ([]*ComplexSelectorComponent, bool) {
		if slicesEqual(a, b) {
			return a, true
		}
		ok, err := complexIsParentSuperselector(a, b)
		if err != nil {
			return nil, false
		}
		if ok {
			return b, true
		}
		ok, err = complexIsParentSuperselector(b, a)
		if err != nil {
			return nil, false
		}
		if ok {
			return a, true
		}
		ok, err = mustUnify(a, b)
		if err != nil {
			return nil, false
		}
		if !ok {
			return nil, false
		}
		unified, err := UnifyComplex([]*ComplexSelector{
			&ComplexSelector{base: NewSelectorBase(span), LeadingCombinators: nil, Components: a, LineBreak: false},
			&ComplexSelector{base: NewSelectorBase(span), LeadingCombinators: nil, Components: b, LineBreak: false},
		}, span)
		if err != nil {
			return nil, false
		}
		if len(unified) == 1 && len(unified[0].Components) > 0 {
			return unified[0].Components, true
		}
		return nil, false
	})

	var choices [][][]*ComplexSelectorComponent
	for _, group := range lcs {
		// Everything before the next shared anchor can interleave freely in
		// either order (chunks); the anchor itself is fixed.
		chunk, err := chunks(g1, g2, func(q *deque[[]*ComplexSelectorComponent]) (bool, error) {
			if q.Len() == 0 {
				return true, nil
			}
			return complexIsParentSuperselector(q.First(), group)
		})
		if err != nil {
			return nil, err
		}
		var positionChoices [][]*ComplexSelectorComponent
		for _, ordering := range chunk {
			var flat []*ComplexSelectorComponent
			for _, g := range ordering {
				flat = append(flat, g...)
			}
			positionChoices = append(positionChoices, flat)
		}
		if len(positionChoices) > 0 {
			choices = append(choices, positionChoices)
		}
		choices = append(choices, [][]*ComplexSelectorComponent{group})
		if g1.Len() > 0 {
			g1.RemoveFirst()
		}
		if g2.Len() > 0 {
			g2.RemoveFirst()
		}
	}

	lastChunk, err := chunks(g1, g2, func(q *deque[[]*ComplexSelectorComponent]) (bool, error) { return q.IsEmpty(), nil })
	if err != nil {
		return nil, err
	}
	var lastPositionChoices [][]*ComplexSelectorComponent
	for _, ordering := range lastChunk {
		var flat []*ComplexSelectorComponent
		for _, g := range ordering {
			flat = append(flat, g...)
		}
		lastPositionChoices = append(lastPositionChoices, flat)
	}
	if len(lastPositionChoices) > 0 {
		choices = append(choices, lastPositionChoices)
	}

	// Trailing-combinator choices close the selector; every path through the
	// accumulated choices becomes one output complex.
	choices = append(choices, trailingResult...)

	var result []*ComplexSelector
	for _, path := range Paths(choices) {
		if len(path) == 0 {
			continue
		}
		var comps []*ComplexSelectorComponent
		for _, g := range path {
			comps = append(comps, g...)
		}
		cs, err := NewComplexSelector(leading, comps, span, prefix.LineBreak || base.LineBreak)
		if err != nil {
			return nil, err
		}
		result = append(result, cs)
	}
	return result, nil
}

// firstIfRootish removes and returns the queue's first component when its
// compound holds a :root-like pseudo-class, which is only meaningful opening
// a complex selector. Anything else leaves the queue untouched and yields nil.
func firstIfRootish(q *deque[*ComplexSelectorComponent]) *ComplexSelectorComponent {
	if q.Len() == 0 {
		return nil
	}
	first := q.First()
	for _, s := range first.Selector.Components {
		if ps, ok := s.(*PseudoSelector); ok && ps.IsClass && rootishPseudoClasses[ps.NormalizedName] {
			q.RemoveFirst()
			return first
		}
	}
	return nil
}

// mergeLeadingCombinators returns the leading combinator list compatible
// with both inputs, or nil when they cannot unify: more than one combinator
// on either side never merges, an empty side defers to the other, and two
// present combinators must be equal.
func mergeLeadingCombinators(c1, c2 []sasscommon.CssValue[Combinator]) []sasscommon.CssValue[Combinator] {
	if len(c1) > 1 || len(c2) > 1 {
		return nil
	}
	if len(c1) == 0 {
		if len(c2) > 0 {
			r := make([]sasscommon.CssValue[Combinator], len(c2))
			copy(r, c2)
			return r
		}
		return []sasscommon.CssValue[Combinator]{}
	}
	if len(c2) == 0 {
		r := make([]sasscommon.CssValue[Combinator], len(c1))
		copy(r, c1)
		return r
	}
	if c1[0].Equal(c2[0]) {
		r := make([]sasscommon.CssValue[Combinator], len(c1))
		copy(r, c1)
		return r
	}
	return nil
}

// mergeTrailingCombinators pops trailing components with combinators off c1
// and c2 and merges them into one choice list per position. Each returned
// element holds the alternative orderings for one slot of the final complex;
// the union over all paths through those choices matches every required
// element. An empty result means no combinators needed merging; nil means the
// sequences cannot merge. span marks any unified components built along the
// way. The static type is any (Dart returns a nullable list); callers assert
// it to [][][]*ComplexSelectorComponent, mirroring Dart's addFirst ordering
// via the slice reversal at the end. The pairings below are special cases,
// not a general rule: `~` with `~` keeps the narrower side (or both orders,
// plus the unification when the compounds merge); `~` with `+` keeps the
// adjacent sibling unless it is already covered; `>` outranks a sibling
// combinator, whose component simply survives; equal combinators unify the
// compounds; and a combinator against a bare descendant drops the redundant
// ancestor when it is already subsumed.
func mergeTrailingCombinators(c1, c2 *deque[*ComplexSelectorComponent], span sasscommon.FileSpan) any {
	stack := [][][]*ComplexSelectorComponent{}

	for {
		var comb1, comb2 []sasscommon.CssValue[Combinator]
		if c1.Len() > 0 {
			comb1 = c1.Last().Combinators
		}
		if c2.Len() > 0 {
			comb2 = c2.Last().Combinators
		}
		if len(comb1) == 0 && len(comb2) == 0 {
			break
		}
		if len(comb1) > 1 || len(comb2) > 1 {
			return nil
		}

		has1 := len(comb1) > 0
		has2 := len(comb2) > 0
		var v1, v2 Combinator
		if has1 {
			v1 = comb1[0].Value
		}
		if has2 {
			v2 = comb2[0].Value
		}

		switch {
		case has1 && v1 == CombinatorFollowingSibling && has2 && v2 == CombinatorFollowingSibling:
			comp1 := c1.RemoveLast()
			comp2 := c2.RemoveLast()
			ok, err := comp1.Selector.IsSuperselector(comp2.Selector)
			if err != nil {
				return nil
			}
			if ok {
				stack = append(stack, [][]*ComplexSelectorComponent{{comp2}})
			} else {
				ok, err = comp2.Selector.IsSuperselector(comp1.Selector)
				if err != nil {
					return nil
				}
				if ok {
					stack = append(stack, [][]*ComplexSelectorComponent{{comp1}})
				} else {
					choices := [][]*ComplexSelectorComponent{{comp1, comp2}, {comp2, comp1}}
					unified, err := unifyCompound(comp1.Selector, comp2.Selector)
					if err != nil {
						return nil
					}
					if unified != nil {
						choices = append(choices, []*ComplexSelectorComponent{NewComplexSelectorComponent(unified, comb1, span)})
					}
					stack = append(stack, choices)
				}
			}

		case (has1 && v1 == CombinatorFollowingSibling && has2 && v2 == CombinatorNextSibling) ||
			(has1 && v1 == CombinatorNextSibling && has2 && v2 == CombinatorFollowingSibling):
			var nc, fc *deque[*ComplexSelectorComponent]
			if v1 == CombinatorNextSibling {
				nc, fc = c1, c2
			} else {
				nc, fc = c2, c1
			}
			next := nc.RemoveLast()
			following := fc.RemoveLast()
			ok, err := following.Selector.IsSuperselector(next.Selector)
			if err != nil {
				return nil
			}
			if ok {
				stack = append(stack, [][]*ComplexSelectorComponent{{next}})
			} else {
				choices := [][]*ComplexSelectorComponent{{following, next}}
				unified, err := unifyCompound(following.Selector, next.Selector)
				if err != nil {
					return nil
				}
				if unified != nil {
					choices = append(choices, []*ComplexSelectorComponent{NewComplexSelectorComponent(unified, next.Combinators, span)})
				}
				stack = append(stack, choices)
			}

		case has1 && v1 == CombinatorChild && has2 && (v2 == CombinatorNextSibling || v2 == CombinatorFollowingSibling):
			stack = append(stack, [][]*ComplexSelectorComponent{{c2.RemoveLast()}})

		case has2 && v2 == CombinatorChild && has1 && (v1 == CombinatorNextSibling || v1 == CombinatorFollowingSibling):
			stack = append(stack, [][]*ComplexSelectorComponent{{c1.RemoveLast()}})

		case has1 && has2 && v1 == v2:
			unified, err := unifyCompound(c1.RemoveLast().Selector, c2.RemoveLast().Selector)
			if err != nil {
				return nil
			}
			if unified == nil {
				return nil
			}
			stack = append(stack, [][]*ComplexSelectorComponent{{NewComplexSelectorComponent(unified, comb1, span)}})

		case has1 && !has2:
			dc := c2
			cc := c1
			if v1 == CombinatorChild && dc.Len() > 0 {
				ok, err := dc.Last().Selector.IsSuperselector(cc.Last().Selector)
				if err != nil {
					return nil
				}
				if ok {
					dc.RemoveLast()
				}
			}
			stack = append(stack, [][]*ComplexSelectorComponent{{cc.RemoveLast()}})

		case !has1 && has2:
			dc := c1
			cc := c2
			if v2 == CombinatorChild && dc.Len() > 0 {
				ok, err := dc.Last().Selector.IsSuperselector(cc.Last().Selector)
				if err != nil {
					return nil
				}
				if ok {
					dc.RemoveLast()
				}
			}
			stack = append(stack, [][]*ComplexSelectorComponent{{cc.RemoveLast()}})

		default:
			return nil
		}
	}

	// Choices accumulate back-to-front (Dart prepends with addFirst), so the
	// stack reverses into front-to-back order before returning.
	slices.Reverse(stack)
	return stack
}

// groupSelectors splits comps into the longest runs where only the final
// component lacks a combinator. For example `(A B > C D + E ~ G)` groups as
// `[(A) (B > C) (D + E ~ G)]`, so the LCS below compares combinator-bound
// units rather than lone components.
func groupSelectors(comps []*ComplexSelectorComponent) *deque[[]*ComplexSelectorComponent] {
	groups := newDeque[[]*ComplexSelectorComponent](nil)
	var g []*ComplexSelectorComponent
	for _, c := range comps {
		g = append(g, c)
		if len(c.Combinators) == 0 {
			groups.items = append(groups.items, g)
			g = nil
		}
	}
	if len(g) > 0 {
		groups.items = append(groups.items, g)
	}
	return groups
}

// chunks returns every ordering of the head subsequences of q1 and q2, where
// done marks each head's extent; both queues are drained through those marks
// as a side effect. Given heads `(A B C)` and `(1 2)` this yields
// `[(A B C 1 2) (1 2 A B C)]`, leaving the tails behind. An empty head on one
// side yields the other head alone; two empty heads yield nothing.
func chunks[T any](q1, q2 *deque[T], done func(*deque[T]) (bool, error)) ([][]T, error) {
	var c1 []T
	for {
		ok, err := done(q1)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
		c1 = append(c1, q1.RemoveFirst())
	}
	var c2 []T
	for {
		ok, err := done(q2)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
		c2 = append(c2, q2.RemoveFirst())
	}
	switch {
	case len(c1) == 0 && len(c2) == 0:
		return nil, nil
	case len(c1) == 0:
		return [][]T{c2}, nil
	case len(c2) == 0:
		return [][]T{c1}, nil
	default:
		a := make([]T, 0, len(c1)+len(c2))
		a = append(a, c1...)
		a = append(a, c2...)
		b := make([]T, 0, len(c1)+len(c2))
		b = append(b, c2...)
		b = append(b, c1...)
		return [][]T{a, b}, nil
	}
}

// Paths returns every path through choices, picking one option per position.
// Given `[[1 2] [3 4] [5]]` it yields `[[1 3 5] [2 3 5] [1 4 5] [2 4 5]]` —
// the cartesian product weaveParents uses to turn per-position orderings into
// full complex selectors.
func Paths[T any](choices [][]T) [][]T {
	paths := [][]T{{}}
	for _, choice := range choices {
		var np [][]T
		for _, opt := range choice {
			for _, p := range paths {
				n := make([]T, len(p)+1)
				copy(n, p)
				n[len(p)] = opt
				np = append(np, n)
			}
		}
		paths = np
	}
	return paths
}

// complexIsParentSuperselector reports whether a acts as a superselector of b
// once both share an implicit base. `B` alone is no superselector of `B A`
// (it misses `A`), but it is a *parent* superselector, since `B X` covers
// `B A X`. The shared bogus placeholder base stands in for that X without
// matching anything real.
func complexIsParentSuperselector(a, b []*ComplexSelectorComponent) (bool, error) {
	if len(a) > len(b) {
		return false, nil
	}
	n := "<temp>"
	compound, err := NewCompoundSelector([]SimpleSelector{NewPlaceholderSelector(n, sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	if err != nil {
		return false, err
	}
	base := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	x := make([]*ComplexSelectorComponent, 0, len(a)+1)
	x = append(x, a...)
	x = append(x, base)
	y := make([]*ComplexSelectorComponent, 0, len(b)+1)
	y = append(y, b...)
	y = append(y, base)
	return complexIsSuperselector(x, y)
}

// mustUnify reports whether two parent groups share a unique simple selector
// (same ID or pseudo-element value), which forces them to merge rather than
// merely interleave. An element has only one ID, so `.a#b` woven with `#b.c`
// must produce `#b`-unified output instead of both orders.
func mustUnify(a, b []*ComplexSelectorComponent) (bool, error) {
	uniq := linkedhashmap.NewLinkedHashSet[SimpleSelector](SimpleSelectorEquals)
	for _, c := range a {
		for _, s := range c.Selector.Components {
			if isUnique(s) {
				uniq.Add(s)
			}
		}
	}
	if uniq.Len() == 0 {
		return false, nil
	}
	for _, c := range b {
		for _, s := range c.Selector.Components {
			if isUnique(s) {
				if uniq.Contains(s) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// isUnique reports whether at most one simple selector of s's kind may appear
// in a compound: IDs and pseudo-elements. Shared IDs are what trigger
// mustUnify above.
func isUnique(s SimpleSelector) bool {
	if _, ok := s.(*IDSelector); ok {
		return true
	}
	if ps, ok := s.(*PseudoSelector); ok && ps.IsElement() {
		return true
	}
	return false
}

// lcsSelection caches one select() verdict: the merged group on a match (ok)
// or nothing on a miss. The table is dense over both group lists so the
// backtrack below replays match decisions without re-running unification.
type lcsSelection[T any] struct {
	value T
	ok    bool
}

// lcsFn is the longest-common-subsequence core weaveParents runs over grouped
// parent components (Dart: longestCommonSubsequence in lib/src/utils.dart).
// sel decides whether two groups anchor together — returning the surviving
// group on a match — and the backtrack prefers earlier matches, keeping
// shared ancestors as far left as the inputs allow.
func lcsFn[T any](list1, list2 []T, sel func(T, T) (T, bool)) []T {
	m, n := len(list1), len(list2)

	lengths := make([][]int, m+1)
	for i := range lengths {
		lengths[i] = make([]int, n+1)
	}

	selections := make([][]lcsSelection[T], m)
	for i := range selections {
		selections[i] = make([]lcsSelection[T], n)
	}

	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			selection, ok := sel(list1[i], list2[j])
			selections[i][j] = lcsSelection[T]{selection, ok}
			if ok {
				lengths[i+1][j+1] = lengths[i][j] + 1
			} else {
				lengths[i+1][j+1] = max(lengths[i+1][j], lengths[i][j+1])
			}
		}
	}

	var backtrack func(i, j int) []T
	backtrack = func(i, j int) []T {
		if i == -1 || j == -1 {
			return nil
		}
		selection := selections[i][j]
		if selection.ok {
			return append(backtrack(i-1, j-1), selection.value)
		}
		if lengths[i+1][j] > lengths[i][j+1] {
			return backtrack(i, j-1)
		}
		return backtrack(i-1, j)
	}

	return backtrack(m-1, n-1)
}

// ListIsSuperselector reports whether list a matches every element matched
// by list b (and possibly more): each complex in b needs at least one complex
// in a covering it. @extend uses this to drop selectors already subsumed by
// the extender.
func ListIsSuperselector(a, b []*ComplexSelector) (bool, error) {
	for _, cb := range b {
		found := false
		for _, ca := range a {
			ok, err := ca.IsSuperselector(cb)
			if err != nil {
				return false, err
			}
			if ok {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

// ComplexIsSuperselector reports whether complex a matches every element
// matched by b. It walks a's components left to right, sliding each one along
// b until its compound covers some component, then checks the combinators
// line up (a bare descendant covers a child `>`, `~` covers `+`, but never
// the reverse). Trailing combinators on either side disqualify immediately —
// an unfinished selector is neither a super- nor a sub-selector.
func ComplexIsSuperselector(a, b []*ComplexSelectorComponent) (bool, error) {
	if len(a) == 0 || len(b) == 0 {
		return false, nil
	}
	if len(a[len(a)-1].Combinators) > 0 {
		return false, nil
	}
	if len(b[len(b)-1].Combinators) > 0 {
		return false, nil
	}
	i1, i2 := 0, 0
	var prev *sasscommon.CssValue[Combinator]
	for {
		r1, r2 := len(a)-i1, len(b)-i2
		if r1 == 0 || r2 == 0 {
			return false, nil
		}
		if r1 > r2 {
			return false, nil
		}
		ca := a[i1]
		if len(ca.Combinators) > 1 {
			return false, nil
		}
		if r1 == 1 {
			if anyMultiCombinator(b) {
				return false, nil
			}
			var parents []*ComplexSelectorComponent
			if ca.Selector.HasComplicatedSuperselectorSemantics() {
				parents = b[i2 : len(b)-1]
			}
			return CompoundIsSuperselector(ca.Selector, b[len(b)-1].Selector, parents...)
		}
		eos := i2
		for {
			cb := b[eos]
			if len(cb.Combinators) > 1 {
				return false, nil
			}
			var parents []*ComplexSelectorComponent
			if ca.Selector.HasComplicatedSuperselectorSemantics() {
				parents = b[i2:eos]
			}
			ok, err := CompoundIsSuperselector(ca.Selector, cb.Selector, parents...)
			if err != nil {
				return false, err
			}
			if ok {
				break
			}
			eos++
			if eos == len(b)-1 {
				return false, nil
			}
		}
		if !compatPrevCombinator(prev, b[i2:eos]) {
			return false, nil
		}
		cb := b[eos]
		var combA, combB *sasscommon.CssValue[Combinator]
		if len(ca.Combinators) > 0 {
			combA = &ca.Combinators[0]
		}
		if len(cb.Combinators) > 0 {
			combB = &cb.Combinators[0]
		}
		if !isSupercombinator(combA, combB) {
			return false, nil
		}
		i1++
		i2 = eos + 1
		prev = combA
		if len(a)-i1 == 1 {
			if combA != nil && combA.Value == CombinatorFollowingSibling {
				for _, c := range b[i2 : len(b)-1] {
					var c2 *sasscommon.CssValue[Combinator]
					if len(c.Combinators) > 0 {
						c2 = &c.Combinators[0]
					}
					if !isSupercombinator(combA, c2) {
						return false, nil
					}
				}
			} else if combA != nil && len(b)-i2 > 1 {
				return false, nil
			}
		}
	}
}

// anyMultiCombinator reports whether any component carries more than one
// combinator. Doubled-up combinators (`.foo + ~ .bar`, kept for
// backwards compatibility) bail the superselector walk out: their semantics
// are too murky to reason about.
func anyMultiCombinator(comps []*ComplexSelectorComponent) bool {
	for _, c := range comps {
		if len(c.Combinators) > 1 {
			return true
		}
	}
	return false
}

// compatPrevCombinator reports whether parents may sit between two matched
// components given the earlier combinator prev. Only the following-sibling
// combinator tolerates interlopers — and even then solely siblings (`~`/`+`):
// `>` and bare descendants demand adjacency.
func compatPrevCombinator(prev *sasscommon.CssValue[Combinator], parents []*ComplexSelectorComponent) bool {
	if len(parents) == 0 || prev == nil {
		return true
	}
	if prev.Value != CombinatorFollowingSibling {
		return false
	}
	for _, p := range parents {
		if len(p.Combinators) == 0 {
			return false
		}
		v := p.Combinators[0].Value
		if v != CombinatorFollowingSibling && v != CombinatorNextSibling {
			return false
		}
	}
	return true
}

// isSupercombinator reports whether `X a Y` covers `X b Y`: equal
// combinators always do, a bare descendant covers a child (`A B` matches
// everything `A > B` does), and `~` covers `+`. Anything else is unrelated.
func isSupercombinator(a, b *sasscommon.CssValue[Combinator]) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil && a.Equal(*b) {
		return true
	}
	if a == nil && b != nil && b.Value == CombinatorChild {
		return true
	}
	if a != nil && a.Value == CombinatorFollowingSibling && b != nil && b.Value == CombinatorNextSibling {
		return true
	}
	return false
}

// CompoundIsSuperselector reports whether compound a matches every element
// matched by b. Plain compounds compare component-wise; compounds with
// pseudo-elements split around the element (which must itself unify —
// pseudo-elements retarget rather than narrow, so both sides need the same
// one); and selector-bearing pseudos (`:not`, `:is`, ...) delegate to
// selectorPseudoIsSuperselector. parents carries b's ancestors for pseudos
// whose arguments must match against them, such as :has().
func CompoundIsSuperselector(a, b *CompoundSelector, parents ...*ComplexSelectorComponent) (bool, error) {
	if !a.HasComplicatedSuperselectorSemantics() && !b.HasComplicatedSuperselectorSemantics() {
		if len(a.Components) > len(b.Components) {
			return false, nil
		}
		for _, sa := range a.Components {
			found := false
			for _, sb := range b.Components {
				ok, err := sa.IsSuperselector(sb)
				if err != nil {
					return false, err
				}
				if ok {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}

	pa, pb := findPseudoElement(a), findPseudoElement(b)
	if pa != nil && pb != nil {
		ok, err := compCompsIsSuperselector(a.Components[:pa.Index], b.Components[:pb.Index], parents...)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		pseudoSup, err := pa.Pseudo.IsSuperselector(pb.Pseudo)
		if err != nil {
			return false, err
		}
		if !pseudoSup {
			return false, nil
		}
		return compCompsIsSuperselector(a.Components[pa.Index+1:], b.Components[pb.Index+1:], parents...)
	}
	if pa != nil || pb != nil {
		return false, nil
	}

	for _, sa := range a.Components {
		if ps, ok := sa.(*PseudoSelector); ok && ps.Selector != nil {
			ok, err := selectorPseudoIsSuperselector(ps, b, parents...)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		} else {
			ok, err := compContains(b.Components, sa)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}
	return true, nil
}

// pseudoIdx pairs a compound's first pseudo-element with its position, so
// CompoundIsSuperselector can split the compounds around it.
type pseudoIdx struct {
	Pseudo *PseudoSelector
	Index  int
}

// findPseudoElement returns the compound's first pseudo-element, if any.
// Only the first matters: a valid compound holds at most one, and the split
// logic treats everything after it as trailing pseudos.
func findPseudoElement(c *CompoundSelector) *pseudoIdx {
	for i, s := range c.Components {
		if ps, ok := s.(*PseudoSelector); ok && ps.IsElement() {
			return &pseudoIdx{Pseudo: ps, Index: i}
		}
	}
	return nil
}

// compCompsIsSuperselector is CompoundIsSuperselector over raw simple lists,
// used for the pre/post-pseudo-element slices. An empty a covers anything;
// an empty b behaves as the universal selector, matching the bogus-span
// compounds Dart builds for the same comparison.
func compCompsIsSuperselector(a, b []SimpleSelector, parents ...*ComplexSelectorComponent) (bool, error) {
	if len(a) == 0 {
		return true, nil
	}
	if len(b) == 0 {
		star := "*"
		b = []SimpleSelector{NewUniversalSelector(sasscommon.BogusSpan, &star)}
	}
	ca, err := NewCompoundSelector(a, sasscommon.BogusSpan)
	if err != nil {
		return false, err
	}
	cb, err := NewCompoundSelector(b, sasscommon.BogusSpan)
	if err != nil {
		return false, err
	}
	return CompoundIsSuperselector(ca, cb, parents...)
}

// compContains reports whether any member of comps covers target — the inner
// loop of the plain-compound fast path in CompoundIsSuperselector.
func compContains(comps []SimpleSelector, target SimpleSelector) (bool, error) {
	for _, s := range comps {
		ok, err := target.IsSuperselector(s)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// selectorPseudoIsSuperselector reports whether the selector-bearing pseudo p
// covers compound c2. Each pseudo family has its own rule: :is/:where-style
// pseudos win when their argument covers one of c2's arguments, or when one
// of their complexes covers c2 in situ (with parents in scope); :not inverts
// the test per negated complex (a bogus negation poisons the whole check);
// :has/:host/:slotted compare argument lists directly; :current needs an
// exactly equal argument; :nth-child also pins the raw argument string. A
// pseudo without a selector argument is a caller bug and errors.
func selectorPseudoIsSuperselector(p *PseudoSelector, c2 *CompoundSelector, parents ...*ComplexSelectorComponent) (bool, error) {
	sel := p.Selector
	if sel == nil {
		return false, &sasscommon.SassScriptException{Message: fmt.Sprintf("Selector %q must have a selector argument", p.Name)}
	}
	l1, ok := sel.(*SelectorList)
	if !ok {
		return false, nil
	}

	switch p.NormalizedName {
	case "is", "matches", "any", "where":
		for _, arg := range selectorPseudoArgs(c2, p.Name, nil) {
			ok, err := l1.IsSuperselector(arg)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		for _, c := range l1.Components {
			if len(c.LeadingCombinators) == 0 {
				span, err := c2.Span()
				if err != nil {
					return false, err
				}
				comps := make([]*ComplexSelectorComponent, 0, len(parents)+1)
				comps = append(comps, parents...)
				comps = append(comps, NewComplexSelectorComponent(c2, nil, span))
				ok, err := complexIsSuperselector(c.Components, comps)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
		}
		return false, nil

	case "has", "host", "host-context":
		for _, arg := range selectorPseudoArgs(c2, p.Name, nil) {
			ok, err := l1.IsSuperselector(arg)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil

	case "slotted":
		for _, arg := range selectorPseudoArgs(c2, p.Name, new(bool)) {
			ok, err := l1.IsSuperselector(arg)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil

	case "not":
		for _, c := range l1.Components {
			if c.IsBogus() {
				return false, nil
			}
			found := false
			for _, sb := range c2.Components {
				switch sb := sb.(type) {
				case *TypeSelector:
					for _, sa := range c.Components[len(c.Components)-1].Selector.Components {
						if ta, ok := sa.(*TypeSelector); ok && (ta.Name.Name != sb.Name.Name || !ptrEq(ta.Name.Namespace, sb.Name.Namespace)) {
							found = true
						}
					}
				case *IDSelector:
					for _, sa := range c.Components[len(c.Components)-1].Selector.Components {
						if id, ok := sa.(*IDSelector); ok && id.Name != sb.Name {
							found = true
						}
					}
				case *PseudoSelector:
					if sb.Selector != nil && sb.Name == p.Name {
						if l2, ok := sb.Selector.(*SelectorList); ok {
							ok, err := ListIsSuperselector(l2.Components, []*ComplexSelector{c})
							if err != nil {
								return false, err
							}
							if ok {
								found = true
							}
						}
					}
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil

	case "current":
		if slices.ContainsFunc(selectorPseudoArgs(c2, p.Name, nil), l1.Equals) {
			return true, nil
		}
		return false, nil

	case "nth-child", "nth-last-child":
		for _, sb := range c2.Components {
			if ps, ok := sb.(*PseudoSelector); ok && ps.Name == p.Name && ptrEq(ps.Argument, p.Argument) {
				if ps.Selector != nil {
					if l2, ok := ps.Selector.(*SelectorList); ok {
						return l1.IsSuperselector(l2)
					}
				}
			}
		}
		return false, nil

	default:
		panic("unreachable")
	}
}

// selectorPseudoArgs collects the selector arguments of every pseudo in c
// with the given name and class sense. isClass defaults to true (functional
// pseudo-classes); :slotted passes false since its argument hangs off a
// pseudo-element. Only pseudos carrying a SelectorList contribute.
func selectorPseudoArgs(c *CompoundSelector, name string, isClass *bool) []*SelectorList {
	class := true
	if isClass != nil {
		class = *isClass
	}
	var r []*SelectorList
	for _, s := range c.Components {
		if ps, ok := s.(*PseudoSelector); ok && ps.IsClass == class && ps.Name == name && ps.Selector != nil {
			if l, ok := ps.Selector.(*SelectorList); ok {
				r = append(r, l)
			}
		}
	}
	return r
}

// ptrEq compares optional namespace/name strings by value, treating two nils
// as equal. Plain == would compare pointer identity, which aliasing makes
// meaningless.
func ptrEq(a, b *string) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// slicesEqual compares component groups by value equality, backing the LCS
// anchor test in weaveParents. Component identity is irrelevant — two
// separately parsed `.foo` components anchor together.
func slicesEqual(a, b []*ComplexSelectorComponent) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
