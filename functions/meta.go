// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/meta.dart

import (
	"fmt"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// metaFunction builds a built-in callable pinned to the sass:meta module URL,
// matching Dart's private _function helper in meta.dart.
func metaFunction(name string, parameters string, fn sasscallable.CallableFn) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, parameters, "sass:meta", fn)
}

// SharedMetaFunctions returns the meta functions that need no evaluator
// state, available both as globals and in the sass:meta module.
//
// It ports Dart's `_shared`: feature-exists, inspect, type-of, and keywords.
// The remaining meta members (variable-exists, function-exists, call and
// friends) need runtime context, so the evaluator registers them itself.
// Every entry here carries a `meta` deprecation warning, as in Dart's
// `meta.global`.
func SharedMetaFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		featureExistsFunction().WithDeprecationWarning("meta", nil),
		inspectFunction().WithDeprecationWarning("meta", nil),
		typeOfFunction().WithDeprecationWarning("meta", nil),
		keywordsFunction().WithDeprecationWarning("meta", nil),
	}
}

// MetaModule returns the sass:meta built-in module.
//
// It ports Dart's `moduleFunctions`: the shared functions plus the
// module-only calc-name, calc-args, and accepts-content members.
func MetaModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		featureExistsFunction(),
		inspectFunction(),
		typeOfFunction(),
		keywordsFunction(),
		calcNameFunction(),
		calcArgsFunction(),
		acceptsContentFunction(),
	}
	return sassmodule.NewBuiltInModule("meta", fns, nil, nil)
}

// sassFeatures holds the feature names feature-exists() recognizes, porting
// Dart's _features set.
var sassFeatures = map[string]bool{
	"global-variable-shadowing":   true,
	"extend-selector-pseudoclass": true,
	"units-level-3":               true,
	"at-error":                    true,
	"custom-property":             true,
}

// featureExistsFunction builds feature-exists($feature). The call itself is
// deprecated, so it warns first, then reports whether the named feature is
// in the supported set.
func featureExistsFunction() *BuiltInCallable {
	return metaFunction("feature-exists", "$feature", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if err := ec.WarnDeprecation("The feature-exists() function is deprecated.\n\nMore info: https://sass-lang.com/d/feature-exists", deprecation.FeatureExists); err != nil {
			return nil, err
		}
		feature, err := value.AssertString(args[0], new("feature"))
		if err != nil {
			return nil, err
		}
		if sassFeatures[feature.Text] {
			return value.SassTrue, nil
		}
		return value.SassFalse, nil
	})
}

// inspectFunction builds inspect($value). It renders the value the way the
// serializer does in inspect mode and returns it as an unquoted string.
func inspectFunction() *BuiltInCallable {
	return metaFunction("inspect", "$value", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		v := args[0]
		text, err := value.SerializeValueInspect(v)
		if err != nil {
			return nil, err
		}
		return &value.SassString{Text: text, HasQuotes: false}, nil
	})
}

// typeOfFunction builds type-of($value). It maps each value type to its Sass
// type name (arglist, bool, color, list, map, null, number, function, mixin,
// calculation, string), following Dart's switch over the value; an unknown
// Go type panics as a compiler bug.
func typeOfFunction() *BuiltInCallable {
	return metaFunction("type-of", "$value", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		v := args[0]
		var typeName string
		switch v.(type) {
		case *value.SassArgumentList:
			typeName = "arglist"
		case *value.SassBoolean:
			typeName = "bool"
		case *value.SassColor:
			typeName = "color"
		case *value.SassList:
			typeName = "list"
		case *value.SassMap:
			typeName = "map"
		case *value.SassNull:
			typeName = "null"
		case value.SassNumber:
			typeName = "number"
		case *value.SassFunction:
			typeName = "function"
		case *value.SassMixin:
			typeName = "mixin"
		case *value.SassCalculation:
			typeName = "calculation"
		case *value.SassString:
			typeName = "string"
		default:
			panic(fmt.Sprintf("[BUG] Unknown value type %T", v))
		}
		return &value.SassString{Text: typeName, HasQuotes: false}, nil
	})
}

// keywordsFunction builds keywords($args). A rest argument list becomes a
// map of its unquoted keyword names to values; anything else fails with
// "$args: ... is not an argument list."
func keywordsFunction() *BuiltInCallable {
	return metaFunction("keywords", "$args", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		argList, ok := args[0].(*value.SassArgumentList)
		if !ok {
			argStr, err := args[0].String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not an argument list.", argStr), new("args"))
		}
		m := value.EmptySassMap()
		for key, val := range argList.Keywords().Entries() {
			m.Set(&value.SassString{Text: key, HasQuotes: false}, val)
		}
		return m, nil
	})
}

// ---- Module-only functions ----

// calcNameFunction builds the module-only calc-name($calc), returning the
// calculation's name as a quoted string.
func calcNameFunction() sasscallable.Callable {
	return metaFunction("calc-name", "$calc", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		calc, err := value.AssertCalculation(args[0], new("calc"))
		if err != nil {
			return nil, err
		}
		return &value.SassString{Text: calc.Name, HasQuotes: true}, nil
	})
}

// calcArgsFunction builds the module-only calc-args($calc). Each calculation
// argument that is already a value is returned as-is; anything else is
// stringified to an unquoted string, and the results come back as a
// comma-separated list.
func calcArgsFunction() sasscallable.Callable {
	return metaFunction("calc-args", "$calc", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		calc, err := value.AssertCalculation(args[0], new("calc"))
		if err != nil {
			return nil, err
		}
		result := make([]value.Value, len(calc.Arguments))
		for i, arg := range calc.Arguments {
			if v, ok := arg.(value.Value); ok {
				result[i] = v
			} else {
				result[i] = &value.SassString{Text: fmt.Sprintf("%v", arg), HasQuotes: false}
			}
		}
		list, err := value.NewSassList(result, value.ListSeparatorComma, false)
		if err != nil {
			return nil, err
		}
		return list, nil
	})
}

// acceptsContentFunction builds the module-only accepts-content($mixin). A
// built-in mixin reports its own flag, while a user-defined one reports
// whether its declaration takes a content block; an unrecognized callable
// panics as a compiler bug, matching Dart's UnsupportedError arm.
func acceptsContentFunction() sasscallable.Callable {
	// Like Dart's _accepts-content closure in meta.dart's moduleFunctions.
	return metaFunction("accepts-content", "$mixin", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		mixin, err := value.AssertMixin(args[0], new("mixin"))
		if err != nil {
			return nil, err
		}
		switch c := mixin.MixinRef.(type) {
		case *BuiltInCallable:
			if c.AcceptsContent() {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		case *UserDefinedCallable:
			if mr, ok := c.Declaration().(*value.MixinRule); ok {
				if mr.HasContent() {
					return value.SassTrue, nil
				}
				return value.SassFalse, nil
			}
			panic(fmt.Sprintf("Unknown callable type %T.", mixin.MixinRef))
		default:
			panic(fmt.Sprintf("Unknown callable type %T.", mixin.MixinRef))
		}
	})
}
