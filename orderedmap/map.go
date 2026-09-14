package orderedmap

import "iter"

// Map is a read-only ordered map interface.
// Matches Dart: Map<K, V>
type Map[K comparable, V any] interface {
	Get(key K) (V, bool)
	Has(key K) bool
	Len() int
	Keys() iter.Seq[K]
	Entries() iter.Seq2[K, V]
}
