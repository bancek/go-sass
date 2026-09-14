package sasscommon

import (
	"testing"
)

func TestCssValueSpan(t *testing.T) {
	fs := NewFileSource([]byte("source"), nil)
	span := NewSimpleFileSpan(fs, 0, 6)
	v := NewCssValue(42, span)
	got, err := v.Span()
	if err != nil {
		t.Fatal(err)
	}
	text, err := got.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "source" {
		t.Errorf("SpanText() = %q, want %q", text, "source")
	}
}

func TestCssValueIsAstNode(t *testing.T) {
	fs := NewFileSource([]byte("x"), nil)
	span := NewSimpleFileSpan(fs, 0, 1)
	v := NewCssValue("hello", span)
	var node AstNode = v
	_ = node
}

func TestCssValueString(t *testing.T) {
	fs := NewFileSource([]byte("x"), nil)
	span := NewSimpleFileSpan(fs, 0, 1)
	v := NewCssValue("hello", span)
	if v.String() != "hello" {
		t.Errorf("String() = %q, want %q", v.String(), "hello")
	}
}

func TestCssValueEqual(t *testing.T) {
	fs1 := NewFileSource([]byte("abc"), nil)
	fs2 := NewFileSource([]byte("xyz"), nil)
	a := NewCssValue(42, NewSimpleFileSpan(fs1, 0, 3))
	b := NewCssValue(42, NewSimpleFileSpan(fs2, 0, 3))
	c := NewCssValue(99, NewSimpleFileSpan(fs1, 0, 3))

	if !a.Equal(b) {
		t.Error("Equal should be true for same values (ignoring span)")
	}
	if a.Equal(c) {
		t.Error("Equal should be false for different values")
	}
}
