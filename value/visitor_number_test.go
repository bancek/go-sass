package value

import "testing"

func TestIsUnitAmbiguousWithExponent(t *testing.T) {
	tests := []struct {
		unit string
		want bool
	}{
		{"", false},
		{"e", false},
		{"e1", true},
		{"e-1", true},
		{"E1", true},
		{"E-1", true},
		{"e-", false},
		{"e-x", false},
		{"ee", false},
		{"px", false},
	}

	for _, tt := range tests {
		if got := isUnitAmbiguousWithExponent(tt.unit); got != tt.want {
			t.Errorf("isUnitAmbiguousWithExponent(%q) = %v, want %v", tt.unit, got, tt.want)
		}
	}
}

// A number whose unit is the empty string is only reachable through the
// JavaScript/Dart APIs and the embedded protocol. It serializes as the bare
// number instead of being treated as an exponent-like unit.
//
// Regression test for sass/dart-sass#2889.
func TestSerializeNumberWithEmptyUnit(t *testing.T) {
	unit := ""
	number := NewSassNumber(1, &unit)

	css, err := SerializeValue(number, false)
	if err != nil {
		t.Fatal(err)
	}
	if css != "1" {
		t.Errorf("SerializeValue(1 with empty unit) = %q, want %q", css, "1")
	}
}

func TestSerializeNumberWithExponentLikeUnit(t *testing.T) {
	number := NewSassNumber(1, new("e1"))

	css, err := SerializeValue(number, false)
	if err != nil {
		t.Fatal(err)
	}
	if css != `1\65 1` {
		t.Errorf("SerializeValue(1e1) = %q, want %q", css, `1\65 1`)
	}
}
