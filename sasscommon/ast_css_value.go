// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/ast/css/value.dart

import (
	"fmt"
	"reflect"
)

// CssValue pairs a plain-CSS-tree value with the source span where it was
// defined, for values that do not otherwise track a location. Equality and
// hashing consider only the wrapped value, never the span, so identical
// values from different locations still compare equal.
//
// It ports Dart's final CssValue class, whose ==/hashCode likewise ignore
// the span.
//
// Matches Dart: CssValue (lib/src/ast/css/value.dart)
type CssValue[T any] struct {
	// Value is the wrapped plain-CSS value.
	Value T
	span  FileSpan
}

// NewCssValue wraps value with span as its source location.
func NewCssValue[T any](value T, span FileSpan) CssValue[T] {
	return CssValue[T]{Value: value, span: span}
}

// Span returns the source span where the value was defined.
func (v CssValue[T]) Span() (FileSpan, error) { return v.span, nil }

// IsAstNode marks CssValue as an AstNode.
func (v CssValue[T]) IsAstNode() {}

// String renders the wrapped value, matching Dart's toString delegation.
func (v CssValue[T]) String() string { return fmt.Sprint(v.Value) }

// Equal reports whether the wrapped values are deeply equal, ignoring both
// spans. This matches Dart's CssValue.operator==, which compares only value.
//
// Note: DeepEqual is the generic-T comparison glue; concrete value types
// elsewhere define their own Equal following Dart's per-type operator==.
func (v CssValue[T]) Equal(other CssValue[T]) bool {
	return reflect.DeepEqual(v.Value, other.Value)
}
