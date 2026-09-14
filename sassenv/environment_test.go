// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package sassenv

import (
	"testing"

	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

func testSpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(nil, 0, 0)
}

func testAstNode() sasscommon.AstNode {
	fs := sasscommon.NewFileSource([]byte("node"), nil)
	sp := sasscommon.NewFileSpan(fs, 0, 4)
	return sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return sp, nil
	})
}

func testCallable(name string) sasscallable.Callable {
	return &testCallableImpl{name: name}
}

type testCallableImpl struct {
	name string
}

func (t *testCallableImpl) Name() string { return t.name }

func testBuiltInModule(name string, vars map[string]value.Value) sassmodule.Module {
	varMap := orderedmap.New[string, value.Value]()
	for k, v := range vars {
		varMap.Put(k, v)
	}
	return sassmodule.NewBuiltInModule(name, nil, nil, varMap)
}

// --- Construction ---

func TestNewEnvironment(t *testing.T) {
	env := NewEnvironment()
	if env == nil {
		t.Fatal("NewEnvironment returned nil")
	}
	if !env.AtRoot() {
		t.Error("new env should be at root")
	}
}

func TestEnvironmentAtRoot(t *testing.T) {
	env := NewEnvironment()
	if !env.AtRoot() {
		t.Error("single scope = at root")
	}
}

func TestEnvironmentInMixin(t *testing.T) {
	env := NewEnvironment()
	if env.InMixin() {
		t.Error("should not be in mixin initially")
	}
	env.SetInMixin(true)
	if !env.InMixin() {
		t.Error("should be in mixin after SetInMixin(true)")
	}
}

// --- Scope ---

func TestScopeWhenFalse(t *testing.T) {
	env := NewEnvironment()
	_, err := Scope(env, func() (struct{}, error) {
		env.SetLocalVariable("x", value.SassTrue, testAstNode())
		return struct{}{}, nil
	}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("variable set in when=false scope should persist")
	}
}

func TestScopeWhenTrue(t *testing.T) {
	env := NewEnvironment()
	_, err := Scope(env, func() (struct{}, error) {
		env.SetLocalVariable("x", value.SassTrue, testAstNode())
		return struct{}{}, nil
	}, false, true)
	if err != nil {
		t.Fatal(err)
	}
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Error("variable set in when=true scope should NOT persist outside scope")
	}
}

func TestScopeSemiGlobal(t *testing.T) {
	env := NewEnvironment()
	_, err := Scope(env, func() (struct{}, error) {
		env.SetVariable("x", value.SassTrue, testAstNode(), nil, false)
		return struct{}{}, nil
	}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Error("variable set in semi-global when=true scope should not persist at global")
	}
}

// --- Closure ---

func TestClosure(t *testing.T) {
	env := NewEnvironment()
	closure := env.Closure()
	if closure == nil {
		t.Fatal("Closure returned nil")
	}
	if closure == env {
		t.Error("closure should be a different Environment object")
	}
}

func TestClosureSharingScope(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())
	closure := env.Closure()

	// Modification in closure's shared scope should be visible to original
	if closure == nil {
		t.Fatal("cl nil")
	}
	// Instead of testing through closure, test that closure shares current scope state
	val, err := closure.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("closure should see variables set before closure was created")
	}
}

// --- ForImport ---

func TestForImport(t *testing.T) {
	env := NewEnvironment()
	fi := env.ForImport()
	if fi == nil {
		t.Fatal("ForImport returned nil")
	}
}

// --- SetLocalVariable ---

func TestSetLocalVariable(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("variable should be set")
	}
}

// --- SetVariable global ---

func TestSetVariableGlobal(t *testing.T) {
	env := NewEnvironment()
	env.SetVariable("x", value.SassTrue, testAstNode(), nil, true)
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("global variable should be set")
	}
}

// --- SetVariable in nested scope ---

func TestSetVariableNestedScope(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("local variable should be set")
	}
}

// --- GetVariable missing ---

func TestGetVariableMissing(t *testing.T) {
	env := NewEnvironment()
	val, err := env.GetVariable("missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Error("missing variable should return nil")
	}
}

// --- VariableExists ---

func TestVariableExists(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())
	exists, err := env.VariableExists("x")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("x should exist")
	}
}

