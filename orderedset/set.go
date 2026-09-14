package orderedset

import "iter"

// Set is a read-only ordered set interface.
// Matches Dart: Set<E>
type Set[K comparable] interface {
	Contains(key K) bool
	Len() int
	Keys() iter.Seq[K]
	Map() map[K]struct{}
}
