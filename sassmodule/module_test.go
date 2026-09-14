// Copyright 2019 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package sassmodule

import (
	"net/url"
	"testing"

	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func testSpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(nil, 0, 0)
}

func testURL() *url.URL {
	u, _ := url.Parse("file:///test.scss")
	return u
}

// testCallable is a minimal Callable implementation for tests.
type testCallable struct {
	name string
}

func (t *testCallable) Name() string { return t.name }

func newTestCallable(name string) sasscallable.Callable {
	return &testCallable{name: name}
}

// --- BuiltInModule ---

func TestBuiltInModuleNew(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)

	u, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u != "sass:math" {
		t.Errorf("URL() = %q, want %q", u, "sass:math")
	}
}

func TestBuiltInModuleUpstream(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	if len(m.Upstream()) != 0 {
		t.Error("upstream should be empty")
	}
}

func TestBuiltInModuleVariables(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("pi", value.NewUnitlessNumber(3.14))
	m := NewBuiltInModule("math", nil, nil, vars)

	v, ok := m.Variables().Get("pi")
	if !ok {
		t.Fatal("variable 'pi' not found")
	}
	n := v.(value.SassNumber)
	if n.NumValue() != 3.14 {
		t.Errorf("pi = %v, want 3.14", n.NumValue())
	}
}

func TestBuiltInModuleFunctions(t *testing.T) {
	fn := newTestCallable("abs")
	m := NewBuiltInModule("math", []sasscallable.Callable{fn}, nil, nil)

	f, ok := m.Functions().Get("abs")
	if !ok {
		t.Fatal("function 'abs' not found")
	}
	if f.Name() != "abs" {
		t.Errorf("Name() = %q, want %q", f.Name(), "abs")
	}
}

func TestBuiltInModuleMixins(t *testing.T) {
	mx := newTestCallable("apply")
	m := NewBuiltInModule("meta", nil, []sasscallable.Callable{mx}, nil)

	mix, ok := m.Mixins().Get("apply")
	if !ok {
		t.Fatal("mixin 'apply' not found")
	}
	if mix.Name() != "apply" {
		t.Errorf("Name() = %q, want %q", mix.Name(), "apply")
	}
}

func TestBuiltInModuleCSS(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	css, err := m.CSS()
	if err != nil {
		t.Fatal(err)
	}
	if len(css.Children()) != 0 {
		t.Error("CSS should be empty")
	}
}

func TestBuiltInModuleExtensionStore(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	store := m.ExtensionStore()
	if _, ok := store.(*extend.EmptyExtensionStore); !ok {
		t.Error("ExtensionStore should be empty")
	}
}

func TestBuiltInModuleTransitivelyContains(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	if m.TransitivelyContainsCss() {
		t.Error("should not transitively contain CSS")
	}
	if m.TransitivelyContainsExtensions() {
		t.Error("should not transitively contain extensions")
	}
}

