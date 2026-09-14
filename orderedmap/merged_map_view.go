package orderedmap

import "iter"

// MergedMapView is a lazy view of multiple ordered maps merged as one.
// Values in later maps take priority. Iteration order follows
// first-seen key order across all maps.
//
// Matches Dart: MergedMapView
type MergedMapView[K comparable, V any] struct {
	mapsByKey *LinkedMap[K, Map[K, V]]
}

func NewMergedMapView[K comparable, V any](maps []Map[K, V]) *MergedMapView[K, V] {
	mapsByKey := New[K, Map[K, V]]()
	for _, m := range maps {
		if inner, ok := m.(*MergedMapView[K, V]); ok {
			for key, childMap := range inner.mapsByKey.Entries() {
				mapsByKey.Put(key, childMap)
			}
		} else {
			for key := range m.Keys() {
				mapsByKey.Put(key, m)
			}
		}
	}
	return &MergedMapView[K, V]{mapsByKey: mapsByKey}
}

func (v *MergedMapView[K, V]) Get(key K) (V, bool) {
	owningMap, ok := v.mapsByKey.Get(key)
	if !ok {
		var zero V
		return zero, false
	}
	return owningMap.Get(key)
}

func (v *MergedMapView[K, V]) Has(key K) bool {
	return v.mapsByKey.Has(key)
}

func (v *MergedMapView[K, V]) Len() int {
	return v.mapsByKey.Len()
}

func (v *MergedMapView[K, V]) Keys() iter.Seq[K] {
	return v.mapsByKey.Keys()
}

func (v *MergedMapView[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for key, owningMap := range v.mapsByKey.Entries() {
			val, _ := owningMap.Get(key)
			if !yield(key, val) {
				return
			}
		}
	}
}

func (v *MergedMapView[K, V]) Put(key K, val V) {
	owningMap, ok := v.mapsByKey.Get(key)
	if !ok {
		panic("Cannot add new keys to a MergedMapView")
	}
	if lm, ok := owningMap.(*LinkedMap[K, V]); ok {
		lm.Put(key, val)
		return
	}
	panic("Cannot put into a non-LinkedMap member of MergedMapView")
}
