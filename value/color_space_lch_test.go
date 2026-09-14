package value

import (
	"testing"
)

func TestLchSpaceMetadata(t *testing.T) {
	if LchColorSpace.Name() != "lch" {
		t.Error()
	}
	if LchColorSpace.IsBounded() != false {
		t.Error()
	}
	if LchColorSpace.IsLegacy() != false {
		t.Error()
	}
	if LchColorSpace.IsPolar() != true {
		t.Error()
	}
}

func TestLchConvert(t *testing.T) {
	l, c, h := 50.0, 10.0, 45.0
	alpha := 1.0
	r, _ := LchColorSpace.Convert(LabColorSpace, &l, &c, &h, &alpha)
	if got := ws(r.Channel0()); got != "50" {
		t.Errorf("lch->Lab ch0 = %s, want 50", got)
	}
	if got := ws(r.Channel1()); got != "7.0710678119" {
		t.Errorf("lch->Lab ch1 = %s, want 7.0710678119", got)
	}
	if got := ws(r.Channel2()); got != "7.0710678119" {
		t.Errorf("lch->Lab ch2 = %s, want 7.0710678119", got)
	}
	r2, _ := LchColorSpace.Convert(SrgbColorSpace, &l, &c, &h, &alpha)
	if got := ws(r2.Channel0()); got != "0.5269006242" {
		t.Errorf("lch->Srgb ch0 = %s, want 0.5269006242", got)
	}
	if got := ws(r2.Channel1()); got != "0.4492146127" {
		t.Errorf("lch->Srgb ch1 = %s, want 0.4492146127", got)
	}
	if got := ws(r2.Channel2()); got != "0.4206062974" {
		t.Errorf("lch->Srgb ch2 = %s, want 0.4206062974", got)
	}
}
