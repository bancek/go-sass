// Copyright 2019 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package configuration

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func testSpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(nil, 0, 0)
}

func testFileSpan(text string) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(text), nil)
	return sasscommon.NewFileSpan(fs, 0, len(text))
}

func testAstNode() sasscommon.AstNode {
	fs := sasscommon.NewFileSource([]byte("node"), nil)
	sp := sasscommon.NewFileSpan(fs, 0, 4)
	return sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return sp, nil
	})
}

// --- ConfiguredValue ---

func TestConfiguredValueExplicit(t *testing.T) {
	val := value.SassTrue
	cfgSpan := testFileSpan("$a: 1")
	node := testAstNode()
	cv := NewConfiguredValueExplicit(val, &cfgSpan, node)

	if cv.Value != val {
		t.Error("Value mismatch")
	}
	if cv.ConfigurationSpan == nil {
		t.Error("ConfigurationSpan should be set")
	}
	if cv.AssignmentNode != node {
		t.Error("AssignmentNode mismatch")
	}
}

func TestConfiguredValueImplicit(t *testing.T) {
	val := value.SassTrue
	node := testAstNode()
	cv := NewConfiguredValueImplicit(val, node)

	if cv.Value != val {
		t.Error("Value mismatch")
	}
	if cv.ConfigurationSpan != nil {
		t.Error("ConfigurationSpan should be nil for implicit")
	}
	if cv.AssignmentNode != node {
		t.Error("AssignmentNode mismatch")
	}
}

func TestConfiguredValueString(t *testing.T) {
	val := value.SassTrue
	node := testAstNode()
	cv := NewConfiguredValueImplicit(val, node)

	s, err := cv.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "true"
	if s != want {
		t.Errorf("String() = %q, want %q", s, want)
	}
}

// --- Configuration construction ---

func TestConfigurationEmpty(t *testing.T) {
	cfg := EmptyConfiguration()
	if cfg == nil {
		t.Fatal("EmptyConfiguration returned nil")
	}
	if !cfg.IsEmpty() {
		t.Error("Empty config should be empty")
	}
	if cfg.IsExplicit() {
		t.Error("Empty config should be implicit")
	}
}

func TestConfigurationImplicit(t *testing.T) {
	val := value.SassTrue
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(val, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	if cfg.IsExplicit() {
		t.Error("Implicit config should not be explicit")
	}
	if cfg.IsEmpty() {
		t.Error("Should not be empty")
	}
	v, ok := cfg.Get("a")
	if !ok || v.Value != val {
		t.Error("Get('a') should return the configured value")
	}
	_, ok = cfg.Get("missing")
	if ok {
		t.Error("Get('missing') should return false")
	}
}

func TestConfigurationExplicit(t *testing.T) {
	val := value.SassTrue
	node := testAstNode()
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueExplicit(val, nil, node),
	}
	cfg := NewExplicitConfiguration(values, node)

	if !cfg.IsExplicit() {
		t.Error("Explicit config should be explicit")
	}
	if cfg.IsEmpty() {
		t.Error("Should not be empty")
	}
}

// --- IsEmpty ---

func TestConfigurationIsEmpty(t *testing.T) {
	cfg := EmptyConfiguration()
	if !cfg.IsEmpty() {
		t.Error("should be empty")
	}
}

func TestConfigurationIsEmptyNonEmpty(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)
	if cfg.IsEmpty() {
		t.Error("should not be empty")
	}
}

// --- Get ---

