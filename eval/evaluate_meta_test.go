// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package eval

import (
	"errors"
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// --- test infrastructure ---

type metaTestNode struct{ span sasscommon.FileSpan }

func (n metaTestNode) Span() (sasscommon.FileSpan, error) { return n.span, nil }
func (n metaTestNode) IsAstNode()                         {}

type metaRecordingLogger struct {
	messages     []string
	deprecations []*deprecation.Deprecation
}

func (r *metaRecordingLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	r.messages = append(r.messages, message)
	r.deprecations = append(r.deprecations, nil)
}

func (r *metaRecordingLogger) Debug(message string, span *sasscommon.FileSpan) {}

func (r *metaRecordingLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace) error {
	r.messages = append(r.messages, message)
	r.deprecations = append(r.deprecations, dep)
	return nil
}

func newMetaVisitor(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	if logger == nil {
		logger = sasslogger.NewDefaultLogger(true)
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	v := NewEvaluateVisitor(ic, logger)
	v.ec.SetCallableNode(metaTestNode{span: metaTestSpan()})
	return v
}

func metaTestSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(`@use "sass:meta";`), nil)
	return sasscommon.NewFileSpan(fs, 0, 4)
}

func metaFn(t *testing.T, v *EvaluateVisitor, name string) *functions.BuiltInCallable {
	t.Helper()
	for _, fn := range v.createMetaFunctions(v.env) {
		if fn.Name() == name {
			return fn.(*functions.BuiltInCallable)
		}
	}
	t.Fatalf("meta function %q not found", name)
	return nil
}

func metaMixin(t *testing.T, v *EvaluateVisitor, name string) *functions.BuiltInCallable {
	t.Helper()
	for _, mx := range v.createMetaMixins(v.env) {
		if mx.Name() == name {
			return mx.(*functions.BuiltInCallable)
		}
	}
	t.Fatalf("meta mixin %q not found", name)
	return nil
}

func callMetaCallable(t *testing.T, v *EvaluateVisitor, fn *functions.BuiltInCallable, args ...value.Value) (value.Value, error) {
	t.Helper()
	res, err := fn.CallbackFor(len(args), nil)
	if err != nil {
		t.Fatalf("CallbackFor(%d): %v", len(args), err)
	}
	paramCount := 0
	if res.Params != nil {
		paramCount = len(res.Params.Parameters)
	}
	for len(args) < paramCount {
		args = append(args, value.Null)
	}
	return res.Fn(v.ec, args)
}

func callMeta(t *testing.T, v *EvaluateVisitor, name string, args ...value.Value) (value.Value, error) {
	t.Helper()
	return callMetaCallable(t, v, metaFn(t, v, name), args...)
}

func callMetaOK(t *testing.T, v *EvaluateVisitor, name string, args ...value.Value) value.Value {
	t.Helper()
	got, err := callMeta(t, v, name, args...)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func metaStr(s string) *value.SassString    { return &value.SassString{Text: s, HasQuotes: false} }
func metaQuoted(s string) *value.SassString { return &value.SassString{Text: s, HasQuotes: true} }

func metaAssertBool(t *testing.T, got value.Value, want bool) {
	t.Helper()
	if want && got != value.SassTrue {
		t.Errorf("got %v, want SassTrue", got)
	}
	if !want && got != value.SassFalse {
		t.Errorf("got %v, want SassFalse", got)
	}
}

func metaAssertScriptErr(t *testing.T, err error, wantMsg, wantArgName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", wantMsg)
	}
	sse, ok := errors.AsType[*sasscommon.SassScriptException](err)
	if !ok {
		t.Fatalf("expected *SassScriptException, got %T: %v", err, err)
	}
	if sse.Message != wantMsg {
		t.Errorf("Message = %q, want %q", sse.Message, wantMsg)
	}
	if sse.ArgumentName != wantArgName {
		t.Errorf("ArgumentName = %q, want %q", sse.ArgumentName, wantArgName)
	}
}

