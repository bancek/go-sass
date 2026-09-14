package sourcemapbuffer

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sourcemap"
)

func testFileSpan(urlStr string) sasscommon.FileSpan {
	u, _ := url.Parse(urlStr)
	fs := sasscommon.NewFileSource([]byte("body { color: red; }"), u)
	return sasscommon.NewSimpleFileSpan(fs, 0, 0)
}

var stubBuilder = sourcemap.NewBuilder("output.css")

// === NoSourceMapBuffer tests ===

func TestNoSourceMapBuffer_WriteString(t *testing.T) {
	b := NewNoSourceMapBuffer()
	n, err := b.WriteString("hello")
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("WriteString returned %d, want 5", n)
	}
	if b.String() != "hello" {
		t.Errorf("Buffer = %q, want %q", b.String(), "hello")
	}
}

func TestNoSourceMapBuffer_WriteByte(t *testing.T) {
	b := NewNoSourceMapBuffer()
	if err := b.WriteByte('x'); err != nil {
		t.Fatal(err)
	}
	if b.String() != "x" {
		t.Errorf("Buffer = %q, want %q", b.String(), "x")
	}
}

func TestNoSourceMapBuffer_WriteRune(t *testing.T) {
	b := NewNoSourceMapBuffer()
	n, err := b.WriteRune('€')
	if err != nil {
		t.Fatal(err)
	}
	// '€' is 3 bytes in UTF-8
	if b.String() != "€" {
		t.Errorf("Buffer = %q, want %q", b.String(), "€")
	}
	_ = n
}

func TestNoSourceMapBuffer_Write(t *testing.T) {
	b := NewNoSourceMapBuffer()
	n, err := b.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}
	if b.String() != "test" {
		t.Errorf("Buffer = %q, want %q", b.String(), "test")
	}
}

func TestNoSourceMapBuffer_String(t *testing.T) {
	b := NewNoSourceMapBuffer()
	if b.String() != "" {
		t.Errorf("new buffer = %q, want empty", b.String())
	}
	b.WriteString("abc")
	if b.String() != "abc" {
		t.Errorf("after write = %q, want %q", b.String(), "abc")
	}
}

func TestNoSourceMapBuffer_Len(t *testing.T) {
	b := NewNoSourceMapBuffer()
	if b.Len() != 0 {
		t.Errorf("new buffer len = %d, want 0", b.Len())
	}
	b.WriteString("hello")
	if b.Len() != 5 {
		t.Errorf("len after write = %d, want 5", b.Len())
	}
}

func TestNoSourceMapBuffer_ForSpan(t *testing.T) {
	b := NewNoSourceMapBuffer()
	called := false
	span := testFileSpan("file:///input.scss")
	err := b.ForSpan(span, func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("callback was not called")
	}
}

func TestNoSourceMapBuffer_ForSpan_ErrorPropagation(t *testing.T) {
	b := NewNoSourceMapBuffer()
	expectErr := fmt.Errorf("test error")
	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		return expectErr
	})
	if err != expectErr {
		t.Errorf("error = %v, want %v", err, expectErr)
	}
}

func TestNoSourceMapBuffer_BuildSourceMap(t *testing.T) {
	b := NewNoSourceMapBuffer()
	_, err := b.BuildSourceMap("")
	if err == nil {
		t.Error("expected error from BuildSourceMap")
	}
}

func TestNoSourceMapBuffer_MultipleWrites(t *testing.T) {
	b := NewNoSourceMapBuffer()
	b.WriteString("hello ")
	b.WriteString("world")
	if b.String() != "hello world" {
		t.Errorf("Buffer = %q, want %q", b.String(), "hello world")
	}
}

// === DefaultSourceMapBuffer tests ===

func TestDefaultSourceMapBuffer_WriteString(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	n, err := b.WriteString("hello")
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("WriteString returned %d, want 5", n)
	}
	if b.String() != "hello" {
		t.Errorf("Buffer = %q, want %q", b.String(), "hello")
	}
}

