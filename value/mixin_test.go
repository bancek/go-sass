package value

import (
	"fmt"
	"strings"
	"testing"
)

type testMixinRef struct{ n int }

func (r *testMixinRef) Name() string { return fmt.Sprintf("test-mixin-ref-%d", r.n) }

type namedMixinRef struct{ name string }

func (r *namedMixinRef) Name() string { return r.name }

func TestMixinEquality(t *testing.T) {
	r1 := &testMixinRef{1}
	r2 := &testMixinRef{2}
	m1 := NewSassMixin(r1)
	m2 := NewSassMixin(r1)
	if !m1.Equals(m2) {
		t.Error("mixins with same ref should be equal")
	}
	m3 := NewSassMixin(r2)
	if m1.Equals(m3) {
		t.Error("mixins with different ref should not be equal")
	}
	if m1.Equals(Null) {
		t.Error("mixin should not equal null")
	}
	f := NewSassFunction(&testCallable{1})
	if m1.Equals(f) {
		t.Error("mixin should not equal function")
	}
}

func TestMixinHashCode(t *testing.T) {
	r := &testMixinRef{42}
	m := NewSassMixin(r)
	h := m.HashCode()
	if h == 0 {
		t.Error("non-nil ref hash should be non-zero")
	}
	if m.HashCode() != h {
		t.Error("hash should be deterministic")
	}
	mNil := &SassMixin{}
	if mNil.HashCode() != 0 {
		t.Errorf("nil ref hash should be 0, got %d", mNil.HashCode())
	}
}

func TestMixinAssertMixin(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	if m.AssertMixin() != m {
		t.Error("AssertMixin should return self")
	}
}

func TestMixinAssertCompileContext(t *testing.T) {
	m := &SassMixin{MixinRef: &testMixinRef{1}}
	_, err := m.AssertCompileContext(nil)
	if err != nil {
		t.Errorf("nil compileContext should be ok: %v", err)
	}
	ctx1 := struct{}{}
	_, err = m.AssertCompileContext(&ctx1)
	if err != nil {
		t.Errorf("matching compileContext should be ok: %v", err)
	}
}

func TestMixinAssertCompileContextMismatch(t *testing.T) {
	ctx1 := &struct{}{}
	ctx2 := &struct{}{}
	m := NewSassMixinWithCompileContext(&namedMixinRef{name: "m"}, ctx1)

	same, err := m.AssertCompileContext(ctx1)
	if err != nil {
		t.Fatalf("matching compileContext should be ok: %v", err)
	}
	if same != m {
		t.Error("AssertCompileContext should return the mixin itself")
	}

	_, err = m.AssertCompileContext(ctx2)
	if err == nil {
		t.Fatal("mismatched compileContext should error")
	}
	want := `get-mixin("m") does not belong to current compilation.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestMixinIsTruthy(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	if !m.IsTruthy() {
		t.Error("mixin should be truthy")
	}
}

func TestMixinListProps(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	if m.Separator() != ListSeparatorUndecided {
		t.Error("separator should be Undecided")
	}
	if m.HasBrackets() {
		t.Error("should not have brackets")
	}
	list, err := m.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != m {
		t.Error("AsList should return [self]")
	}
	if m.LengthAsList() != 1 {
		t.Error("LengthAsList should be 1")
	}
	if m.IsBlank() {
		t.Error("should not be blank")
	}
}

func TestMixinToCssString(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	_, err := m.ToCssString(false)
	if err == nil {
		t.Error("mixin ToCssString should error")
	}
	if !strings.Contains(err.Error(), "isn't a valid CSS value") {
		t.Errorf("error should mention invalid CSS value: %v", err)
	}
}

func TestMixinString(t *testing.T) {
	m := &SassMixin{MixinRef: &namedMixinRef{name: "my-mixin"}}
	got, err := m.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `get-mixin("my-mixin")` {
		t.Errorf("String() = %q, want %q", got, `get-mixin("my-mixin")`)
	}
}

func TestMixinOperators(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	_, err := m.Plus(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Plus should error for mixin (CSS serialization fails)")
	}
	_, err = m.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error")
	}
}
