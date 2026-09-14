// Package orderedmap provides an insert-order preserving map.
//
// Keys are maintained in insertion order. When a key is deleted, it
// is removed from the underlying Go map but retained in the key slice;
// iteration methods skip deleted keys.
package orderedmap

import (
	"iter"
	"maps"
)

// LinkedMap is a map that preserves key insertion order for iteration.
//
// Deleting a key removes it from lookups but does not remove it from
// the insertion-order slice; Keys/All iteration skip deleted keys.
// Re-inserting a previously deleted key reuses its original position.
type LinkedMap[K comparable, V any] struct {
	m    map[K]V
	keys []K
}

// New creates a new ordered map.
func New[K comparable, V any]() *LinkedMap[K, V] {
	return &LinkedMap[K, V]{m: make(map[K]V)}
}

// NewWithCapacity creates a new ordered map with a given initial capacity.
func NewWithCapacity[K comparable, V any](capacity int) *LinkedMap[K, V] {
	return &LinkedMap[K, V]{m: make(map[K]V, capacity), keys: make([]K, 0, capacity)}
}

// Get returns the value for key.
func (om *LinkedMap[K, V]) Get(key K) (V, bool) {
	if om == nil {
		var zero V
		return zero, false
	}
	v, ok := om.m[key]
	return v, ok
}

// Put stores a value for key. If the key is new, it is appended to the
// insertion-order slice. If the key was previously deleted, it is
// re-inserted at its original position.
func (om *LinkedMap[K, V]) Put(key K, val V) {
	if _, ok := om.m[key]; !ok {
		om.keys = append(om.keys, key)
	}
	om.m[key] = val
}

// Delete removes the entry for key from lookups. The key remains in the
// insertion-order slice but is skipped during iteration.
func (om *LinkedMap[K, V]) Delete(key K) {
	if om == nil {
		return
	}
	delete(om.m, key)
}

// Has returns true if key is present (not deleted).
func (om *LinkedMap[K, V]) Has(key K) bool {
	if om == nil {
		return false
	}
	_, ok := om.m[key]
	return ok
}

// Len returns the number of entries currently present.
func (om *LinkedMap[K, V]) Len() int {
	if om == nil {
		return 0
	}
	return len(om.m)
}

// Keys returns an iterator over keys in insertion order, skipping deleted keys.
func (om *LinkedMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if om == nil {
			return
		}
		for _, k := range om.keys {
			if _, ok := om.m[k]; ok {
				if !yield(k) {
					return
				}
			}
		}
	}
}

// Entries returns an iterator over key-value pairs in insertion order,
// skipping deleted keys.
func (om *LinkedMap[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if om == nil {
			return
		}
		for _, k := range om.keys {
			if v, ok := om.m[k]; ok {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// Pair is a key-value pair for use with NewFromPairs.
type Pair[K any, V any] struct {
	Key K
	Val V
}

// NewFromPairs creates an ordered map from a list of key-value pairs in the
// given order. This provides deterministic insertion order.
func NewFromPairs[K comparable, V any](pairs ...Pair[K, V]) *LinkedMap[K, V] {
	om := NewWithCapacity[K, V](len(pairs))
	for _, p := range pairs {
		om.Put(p.Key, p.Val)
	}
	return om
}

// FromMap creates an ordered map from a standard Go map. Keys are added in
// Go map iteration order (non-deterministic). For deterministic order, use
// Put for each entry individually.
func FromMap[K comparable, V any](m map[K]V) *LinkedMap[K, V] {
	om := NewWithCapacity[K, V](len(m))
	for k, v := range m {
		om.Put(k, v)
	}
	return om
}

// ToMap returns a standard Go map with all current entries.
func (om *LinkedMap[K, V]) ToMap() map[K]V {
	if om == nil {
		return nil
	}
	result := make(map[K]V, len(om.m))
	maps.Copy(result, om.m)
	return result
}
