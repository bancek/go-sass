// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (constructor and global functions section)

import (
	"strings"

	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/value"
)

// RegisterBuiltInFunctions populates the environment with built-in functions
// and modules.
//
// Matches Dart: _EvaluateVisitor constructor — global functions are stored in
// _builtInFunctions (not _environment), so that VisitFunctionExpression can
// intercept CSS math functions before falling back to the built-in.
func RegisterBuiltInFunctions(env *sassenv.Environment, evaluator *EvaluateVisitor) error {
	for _, mod := range functions.CoreModules() {
		url, err := mod.URL()
		if err != nil {
			return err
		}
		evaluator.builtInModules[url] = mod
	}
	// Matches Dart: globalFunctions → _builtInFunctions
	// Global functions must NOT be added to env (only user-defined and
	// @use'd module functions go there), so that CSS math functions like
	// round() are intercepted by visitCalculation before falling back to
	// the Sass built-in.
	globalFns := functions.GlobalFunctions()
	for _, fn := range globalFns {
		normalized := strings.ReplaceAll(fn.Name(), "_", "-")
		evaluator.builtInFunctions[normalized] = fn
	}
	evaluator.registerMetaFunctions(env)

	return nil
}

// NewUserDefinedCallable creates a callable from a Sass function/mixin
// definition.
func NewUserDefinedCallable(declaration value.CallableDeclaration, env *sassenv.Environment, inDependency bool) *functions.UserDefinedCallable {
	return functions.NewUserDefinedCallable(declaration, env, inDependency)
}