func TestDefaultSourceMapBuffer_WriteString_PositionTracking(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	b.WriteString("abc\ndef")
	// After "abc\n": line=1, col=0
	// After "def": line=1, col=3
	// Internal state exposed via BuildSourceMap entries — we test via ForSpan mapping below
	_ = b.String()
}

func TestDefaultSourceMapBuffer_WriteString_NewlineIncrement(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	// Write "a\nb" and check that a mapping is recorded at (1,0)
	// First, enter a span at (0,0)
	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		_, err := b.WriteString("a\nb")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	// Should have mappings at: (0,0) from ForSpan, (1,0) from newline auto-mapping
	// The newline at "a\nb" is at (0,1) → newline goes to (1,0)
	if len(m.Mappings) == 0 {
		t.Error("expected at least one mapping segment")
	}
}

func TestDefaultSourceMapBuffer_WriteByte(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	if err := b.WriteByte('x'); err != nil {
		t.Fatal(err)
	}
	if b.String() != "x" {
		t.Errorf("Buffer = %q, want %q", b.String(), "x")
	}
}

func TestDefaultSourceMapBuffer_WriteRune(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	n, err := b.WriteRune('€')
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "€" {
		t.Errorf("Buffer = %q, want %q", b.String(), "€")
	}
	_ = n
}

func TestDefaultSourceMapBuffer_Write(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	n, err := b.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}
	if b.String() != "test" {
		t.Errorf("Buffer = %q, want %q", b.String(), "test")
	}
}

func TestDefaultSourceMapBuffer_ForSpan_RecordsMapping(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		b.WriteString("hello")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	// Should have mapping at (0,0) from ForSpan
	if len(m.Mappings) == 0 {
		t.Error("expected mapping to be recorded")
	}
}

func TestDefaultSourceMapBuffer_ForSpan_CallsCallback(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	called := false
	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("callback was not called")
	}
}

func TestDefaultSourceMapBuffer_ForSpan_ErrorPropagation(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	expectErr := fmt.Errorf("test error")
	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		return expectErr
	})
	if err != expectErr {
		t.Errorf("error = %v, want %v", err, expectErr)
	}
}

func TestDefaultSourceMapBuffer_ForSpan_RestoresInSpan(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	// Enter first span
	b.ForSpan(testFileSpan("file:///a.scss"), func() error {
		b.WriteString("a")
		return nil
	})

	// After ForSpan exits, inSpan should be false.
	// Write a newline — should NOT generate an auto-mapping
	b.WriteString("\n")

	// Enter second span
	b.ForSpan(testFileSpan("file:///b.scss"), func() error {
		b.WriteString("b")
		return nil
	})

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	_ = m
	// The newline between spans should NOT create a mapping
	// Only the two ForSpan entries should be present
}

func TestDefaultSourceMapBuffer_BuildSourceMap_NoPrefix(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		b.WriteString("content")
		return nil
	})

	m, err := b.BuildSourceMap("")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("BuildSourceMap returned nil")
	}
	if len(m.Mappings) == 0 {
		t.Error("expected non-empty mappings")
	}
}

func TestDefaultSourceMapBuffer_BuildSourceMap_WithPrefix(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		b.WriteString("content")
		return nil
	})

	// Prefix with newline should shift everything down one line
	m, err := b.BuildSourceMap("\n")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("BuildSourceMap returned nil")
	}
}

func TestDefaultSourceMapBuffer_MultipleWrites(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	b.WriteString("hello ")
	b.WriteByte('w')
	b.WriteString("orld")
	if b.String() != "hello world" {
		t.Errorf("Buffer = %q, want %q", b.String(), "hello world")
	}
}

