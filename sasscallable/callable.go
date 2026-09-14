// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscallable

// dart-source: lib/src/callable.dart

import (
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/value"
)

// CallableFn is the Go-level function signature for built-in callables.
//
// The evaluation context is threaded explicitly (replacing Dart's ambient
// zones) so warnings resolve against the calling scope. Keyword arguments
// are packed into a SassArgumentList as the last positional arg, so there
// is no separate keywords map.
//
// Matches Dart: Value callback(List<Value> arguments).
type CallableFn func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error)

// Callable is the interface all callables implement: user-defined and
// built-in functions and mixins alike.
//
// Matches Dart: Callable, the shared interface for anything invocable from
// Sass. Dart layers async support beneath it; Go ports only the synchronous
// behavior, so this interface carries just the member every callable
// shares. Concrete callables live in functions/ and are stored under their
// own names in module and environment maps.
type Callable interface {
	// Name returns the callable's Sass-visible name, without argument list.
	Name() string
}
