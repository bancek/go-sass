package value

import (
	"strings"
	"testing"
)

func TestStringProps(t *testing.T) {
	s := &SassString{Text: "hello", HasQuotes: true}
	if s.Text != "hello" {
		t.Error("Text should be stored")
	}
	if !s.HasQuotes {
		t.Error("HasQuotes should be true")
	}
	if !s.IsTruthy() {
		t.Error("string IsTruthy should be true")
	}
	u := &SassString{Text: "hello", HasQuotes: false}
	if !u.IsTruthy() {
		t.Error("unquoted string IsTruthy should be true")
	}
}

func TestStringIsBlank(t *testing.T) {
	unq := &SassString{Text: "", HasQuotes: false}
	if !unq.IsBlank() {
		t.Error("unquoted empty string should be blank")
	}
	q := &SassString{Text: "", HasQuotes: true}
	if q.IsBlank() {
		t.Error("quoted empty string should not be blank")
	}
	unq2 := &SassString{Text: " ", HasQuotes: false}
	if unq2.IsBlank() {
		t.Error("unquoted non-empty string should not be blank")
	}
}

func TestStringEquals(t *testing.T) {
	a := &SassString{Text: "hello", HasQuotes: true}
	b := &SassString{Text: "hello", HasQuotes: false}
	if !a.Equals(b) {
		t.Error("strings with same text but different quotes should be equal")
	}
	c := &SassString{Text: "world", HasQuotes: true}
	if a.Equals(c) {
		t.Error("strings with different text should not be equal")
	}
	if a.Equals(Null) {
		t.Error("string should not equal null")
	}
}

func TestSassStringHashCode(t *testing.T) {
	a := &SassString{Text: "hello", HasQuotes: true}
	b := &SassString{Text: "hello", HasQuotes: false}
	if a.HashCode() != b.HashCode() {
		t.Error("HashCode should ignore quotes")
	}
	c := &SassString{Text: "world", HasQuotes: true}
	if a.HashCode() == c.HashCode() {
		t.Error("different text should produce different hash")
	}
	h1 := a.HashCode()
	h2 := a.HashCode()
	if h1 != h2 {
		t.Error("HashCode should be cached and deterministic")
	}
}

func TestStringSassLength(t *testing.T) {
	s := &SassString{Text: "abc"}
	if s.SassLength() != 3 {
		t.Errorf("SassLength of 'abc' = %d, want 3", s.SassLength())
	}
	s2 := &SassString{Text: "é"}
	if s2.SassLength() != 1 {
		t.Errorf("SassLength of 'é' = %d, want 1", s2.SassLength())
	}
}

func TestStringPlus(t *testing.T) {
	a := &SassString{Text: "hello", HasQuotes: true}
	b := &SassString{Text: " world", HasQuotes: true}
	result, err := a.Plus(b)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("Plus of two strings should return a string")
	}
	if s.Text != "hello world" {
		t.Errorf("got %q, want %q", s.Text, "hello world")
	}
	if !s.HasQuotes {
		t.Error("should preserve HasQuotes of left string")
	}

	num := NewUnitlessNumber(42)
	result, err = a.Plus(num)
	if err != nil {
		t.Fatal(err)
	}
	s2, ok := result.(*SassString)
	if !ok {
		t.Fatal("Plus with number should return a string")
	}
	if !strings.Contains(s2.Text, "42") {
		t.Errorf("result should contain '42', got %q", s2.Text)
	}
}

