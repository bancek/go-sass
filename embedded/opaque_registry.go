// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/opaque_registry.dart

// OpaqueRegistry tracks elements indexed by ID so that the host can invoke
// them.
//
// Matches Dart: class OpaqueRegistry<T>
type OpaqueRegistry struct {
	// Instantiations that have been sent to the host, located at indexes
	// matching their IDs.
	elements []interface{}
	// Reverse map from elements to their indexes in elements.
	idsByElement map[interface{}]int
}

// NewOpaqueRegistry creates a new empty OpaqueRegistry.
func NewOpaqueRegistry() *OpaqueRegistry {
	return &OpaqueRegistry{idsByElement: make(map[interface{}]int)}
}

// GetID returns the compiler-side id associated with element.
//
// Matches Dart: OpaqueRegistry.getId
func (r *OpaqueRegistry) GetID(element interface{}) int {
	if id, ok := r.idsByElement[element]; ok {
		return id
	}
	id := len(r.elements)
	r.elements = append(r.elements, element)
	r.idsByElement[element] = id
	return id
}

// Get returns the compiler-side element associated with id.
//
// If no such element exists, returns nil. Dart's operator [] throws a
// RangeError instead; Go returns nil so host callers get a missing-result
// path rather than an exception.
//
// Matches Dart: OpaqueRegistry.operator []
func (r *OpaqueRegistry) Get(id int) interface{} {
	if id >= len(r.elements) {
		return nil
	}
	return r.elements[id]
}