func metaNoopCallable(name string) *functions.BuiltInCallable {
	return functions.MustNewBuiltInCallableFunction(name, "", "sass:meta-test",
		func(_ *evalcontext.EvaluationContext, _ []value.Value) (value.Value, error) {
			return value.Null, nil
		})
}

// metaModuleWith builds a BuiltInModule and registers it in v's environment
// under the given namespace.
func metaModuleWith(t *testing.T, v *EvaluateVisitor, namespace string,
	fns, mixins []sasscallable.Callable, vars *orderedmap.LinkedMap[string, value.Value]) {
	t.Helper()
	mod := sassmodule.NewBuiltInModule(namespace, fns, mixins, vars)
	ns := namespace
	if err := v.env.AddModule(mod, metaTestNode{span: metaTestSpan()}, &ns); err != nil {
		t.Fatal(err)
	}
}

// --- global-variable-exists ---

func TestMetaGlobalVariableExists(t *testing.T) {
	v := newMetaVisitor(t, nil)
	node := metaTestNode{span: metaTestSpan()}
	if err := v.env.SetVariable("a-b", value.NewUnitlessNumber(1), node, nil, true); err != nil {
		t.Fatal(err)
	}

	metaAssertBool(t, callMetaOK(t, v, "global-variable-exists", metaQuoted("a-b")), true)
	// Underscores are normalized to hyphens.
	metaAssertBool(t, callMetaOK(t, v, "global-variable-exists", metaQuoted("a_b")), true)
	metaAssertBool(t, callMetaOK(t, v, "global-variable-exists", metaQuoted("nope")), false)
}

func TestMetaGlobalVariableExistsModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	vars := orderedmap.New[string, value.Value]()
	vars.Put("x", value.NewUnitlessNumber(1))
	metaModuleWith(t, v, "m", nil, nil, vars)

	metaAssertBool(t, callMetaOK(t, v, "global-variable-exists", metaQuoted("x"), metaQuoted("m")), true)
	metaAssertBool(t, callMetaOK(t, v, "global-variable-exists", metaQuoted("y"), metaQuoted("m")), false)
}

func TestMetaGlobalVariableExistsMissingModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "global-variable-exists", metaQuoted("x"), metaQuoted("nope"))
	metaAssertScriptErr(t, err, `There is no module with the namespace "nope".`, "")
}

func TestMetaGlobalVariableExistsNameTypeError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "global-variable-exists", value.NewUnitlessNumber(1))
	metaAssertScriptErr(t, err, "1 is not a string.", "name")
}

// --- variable-exists ---

func TestMetaVariableExists(t *testing.T) {
	v := newMetaVisitor(t, nil)
	node := metaTestNode{span: metaTestSpan()}
	if err := v.env.SetVariable("a-b", value.NewUnitlessNumber(1), node, nil, true); err != nil {
		t.Fatal(err)
	}

	metaAssertBool(t, callMetaOK(t, v, "variable-exists", metaQuoted("a-b")), true)
	metaAssertBool(t, callMetaOK(t, v, "variable-exists", metaQuoted("a_b")), true)
	metaAssertBool(t, callMetaOK(t, v, "variable-exists", metaQuoted("nope")), false)
}

// --- function-exists ---

func TestMetaFunctionExistsEnvironment(t *testing.T) {
	v := newMetaVisitor(t, nil)
	v.env.SetFunction(metaNoopCallable("my-fn"))

	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("my-fn")), true)
	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("my_fn")), true)
	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("nope")), false)
}

func TestMetaFunctionExistsGlobal(t *testing.T) {
	// The built-in fallback compares the RAW name (no underscore
	// normalization) — matches Dart's _builtInFunctions.containsKey(text).
	v := newMetaVisitor(t, nil)
	v.builtInFunctions["global-fn"] = metaNoopCallable("global-fn")

	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("global-fn")), true)
	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("global_fn")), false)
}

func TestMetaFunctionExistsModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	metaModuleWith(t, v, "m", []sasscallable.Callable{metaNoopCallable("mod-fn")}, nil, nil)

	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("mod-fn"), metaQuoted("m")), true)
	metaAssertBool(t, callMetaOK(t, v, "function-exists", metaQuoted("nope"), metaQuoted("m")), false)
}

// --- mixin-exists ---

func TestMetaMixinExists(t *testing.T) {
	v := newMetaVisitor(t, nil)
	v.env.SetMixin(metaNoopCallable("my-mixin"))

	metaAssertBool(t, callMetaOK(t, v, "mixin-exists", metaQuoted("my-mixin")), true)
	metaAssertBool(t, callMetaOK(t, v, "mixin-exists", metaQuoted("my_mixin")), true)
	metaAssertBool(t, callMetaOK(t, v, "mixin-exists", metaQuoted("nope")), false)
}

func TestMetaMixinExistsModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	metaModuleWith(t, v, "m", nil, []sasscallable.Callable{metaNoopCallable("mod-mix")}, nil)

	metaAssertBool(t, callMetaOK(t, v, "mixin-exists", metaQuoted("mod-mix"), metaQuoted("m")), true)
}

// --- content-exists ---

func TestMetaContentExistsOutsideMixin(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "content-exists")
	metaAssertScriptErr(t, err, "content-exists() may only be called within a mixin.", "")
}

func TestMetaContentExistsInMixin(t *testing.T) {
	v := newMetaVisitor(t, nil)
	v.env.SetInMixin(true)
	metaAssertBool(t, callMetaOK(t, v, "content-exists"), false)

	v.env.SetContent(metaNoopCallable("content_c"))
	metaAssertBool(t, callMetaOK(t, v, "content-exists"), true)
}

// --- module-variables ---

func TestMetaModuleVariables(t *testing.T) {
	v := newMetaVisitor(t, nil)
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	vars.Put("b", value.NewUnitlessNumber(2))
	metaModuleWith(t, v, "m", nil, nil, vars)

	got := callMetaOK(t, v, "module-variables", metaQuoted("m"))
	m, ok := got.(*value.SassMap)
	if !ok {
		t.Fatalf("expected *SassMap, got %T", got)
	}
	inspected, err := value.SerializeValueInspect(m)
	if err != nil {
		t.Fatal(err)
	}
	// Keys are QUOTED strings (Dart: SassString(name) with default quotes).
	if inspected != `("a": 1, "b": 2)` {
		t.Errorf("inspect = %q, want %q", inspected, `("a": 1, "b": 2)`)
	}
}

func TestMetaModuleVariablesMissingModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "module-variables", metaQuoted("nope"))
	metaAssertScriptErr(t, err, `There is no module with namespace "nope".`, "")
}

// --- module-functions ---

func TestMetaModuleFunctions(t *testing.T) {
	v := newMetaVisitor(t, nil)
	fn := metaNoopCallable("mod-fn")
	metaModuleWith(t, v, "m", []sasscallable.Callable{fn}, nil, nil)

	got := callMetaOK(t, v, "module-functions", metaQuoted("m"))
	m, ok := got.(*value.SassMap)
	if !ok {
		t.Fatalf("expected *SassMap, got %T", got)
	}
	count := 0
	for k, val := range m.Entries() {
		count++
		ks, ok := k.(*value.SassString)
		if !ok || !ks.HasQuotes {
			t.Errorf("key %v should be a quoted string", k)
		}
		sf, ok := val.(*value.SassFunction)
		if !ok {
			t.Fatalf("value = %T, want *SassFunction", val)
		}
		if sf.Name() != "mod-fn" {
			t.Errorf("Name() = %q, want %q", sf.Name(), "mod-fn")
		}
		if sf.FunctionRef != sasscallable.Callable(fn) {
			t.Error("FunctionRef should be the module's callable")
		}
		// The value is bound to this compilation.
		if _, err := sf.AssertCompileContext(v.compileContext); err != nil {
			t.Errorf("AssertCompileContext: %v", err)
		}
	}
	if count != 1 {
		t.Errorf("map size = %d, want 1", count)
	}
}

