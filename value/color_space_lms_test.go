package value

import (
	"testing"
)

func TestLmsSpaceMetadata(t *testing.T) {
	if LmsColorSpace.Name() != "lms" {
		t.Error()
	}
	if LmsColorSpace.IsBounded() != false {
		t.Error()
	}
	if LmsColorSpace.IsLegacy() != false {
		t.Error()
	}
	if LmsColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestLmsConvert(t *testing.T) {
	c0, c1, c2 := 0.5, 0.3, 0.2
	alpha := 1.0
	r, _ := LmsColorSpace.Convert(SrgbColorSpace, &c0, &c1, &c2, &alpha)
	if got := ws(r.Channel0()); got != "1.0395111833" {
		t.Errorf("lms->Srgb ch0 = %s, want 1.0395111833", got)
	}
	if got := ws(r.Channel1()); got != "0.3141551416" {
		t.Errorf("lms->Srgb ch1 = %s, want 0.3141551416", got)
	}
	if got := ws(r.Channel2()); got != "0.3935597021" {
		t.Errorf("lms->Srgb ch2 = %s, want 0.3935597021", got)
	}
	r2, _ := LmsColorSpace.Convert(LabColorSpace, &c0, &c1, &c2, &alpha)
	if got := ws(r2.Channel0()); got != "62.3783913344" {
		t.Errorf("lms->Lab ch0 = %s, want 62.3783913344", got)
	}
	if got := ws(r2.Channel1()); got != "70.6051985009" {
		t.Errorf("lms->Lab ch1 = %s, want 70.6051985009", got)
	}
	if got := ws(r2.Channel2()); got != "31.527589847" {
		t.Errorf("lms->Lab ch2 = %s, want 31.527589847", got)
	}
}
