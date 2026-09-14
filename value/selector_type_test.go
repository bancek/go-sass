package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewTypeSelector(t *testing.T) {
	name := NewQualifiedName("div")
	s := NewTypeSelector(name, sasscommon.BogusSpan)
	if s.Name.Name != "div" {
		t.Errorf("Name = %q, want div", s.Name.Name)
	}
}

func TestTypeSelectorIsInvisible(t *testing.T) {
	s := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	if s.IsInvisible() {
		t.Error("TypeSelector should not be invisible")
	}
}

func TestTypeSelectorIsBogus(t *testing.T) {
	s := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	if s.IsBogus() {
		t.Error("TypeSelector should not be bogus")
	}
}

func TestTypeSelectorSpecificity(t *testing.T) {
	s := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	if sp := s.Specificity(); sp != 1 {
		t.Errorf("Specificity = %d, want 1", sp)
	}
}

func TestTypeSelectorAddSuffix(t *testing.T) {
	s := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := s.AddSuffix("bar")
	if err != nil {
		t.Fatal(err)
	}
	ts, ok := result.(*TypeSelector)
	if !ok {
		t.Fatalf("AddSuffix should return *TypeSelector, got %T", result)
	}
	if ts.Name.Name != "divbar" {
		t.Errorf("Name = %q, want divbar", ts.Name.Name)
	}
}

func TestTypeSelectorEquals(t *testing.T) {
	s1 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	s2 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	if !s1.Equals(s2) {
		t.Error("Same name should be equal")
	}
	s3 := NewTypeSelector(NewQualifiedName("span"), sasscommon.BogusSpan)
	if s1.Equals(s3) {
		t.Error("Different names should not be equal")
	}
}

func TestTypeSelectorHashCode(t *testing.T) {
	s1 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	s2 := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same name should produce same hash code")
	}
}

func TestTypeSelectorString(t *testing.T) {
	qn := QualifiedName{Name: "div"}
	s := NewTypeSelector(qn, sasscommon.BogusSpan)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "div"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestTypeSelectorAssertNotBogus(t *testing.T) {
	qn := QualifiedName{Name: "div"}
	s := NewTypeSelector(qn, sasscommon.BogusSpan)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestTypeSelectorIsSuperselectorWildcardNS(t *testing.T) {
	ns := "*"
	s := NewTypeSelector(NewQualifiedNameWithNamespace("div", &ns), sasscommon.BogusSpan)
	target := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := s.IsSuperselector(target)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("TypeSelector with * namespace should be superselector of TypeSelector with no namespace")
	}
}
