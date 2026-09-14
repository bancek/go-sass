package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewClassSelector(t *testing.T) {
	span := sasscommon.BogusSpan
	s := NewClassSelector("foo", span)
	if s.Name != "foo" {
		t.Errorf("Name = %q, want foo", s.Name)
	}
	got, _ := s.Span()
	if got != span {
		t.Error("Span should match constructor arg")
	}
}

func TestClassSelectorIsInvisible(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	if s.IsInvisible() {
		t.Error("ClassSelector should not be invisible")
	}
}

func TestClassSelectorIsBogus(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	if s.IsBogus() {
		t.Error("ClassSelector should not be bogus")
	}
}

func TestClassSelectorIsUseless(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	if s.IsUseless() {
		t.Error("ClassSelector should not be useless")
	}
}

func TestClassSelectorContainsParentSelector(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	got, err := s.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ClassSelector should not contain parent selector")
	}
}

func TestClassSelectorHasComplicatedSuperselectorSemantics(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	if s.HasComplicatedSuperselectorSemantics() {
		t.Error("ClassSelector should not have complicated superselector semantics")
	}
}

func TestClassSelectorSpecificity(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	if sp := s.Specificity(); sp != 1000 {
		t.Errorf("Specificity = %d, want 1000", sp)
	}
}

func TestClassSelectorAddSuffix(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	result, err := s.AddSuffix("bar")
	if err != nil {
		t.Fatal(err)
	}
	cs, ok := result.(*ClassSelector)
	if !ok {
		t.Fatalf("AddSuffix should return *ClassSelector, got %T", result)
	}
	if cs.Name != "foobar" {
		t.Errorf("Name = %q, want foobar", cs.Name)
	}
}

func TestClassSelectorEquals(t *testing.T) {
	s1 := NewClassSelector("foo", sasscommon.BogusSpan)
	s2 := NewClassSelector("foo", sasscommon.BogusSpan)
	if !s1.Equals(s2) {
		t.Error("Same name should be equal")
	}

	s3 := NewClassSelector("bar", sasscommon.BogusSpan)
	if s1.Equals(s3) {
		t.Error("Different names should not be equal")
	}
}

func TestClassSelectorHashCode(t *testing.T) {
	s1 := NewClassSelector("foo", sasscommon.BogusSpan)
	s2 := NewClassSelector("foo", sasscommon.BogusSpan)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same name should produce same hash code")
	}
}

func TestClassSelectorIsSuperselector(t *testing.T) {
	s1 := NewClassSelector("foo", sasscommon.BogusSpan)
	s2 := NewClassSelector("foo", sasscommon.BogusSpan)
	result, err := s1.IsSuperselector(s2)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Same class should be superselector")
	}

	s3 := NewClassSelector("bar", sasscommon.BogusSpan)
	result, err = s1.IsSuperselector(s3)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("Different classes should not be superselector")
	}
}

func TestClassSelectorString(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ".foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestClassSelectorAssertNotBogus(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)
	name := "test"
	if err := s.AssertNotBogus(&name, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus(nil) unexpected error: %v", err)
	}
}

func TestClassSelectorUnify(t *testing.T) {
	s := NewClassSelector("foo", sasscommon.BogusSpan)

	result, err := s.Unify(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("Unify with nil: len = %d, want 1", len(result))
	}
	cs, ok := result[0].(*ClassSelector)
	if !ok {
		t.Fatalf("Wanted *ClassSelector, got %T", result[0])
	}
	if cs.Name != "foo" {
		t.Errorf("Name = %q, want foo", cs.Name)
	}
}
