package orderedmap

import "iter"

// MapKeySet is a read-only view of an ordered map's keys: membership, length,
// and keys in insertion order. It is the Go spelling of Dart's
// `MapKeySet(map)`, narrowed to the operations the verifier uses.
//
// Every Map satisfies it, so callers that only care about keys can take this
// instead of a whole map, without depending on the values' type.
//
// Matches Dart: MapKeySet (package:collection)
type MapKeySet[K comparable] interface {
	Has(key K) bool
	Len() int
	Keys() iter.Seq[K]
}

// Map is a read-only ordered map interface.
// Matches Dart: Map<K, V>
type Map[K comparable, V any] interface {
	MapKeySet[K]
	Get(key K) (V, bool)
	Entries() iter.Seq2[K, V]
}
