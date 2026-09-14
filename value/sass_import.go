// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/import.dart

// Import is implemented by every import type: a DynamicImport that loads a
// Sass file at runtime, or a StaticImport that produces a plain CSS @import
// rule.
//
// Matches Dart: Import (abstract interface).
type Import interface {
	SassNode
	IsImport()
	String() (string, error)
}
