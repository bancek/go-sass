// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package functions

import (
	"testing"

	"github.com/bancek/go-sass/value"
)

// --- GlobalFunctions ---

func TestGlobalFunctionsContainsAllPackages(t *testing.T) {
	names := map[string]bool{}
	for _, fn := range GlobalFunctions() {
		names[fn.Name()] = true
	}
	// One representative per source list, plus if().
	for _, name := range []string{
		"red",            // color
		"nth",            // list
		"map-get",        // map
		"percentage",     // math
		"selector-parse", // selector
		"str-length",     // string
		"inspect",        // meta
		"if",             // hand-authored if()
	} {
		if !names[name] {
			t.Errorf("GlobalFunctions() should contain %q", name)
		}
	}
}

func TestGlobalFunctionsIfIsLast(t *testing.T) {
	fns := GlobalFunctions()
	if fns[len(fns)-1].Name() != "if" {
		t.Errorf("last global function = %q, want %q", fns[len(fns)-1].Name(), "if")
	}
}

func TestIfFunction(t *testing.T) {
	fn := mathBIC(t, GlobalFunctions()[len(GlobalFunctions())-1])
	ifTrue := str("yes")
	ifFalse := str("no")

	got := selEvalOK(t, fn, value.SassTrue, ifTrue, ifFalse)
	if got != ifTrue {
		t.Errorf("if(true, ...) = %v, want $if-true", got)
	}

	got = selEvalOK(t, fn, value.SassFalse, ifTrue, ifFalse)
	if got != ifFalse {
		t.Errorf("if(false, ...) = %v, want $if-false", got)
	}

	got = selEvalOK(t, fn, value.Null, ifTrue, ifFalse)
	if got != ifFalse {
		t.Errorf("if(null, ...) = %v, want $if-false", got)
	}

	// Any non-false, non-null value is truthy.
	got = selEvalOK(t, fn, num(0), ifTrue, ifFalse)
	if got != ifTrue {
		t.Errorf("if(0, ...) = %v, want $if-true", got)
	}
}

// --- CoreModules ---

func TestCoreModules(t *testing.T) {
	modules := CoreModules()
	want := []string{
		"sass:color", "sass:list", "sass:map", "sass:math", "sass:selector", "sass:string",
	}
	if len(modules) != len(want) {
		t.Fatalf("len = %d, want %d", len(modules), len(want))
	}
	for i, m := range modules {
		url, err := m.URL()
		if err != nil {
			t.Fatal(err)
		}
		if url != want[i] {
			t.Errorf("modules[%d].URL() = %q, want %q", i, url, want[i])
		}
	}
}

// --- DisallowedFunctionNames ---

func TestDisallowedFunctionNamesRemovals(t *testing.T) {
	set := DisallowedFunctionNames()
	removed := []string{
		"abs", "alpha", "color", "grayscale", "hsl", "hsla", "hwb", "invert",
		"lab", "lch", "max", "min", "oklab", "oklch", "opacity", "rgb", "rgba",
		"round", "saturate",
	}
	for _, name := range removed {
		if set[name] {
			t.Errorf("DisallowedFunctionNames() should not contain %q", name)
		}
	}
}

func TestDisallowedFunctionNamesContains(t *testing.T) {
	set := DisallowedFunctionNames()
	for _, name := range []string{
		"red", "green", "blue", "mix", "lighten", "darken",
		"length", "nth", "join",
		"map-get", "map-merge",
		"percentage", "comparable", "unit", "unitless", "ceil", "floor", "random",
		"is-superselector", "selector-parse", "simple-selectors",
		"unquote", "quote", "str-length", "str-insert", "str-index", "str-slice",
		"to-upper-case", "to-lower-case", "unique-id",
		"feature-exists", "inspect", "type-of", "keywords",
		"if",
	} {
		if !set[name] {
			t.Errorf("DisallowedFunctionNames() should contain %q", name)
		}
	}
}

func TestDisallowedFunctionNamesCount(t *testing.T) {
	// Every removed name must have been present in GlobalFunctions(), and
	// global function names must be unique.
	all := map[string]bool{}
	for _, fn := range GlobalFunctions() {
		if all[fn.Name()] {
			t.Errorf("duplicate global function name %q", fn.Name())
		}
		all[fn.Name()] = true
	}
	set := DisallowedFunctionNames()
	if len(set) != len(all)-19 {
		t.Errorf("len(DisallowedFunctionNames()) = %d, want %d", len(set), len(all)-19)
	}
}
