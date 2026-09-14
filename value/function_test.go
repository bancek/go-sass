package value

import (
	"fmt"
	"strings"
	"testing"
)

type testCallable struct{ n int }

func (c *testCallable) Name() string { return fmt.Sprintf("test-callable-%d", c.n) }

type namedCallable struct{ name string }

func (c *namedCallable) Name() string { return c.name }

func TestFunctionEquality(t *testing.T) {
	c1 := &testCallable{1}
	c2 := &testCallable{2}
	f1 := NewSassFunction(c1)
	f2 := NewSassFunction(c1)
	if !f1.Equals(f2) {
		t.Error("functions with same ref should be equal")
	}
	f3 := NewSassFunction(c2)
	if f1.Equals(f3) {
		t.Error("functions with different ref should not be equal")
	}
	if f1.Equals(Null) {
		t.Error("function should not equal null")
	}
}

func TestFunctionHashCode(t *testing.T) {
	c := &testCallable{42}
	f := NewSassFunction(c)
	h := f.HashCode()
	if h == 0 {
		t.Error("non-nil ref hash should be non-zero")
	}
	if f.HashCode() != h {
		t.Error("hash should be deterministic")
	}
	fNil := &SassFunction{}
	if fNil.HashCode() != 0 {
		t.Errorf("nil ref hash should be 0, got %d", fNil.HashCode())
	}
}

func TestFunctionAssertFunction(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	if f.AssertFunction() != f {
		t.Error("AssertFunction should return self")
	}
}

func TestFunctionAssertCompileContext(t *testing.T) {
	f := &SassFunction{FunctionRef: &testCallable{1}}
	_, err := f.AssertCompileContext(nil)
	if err != nil {
		t.Errorf("nil compileContext should be ok: %v", err)
	}
	ctx1 := struct{}{}
	_, err = f.AssertCompileContext(&ctx1)
	if err != nil {
		t.Errorf("matching compileContext should be ok: %v", err)
	}
}

func TestFunctionAssertCompileContextMismatch(t *testing.T) {
	ctx1 := &struct{}{}
	ctx2 := &struct{}{}
	f := NewSassFunctionWithCompileContext(&namedCallable{name: "f"}, ctx1)

	same, err := f.AssertCompileContext(ctx1)
	if err != nil {
		t.Fatalf("matching compileContext should be ok: %v", err)
	}
	if same != f {
		t.Error("AssertCompileContext should return the function itself")
	}

	_, err = f.AssertCompileContext(ctx2)
	if err == nil {
		t.Fatal("mismatched compileContext should error")
	}
	want := `get-function("f") does not belong to current compilation.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestFunctionIsTruthy(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	if !f.IsTruthy() {
		t.Error("function should be truthy")
	}
}

func TestFunctionListProps(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	if f.Separator() != ListSeparatorUndecided {
		t.Error("separator should be Undecided")
	}
	if f.HasBrackets() {
		t.Error("should not have brackets")
	}
	list, err := f.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != f {
		t.Error("AsList should return [self]")
	}
	if f.LengthAsList() != 1 {
		t.Error("LengthAsList should be 1")
	}
	if f.IsBlank() {
		t.Error("should not be blank")
	}
}

func TestFunctionToCssString(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	_, err := f.ToCssString(false)
	if err == nil {
		t.Error("function ToCssString should error")
	}
	if !strings.Contains(err.Error(), "isn't a valid CSS value") {
		t.Errorf("error should mention invalid CSS value: %v", err)
	}
}

func TestFunctionString(t *testing.T) {
	f := &SassFunction{FunctionRef: &namedCallable{name: "my-func"}}
	got, err := f.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `get-function("my-func")` {
		t.Errorf("String() = %q, want %q", got, `get-function("my-func")`)
	}
}

func TestFunctionOperators(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	_, err := f.Plus(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Plus should error for function (CSS serialization fails)")
	}
	_, err = f.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error")
	}
}
