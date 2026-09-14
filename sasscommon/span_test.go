package sasscommon

import (
	"net/url"
	"testing"
)

func testFileSource(text string) *FileSource {
	return NewFileSource([]byte(text), nil)
}

func testURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// --- Construction and accessors ---

func TestSimpleFileSpanAccessors(t *testing.T) {
	fs := testFileSource("hello")
	s := NewSimpleFileSpan(fs, 0, 5)

	file, err := s.File()
	if err != nil {
		t.Fatal(err)
	}
	if file != fs {
		t.Error("File() should return the FileSource")
	}

	startLoc, err := s.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc != (SourceLocation{Offset: 0, Line: 0, Column: 0}) {
		t.Errorf("StartLocation() = %v", startLoc)
	}

	endLoc, err := s.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if endLoc != (SourceLocation{Offset: 5, Line: 0, Column: 5}) {
		t.Errorf("EndLocation() = %v", endLoc)
	}

	text, err := s.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("SpanText() = %q", text)
	}

	length, err := s.Length()
	if err != nil {
		t.Fatal(err)
	}
	if length != 5 {
		t.Errorf("Length() = %d", length)
	}

	ctx, err := s.Context()
	if err != nil {
		t.Fatal(err)
	}
	if ctx != "hello" {
		t.Errorf("Context() = %q, want %q", ctx, "hello")
	}
}

func TestNewFileSpanIsFileSpan(t *testing.T) {
	fs := testFileSource("x")
	var f FileSpan = NewFileSpan(fs, 0, 1)
	if f == nil {
		t.Fatal("NewFileSpan returned nil interface")
	}
	text, err := f.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "x" {
		t.Errorf("SpanText() = %q", text)
	}
}

