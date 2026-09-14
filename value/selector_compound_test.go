package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewCompoundSelector(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, err := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(c.Components))
	}
}

func TestNewCompoundSelectorEmpty(t *testing.T) {
	_, err := NewCompoundSelector([]SimpleSelector{}, sasscommon.BogusSpan)
	if err == nil {
		t.Error("Expected error for empty components")
	}
}

func TestCompoundSelectorIsInvisible(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if c.IsInvisible() {
		t.Error("CompoundSelector with ClassSelector should not be invisible")
	}

	ph := NewPlaceholderSelector("foo", sasscommon.BogusSpan)
	c2, _ := NewCompoundSelector([]SimpleSelector{ph}, sasscommon.BogusSpan)
	if !c2.IsInvisible() {
		t.Error("CompoundSelector with PlaceholderSelector should be invisible")
	}
}

func TestCompoundSelectorIsBogus(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if c.IsBogus() {
		t.Error("CompoundSelector with ClassSelector should not be bogus")
	}
}

func TestCompoundSelectorIsUseless(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if c.IsUseless() {
		t.Error("CompoundSelector should not be useless")
	}
}

func TestCompoundSelectorContainsParentSelector(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	got, err := c.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Should not contain parent selector")
	}

	parent := NewParentSelector(sasscommon.BogusSpan, nil)
	c2, _ := NewCompoundSelector([]SimpleSelector{parent}, sasscommon.BogusSpan)
	got, err = c2.ContainsParentSelector()
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Should contain parent selector")
	}
}

func TestCompoundSelectorSingleSimple(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	s := c.SingleSimple()
	if s == nil {
		t.Fatal("Expected non-nil for single component")
	}
	if _, ok := s.(*ClassSelector); !ok {
		t.Error("Expected ClassSelector")
	}

	c2, _ := NewCompoundSelector([]SimpleSelector{class, NewIDSelector("bar", sasscommon.BogusSpan)}, sasscommon.BogusSpan)
	if s := c2.SingleSimple(); s != nil {
		t.Error("Expected nil for multiple components")
	}
}

func TestCompoundSelectorSpecificity(t *testing.T) {
	c1 := NewClassSelector("foo", sasscommon.BogusSpan)
	c2 := NewIDSelector("bar", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{c1, c2}, sasscommon.BogusSpan)
	if sp := c.Specificity(); sp != 1001000 {
		t.Errorf("Specificity = %d, want 1001000", sp)
	}
}

func TestCompoundSelectorHasComplicatedSuperselectorSemantics(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if c.HasComplicatedSuperselectorSemantics() {
		t.Error("ClassSelector compound should not have complicated semantics")
	}

	pe := NewPseudoSelector("before", sasscommon.BogusSpan, true, nil, nil)
	c2, _ := NewCompoundSelector([]SimpleSelector{pe}, sasscommon.BogusSpan)
	if !c2.HasComplicatedSuperselectorSemantics() {
		t.Error("Pseudo-element compound should have complicated semantics")
	}
}

func TestCompoundSelectorString(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	got, err := c.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ".foo"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestCompoundSelectorStringMulti(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	id := NewIDSelector("bar", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class, id}, sasscommon.BogusSpan)
	got, err := c.String()
	if err != nil {
		t.Fatal(err)
	}
	want := ".foo#bar"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestCompoundSelectorAssertNotBogus(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if err := c.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestCompoundSelectorHashCode(t *testing.T) {
	class := NewClassSelector("foo", sasscommon.BogusSpan)
	c1, _ := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	class2 := NewClassSelector("foo", sasscommon.BogusSpan)
	c2, _ := NewCompoundSelector([]SimpleSelector{class2}, sasscommon.BogusSpan)
	if c1.HashCode() != c2.HashCode() {
		t.Error("Same components should produce same hash")
	}
}
