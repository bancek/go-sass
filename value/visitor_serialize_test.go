package value

import (
	"math"
	"testing"
)

func TestSerializeValue(t *testing.T) {
	spaceList, _ := NewSassList(
		[]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)},
		ListSeparatorSpace, false,
	)
	commaList, _ := NewSassList(
		[]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)},
		ListSeparatorComma, false,
	)
	slashList, _ := NewSassList(
		[]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)},
		ListSeparatorSlash, false,
	)
	bracketedList, _ := NewSassList(
		[]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)},
		ListSeparatorSpace, true,
	)
	singleCommaList, _ := NewSassList(
		[]Value{NewUnitlessNumber(42)},
		ListSeparatorComma, false,
	)
	singleSpaceList, _ := NewSassList(
		[]Value{NewUnitlessNumber(42)},
		ListSeparatorSpace, false,
	)

	tests := []struct {
		name  string
		val   Value
		quote bool
		want  string
	}{
		// Boolean
		{"true", SassTrue, false, "true"},
		{"true with quote", SassTrue, true, "true"},
		{"false", SassFalse, false, "false"},

		// Null
		{"null", Null, false, ""},

		// String
		{"quoted string with quotes", &SassString{Text: "hello", HasQuotes: true}, true, "\"hello\""},
		{"quoted string no quotes", &SassString{Text: "hello", HasQuotes: true}, false, "hello"},
		{"unquoted string with quotes", &SassString{Text: "hello", HasQuotes: false}, true, "hello"},
		{"unquoted string no quotes", &SassString{Text: "hello", HasQuotes: false}, false, "hello"},
		{"empty quoted string", &SassString{Text: "", HasQuotes: true}, true, "\"\""},
		{"empty unquoted string", &SassString{Text: "", HasQuotes: false}, false, ""},

		// Number
		{"integer 42", NewUnitlessNumber(42), false, "42"},
		{"float 3.14", NewUnitlessNumber(3.14), false, "3.14"},
		{"12px", NewSingleUnitNumber(12, "px"), false, "12px"},
		{"zero", NewUnitlessNumber(0), false, "0"},
		{"negative -5", NewUnitlessNumber(-5), false, "-5"},
		{"100%", NewSingleUnitNumber(100, "%"), false, "100%"},
		{"1.5em", NewSingleUnitNumber(1.5, "em"), false, "1.5em"},
		{"16px", NewSingleUnitNumber(16, "px"), false, "16px"},

		// Lists
		{"space list", spaceList, false, "1 2 3"},
		{"comma list", commaList, false, "1, 2, 3"},
		{"slash list", slashList, false, "1 / 2"},
		{"bracketed list", bracketedList, false, "[1 2]"},
		{"single comma list", singleCommaList, false, "42"},
		{"single space list", singleSpaceList, false, "42"},

		// Calculations
		{"calc(42px)", NewUnsimplified("calc", NewSingleUnitNumber(42, "px")), false, "calc(42px)"},
		{"min(1px, 2px, 3px)", NewUnsimplified("min",
			NewSingleUnitNumber(1, "px"), NewSingleUnitNumber(2, "px"), NewSingleUnitNumber(3, "px"),
		), false, "min(1px, 2px, 3px)"},
		{"max(10, 20)", NewUnsimplified("max",
			NewUnitlessNumber(10), NewUnitlessNumber(20),
		), false, "max(10, 20)"},
		{"clamp(0, 1, 10)", NewUnsimplified("clamp",
			NewUnitlessNumber(0), NewUnitlessNumber(1), NewUnitlessNumber(10),
		), false, "clamp(0, 1, 10)"},
	}

	// Color tests — these may differ slightly between implementations
	// due to color space conversion floating-point precision
	colorTests := []struct {
		name string
		c    *SassColor
		want string
	}{
		{"red", mustColor(NewColorRGB(255, 0, 0, 1)), "red"},
		{"lime", mustColor(NewColorRGB(0, 255, 0, 1)), "lime"},
		{"blue", mustColor(NewColorRGB(0, 0, 255, 1)), "blue"},
		{"rgba(255,0,0,0.5)", mustColor(NewColorRGB(255, 0, 0, 0.5)), "rgba(255, 0, 0, 0.5)"},
		{"gray 50%", mustColor(NewColorRGB(128, 128, 128, 1)), "gray"},
		{"black", mustColor(NewColorRGB(0, 0, 0, 1)), "black"},
		{"white", mustColor(NewColorRGB(255, 255, 255, 1)), "white"},
		{"hsl(0, 100%, 50%)", mustColor(NewColorHSL(0, 100, 50, 1)), "hsl(0, 100%, 50%)"},
		{"hsla(120,100%,50%,0.5)", mustColor(NewColorHSL(120, 100, 50, 0.5)), "hsla(120, 100%, 50%, 0.5)"},
		{"hsl(180,50%,25%)", mustColor(NewColorHSL(180, 50, 25, 1)), "hsl(180, 50%, 25%)"},
		{"hwb(0,0%,0%)", mustColor(NewColorHWB(0, 0, 0, 1)), "red"},
		{"lab(50% 25 -25)", mustColor(NewColorLab(50, 25, -25, 1)), "lab(50% 25 -25)"},
		{"lch(50% 30 200deg)", mustColor(NewColorLCH(50, 30, 200, 1)), "lch(50% 30 200deg)"},
		{"oklab(0.5 0.1 -0.1)", mustColor(NewColorOKLab(0.5, 0.1, -0.1, 1)), "oklab(50% 0.1 -0.1)"},
		{"oklch(0.5 0.15 180deg)", mustColor(NewColorOKLCH(0.5, 0.15, 180, 1)), "oklch(50% 0.15 180deg)"},
		{"srgb(0.5 0.5 0.5)", mustColor(NewColorSRGB(0.5, 0.5, 0.5, 1)), "color(srgb 0.5 0.5 0.5)"},
		{"display-p3(0.5 0.5 0.5)", mustColor(NewColorDisplayP3(0.5, 0.5, 0.5, 1)), "color(display-p3 0.5 0.5 0.5)"},
		{"xyz(0.5 0.5 0.5)", mustColor(NewColorXYZD65(0.5, 0.5, 0.5, 1)), "color(xyz 0.5 0.5 0.5)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeValue(tt.val, tt.quote)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("SerializeValue(%v, %v) = %q, want %q", tt.val, tt.quote, got, tt.want)
			}
		})
	}
	for _, tt := range colorTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeValue(tt.c, true)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("SerializeValue(color) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSerializeValue_EmptyList(t *testing.T) {
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)
	_, err := SerializeValue(emptyList, false)
	if err == nil {
		t.Error("expected error for empty unbracketed list, got nil")
	}
}

