package functions

import (
	"testing"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/value"
)

// --- helpers ---

func colorEval(c *BuiltInCallable, args ...value.Value) (value.Value, error) {
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

func findModuleFn(t *testing.T, fns orderedmap.Map[string, sasscallable.Callable], name string) *BuiltInCallable {
	t.Helper()
	c, has := fns.Get(name)
	if !has {
		t.Fatalf("function %q not found in module", name)
	}
	bic, ok := c.(*BuiltInCallable)
	if !ok {
		t.Fatalf("function %q is not a *BuiltInCallable, got %T", name, c)
	}
	return bic
}

func findGlobalFn(t *testing.T, callables []sasscallable.Callable, name string) *BuiltInCallable {
	t.Helper()
	for _, c := range callables {
		if c.Name() == name {
			bic, ok := c.(*BuiltInCallable)
			if !ok {
				t.Fatalf("callable %q is not a *BuiltInCallable, got %T", name, c)
			}
			return bic
		}
	}
	t.Fatalf("callable %q not found", name)
	return nil
}

func colorNum(v float64) value.Value {
	return value.NewUnitlessNumber(v)
}

func colorStr(s string) value.Value {
	return &value.SassString{Text: s, HasQuotes: false}
}

func colorQuoted(s string) value.Value {
	return &value.SassString{Text: s, HasQuotes: true}
}

func colorMust(c *value.SassColor, err error) *value.SassColor {
	if err != nil {
		panic(err)
	}
	return c
}

func colorString(c *value.SassColor) string {
	s, err := c.String()
	if err != nil {
		panic(err)
	}
	return s
}

func checkNum(t *testing.T, v value.Value, want float64) {
	t.Helper()
	n, ok := v.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", v)
	}
	if n.NumValue() != want {
		t.Errorf("NumValue() = %v, want %v", n.NumValue(), want)
	}
}

func checkStr(t *testing.T, v value.Value, want string) {
	t.Helper()
	s, ok := v.(*value.SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", v)
	}
	if s.Text != want {
		t.Errorf("Text = %q, want %q", s.Text, want)
	}
	if s.HasQuotes {
		t.Errorf("HasQuotes = true, want false")
	}
}

func checkBool(t *testing.T, v value.Value, want bool) {
	t.Helper()
	if want {
		if v != value.SassTrue {
			t.Errorf("expected SassTrue, got %T", v)
		}
	} else {
		if v != value.SassFalse {
			t.Errorf("expected SassFalse, got %T", v)
		}
	}
}

// ============================================================================
// Test: GlobalColorFunctions
// ============================================================================

func TestGlobalColorFunctions(t *testing.T) {
	fns := GlobalColorFunctions()

	names := make(map[string]bool)
	for _, c := range fns {
		names[c.Name()] = true
	}

	expectedNames := []string{
		"red", "green", "blue",
		"mix",
		"rgb", "rgba",
		"invert",
		"hue", "saturation", "lightness",
		"hsl", "hsla",
		"grayscale",
		"adjust-hue",
		"lighten", "darken",
		"saturate", "desaturate",
		"opacify", "fade-in",
		"transparentize", "fade-out",
		"alpha", "opacity",
		"color",
		"hwb",
		"lab", "lch", "oklab", "oklch",
		"complement",
		"ie-hex-str",
		"adjust-color",
		"scale-color",
		"change-color",
	}

	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("GlobalColorFunctions: missing %q", name)
		}
	}
}

// ============================================================================
// Test: ColorModule
// ============================================================================

func TestColorModule(t *testing.T) {
	m := ColorModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:color" {
		t.Errorf("URL = %q, want %q", url, "sass:color")
	}

	fns := m.Functions()

	moduleNames := []string{
		"red", "green", "blue",
		"mix",
		"invert",
		"hue", "saturation", "lightness",
		"adjust-hue",
		"lighten", "darken",
		"saturate", "desaturate",
		"grayscale",
		"hwb",
		"whiteness", "blackness",
		"opacify", "fade-in",
		"transparentize", "fade-out",
		"alpha", "opacity",
		"space",
		"to-space",
		"is-legacy",
		"is-missing",
		"is-in-gamut",
		"to-gamut",
		"channel",
		"same",
		"is-powerless",
		"complement",
		"adjust",
		"scale",
		"change",
		"ie-hex-str",
	}

	for _, name := range moduleNames {
		if _, has := fns.Get(name); !has {
			t.Errorf("ColorModule: missing function %q", name)
		}
	}

	// Verify no extra functions
	for key := range fns.Keys() {
		found := false
		for _, name := range moduleNames {
			if key == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ColorModule: unexpected function %q", key)
		}
	}
}

