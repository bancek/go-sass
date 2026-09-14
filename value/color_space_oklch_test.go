package value

import (
	"testing"
)

func TestOklchSpaceMetadata(t *testing.T) {
	if OklchColorSpace.Name() != "oklch" {
		t.Error()
	}
	if OklchColorSpace.IsBounded() != false {
		t.Error()
	}
	if OklchColorSpace.IsLegacy() != false {
		t.Error()
	}
	if OklchColorSpace.IsPolar() != true {
		t.Error()
	}
}

func TestOklchConvert(t *testing.T) {
	l, c, h := 0.5, 0.1, 45.0
	alpha := 1.0
	r, _ := OklchColorSpace.Convert(SrgbColorSpace, &l, &c, &h, &alpha)
	if got := ws(r.Channel0()); got != "0.569700204" {
		t.Errorf("oklch->Srgb ch0 = %s, want 0.569700204", got)
	}
	if got := ws(r.Channel1()); got != "0.3088484941" {
		t.Errorf("oklch->Srgb ch1 = %s, want 0.3088484941", got)
	}
	if got := ws(r.Channel2()); got != "0.1856030772" {
		t.Errorf("oklch->Srgb ch2 = %s, want 0.1856030772", got)
	}
	r2, _ := OklchColorSpace.Convert(LabColorSpace, &l, &c, &h, &alpha)
	if got := ws(r2.Channel0()); got != "41.3278676545" {
		t.Errorf("oklch->Lab ch0 = %s, want 41.3278676545", got)
	}
	if got := ws(r2.Channel1()); got != "26.5144151584" {
		t.Errorf("oklch->Lab ch1 = %s, want 26.5144151584", got)
	}
	if got := ws(r2.Channel2()); got != "31.0977049205" {
		t.Errorf("oklch->Lab ch2 = %s, want 31.0977049205", got)
	}
}
