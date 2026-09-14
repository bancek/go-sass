package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewUniversalSelector(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if s.Namespace != nil {
		t.Error("Namespace should be nil")
	}
}

func TestNewUniversalSelectorWithNamespace(t *testing.T) {
	ns := "svg"
	s := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	if s.Namespace == nil || *s.Namespace != "svg" {
		t.Error("Namespace should be 'svg'")
	}
}

func TestUniversalSelectorIsInvisible(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if s.IsInvisible() {
		t.Error("UniversalSelector should not be invisible")
	}
}

func TestUniversalSelectorIsBogus(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if s.IsBogus() {
		t.Error("UniversalSelector should not be bogus")
	}
}

func TestUniversalSelectorSpecificity(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if sp := s.Specificity(); sp != 0 {
		t.Errorf("Specificity = %d, want 0", sp)
	}
}

func TestUniversalSelectorHashCode(t *testing.T) {
	s1 := NewUniversalSelector(sasscommon.BogusSpan, nil)
	s2 := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same namespace should produce same hash code")
	}
}

func TestUniversalSelectorEquals(t *testing.T) {
	s1 := NewUniversalSelector(sasscommon.BogusSpan, nil)
	s2 := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if !s1.Equals(s2) {
		t.Error("Both nil namespace should be equal")
	}

	ns := "svg"
	s3 := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	if s1.Equals(s3) {
		t.Error("Different namespace should not be equal")
	}

	s4 := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	if !s3.Equals(s4) {
		t.Error("Same namespace should be equal")
	}
}

func TestUniversalSelectorIsSuperselectorWildcardNamespace(t *testing.T) {
	ns := "*"
	s := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	target := NewTypeSelector(NewQualifiedName("div"), sasscommon.BogusSpan)
	result, err := s.IsSuperselector(target)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("* namespace should be superselector of everything")
	}
}

func TestUniversalSelectorIsSuperselectorNilNamespace(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	target := NewClassSelector("foo", sasscommon.BogusSpan)
	result, err := s.IsSuperselector(target)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("nil namespace universal should be superselector in default namespace")
	}
}

func TestUniversalSelectorString(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "*"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestUniversalSelectorStringWithNamespace(t *testing.T) {
	ns := "svg"
	s := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "svg|*"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestUniversalSelectorAssertNotBogus(t *testing.T) {
	s := NewUniversalSelector(sasscommon.BogusSpan, nil)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestUniversalSelectorIsSuperselectorTypeNS(t *testing.T) {
	ns := "svg"
	s := NewUniversalSelector(sasscommon.BogusSpan, &ns)
	target := NewTypeSelector(NewQualifiedNameWithNamespace("circle", &ns), sasscommon.BogusSpan)
	result, err := s.IsSuperselector(target)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Same namespace universal should be superselector of TypeSelector")
	}
}
