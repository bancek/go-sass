package value

import (
	"testing"
)

func TestXyzD50SpaceMetadata(t *testing.T) {
	if XyzD50ColorSpace.Name() != "xyz-d50" {
		t.Error()
	}
	if XyzD50ColorSpace.IsBounded() != false {
		t.Error()
	}
	if XyzD50ColorSpace.IsLegacy() != false {
		t.Error()
	}
	if XyzD50ColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestXyzD50Convert(t *testing.T) {
	r, g, b := 0.5, 0.5, 0.5
	alpha := 1.0
	c, _ := XyzD50ColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "76.0692610142" {
		t.Errorf("xyz-d50->Lab ch0 = %s, want 76.0692610142", got)
	}
	if got := ws(c.Channel1()); got != "4.8387310772" {
		t.Errorf("xyz-d50->Lab ch1 = %s, want 4.8387310772", got)
	}
	if got := ws(c.Channel2()); got != "-10.505341671" {
		t.Errorf("xyz-d50->Lab ch2 = %s, want -10.505341671", got)
	}
	c2, _ := XyzD50ColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "0.7438835606" {
		t.Errorf("xyz-d50->Srgb ch0 = %s, want 0.7438835606", got)
	}
	if got := ws(c2.Channel1()); got != "0.7256918895" {
		t.Errorf("xyz-d50->Srgb ch1 = %s, want 0.7256918895", got)
	}
	if got := ws(c2.Channel2()); got != "0.811893154" {
		t.Errorf("xyz-d50->Srgb ch2 = %s, want 0.811893154", got)
	}
}
