package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewParentSelector(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	if s.Suffix != nil {
		t.Error("Suffix should be nil")
	}
}

func TestParentSelectorWithSuffix(t *testing.T) {
	suffix := "suffix"
	s := NewParentSelector(sasscommon.BogusSpan, &suffix)
	if s.Suffix == nil || *s.Suffix != "suffix" {
		t.Error("Suffix should be 'suffix'")
	}
}

func TestParentSelectorContainsParentSelector(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	got, err := s.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("ParentSelector should contain parent selector")
	}
}

func TestParentSelectorIsInvisible(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	if s.IsInvisible() {
		t.Error("ParentSelector should not be invisible")
	}
}

func TestParentSelectorIsBogus(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	if s.IsBogus() {
		t.Error("ParentSelector should not be bogus")
	}
}

func TestParentSelectorSpecificity(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	if sp := s.Specificity(); sp != 1000 {
		t.Errorf("Specificity = %d, want 1000", sp)
	}
}

func TestParentSelectorAddSuffix(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	_, err := s.AddSuffix("x")
	if err == nil {
		t.Error("ParentSelector.AddSuffix should error")
	}
}

func TestParentSelectorEquals(t *testing.T) {
	s1 := NewParentSelector(sasscommon.BogusSpan, nil)
	s2 := NewParentSelector(sasscommon.BogusSpan, nil)
	if !s1.Equals(s2) {
		t.Error("Both nil suffix should be equal")
	}

	suffix := "x"
	s3 := NewParentSelector(sasscommon.BogusSpan, &suffix)
	if s1.Equals(s3) {
		t.Error("Different suffix should not be equal")
	}

	s4 := NewParentSelector(sasscommon.BogusSpan, &suffix)
	if !s3.Equals(s4) {
		t.Error("Same suffix should be equal")
	}
}

func TestParentSelectorString(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "&"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParentSelectorStringWithSuffix(t *testing.T) {
	suffix := "my-suffix"
	s := NewParentSelector(sasscommon.BogusSpan, &suffix)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "&my-suffix"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParentSelectorAssertNotBogus(t *testing.T) {
	s := NewParentSelector(sasscommon.BogusSpan, nil)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestParentSelectorHashCode(t *testing.T) {
	s1 := NewParentSelector(sasscommon.BogusSpan, nil)
	s2 := NewParentSelector(sasscommon.BogusSpan, nil)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same suffix should produce same hash code")
	}
}
