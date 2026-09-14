// Package orderedset provides an insert-order preserving set.
//
// Keys are maintained in insertion order. A nil *LinkedSet is a valid empty,
// immutable set (Contains returns false, Len returns 0, Add returns false).
//
// Backed by orderedmap.LinkedMap[K, struct{}]; inherits insertion-order
// iteration and nil-safety from it.
package orderedset

import (
	"iter"

	"github.com/bancek/go-sass/orderedmap"
)

// LinkedSet is a set that preserves key insertion order for iteration.
//
// A nil *LinkedSet acts as an empty set. This allows pointer fields to
// serve as nullable sets — nil means "no restriction" (all items allowed).
//
// Matches Dart: LinkedHashSet<E>
type LinkedSet[K comparable] struct {
	m *orderedmap.LinkedMap[K, struct{}]
}

// New creates a new empty ordered set.
func New[K comparable]() *LinkedSet[K] {
	return &LinkedSet[K]{m: orderedmap.New[K, struct{}]()}
}

// NewFromSlice creates a new ordered set from the given items,
// preserving order and deduplicating.
func NewFromSlice[K comparable](items []K) *LinkedSet[K] {
	s := &LinkedSet[K]{m: orderedmap.NewWithCapacity[K, struct{}](len(items))}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Contains returns whether key is in the set.
func (s *LinkedSet[K]) Contains(key K) bool {
	if s == nil || s.m == nil {
		return false
	}
	return s.m.Has(key)
}

// Len returns the number of elements.
func (s *LinkedSet[K]) Len() int {
	if s == nil || s.m == nil {
		return 0
	}
	return s.m.Len()
}

// Keys returns an iterator over keys in insertion order.
func (s *LinkedSet[K]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if s == nil || s.m == nil {
			return
		}
		for k := range s.m.Keys() {
			if !yield(k) {
				return
			}
		}
	}
}

// Map returns the underlying set map for O(1) membership tests.
// Returns nil if the receiver is nil (matching old StringSet behavior).
func (s *LinkedSet[K]) Map() map[K]struct{} {
	if s == nil || s.m == nil {
		return nil
	}
	return s.m.ToMap()
}

// Add adds a key to the set.
// Returns true if the key was newly added.
func (s *LinkedSet[K]) Add(key K) bool {
	if s == nil {
		return false
	}
	if s.m.Has(key) {
		return false
	}
	s.m.Put(key, struct{}{})
	return true
}

// Delete removes a key from the set.
func (s *LinkedSet[K]) Delete(key K) {
	if s == nil || s.m == nil {
		return
	}
	s.m.Delete(key)
}
