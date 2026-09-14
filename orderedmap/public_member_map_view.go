package orderedmap

import "iter"

// PublicMemberMapView wraps a Map[string, V] and hides keys that
// start with "_" or "-" (private Sass members).
//
// Matches Dart: PublicMemberMapView
type PublicMemberMapView[V any] struct {
	inner Map[string, V]
}

func NewPublicMemberMapView[V any](inner Map[string, V]) *PublicMemberMapView[V] {
	return &PublicMemberMapView[V]{inner: inner}
}

func isPublic(name string) bool {
	if len(name) == 0 {
		return true
	}
	return name[0] != '-' && name[0] != '_'
}

func (v *PublicMemberMapView[V]) Get(key string) (V, bool) {
	if !isPublic(key) {
		var zero V
		return zero, false
	}
	return v.inner.Get(key)
}

func (v *PublicMemberMapView[V]) Has(key string) bool {
	return isPublic(key) && v.inner.Has(key)
}

func (v *PublicMemberMapView[V]) Len() int {
	n := 0
	for key := range v.inner.Keys() {
		if isPublic(key) {
			n++
		}
	}
	return n
}

func (v *PublicMemberMapView[V]) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range v.inner.Keys() {
			if isPublic(key) {
				if !yield(key) {
					return
				}
			}
		}
	}
}

func (v *PublicMemberMapView[V]) Entries() iter.Seq2[string, V] {
	return func(yield func(string, V) bool) {
		for key, val := range v.inner.Entries() {
			if isPublic(key) {
				if !yield(key, val) {
					return
				}
			}
		}
	}
}