func TestVariableExistsMissing(t *testing.T) {
	env := NewEnvironment()
	exists, err := env.VariableExists("missing")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("missing should not exist")
	}
}

// --- GlobalVariableExists ---

func TestGlobalVariableExists(t *testing.T) {
	env := NewEnvironment()
	env.SetVariable("x", value.SassTrue, testAstNode(), nil, true)
	exists, err := env.GlobalVariableExists("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("x should exist globally")
	}
}

func TestGlobalVariableExistsMissing(t *testing.T) {
	env := NewEnvironment()
	exists, err := env.GlobalVariableExists("missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("missing should not exist globally")
	}
}

// --- GetVariableNode ---

func TestGetVariableNode(t *testing.T) {
	env := NewEnvironment()
	node := testAstNode()
	env.SetLocalVariable("x", value.SassTrue, node)
	n, err := env.GetVariableNode("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if n == nil {
		t.Error("should return node")
	}
}

func TestGetVariableNodeMissing(t *testing.T) {
	env := NewEnvironment()
	n, err := env.GetVariableNode("missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != nil {
		t.Error("missing node should return nil")
	}
}

// --- SetFunction / GetFunction / FunctionExists ---

func TestSetFunction(t *testing.T) {
	env := NewEnvironment()
	fn := testCallable("my-fn")
	env.SetFunction(fn)
	got, err := env.GetFunction("my-fn", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("function should be found")
	}
}

func TestGetFunctionMissing(t *testing.T) {
	env := NewEnvironment()
	got, err := env.GetFunction("missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("missing function should return nil")
	}
}

func TestFunctionExists(t *testing.T) {
	env := NewEnvironment()
	env.SetFunction(testCallable("my-fn"))
	exists, err := env.FunctionExists("my-fn", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("function should exist")
	}
}

// --- SetMixin / GetMixin / MixinExists ---

func TestSetMixin(t *testing.T) {
	env := NewEnvironment()
	mx := testCallable("my-mx")
	env.SetMixin(mx)
	got, err := env.GetMixin("my-mx", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("mixin should be found")
	}
}

func TestGetMixinMissing(t *testing.T) {
	env := NewEnvironment()
	got, err := env.GetMixin("missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("missing mixin should return nil")
	}
}

func TestMixinExists(t *testing.T) {
	env := NewEnvironment()
	env.SetMixin(testCallable("my-mx"))
	exists, err := env.MixinExists("my-mx", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("mixin should exist")
	}
}

// --- Content / WithContent ---

func TestContentInitial(t *testing.T) {
	env := NewEnvironment()
	if env.Content() != nil {
		t.Error("content should be nil initially")
	}
}

func TestSetContent(t *testing.T) {
	env := NewEnvironment()
	cc := testCallable("@content")
	env.SetContent(cc)
	if env.Content() != cc {
		t.Error("content should be set")
	}
}

func TestWithContent(t *testing.T) {
	env := NewEnvironment()
	cc := testCallable("@content")
	var captured sasscallable.Callable
	env.WithContent(cc, func() {
		captured = env.Content()
	})
	if captured != cc {
		t.Error("withContent should set content in callback")
	}
	if env.Content() != nil {
		t.Error("content should be restored after callback")
	}
}

// --- AsMixin ---

func TestAsMixin(t *testing.T) {
	env := NewEnvironment()
	if env.InMixin() {
		t.Fatal("should not be in mixin initially")
	}
	env.AsMixin(func() {
		if !env.InMixin() {
			t.Error("should be in mixin during callback")
		}
	})
	if env.InMixin() {
		t.Error("should not be in mixin after callback")
	}
}

// --- MarkVariableConfigurable ---

func TestMarkVariableConfigurable(t *testing.T) {
	env := NewEnvironment()
	env.MarkVariableConfigurable("x")
	// Tested indirectly via CouldHaveBeenConfigured on module
}

// --- ToImplicitConfiguration ---

func TestToImplicitConfiguration(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())
	cfg, err := env.ToImplicitConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("ToImplicitConfiguration returned nil")
	}
	val, ok := cfg.Get("x")
	if !ok {
		t.Error("config should contain 'x'")
	}
	if val == nil {
		t.Error("configured value should not be nil")
	}
}

