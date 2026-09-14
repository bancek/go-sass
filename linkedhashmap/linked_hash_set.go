package linkedhashmap

import "iter"

// LinkedHashSet is an insertion-ordered set for non-comparable keys.
// Keys must implement HashCoder (HashCode() int); equality is supplied
// at construction.
//
// A nil *LinkedHashSet acts as an empty, immutable set — matching
// Dart's nullable Set<E>? pattern (null means no tracking needed).
//
// Matches Dart: LinkedHashSet<E>
type LinkedHashSet[K HashCoder] struct {
	m *LinkedHashMap[K, struct{}]
}

// NewLinkedHashSet creates a new LinkedHashSet with the given equality function.
func NewLinkedHashSet[K HashCoder](equals func(K, K) bool) *LinkedHashSet[K] {
	return &LinkedHashSet[K]{m: New[K, struct{}](equals)}
}

// Add inserts k into the set. No-op if s is nil.
func (s *LinkedHashSet[K]) Add(k K) {
	if s == nil {
		return
	}
	s.m.Put(k, struct{}{})
}

// Contains returns true if k is in the set. Returns false if s is nil.
func (s *LinkedHashSet[K]) Contains(k K) bool {
	if s == nil {
		return false
	}
	return s.m.Has(k)
}

// Len returns the number of elements. Returns 0 if s is nil.
func (s *LinkedHashSet[K]) Len() int {
	if s == nil {
		return 0
	}
	return s.m.Len()
}

// Keys returns an iterator over the elements in insertion order.
// Returns nil if s is nil.
func (s *LinkedHashSet[K]) Keys() iter.Seq[K] {
	if s == nil {
		return nil
	}
	return s.m.Keys()
}
