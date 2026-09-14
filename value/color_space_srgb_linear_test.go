package value

import (
	"testing"
)

func TestSrgbLinearSpaceMetadata(t *testing.T) {
	if SrgbLinearColorSpace.Name() != "srgb-linear" {
		t.Errorf("Name = %q", SrgbLinearColorSpace.Name())
	}
	if !SrgbLinearColorSpace.IsBounded() {
		t.Error("should be bounded")
	}
	if SrgbLinearColorSpace.IsLegacy() {
		t.Error("should not be legacy")
	}
	if SrgbLinearColorSpace.IsPolar() {
		t.Error("should not be polar")
	}
}

func TestSrgbLinearConvert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := SrgbLinearColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "1" {
		t.Errorf("srgb-linear->Srgb ch0 = %s, want 1", got)
	}
	if got := ws(c.Channel1()); got != "0" {
		t.Errorf("srgb-linear->Srgb ch1 = %s, want 0", got)
	}
	if got := ws(c.Channel2()); got != "0" {
		t.Errorf("srgb-linear->Srgb ch2 = %s, want 0", got)
	}
	c2, _ := SrgbLinearColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "54.2905414047" {
		t.Errorf("srgb-linear->Lab ch0 = %s, want 54.2905414047", got)
	}
	if got := ws(c2.Channel1()); got != "80.8049281704" {
		t.Errorf("srgb-linear->Lab ch1 = %s, want 80.8049281704", got)
	}
	if got := ws(c2.Channel2()); got != "69.8909647686" {
		t.Errorf("srgb-linear->Lab ch2 = %s, want 69.8909647686", got)
	}
}
