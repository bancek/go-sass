// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/callable_invocation.dart

// CallableInvocation is implemented by nodes that invoke a callable (a
// function or mixin): FunctionExpression, InterpolatedFunctionExpression,
// LegacyIfExpression, IncludeRule, and ContentRule.
//
// Matches Dart: CallableInvocation (sealed abstract class; Go keeps the set
// closed through the IsCallableInvocation marker instead).
type CallableInvocation interface {
	SassNode
	Arguments() *ArgumentList
	IsCallableInvocation()
}
