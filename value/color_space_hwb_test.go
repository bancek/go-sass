package value

import (
	"testing"
)

func TestHwbSpaceMetadata(t *testing.T) {
	if HwbColorSpace.Name() != "hwb" {
		t.Error()
	}
	if HwbColorSpace.IsBounded() != true {
		t.Error()
	}
	if HwbColorSpace.IsLegacy() != true {
		t.Error()
	}
	if HwbColorSpace.IsPolar() != true {
		t.Error()
	}
}

func TestHwbConvert(t *testing.T) {
	h, w, bl := 240.0, 0.0, 0.0
	alpha := 1.0
	c, _ := HwbColorSpace.Convert(RgbColorSpace, &h, &w, &bl, &alpha)
	if got := ws(c.Channel0()); got != "0" {
		t.Errorf("hwb->Rgb ch0 = %s, want 0", got)
	}
	if got := ws(c.Channel1()); got != "0" {
		t.Errorf("hwb->Rgb ch1 = %s, want 0", got)
	}
	if got := ws(c.Channel2()); got != "255" {
		t.Errorf("hwb->Rgb ch2 = %s, want 255", got)
	}
	c2, _ := HwbColorSpace.Convert(SrgbColorSpace, &h, &w, &bl, &alpha)
	if got := ws(c2.Channel0()); got != "0" {
		t.Errorf("hwb->Srgb ch0 = %s, want 0", got)
	}
	if got := ws(c2.Channel1()); got != "0" {
		t.Errorf("hwb->Srgb ch1 = %s, want 0", got)
	}
	if got := ws(c2.Channel2()); got != "1" {
		t.Errorf("hwb->Srgb ch2 = %s, want 1", got)
	}
}