func TestNewFileSpanInFile(t *testing.T) {
	fs := testFileSource("..........")
	span := NewFileSpanInFile(fs, 10, 20)
	startLoc, err := span.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := span.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 10 || endLoc.Offset != 20 {
		t.Errorf("offsets: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

func TestLength(t *testing.T) {
	fs := testFileSource("hello")
	s := NewSimpleFileSpan(fs, 5, 10)
	length, err := s.Length()
	if err != nil {
		t.Fatal(err)
	}
	if length != 5 {
		t.Errorf("Length() = %d, want 5", length)
	}
}

func TestFileText(t *testing.T) {
	fs := NewFileSource([]byte("x\ny\n"), nil)
	s := NewSimpleFileSpan(fs, 0, 1)
	file, err := s.File()
	if err != nil {
		t.Fatal(err)
	}
	if file.Text() != "x\ny\n" {
		t.Errorf("File().Text() = %q", file.Text())
	}
}

// --- Expand ---

func TestExpand(t *testing.T) {
	fs := testFileSource("hello..........")
	a := NewSimpleFileSpan(fs, 5, 10)
	b := NewSimpleFileSpan(fs, 0, 15)
	expanded, err := a.Expand(b)
	if err != nil {
		t.Fatal(err)
	}
	startLoc, err := expanded.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := expanded.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 0 || endLoc.Offset != 15 {
		t.Errorf("Expand: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

func TestExpandSameStart(t *testing.T) {
	fs := testFileSource("aaaaaaaaaa")
	a := NewSimpleFileSpan(fs, 0, 10)
	b := NewSimpleFileSpan(fs, 0, 5)
	expanded, err := a.Expand(b)
	if err != nil {
		t.Fatal(err)
	}
	startLoc, err := expanded.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := expanded.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 0 || endLoc.Offset != 10 {
		t.Errorf("ExpandSame: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

// --- TrimRight ---

func TestTrimRightNoWhitespace(t *testing.T) {
	fs := testFileSource("abc")
	s := NewSimpleFileSpan(fs, 0, 3)
	trimmed, err := s.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, err := trimmed.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("TrimRight() = %q", text)
	}
}

func TestTrimRightSpaces(t *testing.T) {
	fs := testFileSource("abc  ")
	s := NewSimpleFileSpan(fs, 0, 5)
	trimmed, err := s.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, err := trimmed.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("TrimRight() = %q, want %q", text, "abc")
	}
	endLoc, err := trimmed.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if endLoc.Offset != 3 {
		t.Errorf("TrimRight end offset = %d, want 3", endLoc.Offset)
	}
}

func TestTrimRightNewlinesAndSpaces(t *testing.T) {
	fs := testFileSource("abc\n\r ")
	s := NewSimpleFileSpan(fs, 0, 6)
	trimmed, err := s.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, err := trimmed.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("TrimRight() = %q, want %q", text, "abc")
	}
}

// --- Before ---

func TestBefore(t *testing.T) {
	fs := testFileSource("abcdefghij")
	parent := NewSimpleFileSpan(fs, 0, 10)
	sub := NewSimpleFileSpan(fs, 3, 6)
	beforeSpan, err := parent.Before(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := beforeSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("Before() = %q, want %q", text, "abc")
	}
}

func TestBeforeDifferentFile(t *testing.T) {
	u1, _ := url.Parse("file:///a.scss")
	u2, _ := url.Parse("file:///b.scss")
	fs1 := NewFileSource([]byte("abcdefghij"), u1)
	fs2 := NewFileSource([]byte("x"), u2)
	parent := NewSimpleFileSpan(fs1, 0, 10)
	sub := NewSimpleFileSpan(fs2, 0, 1)
	_, err := parent.Before(sub)
	if err == nil {
		t.Error("expected error for different file")
	}
}

func TestBeforeNotContained(t *testing.T) {
	fs := testFileSource("fghij")
	parent := NewSimpleFileSpan(fs, 2, 4)
	sub := NewSimpleFileSpan(fs, 0, 5)
	_, err := parent.Before(sub)
	if err == nil {
		t.Error("expected error for sub not contained in parent")
	}
}

func TestBeforeAtStart(t *testing.T) {
	fs := testFileSource("hello")
	parent := NewSimpleFileSpan(fs, 0, 5)
	sub := NewSimpleFileSpan(fs, 0, 2)
	beforeSpan, err := parent.Before(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := beforeSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("Before() at start = %q, want empty", text)
	}
}

// --- After ---

func TestAfter(t *testing.T) {
	fs := testFileSource("abcdefghij")
	parent := NewSimpleFileSpan(fs, 0, 10)
	sub := NewSimpleFileSpan(fs, 3, 6)
	afterSpan, err := parent.After(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := afterSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "ghij" {
		t.Errorf("After() = %q, want %q", text, "ghij")
	}
}

func TestAfterAtEnd(t *testing.T) {
	fs := testFileSource("hello")
	parent := NewSimpleFileSpan(fs, 0, 5)
	sub := NewSimpleFileSpan(fs, 3, 5)
	afterSpan, err := parent.After(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := afterSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("After() at end = %q, want empty", text)
	}
}

// --- Between ---

func TestBetween(t *testing.T) {
	fs := testFileSource("abcdefghij")
	a := NewSimpleFileSpan(fs, 2, 3)
	b := NewSimpleFileSpan(fs, 7, 8)
	betweenSpan, err := a.Between(b)
	if err != nil {
		t.Fatal(err)
	}
	text, err := betweenSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "defg" {
		t.Errorf("Between() = %q, want %q", text, "defg")
	}
}

func TestBetweenWrongOrder(t *testing.T) {
	fs := testFileSource("abcdefghij")
	a := NewSimpleFileSpan(fs, 7, 8)
	b := NewSimpleFileSpan(fs, 2, 3)
	_, err := a.Between(b)
	if err == nil {
		t.Error("expected error for wrong order")
	}
}

// --- Contains ---

func TestContains(t *testing.T) {
	fs := testFileSource("abcdefghij")
	parent := NewSimpleFileSpan(fs, 0, 10)
	sub := NewSimpleFileSpan(fs, 2, 5)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Contains() = false, want true")
	}
}

func TestContainsDifferentFile(t *testing.T) {
	u1, _ := url.Parse("file:///f.scss")
	u2, _ := url.Parse("file:///g.scss")
	fs1 := NewFileSource([]byte("abcdefghij"), u1)
	fs2 := NewFileSource([]byte("cde"), u2)
	parent := NewSimpleFileSpan(fs1, 0, 10)
	sub := NewSimpleFileSpan(fs2, 2, 5)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Contains() = true for different file, want false")
	}
}

func TestContainsOutsideRange(t *testing.T) {
	fs := testFileSource("fghij")
	parent := NewSimpleFileSpan(fs, 5, 10)
	sub := NewSimpleFileSpan(fs, 0, 15)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Contains() = true for out-of-range sub, want false")
	}
}

func TestContainsNilURLs(t *testing.T) {
	fs := testFileSource("abcdefghij")
	parent := NewSimpleFileSpan(fs, 0, 10)
	sub := NewSimpleFileSpan(fs, 2, 5)
	got, err := parent.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Contains() = false for nil URLs in same file, want true")
	}
}

// --- Subspan ---

func TestSubspan(t *testing.T) {
	fs := testFileSource("abcdefghij")
	s := NewSimpleFileSpan(fs, 0, 10)
	sub, err := s.Subspan(2, 5)
	if err != nil {
		t.Fatal(err)
	}
	text, err := sub.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "cde" {
		t.Errorf("Subspan() text = %q, want %q", text, "cde")
	}
	startLoc, err := sub.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := sub.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 2 || endLoc.Offset != 5 {
		t.Errorf("Subspan() offsets: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

func TestSubspanFullRange(t *testing.T) {
	fs := NewFileSource([]byte("abcdefghijklm"), nil)
	s := NewSimpleFileSpan(fs, 3, 13)
	sub, err := s.Subspan(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	text, err := sub.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "defghijklm" {
		t.Errorf("Subspan full = %q", text)
	}
	startLoc, err := sub.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := sub.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 3 || endLoc.Offset != 13 {
		t.Errorf("Subspan offsets: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

func TestSubspanEmpty(t *testing.T) {
	fs := testFileSource("abcdefghij")
	s := NewSimpleFileSpan(fs, 5, 15)
	sub, err := s.Subspan(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	text, err := sub.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("Subspan empty = %q", text)
	}
}

// --- BogusSpan ---

func TestBogusSpan(t *testing.T) {
	file, err := BogusSpan.File()
	if err != nil {
		t.Fatal(err)
	}
	if file != nil {
		t.Error("BogusSpan.File() should be nil")
	}
	text, err := BogusSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("BogusSpan.SpanText() = %q", text)
	}
}

// --- Map key behavior ---

func TestSimpleFileSpanAsMapKey(t *testing.T) {
	fs := testFileSource("hello")
	a := NewSimpleFileSpan(fs, 0, 5)
	a2 := NewSimpleFileSpan(fs, 0, 5)
	b := NewSimpleFileSpan(fs, 1, 6)

	m := map[FileSpan]string{a: "first"}
	m[a2] = "still-first"
	m[b] = "second"

	if m[a] != "still-first" {
		t.Errorf("key a = %q, want %q", m[a], "still-first")
	}
	if m[a2] != "still-first" {
		t.Errorf("key a2 = %q, want %q", m[a2], "still-first")
	}
	if m[b] != "second" {
		t.Errorf("key b = %q, want %q", m[b], "second")
	}
	if len(m) != 2 {
		t.Errorf("map size = %d, want 2", len(m))
	}
}

func TestSimpleFileSpanMapKeyDifferentFiles(t *testing.T) {
	fs1 := testFileSource("hello")
	fs2 := testFileSource("hello")
	a := NewSimpleFileSpan(fs1, 0, 5)
	b := NewSimpleFileSpan(fs2, 0, 5)
	// Different *FileSource pointers → different struct values → different map keys
	m := map[FileSpan]string{a: "first"}
	if _, exists := m[b]; exists {
		t.Error("different *FileSource pointer should NOT match as map key")
	}
}

func TestLazyAndSimpleSpanAsSeparateMapKeys(t *testing.T) {
	fs := testFileSource("abc")
	simple := NewSimpleFileSpan(fs, 0, 3)
	lazy := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 3), nil
	})
	m := map[FileSpan]string{simple: "simple", lazy: "lazy"}
	if len(m) != 2 {
		t.Errorf("LazyFileSpan and SimpleFileSpan should be separate keys, got %d", len(m))
	}
}

// --- String ---

func TestString(t *testing.T) {
	fs := testFileSource("hello")
	s := NewSimpleFileSpan(fs, 0, 5)
	str, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello" {
		t.Errorf("String() = %q", str)
	}
}

// --- FileSource methods ---

func TestFileSourceGetLine(t *testing.T) {
	fs := NewFileSource([]byte("foo\nbar\nbaz"), nil)
	if fs.GetLine(0) != 0 {
		t.Errorf("GetLine(0) = %d", fs.GetLine(0))
	}
	if fs.GetLine(4) != 1 {
		t.Errorf("GetLine(4) = %d", fs.GetLine(4))
	}
	if fs.GetLine(8) != 2 {
		t.Errorf("GetLine(8) = %d", fs.GetLine(8))
	}
}

func TestFileSourceGetColumn(t *testing.T) {
	fs := NewFileSource([]byte("foo\nbar\nbaz"), nil)
	if fs.GetColumn(0) != 0 {
		t.Errorf("GetColumn(0) = %d", fs.GetColumn(0))
	}
	if fs.GetColumn(5) != 1 {
		t.Errorf("GetColumn(5) = %d", fs.GetColumn(5))
	}
	if fs.GetColumn(6) != 2 {
		t.Errorf("GetColumn(6) = %d", fs.GetColumn(6))
	}
}

func TestFileSourceGetOffset(t *testing.T) {
	fs := NewFileSource([]byte("foo\nbar\nbaz"), nil)
	if fs.GetOffset(1, 0) != 4 {
		t.Errorf("GetOffset(1, 0) = %d", fs.GetOffset(1, 0))
	}
	if fs.GetOffset(1) != 4 {
		t.Errorf("GetOffset(1) = %d", fs.GetOffset(1))
	}
}

func TestFileSourceGetText(t *testing.T) {
	fs := NewFileSource([]byte("hello world"), nil)
	if fs.GetText(0, 5) != "hello" {
		t.Errorf("GetText(0, 5) = %q", fs.GetText(0, 5))
	}
	if fs.GetText(6) != "world" {
		t.Errorf("GetText(6) = %q", fs.GetText(6))
	}
}

// --- Free functions ---

func TestSameURL(t *testing.T) {
	if !sameURL(nil, nil) {
		t.Error("sameURL(nil, nil) should be true")
	}
	a := testURL("file:///a.scss")
	b := testURL("file:///b.scss")
	if !sameURL(a, a) {
		t.Error("sameURL(a, a) should be true")
	}
	if sameURL(a, b) {
		t.Error("sameURL(a, b) should be false")
	}
	if sameURL(a, nil) {
		t.Error("sameURL(a, nil) should be false")
	}
	if sameURL(nil, a) {
		t.Error("sameURL(nil, a) should be false")
	}
}

func TestIsWhitespaceByte(t *testing.T) {
	if !isWhitespaceByte(' ') {
		t.Error("' ' should be whitespace")
	}
	if !isWhitespaceByte('\t') {
		t.Error("'\\t' should be whitespace")
	}
	if !isWhitespaceByte('\n') {
		t.Error("'\\n' should be whitespace")
	}
	if !isWhitespaceByte('\r') {
		t.Error("'\\r' should be whitespace")
	}
	if !isWhitespaceByte('\f') {
		t.Error("'\\f' should be whitespace")
	}
	if isWhitespaceByte('a') {
		t.Error("'a' should not be whitespace")
	}
}

func TestIsName(t *testing.T) {
	if !isName('a') {
		t.Error("'a' should be a name char")
	}
	if !isName('Z') {
		t.Error("'Z' should be a name char")
	}
	if !isName('0') {
		t.Error("'0' should be a name char")
	}
	if !isName('_') {
		t.Error("'_' should be a name char")
	}
	if !isName('-') {
		t.Error("'-' should be a name char")
	}
	if !isName(0x80) {
		t.Error("0x80 should be a name char")
	}
	if isName('.') {
		t.Error("'.' should not be a name char")
	}
	if isName(' ') {
		t.Error("' ' should not be a name char")
	}
}

func TestColumnAt(t *testing.T) {
	if columnAt("abc", 0) != 0 {
		t.Errorf("columnAt at 0: %d", columnAt("abc", 0))
	}
	if columnAt("abc", 1) != 1 {
		t.Errorf("columnAt at 1: %d", columnAt("abc", 1))
	}
	if columnAt("a\nb", 2) != 0 {
		t.Errorf("columnAt after newline: %d", columnAt("a\nb", 2))
	}
	if columnAt("a\nbc", 3) != 1 {
		t.Errorf("columnAt after newline + 1: %d", columnAt("a\nbc", 3))
	}
}

// --- MultiSpan ---

func TestMultiSpanDelegatesSpanText(t *testing.T) {
	fs := testFileSource("hello")
	primary := NewSimpleFileSpan(fs, 0, 5)
	ms := NewMultiSpan(primary, "label", nil)
	text, err := ms.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("SpanText() = %q", text)
	}
}

func TestMultiSpanExpandPreservesMetadata(t *testing.T) {
	fs := testFileSource("abcdefghij")
	primary := NewSimpleFileSpan(fs, 3, 8)
	secondary := map[FileSpan]string{
		NewSimpleFileSpan(fs, 0, 1): "note",
	}
	ms := NewMultiSpan(primary, "primary-label", secondary)
	other := NewSimpleFileSpan(fs, 0, 10)
	expanded, err := ms.Expand(other)
	if err != nil {
		t.Fatal(err)
	}
	start, _ := expanded.SpanStart()
	end, _ := expanded.SpanEnd()
	if start.Offset != 0 || end.Offset != 10 {
		t.Errorf("Expand offsets: start=%d end=%d", start.Offset, end.Offset)
	}
	if expanded.PrimaryLabel != "primary-label" {
		t.Errorf("PrimaryLabel = %q", expanded.PrimaryLabel)
	}
	if len(expanded.SecondarySpans) != 1 {
		t.Errorf("lost secondary spans: got %d", len(expanded.SecondarySpans))
	}
}
