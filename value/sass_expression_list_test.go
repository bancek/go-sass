package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestListExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a, b"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	contents := []Expression{
		NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 0, 1), true),
		NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 3, 4), true),
	}
	expr := NewListExpression(contents, ListSeparatorComma, span, false)

	if len(expr.Contents) != 2 {
		t.Errorf("len(Contents) = %d, want 2", len(expr.Contents))
	}
	if expr.Separator != ListSeparatorComma {
		t.Errorf("Separator = %v, want Comma", expr.Separator)
	}
}

func TestListExpressionStringComma(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a, b, c"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 7)
	contents := []Expression{
		NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 0, 1), true),
		NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 3, 4), true),
		NewStringExpressionPlain("c", sasscommon.NewSimpleFileSpan(fs, 6, 7), true),
	}
	expr := NewListExpression(contents, ListSeparatorComma, span, false)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("String() returned empty")
	}
}

func TestListExpressionStringSpace(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a b"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 3)
	contents := []Expression{
		NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 0, 1), true),
		NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 2, 3), true),
	}
	expr := NewListExpression(contents, ListSeparatorSpace, span, false)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("String() returned empty")
	}
}

func TestListExpressionStringBracketed(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("[a, b]"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	contents := []Expression{
		NewStringExpressionPlain("a", sasscommon.NewSimpleFileSpan(fs, 1, 2), true),
		NewStringExpressionPlain("b", sasscommon.NewSimpleFileSpan(fs, 4, 5), true),
	}
	expr := NewListExpression(contents, ListSeparatorComma, span, true)

	if !expr.HasBrackets {
		t.Error("HasBrackets should be true")
	}
}

func TestListExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a, b"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)
	expr := NewListExpression([]Expression{}, ListSeparatorSpace, span, false)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
