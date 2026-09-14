package value

import (
	"testing"
)

func TestXyzD65SpaceMetadata(t *testing.T) {
	if XyzD65ColorSpace.Name() != "xyz" {
		t.Errorf("Name = %q, want xyz", XyzD65ColorSpace.Name())
	}
	if XyzD65ColorSpace.IsBounded() {
		t.Error("should not be bounded")
	}
	if XyzD65ColorSpace.IsLegacy() {
		t.Error("should not be legacy")
	}
}

func TestXyzD65Convert(t *testing.T) {
	r, g, b := 0.5, 0.5, 0.5
	alpha := 1.0
	c, _ := XyzD65ColorSpace.Convert(LabColorSpace, &r, &g, &b, &alpha)
	if got := ws(c.Channel0()); got != "76.1608841835" {
		t.Errorf("xyz-d65->Lab ch0 = %s, want 76.1608841835", got)
	}
	if got := ws(c.Channel1()); got != "7.1944893389" {
		t.Errorf("xyz-d65->Lab ch1 = %s, want 7.1944893389", got)
	}
	if got := ws(c.Channel2()); got != "4.6048603909" {
		t.Errorf("xyz-d65->Lab ch2 = %s, want 4.6048603909", got)
	}
	c2, _ := XyzD65ColorSpace.Convert(SrgbColorSpace, &r, &g, &b, &alpha)
	if got := ws(c2.Channel0()); got != "0.7992092975" {
		t.Errorf("xyz-d65->Srgb ch0 = %s, want 0.7992092975", got)
	}
	if got := ws(c2.Channel1()); got != "0.7180602368" {
		t.Errorf("xyz-d65->Srgb ch1 = %s, want 0.7180602368", got)
	}
	if got := ws(c2.Channel2()); got != "0.7044225805" {
		t.Errorf("xyz-d65->Srgb ch2 = %s, want 0.7044225805", got)
	}
}
