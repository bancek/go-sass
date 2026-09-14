package value

import (
	"math"
	"strings"
	"testing"
)

func TestNewColorRGB(t *testing.T) {
	c, err := NewColorRGB(255, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Space() != RgbColorSpace {
		t.Error("should be RGB space")
	}
	if c.Channel0() != 255 || c.Channel1() != 0 || c.Channel2() != 0 {
		t.Errorf("channels = [%v, %v, %v], want [255, 0, 0]", c.Channel0(), c.Channel1(), c.Channel2())
	}
	if c.Alpha() != 1 {
		t.Errorf("alpha = %v, want 1", c.Alpha())
	}

	_, err = NewColorRGB(255, 0, 0, 1.5)
	if err == nil {
		t.Error("alpha > 1 should error")
	}
}

func TestNewColorRGBInternal(t *testing.T) {
	r := 128.0
	g := 64.0
	b := 0.0
	a := 0.5
	c, err := NewColorRGBInternal(&r, &g, &b, &a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Channel0() != 128 || c.Alpha() != 0.5 {
		t.Error("channels should be stored")
	}
	if c.IsChannel0Missing() || c.IsChannel1Missing() || c.IsChannel2Missing() || c.IsAlphaMissing() {
		t.Error("no channels should be missing")
	}

	r2 := 255.0
	c2, err := NewColorRGBInternal(&r2, nil, &b, &a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c2.IsChannel1Missing() {
		t.Error("channel1 should be missing (nil passed)")
	}
}

func TestNewColorHSL(t *testing.T) {
	c, err := NewColorHSL(0, 100, 50, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Space() != HslColorSpace {
		t.Error("should be HSL space")
	}
	if c.Channel0() != 0 {
		t.Errorf("hue = %v, want 0 (normalized)", c.Channel0())
	}
	if c.Channel1() != 100 {
		t.Errorf("saturation = %v, want 100", c.Channel1())
	}
	if c.Channel2() != 50 {
		t.Errorf("lightness = %v, want 50", c.Channel2())
	}

	_, err = NewColorHSL(0, 100, 50, 1.5)
	if err == nil {
		t.Error("alpha > 1 should error")
	}

	c2, err := NewColorHSL(0, -10, 50, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Channel1() < 0 {
		t.Error("negative saturation should be normalized to positive")
	}
}

func TestNewColorHWB(t *testing.T) {
	c, err := NewColorHWB(120, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Space() != HwbColorSpace {
		t.Error("should be HWB space")
	}
	if c.Channel0() != 120 {
		t.Errorf("hue = %v, want 120", c.Channel0())
	}

	_, err = NewColorHWB(0, 0, 0, 2)
	if err == nil {
		t.Error("alpha > 1 should error")
	}
}

func TestNewColorSRGB(t *testing.T) {
	c, err := NewColorSRGB(0.5, 0.5, 0.5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Space() != SrgbColorSpace {
		t.Error("should be sRGB space")
	}
	if c.Space().IsLegacy() {
		t.Error("sRGB (SrgbColorSpace) is not a legacy space")
	}
}

func TestNewColorModern(t *testing.T) {
	tests := []struct {
		name  string
		space ColorSpace
		fn    func() (*SassColor, error)
	}{
		{"SRGB Linear", SrgbLinearColorSpace, func() (*SassColor, error) { return NewColorSRGBLinear(0.5, 0.5, 0.5, 1) }},
		{"Display P3", DisplayP3ColorSpace, func() (*SassColor, error) { return NewColorDisplayP3(0.5, 0.5, 0.5, 1) }},
		{"Display P3 Linear", DisplayP3LinearColorSpace, func() (*SassColor, error) { return NewColorDisplayP3Linear(0.5, 0.5, 0.5, 1) }},
		{"A98 RGB", A98RgbColorSpace, func() (*SassColor, error) { return NewColorA98RGB(0.5, 0.5, 0.5, 1) }},
		{"ProPhoto RGB", ProphotoRgbColorSpace, func() (*SassColor, error) { return NewColorProPhotoRGB(0.5, 0.5, 0.5, 1) }},
		{"Rec2020", Rec2020ColorSpace, func() (*SassColor, error) { return NewColorRec2020(0.5, 0.5, 0.5, 1) }},
		{"XYZ D50", XyzD50ColorSpace, func() (*SassColor, error) { return NewColorXYZD50(0.5, 0.5, 0.5, 1) }},
		{"XYZ D65", XyzD65ColorSpace, func() (*SassColor, error) { return NewColorXYZD65(0.5, 0.5, 0.5, 1) }},
		{"Lab", LabColorSpace, func() (*SassColor, error) { return NewColorLab(50, 25, -25, 1) }},
		{"LCH", LchColorSpace, func() (*SassColor, error) { return NewColorLCH(50, 30, 200, 1) }},
		{"OKLab", OklabColorSpace, func() (*SassColor, error) { return NewColorOKLab(0.5, 0.1, -0.1, 1) }},
		{"OKLCH", OklchColorSpace, func() (*SassColor, error) { return NewColorOKLCH(0.5, 0.15, 180, 1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := tt.fn()
			if err != nil {
				t.Fatal(err)
			}
			if c.Space() != tt.space {
				t.Errorf("space = %v, want %v", c.Space().Name(), tt.space.Name())
			}
		})
	}
}

func TestNewColorForSpace(t *testing.T) {
	c, err := NewColorForSpace(SrgbColorSpace, [3]float64{0.5, 0.5, 0.5}, 1, [4]bool{false, false, false, false})
	if err != nil {
		t.Fatal(err)
	}
	if c.IsChannel0Missing() {
		t.Error("no channels should be missing")
	}
	c2, err := NewColorForSpace(SrgbColorSpace, [3]float64{0, 0.5, 0.5}, 1, [4]bool{true, false, false, false})
	if err != nil {
		t.Fatal(err)
	}
	if !c2.IsChannel0Missing() {
		t.Error("channel0 should be missing")
	}
}

func TestColorEqualsSameSpace(t *testing.T) {
	c1, _ := NewColorRGB(255, 0, 0, 1)
	c2, _ := NewColorRGB(255, 0, 0, 1)
	if !c1.Equals(c2) {
		t.Error("same color should be equal")
	}
	c3, _ := NewColorRGB(0, 0, 255, 1)
	if c1.Equals(c3) {
		t.Error("different colors should not be equal")
	}
}

func TestColorEqualsLegacy(t *testing.T) {
	c1, _ := NewColorRGB(255, 0, 0, 1)
	c2, _ := NewColorHSL(0, 100, 50, 1)
	if !c1.Equals(c2) {
		t.Error("legacy RGB(255,0,0) should equal HSL(0,100,50)")
	}
	c3, _ := NewColorHWB(0, 0, 0, 1)
	if !c1.Equals(c3) {
		t.Error("legacy RGB(255,0,0) should equal HWB(0,0,0) = red")
	}
}

func TestColorEqualsLegacyVsModern(t *testing.T) {
	legacy, _ := NewColorRGB(255, 0, 0, 1)
	modern, _ := NewColorSRGB(1, 0, 0, 1)
	if legacy.Equals(modern) {
		t.Error("legacy RGB should not equal modern sRGB (different space)")
	}
}

func TestColorEqualsMissing(t *testing.T) {
	c1, _ := NewColorRGBInternal(nil, ptr(0.0), ptr(0.0), ptr(1.0), nil)
	c2, _ := NewColorRGBInternal(nil, ptr(0.0), ptr(0.0), ptr(1.0), nil)
	if !c1.Equals(c2) {
		t.Error("colors with same missing channels should be equal")
	}
}

func TestColorMissingChannels(t *testing.T) {
	r := 255.0
	b := 0.0
	a := 1.0
	c, _ := NewColorRGBInternal(&r, nil, &b, &a, nil)
	if c.Channel0OrNil() == nil {
		t.Error("channel0 should not be nil")
	}
	if c.Channel1OrNil() != nil {
		t.Error("channel1 should be nil (missing)")
	}
	if !c.IsChannel1Missing() {
		t.Error("IsChannel1Missing should be true")
	}
	if c.Channel0() != 255 {
		t.Error("missing channel should still have stored value for getter (0)")
	}
}

func TestColorLegacyGetters(t *testing.T) {
	c, _ := NewColorRGB(255, 128, 0, 1)
	r, err := c.Red()
	if err != nil {
		t.Fatal(err)
	}
	if r != 255 {
		t.Errorf("Red() = %v, want 255", r)
	}
	g, _ := c.Green()
	if g != 128 {
		t.Errorf("Green() = %v, want 128", g)
	}

	modern, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	_, err = modern.Red()
	if err == nil {
		t.Error("Red() on non-legacy should error")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Errorf("error should mention legacy, got: %v", err)
	}
}

func TestColorChannelByName(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	v, err := c.ChannelByName("red")
	if err != nil {
		t.Fatal(err)
	}
	if v != 255 {
		t.Errorf("ChannelByName(red) = %v, want 255", v)
	}
	v, err = c.ChannelByName("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Errorf("ChannelByName(alpha) = %v, want 1", v)
	}
	_, err = c.ChannelByName("bogus")
	if err == nil {
		t.Error("unknown channel should error")
	}
}

func TestColorPowerless(t *testing.T) {
	gray, _ := NewColorHSL(0, 0, 50, 1)
	if !gray.IsChannel0Powerless() {
		t.Error("hsl gray should have powerless hue")
	}

	white, _ := NewColorHWB(0, 100, 0, 1)
	if !white.IsChannel0Powerless() {
		t.Error("hwb white should have powerless hue")
	}

	lchGray, _ := NewColorLCH(50, 0, 0, 1)
	if !lchGray.IsChannel2Powerless() {
		t.Error("lch gray should have powerless hue")
	}
}

func TestColorFormat(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	if c.Format() != nil {
		t.Error("default format should be nil")
	}
	c.SetFormat(ColorFormatRGBFunction)
	if c.Format() != ColorFormatRGBFunction {
		t.Error("format should be stored")
	}
}

func TestColorHashCode(t *testing.T) {
	c1, _ := NewColorRGB(255, 0, 0, 1)
	c2, _ := NewColorHSL(0, 100, 50, 1)
	if c1.HashCode() != c2.HashCode() {
		t.Error("legacy colors that are equal should have same hash")
	}
	c3, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	h := c3.HashCode()
	if h == 0 {
		t.Error("modern color hash should be non-zero")
	}
}

func TestColorToCssString_Legacy(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	got, err := c.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "red" {
		t.Errorf("ToCssString(red) = %q, want %q", got, "red")
	}

	c, _ = NewColorRGB(255, 0, 0, 0.5)
	got, err = c.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "255") || !strings.Contains(got, "0.5") {
		t.Errorf("rgba: got %q", got)
	}
}

func TestColorToCssString_Modern(t *testing.T) {
	c, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	got, err := c.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "srgb") {
		t.Errorf("should contain srgb: got %q", got)
	}

	c, _ = NewColorLab(50, 25, -25, 1)
	got, err = c.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "lab") {
		t.Errorf("should contain lab: got %q", got)
	}
}

func TestColorString(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	got, err := c.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "red" {
		t.Errorf("String() = %q, want %q", got, "red")
	}
}

func TestColorOperators(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	_, err := c.Plus(NewUnitlessNumber(1))
	if err == nil {
		t.Error("color + number should error")
	}
	other, _ := NewColorRGB(0, 0, 255, 1)
	_, err = c.Plus(other)
	if err == nil {
		t.Error("color + color should error")
	}

	got, err := c.Plus(&SassString{Text: "x", HasQuotes: true})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := got.String()
	if !strings.Contains(s, "red") {
		t.Errorf("color + string should concat: got %q", s)
	}
}

func ptr(f float64) *float64 { return &f }

// Degenerate colors (#2840): forSpaceInternal normalizes NaN and negative zero
// linear channels (and alpha) to 0, and non-finite or zero hues to 0.
func TestForSpaceInternalNormalizesDegenerate(t *testing.T) {
	negZero := math.Copysign(0, -1)
	c, err := NewColorForSpaceInternal(SrgbColorSpace, ptr(math.NaN()), ptr(negZero), ptr(0.5), ptr(negZero))
	if err != nil {
		t.Fatal(err)
	}
	if math.Float64bits(c.Channel0()) != math.Float64bits(0) {
		t.Error("channel0 should normalize NaN to +0")
	}
	if math.Float64bits(c.Channel1()) != math.Float64bits(0) {
		t.Error("channel1 should normalize -0 to +0")
	}
	if c.Channel2() != 0.5 {
		t.Errorf("channel2 = %v, want 0.5", c.Channel2())
	}
	if math.Float64bits(c.Alpha()) != math.Float64bits(0) {
		t.Error("alpha should normalize -0 to +0")
	}

	h, err := NewColorForSpaceInternal(HslColorSpace, ptr(math.Inf(1)), ptr(50), ptr(50), ptr(1))
	if err != nil {
		t.Fatal(err)
	}
	if h.Channel0() != 0 {
		t.Errorf("hue = %v, want 0", h.Channel0())
	}

	// The rgb raw constructor bypasses preprocessing (Dart rgbInternal).
	r, err := NewColorRGBInternal(ptr(math.NaN()), ptr(0), ptr(0), ptr(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(r.Channel0()) {
		t.Errorf("rgbInternal channel0 = %v, want NaN", r.Channel0())
	}
}
