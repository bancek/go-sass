package sasscommon

import "testing"

func TestSecondarySpansInsertionOrder(t *testing.T) {
	fs := testFileSource("abcdef")
	a := NewSimpleFileSpan(fs, 0, 1)
	b := NewSimpleFileSpan(fs, 1, 2)
	c := NewSimpleFileSpan(fs, 2, 3)

	spans := NewSecondarySpans(
		SecondarySpan{Span: a, Label: "first"},
		SecondarySpan{Span: b, Label: "second"},
	)
	spans.Put(c, "third")

	var got []string
	for span, label := range spans.Entries() {
		text, err := span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, text+":"+label)
	}
	want := []string{"a:first", "b:second", "c:third"}
	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSecondarySpansPutUpdatesInPlace(t *testing.T) {
	fs := testFileSource("abcdef")
	a := NewSimpleFileSpan(fs, 0, 1)
	b := NewSimpleFileSpan(fs, 1, 2)

	spans := NewSecondarySpans(
		SecondarySpan{Span: a, Label: "first"},
		SecondarySpan{Span: b, Label: "second"},
	)
	spans.Put(a, "updated")

	var labels []string
	for _, label := range spans.Entries() {
		labels = append(labels, label)
	}
	if len(labels) != 2 || labels[0] != "updated" || labels[1] != "second" {
		t.Errorf("labels = %v, want [updated second]", labels)
	}
}

func TestSecondarySpansCloneIsIndependent(t *testing.T) {
	fs := testFileSource("abc")
	a := NewSimpleFileSpan(fs, 0, 1)
	b := NewSimpleFileSpan(fs, 1, 2)

	original := NewSecondarySpans(SecondarySpan{Span: a, Label: "first"})
	clone := original.Clone()
	clone.Put(b, "second")

	if original.Len() != 1 {
		t.Errorf("original length = %d, want 1", original.Len())
	}
	if clone.Len() != 2 {
		t.Errorf("clone length = %d, want 2", clone.Len())
	}
}