func TestConfigurationGet(t *testing.T) {
	valA := value.SassTrue
	valB := value.NewUnitlessNumber(42)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(valA, testAstNode()),
		"b": NewConfiguredValueImplicit(valB, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	v, ok := cfg.Get("a")
	if !ok {
		t.Fatal("Get('a') should return true")
	}
	if v.Value != valA {
		t.Error("Get('a') wrong value")
	}

	v, ok = cfg.Get("b")
	if !ok {
		t.Fatal("Get('b') should return true")
	}
	if v.Value != valB {
		t.Error("Get('b') wrong value")
	}
}

func TestConfigurationGetMissing(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	_, ok := cfg.Get("missing")
	if ok {
		t.Error("Get('missing') should return false")
	}
}

// --- Remove ---

func TestConfigurationRemove(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	removed := cfg.Remove("a")
	if removed == nil {
		t.Fatal("Remove('a') should return a value")
	}
	if !cfg.IsEmpty() {
		t.Error("Should be empty after removing the only key")
	}
}

func TestConfigurationRemoveMissing(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	removed := cfg.Remove("missing")
	if removed != nil {
		t.Error("Remove('missing') should return nil")
	}
}

func TestConfigurationRemoveFromEmpty(t *testing.T) {
	cfg := EmptyConfiguration()

	removed := cfg.Remove("a")
	if removed != nil {
		t.Error("Remove from empty should return nil")
	}
}

// --- Values ---

func TestConfigurationValues(t *testing.T) {
	valA := value.NewUnitlessNumber(1)
	valB := value.NewUnitlessNumber(2)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(valA, testAstNode()),
		"b": NewConfiguredValueImplicit(valB, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	result := cfg.Values()
	if len(result) != 2 {
		t.Fatalf("len(Values()) = %d, want 2", len(result))
	}
	if result["a"].Value != valA {
		t.Error("Values()['a'] wrong")
	}
	if result["b"].Value != valB {
		t.Error("Values()['b'] wrong")
	}
}

func TestConfigurationValuesEmpty(t *testing.T) {
	cfg := EmptyConfiguration()
	result := cfg.Values()
	if len(result) != 0 {
		t.Errorf("len(Values()) = %d, want 0", len(result))
	}
}

// --- SameOriginal ---

func TestConfigurationSameOriginalSameInner(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)
	derived := cfg.ThroughForward(&value.ForwardRule{})

	if !cfg.SameOriginal(derived) {
		t.Error("through-forward config should have same original")
	}
}

func TestConfigurationSameOriginalDifferent(t *testing.T) {
	cfg1 := NewConfigurationImplicit(map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	})
	cfg2 := NewConfigurationImplicit(map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	})

	if cfg1.SameOriginal(cfg2) {
		t.Error("independently created configs should have different originals")
	}
}

// --- ThroughForward ---

func TestConfigurationThroughForwardEmpty(t *testing.T) {
	cfg := EmptyConfiguration()
	result := cfg.ThroughForward(&value.ForwardRule{})

	if result != cfg {
		t.Error("empty config throughForward should return self")
	}
}

