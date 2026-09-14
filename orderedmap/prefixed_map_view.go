package orderedmap

import (
	"iter"
	"strings"
)

// PrefixedMapView wraps a Map[string, V] and exposes all keys
// with an added prefix.
//
// Matches Dart: PrefixedMapView
type PrefixedMapView[V any] struct {
	inner  Map[string, V]
	prefix string
}

func NewPrefixedMapView[V any](inner Map[string, V], prefix string) *PrefixedMapView[V] {
	return &PrefixedMapView[V]{inner: inner, prefix: prefix}
}

func (v *PrefixedMapView[V]) Get(key string) (V, bool) {
	if !strings.HasPrefix(key, v.prefix) {
		var zero V
		return zero, false
	}
	return v.inner.Get(key[len(v.prefix):])
}

func (v *PrefixedMapView[V]) Has(key string) bool {
	if !strings.HasPrefix(key, v.prefix) {
		return false
	}
	return v.inner.Has(key[len(v.prefix):])
}

func (v *PrefixedMapView[V]) Len() int {
	return v.inner.Len()
}

func (v *PrefixedMapView[V]) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range v.inner.Keys() {
			if !yield(v.prefix + key) {
				return
			}
		}
	}
}

func (v *PrefixedMapView[V]) Entries() iter.Seq2[string, V] {
	return func(yield func(string, V) bool) {
		for key, val := range v.inner.Entries() {
			if !yield(v.prefix+key, val) {
				return
			}
		}
	}
}
