// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"iter"

	"github.com/bancek/go-sass/linkedhashmap"
)

// dart-source: lib/src/value/map.dart

// MapEntry is a single key-value pair in a SassMap.
type MapEntry struct {
	Key   Value
	Value Value
}

// SassMap is a SassScript map value.
//
// Entries preserve insertion order in a slice, while an inner keyed map gives
// O(1) structural lookup by value equality (never pointer identity). Dart
// notes a wish to move to persistent data structures; Go keeps the dual
// slice-plus-map shape instead.
type SassMap struct {
	inner      *linkedhashmap.LinkedHashMap[Value, Value]
	entries    []MapEntry
	cachedHash *int
}

// EmptySassMap returns an empty map.
// It ports Dart's const SassMap.empty constructor.
// Matches Dart: SassMap.empty
func EmptySassMap() *SassMap {
	return &SassMap{inner: linkedhashmap.New[Value, Value](ValueEquals)}
}

// NewSassMap creates a new SassMap from a Go map.
//
// Duplicate keys by value equality keep their first occurrence, matching
// Dart's Map.unmodifiableOf behavior of preserving insertion order. (Go map
// iteration itself is unordered, so callers needing a stable order should use
// NewSassMapFromEntries instead.)
func NewSassMap(contents map[Value]Value) *SassMap {
	m := &SassMap{inner: linkedhashmap.New[Value, Value](ValueEquals)}
	for k, v := range contents {
		if m.inner.Has(k) {
			continue
		}
		m.inner.Put(k, v)
		m.entries = append(m.entries, MapEntry{k, v})
	}
	return m
}

func (m *SassMap) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitMap(m) }
func (m *SassMap) isValue()                                              {}
func (m *SassMap) TryMap() *SassMap                                      { return m }

// ToCssString returns the CSS rendering of this map. Maps have no plain-CSS
// form, so this always errors; use String for the inspect rendering.
// (Dart: Value.toCssString, which throws for maps.)
func (m *SassMap) ToCssString(quote bool) (string, error) { return SerializeValue(m, quote) }

// String returns the inspect rendering of this map (Dart toString): comma
// separated `key: value` pairs, rendering `()` when empty.
func (m *SassMap) String() (string, error) { return SerializeValueInspect(m) }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Maps are always truthy, even when empty.
func (m *SassMap) IsTruthy() bool { return true }

// Separator reports this value's list separator: undecided when empty, comma
// otherwise, since a map counts as a list of pairs. It ports Dart's
// SassMap.separator.
func (m *SassMap) Separator() ListSeparator {
	if len(m.entries) == 0 {
		return ListSeparatorUndecided
	}
	return ListSeparatorComma
}
func (m *SassMap) HasBrackets() bool                       { return false }
func (m *SassMap) LengthAsList() int                       { return len(m.entries) }
func (m *SassMap) IsBlank() bool                           { return false }
func (m *SassMap) IsSpecialNumber() bool                   { return false }
func (m *SassMap) IsSpecialVariable() bool                 { return false }
func (m *SassMap) RealNull() Value                         { return DefaultRealNull(m) }
func (m *SassMap) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(m, other) }
func (m *SassMap) Plus(other Value) (Value, error)         { return DefaultPlus(m, other) }
func (m *SassMap) Minus(other Value) (Value, error)        { return DefaultMinus(m, other) }
func (m *SassMap) Times(other Value) (Value, error)        { return DefaultTimes(m, other) }
func (m *SassMap) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(m, other) }
func (m *SassMap) Modulo(other Value) (Value, error)       { return DefaultModulo(m, other) }
func (m *SassMap) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(m, other) }
func (m *SassMap) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(m, other)
}
func (m *SassMap) LessThan(other Value) (Value, error) { return DefaultLessThan(m, other) }
func (m *SassMap) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(m, other)
}
func (m *SassMap) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(m) }
func (m *SassMap) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(m) }
func (m *SassMap) UnaryDivide() (Value, error) { return DefaultUnaryDivide(m) }
func (m *SassMap) UnaryNot() (Value, error)    { return DefaultUnaryNot(m) }

