// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolated_selector.dart

// InterpolatedSelector is a selector that may still contain interpolation.
//
// It is parsed during the initial stylesheet pass (when selector parsing is
// enabled) and resolved to a plain selector once interpolation is evaluated.
// AcceptVoid dispatches to the matching InterpolatedSelectorVisitor method.
type InterpolatedSelector interface {
	SassNode
	IsInterpolatedSelector()
	AcceptVoid(v InterpolatedSelectorVisitor[struct{}]) (struct{}, error)
	String() (string, error)
}