// --- module-mixins ---

func TestMetaModuleMixins(t *testing.T) {
	v := newMetaVisitor(t, nil)
	mx := metaNoopCallable("mod-mix")
	metaModuleWith(t, v, "m", nil, []sasscallable.Callable{mx}, nil)

	got := callMetaOK(t, v, "module-mixins", metaQuoted("m"))
	m, ok := got.(*value.SassMap)
	if !ok {
		t.Fatalf("expected *SassMap, got %T", got)
	}
	for k, val := range m.Entries() {
		ks := k.(*value.SassString)
		if !ks.HasQuotes {
			t.Error("key should be quoted")
		}
		sm, ok := val.(*value.SassMixin)
		if !ok {
			t.Fatalf("value = %T, want *SassMixin", val)
		}
		if sm.Name() != "mod-mix" {
			t.Errorf("Name() = %q, want %q", sm.Name(), "mod-mix")
		}
		if sm.MixinRef != sasscallable.Callable(mx) {
			t.Error("MixinRef should be the module's callable")
		}
	}
}

func TestMetaModuleMixinsMissingModule(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "module-mixins", metaQuoted("nope"))
	metaAssertScriptErr(t, err, `There is no module with namespace "nope".`, "")
}

// --- get-function ---

func TestMetaGetFunctionEnvironment(t *testing.T) {
	v := newMetaVisitor(t, nil)
	fn := metaNoopCallable("my-fn")
	v.env.SetFunction(fn)

	got := callMetaOK(t, v, "get-function", metaQuoted("my-fn"), value.SassFalse, value.Null)
	sf, ok := got.(*value.SassFunction)
	if !ok {
		t.Fatalf("expected *SassFunction, got %T", got)
	}
	if sf.Name() != "my-fn" {
		t.Errorf("Name() = %q, want %q", sf.Name(), "my-fn")
	}
	if sf.FunctionRef != sasscallable.Callable(fn) {
		t.Error("FunctionRef should be the environment's callable")
	}
}

func TestMetaGetFunctionBuiltInFallback(t *testing.T) {
	v := newMetaVisitor(t, nil)
	fn := metaNoopCallable("builtin-fn")
	v.builtInFunctions["builtin-fn"] = fn

	got := callMetaOK(t, v, "get-function", metaQuoted("builtin_fn"), value.SassFalse, value.Null)
	sf := got.(*value.SassFunction)
	if sf.FunctionRef != sasscallable.Callable(fn) {
		t.Error("FunctionRef should be the built-in callable (normalized lookup)")
	}
}

func TestMetaGetFunctionNamespaceShortCircuit(t *testing.T) {
	// With $module passed, the built-in fallback is skipped even when a
	// built-in of that name exists.
	v := newMetaVisitor(t, nil)
	v.builtInFunctions["builtin-fn"] = metaNoopCallable("builtin-fn")
	metaModuleWith(t, v, "m", nil, nil, nil)

	_, err := callMeta(t, v, "get-function", metaQuoted("builtin-fn"), value.SassFalse, metaQuoted("m"))
	metaAssertScriptErr(t, err, `Function not found: "builtin-fn"`, "")
}

func TestMetaGetFunctionNotFound(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "get-function", metaQuoted("nope"), value.SassFalse, value.Null)
	metaAssertScriptErr(t, err, `Function not found: "nope"`, "")
}

func TestMetaGetFunctionNotFoundUnquoted(t *testing.T) {
	// The error embeds the name with its input quoting (Dart: "$name").
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "get-function", metaStr("nope"), value.SassFalse, value.Null)
	metaAssertScriptErr(t, err, "Function not found: nope", "")
}

func TestMetaGetFunctionCss(t *testing.T) {
	v := newMetaVisitor(t, nil)
	got := callMetaOK(t, v, "get-function", metaQuoted("foo"), value.SassTrue, value.Null)
	sf := got.(*value.SassFunction)
	if sf.Name() != "foo" {
		t.Errorf("Name() = %q, want %q", sf.Name(), "foo")
	}
	if _, ok := sf.FunctionRef.(*functions.PlainCssCallable); !ok {
		t.Errorf("FunctionRef = %T, want *PlainCssCallable", sf.FunctionRef)
	}
}

