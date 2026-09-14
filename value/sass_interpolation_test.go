package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestInterpolationPlain(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("hello"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 5)

	interp := NewInterpolationPlain("hello", span)

	if !interp.IsPlain() {
		t.Error("IsPlain() = false, want true")
	}
	if plain := interp.AsPlain(); plain == nil || *plain != "hello" {
		t.Errorf("AsPlain() = %v, want 'hello'", plain)
	}
	if len(interp.Contents) != 1 {
		t.Errorf("len(Contents) = %d, want 1", len(interp.Contents))
	}
	if s, ok := interp.Contents[0].(string); !ok || s != "hello" {
		t.Errorf("Contents[0] = %v, want 'hello'", interp.Contents[0])
	}
	if interp.Spans()[0] != nil {
		t.Error("Spans()[0] should be nil for plain interp")
	}
}

func TestInterpolationWithExpression(t *testing.T) {
	source := "a.#{$expr}.b"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))
	// a.#{$expr}.b
	// 012345678901 (12 chars)
	// $expr at 4-9
	varSpan := sasscommon.NewFileSpan(fs, 4, 9)
	expr := NewVariableExpression("expr", varSpan, nil)
	// #{ spans 2-10
	interpExprSpan := sasscommon.NewFileSpan(fs, 2, 10)

	interp, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, &interpExprSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	if interp.IsPlain() {
		t.Error("IsPlain() = true, want false")
	}
	if interp.AsPlain() != nil {
		t.Error("AsPlain() should be nil for interpolated content")
	}
	if got := interp.InitialPlain(); got != "a." {
		t.Errorf("InitialPlain() = %q, want 'a.'", got)
	}

	result, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if result != "a.#{$expr}.b" {
		t.Errorf("String() = %q, want %q", result, "a.#{$expr}.b")
	}
}

func TestInterpolationAdjacentStringsError(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("ab"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	_, err := NewInterpolation(
		[]any{"a", "b"},
		[]*sasscommon.FileSpan{nil, nil},
		span,
	)
	if err == nil {
		t.Fatal("expected error for adjacent strings")
	}
}

func TestInterpolationMismatchedLengthsError(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	_, err := NewInterpolation(
		[]any{"a"},
		[]*sasscommon.FileSpan{nil, nil},
		span,
	)
	if err == nil {
		t.Fatal("expected error for mismatched lengths")
	}
}

func TestInterpolationNilSpanForExprError(t *testing.T) {
	source := "a.#{$expr}.b"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))
	varSpan := sasscommon.NewFileSpan(fs, 4, 9)
	expr := NewVariableExpression("expr", varSpan, nil)

	_, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, nil, nil},
		fileSpan,
	)
	if err == nil {
		t.Fatal("expected error for nil span on expression")
	}
}

func TestInterpolationSpanForStringElementError(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	_, err := NewInterpolation(
		[]any{"a"},
		[]*sasscommon.FileSpan{new(sasscommon.FileSpan)},
		span,
	)
	if err == nil {
		t.Fatal("expected error for span on string element")
	}
}

func TestInterpolationWrongTypeError(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	_, err := NewInterpolation(
		[]any{42},
		[]*sasscommon.FileSpan{nil},
		span,
	)
	if err == nil {
		t.Fatal("expected error for wrong content type")
	}
}

func TestInterpolationSpanForElementFirstString(t *testing.T) {
	source := "a.#{$expr}.b"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))
	varSpan := sasscommon.NewFileSpan(fs, 4, 9)
	expr := NewVariableExpression("expr", varSpan, nil)
	// #{ spans 2-10
	interpExprSpan := sasscommon.NewFileSpan(fs, 2, 10)

	interp, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, &interpExprSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	elementSpan, err := interp.SpanForElement(0)
	if err != nil {
		t.Fatal(err)
	}
	// "a." starts at offset 0, expression span starts at 2
	startLoc, _ := elementSpan.StartLocation()
	endLoc, _ := elementSpan.EndLocation()
	if startLoc.Offset != 0 || endLoc.Offset != 2 {
		t.Errorf("SpanForElement(0) = [%d, %d), want [0, 2)", startLoc.Offset, endLoc.Offset)
	}
}

func TestInterpolationSpanForElementExpression(t *testing.T) {
	source := "a.#{$expr}.b"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))
	varSpan := sasscommon.NewFileSpan(fs, 4, 9)
	expr := NewVariableExpression("expr", varSpan, nil)
	// #{ spans 2-10
	interpExprSpan := sasscommon.NewFileSpan(fs, 2, 10)

	interp, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, &interpExprSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	elementSpan, err := interp.SpanForElement(1)
	if err != nil {
		t.Fatal(err)
	}
	// SpanForElement(1) should return the expression span (the #{...} span)
	startLoc, _ := elementSpan.StartLocation()
	endLoc, _ := elementSpan.EndLocation()
	if startLoc.Offset != 2 || endLoc.Offset != 10 {
		t.Errorf("SpanForElement(1) = [%d, %d), want [2, 10)", startLoc.Offset, endLoc.Offset)
	}
}

func TestInterpolationAsPlainEmpty(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(""), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 0)

	interp, err := NewInterpolation([]any{}, []*sasscommon.FileSpan{}, span)
	if err != nil {
		t.Fatal(err)
	}

	plain := interp.AsPlain()
	if plain == nil || *plain != "" {
		t.Errorf("AsPlain() = %v, want '' for empty interp", plain)
	}
	if !interp.IsPlain() {
		t.Error("IsPlain() = false for empty interp, want true")
	}
}

func TestInterpolationSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("hello"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 5)
	interp := NewInterpolationPlain("hello", span)

	got, err := interp.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}
