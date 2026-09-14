// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// Package functions implements Sass's built-in function system: the pure
// registries behind the sass:color, sass:math, sass:list, sass:map,
// sass:selector, sass:string, and sass:meta modules, their deprecated global
// aliases, and the concrete callable types invocations resolve to.
//
// This package matches Dart's lib/src/functions.dart module structure.
// Evaluator-owned behavior (the meta functions that re-enter evaluation)
// lives on the evaluation visitor, not here.
// dart-source: lib/src/functions.dart
package functions
