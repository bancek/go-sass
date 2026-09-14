package functions

import (
	"testing"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/value"
)

func colorEvalColor(c *BuiltInCallable, args ...value.Value) (value.Value, error) {
	result, err := c.CallbackFor(len(args), nil)
	if err != nil {
		return nil, err
	}
	paramCount := 0
	if result.Params != nil {
		paramCount = len(result.Params.Parameters)
	}
	for len(args) < paramCount {
		args = append(args, value.Null)
	}
	ec := &evalcontext.EvaluationContext{Logger: &recordingLogger{}}
	return result.Fn(ec, args)
}

// ============================================================================
// grayscale
// ============================================================================

func TestGrayscale(t *testing.T) {
	// grayscale of red = gray
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEvalColor(grayscaleCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Channel0() != 127.5 || result.Channel1() != 127.5 || result.Channel2() != 127.5 {
		t.Errorf("grayscale of red = (%v, %v, %v), want (127.5, 127.5, 127.5)",
			result.Channel0(), result.Channel1(), result.Channel2())
	}

	// grayscale of color with alpha
	c2 := colorMust(value.NewColorRGB(255, 0, 0, 0.5))
	v2, err := colorEvalColor(grayscaleCallable(), c2)
	if err != nil {
		t.Fatal(err)
	}
	r2 := v2.(*value.SassColor)
	if r2.Alpha() != 0.5 {
		t.Errorf("grayscale alpha = %v, want 0.5", r2.Alpha())
	}

	// Module version
	c3 := colorMust(value.NewColorRGB(0, 255, 0, 1))
	v3, err := colorEvalColor(grayscaleModuleCallable(), c3)
	if err != nil {
		t.Fatal(err)
	}
	r3 := v3.(*value.SassColor)
	if r3.Channel0() != 127.5 || r3.Channel1() != 127.5 || r3.Channel2() != 127.5 {
		t.Errorf("module grayscale of green = (%v, %v, %v), want (127.5, 127.5, 127.5)",
			r3.Channel0(), r3.Channel1(), r3.Channel2())
	}

	// Number arg returns a CSS function string
	v3, gErr := colorEvalColor(grayscaleCallable(), colorNum(42))
	if gErr != nil {
		t.Fatal(gErr)
	}
	checkStr(t, v3, "grayscale(42)")
}

// ============================================================================
// saturate
// ============================================================================

func TestSaturate(t *testing.T) {
	// saturate by 50%
	c := colorMust(value.NewColorHSL(0, 50, 50, 1))
	v, err := colorEvalColor(saturateCallable(), c, colorNum(50))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	s, _ := result.Saturation()
	if s != 100 {
		t.Errorf("saturation = %v, want 100", s)
	}

	// Error: non-legacy color
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(saturateCallable(), c2, colorNum(50))
	assertErrMsg(t, err, "saturate() is only supported for legacy colors. Please use color.adjust() instead with an explicit $space argument.")

	// Error: amount out of range
	_, err = colorEvalColor(saturateCallable(), c, colorNum(150))
	assertErrMsg(t, err, "$amount: Expected 150 to be within 0 and 100.")
}

// ============================================================================
// desaturate
// ============================================================================

func TestDesaturate(t *testing.T) {
	c := colorMust(value.NewColorHSL(0, 100, 50, 1))
	v, err := colorEvalColor(desaturateCallable(), c, colorNum(50))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	s, _ := result.Saturation()
	if s != 50 {
		t.Errorf("saturation = %v, want 50", s)
	}

	// Error: non-legacy
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(desaturateCallable(), c2, colorNum(50))
	assertErrMsg(t, err, "desaturate() is only supported for legacy colors. Please use color.adjust() instead with an explicit $space argument.")

	// Error: amount out of range
	_, err = colorEvalColor(desaturateCallable(), c, colorNum(150))
	assertErrMsg(t, err, "$amount: Expected 150 to be within 0 and 100.")
}

// ============================================================================
// adjust-hue
// ============================================================================

func TestAdjustHue(t *testing.T) {
	c := colorMust(value.NewColorHSL(0, 100, 50, 1))
	v, err := colorEvalColor(adjustHueCallable(), c, colorNum(90))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	h, _ := result.Hue()
	if h != 90 {
		t.Errorf("hue = %v, want 90", h)
	}

	// Error: non-legacy
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(adjustHueCallable(), c2, colorNum(90))
	assertErrMsg(t, err, "adjust-hue() is only supported for legacy colors. Please use color.adjust() instead with an explicit $space argument.")

	// Error: not a color
	_, err = colorEvalColor(adjustHueCallable(), colorNum(42), colorNum(90))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: not a number
	_, err = colorEvalColor(adjustHueCallable(), c, colorStr("bad"))
	assertErrMsg(t, err, "$degrees: bad is not a number.")
}

// ============================================================================
// lighten
// ============================================================================

func TestLighten(t *testing.T) {
	c := colorMust(value.NewColorHSL(0, 100, 50, 1))
	v, err := colorEvalColor(lightenCallable(), c, colorNum(20))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	l, _ := result.Lightness()
	if l != 70 {
		t.Errorf("lightness = %v, want 70", l)
	}

	// Error: non-legacy
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(lightenCallable(), c2, colorNum(20))
	assertErrMsg(t, err, "lighten() is only supported for legacy colors. Please use color.adjust() instead with an explicit $space argument.")

	// Error: amount out of range
	_, err = colorEvalColor(lightenCallable(), c, colorNum(150))
	assertErrMsg(t, err, "$amount: Expected 150 to be within 0 and 100.")
}

// ============================================================================
// darken
// ============================================================================

func TestDarken(t *testing.T) {
	c := colorMust(value.NewColorHSL(0, 100, 50, 1))
	v, err := colorEvalColor(darkenCallable(), c, colorNum(20))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	l, _ := result.Lightness()
	if l != 30 {
		t.Errorf("lightness = %v, want 30", l)
	}
}

// ============================================================================
// opacify / fade-in
// ============================================================================

func TestOpacify(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 0.5))
	v, err := colorEvalColor(opacifyCallable("opacify"), c, colorNum(0.3))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Alpha() != 0.8 {
		t.Errorf("alpha = %v, want 0.8", result.Alpha())
	}

	// Error: non-legacy
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(opacifyCallable("opacify"), c2, colorNum(0.3))
	assertErrMsg(t, err, "opacify() is only supported for legacy colors. Please use color.adjust() instead with an explicit $space argument.")

	// Error: amount out of range
	_, err = colorEvalColor(opacifyCallable("opacify"), c, colorNum(2))
	assertErrMsg(t, err, "$amount: Expected 2 to be within 0 and 1.")
}

