package value

import (
	"testing"
)

func TestRec2020SpaceMetadata(t *testing.T) {
	if Rec2020ColorSpace.Name() != "rec2020" {
		t.Error()
	}
	if !Rec2020ColorSpace.IsBounded() {
		t.Error()
	}
}

func TestRec2020Convert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := Rec2020ColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "59.8036299926" {
		t.Errorf("rec2020->Lab ch0 = %s, want 59.8036299926", got)
	}
	if got := ws(c.Channel1()); got != "116.8849865694" {
		t.Errorf("rec2020->Lab ch1 = %s, want 116.8849865694", got)
	}
	if got := ws(c.Channel2()); got != "106.7572157253" {
		t.Errorf("rec2020->Lab ch2 = %s, want 106.7572157253", got)
	}
	c2, _ := Rec2020ColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "1.2482198282" {
		t.Errorf("rec2020->Srgb ch0 = %s, want 1.2482198282", got)
	}
	if got := ws(c2.Channel1()); got != "-0.3879075029" {
		t.Errorf("rec2020->Srgb ch1 = %s, want -0.3879075029", got)
	}
	if got := ws(c2.Channel2()); got != "-0.1435143925" {
		t.Errorf("rec2020->Srgb ch2 = %s, want -0.1435143925", got)
	}
}