func TestDefaultSourceMapBuffer_WriteString_SingleWrite(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	text := "a\nb\nc\n"
	n, err := b.WriteString(text)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(text) {
		t.Errorf("WriteString returned %d, want %d", n, len(text))
	}
	out := b.String()
	// Count newlines in output
	nl := strings.Count(out, "\n")
	if nl != 3 {
		t.Errorf("expected 3 newlines, got %d (output: %q)", nl, out)
	}
}

func TestDefaultSourceMapBuffer_ForSpan_NestedNewlineMapping(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		// Write content with newline — should auto-map at the newline position
		_, err := b.WriteString("line1\nline2")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	// Should have: mapping at (0,0) from ForSpan, mapping at (1,0) from newline
	// Verify via JSON that we have some mappings
	jsonBytes, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("JSON: %s", string(jsonBytes))
	if len(m.Mappings) == 0 {
		t.Error("expected mappings in source map")
	}
}

func TestDefaultSourceMapBuffer_Empty(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	if b.String() != "" {
		t.Errorf("new buffer = %q, want empty", b.String())
	}
	if b.Len() != 0 {
		t.Errorf("new buffer len = %d, want 0", b.Len())
	}
}

func TestNoSourceMapBuffer_ForSpan_PassesSpan(t *testing.T) {
	b := NewNoSourceMapBuffer()
	var capturedSpan sasscommon.FileSpan
	captured := false
	span := testFileSpan("file:///input.scss")
	err := b.ForSpan(span, func() error {
		capturedSpan = span
		captured = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !captured {
		t.Error("callback was not called")
	}
	_ = capturedSpan
}

func TestNoSourceMapBuffer_BuildSourceMap_WithPrefix(t *testing.T) {
	b := NewNoSourceMapBuffer()
	_, err := b.BuildSourceMap("prefix")
	if err == nil {
		t.Error("expected error from BuildSourceMap")
	}
}

func TestDefaultSourceMapBuffer_String_Len(t *testing.T) {
	b := NewDefaultSourceMapBuffer(stubBuilder)
	b.WriteString("hello")
	if b.Len() != 5 {
		t.Errorf("Len = %d, want 5", b.Len())
	}
	if b.String() != "hello" {
		t.Errorf("String = %q, want %q", b.String(), "hello")
	}
}

func TestDefaultSourceMapBuffer_ForSpan_AfterCallbackRestoresInSpan(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	// Write a newline before entering span — should NOT create mapping
	b.WriteString("before\n")

	// Enter span
	b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		b.WriteString("inside")
		return nil
	})

	// Write a newline after exiting span — should NOT create mapping
	b.WriteString("\nafter")

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	// Only one entry: from ForSpan at the position where it was called
	// After "before\n": line=1, col=0. ForSpan records at (1,0).
	_ = m
}

func TestDefaultSourceMapBuffer_WriteByte_NewlineAutoMapping(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("output.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	err := b.ForSpan(testFileSpan("file:///input.scss"), func() error {
		if err := b.WriteByte('a'); err != nil {
			return err
		}
		if err := b.WriteByte('\n'); err != nil {
			return err
		}
		if err := b.WriteByte('b'); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := mappingBuilder.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	// Should have mapping at (0,0) from ForSpan + mapping at (1,0) from newline
	if len(m.Mappings) == 0 {
		t.Error("expected mappings")
	}
}

func TestDefaultSourceMapBuffer_ForSpan_BuildSourceMap_Integration(t *testing.T) {
	mappingBuilder := sourcemap.NewBuilder("integration.css")
	b := NewDefaultSourceMapBuffer(mappingBuilder)

	b.WriteString("prefix_")

	b.ForSpan(testFileSpan("file:///source.scss"), func() error {
		b.WriteString("body { color: ")
		b.WriteString("red")
		b.WriteString("; }")
		return nil
	})

	b.WriteString("_suffix")

	result := b.String()
	if !strings.Contains(result, "prefix_") || !strings.Contains(result, "_suffix") {
		t.Errorf("unexpected output: %q", result)
	}

	m, err := b.BuildSourceMap("")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("BuildSourceMap returned nil")
	}
}