// ============================================================================
// transparentize / fade-out
// ============================================================================

func TestTransparentize(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 0.5))
	v, err := colorEvalColor(transparentizeCallable("transparentize"), c, colorNum(0.3))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Alpha() != 0.2 {
		t.Errorf("alpha = %v, want 0.2", result.Alpha())
	}
}

// ============================================================================
// invert
// ============================================================================

func TestInvert(t *testing.T) {
	// invert of white with default weight = black
	c := colorMust(value.NewColorRGB(255, 255, 255, 1))
	v, err := colorEvalColor(invertCallable(), c, colorNum(100))
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Channel0() != 0 || result.Channel1() != 0 || result.Channel2() != 0 {
		t.Errorf("invert of white = (%v, %v, %v), want (0, 0, 0)",
			result.Channel0(), result.Channel1(), result.Channel2())
	}

	// invert with weight=0 returns original
	w0 := value.NewSingleUnitNumber(0, "%")
	v2, err := colorEvalColor(invertCallable(), c, w0)
	if err != nil {
		t.Fatal(err)
	}
	r2 := v2.(*value.SassColor)
	if r2.Channel0() != 255 || r2.Channel1() != 255 || r2.Channel2() != 255 {
		t.Errorf("invert weight=0 = (%v, %v, %v), want (255, 255, 255)",
			r2.Channel0(), r2.Channel1(), r2.Channel2())
	}

	// Error: wrong unit on weight (via plain-CSS path with number + non-100 weight)
	_, err = colorEvalColor(invertCallable(), colorNum(42), colorNum(50))
	assertErrMsg(t, err, "Only one argument may be passed to the plain-CSS invert() function.")
}

