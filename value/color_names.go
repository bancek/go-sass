// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"math"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/util"
)

// dart-source: lib/src/color_names.dart

// namedColor holds one CSS named color as integer RGB plus alpha.
type namedColor struct {
	r, g, b int
	a       float64
}

// colorsByName maps lowercase CSS color names to their RGBA values.
// Matches Dart: colorsByName.
//
// Entries are in reverse alphabetical order (matching Dart's source) so
// that [namesByColor] resolving to the alphabetically-first name for
// duplicate RGB values by last-write-wins. Note Dart's map holds SassColor
// values; Go stores plain RGBA triples and rebuilds colors at lookup.
var colorsByName = orderedmap.NewFromPairs(
	orderedmap.Pair[string, namedColor]{Key: "yellowgreen", Val: namedColor{0x9A, 0xCD, 0x32, 1}},
	orderedmap.Pair[string, namedColor]{Key: "yellow", Val: namedColor{0xFF, 0xFF, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "whitesmoke", Val: namedColor{0xF5, 0xF5, 0xF5, 1}},
	orderedmap.Pair[string, namedColor]{Key: "white", Val: namedColor{0xFF, 0xFF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "wheat", Val: namedColor{0xF5, 0xDE, 0xB3, 1}},
	orderedmap.Pair[string, namedColor]{Key: "violet", Val: namedColor{0xEE, 0x82, 0xEE, 1}},
	orderedmap.Pair[string, namedColor]{Key: "turquoise", Val: namedColor{0x40, 0xE0, 0xD0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "transparent", Val: namedColor{0, 0, 0, 0}},
	orderedmap.Pair[string, namedColor]{Key: "tomato", Val: namedColor{0xFF, 0x63, 0x47, 1}},
	orderedmap.Pair[string, namedColor]{Key: "thistle", Val: namedColor{0xD8, 0xBF, 0xD8, 1}},
	orderedmap.Pair[string, namedColor]{Key: "teal", Val: namedColor{0x00, 0x80, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "tan", Val: namedColor{0xD2, 0xB4, 0x8C, 1}},
	orderedmap.Pair[string, namedColor]{Key: "steelblue", Val: namedColor{0x46, 0x82, 0xB4, 1}},
	orderedmap.Pair[string, namedColor]{Key: "springgreen", Val: namedColor{0x00, 0xFF, 0x7F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "snow", Val: namedColor{0xFF, 0xFA, 0xFA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "slategrey", Val: namedColor{0x70, 0x80, 0x90, 1}},
	orderedmap.Pair[string, namedColor]{Key: "slategray", Val: namedColor{0x70, 0x80, 0x90, 1}},
	orderedmap.Pair[string, namedColor]{Key: "slateblue", Val: namedColor{0x6A, 0x5A, 0xCD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "skyblue", Val: namedColor{0x87, 0xCE, 0xEB, 1}},
	orderedmap.Pair[string, namedColor]{Key: "silver", Val: namedColor{0xC0, 0xC0, 0xC0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "sienna", Val: namedColor{0xA0, 0x52, 0x2D, 1}},
	orderedmap.Pair[string, namedColor]{Key: "seashell", Val: namedColor{0xFF, 0xF5, 0xEE, 1}},
	orderedmap.Pair[string, namedColor]{Key: "seagreen", Val: namedColor{0x2E, 0x8B, 0x57, 1}},
	orderedmap.Pair[string, namedColor]{Key: "sandybrown", Val: namedColor{0xF4, 0xA4, 0x60, 1}},
	orderedmap.Pair[string, namedColor]{Key: "salmon", Val: namedColor{0xFA, 0x80, 0x72, 1}},
	orderedmap.Pair[string, namedColor]{Key: "saddlebrown", Val: namedColor{0x8B, 0x45, 0x13, 1}},
	orderedmap.Pair[string, namedColor]{Key: "royalblue", Val: namedColor{0x41, 0x69, 0xE1, 1}},
	orderedmap.Pair[string, namedColor]{Key: "rosybrown", Val: namedColor{0xBC, 0x8F, 0x8F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "red", Val: namedColor{0xFF, 0x00, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "rebeccapurple", Val: namedColor{0x66, 0x33, 0x99, 1}},
	orderedmap.Pair[string, namedColor]{Key: "purple", Val: namedColor{0x80, 0x00, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "powderblue", Val: namedColor{0xB0, 0xE0, 0xE6, 1}},
	orderedmap.Pair[string, namedColor]{Key: "plum", Val: namedColor{0xDD, 0xA0, 0xDD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "pink", Val: namedColor{0xFF, 0xC0, 0xCB, 1}},
	orderedmap.Pair[string, namedColor]{Key: "peru", Val: namedColor{0xCD, 0x85, 0x3F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "peachpuff", Val: namedColor{0xFF, 0xDA, 0xB9, 1}},
	orderedmap.Pair[string, namedColor]{Key: "papayawhip", Val: namedColor{0xFF, 0xEF, 0xD5, 1}},
	orderedmap.Pair[string, namedColor]{Key: "palevioletred", Val: namedColor{0xDB, 0x70, 0x93, 1}},
	orderedmap.Pair[string, namedColor]{Key: "paleturquoise", Val: namedColor{0xAF, 0xEE, 0xEE, 1}},
	orderedmap.Pair[string, namedColor]{Key: "palegreen", Val: namedColor{0x98, 0xFB, 0x98, 1}},
	orderedmap.Pair[string, namedColor]{Key: "palegoldenrod", Val: namedColor{0xEE, 0xE8, 0xAA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "orchid", Val: namedColor{0xDA, 0x70, 0xD6, 1}},
	orderedmap.Pair[string, namedColor]{Key: "orangered", Val: namedColor{0xFF, 0x45, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "orange", Val: namedColor{0xFF, 0xA5, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "olivedrab", Val: namedColor{0x6B, 0x8E, 0x23, 1}},
	orderedmap.Pair[string, namedColor]{Key: "olive", Val: namedColor{0x80, 0x80, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "oldlace", Val: namedColor{0xFD, 0xF5, 0xE6, 1}},
	orderedmap.Pair[string, namedColor]{Key: "navy", Val: namedColor{0x00, 0x00, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "navajowhite", Val: namedColor{0xFF, 0xDE, 0xAD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "moccasin", Val: namedColor{0xFF, 0xE4, 0xB5, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mistyrose", Val: namedColor{0xFF, 0xE4, 0xE1, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mintcream", Val: namedColor{0xF5, 0xFF, 0xFA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "midnightblue", Val: namedColor{0x19, 0x19, 0x70, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumvioletred", Val: namedColor{0xC7, 0x15, 0x85, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumturquoise", Val: namedColor{0x48, 0xD1, 0xCC, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumspringgreen", Val: namedColor{0x00, 0xFA, 0x9A, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumslateblue", Val: namedColor{0x7B, 0x68, 0xEE, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumseagreen", Val: namedColor{0x3C, 0xB3, 0x71, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumpurple", Val: namedColor{0x93, 0x70, 0xDB, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumorchid", Val: namedColor{0xBA, 0x55, 0xD3, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumblue", Val: namedColor{0x00, 0x00, 0xCD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "mediumaquamarine", Val: namedColor{0x66, 0xCD, 0xAA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "maroon", Val: namedColor{0x80, 0x00, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "magenta", Val: namedColor{0xFF, 0x00, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "linen", Val: namedColor{0xFA, 0xF0, 0xE6, 1}},
	orderedmap.Pair[string, namedColor]{Key: "limegreen", Val: namedColor{0x32, 0xCD, 0x32, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lime", Val: namedColor{0x00, 0xFF, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightyellow", Val: namedColor{0xFF, 0xFF, 0xE0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightsteelblue", Val: namedColor{0xB0, 0xC4, 0xDE, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightslategrey", Val: namedColor{0x77, 0x88, 0x99, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightslategray", Val: namedColor{0x77, 0x88, 0x99, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightskyblue", Val: namedColor{0x87, 0xCE, 0xFA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightseagreen", Val: namedColor{0x20, 0xB2, 0xAA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightsalmon", Val: namedColor{0xFF, 0xA0, 0x7A, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightpink", Val: namedColor{0xFF, 0xB6, 0xC1, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightgrey", Val: namedColor{0xD3, 0xD3, 0xD3, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightgreen", Val: namedColor{0x90, 0xEE, 0x90, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightgray", Val: namedColor{0xD3, 0xD3, 0xD3, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightgoldenrodyellow", Val: namedColor{0xFA, 0xFA, 0xD2, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightcyan", Val: namedColor{0xE0, 0xFF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightcoral", Val: namedColor{0xF0, 0x80, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lightblue", Val: namedColor{0xAD, 0xD8, 0xE6, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lemonchiffon", Val: namedColor{0xFF, 0xFA, 0xCD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lawngreen", Val: namedColor{0x7C, 0xFC, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lavenderblush", Val: namedColor{0xFF, 0xF0, 0xF5, 1}},
	orderedmap.Pair[string, namedColor]{Key: "lavender", Val: namedColor{0xE6, 0xE6, 0xFA, 1}},
	orderedmap.Pair[string, namedColor]{Key: "khaki", Val: namedColor{0xF0, 0xE6, 0x8C, 1}},
	orderedmap.Pair[string, namedColor]{Key: "ivory", Val: namedColor{0xFF, 0xFF, 0xF0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "indigo", Val: namedColor{0x4B, 0x00, 0x82, 1}},
	orderedmap.Pair[string, namedColor]{Key: "indianred", Val: namedColor{0xCD, 0x5C, 0x5C, 1}},
	orderedmap.Pair[string, namedColor]{Key: "hotpink", Val: namedColor{0xFF, 0x69, 0xB4, 1}},
	orderedmap.Pair[string, namedColor]{Key: "honeydew", Val: namedColor{0xF0, 0xFF, 0xF0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "grey", Val: namedColor{0x80, 0x80, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "greenyellow", Val: namedColor{0xAD, 0xFF, 0x2F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "green", Val: namedColor{0x00, 0x80, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "gray", Val: namedColor{0x80, 0x80, 0x80, 1}},
	orderedmap.Pair[string, namedColor]{Key: "goldenrod", Val: namedColor{0xDA, 0xA5, 0x20, 1}},
	orderedmap.Pair[string, namedColor]{Key: "gold", Val: namedColor{0xFF, 0xD7, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "ghostwhite", Val: namedColor{0xF8, 0xF8, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "gainsboro", Val: namedColor{0xDC, 0xDC, 0xDC, 1}},
	orderedmap.Pair[string, namedColor]{Key: "fuchsia", Val: namedColor{0xFF, 0x00, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "forestgreen", Val: namedColor{0x22, 0x8B, 0x22, 1}},
	orderedmap.Pair[string, namedColor]{Key: "floralwhite", Val: namedColor{0xFF, 0xFA, 0xF0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "firebrick", Val: namedColor{0xB2, 0x22, 0x22, 1}},
	orderedmap.Pair[string, namedColor]{Key: "dodgerblue", Val: namedColor{0x1E, 0x90, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "dimgrey", Val: namedColor{0x69, 0x69, 0x69, 1}},
	orderedmap.Pair[string, namedColor]{Key: "dimgray", Val: namedColor{0x69, 0x69, 0x69, 1}},
	orderedmap.Pair[string, namedColor]{Key: "deepskyblue", Val: namedColor{0x00, 0xBF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "deeppink", Val: namedColor{0xFF, 0x14, 0x93, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkviolet", Val: namedColor{0x94, 0x00, 0xD3, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkturquoise", Val: namedColor{0x00, 0xCE, 0xD1, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkslategrey", Val: namedColor{0x2F, 0x4F, 0x4F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkslategray", Val: namedColor{0x2F, 0x4F, 0x4F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkslateblue", Val: namedColor{0x48, 0x3D, 0x8B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkseagreen", Val: namedColor{0x8F, 0xBC, 0x8F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darksalmon", Val: namedColor{0xE9, 0x96, 0x7A, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkred", Val: namedColor{0x8B, 0x00, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkorchid", Val: namedColor{0x99, 0x32, 0xCC, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkorange", Val: namedColor{0xFF, 0x8C, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkolivegreen", Val: namedColor{0x55, 0x6B, 0x2F, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkmagenta", Val: namedColor{0x8B, 0x00, 0x8B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkkhaki", Val: namedColor{0xBD, 0xB7, 0x6B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkgrey", Val: namedColor{0xA9, 0xA9, 0xA9, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkgreen", Val: namedColor{0x00, 0x64, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkgray", Val: namedColor{0xA9, 0xA9, 0xA9, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkgoldenrod", Val: namedColor{0xB8, 0x86, 0x0B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkcyan", Val: namedColor{0x00, 0x8B, 0x8B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "darkblue", Val: namedColor{0x00, 0x00, 0x8B, 1}},
	orderedmap.Pair[string, namedColor]{Key: "cyan", Val: namedColor{0x00, 0xFF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "crimson", Val: namedColor{0xDC, 0x14, 0x3C, 1}},
	orderedmap.Pair[string, namedColor]{Key: "cornsilk", Val: namedColor{0xFF, 0xF8, 0xDC, 1}},
	orderedmap.Pair[string, namedColor]{Key: "cornflowerblue", Val: namedColor{0x64, 0x95, 0xED, 1}},
	orderedmap.Pair[string, namedColor]{Key: "coral", Val: namedColor{0xFF, 0x7F, 0x50, 1}},
	orderedmap.Pair[string, namedColor]{Key: "chocolate", Val: namedColor{0xD2, 0x69, 0x1E, 1}},
	orderedmap.Pair[string, namedColor]{Key: "chartreuse", Val: namedColor{0x7F, 0xFF, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "cadetblue", Val: namedColor{0x5F, 0x9E, 0xA0, 1}},
	orderedmap.Pair[string, namedColor]{Key: "burlywood", Val: namedColor{0xDE, 0xB8, 0x87, 1}},
	orderedmap.Pair[string, namedColor]{Key: "brown", Val: namedColor{0xA5, 0x2A, 0x2A, 1}},
	orderedmap.Pair[string, namedColor]{Key: "blueviolet", Val: namedColor{0x8A, 0x2B, 0xE2, 1}},
	orderedmap.Pair[string, namedColor]{Key: "blue", Val: namedColor{0x00, 0x00, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "blanchedalmond", Val: namedColor{0xFF, 0xEB, 0xCD, 1}},
	orderedmap.Pair[string, namedColor]{Key: "black", Val: namedColor{0x00, 0x00, 0x00, 1}},
	orderedmap.Pair[string, namedColor]{Key: "bisque", Val: namedColor{0xFF, 0xE4, 0xC4, 1}},
	orderedmap.Pair[string, namedColor]{Key: "beige", Val: namedColor{0xF5, 0xF5, 0xDC, 1}},
	orderedmap.Pair[string, namedColor]{Key: "azure", Val: namedColor{0xF0, 0xFF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "aquamarine", Val: namedColor{0x7F, 0xFF, 0xD4, 1}},
	orderedmap.Pair[string, namedColor]{Key: "aqua", Val: namedColor{0x00, 0xFF, 0xFF, 1}},
	orderedmap.Pair[string, namedColor]{Key: "antiquewhite", Val: namedColor{0xFA, 0xEB, 0xD7, 1}},
	orderedmap.Pair[string, namedColor]{Key: "aliceblue", Val: namedColor{0xF0, 0xF8, 0xFF, 1}},
)

// ---- Color name lookup ----

// rgbToName is the reverse index from integer RGB to the alphabetically
// first name for that triple, built last-write-wins from colorsByName order.
//
// Matches Dart: namesByColor collection-for.
var rgbToName map[[3]int]string

func init() {
	rgbToName = make(map[[3]int]string)
	// Iterate colorsByName in insertion order (reverse-alphabetical, matching
	// Dart's colorsByName.pairs). Last-write-wins so that alphabetically-first
	// names overwrite later duplicates — structurally matching Dart's
	// [namesByColor] collection-for.
	for name, c := range colorsByName.Entries() {
		key := [3]int{c.r, c.g, c.b}
		rgbToName[key] = name
	}
}

// colorNameFor is the unexported alias of ColorNameFor kept for call-site parity.
func colorNameFor(c *SassColor) (string, error) {
	return ColorNameFor(c)
}

// colorNameForSassColor resolves an in-space RGB color to its name without
// errors, returning "" for non-RGB spaces or non-integer channels. A black
// with transparent/missing alpha resolves to "transparent".
func colorNameForSassColor(rgb *SassColor) string {
	if rgb.Space() != RgbColorSpace {
		return ""
	}
	r, g, b := rgb.Channel0(), rgb.Channel1(), rgb.Channel2()
	if !canUseHexForChannel(r) || !canUseHexForChannel(g) || !canUseHexForChannel(b) {
		return ""
	}
	rInt := int(math.Round(r))
	gInt := int(math.Round(g))
	bInt := int(math.Round(b))
	name := rgbToName[[3]int{rInt, gInt, bInt}]
	if name != "" &&
		util.FuzzyEquals(r, float64(rInt)) &&
		util.FuzzyEquals(g, float64(gInt)) &&
		util.FuzzyEquals(b, float64(bInt)) {
		if name == "black" && (rgb.IsAlphaMissing() || util.FuzzyEquals(rgb.Alpha(), 0)) {
			return "transparent"
		}
		return name
	}
	return ""
}

// ColorNameFor returns the name of the color with the given SassColor, or ""
// if no named color matches.
//
// Matches Dart: namesByColor
func ColorNameFor(c *SassColor) (string, error) {
	r, err := c.Red()
	if err != nil {
		return "", err
	}
	g, err := c.Green()
	if err != nil {
		return "", err
	}
	b, err := c.Blue()
	if err != nil {
		return "", err
	}
	ri, gi, bi := int(math.Round(r)), int(math.Round(g)), int(math.Round(b))
	name := ColorNameForInts(ri, gi, bi)
	if name == "black" && (c.IsAlphaMissing() || util.FuzzyEquals(c.Alpha(), 0)) {
		return "transparent", nil
	}
	return name, nil
}

// ColorNameForInts returns the name of the color with the given RGB values, or
// "" if no named color matches.
//
// Matches Dart: namesByColor. Integer triples only; callers round float
// channels before lookup.
func ColorNameForInts(r, g, b int) string {
	return rgbToName[[3]int{r, g, b}]
}

// colorNameForInts is the unexported alias of ColorNameForInts.
func colorNameForInts(r, g, b int) string {
	return rgbToName[[3]int{r, g, b}]
}
