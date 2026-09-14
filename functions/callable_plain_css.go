// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/callable/plain_css.dart

// PlainCssCallable is a callable that passes a call through as a plain CSS
// function, emitted verbatim as name(args...) once its arguments evaluate to
// CSS.
//
// The evaluator constructs one on the fly when neither a user-defined nor a
// built-in function matches a call; it is never stored in an environment or
// registry. A plain-CSS callable can never back a mixin.
//
// Matches Dart: PlainCssCallable (lib/src/callable/plain_css.dart)
type PlainCssCallable struct {
	name string
}

// Name returns the callable's name.
// Matches Dart: Callable.name
func (c *PlainCssCallable) Name() string { return c.name }

// NewPlainCssCallable creates a PlainCssCallable that passes calls to the
// plain CSS function name through to the output unchanged.
func NewPlainCssCallable(name string) *PlainCssCallable {
	return &PlainCssCallable{
		name: name,
	}
}
