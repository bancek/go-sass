package sasscommon

import (
	"net/url"
	"testing"
)

// --- Lazy creation ---

func TestLazyFileSpanBuilderNotCalledUntilGet(t *testing.T) {
	called := 0
	fs := NewFileSource([]byte("x"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		called++
		return NewSimpleFileSpan(fs, 0, 1), nil
	})
	if called != 0 {
		t.Error("builder called before Get()")
	}
	_, err := l.Get()
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Errorf("builder called %d times after first Get(), want 1", called)
	}
}

func TestLazyFileSpanCachesResult(t *testing.T) {
	called := 0
	fs := NewFileSource([]byte("x"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		called++
		return NewSimpleFileSpan(fs, 0, 1), nil
	})
	_, err := l.Get()
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.Get()
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.Get()
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Errorf("builder called %d times, want 1", called)
	}
}

// --- Delegation: Span methods ---

func TestLazyFileSpanFile(t *testing.T) {
	fs := NewFileSource([]byte("a"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 1), nil
	})
	file, err := l.File()
	if err != nil {
		t.Fatal(err)
	}
	if file != fs {
		t.Error("File() should return the FileSource")
	}
}

func TestLazyFileSpanStartLocation(t *testing.T) {
	fs := NewFileSource([]byte("hello\n"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	loc, err := l.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	if loc.Offset != 0 || loc.Line != 0 || loc.Column != 0 {
		t.Errorf("StartLocation() = %v", loc)
	}
}

func TestLazyFileSpanEndLocation(t *testing.T) {
	fs := NewFileSource([]byte("hello\n"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	loc, err := l.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if loc.Offset != 5 || loc.Line != 0 || loc.Column != 5 {
		t.Errorf("EndLocation() = %v", loc)
	}
}

func TestLazyFileSpanSpanText(t *testing.T) {
	fs := NewFileSource([]byte("hello"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	text, err := l.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("SpanText() = %q", text)
	}
}

func TestLazyFileSpanLength(t *testing.T) {
	fs := NewFileSource([]byte("hello"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 2, 7), nil
	})
	length, err := l.Length()
	if err != nil {
		t.Fatal(err)
	}
	if length != 5 {
		t.Errorf("Length() = %d", length)
	}
}

// --- Delegation: FileSpan methods ---

func TestLazyFileSpanSourceURL(t *testing.T) {
	u := &url.URL{Scheme: "file", Path: "/test.scss"}
	fs := NewFileSource([]byte("a"), u)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 1), nil
	})
	sourceURL, err := l.SourceURL()
	if err != nil {
		t.Fatal(err)
	}
	if sourceURL != u {
		t.Error("SourceURL() returned wrong pointer")
	}
}

func TestLazyFileSpanContext(t *testing.T) {
	fs := NewFileSource([]byte("body { }\n"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 8), nil
	})
	c, err := l.Context()
	if err != nil {
		t.Fatal(err)
	}
	if c == "" {
		t.Error("Context() returned empty")
	}
}

func TestLazyFileSpanString(t *testing.T) {
	fs := NewFileSource([]byte("hello"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	str, err := l.String()
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello" {
		t.Errorf("String() = %q", str)
	}
}

func TestLazyFileSpanExpand(t *testing.T) {
	fs := NewFileSource([]byte("..........\n"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 3, 8), nil
	})
	other := NewSimpleFileSpan(fs, 0, 10)
	expanded, err := l.Expand(other)
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
		t.Errorf("Expand: start=%d end=%d", startLoc.Offset, endLoc.Offset)
	}
}

func TestLazyFileSpanSubspan(t *testing.T) {
	fs := NewFileSource([]byte("abcde"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	sub, err := l.Subspan(1, 4)
	if err != nil {
		t.Fatal(err)
	}
	text, err := sub.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "bcd" {
		t.Errorf("Subspan = %q", text)
	}
}

func TestLazyFileSpanTrimRight(t *testing.T) {
	fs := NewFileSource([]byte("abc  "), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 5), nil
	})
	trimmed, err := l.TrimRight()
	if err != nil {
		t.Fatal(err)
	}
	text, err := trimmed.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("TrimRight = %q", text)
	}
}

func TestLazyFileSpanBefore(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 6), nil
	})
	sub := NewSimpleFileSpan(fs, 3, 6)
	beforeSpan, err := l.Before(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := beforeSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Errorf("Before = %q", text)
	}
}

func TestLazyFileSpanAfter(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 6), nil
	})
	sub := NewSimpleFileSpan(fs, 0, 3)
	afterSpan, err := l.After(sub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := afterSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "def" {
		t.Errorf("After = %q", text)
	}
}

func TestLazyFileSpanBetween(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	a := NewSimpleFileSpan(fs, 1, 2)
	b := NewSimpleFileSpan(fs, 4, 5)
	betweenSpan, err := a.Between(b)
	if err != nil {
		t.Fatal(err)
	}
	text, err := betweenSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "cd" {
		t.Errorf("Between = %q", text)
	}
}

func TestLazyFileSpanContains(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	l := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 6), nil
	})
	sub := NewSimpleFileSpan(fs, 2, 4)
	got, err := l.Contains(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Contains = false")
	}
}

// --- LazyFileSpan as FileSpan argument ---

func TestLazyFileSpanPassedWhereFileSpanExpected(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	inner := NewSimpleFileSpan(fs, 2, 4)
	lazy := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 0, 6), nil
	})

	got, err := lazy.Contains(inner)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("LazyFileSpan.Contains(SimpleFileSpan) = false")
	}
}

func TestLazyFileSpanSubPassedToSimpleFileSpan(t *testing.T) {
	fs := NewFileSource([]byte("abcdef"), nil)
	parent := NewSimpleFileSpan(fs, 0, 6)
	lazySub := NewLazyFileSpan(func() (FileSpan, error) {
		return NewSimpleFileSpan(fs, 2, 4), nil
	})

	beforeSpan, err := parent.Before(lazySub)
	if err != nil {
		t.Fatal(err)
	}
	text, err := beforeSpan.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "ab" {
		t.Errorf("Before(lazySub) = %q, want %q", text, "ab")
	}
}
