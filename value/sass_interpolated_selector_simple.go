// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector/simple.dart

// InterpolatedSimpleSelector is a single simple selector that may still
// contain interpolation.
//
// It is parsed during the initial stylesheet pass and resolved to a plain
// simple selector once interpolation is evaluated.
type InterpolatedSimpleSelector interface {
	InterpolatedSelector
	IsInterpolatedSimpleSelector()
}