func TestConfigurationThroughForwardPrefix(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	values := map[string]*ConfiguredValue{
		"ns-a": NewConfiguredValueImplicit(val, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	prefix := "ns-"
	rule := &value.ForwardRule{Prefix: &prefix}
	derived := cfg.ThroughForward(rule)

	v, ok := derived.Get("a")
	if !ok {
		t.Fatal("Get('a') should find ns-a through prefix filter")
	}
	if v.Value != val {
		t.Error("wrong value through prefix filter")
	}
}

func TestConfigurationThroughForwardSafelist(t *testing.T) {
	valA := value.NewUnitlessNumber(1)
	valB := value.NewUnitlessNumber(2)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(valA, testAstNode()),
		"b": NewConfiguredValueImplicit(valB, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	shown := orderedset.New[string]()
	shown.Add("a")
	rule := &value.ForwardRule{ShownVariables: shown}
	derived := cfg.ThroughForward(rule)

	v, ok := derived.Get("a")
	if !ok {
		t.Fatal("Get('a') should be visible through safelist")
	}
	if v.Value != valA {
		t.Error("wrong value for 'a'")
	}

	_, ok = derived.Get("b")
	if ok {
		t.Error("Get('b') should be blocked by safelist")
	}
}

func TestConfigurationThroughForwardBlocklist(t *testing.T) {
	valA := value.NewUnitlessNumber(1)
	valB := value.NewUnitlessNumber(2)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(valA, testAstNode()),
		"b": NewConfiguredValueImplicit(valB, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	hidden := orderedset.New[string]()
	hidden.Add("b")
	rule := &value.ForwardRule{HiddenVariables: hidden}
	derived := cfg.ThroughForward(rule)

	v, ok := derived.Get("a")
	if !ok {
		t.Fatal("Get('a') should be visible")
	}
	if v.Value != valA {
		t.Error("wrong value for 'a'")
	}

	_, ok = derived.Get("b")
	if ok {
		t.Error("Get('b') should be blocked")
	}
}

func TestConfigurationThroughForwardPrefixAndSafelist(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	values := map[string]*ConfiguredValue{
		"ns-a": NewConfiguredValueImplicit(val, testAstNode()),
		"ns-b": NewConfiguredValueImplicit(value.NewUnitlessNumber(2), testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	prefix := "ns-"
	shown := orderedset.New[string]()
	shown.Add("a")
	rule := &value.ForwardRule{Prefix: &prefix, ShownVariables: shown}
	derived := cfg.ThroughForward(rule)

	v, ok := derived.Get("a")
	if !ok {
		t.Fatal("Get('a') should be visible through prefix + safelist")
	}
	if v.Value != val {
		t.Error("wrong value for 'a'")
	}

	_, ok = derived.Get("b")
	if ok {
		t.Error("Get('b') should be blocked by safelist")
	}
}

func TestConfigurationThroughForwardPreservesExplicit(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	node := testAstNode()
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueExplicit(val, nil, node),
	}
	cfg := NewExplicitConfiguration(values, node)

	prefix := "ns-"
	rule := &value.ForwardRule{Prefix: &prefix}
	derived := cfg.ThroughForward(rule)

	if !derived.IsExplicit() {
		t.Error("explicit config should stay explicit through forward")
	}
}

func TestConfigurationThroughForwardPreservesImplicit(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(val, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	prefix := "ns-"
	rule := &value.ForwardRule{Prefix: &prefix}
	derived := cfg.ThroughForward(rule)

	if derived.IsExplicit() {
		t.Error("implicit config should stay implicit through forward")
	}
}

// --- ToString ---

func TestConfigurationToString(t *testing.T) {
	val := value.NewUnitlessNumber(42)
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(val, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	s, err := cfg.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "($a: 42)"
	if s != want {
		t.Errorf("String() = %q, want %q", s, want)
	}
}

func TestConfigurationToStringMultiple(t *testing.T) {
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueImplicit(value.NewUnitlessNumber(1), testAstNode()),
		"b": NewConfiguredValueImplicit(value.SassTrue, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	s, err := cfg.String()
	if err != nil {
		t.Fatal(err)
	}
	// order not guaranteed across Go versions, check contains
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		t.Errorf("String() = %q, should be formatted as (values...)", s)
	}
}

func TestConfigurationToStringEmpty(t *testing.T) {
	cfg := EmptyConfiguration()
	s, err := cfg.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "()"
	if s != want {
		t.Errorf("String() = %q, want %q", s, want)
	}
}

// --- Remove through filter chain ---

func TestConfigurationRemoveThroughFilterChain(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	values := map[string]*ConfiguredValue{
		"ns-a": NewConfiguredValueImplicit(val, testAstNode()),
	}
	cfg := NewConfigurationImplicit(values)

	prefix := "ns-"
	rule := &value.ForwardRule{Prefix: &prefix}
	derived := cfg.ThroughForward(rule)

	removed := derived.Remove("a")
	if removed == nil {
		t.Fatal("Remove('a') through prefix should succeed")
	}
	if !derived.IsEmpty() {
		t.Error("derived should report empty")
	}
	if !cfg.IsEmpty() {
		t.Error("original inner map should also be empty (shared via Remove)")
	}
}

// --- ExplicitConfiguration error paths (spans) ---

func TestExplicitConfigurationThroughForward(t *testing.T) {
	val := value.NewUnitlessNumber(1)
	node := testAstNode()
	values := map[string]*ConfiguredValue{
		"a": NewConfiguredValueExplicit(val, nil, node),
	}
	cfg := NewExplicitConfiguration(values, node)

	derived := cfg.ThroughForward(&value.ForwardRule{})
	if !derived.IsExplicit() {
		t.Error("explicit should remain explicit")
	}
}
