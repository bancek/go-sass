// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/utils.dart (listHash, mapHash) + package:collection hashCombine

import "fmt"

// hashCombine combines two hash codes using a golden-ratio-based
// algorithm, matching the hash distribution of Dart's package:collection.
func hashCombine(seed, value int) int {
	return seed ^ (value + 0x9e3779b9 + (seed << 6) + (seed >> 2))
}

// stringHashCode returns a hash code for s matching Java/Dart String.hashCode.
func stringHashCode(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	return h
}

// listHash returns a combined hash for values, matching Dart's ListEquality.hash.
// E.g. Dart: const ListEquality<Object>().hash(asList)
func listHash(values []Value) int {
	hash := 0
	for _, v := range values {
		hash = hashCombine(hash, v.HashCode())
	}
	return hash
}

// mapHash returns a combined hash for entries in insertion order,
// matching Dart's MapEquality.hash.
// E.g. Dart: const MapEquality<Object, Object>().hash(map)
func mapHash(entries []MapEntry) int {
	hash := 0
	for _, e := range entries {
		hash = hashCombine(hash, e.Key.HashCode())
		hash = hashCombine(hash, e.Value.HashCode())
	}
	return hash
}

// hashPtr returns a hash code for a pointer value, returning 0 for nil.
func hashPtr[T comparable](p *T) int {
	if p == nil {
		return 0
	}
	return stringHashCode(fmt.Sprintf("%v", *p))
}

// boolHashCode returns the hash code for a bool, matching Dart's bool.hashCode.
func boolHashCode(v bool) int {
	if v {
		return 1231
	}
	return 1237
}
