package value

import (
	"testing"
)

func TestSrgbSpaceMetadata(t *testing.T) {
	if SrgbColorSpace.Name() != "srgb" {
		t.Error()
	}
	if SrgbColorSpace.IsBounded() != true {
		t.Error()
	}
	if SrgbColorSpace.IsLegacy() != false {
		t.Error()
	}
	if SrgbColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestSrgbConvert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := SrgbColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "54.2905414047" {
		t.Errorf("srgb->Lab ch0 = %s, want 54.2905414047", got)
	}
	if got := ws(c.Channel1()); got != "80.8049281704" {
		t.Errorf("srgb->Lab ch1 = %s, want 80.8049281704", got)
	}
	if got := ws(c.Channel2()); got != "69.8909647686" {
		t.Errorf("srgb->Lab ch2 = %s, want 69.8909647686", got)
	}
	c2, _ := SrgbColorSpace.Convert(HslColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "0" {
		t.Errorf("srgb->Hsl ch0 = %s, want 0", got)
	}
	if got := ws(c2.Channel1()); got != "100" {
		t.Errorf("srgb->Hsl ch1 = %s, want 100", got)
	}
	if got := ws(c2.Channel2()); got != "50" {
		t.Errorf("srgb->Hsl ch2 = %s, want 50", got)
	}
}
