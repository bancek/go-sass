// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions.dart

import (
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalFunctions returns the Sass core functions available without a
// module namespace: the deprecated global aliases of every built-in module
// plus the shared meta functions and if().
//
// This excludes the evaluator-owned meta functions (call, variable-exists,
// load-css and friends), which need re-entrant access to evaluation and are
// defined on the evaluation visitor instead.
//
// Matches Dart: globalFunctions (lib/src/functions.dart)
func GlobalFunctions() []sasscallable.Callable {
	var result []sasscallable.Callable
	result = append(result, GlobalColorFunctions()...)
	result = append(result, GlobalListFunctions()...)
	result = append(result, GlobalMapFunctions()...)
	result = append(result, GlobalMathFunctions()...)
	result = append(result, GlobalSelectorFunctions()...)
	result = append(result, GlobalStringFunctions()...)
	result = append(result, SharedMetaFunctions()...)
	// This is only invoked using call(). Hand-authored if()s are parsed as
	// LegacyIfExpression.
	result = append(result, ifFunction())
	return result
}

// CoreModules returns Sass's core library modules: color, list, map, math,
// selector, and string.
//
// This doesn't include the meta module, because that needs additional
// functions that can only be defined in the evaluator itself.
//
// Matches Dart: coreModules (lib/src/functions.dart)
func CoreModules() []*sassmodule.BuiltInModule {
	return []*sassmodule.BuiltInModule{
		ColorModule(),
		ListModule(),
		MapModule(),
		MathModule(),
		SelectorModule(),
		StringModule(),
	}
}

// ifFunction implements the built-in if() function, returning its second
// argument when the condition is truthy and its third otherwise.
// This is only invoked using call(). Hand-authored if()s are parsed as
// LegacyIfExpression.
//
// Matches Dart: the "if" entry in globalFunctions (lib/src/functions.dart)
func ifFunction() sasscallable.Callable {
	return MustNewBuiltInCallableFunction("if", "$condition, $if-true, $if-false", "", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if args[0].IsTruthy() {
			return args[1], nil
		}
		return args[2], nil
	})

}
