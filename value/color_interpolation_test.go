package value

import (
	"testing"
)

func TestInterpolateColorsIdentity(t *testing.T) {
	c1, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	c2, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	method, _ := NewInterpolationMethod(SrgbColorSpace, nil)
	w := 0.5
	result, err := c1.Interpolate(c2, method, false, &w)
	if err != nil {
		t.Fatal(err)
	}
	if result.Space() != SrgbColorSpace {
		t.Error("should stay in sRGB")
	}
	if got := ws(result.Channel0()); got != "0.5" {
		t.Errorf("ch0 = %s, want 0.5", got)
	}
	if got := ws(result.Channel1()); got != "0.5" {
		t.Errorf("ch1 = %s, want 0.5", got)
	}
	if got := ws(result.Channel2()); got != "0.5" {
		t.Errorf("ch2 = %s, want 0.5", got)
	}
	if got := ws(result.Alpha()); got != "1" {
		t.Errorf("alpha = %s, want 1", got)
	}
}

func TestInterpolateColorsWeight0(t *testing.T) {
	c1, _ := NewColorSRGB(1, 0, 0, 1)
	c2, _ := NewColorSRGB(0, 0, 1, 1)
	method, _ := NewInterpolationMethod(SrgbColorSpace, nil)
	w := 0.0
	result, err := c1.Interpolate(c2, method, false, &w)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(result.Channel0()); got != "0" {
		t.Errorf("ch0 = %s, want 0", got)
	}
	if got := ws(result.Channel1()); got != "0" {
		t.Errorf("ch1 = %s, want 0", got)
	}
	if got := ws(result.Channel2()); got != "1" {
		t.Errorf("ch2 = %s, want 1", got)
	}
}

func TestInterpolateColorsWeight1(t *testing.T) {
	c1, _ := NewColorSRGB(1, 0, 0, 1)
	c2, _ := NewColorSRGB(0, 0, 1, 1)
	method, _ := NewInterpolationMethod(SrgbColorSpace, nil)
	w := 1.0
	result, err := c1.Interpolate(c2, method, false, &w)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(result.Channel0()); got != "1" {
		t.Errorf("ch0 = %s, want 1", got)
	}
	if got := ws(result.Channel1()); got != "0" {
		t.Errorf("ch1 = %s, want 0", got)
	}
	if got := ws(result.Channel2()); got != "0" {
		t.Errorf("ch2 = %s, want 0", got)
	}
}

// Missing channels take the other color's value; missing on both sides stays
// missing (#2810: conversions propagate analogous missingness, so
// interpolation only checks direct missingness).
func TestInterpolateMissingChannels(t *testing.T) {
	present, err := NewColorForSpace(LabColorSpace, [3]float64{60, 10, 20}, 1, [4]bool{})
	if err != nil {
		t.Fatal(err)
	}
	missing, err := NewColorForSpace(LabColorSpace, [3]float64{50, 0, 0}, 1, [4]bool{false, true, true, false})
	if err != nil {
		t.Fatal(err)
	}
	method, err := NewInterpolationMethod(LabColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := 0.5
	result, err := missing.Interpolate(present, method, false, &w)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsChannel1Missing() || result.IsChannel2Missing() {
		t.Error("channels should take the present color's values")
	}

	both, err := missing.Interpolate(missing, method, false, &w)
	if err != nil {
		t.Fatal(err)
	}
	if !both.IsChannel1Missing() || !both.IsChannel2Missing() {
		t.Error("channels missing on both sides should stay missing")
	}
}