func TestStringIsSpecialNumber(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"calc(1px + 2px)", true},
		{"attr(data-val)", true},
		{"clamp(0, 1, 2)", true},
		{"env(SAFE)", true},
		{"if(true, 1, 2)", true},
		{"min(1, 2)", true},
		{"max(1, 2)", true},
		{"var(--x)", true},
		{"normal", false},
		{"ab", false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			s := &SassString{Text: tt.text, HasQuotes: false}
			if got := s.IsSpecialNumber(); got != tt.expected {
				t.Errorf("IsSpecialNumber(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}

	quoted := &SassString{Text: "calc(1px + 2px)", HasQuotes: true}
	if quoted.IsSpecialNumber() {
		t.Error("quoted calc should not be special number")
	}
}

func TestStringIsSpecialVariable(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"var(--x)", true},
		{"attr(data-val)", true},
		{"if(true, 1, 2)", true},
		{"calc(1px)", false},
		{"normal", false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			s := &SassString{Text: tt.text, HasQuotes: false}
			if got := s.IsSpecialVariable(); got != tt.expected {
				t.Errorf("IsSpecialVariable(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
	quoted := &SassString{Text: "var(--x)", HasQuotes: true}
	if quoted.IsSpecialVariable() {
		t.Error("quoted var should not be special variable")
	}
}

func TestStringAssertQuoted(t *testing.T) {
	q := &SassString{Text: "hello", HasQuotes: true}
	if err := q.AssertQuoted(); err != nil {
		t.Errorf("quoted string should not error: %v", err)
	}
	uq := &SassString{Text: "hello", HasQuotes: false}
	if err := uq.AssertQuoted(); err == nil {
		t.Error("unquoted string should error on AssertQuoted")
	}
}

func TestStringAssertUnquoted(t *testing.T) {
	q := &SassString{Text: "hello", HasQuotes: true}
	if err := q.AssertUnquoted(); err == nil {
		t.Error("quoted string should error on AssertUnquoted")
	}
	uq := &SassString{Text: "hello", HasQuotes: false}
	if err := uq.AssertUnquoted(); err != nil {
		t.Errorf("unquoted string should not error: %v", err)
	}
}

func TestEmptySassString(t *testing.T) {
	q := EmptySassString(nil)
	if q.Text != "" || !q.HasQuotes {
		t.Error("EmptySassString(nil) should be quoted empty")
	}
	f := false
	uq := EmptySassString(&f)
	if uq.Text != "" || uq.HasQuotes {
		t.Error("EmptySassString(&false) should be unquoted empty")
	}
	f2 := false
	uq2 := EmptySassString(&f2)
	if uq != uq2 {
		t.Error("EmptySassString should return singletons for same args")
	}
}

func TestStringToCssString_Quoted(t *testing.T) {
	s := &SassString{Text: "hello", HasQuotes: true}
	got, err := s.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != `"hello"` {
		t.Errorf("quoted string ToCssString(true) = %q, want %q", got, `"hello"`)
	}
	got, err = s.ToCssString(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("quoted string ToCssString(false) = %q, want %q", got, "hello")
	}
}

func TestStringToCssString_Unquoted(t *testing.T) {
	s := &SassString{Text: "hello", HasQuotes: false}
	got, err := s.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("unquoted string ToCssString(true) = %q, want %q", got, "hello")
	}
	got, err = s.ToCssString(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("unquoted string ToCssString(false) = %q, want %q", got, "hello")
	}
}

func TestStringToCssString_Escape(t *testing.T) {
	s := &SassString{Text: "a\\b", HasQuotes: true}
	got, err := s.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\"") {
		t.Errorf("quoted string should be wrapped in quotes, got %q", got)
	}
}

func TestStringString(t *testing.T) {
	s := &SassString{Text: "hello", HasQuotes: true}
	got, err := s.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `"hello"` {
		t.Errorf("String() = %q, want %q", got, `"hello"`)
	}

	u := &SassString{Text: "hello", HasQuotes: false}
	got, err = u.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("unquoted String() = %q, want %q", got, "hello")
	}
}

func TestStringOperators(t *testing.T) {
	s := &SassString{Text: "hello", HasQuotes: true}
	_, err := s.Minus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error for strings")
	}
}

func TestSassStringPlusNonString(t *testing.T) {
	s := &SassString{Text: "a", HasQuotes: false}
	result, err := s.Plus(SassTrue)
	if err != nil {
		t.Fatal(err)
	}
	ss, ok := result.(*SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", result)
	}
	want := "atrue"
	if ss.Text != want {
		t.Errorf("got %q, want %q", ss.Text, want)
	}
	if ss.HasQuotes {
		t.Error("should preserve HasQuotes of left string (false)")
	}
}