func TestBuiltInModuleSetVariableUndefined(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	err := m.SetVariable("x", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
	want := "Undefined variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestBuiltInModuleSetVariableCannotModify(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("x", value.NewUnitlessNumber(1))
	m := NewBuiltInModule("math", nil, nil, vars)
	err := m.SetVariable("x", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	want := "Cannot modify built-in variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestBuiltInModuleVariableIdentity(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("x", value.NewUnitlessNumber(1))
	m := NewBuiltInModule("math", nil, nil, vars)

	id, err := m.VariableIdentity("x")
	if err != nil {
		t.Fatal(err)
	}
	if id != m {
		t.Error("VariableIdentity should return the module itself")
	}
}

func TestBuiltInModuleCouldHaveBeenConfigured(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	if m.CouldHaveBeenConfigured(map[string]struct{}{"x": {}}) {
		t.Error("BuiltInModule should never be configurable")
	}
}

func TestBuiltInModuleCloneCss(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	cloned, err := m.CloneCss()
	if err != nil {
		t.Fatal(err)
	}
	if cloned != m {
		t.Error("cloneCss of BuiltInModule should return self")
	}
}

// --- ForwardedModuleView ---

func TestForwardedModuleViewNew(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)
	rule := value.NewForwardRule(testURL(), testSpan(), nil, nil)
	view := NewForwardedModuleView(inner, rule)

	v, ok := view.Variables().Get("a")
	if !ok {
		t.Fatal("variable 'a' not forwarded")
	}
	n := v.(value.SassNumber)
	if n.NumValue() != 1.0 {
		t.Errorf("a = %v, want 1.0", n.NumValue())
	}
}

func TestForwardedModuleViewIfNecessaryReturnsInner(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	rule := value.NewForwardRule(testURL(), testSpan(), nil, nil)
	result := ForwardedModuleViewIfNecessary(inner, rule)

	if result != inner {
		t.Error("no prefix/show/hide → should return inner as-is")
	}
}

func TestForwardedModuleViewIfNecessaryWraps(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	pfx := "ns-"
	rule := value.NewForwardRule(testURL(), testSpan(), &pfx, nil)
	result := ForwardedModuleViewIfNecessary(inner, rule)

	if result == inner {
		t.Error("with prefix → should wrap in ForwardedModuleView")
	}
}

func TestForwardedModuleViewPrefix(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	vars.Put("b", value.NewUnitlessNumber(2))
	inner := NewBuiltInModule("test", nil, nil, vars)

	pfx := "ns-"
	rule := value.NewForwardRule(testURL(), testSpan(), &pfx, nil)
	view := NewForwardedModuleView(inner, rule)

	// Go PrefixedMapView ADDS prefix: inner key "a" → visible as "ns-a"
	v, ok := view.Variables().Get("ns-a")
	if !ok {
		t.Fatal("prefixed variable 'ns-a' not found")
	}
	n := v.(value.SassNumber)
	if n.NumValue() != 1.0 {
		t.Errorf("a = %v, want 1.0", n.NumValue())
	}

	// "b" should also be visible (no safelist/blocklist, with prefix)
	_, ok = view.Variables().Get("ns-b")
	if !ok {
		t.Errorf("prefixed 'ns-b' should be visible when no safelist/blocklist")
	}

	// Original unprefixed key should not be directly visible
	_, ok = view.Variables().Get("a")
	if ok {
		t.Error("unprefixed 'a' should not be directly visible through prefixed view")
	}
}

func TestForwardedModuleViewUrlDelegates(t *testing.T) {
	inner := NewBuiltInModule("math", nil, nil, nil)
	view := NewForwardedModuleView(inner, value.NewForwardRule(testURL(), testSpan(), nil, nil))

	u, err := view.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u != "sass:math" {
		t.Errorf("URL() = %q, want %q", u, "sass:math")
	}
}

func TestForwardedModuleViewSetVariablePrefix(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)
	pfx := "ns-"
	rule := value.NewForwardRule(testURL(), testSpan(), &pfx, nil)
	view := NewForwardedModuleView(inner, rule)

	// PrefixedMapView ADDS prefix: inner "a" visible as "ns-a"
	// Trying to set "b" (no inner key) → undefined
	err := view.SetVariable("b", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error setting non-existent variable")
	}
	want := "Undefined variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestForwardedModuleViewSetVariableShown(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	vars.Put("b", value.NewUnitlessNumber(2))
	inner := NewBuiltInModule("test", nil, nil, vars)

	shown := orderedset.New[string]()
	shown.Add("a")
	rule := value.NewForwardRuleShow(testURL(), nil, shown, testSpan(), nil, nil)
	view := NewForwardedModuleView(inner, rule)

	// "b" should not be visible through show filter
	err := view.SetVariable("b", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error setting unshown variable")
	}
	want := "Undefined variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestForwardedModuleViewVariableIdentity(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)
	pfx := "ns-"
	rule := value.NewForwardRule(testURL(), testSpan(), &pfx, nil)
	view := NewForwardedModuleView(inner, rule)

	// Go PrefixedMapView ADDS prefix: inner key "a" → visible as "ns-a"
	id, err := view.VariableIdentity("ns-a")
	if err != nil {
		t.Fatal(err)
	}
	if id != inner {
		t.Error("VariableIdentity should delegate to inner's identity after prefix stripping")
	}
}

func TestForwardedModuleViewCouldHaveBeenConfigured(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)
	pfx := "ns-"
	rule := value.NewForwardRule(testURL(), testSpan(), &pfx, nil)
	view := NewForwardedModuleView(inner, rule)

	// BuiltInModule always returns false
	// Prefixed keys: inner "a" appears as "ns-a" in forwarded view
	configurable := view.CouldHaveBeenConfigured(map[string]struct{}{"ns-a": {}})
	if configurable {
		t.Error("forwarded BuiltInModule should not be configurable")
	}
}

func TestForwardedModuleViewCloneCss(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	view := NewForwardedModuleView(inner, value.NewForwardRule(testURL(), testSpan(), nil, nil))

	cloned, err := view.CloneCss()
	if err != nil {
		t.Fatal(err)
	}
	// Cloned is still a forwarded view of the cloned inner
	u, err := cloned.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u != "sass:test" {
		t.Errorf("cloned URL() = %q, want %q", u, "sass:test")
	}
}

// --- ShadowedModuleView ---