func TestSerializeValue_EmptyBracketedList(t *testing.T) {
	emptyBracketedList, _ := NewSassList(nil, ListSeparatorSpace, true)
	got, err := SerializeValue(emptyBracketedList, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Errorf("expected %q, got %q", "[]", got)
	}
}

func TestSerializeValue_MissingChannels(t *testing.T) {
	zero := new(float64)
	fifty := 50.0
	hundred := 100.0
	one := 1.0

	emptyBracketedList, _ := NewSassList(nil, ListSeparatorSpace, true)

	tests := []struct {
		name string
		val  Value
		want string
	}{
		// RGB with missing channels (modern space-separated syntax)
		{"rgb green missing",
			mustColor(NewColorRGBInternal(new(float64(255)), nil, new(float64(0)), new(1.0), nil)),
			"rgb(255 none 0)"},
		{"rgb blue missing",
			mustColor(NewColorRGBInternal(new(float64(255)), new(float64(0)), nil, new(1.0), nil)),
			"rgb(255 0 none)"},
		{"rgb red missing",
			mustColor(NewColorRGBInternal(nil, new(float64(0)), new(float64(0)), new(1.0), nil)),
			"rgb(none 0 0)"},

		// HSL with missing channels
		{"hsl saturation missing",
			mustColor(NewColorForSpaceInternal(HslColorSpace, zero, nil, &fifty, &one)),
			"hsl(0deg none 50%)"},
		{"hsl lightness missing",
			mustColor(NewColorForSpaceInternal(HslColorSpace, zero, &hundred, nil, &one)),
			"hsl(0deg 100% none)"},

		// HWB with missing channels
		{"hwb whiteness missing",
			mustColor(NewColorForSpaceInternal(HwbColorSpace, zero, nil, zero, &one)),
			"hwb(0deg none 0%)"},

		// Modern spaces
		{"lab", mustColor(NewColorLab(50, 25, -25, 1)), "lab(50% 25 -25)"},
		{"lch", mustColor(NewColorLCH(50, 30, 200, 1)), "lch(50% 30 200deg)"},
		{"oklab", mustColor(NewColorOKLab(0.5, 0.1, -0.1, 1)), "oklab(50% 0.1 -0.1)"},
		{"oklch", mustColor(NewColorOKLCH(0.5, 0.15, 180, 1)), "oklch(50% 0.15 180deg)"},

		// Special number values
		{"NaN number", NewUnitlessNumber(math.NaN()), "calc(NaN)"},
		{"+Inf number", NewUnitlessNumber(math.Inf(1)), "calc(infinity)"},
		{"-Inf number", NewUnitlessNumber(math.Inf(-1)), "calc(-infinity)"},
		{"NaN with unit px", NewSingleUnitNumber(math.NaN(), "px"), "calc(NaN * 1px)"},

		// Null
		{"null", Null, ""},

		// Booleans with quote=true
		{"true with quote", SassTrue, "true"},
		{"false with quote", SassFalse, "false"},

		// Empty bracketed list
		{"empty bracketed list", emptyBracketedList, "[]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeValue(tt.val, true)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("SerializeValue(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestSerializeValueInspect(t *testing.T) {
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)

	tests := []struct {
		name string
		val  Value
		want string
	}{
		// Boolean
		{"true", SassTrue, "true"},
		{"false", SassFalse, "false"},

		// Null
		{"null", Null, "null"},

		// String
		{"quoted string", &SassString{Text: "hello", HasQuotes: true}, "\"hello\""},

		// List
		{"empty list", emptyList, "()"},

		// Map
		{"simple map", NewSassMap(map[Value]Value{&SassString{Text: "a", HasQuotes: true}: NewUnitlessNumber(1)}), "(\"a\": 1)"},
	}

	colorTests := []struct {
		name string
		c    *SassColor
		want string
	}{
		{"red", mustColor(NewColorRGB(255, 0, 0, 1)), "red"},
		{"rgba(255,0,0,0.5)", mustColor(NewColorRGB(255, 0, 0, 0.5)), "rgba(255, 0, 0, 0.5)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeValueInspect(tt.val)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("SerializeValueInspect(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
	for _, tt := range colorTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeValueInspect(tt.c)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("SerializeValueInspect(color) = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustColor(c *SassColor, err error) *SassColor {
	if err != nil {
		panic(err)
	}
	return c
}
