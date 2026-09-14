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
	if got != `\2d` {
		t.Errorf("ToCssIdentifier(-) = %q, want %q", got, `\2d`)
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
