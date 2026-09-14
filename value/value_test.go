package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestAssertBoolean(t *testing.T) {
	b, err := AssertBoolean(SassTrue, nil)
	if err != nil || b != SassTrue {
		t.Errorf("AssertBoolean(true) = (%v, %v)", b, err)
	}
	_, err = AssertBoolean(NewUnitlessNumber(1), nil)
	if err == nil {
		t.Error("AssertBoolean on number should error")
	}
}

func TestAssertNumber(t *testing.T) {
	n, err := AssertNumber(NewUnitlessNumber(1), nil)
	if err != nil || n.NumValue() != 1 {
		t.Errorf("AssertNumber(1) = (%v, %v)", n, err)
	}
	_, err = AssertNumber(SassTrue, nil)
	if err == nil {
		t.Error("AssertNumber on bool should error")
	}
}

func TestAssertString(t *testing.T) {
	s, err := AssertString(&SassString{Text: "hello", HasQuotes: true}, nil)
	if err != nil || s.Text != "hello" {
		t.Errorf("AssertString = (%v, %v)", s, err)
	}
	_, err = AssertString(SassTrue, nil)
	if err == nil {
		t.Error("AssertString on bool should error")
	}
}

func TestAssertColor(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	result, err := AssertColor(c, nil)
	if err != nil || result != c {
		t.Errorf("AssertColor = (%v, %v)", result, err)
	}
	_, err = AssertColor(SassTrue, nil)
	if err == nil {
		t.Error("AssertColor on bool should error")
	}
}

func TestAssertList(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, false)
	result, err := AssertList(l, nil)
	if err != nil || result != l {
		t.Errorf("AssertList(list) = (%v, %v)", result, err)
	}
	_, err = AssertList(NewUnitlessNumber(1), nil)
	if err == nil {
		t.Error("AssertList on number should error")
	}
}

func TestAssertList_ArgumentList(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1)}, nil, ListSeparatorSpace)
	result, err := AssertList(al, nil)
	if err != nil || result == nil {
		t.Fatalf("AssertList(argList) = (%v, %v)", result, err)
	}
}

func TestAssertMap(t *testing.T) {
	m := NewSassMap(map[Value]Value{NewUnitlessNumber(1): NewUnitlessNumber(2)})
	result, err := AssertMap(m, nil)
	if err != nil || result != m {
		t.Errorf("AssertMap(map) = (%v, %v)", result, err)
	}
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)
	result, err = AssertMap(emptyList, nil)
	if err != nil {
		t.Errorf("AssertMap(empty list) should return empty map: %v", err)
	}
	if result == nil || result.LengthAsList() != 0 {
		t.Error("AssertMap(empty list) should return empty map")
	}
	_, err = AssertMap(NewUnitlessNumber(1), nil)
	if err == nil {
		t.Error("AssertMap on number should error")
	}
}

func TestAssertFunction(t *testing.T) {
	f := NewSassFunction(&testCallable{1})
	result, err := AssertFunction(f, nil)
	if err != nil || result != f {
		t.Errorf("AssertFunction = (%v, %v)", result, err)
	}
	_, err = AssertFunction(NewUnitlessNumber(1), nil)
	if err == nil {
		t.Error("AssertFunction on number should error")
	}
}

func TestAssertMixin(t *testing.T) {
	m := NewSassMixin(&testMixinRef{1})
	result, err := AssertMixin(m, nil)
	if err != nil || result != m {
		t.Errorf("AssertMixin = (%v, %v)", result, err)
	}
	_, err = AssertMixin(NewUnitlessNumber(1), nil)
	if err == nil {
		t.Error("AssertMixin on number should error")
	}
}

func TestDefaultSingleEquals(t *testing.T) {
	result, err := DefaultSingleEquals(SassTrue, SassFalse)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.Contains(s.Text, "=") {
		t.Errorf("should contain '=', got %q", s.Text)
	}
}

func TestDefaultPlus(t *testing.T) {
	result, err := DefaultPlus(SassTrue, SassFalse)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if s.Text != "truefalse" {
		t.Errorf("DefaultPlus(true, false) = %q, want %q", s.Text, "truefalse")
	}

	s2 := &SassString{Text: "world", HasQuotes: true}
	result2, err := DefaultPlus(SassTrue, s2)
	if err != nil {
		t.Fatal(err)
	}
	s3, ok := result2.(*SassString)
	if !ok {
		t.Fatal("plus with string should return string")
	}
	if !strings.Contains(s3.Text, "world") {
		t.Errorf("should preserve right string, got %q", s3.Text)
	}
}