// ============================================================================
// Test: Module inline function - space
// ============================================================================

func TestColorModuleSpace(t *testing.T) {
	m := ColorModule()
	spaceFn := findModuleFn(t, m.Functions(), "space")

	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEval(spaceFn, c)
	if err != nil {
		t.Fatal(err)
	}
	checkStr(t, v, "rgb")

	c2 := colorMust(value.NewColorHSL(120, 100, 50, 1))
	v, err = colorEval(spaceFn, c2)
	if err != nil {
		t.Fatal(err)
	}
	checkStr(t, v, "hsl")

	c3 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	v, err = colorEval(spaceFn, c3)
	if err != nil {
		t.Fatal(err)
	}
	checkStr(t, v, "srgb")

	c4 := colorMust(value.NewColorLab(50, 0, 0, 1))
	v, err = colorEval(spaceFn, c4)
	if err != nil {
		t.Fatal(err)
	}
	checkStr(t, v, "lab")

	_, err = colorEval(spaceFn, colorNum(42))
	assertErrMsg(t, err, "$color: 42 is not a color.")
}

// ============================================================================
// Test: Module inline function - is-legacy
// ============================================================================

func TestColorModuleIsLegacy(t *testing.T) {
	m := ColorModule()
	isLegacyFn := findModuleFn(t, m.Functions(), "is-legacy")

	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEval(isLegacyFn, c)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c2 := colorMust(value.NewColorHSL(120, 100, 50, 1))
	v, err = colorEval(isLegacyFn, c2)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c3 := colorMust(value.NewColorHWB(0, 0, 0, 1))
	v, err = colorEval(isLegacyFn, c3)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c4 := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	v, err = colorEval(isLegacyFn, c4)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	c5 := colorMust(value.NewColorLab(50, 0, 0, 1))
	v, err = colorEval(isLegacyFn, c5)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	_, err = colorEval(isLegacyFn, colorNum(42))
	assertErrMsg(t, err, "$color: 42 is not a color.")
}

// ============================================================================
// Test: Module inline function - is-missing
// ============================================================================

