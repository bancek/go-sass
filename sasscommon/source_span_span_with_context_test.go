package sasscommon

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/termglyph"
)

func sscLoc(offset, line, column int) SourceLocation {
	return SourceLocation{Offset: offset, Line: line, Column: column}
}

func sscURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// --- Construction ---

func TestNewSourceSpanWithContextValid(t *testing.T) {
	start := sscLoc(0, 0, 0)
	end := sscLoc(5, 0, 5)
	ssc, err := NewSourceSpanWithContext(start, end, "hello", "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := ssc.SpanText()
	if text != "hello" {
		t.Errorf("SpanText() = %q, want %q", text, "hello")
	}
	ctx, _ := ssc.Context()
	if ctx != "hello" {
		t.Errorf("Context() = %q, want %q", ctx, "hello")
	}
}

func TestNewSourceSpanWithContextTextNotInContext(t *testing.T) {
	start := sscLoc(0, 0, 0)
	end := sscLoc(5, 0, 5)
	_, err := NewSourceSpanWithContext(start, end, "hello", "goodbye", nil)
	if err == nil {
		t.Fatal("expected error when text not in context")
	}
}

func TestNewSourceSpanWithContextBadColumn(t *testing.T) {
	start := sscLoc(0, 0, 5)
	end := sscLoc(5, 0, 10)
	_, err := NewSourceSpanWithContext(start, end, "hello", "hello world", nil)
	if err == nil {
		t.Fatal("expected error when column doesn't match")
	}
}

// --- Accessors ---

func TestSSWCAccessors(t *testing.T) {
	start := sscLoc(0, 0, 0)
	end := sscLoc(5, 0, 5)
	ssc, err := NewSourceSpanWithContext(start, end, "hello", "hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	text, err := ssc.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("SpanText() = %q", text)
	}

	ctx, err := ssc.Context()
	if err != nil {
		t.Fatal(err)
	}
	if ctx != "hello" {
		t.Errorf("Context() = %q", ctx)
	}

	loc, err := ssc.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	if loc != start {
		t.Errorf("StartLocation() = %v, want %v", loc, start)
	}

	loc, err = ssc.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if loc != end {
		t.Errorf("EndLocation() = %v, want %v", loc, end)
	}

	length, err := ssc.Length()
	if err != nil {
		t.Fatal(err)
	}
	if length != 5 {
		t.Errorf("Length() = %d, want 5", length)
	}

	file, err := ssc.File()
	if err != nil {
		t.Fatal(err)
	}
	if file != nil {
		t.Error("File() should return nil")
	}
}

func TestSSWCSourceURL(t *testing.T) {
	u := sscURL("file:///test.scss")
	ssc, err := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 5), "hello", "hello", u)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ssc.SourceURL()
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "file:///test.scss" {
		t.Errorf("SourceURL() = %v", got)
	}
}

func TestSSWCString(t *testing.T) {
	ssc, err := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 5), "hello", "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssc.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q", s)
	}
}

// --- Subspan ---

func TestSSWCSubspan(t *testing.T) {
	ssc, err := NewSourceSpanWithContext(sscLoc(10, 0, 0), sscLoc(16, 0, 0), "abcdef", "abcdef", nil)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := ssc.Subspan(2, 5)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := sub.SpanText()
	if text != "cde" {
		t.Errorf("Subspan text = %q, want %q", text, "cde")
	}
	start, _ := sub.StartLocation()
	if start.Offset != 12 {
		t.Errorf("Subspan start offset = %d, want 12", start.Offset)
	}
	end, _ := sub.EndLocation()
	if end.Offset != 15 {
		t.Errorf("Subspan end offset = %d, want 15", end.Offset)
	}
}

func TestSSWCSubspanFullRange(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, err := ssc.Subspan(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := sub.SpanText()
	if text != ssc.text {
		t.Errorf("Subspan full = %q, want %q", text, ssc.text)
	}
}

func TestSSWCSubspanEmpty(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, err := ssc.Subspan(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := sub.SpanText()
	if text != "" {
		t.Errorf("Subspan empty = %q", text)
	}
}

func TestSSWCSubspanOutOfBounds(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "hello", "hello", nil)
	_, err := ssc.Subspan(2, 10)
	if err == nil {
		t.Fatal("expected error for out-of-bounds subspan")
	}
}

func TestSSWCSubspanPreservesContext(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "hello", "hello world", nil)
	sub, err := ssc.Subspan(0, 3)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := sub.Context()
	if ctx != "hello world" {
		t.Errorf("Context after subspan = %q, want %q", ctx, "hello world")
	}
}

