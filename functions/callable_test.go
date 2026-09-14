// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package functions

import (
	"testing"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/value"
)

func testSpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(nil, 0, 0)
}

func noopCallback() sasscallable.CallableFn {
	return func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		return value.Null, nil
	}
}

func makeParam(name string) *value.Parameter {
	return value.NewParameter(name, testSpan(), nil)
}

func makeParamWithDefault(name string, defaultValue value.Value) *value.Parameter {
	expr := value.NewValueExpression(defaultValue, testSpan())
	return value.NewParameter(name, testSpan(), expr)
}

// --- BuiltInCallable ---

func TestBuiltInCallableNew(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("test-fn", pl, noopCallback())

	if fn.Name() != "test-fn" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "test-fn")
	}
}

func TestBuiltInCallableNewOverloaded(t *testing.T) {
	pl1 := value.NewParameterList(nil, testSpan(), nil)
	pl2 := value.NewParameterList(nil, testSpan(), nil)
	overloads := []BuiltInCallableOverload{
		{Params: pl1, Fn: noopCallback()},
		{Params: pl2, Fn: noopCallback()},
	}
	fn := NewBuiltInCallableOverloaded("overloaded", overloads)

	if fn.Name() != "overloaded" {
		t.Errorf("Name() = %q, want %q", fn.Name(), "overloaded")
	}
}

func TestBuiltInCallableAcceptsContentDefault(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("test", pl, noopCallback())

	if fn.AcceptsContent() {
		t.Error("AcceptsContent should default to false")
	}
}

func TestBuiltInCallableSetAcceptsContent(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("test", pl, noopCallback())
	fn.SetAcceptsContent(true)

	if !fn.AcceptsContent() {
		t.Error("AcceptsContent should be true after SetAcceptsContent(true)")
	}
}

func TestBuiltInCallableWithName(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("original", pl, noopCallback())
	renamed := fn.WithName("renamed")

	if fn.Name() != "original" {
		t.Errorf("original Name() = %q, want %q", fn.Name(), "original")
	}
	if renamed.Name() != "renamed" {
		t.Errorf("renamed Name() = %q, want %q", renamed.Name(), "renamed")
	}
}

// --- BuiltInCallable.CallbackFor ---

