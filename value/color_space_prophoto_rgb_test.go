package value

import (
	"testing"
)

func TestProphotoRgbSpaceMetadata(t *testing.T) {
	if ProphotoRgbColorSpace.Name() != "prophoto-rgb" {
		t.Error()
	}
	if !ProphotoRgbColorSpace.IsBounded() {
		t.Error()
	}
	if ProphotoRgbColorSpace.IsLegacy() {
		t.Error()
	}
}

func TestProphotoRgbConvert(t *testing.T) {
	r, g, b := 1.0, 0.0, 0.0
	alpha := 1.0
	c, _ := ProphotoRgbColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "60.6113461371" {
		t.Errorf("prophoto-rgb->Lab ch0 = %s, want 60.6113461371", got)
	}
	if got := ws(c.Channel1()); got != "139.1593742457" {
		t.Errorf("prophoto-rgb->Lab ch1 = %s, want 139.1593742457", got)
	}
	if got := ws(c.Channel2()); got != "104.502320926" {
		t.Errorf("prophoto-rgb->Lab ch2 = %s, want 104.502320926", got)
	}
	c2, _ := ProphotoRgbColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "1.3632928088" {
		t.Errorf("prophoto-rgb->Srgb ch0 = %s, want 1.3632928088", got)
	}
	if got := ws(c2.Channel1()); got != "-0.5156626969" {
		t.Errorf("prophoto-rgb->Srgb ch1 = %s, want -0.5156626969", got)
	}
	if got := ws(c2.Channel2()); got != "-0.090130414" {
		t.Errorf("prophoto-rgb->Srgb ch2 = %s, want -0.090130414", got)
	}
}
