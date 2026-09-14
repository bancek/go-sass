// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/node.dart

// SassNode is a node in the abstract syntax tree for an unevaluated Sass or
// SCSS file, implemented by every Sass AST node.
//
// Matches Dart: SassNode (sealed interface; Go keeps the set closed through
// the IsSassNode marker instead).
type SassNode interface {
	sasscommon.AstNode
	IsSassNode()
}
