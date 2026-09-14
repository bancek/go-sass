package value

import "testing"

func TestHueInterpolationMethodStrings(t *testing.T) {
	if HueInterpolationShorter.String() != "shorter" {
		t.Error()
	}
	if HueInterpolationLonger.String() != "longer" {
		t.Error()
	}
	if HueInterpolationIncreasing.String() != "increasing" {
		t.Error()
	}
	if HueInterpolationDecreasing.String() != "decreasing" {
		t.Error()
	}
}

func TestNewInterpolationMethodPolar(t *testing.T) {
	method, err := NewInterpolationMethod(HslColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if method.Space != HslColorSpace {
		t.Error("space mismatch")
	}
	if method.Hue == nil || *method.Hue != HueInterpolationShorter {
		t.Error("polar spaces should default to shorter hue method")
	}
}

func TestNewInterpolationMethodRectangular(t *testing.T) {
	method, err := NewInterpolationMethod(SrgbColorSpace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if method.Space != SrgbColorSpace {
		t.Error("space mismatch")
	}
	if method.Hue != nil {
		t.Error("rectangular space should not have hue method")
	}
}

func TestNewInterpolationMethodRectangularWithHue(t *testing.T) {
	hue := HueInterpolationLonger
	_, err := NewInterpolationMethod(SrgbColorSpace, &hue)
	if err == nil {
		t.Error("rectangular space with hue method should error")
	}
}
