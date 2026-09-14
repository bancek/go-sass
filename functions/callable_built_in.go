// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/callable/built_in.dart

import (
	"fmt"
	"net/url"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// overload pairs one parameter declaration with the callback to run when
// that declaration matches an invocation.
//
// Matches Dart: (ParameterList, Callback) tuple in BuiltInCallable._overloads
// (lib/src/callable/built_in.dart)
type overload struct {
	params *value.ParameterList
	fn     sasscallable.CallableFn
}

// CallbackForResult is the overload selected by CallbackFor, exposing the
// matched parameter declaration alongside the callback the evaluator should
// invoke.
//
// Matches Dart: (ParameterList, Callback) tuple returned by
// BuiltInCallable.callbackFor
type CallbackForResult struct {
	// Params is the parameter declaration of the winning overload.
	Params *value.ParameterList
	// Fn is the callback to run for the winning overload.
	Fn sasscallable.CallableFn
}

// BuiltInCallableOverload is a single overload passed to
// NewBuiltInCallableOverloaded: a parameter declaration plus the callback to
// run when that declaration matches.
//
// Matches Dart: Map entry in BuiltInCallable.overloadedFunction overloads parameter
type BuiltInCallableOverload struct {
	// Params is the parameter declaration for this overload.
	Params *value.ParameterList
	// Fn is the callback to run when Params matches an invocation.
	Fn sasscallable.CallableFn
}

// BuiltInCallable is a function or mixin implemented in Go.
//
// Unlike user-defined callables, a built-in callable may declare several
// overloads with different parameter declarations. When invoked, the first
// overload whose declaration matches the call runs; CallbackFor resolves
// which one that is.
//
// Matches Dart: BuiltInCallable (lib/src/callable/built_in.dart). The async
// twin (AsyncBuiltInCallable) is not ported: Go implements sync compilation
// only.
type BuiltInCallable struct {
	name      string
	arguments *value.ParameterList

	// Matches Dart: BuiltInCallable._overloads
	overloads      []overload
	acceptsContent bool
}

// Name returns the callable's name.
//
// Matches Dart: Callable.name
func (c *BuiltInCallable) Name() string { return c.name }

// Arguments returns the parameter declaration of the single-overload form.
// Overloaded callables built with NewBuiltInCallableOverloaded carry no
// top-level declaration, so this reports nil for them; use CallbackFor to
// resolve the winning overload's declaration instead.
func (c *BuiltInCallable) Arguments() *value.ParameterList { return c.arguments }

// NewBuiltInCallable creates a BuiltInCallable with a single parameter
// declaration and a single callback.
//
// Matches Dart: BuiltInCallable.parsed
func NewBuiltInCallable(name string, args *value.ParameterList, fn sasscallable.CallableFn) *BuiltInCallable {
	return &BuiltInCallable{
		name:      name,
		arguments: args,
		overloads: []overload{{params: args, fn: fn}},
	}
}

// NewBuiltInCallableOverloaded creates a BuiltInCallable with multiple
// overloads, tried in the given order: the first declaration matching a call
// wins.
//
// Matches Dart: BuiltInCallable.overloadedFunction (whose map preserves
// insertion order; the Go slice carries that order directly)
func NewBuiltInCallableOverloaded(name string, overloads []BuiltInCallableOverload) *BuiltInCallable {
	ovs := make([]overload, len(overloads))
	for i, o := range overloads {
		ovs[i] = overload{params: o.Params, fn: o.Fn}
	}
	return &BuiltInCallable{
		name:      name,
		overloads: ovs,
	}
}

// NewBuiltInCallableMixin creates a mixin callable whose parameter
// declaration is parsed from parameters (without parentheses), so every
// parameter carries a real span.
//
// If passed, [url] is the URL of the module in which the mixin is defined.
//
// Matches Dart: BuiltInCallable.mixin
func NewBuiltInCallableMixin(name string, parameters string, fn func([]value.Value), urlStr string, acceptsContent bool) (sasscallable.Callable, error) {
	var parsedURL *url.URL
	if urlStr != "" {
		parsedURL, _ = url.Parse(urlStr)
	}
	contents := fmt.Sprintf("@mixin %s(%s) {", name, parameters)
	params, err := value.ParseParameterList(contents, parsedURL)
	if err != nil {
		return nil, err
	}
	wrapped := func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		fn(args)
		// Mixins return no value, but the callback type is shared with
		// functions, so report Sass null and let the call site ignore it.
		// Matches Dart: mixin constructor's "(arguments) { callback(arguments);
		// return sassNull; }" wrapper.
		return value.Null, nil
	}
	result := NewBuiltInCallable(name, params, wrapped)
	result.SetAcceptsContent(acceptsContent)
	return result, nil
}

