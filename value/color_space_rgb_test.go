package value

import (
	"testing"
)

func TestRgbSpaceMetadata(t *testing.T) {
	if RgbColorSpace.Name() != "rgb" {
		t.Error()
	}
	if RgbColorSpace.IsBounded() != true {
		t.Error()
	}
	if RgbColorSpace.IsLegacy() != true {
		t.Error()
	}
	if RgbColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestRgbConvert(t *testing.T) {
	r, g, b := 1.0, 0.5, 0.0
	alpha := 1.0
	c, _ := RgbColorSpace.Convert(HslColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "30" {
		t.Errorf("rgb->Hsl ch0 = %s, want 30", got)
	}
	if got := ws(c.Channel1()); got != "100" {
		t.Errorf("rgb->Hsl ch1 = %s, want 100", got)
	}
	if got := ws(c.Channel2()); got != "0.1960784314" {
		t.Errorf("rgb->Hsl ch2 = %s, want 0.1960784314", got)
	}
	c2, _ := RgbColorSpace.Convert(HwbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "30" {
		t.Errorf("rgb->Hwb ch0 = %s, want 30", got)
	}
	if got := ws(c2.Channel1()); got != "0" {
		t.Errorf("rgb->Hwb ch1 = %s, want 0", got)
	}
	if got := ws(c2.Channel2()); got != "99.6078431373" {
		t.Errorf("rgb->Hwb ch2 = %s, want 99.6078431373", got)
	}
}
