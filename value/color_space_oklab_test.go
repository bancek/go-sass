package value

import (
	"testing"
)

func TestOklabSpaceMetadata(t *testing.T) {
	if OklabColorSpace.Name() != "oklab" {
		t.Error()
	}
	if OklabColorSpace.IsBounded() != false {
		t.Error()
	}
	if OklabColorSpace.IsLegacy() != false {
		t.Error()
	}
	if OklabColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestOklabConvert(t *testing.T) {
	l, a, b := 0.5, 0.0, 0.0
	alpha := 1.0
	c, _ := OklabColorSpace.Convert(SrgbColorSpace, &l, &a, &b, &alpha)
	if got := ws(c.Channel0()); got != "0.388572859" {
		t.Errorf("oklab->Srgb ch0 = %s, want 0.388572859", got)
	}
	if got := ws(c.Channel1()); got != "0.388572859" {
		t.Errorf("oklab->Srgb ch1 = %s, want 0.388572859", got)
	}
	if got := ws(c.Channel2()); got != "0.388572859" {
		t.Errorf("oklab->Srgb ch2 = %s, want 0.388572859", got)
	}
	c2, _ := OklabColorSpace.Convert(LabColorSpace, &l, &a, &b, &alpha)
	if got := ws(c2.Channel0()); got != "42" {
		t.Errorf("oklab->Lab ch0 = %s, want 42", got)
	}
	if got := ws(c2.Channel1()); got != "0" {
		t.Errorf("oklab->Lab ch1 = %s, want 0", got)
	}
	if got := ws(c2.Channel2()); got != "0" {
		t.Errorf("oklab->Lab ch2 = %s, want 0", got)
	}
}