func TestColorModuleIsMissing(t *testing.T) {
	m := ColorModule()
	isMissingFn := findModuleFn(t, m.Functions(), "is-missing")

	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEval(isMissingFn, c, colorQuoted("red"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	c2 := colorMust(value.NewColorForSpace(value.RgbColorSpace, [3]float64{0, 0, 0}, 1, [4]bool{true, false, false, false}))
	v, err = colorEval(isMissingFn, c2, colorQuoted("red"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c3 := colorMust(value.NewColorForSpace(value.RgbColorSpace, [3]float64{255, 0, 0}, 0.5, [4]bool{false, false, false, true}))
	v, err = colorEval(isMissingFn, c3, colorQuoted("alpha"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	v, err = colorEval(isMissingFn, c2, colorQuoted("green"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	// Error: not a color
	_, err = colorEval(isMissingFn, colorNum(42), colorQuoted("red"))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: channel not a string
	_, err = colorEval(isMissingFn, c, colorNum(1))
	assertErrMsg(t, err, "$channel: 1 is not a string.")

	// Error: channel unquoted
	_, err = colorEval(isMissingFn, c, colorStr("red"))
	assertErrMsg(t, err, "$channel: Expected red to be a quoted string.")

	// Error: invalid channel name
	_, err = colorEval(isMissingFn, c, colorQuoted("nonexistent"))
	assertErrMsg(t, err, "$channel: Color red doesn't have a channel named \"nonexistent\".")
}

// ============================================================================
// Test: Module inline function - to-gamut (error cases)
// ============================================================================

func TestColorModuleToGamut(t *testing.T) {
	m := ColorModule()
	toGamutFn := findModuleFn(t, m.Functions(), "to-gamut")

	// Error: not a color
	_, err := colorEval(toGamutFn, colorNum(42), colorStr("srgb"), colorQuoted("clip"))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: $method is null
	c := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEval(toGamutFn, c, colorStr("srgb"))
	assertErrMsg(t, err, "$method: color.to-gamut() requires a $method argument for forwards-"+
		"compatibility with changes in the CSS spec. Suggestion:\n"+
		"\n"+
		"$method: local-minde")

	// Error: $method not a string
	_, err = colorEval(toGamutFn, c, colorStr("srgb"), colorNum(1))
	assertErrMsg(t, err, "$method: 1 is not a string.")

	// Error: $method quoted (AssertUnquoted fails)
	_, err = colorEval(toGamutFn, c, colorStr("srgb"), colorQuoted("clip"))
	assertErrMsg(t, err, `$method: Expected "clip" to be an unquoted string.`)

	// Error: $space quoted (AssertUnquoted fails)
	_, err = colorEval(toGamutFn, c, colorQuoted("srgb"), colorQuoted("clip"))
	assertErrMsg(t, err, `$space: Expected "srgb" to be an unquoted string.`)

	// Error: $space not a string
	_, err = colorEval(toGamutFn, c, colorNum(1), colorQuoted("clip"))
	assertErrMsg(t, err, "$space: 1 is not a string.")

	// Error: unknown space name
	_, err = colorEval(toGamutFn, c, colorStr("not-a-space"), colorQuoted("clip"))
	assertErrMsg(t, err, `$space: Unknown color space "not-a-space".`)

	// Error: unknown gamut map method
	_, err = colorEval(toGamutFn, c, colorStr("srgb"), colorStr("not-a-method"))
	assertErrMsg(t, err, `Unknown gamut map method "not-a-method".`)
}

// ============================================================================
// Test: Module inline function - to-space (error cases)
// ============================================================================

func TestColorModuleToSpace(t *testing.T) {
	m := ColorModule()
	toSpaceFn := findModuleFn(t, m.Functions(), "to-space")

	// Error: not a color
	_, err := colorEval(toSpaceFn, colorNum(42), colorStr("srgb"))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: $space not a string
	c := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1))
	_, err = colorEval(toSpaceFn, c, colorNum(1))
	assertErrMsg(t, err, "$space: 1 is not a string.")

	// Error: $space quoted
	_, err = colorEval(toSpaceFn, c, colorQuoted("srgb"))
	assertErrMsg(t, err, `$space: Expected "srgb" to be an unquoted string.`)

	// Error: unknown space name
	_, err = colorEval(toSpaceFn, c, colorStr("not-a-space"))
	assertErrMsg(t, err, `$space: Unknown color space "not-a-space".`)
}

// ============================================================================
// Test: Module inline function - channel (error cases)
// ============================================================================

func TestColorModuleChannel(t *testing.T) {
	m := ColorModule()
	channelFn := findModuleFn(t, m.Functions(), "channel")

	// Error: not a color
	_, err := colorEval(channelFn, colorNum(42), colorQuoted("red"))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: channel not a string
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	_, err = colorEval(channelFn, c, colorNum(1))
	assertErrMsg(t, err, "$channel: 1 is not a string.")

	// Error: channel unquoted
	_, err = colorEval(channelFn, c, colorStr("red"))
	assertErrMsg(t, err, "$channel: Expected red to be a quoted string.")
}

// ============================================================================
// Test: Module inline function - same (error cases)
// ============================================================================

func TestColorModuleSame(t *testing.T) {
	m := ColorModule()
	sameFn := findModuleFn(t, m.Functions(), "same")

	// Error: not a color for first arg
	_, err := colorEval(sameFn, colorNum(42), colorMust(value.NewColorRGB(255, 0, 0, 1)))
	assertErrMsg(t, err, "$color1: 42 is not a color.")

	// Error: not a color for second arg
	_, err = colorEval(sameFn, colorMust(value.NewColorRGB(255, 0, 0, 1)), colorNum(42))
	assertErrMsg(t, err, "$color2: 42 is not a color.")
}

// ============================================================================
// Test: Module inline function - is-powerless (error cases)
// ============================================================================

func TestColorModuleIsPowerless(t *testing.T) {
	m := ColorModule()
	isPowerlessFn := findModuleFn(t, m.Functions(), "is-powerless")

	// Error: not a color
	_, err := colorEval(isPowerlessFn, colorNum(42), colorQuoted("hue"))
	assertErrMsg(t, err, "$color: 42 is not a color.")

	// Error: channel not a string
	c := colorMust(value.NewColorRGB(255, 0, 0, 1))
	_, err = colorEval(isPowerlessFn, c, colorNum(1))
	assertErrMsg(t, err, "$channel: 1 is not a string.")

	// Error: channel unquoted
	_, err = colorEval(isPowerlessFn, c, colorStr("hue"))
	assertErrMsg(t, err, "$channel: Expected hue to be a quoted string.")
}

// ============================================================================
// Test: Module inline function - is-in-gamut (error case)
// ============================================================================

func TestColorModuleIsInGamut(t *testing.T) {
	m := ColorModule()
	isInGamutFn := findModuleFn(t, m.Functions(), "is-in-gamut")

	_, err := colorEval(isInGamutFn, colorNum(42))
	assertErrMsg(t, err, "$color: 42 is not a color.")
}

// ============================================================================
// Test: Removed color functions in module
// ============================================================================

func TestColorModuleRemovedFunctions(t *testing.T) {
	m := ColorModule()
	fns := m.Functions()

	tests := []struct {
		name    string
		wantMsg string
	}{
		{
			name: "adjust-hue",
			wantMsg: "The function adjust-hue() isn't in the sass:color module.\n\n" +
				"Recommendation: color.adjust(42, $hue: 10)\n\n" +
				"More info: https://sass-lang.com/documentation/functions/color#adjust-hue",
		},
		{
			name: "lighten",
			wantMsg: "The function lighten() isn't in the sass:color module.\n\n" +
				"Recommendation: color.adjust(42, $lightness: 10)\n\n" +
				"More info: https://sass-lang.com/documentation/functions/color#lighten",
		},
		{
			name: "saturate",
			wantMsg: "The function saturate() isn't in the sass:color module.\n\n" +
				"Recommendation: color.adjust(42, $saturation: 10)\n\n" +
				"More info: https://sass-lang.com/documentation/functions/color#saturate",
		},
		{
			name: "opacify",
			wantMsg: "The function opacify() isn't in the sass:color module.\n\n" +
				"Recommendation: color.adjust(42, $alpha: 10)\n\n" +
				"More info: https://sass-lang.com/documentation/functions/color#opacify",
		},
		{
			name: "fade-in",
			wantMsg: "The function fade-in() isn't in the sass:color module.\n\n" +
				"Recommendation: color.adjust(42, $alpha: 10)\n\n" +
				"More info: https://sass-lang.com/documentation/functions/color#fade-in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bic := findModuleFn(t, fns, tt.name)
			_, err := colorEval(bic, colorNum(42), colorNum(10))
			assertErrMsg(t, err, tt.wantMsg)
		})
	}

	darkenFn := findModuleFn(t, fns, "darken")
	_, err := colorEval(darkenFn, colorNum(42), colorNum(10))
	assertErrMsg(t, err, "The function darken() isn't in the sass:color module.\n\n"+
		"Recommendation: color.adjust(42, $lightness: -10)\n\n"+
		"More info: https://sass-lang.com/documentation/functions/color#darken")

	desaturateFn := findModuleFn(t, fns, "desaturate")
	_, err = colorEval(desaturateFn, colorNum(42), colorNum(10))
	assertErrMsg(t, err, "The function desaturate() isn't in the sass:color module.\n\n"+
		"Recommendation: color.adjust(42, $saturation: -10)\n\n"+
		"More info: https://sass-lang.com/documentation/functions/color#desaturate")

	transparentizeFn := findModuleFn(t, fns, "transparentize")
	_, err = colorEval(transparentizeFn, colorNum(42), colorNum(10))
	assertErrMsg(t, err, "The function transparentize() isn't in the sass:color module.\n\n"+
		"Recommendation: color.adjust(42, $alpha: -10)\n\n"+
		"More info: https://sass-lang.com/documentation/functions/color#transparentize")

	fadeOutFn := findModuleFn(t, fns, "fade-out")
	_, err = colorEval(fadeOutFn, colorNum(42), colorNum(10))
	assertErrMsg(t, err, "The function fade-out() isn't in the sass:color module.\n\n"+
		"Recommendation: color.adjust(42, $alpha: -10)\n\n"+
		"More info: https://sass-lang.com/documentation/functions/color#fade-out")
}

// ============================================================================
// Test: Happy path - inline functions that don't need colorInSpace
// ============================================================================

func TestColorModuleIsInGamutHappy(t *testing.T) {
	m := ColorModule()
	isInGamutFn := findModuleFn(t, m.Functions(), "is-in-gamut")

	c := colorMust(value.NewColorSRGB(0.5, 0.5, 0.5, 1.0))
	v, err := colorEval(isInGamutFn, c)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c2 := colorMust(value.NewColorXYZD65(0.5, 0.5, 0.5, 1.0))
	v, err = colorEval(isInGamutFn, c2)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	outGamut := colorMust(value.NewColorSRGB(1.5, 0.5, 0.5, 1.0))
	v, err = colorEval(isInGamutFn, outGamut)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)
}

func TestColorModuleIsPowerlessHappy(t *testing.T) {
	m := ColorModule()
	isPowerlessFn := findModuleFn(t, m.Functions(), "is-powerless")

	c := colorMust(value.NewColorHSL(120, 0, 50, 1))
	v, err := colorEval(isPowerlessFn, c, colorQuoted("hue"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c2 := colorMust(value.NewColorHSL(120, 100, 50, 1))
	v, err = colorEval(isPowerlessFn, c2, colorQuoted("hue"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	v, err = colorEval(isPowerlessFn, c, colorQuoted("saturation"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)

	lch := colorMust(value.NewColorLCH(50, 0, 200, 1))
	v, err = colorEval(isPowerlessFn, lch, colorQuoted("hue"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	v, err = colorEval(isPowerlessFn, c, colorQuoted("alpha"))
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)
}

func TestColorModuleSameHappy(t *testing.T) {
	m := ColorModule()
	sameFn := findModuleFn(t, m.Functions(), "same")

	c1 := colorMust(value.NewColorRGB(255, 0, 0, 1))
	c2 := colorMust(value.NewColorRGB(255, 0, 0, 1))
	v, err := colorEval(sameFn, c1, c2)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, true)

	c3 := colorMust(value.NewColorRGB(0, 255, 0, 1))
	v, err = colorEval(sameFn, c1, c3)
	if err != nil {
		t.Fatal(err)
	}
	checkBool(t, v, false)
}

func TestColorModuleToSpaceHappy(t *testing.T) {
	m := ColorModule()
	toSpaceFn := findModuleFn(t, m.Functions(), "to-space")

	c := colorMust(value.NewColorSRGB(0.5, 0, 0, 1))
	result, err := colorEval(toSpaceFn, c, colorStr("lab"))
	if err != nil {
		t.Fatal(err)
	}
	colorResult, ok := result.(*value.SassColor)
	if !ok {
		t.Fatalf("expected *SassColor, got %T", result)
	}
	if colorResult.Space() != value.LabColorSpace {
		t.Errorf("space = %v, want LabColorSpace", colorResult.Space())
	}
}

func TestColorModuleChannelHappy(t *testing.T) {
	m := ColorModule()
	channelFn := findModuleFn(t, m.Functions(), "channel")

	c := colorMust(value.NewColorRGB(128, 64, 32, 1))
	v, err := colorEval(channelFn, c, colorQuoted("red"))
	if err != nil {
		t.Fatal(err)
	}
	checkNum(t, v, 128)

	v, err = colorEval(channelFn, c, colorQuoted("alpha"))
	if err != nil {
		t.Fatal(err)
	}
	checkNum(t, v, 1)

	// Error: channel not found in color space
	c2 := colorMust(value.NewColorHSL(90, 100, 50, 1))
	_, err = colorEval(channelFn, c2, colorQuoted("red"))
	assertErrMsg(t, err, "$channel: Color "+colorString(c2)+" has no channel named red.")
}

func TestPercentageOrUnitlessErrorMessage(t *testing.T) {
	// Unitless and % pass through.
	v, err := percentageOrUnitless(value.NewUnitlessNumber(0.5), 255, "alpha")
	if err != nil || v != 0.5 {
		t.Errorf("percentageOrUnitless(0.5) = (%v, %v), want (0.5, nil)", v, err)
	}
	v, err = percentageOrUnitless(value.NewSingleUnitNumber(50, "%"), 255, "alpha")
	if err != nil || v != 127.5 {
		t.Errorf("percentageOrUnitless(50%%) = (%v, %v), want (127.5, nil)", v, err)
	}

	// Other units produce an exact error.
	_, err = percentageOrUnitless(value.NewSingleUnitNumber(0.5, "px"), 255, "alpha")
	if err == nil {
		t.Fatal("expected error")
	}
	if want := `$alpha: Expected 0.5px to have unit "%" or no units.`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
