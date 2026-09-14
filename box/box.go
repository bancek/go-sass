// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package box

// dart-source: lib/src/util/box.dart

import "fmt"

// ModifiableBox is a mutable reference to a (presumably immutable) value.
// It always uses reference equality, even when the underlying type uses
// value equality: the extension store swaps Value in place during
// evaluation while readers hold the sealed view.
//
// Matches Dart: ModifiableBox (util/box.dart)
type ModifiableBox[T any] struct {
	// Value holds the referenced value; the owner mutates it in place.
	Value T
	// id keys identity-keyed collections; standalone boxes carry 0 and
	// are never used as map keys.
	id uint64
}

// Box is an unmodifiable reference to a value that may be mutated
// elsewhere. Like ModifiableBox it uses reference equality based on the
// underlying box, even when the underlying type uses value equality.
//
// Matches Dart: Box (util/box.dart)
type Box[T any] struct {
	inner *ModifiableBox[T]
	id    uint64
}

// NewModifiableBox creates a standalone ModifiableBox with id: 0.
// Standalone boxes are never used as identity-keyed map keys; their id is
// only meaningful for store-managed boxes.
//
// Matches Dart: ModifiableBox constructor (util/box.dart)
func NewModifiableBox[T any](value T) *ModifiableBox[T] {
	return &ModifiableBox[T]{Value: value, id: 0}
}

// NewModifiableBoxWithID creates a ModifiableBox with an explicit id,
// used by the extension store for identity-keyed collections. This is a
// Go-only constructor: Dart's store tracks identity by reference, while Go
// threads an explicit numeric identity alongside the reference.
//
// Go-only (no Dart counterpart).
func NewModifiableBoxWithID[T any](value T, id uint64) *ModifiableBox[T] {
	return &ModifiableBox[T]{Value: value, id: id}
}

// ID returns the identity of this box, used to key identity-based
// collections.
func (mb *ModifiableBox[T]) ID() uint64 { return mb.id }

// HashCode returns the hash code for use with linkedhashmap collections.
// The hash is based on id for identity-based equality: two boxes are
// equal exactly when their ids match, regardless of the wrapped value.
func (mb *ModifiableBox[T]) HashCode() int { return int(mb.id) }

// Seal returns an unmodifiable reference to this box. The underlying
// modifiable box may still be modified through its owner.
//
// Matches Dart: ModifiableBox.seal (util/box.dart)
func (mb *ModifiableBox[T]) Seal() *Box[T] {
	return &Box[T]{inner: mb, id: mb.id}
}

// ID returns the identity of the underlying modifiable box.
func (b *Box[T]) ID() uint64 { return b.id }

// Value returns the value wrapped by this sealed box, reading through to
// the current value of the underlying modifiable box.
//
// Matches Dart: Box.value (util/box.dart)
func (b *Box[T]) Value() T { return b.inner.Value }

// SetValue sets the value wrapped by this sealed box, writing through to
// the inner ModifiableBox. This is a Go-only affordance: Dart exposes only
// a value getter on Box, with mutation going through the ModifiableBox.
//
// Go-only (no Dart counterpart).
func (b *Box[T]) SetValue(v T) { b.inner.Value = v }

// String renders the box by delegating to the wrapped value when it
// carries its own rendering, and falling back to an explicit
// "<modifiable box: ...>" wrapper otherwise.
//
// Matches Dart: ModifiableBox.toString (util/box.dart)
func (mb *ModifiableBox[T]) String() (string, error) {
	if s, ok := any(mb.Value).(interface{ String() (string, error) }); ok {
		return s.String()
	}
	return fmt.Sprintf("<modifiable box: %v>", mb.Value), nil
}

// String renders the sealed box by delegating to the wrapped value when
// it carries its own rendering, and falling back to an explicit
// "<box: ...>" wrapper otherwise.
//
// Matches Dart: Box.toString (util/box.dart)
func (b *Box[T]) String() (string, error) {
	if s, ok := any(b.Value()).(interface{ String() (string, error) }); ok {
		return s.String()
	}
	return fmt.Sprintf("<box: %v>", b.Value()), nil
}
