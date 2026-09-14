// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/callable/user_defined.dart

import (
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/value"
)

// UserDefinedCallable is a function or mixin defined in a Sass stylesheet:
// its source declaration plus the environment captured where it was defined.
//
// Dart spells this UserDefinedCallable<E> over sync and async environments;
// Go compiles synchronously only, so the environment is always
// *sassenv.Environment.
//
// Matches Dart: UserDefinedCallable (lib/src/callable/user_defined.dart)
type UserDefinedCallable struct {
	declaration  value.CallableDeclaration
	env          *sassenv.Environment
	inDependency bool
}

// Name returns the declared name of the function or mixin.
//
// Matches Dart: UserDefinedCallable.name => declaration.name
func (u *UserDefinedCallable) Name() string { return u.declaration.Name() }

// Arguments returns the parameter declaration from the source declaration.
// There is no isMixin field: call sites switch on the declaration kind
// instead.
//
// Matches Dart: declaration.parameters
func (u *UserDefinedCallable) Arguments() *value.ParameterList { return u.declaration.Parameters() }

// NewUserDefinedCallable creates a UserDefinedCallable for declaration,
// capturing env as the lexical scope the body will run in and recording
// whether the definition came from a dependency.
//
// Matches Dart: UserDefinedCallable constructor
func NewUserDefinedCallable(declaration value.CallableDeclaration, env *sassenv.Environment, inDependency bool) *UserDefinedCallable {
	return &UserDefinedCallable{
		declaration:  declaration,
		env:          env,
		inDependency: inDependency,
	}
}

// Env returns the environment captured where the callable was declared. The
// body evaluates in this scope, so it sees the definition-site bindings
// rather than the call-site ones.
func (u *UserDefinedCallable) Env() *sassenv.Environment { return u.env }

// InDependency reports whether the callable was defined in a dependency:
// transitively loaded through a load path or importer rather than relative
// to the entrypoint. Dependency definitions stay quiet under quietDeps.
//
// Matches Dart: UserDefinedCallable.inDependency
func (u *UserDefinedCallable) InDependency() bool { return u.inDependency }

// Declaration returns the function, mixin, or content-block declaration the
// callable was defined with.
func (u *UserDefinedCallable) Declaration() value.CallableDeclaration { return u.declaration }
