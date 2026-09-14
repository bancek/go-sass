package orderedmap

import "iter"

// LimitedMapView wraps a Map[K, V] and only exposes keys in a
// pre-computed allowed set.
//
// Matches Dart: LimitedMapView
type LimitedMapView[K comparable, V any] struct {
	inner Map[K, V]
	keys  map[K]struct{}
}

func NewLimitedMapViewSafelist[K comparable, V any](m Map[K, V], safelist map[K]struct{}) *LimitedMapView[K, V] {
	keys := make(map[K]struct{})
	for k := range m.Keys() {
		if _, ok := safelist[k]; ok {
			keys[k] = struct{}{}
		}
	}
	return &LimitedMapView[K, V]{inner: m, keys: keys}
}

func NewLimitedMapViewBlocklist[K comparable, V any](m Map[K, V], blocklist map[K]struct{}) *LimitedMapView[K, V] {
	keys := make(map[K]struct{})
	for k := range m.Keys() {
		if _, ok := blocklist[k]; !ok {
			keys[k] = struct{}{}
		}
	}
	return &LimitedMapView[K, V]{inner: m, keys: keys}
}

func (v *LimitedMapView[K, V]) Get(key K) (V, bool) {
	if _, ok := v.keys[key]; !ok {
		var zero V
		return zero, false
	}
	return v.inner.Get(key)
}

func (v *LimitedMapView[K, V]) Has(key K) bool {
	_, ok := v.keys[key]
	return ok
}

func (v *LimitedMapView[K, V]) Len() int {
	return len(v.keys)
}

func (v *LimitedMapView[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for key := range v.inner.Keys() {
			if _, ok := v.keys[key]; ok {
				if !yield(key) {
					return
				}
			}
		}
	}
}

func (v *LimitedMapView[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for key, val := range v.inner.Entries() {
			if _, ok := v.keys[key]; ok {
				if !yield(key, val) {
					return
				}
			}
		}
	}
}

func (v *LimitedMapView[K, V]) Delete(key K) {
	if _, ok := v.keys[key]; !ok {
		return
	}
	delete(v.keys, key)
	if lm, ok := v.inner.(*LinkedMap[K, V]); ok {
		lm.Delete(key)
	}
}
