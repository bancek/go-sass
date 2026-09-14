package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func makeTestSpan(text string) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(text), nil)
	return sasscommon.NewSimpleFileSpan(fs, 0, len(text))
}

func TestBufferEmpty(t *testing.T) {
	var buf InterpolationBuffer
	if !buf.IsEmpty() {
		t.Error("new buffer should be empty")
	}
	if got := buf.TrailingString(); got != "" {
		t.Errorf("TrailingString() = %q, want ''", got)
	}
}

func TestBufferWrite(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("hello")
	if buf.IsEmpty() {
		t.Error("buffer should not be empty after write")
	}
	if got := buf.TrailingString(); got != "hello" {
		t.Errorf("TrailingString() = %q, want 'hello'", got)
	}
}

func TestBufferWriteCharCode(t *testing.T) {
	var buf InterpolationBuffer
	buf.WriteCharCode('A')
	if got := buf.TrailingString(); got != "A" {
		t.Errorf("TrailingString() = %q, want 'A'", got)
	}
}

func TestBufferWriteln(t *testing.T) {
	var buf InterpolationBuffer
	buf.Writeln("hello")
	if got := buf.TrailingString(); got != "hello\n" {
		t.Errorf("TrailingString() = %q, want 'hello\\n'", got)
	}
}

func TestBufferWriteAll(t *testing.T) {
	var buf InterpolationBuffer
	separator := ","
	buf.WriteAll([]string{"a", "b", "c"}, &separator)
	if got := buf.TrailingString(); got != "a,b,c" {
		t.Errorf("TrailingString() = %q, want 'a,b,c'", got)
	}
}

func TestBufferWriteAllDefaultSeparator(t *testing.T) {
	var buf InterpolationBuffer
	buf.WriteAll([]string{"a", "b", "c"}, nil)
	if got := buf.TrailingString(); got != "abc" {
		t.Errorf("TrailingString() = %q, want 'abc'", got)
	}
}

func TestBufferClear(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("hello")
	buf.Clear()
	if !buf.IsEmpty() {
		t.Error("buffer should be empty after clear")
	}
	if got := buf.TrailingString(); got != "" {
		t.Errorf("TrailingString() = %q after clear, want ''", got)
	}
}

func TestBufferAddExpression(t *testing.T) {
	var buf InterpolationBuffer
	span := makeTestSpan("#{$expr}")
	varSpan := makeTestSpan("$expr")
	expr := NewVariableExpression("expr", varSpan, nil)

	buf.Add(expr, span)

	// After flush, contents should have one element, spans should have one
	if buf.IsEmpty() {
		t.Error("buffer should not be empty after add")
	}
}

func TestBufferAddInterpolation(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("a.")

	span := makeTestSpan("#{$expr}")
	varSpan := makeTestSpan("$expr")
	expr := NewVariableExpression("expr", varSpan, nil)
	interp, err := NewInterpolation(
		[]any{expr},
		[]*sasscommon.FileSpan{&span},
		span,
	)
	if err != nil {
		t.Fatal(err)
	}

	buf.AddInterpolation(interp)

	// After AddInterpolation, the buffer should not be empty
	if buf.IsEmpty() {
		t.Error("buffer should not be empty after addInterpolation")
	}
}

func TestBufferAddInterpolationEmpty(t *testing.T) {
	var buf InterpolationBuffer
	fs := sasscommon.NewFileSource([]byte(""), nil)
	emptySpan := sasscommon.NewSimpleFileSpan(fs, 0, 0)
	interp, err := NewInterpolation([]any{}, []*sasscommon.FileSpan{}, emptySpan)
	if err != nil {
		t.Fatal(err)
	}

	buf.AddInterpolation(interp)
	// Should be a no-op
	if !buf.IsEmpty() {
		t.Error("buffer should be empty after adding empty interpolation")
	}
}

func TestBufferInterpolationPlain(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("hello")
	span := makeTestSpan("hello")
	interp, err := buf.Interpolation(span)
	if err != nil {
		t.Fatal(err)
	}

	if !interp.IsPlain() {
		t.Error("interpolation should be plain for text-only buffer")
	}
	if plain := interp.AsPlain(); plain == nil || *plain != "hello" {
		t.Errorf("AsPlain() = %v, want 'hello'", plain)
	}
}

func TestBufferInterpolationMixed(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("a.")
	exprSpan := makeTestSpan("#{$expr}")
	varSpan := makeTestSpan("$expr")
	expr := NewVariableExpression("expr", varSpan, nil)
	buf.Add(expr, exprSpan)
	buf.Write(".b")
	fileSpan := makeTestSpan("a.#{$expr}.b")

	interp, err := buf.Interpolation(fileSpan)
	if err != nil {
		t.Fatal(err)
	}

	if interp.IsPlain() {
		t.Error("interpolation should not be plain for mixed buffer")
	}
	if len(interp.Contents) != 3 {
		t.Errorf("len(Contents) = %d, want 3", len(interp.Contents))
	}
}

func TestBufferString(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("a.")
	exprSpan := makeTestSpan("#{$expr}")
	varSpan := makeTestSpan("$expr")
	expr := NewVariableExpression("expr", varSpan, nil)
	buf.Add(expr, exprSpan)
	buf.Write(".b")

	result, err := buf.String()
	if err != nil {
		t.Fatal(err)
	}
	if result != "a.#{$expr}.b" {
		t.Errorf("String() = %q, want 'a.#{$expr}.b'", result)
	}
}

func TestBufferFlushTextBehavior(t *testing.T) {
	var buf InterpolationBuffer
	buf.Write("before")

	exprSpan := makeTestSpan("#{$expr}")
	varSpan := makeTestSpan("$expr")
	expr := NewVariableExpression("expr", varSpan, nil)
	buf.Add(expr, exprSpan)

	// After Add, the text "before" should have been flushed
	// New text written after should be in _text buffer
	buf.Write("after")

	interpSpan := makeTestSpan("before#{$expr}after")
	interp, err := buf.Interpolation(interpSpan)
	if err != nil {
		t.Fatal(err)
	}

	if len(interp.Contents) != 3 {
		t.Errorf("len(Contents) = %d, want 3", len(interp.Contents))
	}
	// First should be "before"
	if s, ok := interp.Contents[0].(string); !ok || s != "before" {
		t.Errorf("Contents[0] = %v, want 'before'", interp.Contents[0])
	}
	// Last should be "after"
	if s, ok := interp.Contents[len(interp.Contents)-1].(string); !ok || s != "after" {
		t.Errorf("last Contents = %v, want 'after'", interp.Contents[len(interp.Contents)-1])
	}
}