func TestDefaultMinus(t *testing.T) {
	result, err := DefaultMinus(SassTrue, SassFalse)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.Contains(s.Text, "-") {
		t.Errorf("should contain '-', got %q", s.Text)
	}
}

func TestDefaultTimes(t *testing.T) {
	_, err := DefaultTimes(SassTrue, SassFalse)
	if err == nil {
		t.Error("DefaultTimes should error")
	}
	if !strings.Contains(err.Error(), "Undefined operation") {
		t.Errorf("error should mention Undefined operation: %v", err)
	}
}

func TestDefaultDividedBy(t *testing.T) {
	result, err := DefaultDividedBy(SassTrue, SassFalse)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.Contains(s.Text, "/") {
		t.Errorf("should contain '/', got %q", s.Text)
	}
}

func TestDefaultModulo(t *testing.T) {
	_, err := DefaultModulo(SassTrue, SassFalse)
	if err == nil {
		t.Error("DefaultModulo should error")
	}
}

func TestDefaultComparison(t *testing.T) {
	_, err := DefaultGreaterThan(SassTrue, SassFalse)
	if err == nil {
		t.Error("DefaultGreaterThan should error")
	}
	if !strings.Contains(err.Error(), "Undefined operation") {
		t.Errorf("error should mention Undefined operation: %v", err)
	}

	_, err = DefaultLessThan(SassTrue, SassFalse)
	if err == nil {
		t.Error("DefaultLessThan should error")
	}
}

func TestDefaultUnaryPlus(t *testing.T) {
	result, err := DefaultUnaryPlus(SassTrue)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.HasPrefix(s.Text, "+") {
		t.Errorf("should start with '+', got %q", s.Text)
	}
}

func TestDefaultUnaryMinus(t *testing.T) {
	result, err := DefaultUnaryMinus(SassTrue)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.HasPrefix(s.Text, "-") {
		t.Errorf("should start with '-', got %q", s.Text)
	}
}

func TestDefaultUnaryDivide(t *testing.T) {
	result, err := DefaultUnaryDivide(SassTrue)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.HasPrefix(s.Text, "/") {
		t.Errorf("should start with '/', got %q", s.Text)
	}
}

func TestDefaultUnaryNot(t *testing.T) {
	result, err := DefaultUnaryNot(SassTrue)
	if err != nil {
		t.Fatal(err)
	}
	if result != SassFalse {
		t.Errorf("DefaultUnaryNot should return SassFalse, got %v", result)
	}
}

func TestDefaultRealNull(t *testing.T) {
	if DefaultRealNull(SassTrue) != SassTrue {
		t.Error("non-null should return self")
	}
}

func TestWithListContents(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, true)
	result := WithListContents(l, []Value{NewUnitlessNumber(2), NewUnitlessNumber(3)}, nil)
	if !result.HasBrackets() {
		t.Error("should preserve brackets")
	}
	if result.Separator() != ListSeparatorSpace {
		t.Error("should preserve separator")
	}
	list, _ := result.AsList()
	if len(list) != 2 {
		t.Error("should have new contents")
	}

	comma := ListSeparatorComma
	noBrackets := false
	result2 := WithListContents(l, []Value{NewUnitlessNumber(4)}, &ListOptions{Separator: &comma, Brackets: &noBrackets})
	if result2.Separator() != ListSeparatorComma {
		t.Error("should override separator")
	}
	if result2.HasBrackets() {
		t.Error("should override brackets")
	}
}

func TestSassIndexToListIndex(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(10), NewUnitlessNumber(20), NewUnitlessNumber(30)}, ListSeparatorSpace, false)
	i, err := SassIndexToListIndex(l, NewUnitlessNumber(1), "list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Errorf("index 1 should be 0-based 0, got %d", i)
	}
	i, err = SassIndexToListIndex(l, NewUnitlessNumber(-1), "list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if i != 2 {
		t.Errorf("index -1 should be 0-based 2 (last), got %d", i)
	}
	_, err = SassIndexToListIndex(l, NewUnitlessNumber(0), "list", nil)
	if err == nil {
		t.Error("index 0 should error")
	}
	_, err = SassIndexToListIndex(l, NewUnitlessNumber(10), "list", nil)
	if err == nil {
		t.Error("index 10 outside list length 3 should error")
	}
}

