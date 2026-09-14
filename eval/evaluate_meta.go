// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (meta functions section)

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/configuration"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// registerMetaFunctions builds the evaluator-context sass:meta members and
// assembles the sass:meta module from the shared meta functions plus these
// local functions and mixins. They live on the visitor because they need
// access to the environment and other evaluator state. The local functions
// are additionally registered globally with a meta deprecation warning; the
// shared ones (feature-exists, inspect, type-of, keywords) already carry it
// via the global registry and are not re-registered here.
func (v *EvaluateVisitor) registerMetaFunctions(env *sassenv.Environment) {
	metaFunctions := v.createMetaFunctions(env)
	metaMixins := v.createMetaMixins(env)

	// Create the meta module from shared meta functions + evaluator-context
	// functions + evaluator-context mixins.
	metaMod := functions.MetaModule()
	// Add evaluator-context functions and mixins to the module
	for _, fn := range metaFunctions {
		metaMod.AddFunction(fn)
	}
	for _, mx := range metaMixins {
		metaMod.AddMixin(mx)
	}

	// Store the meta module for resolution by @use 'sass:meta'
	v.builtInModules["sass:meta"] = metaMod

	// Only the evaluator-context functions are registered globally here, each
	// wrapped so global use warns with a meta deprecation.
	for _, fn := range metaFunctions {
		if bc, ok := fn.(*functions.BuiltInCallable); ok {
			wrapped := bc.WithDeprecationWarning("meta", nil)
			normalized := strings.ReplaceAll(wrapped.Name(), "_", "-")
			v.builtInFunctions[normalized] = wrapped
		}
	}
}

