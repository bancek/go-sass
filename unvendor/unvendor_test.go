package unvendor

import "testing"

func TestUnvendorNoPrefix(t *testing.T) {
	result := Unvendor("keyframes")
	if result != "keyframes" {
		t.Errorf("Unvendor(\"keyframes\") = %q, want %q", result, "keyframes")
	}
}

func TestUnvendorWebkit(t *testing.T) {
	result := Unvendor("-webkit-keyframes")
	if result != "keyframes" {
		t.Errorf("Unvendor(\"-webkit-keyframes\") = %q, want %q", result, "keyframes")
	}
}

func TestUnvendorMoz(t *testing.T) {
	result := Unvendor("-moz-foo")
	if result != "foo" {
		t.Errorf("Unvendor(\"-moz-foo\") = %q, want %q", result, "foo")
	}
}

func TestUnvendorDoubleDash(t *testing.T) {
	result := Unvendor("--custom")
	if result != "--custom" {
		t.Errorf("Unvendor(\"--custom\") = %q, want %q", result, "--custom")
	}
}

func TestUnvendorNoSecondDash(t *testing.T) {
	result := Unvendor("-x")
	if result != "-x" {
		t.Errorf("Unvendor(\"-x\") = %q, want %q", result, "-x")
	}
}

func TestUnvendorEmpty(t *testing.T) {
	result := Unvendor("")
	if result != "" {
		t.Errorf("Unvendor(\"\") = %q, want %q", result, "")
	}
}