func TestAssertCommonListStyle(t *testing.T) {
	spaceList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	result, err := AssertCommonListStyle(spaceList, "val", false)
	if err != nil {
		t.Errorf("space list should be ok: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("should return contents, got %d", len(result))
	}

	commaList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorComma, false)
	_, err = AssertCommonListStyle(commaList, "val", false)
	if err == nil {
		t.Error("comma list should error")
	}

	bracketed, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, true)
	_, err = AssertCommonListStyle(bracketed, "val", false)
	if err == nil {
		t.Error("bracketed list should error")
	}
}

func TestValueEquals(t *testing.T) {
	if !ValueEquals(SassTrue, SassTrue) {
		t.Error("ValueEquals(true, true) should be true")
	}
	if ValueEquals(SassTrue, SassFalse) {
		t.Error("ValueEquals(true, false) should be false")
	}
}

func TestSprintAny(t *testing.T) {
	s, err := SprintAny(NewUnitlessNumber(5))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "5") {
		t.Errorf("SprintAny(5) = %q, should contain '5'", s)
	}

	s, err = SprintAny(&SassNull{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "null") {
		t.Errorf("SprintAny(null) = %q, should contain 'null'", s)
	}

	s, err = SprintAny(42)
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Error("SprintAny(int) should return non-empty")
	}
}

func TestErrorMessage(t *testing.T) {
	msg, err := ErrorMessage("color", NewUnitlessNumber(5), "a color")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "color") || !strings.Contains(msg, "is not a color") {
		t.Errorf("ErrorMessage = %q", msg)
	}
}

func TestDefaultPlus_NilError(t *testing.T) {
	_, err := DefaultPlus(&SassFunction{}, SassTrue)
	if err == nil {
		t.Error("DefaultPlus with function should error")
	}
	if _, ok := err.(*sasscommon.SassScriptException); ok {
		return
	}
	if strings.Contains(err.Error(), "isn't a valid CSS value") {
		return
	}
	t.Logf("got error type: %T: %v", err, err)
}

// --- SassScriptException field split (locked for the Rust port) ---
//
// Assert* functions put the bare message in Message and the argument name in
// ArgumentName; Error() composes "$name: message".

func assertScriptExceptionFields(t *testing.T, err error, wantMsg, wantArg string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	ex, ok := err.(*sasscommon.SassScriptException)
	if !ok {
		t.Fatalf("expected *SassScriptException, got %T", err)
	}
	if ex.Message != wantMsg {
		t.Errorf("Message = %q, want %q", ex.Message, wantMsg)
	}
	if ex.ArgumentName != wantArg {
		t.Errorf("ArgumentName = %q, want %q", ex.ArgumentName, wantArg)
	}
}

func TestAssertNumberFields(t *testing.T) {
	_, err := AssertNumber(SassTrue, ptrStr("x"))
	assertScriptExceptionFields(t, err, "true is not a number.", "x")
	if want := "$x: true is not a number."; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}

	_, err = AssertNumber(SassTrue, nil)
	assertScriptExceptionFields(t, err, "true is not a number.", "")
}

func TestAssertBooleanFields(t *testing.T) {
	_, err := AssertBoolean(NewUnitlessNumber(1), ptrStr("cond"))
	assertScriptExceptionFields(t, err, "1 is not a boolean.", "cond")
}

func TestAssertStringFields(t *testing.T) {
	_, err := AssertString(NewUnitlessNumber(1), ptrStr("string"))
	assertScriptExceptionFields(t, err, "1 is not a string.", "string")
}

func TestAssertColorFields(t *testing.T) {
	_, err := AssertColor(NewUnitlessNumber(42), ptrStr("color"))
	assertScriptExceptionFields(t, err, "42 is not a color.", "color")
}

func TestAssertListFields(t *testing.T) {
	_, err := AssertList(SassTrue, ptrStr("list"))
	assertScriptExceptionFields(t, err, "true is not a list.", "list")
}

func TestAssertMapFields(t *testing.T) {
	_, err := AssertMap(NewUnitlessNumber(1), ptrStr("map"))
	assertScriptExceptionFields(t, err, "1 is not a map.", "map")
}

func TestAssertFunctionFields(t *testing.T) {
	_, err := AssertFunction(NewUnitlessNumber(1), ptrStr("function"))
	assertScriptExceptionFields(t, err, "1 is not a function reference.", "function")
}

func TestAssertMixinFields(t *testing.T) {
	_, err := AssertMixin(NewUnitlessNumber(1), ptrStr("mixin"))
	assertScriptExceptionFields(t, err, "1 is not a mixin reference.", "mixin")
}
