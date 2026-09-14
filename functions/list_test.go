// Copyright 2019 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package functions

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/value"
)

func eval(c *BuiltInCallable, args ...value.Value) (value.Value, error) {
	result, err := c.CallbackFor(len(args), nil)
	if err != nil {
		return nil, err
	}
	// Pad args with Null to match the overload's parameter count
	paramCount := 0
	if result.Params != nil {
		paramCount = len(result.Params.Parameters)
	}
	for len(args) < paramCount {
		args = append(args, value.Null)
	}
	ec := &evalcontext.EvaluationContext{}
	return result.Fn(ec, args)
}

func num(v float64) value.Value {
	return value.NewUnitlessNumber(v)
}

func str(s string) value.Value {
	return &value.SassString{Text: s, HasQuotes: false}
}

func quoted(s string) value.Value {
	return &value.SassString{Text: s, HasQuotes: true}
}

func list(sep value.ListSeparator, bracketed bool, items ...value.Value) value.Value {
	l, _ := value.NewSassList(items, sep, bracketed)
	return l
}

func commaList(items ...value.Value) value.Value {
	return list(value.ListSeparatorComma, false, items...)
}

func spaceList(items ...value.Value) value.Value {
	return list(value.ListSeparatorSpace, false, items...)
}

func bracketedList(items ...value.Value) value.Value {
	return list(value.ListSeparatorSpace, true, items...)
}

func assertNum(t *testing.T, v value.Value, want float64) {
	t.Helper()
	n, ok := v.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", v)
	}
	if n.NumValue() != want {
		t.Errorf("NumValue() = %v, want %v", n.NumValue(), want)
	}
}

func assertString(t *testing.T, v value.Value, want string) {
	t.Helper()
	s, ok := v.(*value.SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", v)
	}
	if s.Text != want {
		t.Errorf("Text = %q, want %q", s.Text, want)
	}
}

func assertBool(t *testing.T, v value.Value, want bool) {
	t.Helper()
	if want {
		if v != value.SassTrue {
			t.Errorf("expected SassTrue, got %T", v)
		}
	} else {
		if v != value.SassFalse {
			t.Errorf("expected SassFalse, got %T", v)
		}
	}
}

func assertNull(t *testing.T, v value.Value) {
	t.Helper()
	if v != value.Null {
		t.Errorf("expected Null, got %T", v)
	}
}

func assertListLen(t *testing.T, v value.Value, want int) {
	t.Helper()
	if v.LengthAsList() != want {
		t.Errorf("LengthAsList() = %v, want %v", v.LengthAsList(), want)
	}
}

func assertListSep(t *testing.T, v value.Value, want value.ListSeparator) {
	t.Helper()
	if v.Separator() != want {
		t.Errorf("Separator() = %v, want %v", v.Separator(), want)
	}
}

func assertListBrackets(t *testing.T, v value.Value, want bool) {
	t.Helper()
	if v.HasBrackets() != want {
		t.Errorf("HasBrackets() = %v, want %v", v.HasBrackets(), want)
	}
}

// --- length ---

func TestListLength(t *testing.T) {
	v, err := eval(lengthFunction(), commaList(num(1), num(2), num(3)))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, v, 3)
}

func TestListLengthEmpty(t *testing.T) {
	v, err := eval(lengthFunction(), commaList())
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, v, 0)
}

// --- nth ---

func TestListNth(t *testing.T) {
	v, err := eval(nthFunction(), commaList(num(10), num(20), num(30)), num(2))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, v, 20)
}

func TestListNthNegative(t *testing.T) {
	v, err := eval(nthFunction(), commaList(num(10), num(20), num(30)), num(-1))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, v, 30)
}

func TestListNthOutOfBounds(t *testing.T) {
	_, err := eval(nthFunction(), commaList(num(10)), num(5))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListNthZeroIndex(t *testing.T) {
	_, err := eval(nthFunction(), commaList(num(10)), num(0))
	if err == nil {
		t.Fatal("expected error for index 0")
	}
}

// --- set-nth ---

func TestListSetNth(t *testing.T) {
	v, err := eval(setNthFunction(), commaList(num(1), num(2), num(3)), num(2), str("replaced"))
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 3)
	l, _ := v.AsList()
	s := l[1].(*value.SassString)
	if s.Text != "replaced" {
		t.Errorf("element = %q, want %q", s.Text, "replaced")
	}
}

// --- join ---

