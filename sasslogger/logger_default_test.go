package sasslogger

import (
	"bytes"
	"testing"
)

func TestNewDefaultLogger(t *testing.T) {
	span := testFileSpan("a { b: c; }\n", 0, 1)

	// Unicode=true
	var buf bytes.Buffer
	l := NewDefaultLogger(true)
	l.Writer = &buf
	l.color = false
	l.Warn("test", span, nil)
	expected := "WARNING on line 1, column 1: \n" +
		"test\n" +
		"  ╷\n" +
		"1 │ a { b: c; }\n" +
		"  │ ^\n" +
		"  ╵\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("Unicode true: got %q, want %q", out, expected)
	}

	// Unicode=false
	buf.Reset()
	l = NewDefaultLogger(false)
	l.Writer = &buf
	l.color = false
	l.Warn("test", span, nil)
	expected = "WARNING on line 1, column 1: \n" +
		"test\n" +
		"  ,\n" +
		"1 | a { b: c; }\n" +
		"  | ^\n" +
		"  '\n" +
		"\n"
	if out := buf.String(); out != expected {
		t.Errorf("Unicode false: got %q, want %q", out, expected)
	}
}
