package value

import (
	"testing"
)

func TestA98RgbSpaceMetadata(t *testing.T) {
	if A98RgbColorSpace.Name() != "a98-rgb" {
		t.Errorf("Name = %q", A98RgbColorSpace.Name())
	}
	if !A98RgbColorSpace.IsBounded() {
		t.Error("should be bounded")
	}
	if A98RgbColorSpace.IsLegacy() {
		t.Error("should not be legacy")
	}
	if A98RgbColorSpace.IsPolar() {
		t.Error("should not be polar")
	}
}

func TestA98RgbConvert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := A98RgbColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "62.6024552479" {
		t.Errorf("a98-rgb->Lab ch0 = %s, want 62.6024552479", got)
	}
	if got := ws(c.Channel1()); got != "90.3601768232" {
		t.Errorf("a98-rgb->Lab ch1 = %s, want 90.3601768232", got)
	}
	if got := ws(c.Channel2()); got != "78.1556283488" {
		t.Errorf("a98-rgb->Lab ch2 = %s, want 78.1556283488", got)
	}
	c2, _ := A98RgbColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "1.1581834834" {
		t.Errorf("a98-rgb->Srgb ch0 = %s, want 1.1581834834", got)
	}
	if got := ws(c2.Channel1()); got != "0" {
		t.Errorf("a98-rgb->Srgb ch1 = %s, want 0", got)
	}
	if got := ws(c2.Channel2()); got != "0" {
		t.Errorf("a98-rgb->Srgb ch2 = %s, want 0", got)
	}
}