func TestShadowedModuleViewNew(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	vars.Put("b", value.NewUnitlessNumber(2))
	inner := NewBuiltInModule("test", nil, nil, vars)

	blocked := map[string]struct{}{"b": {}}
	shadowed := NewShadowedModuleView(inner, blocked, nil, nil)

	if shadowed == nil {
		t.Fatal("NewShadowedModuleView returned nil")
	}
	_, ok := shadowed.Variables().Get("a")
	if !ok {
		t.Error("unblocked variable 'a' not found")
	}
	_, ok = shadowed.Variables().Get("b")
	if ok {
		t.Error("blocked variable 'b' should not be visible")
	}
}

func TestShadowedModuleViewIfNecessaryNone(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	result := NewShadowedModuleViewIfNecessary(inner, map[string]struct{}{}, nil, nil)
	if result != nil {
		t.Error("no overlap → should return nil")
	}
}

func TestShadowedModuleViewIfNecessaryOverlap(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)
	blocked := map[string]struct{}{"a": {}}
	result := NewShadowedModuleViewIfNecessary(inner, blocked, nil, nil)
	if result == nil {
		t.Error("overlap exists → should return shadowed view")
	}
}

func TestShadowedModuleViewIsEmpty(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	// Block ALL variables → shadowed view has no members
	blocked := map[string]struct{}{"a": {}}
	shadowed := NewShadowedModuleView(inner, blocked, nil, nil)

	empty, err := shadowed.IsEmpty()
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Error("ShadowedModuleView with all vars blocked should be empty (no CSS children)")
	}
}

func TestShadowedModuleViewIsEmptyWithCSS(t *testing.T) {
	// BuiltInModule has empty CSS, so after blocking all vars, still empty
	// Here we just verify isEmpty works on a non-empty case
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	// Block only some variables → not empty
	blocked := map[string]struct{}{"b": {}}
	shadowed := NewShadowedModuleView(inner, blocked, nil, nil)
	empty, err := shadowed.IsEmpty()
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Error("Should not be empty when var remains")
	}
}

func TestShadowedModuleViewSetVariable(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	blocked := map[string]struct{}{"b": {}}
	shadowed := NewShadowedModuleView(inner, blocked, nil, nil)

	// 'a' is visible → delegates to inner (BuiltInModule throws "Cannot modify")
	err := shadowed.SetVariable("a", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error from inner (BuiltInModule)")
	}
	want := "Cannot modify built-in variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestShadowedModuleViewSetVariableBlocked(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	blocked := map[string]struct{}{"a": {}}
	shadowed := NewShadowedModuleView(inner, blocked, nil, nil)

	err := shadowed.SetVariable("a", value.Null, sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return testSpan(), nil
	}))
	if err == nil {
		t.Fatal("expected error for blocked variable")
	}
	want := "Undefined variable."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestShadowedModuleViewVariableIdentity(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	shadowed := NewShadowedModuleView(inner, nil, nil, nil)
	id, err := shadowed.VariableIdentity("a")
	if err != nil {
		t.Fatal(err)
	}
	if id != inner {
		t.Error("VariableIdentity should delegate to inner")
	}
}

func TestShadowedModuleViewCouldHaveBeenConfigured(t *testing.T) {
	vars := orderedmap.New[string, value.Value]()
	vars.Put("a", value.NewUnitlessNumber(1))
	inner := NewBuiltInModule("test", nil, nil, vars)

	shadowed := NewShadowedModuleView(inner, nil, nil, nil)
	// When no variables are blocked, delegates to inner
	configurable := shadowed.CouldHaveBeenConfigured(map[string]struct{}{"a": {}})
	if configurable {
		t.Error("BuiltInModule should not be configurable")
	}
}

func TestShadowedModuleViewCloneCss(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	shadowed := NewShadowedModuleView(inner, nil, nil, nil)

	cloned, err := shadowed.CloneCss()
	if err != nil {
		t.Fatal(err)
	}
	u, err := cloned.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u != "sass:test" {
		t.Errorf("cloned URL() = %q, want %q", u, "sass:test")
	}
}

// --- Module interface dispatch ---

func TestBuiltInModuleIsModule(t *testing.T) {
	m := NewBuiltInModule("math", nil, nil, nil)
	var iface Module = m
	if iface == nil {
		t.Fatal("BuiltInModule should satisfy Module interface")
	}
}

func TestForwardedModuleViewIsModule(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	view := NewForwardedModuleView(inner, value.NewForwardRule(testURL(), testSpan(), nil, nil))
	var iface Module = view
	if iface == nil {
		t.Fatal("ForwardedModuleView should satisfy Module interface")
	}
}

func TestShadowedModuleViewIsModule(t *testing.T) {
	inner := NewBuiltInModule("test", nil, nil, nil)
	shadowed := NewShadowedModuleView(inner, nil, nil, nil)
	var iface Module = shadowed
	if iface == nil {
		t.Fatal("ShadowedModuleView should satisfy Module interface")
	}
}
