package functions

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/value"
)

// ============================================================================
// rgb / rgba
// ============================================================================

func TestRgbSpaces(t *testing.T) {
	// rgb(255, 0, 0) = red in legacy RGB space
	v, err := colorEvalColor(rgbCallable(), colorNum(255), colorNum(0), colorNum(0))
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.RgbColorSpace {
		t.Error("expected RGB space")
	}
	if c.Channel0() != 255 || c.Channel1() != 0 || c.Channel2() != 0 {
		t.Errorf("rgb(255,0,0) = (%v,%v,%v)", c.Channel0(), c.Channel1(), c.Channel2())
	}

	// rgb(255, 0, 0, 0.5) = red with alpha
	v2, err := colorEvalColor(rgbCallable(), colorNum(255), colorNum(0), colorNum(0), colorNum(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if c2 := v2.(*value.SassColor); c2.Alpha() != 0.5 {
		t.Errorf("alpha = %v, want 0.5", c2.Alpha())
	}

	// Error: bad red channel
	_, err = colorEvalColor(rgbCallable(), colorStr("bad"), colorNum(0), colorNum(0))
	assertErrMsg(t, err, "$red: bad is not a number.")
}

func TestRgbaSpaces(t *testing.T) {
	v, err := colorEvalColor(rgbaCallable(), colorNum(255), colorNum(0), colorNum(0), colorNum(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if c := v.(*value.SassColor); c.Alpha() != 0.5 {
		t.Errorf("alpha = %v, want 0.5", c.Alpha())
	}

	// Also works with 3 args
	v2, err := colorEvalColor(rgbaCallable(), colorNum(0), colorNum(255), colorNum(0))
	if err != nil {
		t.Fatal(err)
	}
	if c2 := v2.(*value.SassColor); c2.Channel1() != 255 {
		t.Errorf("green = %v, want 255", c2.Channel1())
	}
}

// ============================================================================
// hsl / hsla
// ============================================================================

func TestHslSpaces(t *testing.T) {
	v, err := colorEvalColor(hslCallable(), colorNum(0), colorNum(100), colorNum(50))
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.HslColorSpace {
		t.Error("expected HSL space")
	}

	// With alpha
	v2, err := colorEvalColor(hslCallable(), colorNum(0), colorNum(100), colorNum(50), colorNum(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if c2 := v2.(*value.SassColor); c2.Alpha() != 0.5 {
		t.Errorf("alpha = %v, want 0.5", c2.Alpha())
	}

	// Error: missing lightness
	_, err = colorEvalColor(hslCallable(), colorNum(0), colorNum(100))
	assertErrMsg(t, err, "Missing argument $lightness.")
}

func TestHslaSpaces(t *testing.T) {
	v, err := colorEvalColor(hslaCallable(), colorNum(0), colorNum(100), colorNum(50), colorNum(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if c := v.(*value.SassColor); c.Alpha() != 0.5 {
		t.Errorf("alpha = %v, want 0.5", c.Alpha())
	}
}

// ============================================================================
// hwb
// ============================================================================

func TestHwbSpaces(t *testing.T) {
	w0 := value.NewSingleUnitNumber(0, "%")
	b0 := value.NewSingleUnitNumber(0, "%")
	v, err := colorEvalColor(hwbCallable(), colorNum(0), w0, b0, colorNum(1))
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.HwbColorSpace {
		t.Error("expected HWB space")
	}
}

// ============================================================================
// lab, lch, oklab, oklch
// ============================================================================

func TestLabSpaces(t *testing.T) {
	// lab with 3 channel values as space-separated list
	spaceList, _ := value.NewSassList(
		[]value.Value{
			value.NewSingleUnitNumber(50, "%"),
			colorNum(0),
			colorNum(0),
		},
		value.ListSeparatorSpace,
		false,
	)
	v, err := colorEvalColor(labCallable(), spaceList)
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.LabColorSpace {
		t.Error("expected Lab space")
	}
}

func TestLchSpaces(t *testing.T) {
	spaceList, _ := value.NewSassList(
		[]value.Value{
			value.NewSingleUnitNumber(50, "%"),
			colorNum(30),
			colorNum(200),
		},
		value.ListSeparatorSpace,
		false,
	)
	v, err := colorEvalColor(lchCallable(), spaceList)
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.LchColorSpace {
		t.Error("expected Lch space")
	}
}

func TestOklabSpaces(t *testing.T) {
	spaceList, _ := value.NewSassList(
		[]value.Value{
			value.NewSingleUnitNumber(50, "%"),
			value.NewUnitlessNumber(0.1),
			value.NewUnitlessNumber(-0.1),
		},
		value.ListSeparatorSpace,
		false,
	)
	v, err := colorEvalColor(oklabCallable(), spaceList)
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.OklabColorSpace {
		t.Error("expected Oklab space")
	}
}

func TestOklchSpaces(t *testing.T) {
	spaceList, _ := value.NewSassList(
		[]value.Value{
			value.NewSingleUnitNumber(50, "%"),
			value.NewUnitlessNumber(0.1),
			colorNum(180),
		},
		value.ListSeparatorSpace,
		false,
	)
	v, err := colorEvalColor(oklchCallable(), spaceList)
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.OklchColorSpace {
		t.Error("expected Oklch space")
	}
}

// ============================================================================
// color() global function
// ============================================================================

func TestColorGlobalFunctionSpaces(t *testing.T) {
	fns := GlobalColorFunctions()
	colorFn := findGlobalFn(t, fns, "color")

	// color(srgb 1 0 0) — must be a list, not a string
	spaceList, _ := value.NewSassList(
		[]value.Value{
			colorStr("srgb"),
			colorNum(1),
			colorNum(0),
			colorNum(0),
		},
		value.ListSeparatorSpace,
		false,
	)
	v, err := colorEvalColor(colorFn, spaceList)
	if err != nil {
		t.Fatal(err)
	}
	c := v.(*value.SassColor)
	if c.Space() != value.SrgbColorSpace {
		t.Errorf("space = %v, want SrgbColorSpace", c.Space())
	}

	// color("red") errors — it passes through to ColorSpaceFromName which fails
	_, err = colorEvalColor(colorFn, colorStr("red"))
	assertErrMsg(t, err, `$description: Unknown color space "red".`)

	// color((not-a-space)) errors — unknown color space
	badList, _ := value.NewSassList(
		[]value.Value{colorStr("not-a-space")},
		value.ListSeparatorSpace,
		false,
	)
	_, err = colorEvalColor(colorFn, badList)
	assertErrMsg(t, err, `$description: Unknown color space "not-a-space".`)
}

// ============================================================================
// adjust / scale / change
// ============================================================================

func TestAdjust(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	// color.adjust($color, $lightness: 20%)
	args := []value.Value{c, makeKwargList(t, map[string]value.Value{"lightness": value.NewSingleUnitNumber(20, "%")})}
	v, err := colorEvalColor(adjustColorCallable(), args...)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	l, _ := result.Lightness()
	if l != 70 {
		t.Errorf("lightness = %v, want 70", l)
	}
}

func TestScale(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	args := []value.Value{c, makeKwargList(t, map[string]value.Value{"lightness": value.NewSingleUnitNumber(50, "%")})}
	v, err := colorEvalColor(scaleColorCallable(), args...)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	l, _ := result.Lightness()
	if l != 75 {
		t.Errorf("lightness = %v, want 75", l)
	}
}

func TestChange(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	args := []value.Value{c, makeKwargList(t, map[string]value.Value{"red": colorNum(100)})}
	v, err := colorEvalColor(changeColorCallable(), args...)
	if err != nil {
		t.Fatal(err)
	}
	result := v.(*value.SassColor)
	if result.Channel0() != 100 {
		t.Errorf("red = %v, want 100", result.Channel0())
	}
}

func TestChangePositionalError(t *testing.T) {
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	al, _ := value.NewSassArgumentList([]value.Value{colorNum(1)}, orderedmap.New[string, value.Value](), value.ListSeparatorComma)
	_, err := colorEvalColor(changeColorCallable(), c, al)
	assertErrMsg(t, err, "Only one positional argument is allowed. All other arguments must be passed by name.")
}

// makeKwargList creates an argument list with keywords for testing.
func makeKwargList(t *testing.T, kwargs map[string]value.Value) value.Value {
	t.Helper()
	kmap := orderedmap.New[string, value.Value]()
	for k, v := range kwargs {
		kmap.Put(k, v)
	}
	al, err := value.NewSassArgumentList(nil, kmap, value.ListSeparatorComma)
	if err != nil {
		t.Fatal(err)
	}
	return al
}
