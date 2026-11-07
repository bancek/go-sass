package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func trimTestSpan(text string) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(text), nil)
	return sasscommon.NewSimpleFileSpan(fs, 0, len(text))
}

func trimTestSpanText(t *testing.T, s sasscommon.FileSpan) string {
	t.Helper()
	text, err := s.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestTrimLeft(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"doesn't trim non-whitespace", "foo", "foo"},
		{"doesn't trim right-hand whitespace", "foo ", "foo "},
		{"trims whitespace", " \t\n\r\ffoo", "foo"},
		{"trims the entire span", "   ", ""},
		{"trims silent comments", "// foo\nbar", "bar"},
		{"trims loud comments", "/* foo */bar", "bar"},
		{"ignores a commented-out silent comment", "/* // */bar", "bar"},
		{"ignores a commented-out loud comment opener", "// /*\n\"*/\"", "\"*/\""},
		{"respects a loud comment closer that looks like it's in a string", "/* \"*/\" bar\"", "\" bar\""},
		{"ignores a silent comment opener that's in a string", "\"//\"", "\"//\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed, err := trimLeftSpan(trimTestSpan(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if got := trimTestSpanText(t, trimmed); got != tt.want {
				t.Errorf("trimLeftSpan(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimLeftTrimsTheEntireSpan(t *testing.T) {
	trimmed, err := trimLeftSpan(trimTestSpan("   "))
	if err != nil {
		t.Fatal(err)
	}
	end, err := trimmed.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if end.Offset != 3 {
		t.Errorf("trimLeft end offset = %d, want 3", end.Offset)
	}
}

func TestTrimRight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"doesn't trim non-whitespace", "foo", "foo"},
		{"doesn't trim left-hand whitespace", " foo", " foo"},
		{"trims whitespace", "foo \t\n\r\f", "foo"},
		{"trims the entire span", "   ", ""},
		{"trims silent comments", "foo\n// bar", "foo"},
		{"trims loud comments", "foo/* bar */", "foo"},
		{"ignores a commented-out loud comment closer with a string opener", "\"/*\"// */", "\"/*\""},
		{"ignores a commented-out loud comment closer with a commented-out opener", "// /*\nfoo\n// */ bar", "// /*\nfoo"},
		{"ignores a silent comment opener that's commented out on the same line", "/* // */ foo", "/* // */ foo"},
		{"ignores a silent comment opener that's commented out on an earlier line", "/*\nfoo\n// */ bar", "/*\nfoo\n// */ bar"},
		{"respects a loud comment opener that looks like it's in a string", "\"foo\"/*\" */", "\"foo\""},
		{"ignores a silent comment opener that's in a string", "\"//\" foo", "\"//\" foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed, err := trimRightSpan(trimTestSpan(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if got := trimTestSpanText(t, trimmed); got != tt.want {
				t.Errorf("trimRightSpan(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimRightTrimsTheEntireSpan(t *testing.T) {
	trimmed, err := trimRightSpan(trimTestSpan("   "))
	if err != nil {
		t.Fatal(err)
	}
	start, err := trimmed.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	if start.Offset != 0 {
		t.Errorf("trimRight start offset = %d, want 0", start.Offset)
	}
}

func TestTrim(t *testing.T) {
	trimmed, err := trimSpan(trimTestSpan("  /* c */ foo /* c */  "))
	if err != nil {
		t.Fatal(err)
	}
	if got := trimTestSpanText(t, trimmed); got != "foo" {
		t.Errorf("trimSpan = %q, want %q", got, "foo")
	}
}

func TestTrimIsLazy(t *testing.T) {
	for name, trim := range map[string]func(sasscommon.FileSpan) (sasscommon.FileSpan, error){
		"trim":      trimSpan,
		"trimLeft":  trimLeftSpan,
		"trimRight": trimRightSpan,
	} {
		t.Run(name, func(t *testing.T) {
			fs := sasscommon.NewFileSource([]byte(" abc "), nil)
			calls := 0
			lazy := sasscommon.NewLazyFileSpan(func() (sasscommon.FileSpan, error) {
				calls++
				return sasscommon.NewSimpleFileSpan(fs, 0, 5), nil
			})

			trimmed, err := trim(lazy)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatalf("%s resolved the underlying span eagerly (%d calls)", name, calls)
			}
			if _, err := trimmed.SpanText(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Errorf("underlying span resolved %d times, want 1", calls)
			}
		})
	}
}
