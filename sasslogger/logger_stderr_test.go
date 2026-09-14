package sasslogger

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

func testFileSpan(text string, start, end int) *sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(text), nil)
	var s sasscommon.FileSpan = sasscommon.NewSimpleFileSpan(fs, start, end)
	return &s
}

func TestStderrWarnNoSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	l.Warn("test message", nil, nil)
	expected := "WARNING: test message\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrWarnWithSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, false, &buf)
	l.Warn("test message", testFileSpan("body { color: red; }\n", 0, 10), nil)
	expected := "WARNING on line 1, column 1: \n" +
		"test message\n" +
		"  ,\n" +
		"1 | body { color: red; }\n" +
		"  | ^^^^^^^^^^\n" +
		"  '\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrWarnWithStack(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, false, &buf)
	l.Warn("test message", testFileSpan("body { color: red; }\n", 0, 10), testTrace("stack trace"))
	expected := "WARNING: test message\n" +
		"\n" +
		"  ,\n" +
		"1 | body { color: red; }\n" +
		"  | ^^^^^^^^^^\n" +
		"  '\n" +
		"    - 0:0  stack trace\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrWarnDeprecation(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	if err := l.WarnDeprecation("deprecated feature", nil, deprecation.CallString, nil); err != nil {
		t.Fatalf("WarnDeprecation should not error: %v", err)
	}
	expected := "DEPRECATION WARNING [call-string]: deprecated feature\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrWarnDeprecationNoSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	if err := l.WarnDeprecation("msg", nil, deprecation.SlashDiv, nil); err != nil {
		t.Fatalf("WarnDeprecation should not error: %v", err)
	}
	expected := "DEPRECATION WARNING [slash-div]: msg\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrWarnUserAuthored(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	if err := l.WarnDeprecation("msg", nil, deprecation.UserAuthored, nil); err != nil {
		t.Fatalf("WarnDeprecation should not error: %v", err)
	}
	expected := "DEPRECATION WARNING: msg\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrDebugNoSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	l.Debug("debug msg", nil)
	expected := "-:1 DEBUG: debug msg\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrDebugWithSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	u, _ := url.Parse("file:///test.scss")
	fs := sasscommon.NewFileSource([]byte("body { }\n"), u)
	var s sasscommon.FileSpan = sasscommon.NewSimpleFileSpan(fs, 0, 8)
	l.Debug("debug msg", &s)
	expected := "/test.scss:1 DEBUG: debug msg\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrColorOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(true, true, &buf)
	l.Warn("test", nil, nil)
	expected := "\x1b[33m\x1b[1mWarning\x1b[0m: test\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrColorDeprecationOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(true, true, &buf)
	_ = l.WarnDeprecation("test", nil, deprecation.CallString, nil)
	expected := "\x1b[33m\x1b[1mDeprecation Warning\x1b[0m [\x1b[34mcall-string\x1b[0m]: test\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrUnicodeGlyphs(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, true, &buf)
	l.Warn("test", testFileSpan("body { color: red; }\n", 0, 10), nil)
	expected := "WARNING on line 1, column 1: \n" +
		"test\n" +
		"  ╷\n" +
		"1 │ body { color: red; }\n" +
		"  │ ^^^^^^^^^^\n" +
		"  ╵\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestStderrAsciiGlyphs(t *testing.T) {
	var buf bytes.Buffer
	l := NewStderrLoggerWithWriter(false, false, &buf)
	l.Warn("test", testFileSpan("body { color: red; }\n", 0, 10), nil)
	expected := "WARNING on line 1, column 1: \n" +
		"test\n" +
		"  ,\n" +
		"1 | body { color: red; }\n" +
		"  | ^^^^^^^^^^\n" +
		"  '\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}
