package util

import (
	"strings"
	"testing"
)

func TestToCssIdentifierSimple(t *testing.T) {
	got, err := ToCssIdentifier("foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("ToCssIdentifier(foo) = %q, want %q", got, "foo")
	}
}

func TestToCssIdentifierLeadingDash(t *testing.T) {
	got, err := ToCssIdentifier("-foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "-foo" {
		t.Errorf("ToCssIdentifier(-foo) = %q, want %q", got, "-foo")
	}
}

func TestToCssIdentifierDoubleDash(t *testing.T) {
	got, err := ToCssIdentifier("--foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "--foo" {
		t.Errorf("ToCssIdentifier(--foo) = %q, want %q", got, "--foo")
	}
}

func TestToCssIdentifierDashAlone(t *testing.T) {
	got, err := ToCssIdentifier("-")
	if err != nil {
		t.Fatal(err)
	}
	if got != `\2d ` {
		t.Errorf("ToCssIdentifier(-) = %q, want %q", got, `\2d `)
	}
}

func TestToCssIdentifierEmpty(t *testing.T) {
	_, err := ToCssIdentifier("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestToCssIdentifierStartsWithDigit(t *testing.T) {
	got, err := ToCssIdentifier("1foo")
	if err != nil {
		t.Fatal(err)
	}
	// Should escape the leading digit
	if !strings.Contains(got, "\\") || !strings.Contains(got, "31") {
		t.Errorf("ToCssIdentifier(1foo) = %q, expected escaped form", got)
	}
}

func TestToCssIdentifierUnderscore(t *testing.T) {
	got, err := ToCssIdentifier("_foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "_foo" {
		t.Errorf("ToCssIdentifier(_foo) = %q, want %q", got, "_foo")
	}
}

func TestToCssIdentifierBody(t *testing.T) {
	got, err := ToCssIdentifierBody("1foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1foo" {
		t.Errorf("ToCssIdentifierBody(1foo) = %q, want %q", got, "1foo")
	}
}

func TestToCssIdentifierEscapes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.bar", `foo\.bar`},
		{"\n", `\a `},
		{"é", "é"},
		{"\uE000", `\e000 `},
	}
	for _, tt := range tests {
		got, err := ToCssIdentifier(tt.input)
		if err != nil {
			t.Errorf("ToCssIdentifier(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ToCssIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsSimpleIdentifier(t *testing.T) {
	if !IsSimpleIdentifier("foo-bar") {
		t.Error("foo-bar should be simple")
	}
	if IsSimpleIdentifier("foo.bar") {
		t.Error("foo.bar should not be simple")
	}
	if IsSimpleIdentifier("\uE000") {
		t.Error("private-use BMP should not be simple")
	}
}

func TestToCssIdentifierSupplementaryPrivateUse(t *testing.T) {
	got, err := ToCssIdentifier("\U000F0000")
	if err != nil {
		t.Fatal(err)
	}
	if got != `\f0000 ` {
		t.Errorf("ToCssIdentifier(U+F0000) = %q, want %q", got, `\f0000 `)
	}
}

func TestTryCssIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"foo", "foo", true},
		{"--foo", "--foo", true},
		{"", "", false},
		{"\x00", "", false},
	}

	for _, tt := range tests {
		got, ok := TryCssIdentifier(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("TryCssIdentifier(%q) = (%q, %v), want (%q, %v)",
				tt.input, got, ok, tt.want, tt.ok)
		}
	}
}