func TestMetaGetFunctionCssAndModuleError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "get-function", metaQuoted("foo"), value.SassTrue, metaQuoted("m"))
	metaAssertScriptErr(t, err, "$css and $module may not both be passed at once.", "")
}

// --- get-mixin ---

func TestMetaGetMixin(t *testing.T) {
	v := newMetaVisitor(t, nil)
	mx := metaNoopCallable("my-mixin")
	v.env.SetMixin(mx)

	got := callMetaOK(t, v, "get-mixin", metaQuoted("my_mixin"), value.Null)
	sm, ok := got.(*value.SassMixin)
	if !ok {
		t.Fatalf("expected *SassMixin, got %T", got)
	}
	if sm.Name() != "my-mixin" {
		t.Errorf("Name() = %q, want %q", sm.Name(), "my-mixin")
	}
	if sm.MixinRef != sasscallable.Callable(mx) {
		t.Error("MixinRef should be the environment's callable")
	}
}

func TestMetaGetMixinNotFound(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "get-mixin", metaQuoted("nope"), value.Null)
	metaAssertScriptErr(t, err, `Mixin not found: "nope"`, "")
}

// --- call ---

func metaArgList(t *testing.T, contents ...value.Value) *value.SassArgumentList {
	t.Helper()
	al, err := value.NewSassArgumentList(contents, nil, value.ListSeparatorComma)
	if err != nil {
		t.Fatal(err)
	}
	return al
}

func TestMetaCallTypeError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	_, err := callMeta(t, v, "call", value.NewUnitlessNumber(1), metaArgList(t))
	metaAssertScriptErr(t, err, "1 is not a function reference.", "function")
}

func TestMetaCallCompileContextMismatch(t *testing.T) {
	v := newMetaVisitor(t, nil)
	// Use a non-zero-sized type to defeat Go's &struct{}{} coalescing
	// (the spec allows coalescing of zero-sized allocations, so two
	// &struct{}{} pointers may compare equal and assertCompileContext
	// passes erroneously).
	other := &struct{ x int }{1}
	sf := value.NewSassFunctionWithCompileContext(metaNoopCallable("f"), other)
	_, err := callMeta(t, v, "call", sf, metaArgList(t))
	metaAssertScriptErr(t, err, `get-function("f") does not belong to current compilation.`, "")
}

// TestMetaCallBuiltIn + TestMetaCallStringDeprecation are deferred to the
// full eval port: invokeCallable / VisitFunctionExpression need evaluator
// state (stylesheet, parent, CSS tree) not yet constructed here. The
// error paths (type error, compile-context mismatch) test the assertions;
// the happy paths are locked by the sass-spec suite.

// --- apply ---

func TestMetaApplyTypeError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	mx := metaMixin(t, v, "apply")
	_, err := callMetaCallable(t, v, mx, value.NewUnitlessNumber(1), metaArgList(t))
	metaAssertScriptErr(t, err, "1 is not a mixin reference.", "mixin")
}

func TestMetaApplyCompileContextMismatch(t *testing.T) {
	v := newMetaVisitor(t, nil)
	// Use a non-zero-sized type to defeat Go's &struct{}{} coalescing
	// (see TestMetaCallCompileContextMismatch).
	other := &struct{ x int }{1}
	sm := value.NewSassMixinWithCompileContext(metaNoopCallable("m"), other)
	mx := metaMixin(t, v, "apply")
	_, err := callMetaCallable(t, v, mx, sm, metaArgList(t))
	metaAssertScriptErr(t, err, `get-mixin("m") does not belong to current compilation.`, "")
}