// Module version
func TestInvertModule(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 255, 255, 1))
	w100 := value.NewSingleUnitNumber(100, "%")
	v, err := colorEvalColor(invertModuleCallable(), c, w100)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Channel0() != 0 || result.Channel1() != 0 || result.Channel2() != 0 {
		t.Errorf("invert of white = (%v, %v, %v), want (0, 0, 0)",
			result.Channel0(), result.Channel1(), result.Channel2())
	}
}

// ============================================================================
// complement
// ============================================================================

func TestComplement(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEvalColor(complementCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Channel0() != 0 {
		t.Errorf("complement red channel = %v, want 0", result.Channel0())
	}
	// Green and blue may be ~255 due to floating point
	if result.Channel1() < 254.9 || result.Channel1() > 255.1 {
		t.Errorf("complement green channel = %v, want ~255", result.Channel1())
	}
	if result.Channel2() < 254.9 || result.Channel2() > 255.1 {
		t.Errorf("complement blue channel = %v, want ~255", result.Channel2())
	}

	// Error: non-polar space
	_, err = colorEvalColor(complementCallable(), c, colorStr("srgb"))
	assertErrMsg(t, err, "$space: Color space srgb doesn't have a hue channel.")
}

// ============================================================================
// mix
// ============================================================================

func TestMix(t *testing.T) {
	c1 := colorMust(value.NewColorRGB(255, 0, 0, 1))
	c2 := colorMust(value.NewColorRGB(0, 0, 255, 1))
	w50 := value.NewSingleUnitNumber(50, "%")
	v, err := colorEvalColor(mixCallable(), c1, c2, w50)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	r := result.Channel0()
	// mix red+blue 50% should give ~127.5
	if r != 127.5 && r != 128 {
		t.Errorf("mix red+blue 50%% red channel = %v, want ~127.5", r)
	}

	// Error: non-legacy without method
	c3 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(mixCallable(), c3, c2, w50)
	c3Str := colorString(c3)
	assertErrMsg(t, err, "$color1: To use color.mix() with non-legacy color "+c3Str+", you must provide a $method.")
}

// ============================================================================
// ie-hex-str
// ============================================================================

func TestIeHexStr(t *testing.T) {
	c := colorMust(value.NewColorRGB(0, 255, 0, 0.5))
	v, err := colorEvalColor(ieHexStrCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	checkStr(t, v, "#8000FF00")

	// Error: not a color
	_, err = colorEvalColor(ieHexStrCallable(), colorNum(42))
	assertErrMsg(t, err, "$color: 42 is not a color.")
}

// ============================================================================
// alpha
// ============================================================================

func TestAlpha(t *testing.T) {
	// Single arg: get alpha
	c := colorMust(value.NewColorRGB(255, 0, 0, 0.75))
	v, err := colorEvalColor(alphaCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	checkNum(t, v, 0.75)

	// Module version
	v2, err := colorEvalColor(alphaModuleCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	checkNum(t, v2, 0.75)

	// Error: non-legacy color
	c2 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEvalColor(alphaModuleCallable(), c2)
	assertErrMsg(t, err, "color.alpha() is only supported for legacy colors. Please use color.channel() instead.")

	// Error: not a color
	_, err = colorEvalColor(alphaCallable(), colorNum(42))
	assertErrMsg(t, err, "$color: 42 is not a color.")
}

// ============================================================================
// opacity
// ============================================================================

func TestOpacity(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 0.75))
	v, err := colorEvalColor(opacityCallable(), c)
	if err != nil {
		t.Fatal(err)
	}
	checkNum(t, v, 0.75)

	// Module version
	v2, e2 := colorEvalColor(opacityModuleCallable(), c)
	if e2 != nil {
		t.Fatal(e2)
	}
	checkNum(t, v2, 0.75)

	// Number arg returns CSS function string
	v3, e3 := colorEvalColor(opacityCallable(), colorNum(42))
	if e3 != nil {
		t.Fatal(e3)
	}
	checkStr(t, v3, "opacity(42)")
}