func TestSSWCSubspanMultiline(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(
		sscLoc(0, 0, 0),
		sscLoc(12, 2, 0),
		"hello\nworld\n",
		"hello\nworld\n",
		nil,
	)
	sub, err := ssc.Subspan(6, 11)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := sub.SpanText()
	if text != "world" {
		t.Errorf("Subspan multiline text = %q, want %q", text, "world")
	}
	start, _ := sub.StartLocation()
	if start.Offset != 6 || start.Line != 1 || start.Column != 0 {
		t.Errorf("Subspan multiline start = %v, want {6 1 0}", start)
	}
	end, _ := sub.EndLocation()
	if end.Offset != 11 || end.Line != 1 || end.Column != 5 {
		t.Errorf("Subspan multiline end = %v, want {11 1 5}", end)
	}
}

// --- TrimRight ---

func TestSSWCTrimRightNoOp(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(3, 0, 3), "abc", "abc", nil)
	trimmed, err := ssc.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("TrimRight no-op = %q", text)
	}
}

func TestSSWCTrimRight(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(6, 0, 0), "abc   ", "abc   ", nil)
	trimmed, err := ssc.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("TrimRight = %q, want %q", text, "abc")
	}
	end, _ := trimmed.EndLocation()
	if end.Offset != 3 {
		t.Errorf("TrimRight end offset = %d, want 3", end.Offset)
	}
}

func TestSSWCTrimRightNewlinesAndSpaces(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(6, 0, 0), "abc\n\r ", "abc\n\r ", nil)
	trimmed, err := ssc.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("TrimRight with newlines = %q, want %q", text, "abc")
	}
}

// --- TrimLeft ---

func TestSSWCTrimLeftNoOp(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(3, 0, 3), "abc", "abc", nil)
	trimmed, err := ssc.TrimLeft()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("TrimLeft no-op = %q", text)
	}
}

func TestSSWCTrimLeft(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "  abc", "  abc", nil)
	trimmed, err := ssc.TrimLeft()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("TrimLeft = %q, want %q", text, "abc")
	}
	start, _ := trimmed.StartLocation()
	if start.Offset != 2 {
		t.Errorf("TrimLeft start offset = %d, want 2", start.Offset)
	}
}

// --- Trim ---

func TestSSWCTrim(t *testing.T) {
	ssc, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(7, 0, 0), "  abc  ", "  abc  ", nil)
	trimmed, err := ssc.Trim()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := trimmed.SpanText()
	if text != "abc" {
		t.Errorf("Trim = %q, want %q", text, "abc")
	}
	start, _ := trimmed.StartLocation()
	end, _ := trimmed.EndLocation()
	if start.Offset != 2 || end.Offset != 5 {
		t.Errorf("Trim offsets = %d-%d, want 2-5", start.Offset, end.Offset)
	}
}

// --- Expand ---

func TestSSWCExpand(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(5, 0, 0), sscLoc(10, 0, 0), "hello", "hello..........", nil)
	b, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(15, 0, 0), "hello..........", "hello..........", nil)
	expanded, err := a.Expand(b)
	if err != nil {
		t.Fatal(err)
	}
	start, _ := expanded.StartLocation()
	end, _ := expanded.EndLocation()
	if start.Offset != 0 || end.Offset != 15 {
		t.Errorf("Expand offsets = %d-%d, want 0-15", start.Offset, end.Offset)
	}
}

func TestSSWCExpandSameStart(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "aaaaaaaaaa", "aaaaaaaaaa", nil)
	b, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "aaaaa", "aaaaaaaaaa", nil)
	expanded, err := a.Expand(b)
	if err != nil {
		t.Fatal(err)
	}
	start, _ := expanded.StartLocation()
	end, _ := expanded.EndLocation()
	if start.Offset != 0 || end.Offset != 10 {
		t.Errorf("ExpandSame offsets = %d-%d, want 0-10", start.Offset, end.Offset)
	}
}