// createMetaFunctions returns the evaluator-context sass:meta functions,
// also registered globally with a deprecation warning (unlike the shared
// module members, these close over the visitor's environment and state).
// The newFn helper stamps every entry with the sass:meta URL in one place.
func (v *EvaluateVisitor) createMetaFunctions(env *sassenv.Environment) []sasscallable.Callable {
	newFn := func(name, params string, fn sasscallable.CallableFn) sasscallable.Callable {
		c, err := functions.NewBuiltInCallableFunction(name, params, "sass:meta", fn)
		if err != nil {
			panic("BUG: ParseParameterList for " + name + "(" + params + "): " + err.Error())
		}
		return c
	}

	fns := []sasscallable.Callable{
		newFn("global-variable-exists", "$name, $module: null", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			normalizedName := strings.ReplaceAll(name.Text, "_", "-")
			var namespace *string
			if len(args) > 1 && args[1] != nil && args[1] != value.Null {
				modStr, err := value.AssertString(args[1], new("module"))
				if err != nil {
					return nil, err
				}
				namespace = &modStr.Text
			}
			exists, err := v.env.GlobalVariableExists(normalizedName, namespace)
			if err != nil {
				return nil, err
			}
			if exists {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		newFn("variable-exists", "$name", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			exists, err := v.env.VariableExists(strings.ReplaceAll(name.Text, "_", "-"))
			if err != nil {
				return nil, err
			}
			if exists {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		newFn("function-exists", "$name, $module: null", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			normalizedName := strings.ReplaceAll(name.Text, "_", "-")
			var namespace *string
			if len(args) > 1 && args[1] != nil && args[1] != value.Null {
				modStr, err := value.AssertString(args[1], new("module"))
				if err != nil {
					return nil, err
				}
				namespace = &modStr.Text
			}
			exists, err := v.env.FunctionExists(normalizedName, namespace)
			if err != nil {
				return nil, err
			}
			if exists {
				return value.SassTrue, nil
			}
			// The built-in table is keyed by raw name, so it is consulted with
			// the unnormalized spelling rather than the dashed one.
			if _, ok := v.builtInFunctions[name.Text]; ok {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		newFn("mixin-exists", "$name, $module: null", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			normalizedName := strings.ReplaceAll(name.Text, "_", "-")
			var namespace *string
			if len(args) > 1 && args[1] != nil && args[1] != value.Null {
				modStr, err := value.AssertString(args[1], new("module"))
				if err != nil {
					return nil, err
				}
				namespace = &modStr.Text
			}
			exists, err := v.env.MixinExists(normalizedName, namespace)
			if err != nil {
				return nil, err
			}
			if exists {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		newFn("content-exists", "", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if !v.env.InMixin() {
				return nil, sasscommon.NewSassScriptException("content-exists() may only be called within a mixin.", nil)
			}
			if v.env.Content() != nil {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		newFn("module-variables", "$module", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			modStr, err := value.AssertString(args[0], new("module"))
			if err != nil {
				return nil, err
			}
			mod := v.env.Modules()[modStr.Text]
			if mod == nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("There is no module with namespace %q.", modStr.Text), nil)
			}
			m := value.EmptySassMap()
			for name, val := range mod.Variables().Entries() {
				m.Set(&value.SassString{Text: name, HasQuotes: true}, val)
			}
			return m, nil
		}),

		newFn("module-functions", "$module", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			modStr, err := value.AssertString(args[0], new("module"))
			if err != nil {
				return nil, err
			}
			mod := v.env.Modules()[modStr.Text]
			if mod == nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("There is no module with namespace %q.", modStr.Text), nil)
			}
			m := value.EmptySassMap()
			for name, fn := range mod.Functions().Entries() {
				sf := value.NewSassFunctionWithCompileContext(fn, v.compileContext)
				m.Set(&value.SassString{Text: name, HasQuotes: true}, sf)
			}
			return m, nil
		}),

		newFn("module-mixins", "$module", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			modStr, err := value.AssertString(args[0], new("module"))
			if err != nil {
				return nil, err
			}
			mod := v.env.Modules()[modStr.Text]
			if mod == nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("There is no module with namespace %q.", modStr.Text), nil)
			}
			m := value.EmptySassMap()
			for name, mix := range mod.Mixins().Entries() {
				sm := value.NewSassMixinWithCompileContext(mix, v.compileContext)
				m.Set(&value.SassString{Text: name, HasQuotes: true}, sm)
			}
			return m, nil
		}),

		newFn("get-function", "$name, $css: false, $module: null", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			// $css wraps the name as plain CSS without any lookup, and is
			// mutually exclusive with $module. Otherwise the lookup runs
			// under the call-site span; a namespaced miss does not fall back
			// to built-ins, but an unnamespaced miss does.
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			css := args[1].IsTruthy()
			module := args[2].RealNull()

			if css {
				if module != nil {
					return nil, sasscommon.NewSassScriptException("$css and $module may not both be passed at once.", nil)
				}
				pc := functions.NewPlainCssCallable(name.Text)
				sf := value.NewSassFunctionWithCompileContext(pc, v.compileContext)
				return sf, nil
			}

			var callable sasscallable.Callable
			// The environment miss reports at the call site, not inside the
			// meta machinery.
			callable, err = addExceptionSpan(v, ec.CallableNode(), func() (sasscallable.Callable, error) {
				normalizedName := strings.ReplaceAll(name.Text, "_", "-")
				var namespace *string
				if module != nil {
					modStr, err := value.AssertString(module, new("module"))
					if err != nil {
						return nil, err
					}
					namespace = &modStr.Text
				}
				local, err := v.env.GetFunction(normalizedName, namespace)
				if err != nil {
					return nil, err
				}
				if local != nil || namespace != nil {
					return local, nil
				}
				return v.builtInFunctions[normalizedName], nil
			}, nil)
			if err != nil {
				return nil, err
			}
			if callable == nil {
				nameStr, err := name.String()
				if err != nil {
					return nil, err
				}
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Function not found: %s", nameStr), nil)
			}
			sf := value.NewSassFunctionWithCompileContext(callable, v.compileContext)
			return sf, nil
		}),

		newFn("get-mixin", "$name, $module: null", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			name, err := value.AssertString(args[0], new("name"))
			if err != nil {
				return nil, err
			}
			normalizedName := strings.ReplaceAll(name.Text, "_", "-")
			var namespace *string
			if len(args) > 1 && args[1] != nil && args[1] != value.Null {
				modStr, err := value.AssertString(args[1], new("module"))
				if err != nil {
					return nil, err
				}
				namespace = &modStr.Text
			}
			// Mixins resolve the same way, without the built-in fallback that
			// functions get.
			callable, err := addExceptionSpan(v, ec.CallableNode(), func() (sasscallable.Callable, error) {
				return v.env.GetMixin(normalizedName, namespace)
			}, nil)
			if err != nil {
				return nil, err
			}
			if callable == nil {
				nameStr, err := name.String()
				if err != nil {
					return nil, err
				}
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Mixin not found: %s", nameStr), nil)
			}
			sm := value.NewSassMixinWithCompileContext(callable, v.compileContext)
			return sm, nil
		}),

		// call($function, $args...): re-enters the evaluator with a synthetic
		// argument list built from the rest value (plus a keyword-rest map
		// when keywords are present), all spanned at the call site. A string
		// argument warns and re-evaluates as a named call; a SassFunction
		// asserts its compile context before invoking.
		newFn("call", "$function, $args...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			fn := args[0]
			fnArgs := args[1].(*value.SassArgumentList)
			callableNode := ec.CallableNode()
			callableSpan, err := callableNode.Span()
			if err != nil {
				return nil, err
			}

			// Build an invocation whose rest holds the argument list and whose
			// keyword rest holds the keywords map, mirroring how a call site
			// passes rest/keyword-rest through.
			invocation := value.NewArgumentList(
				[]value.Expression{},
				orderedmap.New[string, value.Expression](),
				map[string]sasscommon.FileSpan{},
				callableSpan,
				value.NewValueExpression(fnArgs, callableSpan),
				nil,
			)
			// Keyword rest only exists when keywords were actually passed.
			kw := fnArgs.Keywords()
			if kw.Len() > 0 {
				kwMap := value.EmptySassMap()
				for name, val := range kw.Entries() {
					kwMap.Set(&value.SassString{Text: name, HasQuotes: false}, val)
				}
				invocation.KeywordRest = value.NewValueExpression(kwMap, callableSpan)
			}

			// A string names a function to look up by re-evaluating as a plain
			// call; this path is deprecated in favor of get-function().
			if str, ok := fn.(*value.SassString); ok {
				serialized, err := value.SerializeValue(str, true)
				if err != nil {
					return nil, err
				}
				if err := ec.WarnDeprecation(
					"Passing a string to call() is deprecated and will be illegal in Dart Sass 2.0.0.\n"+
						"\n"+
						"Recommendation: call(get-function("+serialized+"))",
					deprecation.CallString); err != nil {
					return nil, err
				}
				expr := value.NewFunctionExpression(str.Text, invocation, callableSpan, nil)
				return v.VisitFunctionExpression(expr)
			}

			// Otherwise the function value must come from this compilation;
			// anything else (e.g. an async plugin callable) is a plugin bug.
			sassFn, err := value.AssertFunction(fn, new("function"))
			if err != nil {
				return nil, err
			}
			sassFn, err = sassFn.AssertCompileContext(v.compileContext)
			if err != nil {
				return nil, err
			}
			// The function reference is always a callable here (Dart's
			// SassFunction.callable is non-nullable); only the Go type
			// assertion can fail.
			ref := sassFn.FunctionRef
			callable, ok := ref.(sasscallable.Callable)
			if !ok {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("The function %s is asynchronous.\nThis is probably caused by a bug in a Sass plugin.", sassFn.Name()), nil)
			}

			// The raw argument list is passed through; evaluation happens
			// inside the callable runner, not here.
			return v.invokeCallable(callable, ec.CallableNode(), invocation)
		}),
	}

	// The returned functions are registered globally under a meta deprecation
	// warning, not in the environment, so global use warns but module use
	// does not.
	return fns
}

