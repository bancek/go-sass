package value

import (
	"strings"
	"testing"
)

type testModuleRef struct{ url string }

func (m testModuleRef) URL() (string, error) { return m.url, nil }
func (m testModuleRef) Equal(other ModuleRef) bool {
	o, ok := other.(testModuleRef)
	return ok && m.url == o.url
}
func (m testModuleRef) HashCode() int { return len(m.url) }

func testModule(name string) *SassModule {
	return NewSassModule(testModuleRef{url: "sass:" + name})
}

func TestSassModuleInspect(t *testing.T) {
	got, err := testModule("meta").String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `get-module("meta")` {
		t.Errorf("String() = %q, want %q", got, `get-module("meta")`)
	}
}

func TestSassModuleCSSError(t *testing.T) {
	_, err := testModule("meta").ToCssString(true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `get-module("meta") isn't a valid CSS value.`) {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSassModuleCompileContext(t *testing.T) {
	ctx := &struct{}{}
	other := &struct{}{}
	module := NewSassModuleWithCompileContext(testModule("meta").Module, ctx)
	if _, err := module.AssertCompileContext(ctx); err != nil {
		t.Fatalf("same context: %v", err)
	}
	_, err := module.AssertCompileContext(other)
	if err == nil || !strings.Contains(err.Error(), "does not belong to current compilation") {
		t.Errorf("wrong context error = %v", err)
	}
}

func TestSassModuleOperators(t *testing.T) {
	module := testModule("meta")
	if _, err := module.Plus(SassTrue); err == nil || !strings.Contains(err.Error(), `Undefined operation "get-module("meta") + true".`) {
		t.Errorf("Plus error = %v", err)
	}
	if _, err := module.UnaryMinus(); err == nil || !strings.Contains(err.Error(), `Undefined operation "-(get-module("meta"))".`) {
		t.Errorf("UnaryMinus error = %v", err)
	}
}

func TestSassModuleStructuralEquality(t *testing.T) {
	a := NewSassModule(testModuleRef{url: "sass:meta"})
	b := NewSassModule(testModuleRef{url: "sass:meta"})
	if !a.Equals(b) {
		t.Fatal("modules with equal module refs should compare equal")
	}
	if a.HashCode() != b.HashCode() {
		t.Fatal("equal modules must hash equally")
	}
	c := NewSassModule(testModuleRef{url: "sass:math"})
	if a.Equals(c) {
		t.Fatal("modules with different refs should not compare equal")
	}
}
