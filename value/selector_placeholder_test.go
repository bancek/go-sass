package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewPlaceholderSelector(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if s.Name != "foo" {
		t.Errorf("Name = %q, want foo", s.Name)
	}
}

func TestPlaceholderSelectorIsInvisible(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if !s.IsInvisible() {
		t.Error("PlaceholderSelector should be invisible")
	}
}

func TestPlaceholderSelectorIsBogus(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if s.IsBogus() {
		t.Error("PlaceholderSelector should not be bogus")
	}
}

func TestPlaceholderSelectorIsPrivate(t *testing.T) {
	s1 := NewPlaceholderSelector("-foo", sasscommon.BogusSpan)
	if !s1.IsPrivate() {
		t.Error("Name starting with - should be private")
	}
	s2 := NewPlaceholderSelector("_bar", sasscommon.BogusSpan)
	if !s2.IsPrivate() {
		t.Error("Name starting with _ should be private")
	}
	s3 := NewPlaceholderSelector("baz", sasscommon.BogusSpan)
	if s3.IsPrivate() {
		t.Error("Name starting with letter should not be private")
	}
}

func TestPlaceholderSelectorSpecificity(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if sp := s.Specificity(); sp != 1000 {
		t.Errorf("Specificity = %d, want 1000", sp)
	}
}

func TestPlaceholderSelectorAddSuffix(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	result, err := s.AddSuffix("bar")
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := result.(*PlaceholderSelector)
	if !ok {
		t.Fatalf("AddSuffix should return *PlaceholderSelector, got %T", result)
	}
	if ps.Name != "foobar" {
		t.Errorf("Name = %q, want foobar", ps.Name)
	}
}

func TestPlaceholderSelectorEquals(t *testing.T) {
	s1 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	s2 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if !s1.Equals(s2) {
		t.Error("Same name should be equal")
	}
	s3 := NewPlaceholderSelector("bar", sasscommon.BogusSpan)
	if s1.Equals(s3) {
		t.Error("Different names should not be equal")
	}
}

func TestPlaceholderSelectorHashCode(t *testing.T) {
	s1 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	s2 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same name should produce same hash code")
	}
}

func TestPlaceholderSelectorString(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "%foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPlaceholderSelectorAssertNotBogus(t *testing.T) {
	s := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestPlaceholderSelectorIsSuperselector(t *testing.T) {
	s1 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	s2 := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	result, err := s1.IsSuperselector(s2)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Same placeholder should be superselector")
	}
}