// createMetaMixins returns the evaluator-context sass:meta mixins, which close
// over the visitor to re-enter evaluation (load-css) or apply a mixin value
// (apply).
func (v *EvaluateVisitor) createMetaMixins(env *sassenv.Environment) []sasscallable.Callable {
	newMixin := func(name, params string, acceptsContent bool, fn sasscallable.CallableFn) sasscallable.Callable {
		c, err := functions.NewBuiltInCallableFunction(name, params, "sass:meta", fn)
		if err != nil {
			panic("BUG: ParseParameterList for " + name + "(" + params + "): " + err.Error())
		}
		if acceptsContent {
			c.SetAcceptsContent(true)
		}
		return c
	}

	mixins := []sasscallable.Callable{
		newMixin("load-css", "$url, $with: null", false, func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			// A null $with means an empty configuration, never the ambient
			// one: only explicitly configured variables reach the module.
			urlStr, err := value.AssertString(args[0], new("url"))
			if err != nil {
				return nil, err
			}
			parsed, err := url.Parse(urlStr.Text)
			// Go's URL parser can reject inputs Dart's accepts; fall back to
			// a path-only URL rather than rejecting the input.
			if err != nil {
				parsed = &url.URL{Path: urlStr.Text}
			}
			var withMap *value.SassMap
			if withVal := args[1].RealNull(); withVal != nil {
				withMap, err = value.AssertMap(withVal, new("with"))
				if err != nil {
					return nil, err
				}
			}

			callableNode := ec.CallableNode()
			var config *configuration.Configuration
			if withMap != nil {
				values := make(map[string]*configuration.ConfiguredValue)
				span, err := callableNode.Span()
				if err != nil {
					return nil, err
				}
				privateDeprecation := false
				for variable, val := range withMap.Entries() {
					nameStr, err := value.AssertString(variable, new("with key"))
					if err != nil {
						return nil, err
					}
					name := strings.ReplaceAll(nameStr.Text, "_", "-")
					if _, exists := values[name]; exists {
						return nil, sasscommon.NewSassScriptException(
							fmt.Sprintf("The variable $%s was configured twice.", name), nil)
					}
					// Configuring private variables warns only once per call.
					if strings.HasPrefix(name, "-") && !privateDeprecation {
						privateDeprecation = true
						if err := ec.WarnDeprecation(
							fmt.Sprintf("Configuring private variables (such as $%s) is "+
								"deprecated.\n"+
								"This will be an error in Dart Sass 2.0.0.", name),
							deprecation.WithPrivate); err != nil {
							return nil, err
						}
					}
					values[name] = configuration.NewConfiguredValueExplicit(val, &span, callableNode)
				}
				explicit := configuration.NewExplicitConfiguration(values, callableNode)
				config = explicit.Configuration
			}

			callableSpan, err := callableNode.Span()
			if err != nil {
				return nil, err
			}
			baseURL, err := callableSpan.SourceURL()
			if err != nil {
				return nil, err
			}

			// The module loads relative to the call site, its CSS is combined
			// (cloned, since the module may be reused) and evaluated into the
			// current stylesheet.
			err = v.loadModule(parsed, "load-css()", callableNode, config, true, baseURL, func(module sassmodule.Module, firstLoad bool) error {
				combined, err := v.combineCss(module, true)
				if err != nil {
					return err
				}
				_, err = v.VisitCssStylesheet(combined)
				return err
			})
			if err != nil {
				return nil, err
			}

			// Configured variables that the module never declared with !default
			// are an error, reported at the configuration span.
			if config != nil && config.IsExplicit() && !config.IsEmpty() {
				vals := config.Values()
				for name, cv := range vals {
					return nil, v.exception(
						fmt.Sprintf("$%s was not declared with !default in the @used module.", name),
						cv.ConfigurationSpan)
				}
			}

			return nil, nil
		}),

		newMixin("apply", "$mixin, $args...", true, func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			// Applies a first-class mixin value with the ambient @content
			// block, re-entering mixin application with a synthetic argument
			// list built from the rest value. Evaluation of the raw list
			// happens inside the mixin runner, not here.
			mixin, err := value.AssertMixin(args[0], new("mixin"))
			if err != nil {
				return nil, err
			}
			mixin, err = mixin.AssertCompileContext(v.compileContext)
			if err != nil {
				return nil, err
			}
			fnArgs := args[1].(*value.SassArgumentList)
			callableNode := ec.CallableNode()
			callableSpan, err := callableNode.Span()
			if err != nil {
				return nil, err
			}

			invocation := value.NewArgumentList(
				[]value.Expression{},
				orderedmap.New[string, value.Expression](),
				map[string]sasscommon.FileSpan{},
				callableSpan,
				value.NewValueExpression(fnArgs, callableSpan),
				nil,
			)

			callable, ok := mixin.MixinRef.(sasscallable.Callable)
			if !ok {
				return nil, sasscommon.NewSassScriptException(
					fmt.Sprintf("The mixin %s is asynchronous.\nThis is probably caused by a bug in a Sass plugin.", mixin.Name()), nil)
			}

			var content *functions.UserDefinedCallable
			if c := v.env.Content(); c != nil {
				content = c.(*functions.UserDefinedCallable)
			}

			err = v.applyMixin(callable, content, callableNode, invocation)
			return nil, err
		}),
	}

	return mixins
}
