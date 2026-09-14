package sasscommon

import (
	"net/url"
	"strings"
	"testing"
)

func TestExceptionToCssStringSimple(t *testing.T) {
	got := exceptionToCssString("test error")
	want := strings.Join([]string{
		`/* test error */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: test error;`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "test error", got, want)
	}
}

func TestExceptionToCssStringMultiline(t *testing.T) {
	got := exceptionToCssString("line1\nline2")
	want := strings.Join([]string{
		`/* line1`,
		` * line2 */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		"  content: line1\nline2;",
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "line1\\nline2", got, want)
	}
}

func TestExceptionToCssStringCloseComment(t *testing.T) {
	got := exceptionToCssString("hello */ world")
	want := strings.Join([]string{
		`/* hello *` + "\u2215" + ` world */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: hello */ world;`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "hello */ world", got, want)
	}
}

func TestExceptionToCssStringCrLf(t *testing.T) {
	got := exceptionToCssString("line1\r\nline2")
	want := strings.Join([]string{
		`/* line1`,
		` * line2 */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		"  content: line1\r\nline2;",
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "line1\\r\\nline2", got, want)
	}
}

func TestExceptionToCssStringNonAscii(t *testing.T) {
	got := exceptionToCssString("caf\u00e9")
	want := strings.Join([]string{
		`/* caf` + "\u00e9" + ` */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: caf\e9 ;`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "caf\\u00e9", got, want)
	}
}