func TestSSWCExpandDifferentURL(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "hello", "hello", sscURL("file:///a.scss"))
	b, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "hello", "hello", sscURL("file:///b.scss"))
	_, err := a.Expand(b)
	if err == nil {
		t.Fatal("expected error for different URLs")
	}
}

// --- Before ---

func TestSSWCBefore(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(3, 0, 3), sscLoc(6, 0, 6), "def", "abcdefghij", nil)
	beforeSpan, err := parent.Before(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := beforeSpan.SpanText()
	if text != "abc" {
		t.Errorf("Before = %q, want %q", text, "abc")
	}
}

func TestSSWCBeforeAtStart(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 5), "hello", "hello", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(2, 0, 2), "he", "hello", nil)
	_, err := parent.Before(sub)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSSWCBeforeDifferentURL(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", sscURL("file:///a.scss"))
	sub, _ := NewSourceSpanWithContext(sscLoc(3, 0, 3), sscLoc(6, 0, 6), "def", "abcdefghij", sscURL("file:///b.scss"))
	_, err := parent.Before(sub)
	if err == nil {
		t.Fatal("expected error for different URLs")
	}
}

func TestSSWCBeforeNotContained(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(4, 0, 4), "fg", "fghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 0), "fghij", "fghij", nil)
	_, err := parent.Before(sub)
	if err == nil {
		t.Fatal("expected error for sub not contained")
	}
}

// --- After ---

func TestSSWCAfter(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(3, 0, 3), sscLoc(6, 0, 6), "def", "abcdefghij", nil)
	afterSpan, err := parent.After(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := afterSpan.SpanText()
	if text != "ghij" {
		t.Errorf("After = %q, want %q", text, "ghij")
	}
}

func TestSSWCAfterAtEnd(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(5, 0, 5), "hello", "hello", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(3, 0, 3), sscLoc(5, 0, 5), "lo", "hello", nil)
	_, err := parent.After(sub)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSSWCAfterDifferentURL(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", sscURL("file:///a.scss"))
	sub, _ := NewSourceSpanWithContext(sscLoc(3, 0, 3), sscLoc(6, 0, 6), "def", "abcdefghij", sscURL("file:///b.scss"))
	_, err := parent.After(sub)
	if err == nil {
		t.Fatal("expected error for different URLs")
	}
}

// --- Between ---

func TestSSWCBetween(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(3, 0, 3), "c", "abcdefghij", nil)
	b, _ := NewSourceSpanWithContext(sscLoc(7, 0, 7), sscLoc(8, 0, 8), "h", "abcdefghij", nil)
	betweenSpan, err := a.Between(b)
	if err != nil {
		t.Fatal(err)
	}
	start, _ := betweenSpan.StartLocation()
	if start.Offset != 3 {
		t.Errorf("Between start offset = %d, want 3", start.Offset)
	}
	end, _ := betweenSpan.EndLocation()
	if end.Offset != 7 {
		t.Errorf("Between end offset = %d, want 7", end.Offset)
	}
}

func TestSSWCBetweenWrongOrder(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(7, 0, 7), sscLoc(8, 0, 8), "h", "abcdefghij", nil)
	b, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(3, 0, 3), "c", "abcdefghij", nil)
	_, err := a.Between(b)
	if err == nil {
		t.Fatal("expected error for wrong order")
	}
}

func TestSSWCBetweenDifferentURL(t *testing.T) {
	a, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(3, 0, 3), "c", "abcdefghij", sscURL("file:///a.scss"))
	b, _ := NewSourceSpanWithContext(sscLoc(7, 0, 7), sscLoc(8, 0, 8), "h", "abcdefghij", sscURL("file:///b.scss"))
	_, err := a.Between(b)
	if err == nil {
		t.Fatal("expected error for different URLs")
	}
}

// --- Contains ---

func TestSSWCContains(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(5, 0, 5), "cde", "abcdefghij", nil)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Contains = false, want true")
	}
}

func TestSSWCContainsFalse(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(5, 0, 0), sscLoc(10, 0, 0), "fghij", "fghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(15, 0, 0), "fghij", "fghij", nil)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Contains = true for out-of-range, want false")
	}
}