func TestListJoinDefault(t *testing.T) {
	l1 := commaList(num(1), num(2))
	l2 := commaList(num(3), num(4))
	v, err := eval(joinFunction(), l1, l2)
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 4)
	assertListSep(t, v, value.ListSeparatorComma)
}

func TestListJoinComma(t *testing.T) {
	l1 := spaceList(num(1))
	l2 := spaceList(num(2))
	v, err := eval(joinFunction(), l1, l2, str("comma"))
	if err != nil {
		t.Fatal(err)
	}
	assertListSep(t, v, value.ListSeparatorComma)
}

func TestListJoinSpace(t *testing.T) {
	l1 := commaList(num(1))
	l2 := commaList(num(2))
	v, err := eval(joinFunction(), l1, l2, str("space"))
	if err != nil {
		t.Fatal(err)
	}
	assertListSep(t, v, value.ListSeparatorSpace)
}

func TestListJoinBracketed(t *testing.T) {
	l1 := bracketedList(num(1))
	l2 := commaList(num(2))
	v, err := eval(joinFunction(), l1, l2, str("comma"), str("auto"))
	if err != nil {
		t.Fatal(err)
	}
	assertListBrackets(t, v, true)
}

func TestListJoinInvalidSeparator(t *testing.T) {
	_, err := eval(joinFunction(), commaList(num(1)), commaList(num(2)), str("invalid"))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- append ---

func TestListAppend(t *testing.T) {
	v, err := eval(appendFunction(), commaList(num(1), num(2)), num(3))
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 3)
}

func TestListAppendSpace(t *testing.T) {
	l := spaceList(num(1), num(2))
	v, err := eval(appendFunction(), l, num(3))
	if err != nil {
		t.Fatal(err)
	}
	assertListSep(t, v, value.ListSeparatorSpace)
}

// --- zip ---

func TestListZip(t *testing.T) {
	l1 := commaList(num(1), num(2))
	l2 := commaList(str("a"), str("b"))
	v, err := eval(zipFunction(), commaList(l1, l2))
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 2)
	assertListSep(t, v, value.ListSeparatorComma)
}

func TestListZipEmpty(t *testing.T) {
	v, err := eval(zipFunction(), commaList())
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 0)
}

func TestListZipUneven(t *testing.T) {
	l1 := commaList(num(1), num(2), num(3))
	l2 := commaList(str("a"))
	v, err := eval(zipFunction(), commaList(l1, l2))
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 1) // only one row: short list truncates
}

// --- index ---

func TestListIndex(t *testing.T) {
	v, err := eval(indexFunction(), commaList(num(10), num(20), num(30)), num(20))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, v, 2)
}

func TestListIndexNotFound(t *testing.T) {
	v, err := eval(indexFunction(), commaList(num(10)), num(99))
	if err != nil {
		t.Fatal(err)
	}
	assertNull(t, v)
}

// --- is-bracketed ---

func TestListIsBracketedTrue(t *testing.T) {
	v, err := eval(isBracketedFunction(), bracketedList(num(1)))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, v, true)
}

func TestListIsBracketedFalse(t *testing.T) {
	v, err := eval(isBracketedFunction(), spaceList(num(1)))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, v, false)
}

// --- separator ---

func TestListSeparatorComma(t *testing.T) {
	v, err := eval(separatorFunction(), commaList(num(1)))
	if err != nil {
		t.Fatal(err)
	}
	assertString(t, v, "comma")
}

func TestListSeparatorSpace(t *testing.T) {
	v, err := eval(separatorFunction(), spaceList(num(1)))
	if err != nil {
		t.Fatal(err)
	}
	assertString(t, v, "space")
}

func TestListSeparatorSlash(t *testing.T) {
	l := list(value.ListSeparatorSlash, false, num(1), num(2))
	v, err := eval(separatorFunction(), l)
	if err != nil {
		t.Fatal(err)
	}
	assertString(t, v, "slash")
}

// --- slash ---

func TestListSlash(t *testing.T) {
	v, err := eval(slashFunction(), commaList(num(1), num(2), num(3)))
	if err != nil {
		t.Fatal(err)
	}
	assertListLen(t, v, 3)
	assertListSep(t, v, value.ListSeparatorSlash)
}

func TestListSlashTooFew(t *testing.T) {
	_, err := eval(slashFunction(), commaList(num(1)))
	if err == nil {
		t.Fatal("expected error for fewer than 2 elements")
	}
	want := "At least two elements are required."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want contains %q", err.Error(), want)
	}
}
