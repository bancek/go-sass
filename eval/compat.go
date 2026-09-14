// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (isJS compile-time constant,
// re-exported from package:cli_pkg/js.dart; use site at the package: URL
// importer error)

// IsJS reports whether the code runs in a JavaScript/Node.js context.
// Defaults to false (native Go). Set to true when creating Node.js bindings
// so that platform-specific behavior gated on Dart's isJS constant (for
// example the "package:" URL error message in the importer path) is
// preserved.
//
// Matches Dart: the isJS compile-time constant from package:cli_pkg/js.dart,
// referenced in lib/src/visitor/evaluate.dart. Unlike Dart's constant, this
// is a mutable variable because Go has no JS build target to specialize on.
var IsJS bool