func TestSSWCContainsDifferentURL(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", sscURL("file:///a.scss"))
	sub, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(5, 0, 5), "cde", "abcdefghij", sscURL("file:///b.scss"))
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Contains = true for different URL, want false")
	}
}

func TestSSWCContainsNilURLs(t *testing.T) {
	parent, _ := NewSourceSpanWithContext(sscLoc(0, 0, 0), sscLoc(10, 0, 0), "abcdefghij", "abcdefghij", nil)
	sub, _ := NewSourceSpanWithContext(sscLoc(2, 0, 2), sscLoc(5, 0, 5), "cde", "abcdefghij", nil)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Contains = false for nil URLs, want true")
	}
}

// --- Message ---

func TestSSWCMessage(t *testing.T) {
	start := sscLoc(0, 0, 4)
	end := sscLoc(7, 0, 7)
	ssc, err := NewSourceSpanWithContext(start, end, "bar", "foo bar baz", sscURL("foo.dart"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := ssc.Message("oh no", HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"line 1, column 5 of foo.dart: oh no",
		"  ,",
		"1 | foo bar baz",
		"  |     ^^^",
		"  '",
	}, "\n")
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestSSWCMessageNoURL(t *testing.T) {
	start := sscLoc(0, 0, 4)
	end := sscLoc(7, 0, 7)
	ssc, err := NewSourceSpanWithContext(start, end, "bar", "foo bar baz", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ssc.Message("oh no", HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"line 1, column 5: oh no",
		"  ,",
		"1 | foo bar baz",
		"  |     ^^^",
		"  '",
	}, "\n")
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestSSWCMessageNoColor(t *testing.T) {
	start := sscLoc(0, 0, 4)
	end := sscLoc(7, 0, 7)
	u := sscURL("foo.dart")
	ssc, err := NewSourceSpanWithContext(start, end, "bar", "foo bar baz", u)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ssc.Message("oh no", HighlightOptions{Color: false, Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("output contains ANSI code when color is false:\n%q", out)
	}
}

// --- Highlight ---

func TestSSWCHighlight(t *testing.T) {
	start := sscLoc(0, 0, 4)
	end := sscLoc(7, 0, 7)
	ssc, err := NewSourceSpanWithContext(start, end, "bar", "foo bar baz", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ssc.Highlight(HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"  ,",
		"1 | foo bar baz",
		"  |     ^^^",
		"  '",
	}, "\n")
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

// --- MessageMultiple ---

func TestSSWCMessageMultiple(t *testing.T) {
	text := "foo bar baz\nwhiz bang boom\nzip zap zop\n"
	fs := NewFileSource([]byte(text), nil)
	u := sscURL("file1.txt")
	start := sscLoc(17, 1, 5)
	end := sscLoc(21, 1, 9)
	ssc, err := NewSourceSpanWithContext(start, end, "bang", "whiz bang boom", u)
	if err != nil {
		t.Fatal(err)
	}
	secondary := map[FileSpan]string{
		NewSimpleFileSpan(fs, 4, 7): "three",
	}
	out, err := ssc.MessageMultiple("oh no", "one", secondary, HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	// The output should contain the message prefix and the highlight markers.
	if !strings.Contains(out, "line 2, column 6 of file1.txt: oh no") {
		t.Errorf("output missing message prefix:\n%s", out)
	}
	if !strings.Contains(out, "one") || !strings.Contains(out, "three") {
		t.Errorf("output missing labels:\n%s", out)
	}
}

// --- HighlightMultiple ---

func TestSSWCHighlightMultiple(t *testing.T) {
	text := "foo bar baz\nwhiz bang boom\nzip zap zop\n"
	fs := NewFileSource([]byte(text), nil)
	start := sscLoc(4, 0, 4)
	end := sscLoc(7, 0, 7)
	ssc, err := NewSourceSpanWithContext(start, end, "bar", "foo bar baz", nil)
	if err != nil {
		t.Fatal(err)
	}
	secondary := map[FileSpan]string{
		NewSimpleFileSpan(fs, 8, 11): "baz",
	}
	out, err := ssc.HighlightMultiple("primary", secondary, HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "primary") || !strings.Contains(out, "baz") {
		t.Errorf("output missing labels:\n%s", out)
	}
}
