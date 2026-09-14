package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestStringExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(`"hello"`), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 7)
	expr := NewStringExpressionPlain("hello", span, true)

	if !expr.HasQuotes {
		t.Error("HasQuotes should be true")
	}
}

func TestStringExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(`"hello"`), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 7)
	expr := NewStringExpressionPlain("hello", span, true)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `"hello"` {
		t.Errorf("String() = %q, want %q", got, `"hello"`)
	}
}

func TestStringExpressionUnquoted(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("hello"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 5)
	expr := NewStringExpressionPlain("hello", span, false)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("String() = %q, want 'hello'", got)
	}
}

func TestStringExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(`"hello"`), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 7)
	expr := NewStringExpressionPlain("hello", span, true)

	si := expr.SourceInterpolation()
	if si == nil {
		t.Fatal("SourceInterpolation() should not be nil for StringExpression")
	}
	if !si.IsPlain() {
		t.Error("SourceInterpolation should be plain")
	}
}
