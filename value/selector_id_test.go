package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewIDSelector(t *testing.T) {
	s := NewIDSelector("main", sasscommon.BogusSpan)
	if s.Name != "main" {
		t.Errorf("Name = %q, want main", s.Name)
	}
}

func TestIDSelectorIsInvisible(t *testing.T) {
	s := NewIDSelector("main", sasscommon.BogusSpan)
	if s.IsInvisible() {
		t.Error("IDSelector should not be invisible")
	}
}

func TestIDSelectorIsBogus(t *testing.T) {
	s := NewIDSelector("main", sasscommon.BogusSpan)
	if s.IsBogus() {
		t.Error("IDSelector should not be bogus")
	}
}

func TestIDSelectorSpecificity(t *testing.T) {
	s := NewIDSelector("main", sasscommon.BogusSpan)
	if sp := s.Specificity(); sp != 1000000 {
		t.Errorf("Specificity = %d, want 1000000", sp)
	}
}

func TestIDSelectorAddSuffix(t *testing.T) {
	s := NewIDSelector("main", sasscommon.BogusSpan)
	result, err := s.AddSuffix("-suffix")
	if err != nil {
		t.Fatal(err)
	}
	is, ok := result.(*IDSelector)
	if !ok {
		t.Fatalf("AddSuffix should return *IDSelector, got %T", result)
	}
	if is.Name != "main-suffix" {
		t.Errorf("Name = %q, want main-suffix", is.Name)
	}
}

func TestIDSelectorEquals(t *testing.T) {
	s1 := NewIDSelector("main", sasscommon.BogusSpan)
	s2 := NewIDSelector("main", sasscommon.BogusSpan)
	if !s1.Equals(s2) {
		t.Error("Same name should be equal")
	}
	s3 := NewIDSelector("other", sasscommon.BogusSpan)
	if s1.Equals(s3) {
		t.Error("Different names should not be equal")
	}
}

func TestIDSelectorHashCode(t *testing.T) {
	s1 := NewIDSelector("main", sasscommon.BogusSpan)
	s2 := NewIDSelector("main", sasscommon.BogusSpan)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same name should produce same hash code")
	}
}

func TestIDSelectorString(t *testing.T) {
	s := NewIDSelector("foo", sasscommon.BogusSpan)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "#foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestIDSelectorAssertNotBogus(t *testing.T) {
	s := NewIDSelector("foo", sasscommon.BogusSpan)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestIDSelectorIsSuperselector(t *testing.T) {
	s1 := NewIDSelector("main", sasscommon.BogusSpan)
	s2 := NewIDSelector("main", sasscommon.BogusSpan)
	result, err := s1.IsSuperselector(s2)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Same ID should be superselector")
	}

	s3 := NewIDSelector("other", sasscommon.BogusSpan)
	result, err = s1.IsSuperselector(s3)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("Different IDs should not be superselector")
	}
}
