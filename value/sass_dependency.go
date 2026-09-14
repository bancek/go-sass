// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/dependency.dart

import (
	"net/url"

	"github.com/bancek/go-sass/sasscommon"
)

// SassDependency is implemented by UseRule, ForwardRule, and DynamicImport:
// the three nodes that load another stylesheet as a module dependency.
//
// Matches Dart: SassDependency (sealed interface; Go keeps the set closed
// through the unexported isSassDependency marker instead).
type SassDependency interface {
	SassNode
	URL() *url.URL
	URLSpan() (sasscommon.FileSpan, error)
	isSassDependency()
}
