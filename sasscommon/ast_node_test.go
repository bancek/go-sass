package sasscommon

import (
	"testing"
)

func TestFakeAstNodeSpan(t *testing.T) {
	fs := NewFileSource([]byte("hello"), nil)
	span := NewSimpleFileSpan(fs, 0, 5)
	node := NewFakeAstNode(func() (FileSpan, error) {
		return span, nil
	})
	got, err := node.Span()
	if err != nil {
		t.Fatal(err)
	}
	text, err := got.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("Span().SpanText() = %q, want %q", text, "hello")
	}
}

func TestFakeAstNodeIsAstNode(t *testing.T) {
	// Verify that FakeAstNode implements AstNode interface without panicking.
	var node AstNode = NewFakeAstNode(func() (FileSpan, error) {
		return nil, nil
	})
	_ = node
}

func TestFakeAstNodeSpanError(t *testing.T) {
	node := NewFakeAstNode(func() (FileSpan, error) {
		return nil, &ArgumentError{Message: "test error"}
	})
	_, err := node.Span()
	if err == nil {
		t.Fatal("expected error from Span()")
	}
}
