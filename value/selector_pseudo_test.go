package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewPseudoSelectorClass(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s.Name != "hover" {
		t.Errorf("Name = %q, want hover", s.Name)
	}
	if !s.IsClass {
		t.Error("hover should be a class")
	}
	if s.IsElement() {
		t.Error("hover should not be element")
	}
	if s.NormalizedName != "hover" {
		t.Errorf("NormalizedName = %q, want hover", s.NormalizedName)
	}
}

func TestNewPseudoSelectorElement(t *testing.T) {
	s := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	if s.IsClass {
		t.Error("before should not be class")
	}
	if !s.IsElement() {
		t.Error("before should be element")
	}
}

func TestIsFakePseudoElement(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"after", true},
		{"After", true},
		{"before", true},
		{"first-line", true},
		{"first-letter", true},
		{"hover", false},
		{"before-stuff", false},
	}
	for _, tt := range tests {
		if got := isFakePseudoElement(tt.name); got != tt.want {
			t.Errorf("isFakePseudoElement(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPseudoSelectorSpecificityElement(t *testing.T) {
	s := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	if sp := s.Specificity(); sp != 1 {
		t.Errorf("Specificity = %d, want 1", sp)
	}
}

func TestPseudoSelectorSpecificityClass(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if sp := s.Specificity(); sp != 1000 {
		t.Errorf("Specificity = %d, want 1000", sp)
	}
}

func TestPseudoSelectorIsInvisible(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s.IsInvisible() {
		t.Error("hover without selector should not be invisible")
	}
}

func TestPseudoSelectorIsBogus(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s.IsBogus() {
		t.Error("hover without selector should not be bogus")
	}
}

func TestPseudoSelectorIsUseless(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s.IsUseless() {
		t.Error("hover should not be useless")
	}
}

func TestPseudoSelectorIsHost(t *testing.T) {
	s := NewPseudoSelector("host", sasscommon.BogusSpan, false, nil, nil)
	if !s.IsHost() {
		t.Error("host should be detected via IsHost()")
	}
}

func TestPseudoSelectorAddSuffixSimple(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	result, err := s.AddSuffix("-x")
	if err != nil {
		t.Fatal(err)
	}
	ps, ok := result.(*PseudoSelector)
	if !ok {
		t.Fatalf("AddSuffix should return *PseudoSelector, got %T", result)
	}
	if ps.Name != "hover-x" {
		t.Errorf("Name = %q, want hover-x", ps.Name)
	}
}

func TestPseudoSelectorHashCode(t *testing.T) {
	s1 := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	s2 := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same fields should produce same hash code")
	}
}

func TestPseudoSelectorEquals(t *testing.T) {
	s1 := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	s2 := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if !s1.Equals(s2) {
		t.Error("Same name/class should be equal")
	}

	s3 := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	if s1.Equals(s3) {
		t.Error("Different name/element should not be equal")
	}
}

func TestPseudoSelectorHasComplicatedSuperselectorSemantics(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if s.HasComplicatedSuperselectorSemantics() {
		t.Error("hover without selector should not have complicated semantics")
	}

	s2 := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	if !s2.HasComplicatedSuperselectorSemantics() {
		t.Error("pseudo-element should have complicated semantics")
	}
}

func TestPseudoSelectorStringClass(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ":hover"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPseudoSelectorStringElement(t *testing.T) {
	s := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "::before"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPseudoSelectorStringWithArg(t *testing.T) {
	arg := "2n+1"
	s := NewPseudoSelector("nth-child", sasscommon.BogusSpan, false, &arg, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ":nth-child(2n+1)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPseudoSelectorAssertNotBogus(t *testing.T) {
	s := NewPseudoSelector("hover", sasscommon.BogusSpan, false, nil, nil)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestPseudoSelectorWithSelector(t *testing.T) {
	comp, _ := NewCompoundSelector([]SimpleSelector{NewClassSelector("foo", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	complex, _ := NewComplexSelector(nil, []*ComplexSelectorComponent{NewComplexSelectorComponent(comp, nil, sasscommon.BogusSpan)}, sasscommon.BogusSpan, false)
	list, _ := NewSelectorList([]*ComplexSelector{complex}, sasscommon.BogusSpan)
	s := NewPseudoSelector("is", sasscommon.BogusSpan, false, nil, list)
	result, err := s.WithSelector(list)
	if err != nil {
		t.Fatal(err)
	}
	if result.Selector == nil {
		t.Error("WithSelector should set selector")
	}
}