// Equals reports whether other holds the same key-value pairs by value
// equality, regardless of insertion order.
//
// An empty map also equals an empty list (and argument list), since `()`
// doubles as the empty map in Sass.
//
// Matches Dart: SassMap.== (plus the empty-list equivalence).
func (m *SassMap) Equals(other Value) bool {
	if len(m.entries) == 0 {
		switch other.(type) {
		case *SassList, *SassArgumentList:
			return other.LengthAsList() == 0
		}
	}
	if om, ok := other.(*SassMap); ok {
		if len(m.entries) != len(om.entries) {
			return false
		}
		for _, ea := range m.entries {
			v, ok := om.inner.Get(ea.Key)
			if !ok || !ea.Value.Equals(v) {
				return false
			}
		}
		return true
	}
	return false
}

// AsList returns the map as a list of two-element space-separated (key, value)
// pair lists, preserving insertion order. It ports Dart's SassMap.asList.
func (m *SassMap) AsList() ([]Value, error) {
	result := make([]Value, 0, len(m.entries))
	for _, e := range m.entries {
		pairList, err := NewSassList([]Value{e.Key, e.Value}, ListSeparatorSpace, false)
		if err != nil {
			return nil, err
		}
		result = append(result, pairList)
	}
	return result, nil
}

// Entries returns an iterator over key-value pairs in insertion order,
// skipping deleted keys.
func (m *SassMap) Entries() iter.Seq2[Value, Value] {
	return func(yield func(Value, Value) bool) {
		for _, entry := range m.entries {
			if !yield(entry.Key, entry.Value) {
				return
			}
		}
	}
}

// Get looks up key using O(1) structural equality via HashCode()/Equal().
func (m *SassMap) Get(key Value) (Value, bool) {
	return m.inner.Get(key)
}

// Contains returns true if the map contains the given key.
func (m *SassMap) Contains(key Value) bool {
	return m.inner.Has(key)
}

// Set adds or updates a key-value pair using O(1) structural equality.
//
// Updating an existing key rewrites the entry in place so insertion order is
// preserved; only fresh keys append. The cached hash is discarded.
func (m *SassMap) Set(key, val Value) {
	if m.inner.Has(key) {
		for i, e := range m.entries {
			if e.Key.Equals(key) {
				m.entries[i].Value = val
				break
			}
		}
		m.inner.Put(key, val)
	} else {
		m.inner.Put(key, val)
		m.entries = append(m.entries, MapEntry{key, val})
	}
	m.cachedHash = nil
}

// Copy returns a copy of the map, preserving insertion order.
//
// Both the ordered entries and the lookup index are duplicated, so mutating
// the copy never affects the original.
func (m *SassMap) Copy() *SassMap {
	copied := make([]MapEntry, len(m.entries))
	copy(copied, m.entries)
	c := &SassMap{entries: copied, inner: m.inner.Copy()}
	return c
}

// Delete removes a key using O(1) structural equality.
//
// The entry is spliced out of the ordered slice so later entries keep their
// relative order; the cached hash is discarded.
func (m *SassMap) Delete(key Value) {
	if !m.inner.Has(key) {
		return
	}
	m.inner.Delete(key)
	for i, e := range m.entries {
		if e.Key.Equals(key) {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			break
		}
	}
	m.cachedHash = nil
}

// NewSassMapFromEntries creates a new SassMap from an iterator of key-value pairs.
//
// Pairs apply in order and duplicate keys keep their first occurrence, so
// unlike NewSassMap the resulting order is fully deterministic.
func NewSassMapFromEntries(entries iter.Seq2[Value, Value]) *SassMap {
	m := &SassMap{inner: linkedhashmap.New[Value, Value](ValueEquals)}
	for k, v := range entries {
		if m.inner.Has(k) {
			continue
		}
		m.inner.Put(k, v)
		m.entries = append(m.entries, MapEntry{k, v})
	}
	return m
}

// HashCode returns a hash matching Equal semantics.
//
// An empty map hashes like the empty list (they compare equal); otherwise the
// hash folds the entries. It is cached until the next Set or Delete.
// Matches Dart: SassMap.hashCode => contents.isEmpty ? SassList.empty().hashCode : mapHash(contents).
func (m *SassMap) HashCode() int {
	if m.cachedHash != nil {
		return *m.cachedHash
	}
	var h int
	if len(m.entries) == 0 {
		h = (&SassList{}).HashCode()
	} else {
		h = mapHash(m.entries)
	}
	m.cachedHash = &h
	return h
}
