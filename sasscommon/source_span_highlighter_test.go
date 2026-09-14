package sasscommon

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/termglyph"
)

const hlTestText = "foo bar baz\nwhiz bang boom\nzip zap zop\n"

func asciiOpts() HighlightOptions { return HighlightOptions{Glyphs: termglyph.AsciiGlyphs} }
func colorOpts() HighlightOptions {
	return HighlightOptions{Color: true, Glyphs: termglyph.AsciiGlyphs}
}
func noColorOpts() HighlightOptions {
	return HighlightOptions{Color: false, Glyphs: termglyph.AsciiGlyphs}
}

func highlight(fs *FileSource, start, end int) string {
	h, _ := NewHighlighter(NewSimpleFileSpan(fs, start, end), asciiOpts())
	out, _ := h.Highlight()
	return out
}

func highlightMultiple(primary FileSpan, label string, secondary map[FileSpan]string) string {
	h, _ := NewHighlighterMultiple(primary, label, secondary, asciiOpts())
	out, _ := h.Highlight()
	return out
}

// --- Single span ---

func TestHighlightPointsToSpan(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 7)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |     ^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightNoURL(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 7)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |     ^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightSingleLineFile(t *testing.T) {
	fs := NewFileSource([]byte("foo bar"), nil)
	got := highlight(fs, 0, 7)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar",
		"  | ^^^^^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightIncludesTrailingNewline(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 8, 12)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |         ^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// --- Multiline span ---

func TestHighlightMultiline(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 34)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,-----^",
		"2 | | whiz bang boom",
		"3 | | zip zap zop",
		"  | '-------^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightFullFirstLine(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 0, 34)
	want := strings.Join([]string{
		"  ,",
		"1 | / foo bar baz",
		"2 | | whiz bang boom",
		"3 | | zip zap zop",
		"  | '-------^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightBeginsAtEndOfLine(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 11, 34)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,------------^",
		"2 | | whiz bang boom",
		"3 | | zip zap zop",
		"  | '-------^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightEndsAtBeginningOfLine(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 28)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,-----^",
		"2 | | whiz bang boom",
		"3 | | zip zap zop",
		"  | '-^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightFullLastLine(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 27)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,-----^",
		"2 | \\ whiz bang boom",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightFullLastLineNoTrailingNewline(t *testing.T) {
	fs := NewFileSource([]byte("foo bar baz\nwhiz bang boom\nzip zap zop"), nil)
	got := highlight(fs, 4, 26)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,-----^",
		"2 | \\ whiz bang boom",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightFullLastLineAtEndOfFile(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlight(fs, 4, 39)
	want := strings.Join([]string{
		"  ,",
		"1 |   foo bar baz",
		"  | ,-----^",
		"2 | | whiz bang boom",
		"3 | \\ zip zap zop",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// --- Tabs ---

func TestHighlightTabsBeforeSpan(t *testing.T) {
	fs := NewFileSource([]byte("foo\tbar baz"), nil)
	got := highlight(fs, 4, 7)
	want := strings.Join([]string{
		"  ,",
		"1 | foo    bar baz",
		"  |        ^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightTabsWithinSpan(t *testing.T) {
	fs := NewFileSource([]byte("foo bar\tbaz bang"), nil)
	got := highlight(fs, 4, 11)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar    baz bang",
		"  |     ^^^^^^^^^^",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// --- Multiple spans (same URL, grouped together) ---

func TestHighlightMultipleSeparate(t *testing.T) {
	u, _ := url.Parse("file1.txt")
	ssc1, _ := NewSourceSpanWithContext(
		SourceLocation{Offset: 17, Line: 1, Column: 5},
		SourceLocation{Offset: 21, Line: 1, Column: 9},
		"bang", "whiz bang boom", u)
	ssc2, _ := NewSourceSpanWithContext(
		SourceLocation{Offset: 4, Line: 0, Column: 4},
		SourceLocation{Offset: 7, Line: 0, Column: 7},
		"bar", "foo bar baz", u)
	ssc3, _ := NewSourceSpanWithContext(
		SourceLocation{Offset: 31, Line: 2, Column: 4},
		SourceLocation{Offset: 34, Line: 2, Column: 7},
		"zap", "zip zap zop", u)
	got := highlightMultiple(ssc1, "one",
		map[FileSpan]string{ssc2: "three", ssc3: "two"},
	)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |     === three",
		"2 | whiz bang boom",
		"  |      ^^^^ one",
		"3 | zip zap zop",
		"  |     === two",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHighlightMultipleSameLine(t *testing.T) {
	fs := NewFileSource([]byte("foo bar baz\n"), nil)
	got := highlightMultiple(
		NewSimpleFileSpan(fs, 4, 7), "bar",
		map[FileSpan]string{
			NewSimpleFileSpan(fs, 8, 11): "z",
		},
	)
	if !strings.Contains(got, "bar") || !strings.Contains(got, "z") {
		t.Errorf("output missing expected labels:\n%s", got)
	}
}

func TestHighlightMultipleMultiline(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	got := highlightMultiple(
		NewSimpleFileSpan(fs, 4, 34), "outer",
		map[FileSpan]string{
			NewSimpleFileSpan(fs, 14, 18): "inner",
		},
	)
	if !strings.Contains(got, "inner") || !strings.Contains(got, "outer") {
		t.Errorf("output missing expected multiline labels:\n%s", got)
	}
}

// --- Multiple spans from different nil-url sources ---

func TestHighlightMultipleDifferentSources(t *testing.T) {
	fs1 := NewFileSource([]byte("a b c\n"), nil)
	fs2 := NewFileSource([]byte("x y z\n"), nil)
	got := highlightMultiple(
		NewSimpleFileSpan(fs1, 2, 3), "primary",
		map[FileSpan]string{
			NewSimpleFileSpan(fs2, 2, 3): "note",
		},
	)
	if !strings.Contains(got, "primary") || !strings.Contains(got, "note") {
		t.Errorf("output missing labels:\n%s", got)
	}
}

// --- Multiple spans from same file URL ---

func TestHighlightMultipleSameFileURL(t *testing.T) {
	u, _ := url.Parse("file1.txt")
	ssc1, _ := NewSourceSpanWithContext(
		SourceLocation{Offset: 17, Line: 1, Column: 5},
		SourceLocation{Offset: 21, Line: 1, Column: 9},
		"bang", "whiz bang boom", u,
	)
	ssc2, _ := NewSourceSpanWithContext(
		SourceLocation{Offset: 4, Line: 0, Column: 4},
		SourceLocation{Offset: 7, Line: 0, Column: 7},
		"bar", "foo bar baz", u,
	)
	got := highlightMultiple(ssc1, "one",
		map[FileSpan]string{ssc2: "three"},
	)
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |     === three",
		"2 | whiz bang boom",
		"  |      ^^^^ one",
		"  '",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// --- Color ---

func TestHighlightWithColor(t *testing.T) {
	fs := NewFileSource([]byte("foo bar baz"), nil)
	h, _ := NewHighlighter(NewSimpleFileSpan(fs, 4, 7), colorOpts())
	out, _ := h.Highlight()
	if !strings.Contains(out, "\x1b[31m") {
		t.Errorf("expected red ANSI code:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("expected reset ANSI code:\n%q", out)
	}
}

func TestHighlightNoColor(t *testing.T) {
	fs := NewFileSource([]byte("foo bar baz"), nil)
	h, _ := NewHighlighter(NewSimpleFileSpan(fs, 4, 7), noColorOpts())
	out, _ := h.Highlight()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("unexpected ANSI code with color=false:\n%q", out)
	}
}

func TestHighlightCustomColor(t *testing.T) {
	fs := NewFileSource([]byte("foo bar baz"), nil)
	h, _ := NewHighlighter(NewSimpleFileSpan(fs, 4, 7),
		HighlightOptions{Color: "\x1b[35m", Glyphs: termglyph.AsciiGlyphs},
	)
	out, _ := h.Highlight()
	if !strings.Contains(out, "\x1b[35m") {
		t.Errorf("expected custom color:\n%q", out)
	}
}

// --- Unicode glyphs (default) ---

func TestHighlightUnicodeGlyphs(t *testing.T) {
	fs := NewFileSource([]byte(hlTestText), nil)
	h, _ := NewHighlighter(NewSimpleFileSpan(fs, 4, 34), HighlightOptions{})
	out, _ := h.Highlight()
	if !strings.Contains(out, "│") {
		t.Errorf("expected Unicode vertical line:\n%s", out)
	}
	if !strings.Contains(out, "┌") {
		t.Errorf("expected Unicode corner:\n%s", out)
	}
}
