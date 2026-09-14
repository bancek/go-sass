package value

import (
	"testing"
)

func TestHslSpaceMetadata(t *testing.T) {
	if HslColorSpace.Name() != "hsl" {
		t.Error()
	}
	if HslColorSpace.IsBounded() != true {
		t.Error()
	}
	if HslColorSpace.IsLegacy() != true {
		t.Error()
	}
	if HslColorSpace.IsPolar() != true {
		t.Error()
	}
}

func TestHslConvert(t *testing.T) {
	h, s, l := 120.0, 100.0, 50.0
	alpha := 1.0
	c, _ := HslColorSpace.Convert(RgbColorSpace, &h, &s, &l, &alpha)
	if got := ws(c.Channel0()); got != "0" {
		t.Errorf("hsl->Rgb ch0 = %s, want 0", got)
	}
	if got := ws(c.Channel1()); got != "255" {
		t.Errorf("hsl->Rgb ch1 = %s, want 255", got)
	}
	if got := ws(c.Channel2()); got != "0" {
		t.Errorf("hsl->Rgb ch2 = %s, want 0", got)
	}
	c2, _ := HslColorSpace.Convert(SrgbColorSpace, &h, &s, &l, &alpha)
	if got := ws(c2.Channel0()); got != "0" {
		t.Errorf("hsl->Srgb ch0 = %s, want 0", got)
	}
	if got := ws(c2.Channel1()); got != "1" {
		t.Errorf("hsl->Srgb ch1 = %s, want 1", got)
	}
	if got := ws(c2.Channel2()); got != "0" {
		t.Errorf("hsl->Srgb ch2 = %s, want 0", got)
	}
}
