// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/declaration.dart

// SassDeclaration is implemented by any node that declares a Sass member
// (parameters, configured variables, callable declarations).
//
// Matches Dart: SassDeclaration (sealed interface; Go keeps the set closed
// through the IsSassDeclaration marker instead).
type SassDeclaration interface {
	SassNode
	Name() string
	NameSpan() (sasscommon.FileSpan, error)
	IsSassDeclaration()
}
