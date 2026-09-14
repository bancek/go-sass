// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/reference.dart

// SassReference is implemented by any node that references a Sass member:
// VariableExpression and FunctionExpression.
//
// Matches Dart: SassReference (interface; Go keeps the set closed through
// the unexported isSassReference marker instead).
type SassReference interface {
	SassNode
	Namespace() *string
	Name() string
	NameSpan() (sasscommon.FileSpan, error)
	NamespaceSpan() (*sasscommon.FileSpan, error)
	isSassReference()
}