func TestCallbackForExactMatchSingleOverload(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("test", pl, noopCallback())

	result, err := fn.CallbackFor(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params != pl {
		t.Error("CallbackFor should return the matching overload")
	}
}

func TestCallbackForExactMatchFirstWins(t *testing.T) {
	pl1 := value.NewParameterList(nil, testSpan(), nil)
	pl2 := value.NewParameterList(
		[]*value.Parameter{makeParamWithDefault("a", value.Null)},
		testSpan(),
		nil,
	)
	overloads := []BuiltInCallableOverload{
		{Params: pl1, Fn: noopCallback()},
		{Params: pl2, Fn: noopCallback()},
	}
	fn := NewBuiltInCallableOverloaded("test", overloads)

	result, err := fn.CallbackFor(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params != pl1 {
		t.Error("first overload should match for 0 positional, 0 named")
	}
}

func TestCallbackForExactMatchSecondOverload(t *testing.T) {
	pl1 := value.NewParameterList(
		[]*value.Parameter{makeParam("a")},
		testSpan(),
		nil,
	)
	pl2 := value.NewParameterList(nil, testSpan(), nil)
	overloads := []BuiltInCallableOverload{
		{Params: pl1, Fn: noopCallback()},
		{Params: pl2, Fn: noopCallback()},
	}
	fn := NewBuiltInCallableOverloaded("test", overloads)

	result, err := fn.CallbackFor(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params != pl2 {
		t.Error("second overload should match for 0 positional, 0 named (first has required param)")
	}
}

func TestCallbackForFuzzyMatchCloserDistance(t *testing.T) {
	// Both overloads fail exact match for 1 positional (required params, no defaults)
	// pl2: 2 required params, distance abs(2-1)=1
	// pl3: 3 required params, distance abs(3-1)=2
	// pl2 is closer → wins
	pl3 := value.NewParameterList(
		[]*value.Parameter{
			makeParam("a"),
			makeParam("b"),
			makeParam("c"),
		},
		testSpan(),
		nil,
	)
	pl2 := value.NewParameterList(
		[]*value.Parameter{
			makeParam("x"),
			makeParam("y"),
		},
		testSpan(),
		nil,
	)
	overloads := []BuiltInCallableOverload{
		{Params: pl3, Fn: noopCallback()},
		{Params: pl2, Fn: noopCallback()},
	}
	fn := NewBuiltInCallableOverloaded("test", overloads)

	result, err := fn.CallbackFor(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params != pl2 {
		t.Error("overload with closer mismatch distance should win")
	}
}

func TestCallbackForNoMatchEmptyOverloads(t *testing.T) {
	overloads := []BuiltInCallableOverload{}
	fn := NewBuiltInCallableOverloaded("test", overloads)

	_, err := fn.CallbackFor(0, nil)
	if err == nil {
		t.Fatal("expected error for empty overloads")
	}
	want := "BuiltInCallable test may not have empty overloads"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// --- PlainCssCallable ---

func TestPlainCssCallableName(t *testing.T) {
	pc := NewPlainCssCallable("rotate")
	if pc.Name() != "rotate" {
		t.Errorf("Name() = %q, want %q", pc.Name(), "rotate")
	}
}

func TestPlainCssCallableNew(t *testing.T) {
	pc := NewPlainCssCallable("calc")
	if pc == nil {
		t.Fatal("NewPlainCssCallable returned nil")
	}
	if pc.Name() != "calc" {
		t.Errorf("Name() = %q, want %q", pc.Name(), "calc")
	}
}

// --- UserDefinedCallable ---

func TestUserDefinedCallableNew(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fnRule := value.NewFunctionRule("my-fn", pl, nil, testSpan(), nil)
	env := sassenv.NewEnvironment()
	ud := NewUserDefinedCallable(fnRule, env, false)

	if ud.Name() != "my-fn" {
		t.Errorf("Name() = %q, want %q", ud.Name(), "my-fn")
	}
	if ud.Arguments() != pl {
		t.Error("Arguments should return the parameter list")
	}
	if _, isMixin := ud.Declaration().(*value.MixinRule); isMixin {
		t.Error("a FunctionRule declaration should not be a mixin")
	}
	if ud.InDependency() {
		t.Error("InDependency should be false")
	}
	if ud.Env() != env {
		t.Error("Env should return the environment")
	}
}

func TestUserDefinedCallableIsMixin(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	mixinRule := value.NewMixinRule("my-mixin", pl, nil, testSpan(), nil)
	env := sassenv.NewEnvironment()
	ud := NewUserDefinedCallable(mixinRule, env, false)

	if _, isMixin := ud.Declaration().(*value.MixinRule); !isMixin {
		t.Error("a MixinRule declaration should be a mixin")
	}
}

func TestUserDefinedCallableInDependency(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fnRule := value.NewFunctionRule("dep-fn", pl, nil, testSpan(), nil)
	env := sassenv.NewEnvironment()
	ud := NewUserDefinedCallable(fnRule, env, true)

	if !ud.InDependency() {
		t.Error("InDependency should be true")
	}
}

func TestUserDefinedCallableDeclaration(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	mixinRule := value.NewMixinRule("decl-fn", pl, nil, testSpan(), nil)
	env := sassenv.NewEnvironment()
	ud := NewUserDefinedCallable(mixinRule, env, false)

	if ud.Declaration() != mixinRule {
		t.Error("Declaration should return the declaration")
	}
}

// --- Callable interface dispatch ---

func TestBuiltInCallableImplementsCallable(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fn := NewBuiltInCallable("test", pl, noopCallback())
	var c sasscallable.Callable = fn
	if c.Name() != "test" {
		t.Errorf("Name() via interface = %q, want %q", c.Name(), "test")
	}
}

func TestUserDefinedCallableImplementsCallable(t *testing.T) {
	pl := value.NewParameterList(nil, testSpan(), nil)
	fnRule := value.NewFunctionRule("ud-fn", pl, nil, testSpan(), nil)
	env := sassenv.NewEnvironment()
	ud := NewUserDefinedCallable(fnRule, env, false)
	var c sasscallable.Callable = ud
	if c.Name() != "ud-fn" {
		t.Errorf("Name() via interface = %q, want %q", c.Name(), "ud-fn")
	}
}

func TestPlainCssCallableImplementsCallable(t *testing.T) {
	pc := NewPlainCssCallable("css-fn")
	var c sasscallable.Callable = pc
	if c.Name() != "css-fn" {
		t.Errorf("Name() via interface = %q, want %q", c.Name(), "css-fn")
	}
}