// AcceptsContent returns whether this callable may accept a @content block.
// Only mixins ever accept content.
//
// Matches Dart: BuiltInCallable.acceptsContent
func (b *BuiltInCallable) AcceptsContent() bool {
	return b.acceptsContent
}

// SetAcceptsContent marks this callable as accepting (or not accepting) a
// @content block. It mutates the receiver, unlike WithName and
// WithDeprecationWarning which return copies.
func (b *BuiltInCallable) SetAcceptsContent(v bool) {
	b.acceptsContent = v
}

// WithName returns a copy of this callable with the given [name], sharing the
// original's overloads. The receiver is never mutated.
//
// Matches Dart: BuiltInCallable.withName
func (b *BuiltInCallable) WithName(name string) *BuiltInCallable {
	result := *b
	result.name = name
	return &result
}

// WithDeprecationWarning returns a copy of this callable that emits a
// GlobalBuiltin deprecation warning before invoking the original overload.
//
// If [newName] is non-nil, it is used in the deprecation message instead of
// the callable's own name. Every overload is wrapped individually so the
// warning fires no matter which overload a call resolves to; the receiver
// itself is never mutated.
//
// Matches Dart: BuiltInCallable.withDeprecationWarning(String module, [String? newName])
func (b *BuiltInCallable) WithDeprecationWarning(module string, newName *string) *BuiltInCallable {
	name := b.name
	if newName != nil {
		name = *newName
	}
	newOverloads := make([]overload, len(b.overloads))
	for i, ov := range b.overloads {
		original := ov.fn
		newOverloads[i] = overload{
			params: ov.params,
			fn: func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
				if err := warnForGlobalBuiltIn(ec, module, name); err != nil {
					return nil, err
				}
				return original(ec, args)
			},
		}
	}
	return &BuiltInCallable{
		name:           b.name,
		arguments:      b.arguments,
		overloads:      newOverloads,
		acceptsContent: b.acceptsContent,
	}
}

// CallbackFor returns the parameter declaration and callback for the given
// positional argument count and named argument set.
//
// If an exact match is found, it is returned immediately. Otherwise the
// closest approximation is returned, selected by mismatch distance
// (preferring smaller absolute distance, and on ties the overload with more
// parameters). Note that the fallback does not guarantee the returned
// declaration actually accepts the call; the caller still verifies. An error
// is returned only when the callable declares no overloads at all, which is
// an invariant violation.
//
// Matches Dart: BuiltInCallable.callbackFor
func (b *BuiltInCallable) CallbackFor(positional int, names map[string]struct{}) (CallbackForResult, error) {
	overload, err := b.callbackFor(positional, names)
	if err != nil {
		return CallbackForResult{}, err
	}
	return CallbackForResult{Params: overload.params, Fn: overload.fn}, nil
}

// callbackFor is the unexported overload search behind CallbackFor: it
// returns the winning overload entry itself rather than a detached result.
func (b *BuiltInCallable) callbackFor(positional int, names map[string]struct{}) (*overload, error) {
	var fuzzyMatch *overload
	var minMismatchDistance int
	var hasFuzzy bool

	for i := range b.overloads {
		overload := &b.overloads[i]

		// Ideally, find an exact match.
		if matchesSignatureExact(overload.params, positional, names) {
			return overload, nil
		}

		paramCount := 0
		if overload.params != nil {
			paramCount = len(overload.params.Parameters)
		}
		mismatchDistance := paramCount - positional

		if hasFuzzy {
			if absInt(mismatchDistance) > absInt(minMismatchDistance) {
				continue
			}
			// If two overloads have the same mismatch distance, favor the
			// overload that has more parameters.
			if absInt(mismatchDistance) == absInt(minMismatchDistance) &&
				mismatchDistance < 0 {
				continue
			}
		}

		minMismatchDistance = mismatchDistance
		fuzzyMatch = overload
		hasFuzzy = true
	}

	if fuzzyMatch != nil {
		return fuzzyMatch, nil
	}
	// A callable with no overloads at all can never match; Dart throws
	// StateError here, Go reports it as an error since callbacks return
	// (Value, error).
	return nil, &sasscommon.StateError{Message: fmt.Sprintf("BuiltInCallable %s may not have empty overloads", b.name)}
}