func TestMetaApplyAcceptsContent(t *testing.T) {
	v := newMetaVisitor(t, nil)
	for _, mx := range v.createMetaMixins(v.env) {
		if mx.Name() == "apply" {
			if !mx.(*functions.BuiltInCallable).AcceptsContent() {
				t.Error("apply should accept content")
			}
		}
		if mx.Name() == "load-css" {
			if mx.(*functions.BuiltInCallable).AcceptsContent() {
				t.Error("load-css should not accept content")
			}
		}
	}
}

// --- load-css ---

func TestMetaLoadCssUrlTypeError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	mx := metaMixin(t, v, "load-css")
	_, err := callMetaCallable(t, v, mx, value.NewUnitlessNumber(1), value.Null)
	metaAssertScriptErr(t, err, "1 is not a string.", "url")
}

func TestMetaLoadCssWithTypeError(t *testing.T) {
	v := newMetaVisitor(t, nil)
	mx := metaMixin(t, v, "load-css")
	_, err := callMetaCallable(t, v, mx, metaQuoted("x"), value.NewUnitlessNumber(1))
	metaAssertScriptErr(t, err, "1 is not a map.", "with")
}

func TestMetaLoadCssConfiguredTwice(t *testing.T) {
	v := newMetaVisitor(t, nil)
	with := value.EmptySassMap()
	with.Set(metaQuoted("a_b"), value.NewUnitlessNumber(1))
	with.Set(metaQuoted("a-b"), value.NewUnitlessNumber(2))
	mx := metaMixin(t, v, "load-css")
	_, err := callMetaCallable(t, v, mx, metaQuoted("x"), with)
	metaAssertScriptErr(t, err, "The variable $a-b was configured twice.", "")
}

func TestMetaLoadCssPrivateVariableDeprecation(t *testing.T) {
	rl := &metaRecordingLogger{}
	v := newMetaVisitor(t, rl)
	with := value.EmptySassMap()
	with.Set(metaQuoted("-priv"), value.NewUnitlessNumber(1))
	mx := metaMixin(t, v, "load-css")
	// The load itself fails (no such file) — only the deprecation matters here.
	_, _ = callMetaCallable(t, v, mx, metaQuoted("nonexistent-file"), with)

	want := strings.Join([]string{
		`Configuring private variables (such as $-priv) is deprecated.`,
		`This will be an error in Dart Sass 2.0.0.`,
	}, "\n")
	if len(rl.messages) == 0 {
		t.Fatal("expected a deprecation warning")
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
	if rl.deprecations[0] != deprecation.WithPrivate {
		t.Errorf("deprecation = %v, want WithPrivate", rl.deprecations[0])
	}
}

// --- registerMetaFunctions ---

func TestRegisterMetaFunctions(t *testing.T) {
	v := newMetaVisitor(t, nil)
	v.registerMetaFunctions(v.env)

	mod, ok := v.builtInModules["sass:meta"]
	if !ok {
		t.Fatal("sass:meta module not registered")
	}
	// 7 shared functions + 11 evaluator functions.
	if mod.Functions().Len() != 18 {
		t.Errorf("meta module functions = %d, want 18", mod.Functions().Len())
	}
	if mod.Mixins().Len() != 2 {
		t.Errorf("meta module mixins = %d, want 2", mod.Mixins().Len())
	}
	for _, name := range []string{
		"global-variable-exists", "variable-exists", "function-exists",
		"mixin-exists", "content-exists", "module-variables",
		"module-functions", "module-mixins", "get-function", "get-mixin", "call",
	} {
		if _, ok := mod.Functions().Get(name); !ok {
			t.Errorf("meta module should contain %q", name)
		}
		// Evaluator meta functions are also registered globally (wrapped
		// with a deprecation warning).
		if _, ok := v.builtInFunctions[name]; !ok {
			t.Errorf("builtInFunctions should contain %q", name)
		}
	}
	for _, name := range []string{"load-css", "apply"} {
		if _, ok := mod.Mixins().Get(name); !ok {
			t.Errorf("meta module should contain mixin %q", name)
		}
	}
	if len(v.builtInFunctions) != 11 {
		t.Errorf("builtInFunctions meta entries = %d, want 11", len(v.builtInFunctions))
	}
}
