package value

import (
	"testing"
)

func TestDisplayP3LinearSpaceMetadata(t *testing.T) {
	if DisplayP3LinearColorSpace.Name() != "display-p3-linear" {
		t.Error()
	}
	if DisplayP3LinearColorSpace.IsBounded() != true {
		t.Error()
	}
	if DisplayP3LinearColorSpace.IsLegacy() != false {
		t.Error()
	}
	if DisplayP3LinearColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestDisplayP3LinearConvert(t *testing.T) {
	r, g, b := 1.0, 0.5, 0.0
	alpha := 1.0
	c, _ := DisplayP3LinearColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "1.0479079546" {
		t.Errorf("p3-linear->Srgb ch0 = %s, want 1.0479079546", got)
	}
	if got := ws(c.Channel1()); got != "0.7213332105" {
		t.Errorf("p3-linear->Srgb ch1 = %s, want 0.7213332105", got)
	}
	if got := ws(c.Channel2()); got != "-0.2693180848" {
		t.Errorf("p3-linear->Srgb ch2 = %s, want -0.2693180848", got)
	}
	c2, _ := DisplayP3LinearColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "81.1435969949" {
		t.Errorf("p3-linear->Lab ch0 = %s, want 81.1435969949", got)
	}
	if got := ws(c2.Channel1()); got != "22.1709804559" {
		t.Errorf("p3-linear->Lab ch1 = %s, want 22.1709804559", got)
	}
	if got := ws(c2.Channel2()); got != "109.7151144136" {
		t.Errorf("p3-linear->Lab ch2 = %s, want 109.7151144136", got)
	}
}