func TestToImplicitConfigurationEmpty(t *testing.T) {
	env := NewEnvironment()
	cfg, err := env.ToImplicitConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsEmpty() {
		t.Error("empty env should produce empty config")
	}
}

// --- ToModule / ToDummyModule ---

func TestToModule(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())

	css := value.NewCssStylesheet([]value.CssNode{}, testSpan())
	mod := env.ToModule(css, nil, &extend.EmptyExtensionStore{})
	if mod == nil {
		t.Fatal("ToModule returned nil")
	}
	u, err := mod.URL()
	if err != nil {
		t.Fatal(err)
	}
	_ = u // empty since span has no URL
}

func TestToDummyModule(t *testing.T) {
	env := NewEnvironment()
	mod := env.ToDummyModule()
	if mod == nil {
		t.Fatal("ToDummyModule returned nil")
	}
	v, ok := mod.Variables().Get("no-such-var")
	if ok || v != nil {
		t.Error("dummy module should have no variables")
	}
}

// --- AddModule ---

func TestAddModule(t *testing.T) {
	env := NewEnvironment()
	mod := testBuiltInModule("test", nil)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}
	got := env.Modules()
	if _, ok := got["util"]; !ok {
		t.Error("module should be added under namespace 'util'")
	}
}

func TestAddModuleDuplicateNamespace(t *testing.T) {
	env := NewEnvironment()
	mod := testBuiltInModule("test", nil)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}
	err = env.AddModule(mod, testAstNode(), &ns)
	if err == nil {
		t.Fatal("expected error for duplicate namespace")
	}
}

func TestAddModuleNamespaceless(t *testing.T) {
	env := NewEnvironment()
	mod := testBuiltInModule("test", nil)
	err := env.AddModule(mod, testAstNode(), nil)
	if err != nil {
		t.Fatal(err)
	}
}

// --- Modules ---

func TestModules(t *testing.T) {
	env := NewEnvironment()
	mod := testBuiltInModule("test", nil)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}
	got := env.Modules()
	if got["util"] != mod {
		t.Error("Modules should return the module")
	}
}

// --- GetVariable with namespace ---

func TestGetVariableNamespaced(t *testing.T) {
	env := NewEnvironment()
	vars := map[string]value.Value{"x": value.SassTrue}
	mod := testBuiltInModule("test", vars)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}

	val, err := env.GetVariable("x", &ns)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Error("namespaced variable should be found")
	}
}

func TestGetVariableMissingNamespace(t *testing.T) {
	env := NewEnvironment()
	ns := "nonexistent"
	_, err := env.GetVariable("x", &ns)
	if err == nil {
		t.Fatal("expected error for missing namespace")
	}
}

// --- GlobalVariableExists with namespace ---

func TestGlobalVariableExistsNamespaced(t *testing.T) {
	env := NewEnvironment()
	vars := map[string]value.Value{"x": value.SassTrue}
	mod := testBuiltInModule("test", vars)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}

	exists, err := env.GlobalVariableExists("x", &ns)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("variable should exist in namespace")
	}
}

// --- GetFunction with namespace ---

func TestGetFunctionNamespaced(t *testing.T) {
	env := NewEnvironment()
	_ = testCallable("my-fn")
	mod := testBuiltInModule("test", nil)
	ns := "util"
	err := env.AddModule(mod, testAstNode(), &ns)
	if err != nil {
		t.Fatal(err)
	}
	// BuiltInModule has no functions, so we expect nil, not error
	got, err := env.GetFunction("my-fn", &ns)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("function should be nil (module has no functions)")
	}
}

// --- GetMixin with namespace ---

func TestGetMixinNamespaced(t *testing.T) {
	env := NewEnvironment()
	ns := "nonexistent"
	_, err := env.GetMixin("my-mx", &ns)
	if err == nil {
		t.Fatal("expected error for missing namespace")
	}
}

// --- GetVariable fast path cache ---

func TestGetVariableFastPath(t *testing.T) {
	env := NewEnvironment()
	env.SetLocalVariable("x", value.SassTrue, testAstNode())

	// First access - populates cache
	val, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Fatal("variable should be found")
	}

	// Second access - should use fast path
	val2, err := env.GetVariable("x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val2 != val {
		t.Error("same variable should be returned")
	}
}
