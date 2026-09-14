package value

import (
	"testing"
)

func TestDisplayP3SpaceMetadata(t *testing.T) {
	if DisplayP3ColorSpace.Name() != "display-p3" {
		t.Errorf("Name = %q", DisplayP3ColorSpace.Name())
	}
	if !DisplayP3ColorSpace.IsBounded() {
		t.Error("should be bounded")
	}
	if DisplayP3ColorSpace.IsLegacy() {
		t.Error("should not be legacy")
	}
	if DisplayP3ColorSpace.IsPolar() {
		t.Error("should not be polar")
	}
}

func TestDisplayP3Convert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := DisplayP3ColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "56.2077729169" {
		t.Errorf("display-p3->Lab ch0 = %s, want 56.2077729169", got)
	}
	if got := ws(c.Channel1()); got != "94.464418467" {
		t.Errorf("display-p3->Lab ch1 = %s, want 94.464418467", got)
	}
	if got := ws(c.Channel2()); got != "98.8921195438" {
		t.Errorf("display-p3->Lab ch2 = %s, want 98.8921195438", got)
	}
	c2, _ := DisplayP3ColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "1.0930663624" {
		t.Errorf("display-p3->Srgb ch0 = %s, want 1.0930663624", got)
	}
	if got := ws(c2.Channel1()); got != "-0.2267419736" {
		t.Errorf("display-p3->Srgb ch1 = %s, want -0.2267419736", got)
	}
	if got := ws(c2.Channel2()); got != "-0.1501345809" {
		t.Errorf("display-p3->Srgb ch2 = %s, want -0.1501345809", got)
	}
}
