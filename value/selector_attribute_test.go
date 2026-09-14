package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewAttributeSelector(t *testing.T) {
	name := NewQualifiedName("href")
	s := NewAttributeSelector(name, sasscommon.BogusSpan)
	if s.Name.Name != "href" {
		t.Errorf("Name = %q, want href", s.Name.Name)
	}
	if s.Op != nil {
		t.Error("Op should be nil")
	}
	if s.Value != nil {
		t.Error("Value should be nil")
	}
}

func TestNewAttributeSelectorWithOperator(t *testing.T) {
	name := NewQualifiedName("class")
	s := NewAttributeSelectorWithOperator(name, AttributeOperatorEqual, "foo", sasscommon.BogusSpan, nil)
	if *s.Op != AttributeOperatorEqual {
		t.Error("Op should be Equal")
	}
	if *s.Value != "foo" {
		t.Errorf("Value = %q, want foo", *s.Value)
	}
}

func TestAttributeOperatorString(t *testing.T) {
	tests := []struct {
		op   AttributeOperator
		want string
	}{
		{AttributeOperatorEqual, "="},
		{AttributeOperatorInclude, "~="},
		{AttributeOperatorDash, "|="},
		{AttributeOperatorPrefix, "^="},
		{AttributeOperatorSuffix, "$="},
		{AttributeOperatorSubstring, "*="},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("AttributeOperator(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestAttributeSelectorIsInvisible(t *testing.T) {
	s := NewAttributeSelector(NewQualifiedName("href"), sasscommon.BogusSpan)
	if s.IsInvisible() {
		t.Error("AttributeSelector should not be invisible")
	}
}

func TestAttributeSelectorIsBogus(t *testing.T) {
	s := NewAttributeSelector(NewQualifiedName("href"), sasscommon.BogusSpan)
	if s.IsBogus() {
		t.Error("AttributeSelector should not be bogus")
	}
}

func TestAttributeSelectorSpecificity(t *testing.T) {
	s := NewAttributeSelector(NewQualifiedName("href"), sasscommon.BogusSpan)
	if sp := s.Specificity(); sp != 1000 {
		t.Errorf("Specificity = %d, want 1000", sp)
	}
}

func TestAttributeSelectorAddSuffix(t *testing.T) {
	s := NewAttributeSelector(NewQualifiedName("href"), sasscommon.BogusSpan)
	_, err := s.AddSuffix("x")
	if err == nil {
		t.Error("AttributeSelector.AddSuffix should error")
	}
}

func TestAttributeSelectorEquals(t *testing.T) {
	name := NewQualifiedName("href")
	s1 := NewAttributeSelector(name, sasscommon.BogusSpan)
	s2 := NewAttributeSelector(name, sasscommon.BogusSpan)
	if !s1.Equals(s2) {
		t.Error("Same name, no op: should be equal")
	}

	s3 := NewAttributeSelectorWithOperator(name, AttributeOperatorEqual, "x", sasscommon.BogusSpan, nil)
	if s1.Equals(s3) {
		t.Error("Different op: should not be equal")
	}

	s4 := NewAttributeSelectorWithOperator(name, AttributeOperatorEqual, "x", sasscommon.BogusSpan, nil)
	if !s3.Equals(s4) {
		t.Error("Same op+value: should be equal")
	}
}

func TestAttributeSelectorString(t *testing.T) {
	name := NewQualifiedName("href")
	s := NewAttributeSelector(name, sasscommon.BogusSpan)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "[href]"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestAttributeSelectorStringWithValue(t *testing.T) {
	name := NewQualifiedName("class")
	s := NewAttributeSelectorWithOperator(name, AttributeOperatorEqual, "foo", sasscommon.BogusSpan, nil)
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "[class=foo]"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestAttributeSelectorAssertNotBogus(t *testing.T) {
	s := NewAttributeSelector(NewQualifiedName("href"), sasscommon.BogusSpan)
	if err := s.AssertNotBogus(nil, nil); err != nil {
		t.Errorf("AssertNotBogus() unexpected error: %v", err)
	}
}

func TestAttributeSelectorHashCode(t *testing.T) {
	name := NewQualifiedName("href")
	s1 := NewAttributeSelector(name, sasscommon.BogusSpan)
	s2 := NewAttributeSelector(name, sasscommon.BogusSpan)
	if s1.HashCode() != s2.HashCode() {
		t.Error("Same fields should produce same hash code")
	}
}
