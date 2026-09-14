package value

import (
	"testing"
)

func TestToSpaceIdentity(t *testing.T) {
	c, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	result, err := c.ToSpace(SrgbColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c != result {
		t.Error("ToSpace to same space should return original")
	}
}

func TestToSpaceConversion(t *testing.T) {
	c, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	result, err := c.ToSpace(LabColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Space() != LabColorSpace {
		t.Errorf("space = %v, want Lab", result.Space().Name())
	}
	if got := ws(result.Channel0()); got != "53.3889647411" {
		t.Errorf("lightness = %s, want 53.3889647411", got)
	}
	if got := ws(result.Channel1()); got != "0" {
		t.Errorf("a = %s, want 0", got)
	}
	if got := ws(result.Channel2()); got != "0" {
		t.Errorf("b = %s, want 0", got)
	}
	if got := ws(result.Alpha()); got != "1" {
		t.Errorf("alpha = %s, want 1", got)
	}
}

func TestToSpaceMissingChannels(t *testing.T) {
	r := 255.0
	b := 0.0
	ac := 1.0
	c, _ := NewColorRGBInternal(&r, nil, &b, &ac, nil)
	if !c.IsChannel1Missing() {
		t.Fatal("expected channel1 to be missing")
	}
	converted, err := c.ToSpace(LabColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if converted.Space() != LabColorSpace {
		t.Error("wrong destination space")
	}
	if got := ws(converted.Channel0()); got != "54.2905414047" {
		t.Errorf("lightness = %s, want 54.2905414047", got)
	}
	if got := ws(converted.Channel1()); got != "80.8049281704" {
		t.Errorf("a = %s, want 80.8049281704", got)
	}
	if got := ws(converted.Channel2()); got != "69.8909647686" {
		t.Errorf("b = %s, want 69.8909647686", got)
	}
}

func TestToSpaceLegacyMissing(t *testing.T) {
	r := 255.0
	b := 0.0
	ac := 1.0
	c, _ := NewColorRGBInternal(&r, nil, &b, &ac, nil)
	lm := false
	converted, err := c.ToSpace(HslColorSpace, &lm)
	if err != nil {
		t.Fatal(err)
	}
	if converted.Space() != HslColorSpace {
		t.Error("should be HSL")
	}
	if got := ws(converted.Channel0()); got != "0" {
		t.Errorf("hue = %s, want 0", got)
	}
	if got := ws(converted.Channel1()); got != "100" {
		t.Errorf("saturation = %s, want 100", got)
	}
	if got := ws(converted.Channel2()); got != "50" {
		t.Errorf("lightness = %s, want 50", got)
	}
}

func TestToSpaceLegacyMissingTrue(t *testing.T) {
	r := 255.0
	b := 0.0
	ac := 1.0
	c, _ := NewColorRGBInternal(&r, nil, &b, &ac, nil)
	lm := true
	converted, err := c.ToSpace(RgbColorSpace, &lm)
	if err != nil {
		t.Fatal(err)
	}
	if !converted.IsChannel1Missing() {
		t.Error("legacyMissing=true should preserve missing channels")
	}
}

func TestToRGB(t *testing.T) {
	c, _ := NewColorRGB(255, 128, 0, 1)
	r, g, b, a, err := c.ToRGB()
	if err != nil {
		t.Fatal(err)
	}
	if ws(r) != "255" {
		t.Errorf("r = %s, want 255", ws(r))
	}
	if ws(g) != "128" {
		t.Errorf("g = %s, want 128", ws(g))
	}
	if ws(b) != "0" {
		t.Errorf("b = %s, want 0", ws(b))
	}
	if ws(a) != "1" {
		t.Errorf("a = %s, want 1", ws(a))
	}
}

func TestToHSL(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	h, s, l, a, err := c.ToHSL()
	if err != nil {
		t.Fatal(err)
	}
	if ws(h) != "0" {
		t.Errorf("h = %s, want 0", ws(h))
	}
	if ws(s) != "100" {
		t.Errorf("s = %s, want 100", ws(s))
	}
	if ws(l) != "50" {
		t.Errorf("l = %s, want 50", ws(l))
	}
	if ws(a) != "1" {
		t.Errorf("a = %s, want 1", ws(a))
	}
}

func TestToHWB(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	h, w, bl, a, err := c.ToHWB()
	if err != nil {
		t.Fatal(err)
	}
	if ws(h) != "0" {
		t.Errorf("h = %s, want 0", ws(h))
	}
	if ws(w) != "0" {
		t.Errorf("w = %s, want 0", ws(w))
	}
	if ws(bl) != "0" {
		t.Errorf("b = %s, want 0", ws(bl))
	}
	if ws(a) != "1" {
		t.Errorf("a = %s, want 1", ws(a))
	}
}