// absInt reports the absolute value of n. Used to compare overload mismatch
// distances without bias toward too-few or too-many parameters.
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// matchesSignatureExact reports whether a call with positional positional
// arguments and the named argument set names is exactly compatible with the
// parameter declaration params: every positional slot must not also be passed
// by name, every required parameter must be filled positionally or by name,
// no unknown names may remain, and extra positionals are allowed only with a
// rest parameter.
//
// Matches Dart: ParameterList.matches
func matchesSignatureExact(params *value.ParameterList, positional int, names map[string]struct{}) bool {
	if params == nil {
		return positional == 0 && len(names) == 0
	}

	namedUsed := 0
	for i, p := range params.Parameters {
		_, named := names[p.Name()]
		if i < positional {
			if named {
				return false
			}
		} else if named {
			namedUsed++
		} else if p.DefaultValue == nil {
			return false
		}
	}

	if params.RestParameter != nil {
		return true
	}
	if positional > len(params.Parameters) {
		return false
	}
	if namedUsed < len(names) {
		return false
	}
	return true
}

// NewBuiltInCallableFunction creates a BuiltInCallable from a raw parameter
// string (without parentheses), parsed as a Sass @function declaration so
// that every parameter carries a real span. It returns a SassFormatException
// error if parsing fails.
//
// Matches Dart: BuiltInCallable.function
func NewBuiltInCallableFunction(name, parameters, urlStr string, fn sasscallable.CallableFn) (*BuiltInCallable, error) {
	parsedURL, _ := url.Parse(urlStr)
	contents := fmt.Sprintf("@function %s(%s) {", name, parameters)
	params, err := value.ParseParameterList(contents, parsedURL)
	if err != nil {
		return nil, err
	}
	return NewBuiltInCallable(name, params, fn), nil
}

// OverloadDef defines a single overload in an overloaded callable: a raw
// parameter string plus the callback to run when it matches.
//
// Matches Dart: entries in BuiltInCallable.overloadedFunction parameter map
type OverloadDef struct {
	// Params is the raw parameter declaration (without parentheses).
	Params string
	// Fn is the callback to run when Params matches an invocation.
	Fn sasscallable.CallableFn
}

// NewBuiltInCallableOverloadedFunction creates a BuiltInCallable with multiple
// overloads, each defined by a raw parameter string parsed as a Sass
// @function declaration. Overloads are tried in the given order: the first
// declaration matching a call wins. It returns a SassFormatException error
// if any declaration fails to parse.
//
// Matches Dart: BuiltInCallable.overloadedFunction
func NewBuiltInCallableOverloadedFunction(name, urlStr string, overloads ...OverloadDef) (*BuiltInCallable, error) {
	parsedURL, _ := url.Parse(urlStr)
	result := make([]BuiltInCallableOverload, 0, len(overloads))
	for _, o := range overloads {
		contents := fmt.Sprintf("@function %s(%s) {", name, o.Params)
		parsed, err := value.ParseParameterList(contents, parsedURL)
		if err != nil {
			return nil, err
		}
		result = append(result, BuiltInCallableOverload{Params: parsed, Fn: o.Fn})
	}
	return NewBuiltInCallableOverloaded(name, result), nil
}

// MustNewBuiltInCallableFunction is the panicking equivalent of
// NewBuiltInCallableFunction for hardcoded built-in function definitions at
// package init time. A hardcoded declaration that fails to parse is a
// programming error, so it panics instead of returning an error.
//
// Matches Dart: _function() helper in list.dart, math.dart, etc.
func MustNewBuiltInCallableFunction(name, parameters, urlStr string, fn sasscallable.CallableFn) *BuiltInCallable {
	c, err := NewBuiltInCallableFunction(name, parameters, urlStr, fn)
	if err != nil {
		panic("BUG: ParseParameterList for " + name + "(" + parameters + "): " + err.Error())
	}
	return c
}

// MustNewBuiltInCallableOverloadedFunction is the panicking equivalent of
// NewBuiltInCallableOverloadedFunction for hardcoded built-in function
// definitions at package init time. A hardcoded declaration that fails to
// parse is a programming error, so it panics instead of returning an error.
//
// Matches Dart: _overloadedFunction() helper in selector.dart, etc.
func MustNewBuiltInCallableOverloadedFunction(name, urlStr string, overloads ...OverloadDef) *BuiltInCallable {
	c, err := NewBuiltInCallableOverloadedFunction(name, urlStr, overloads...)
	if err != nil {
		panic("BUG: ParseParameterList for overloaded " + name + ": " + err.Error())
	}
	return c
}

// NewCallableFromSignature parses a full function signature like "name(args)"
// and returns a single-overload callable for it.
//
// If requireParens is false, parentheses may be omitted. A SassFormatException
// error is returned if parsing fails.
//
// Matches Dart: Callable.fromSignature
func NewCallableFromSignature(signature string, fn sasscallable.CallableFn, requireParens bool) (sasscallable.Callable, error) {
	name, params, err := value.ParseSignature(signature, requireParens)
	if err != nil {
		return nil, err
	}
	return NewBuiltInCallable(name, params, fn), nil
}