func TestExceptionToCssStringRealError(t *testing.T) {
	msg := strings.Join([]string{
		`Error: expected selector.`,
		`  ` + "\u2557",
		`1 ` + "\u2502" + ` .foo {`,
		`  ` + "\u2502" + `      ^`,
		`  ` + "\u2555",
		`  - 1:6  root stylesheet`,
	}, "\n")
	got := exceptionToCssString(msg)
	// Box-drawing chars are > 0x7F so they are hex-escaped in the content:
	// U+2557 → \2557, U+2502 → \2502, U+2555 → \2555
	// In the comment they are preserved as-is.
	want := strings.Join([]string{
		`/* Error: expected selector.`,
		` *   ` + "\u2557",
		` * 1 ` + "\u2502" + ` .foo {`,
		` *   ` + "\u2502" + `      ^`,
		` *   ` + "\u2555",
		` *   - 1:6  root stylesheet */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		"  content: Error: expected selector.\n  \\2557 \n1 \\2502  .foo {\n  \\2502       ^\n  \\2555 \n  - 1:6  root stylesheet;",
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(real error) = %q, want %q", got, want)
	}
}

func TestExceptionToCssStringEmpty(t *testing.T) {
	got := exceptionToCssString("")
	want := strings.Join([]string{
		`/*  */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: ;`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", "", got, want)
	}
}

func TestExceptionToCssStringQuoteChars(t *testing.T) {
	got := exceptionToCssString(`"quoted"`)
	want := strings.Join([]string{
		`/* "quoted" */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: "quoted";`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", `"quoted"`, got, want)
	}
}

func TestExceptionToCssStringBackslash(t *testing.T) {
	got := exceptionToCssString(`path\to\file`)
	want := strings.Join([]string{
		`/* path\to\file */`,
		``,
		`body::before {`,
		`  font-family: "Source Code Pro", "SF Mono", Monaco, Inconsolata, "Fira Mono",`,
		`      "Droid Sans Mono", monospace, monospace;`,
		`  white-space: pre;`,
		`  display: block;`,
		`  padding: 1em;`,
		`  margin-bottom: 1em;`,
		`  border-bottom: 2px solid black;`,
		`  content: path\to\file;`,
		`}`,
	}, "\n")
	if got != want {
		t.Errorf("exceptionToCssString(%q) = %q, want %q", `path\to\file`, got, want)
	}
}

func TestFrameForSpan(t *testing.T) {
	fs := NewFileSource([]byte("a { color: red; }"), nil)
	span := NewSimpleFileSpan(fs, 3, 4)
	frame, err := FrameForSpan(span, "root stylesheet")
	if err != nil {
		t.Fatal(err)
	}
	if frame.Line != 1 {
		t.Errorf("Line = %d, want 1", frame.Line)
	}
	if frame.Column != 4 {
		t.Errorf("Column = %d, want 4", frame.Column)
	}
	if frame.Member != "root stylesheet" {
		t.Errorf("Member = %q, want %q", frame.Member, "root stylesheet")
	}
	// URI may be nil since no URL set on FileSource — PrettyUri returns "-" for nil
}

func TestFrameLocation(t *testing.T) {
	// Frame with URI — PrettyUri strips file:// prefix
	u, _ := url.Parse("file:///a.scss")
	fs := NewFileSource([]byte("body { }\n"), u)
	span := NewSimpleFileSpan(fs, 0, 8)
	frame, err := FrameForSpan(span, "root stylesheet")
	if err != nil {
		t.Fatal(err)
	}
	loc := frame.location()
	want := "/a.scss 1:1"
	if loc != want {
		t.Errorf("location() = %q, want %q", loc, want)
	}
}

func TestFrameLocationNilUri(t *testing.T) {
	frame := Frame{URI: nil, Line: 1, Column: 7, Member: "root stylesheet"}
	loc := frame.location()
	want := "- 1:7"
	if loc != want {
		t.Errorf("location() = %q, want %q", loc, want)
	}
}

func TestTraceStringSingle(t *testing.T) {
	frame := Frame{URI: nil, Line: 1, Column: 7, Member: "root stylesheet"}
	trace := NewTrace([]Frame{frame})
	got := trace.String()
	want := "- 1:7  root stylesheet"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestTraceStringMultiple(t *testing.T) {
	frames := []Frame{
		{URI: nil, Line: 1, Column: 7, Member: "root stylesheet"},
		{URI: nil, Line: 123, Column: 45, Member: "some other member"},
	}
	trace := NewTrace(frames)
	got := trace.String()
	// longest location = "- 123:45" len=8
	// "- 1:7" len=5, padding = 8-5+2 = 5
	want := strings.Join([]string{
		"- 1:7     root stylesheet",
		"- 123:45  some other member",
	}, "\n")
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestTraceStringEmpty(t *testing.T) {
	trace := NewTrace([]Frame{})
	got := trace.String()
	if got != "" {
		t.Errorf("NewTrace with no frames: String() = %q, want empty", got)
	}
}

func TestTraceStringNil(t *testing.T) {
	var trace *Trace
	got := trace.String()
	if got != "" {
		t.Errorf("nil Trace.String() = %q, want empty", got)
	}
}

func TestSassExceptionErrorWithOptions(t *testing.T) {
	u, _ := url.Parse("file:///a.scss")
	fs := NewFileSource([]byte("a { color: red; }"), u)
	span := NewSimpleFileSpan(fs, 3, 4)
	exc := &SassException{
		Message: "expected selector.",
		Span:    span,
	}
	got := exc.ErrorWithOptions(HighlightOptions{})
	want := strings.Join([]string{
		`Error: expected selector.`,
		`  ` + "\u2577",
		`1 ` + "\u2502" + ` a { color: red; }`,
		`  ` + "\u2502" + `    ^`,
		`  ` + "\u2575",
		`  /a.scss 1:4  root stylesheet`,
	}, "\n")
	if got != want {
		t.Errorf("ErrorWithOptions = %q, want %q", got, want)
	}
}

func TestSassRuntimeExceptionErrorWithOptions(t *testing.T) {
	u, _ := url.Parse("file:///a.scss")
	fs := NewFileSource([]byte("a { color: red; }"), u)
	span := NewSimpleFileSpan(fs, 3, 4)
	trace := NewTrace([]Frame{
		{URI: nil, Line: 1, Column: 7, Member: "root stylesheet"},
		{URI: nil, Line: 123, Column: 45, Member: "some other member"},
	})
	exc := &SassRuntimeException{
		Message: "something broke",
		Span:    span,
		Trace:   trace,
	}
	got := exc.ErrorWithOptions(HighlightOptions{})
	want := strings.Join([]string{
		`Error: something broke`,
		`  ` + "\u2577",
		`1 ` + "\u2502" + ` a { color: red; }`,
		`  ` + "\u2502" + `    ^`,
		`  ` + "\u2575",
		`  - 1:7     root stylesheet`,
		`  - 123:45  some other member`,
	}, "\n")
	if got != want {
		t.Errorf("ErrorWithOptions = %q, want %q", got, want)
	}
}

func TestSassFormatExceptionErrorWithOptions(t *testing.T) {
	u, _ := url.Parse("file:///a.scss")
	fs := NewFileSource([]byte("a { color: red; }"), u)
	span := NewSimpleFileSpan(fs, 3, 4)
	exc := &SassFormatException{
		Message: "expected \"}\".",
		Span:    span,
	}
	got := exc.ErrorWithOptions(HighlightOptions{})
	want := strings.Join([]string{
		`Error: expected "}".`,
		`  ` + "\u2577",
		`1 ` + "\u2502" + ` a { color: red; }`,
		`  ` + "\u2502" + `    ^`,
		`  ` + "\u2575",
		`  /a.scss 1:4  root stylesheet`,
	}, "\n")
	if got != want {
		t.Errorf("ErrorWithOptions = %q, want %q", got, want)
	}
}

func TestSassScriptExceptionError(t *testing.T) {
	exc := &SassScriptException{Message: "true is not a number.", ArgumentName: "x"}
	got := exc.Error()
	want := "$x: true is not a number."
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestSassScriptExceptionErrorNoArg(t *testing.T) {
	exc := &SassScriptException{Message: "bad"}
	got := exc.Error()
	want := "bad"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestSassScriptExceptionWithSpanKeepsArgumentName(t *testing.T) {
	exc := &SassScriptException{Message: "true is not a number.", ArgumentName: "x"}
	got := exc.WithSpan(nil)
	want := "$x: true is not a number."
	if got.Message != want {
		t.Errorf("WithSpan().Message = %q, want %q", got.Message, want)
	}
	if got.Span != nil {
		t.Errorf("WithSpan().Span = %v, want nil", got.Span)
	}
}

func TestSassScriptExceptionWithSpanNoArg(t *testing.T) {
	exc := &SassScriptException{Message: "bad"}
	got := exc.WithSpan(nil)
	if got.Message != "bad" {
		t.Errorf("WithSpan().Message = %q, want %q", got.Message, "bad")
	}
}

func TestSassExceptionToCssStringIncludesHighlight(t *testing.T) {
	u, _ := url.Parse("file:///a.scss")
	fs := NewFileSource([]byte("a { color: red; }"), u)
	span := NewSimpleFileSpan(fs, 3, 4)
	exc := &SassException{Message: "expected selector.", Span: span}
	got := exc.ToCssString()
	// Verify the comment contains the full formatted error with highlight
	if !strings.Contains(got, "/* Error: expected selector.") {
		t.Error("expected comment to contain 'Error: expected selector.'")
	}
	if !strings.Contains(got, "╷") {
		t.Error("expected comment to contain highlight box-drawing chars")
	}
	if !strings.Contains(got, "/a.scss") {
		t.Error("expected comment to contain source file path")
	}
}
