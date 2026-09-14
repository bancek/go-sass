// Package linkedhashmap provides an insertion-order-preserving map with
// custom hash/equality. Hash is derived from the key's HashCode() method;
// equality is supplied as a function at construction.
//
// Matches Dart: JsLinkedHashMap (js_runtime/lib/linked_hash_map.dart)
package linkedhashmap

import "iter"

// HashCoder is the constraint for map keys.
type HashCoder interface {
	HashCode() int
}

type entry[K HashCoder, V any] struct {
	key   K
	value V
	prev  *entry[K, V]
	next  *entry[K, V]
}

// LinkedHashMap is a map keyed by custom hash and equality, preserving
// insertion order for iteration. Keys must satisfy the HashCoder constraint
// (HashCode() int). Equality is a user-supplied function.
type LinkedHashMap[K HashCoder, V any] struct {
	equals  func(K, K) bool
	buckets map[uint64][]*entry[K, V]
	head    *entry[K, V]
	tail    *entry[K, V]
	len     int
}

// New creates a new LinkedHashMap with the given equality function.
func New[K HashCoder, V any](equals func(K, K) bool) *LinkedHashMap[K, V] {
	return &LinkedHashMap[K, V]{
		equals:  equals,
		buckets: make(map[uint64][]*entry[K, V]),
	}
}

// findInBucket returns the index of the entry in bucket whose key equals key
// via equals(), or -1 if not found.
// Matches Dart: internalFindBucketIndex:282-289
func (m *LinkedHashMap[K, V]) findInBucket(bucket []*entry[K, V], key K) int {
	for i, e := range bucket {
		if m.equals(e.key, key) {
			return i
		}
	}
	return -1
}

// unlinkCell removes e from the insertion-order doubly-linked list.
// Matches Dart: _unlinkCell:243-260
func (m *LinkedHashMap[K, V]) unlinkCell(e *entry[K, V]) {
	if e.prev == nil {
		m.head = e.next
	} else {
		e.prev.next = e.next
	}
	if e.next == nil {
		m.tail = e.prev
	} else {
		e.next.prev = e.prev
	}
	e.prev = nil
	e.next = nil
	m.len--
}

// newCell creates a new entry, links it at the tail of the order list,
// and returns it.
// Matches Dart: _newLinkedCell:229-240
func (m *LinkedHashMap[K, V]) newCell(key K, val V) *entry[K, V] {
	e := &entry[K, V]{key: key, value: val}
	if m.head == nil {
		m.head = e
		m.tail = e
	} else {
		e.prev = m.tail
		m.tail.next = e
		m.tail = e
	}
	m.len++
	return e
}

// Get returns the value for key, or false if not present.
// Matches Dart: internalGet:100-108
func (m *LinkedHashMap[K, V]) Get(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}
	hash := uint64(key.HashCode())
	bucket := m.buckets[hash]
	if idx := m.findInBucket(bucket, key); idx != -1 {
		return bucket[idx].value, true
	}
	var zero V
	return zero, false
}

// Has returns true if key is present.
func (m *LinkedHashMap[K, V]) Has(key K) bool {
	if m == nil {
		return false
	}
	hash := uint64(key.HashCode())
	bucket := m.buckets[hash]
	return m.findInBucket(bucket, key) != -1
}

// Put inserts or updates key→val. New keys go to the tail of insertion order.
// Existing keys have their value replaced; insertion position is unchanged.
// Matches Dart: []=:111-122 (insert), []=:205-207 (update)
func (m *LinkedHashMap[K, V]) Put(key K, val V) {
	hash := uint64(key.HashCode())
	bucket := m.buckets[hash]
	if idx := m.findInBucket(bucket, key); idx != -1 {
		bucket[idx].value = val
		return
	}
	e := m.newCell(key, val)
	m.buckets[hash] = append(bucket, e)
}

// Delete removes key from the map. No-op if not present.
// Matches Dart: internalRemove:161-177
func (m *LinkedHashMap[K, V]) Delete(key K) {
	if m == nil {
		return
	}
	hash := uint64(key.HashCode())
	bucket := m.buckets[hash]
	idx := m.findInBucket(bucket, key)
	if idx == -1 {
		return
	}
	e := bucket[idx]
	m.buckets[hash] = append(bucket[:idx], bucket[idx+1:]...)
	if len(m.buckets[hash]) == 0 {
		delete(m.buckets, hash)
	}
	m.unlinkCell(e)
}

// Len returns the number of entries.
func (m *LinkedHashMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return m.len
}

// Keys returns an iterator over keys in insertion order.
func (m *LinkedHashMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if m == nil {
			return
		}
		for e := m.head; e != nil; e = e.next {
			if !yield(e.key) {
				return
			}
		}
	}
}

// Entries returns an iterator over key-value pairs in insertion order.
func (m *LinkedHashMap[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}
		for e := m.head; e != nil; e = e.next {
			if !yield(e.key, e.value) {
				return
			}
		}
	}
}

// Copy returns a deep copy of the map (new cells, same insertion order,
// shared equals function).
func (m *LinkedHashMap[K, V]) Copy() *LinkedHashMap[K, V] {
	if m == nil {
		return nil
	}
	c := New[K, V](m.equals)
	for e := m.head; e != nil; e = e.next {
		c.Put(e.key, e.value)
	}
	return c
}
