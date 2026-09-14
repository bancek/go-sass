// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ColorSpace represents a color space whose channel names and semantics Sass knows.
//
// Matches Dart: ColorSpace
type ColorSpace interface {
	// Name returns the CSS name of the color space (e.g., "rgb", "srgb-linear").
	Name() string

	// Channels returns the 3 channel descriptors for this space.
	Channels() [3]LinearChannel

	// IsBounded returns whether this space has a bounded gamut.
	IsBounded() bool

	// IsLegacy returns whether this is a legacy color space (rgb, hsl, hwb).
	IsLegacy() bool

	// IsPolar returns whether this space uses a polar coordinate system.
	IsPolar() bool

	// ToLinear converts a channel value in this space to linear form.
	ToLinear(channel float64) float64

	// FromLinear converts a linear channel value to this space's representation.
	FromLinear(channel float64) float64

	// Convert converts a color from this space to dest.
	Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error)

	// TransformationMatrix returns the 3×3 matrix for linear transform from
	// this space to dest. Returns nil if conversion is handled explicitly in Convert.
	TransformationMatrix(dest ColorSpace) []float64
}

// --- Singleton instances (matching Dart's static const ColorSpace instances) ---

// The legacy RGB color space.
var RgbColorSpace ColorSpace = &rgbColorSpace{}

// The legacy HSL color space.
var HslColorSpace ColorSpace = &hslColorSpace{}

// The legacy HWB color space.
var HwbColorSpace ColorSpace = &hwbColorSpace{}

// The sRGB color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-sRGB
var SrgbColorSpace ColorSpace = &srgbColorSpace{}

// The linear-light sRGB color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-sRGB-linear
var SrgbLinearColorSpace ColorSpace = &srgbLinearColorSpace{}

// The display-p3 color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-display-p3
var DisplayP3ColorSpace ColorSpace = &displayP3ColorSpace{}

// The display-p3-linear color space.
//
// https://drafts.csswg.org/css-color/#predefined-display-p3-linear
var DisplayP3LinearColorSpace ColorSpace = &displayP3LinearColorSpace{}

// The a98-rgb color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-a98-rgb
var A98RgbColorSpace ColorSpace = &a98RgbColorSpace{}

// The prophoto-rgb color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-prophoto-rgb
var ProphotoRgbColorSpace ColorSpace = &prophotoRgbColorSpace{}

// The rec2020 color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-rec2020
var Rec2020ColorSpace ColorSpace = &rec2020ColorSpace{}

// The xyz-d65 color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-xyz
var XyzD65ColorSpace ColorSpace = &xyzD65ColorSpace{}

// The xyz-d50 color space.
//
// https://www.w3.org/TR/css-color-4/#predefined-xyz
var XyzD50ColorSpace ColorSpace = &xyzD50ColorSpace{}

// The CIE Lab color space.
//
// https://www.w3.org/TR/css-color-4/#cie-lab
var LabColorSpace ColorSpace = &labColorSpace{}

// The CIE LCH color space.
//
// https://www.w3.org/TR/css-color-4/#cie-lab
var LchColorSpace ColorSpace = &lchColorSpace{}

// The Oklab color space.
//
// https://www.w3.org/TR/css-color-4/#ok-lab
var OklabColorSpace ColorSpace = &oklabColorSpace{}

// The Oklch color space.
//
// https://www.w3.org/TR/css-color-4/#ok-lab
var OklchColorSpace ColorSpace = &oklchColorSpace{}

// The internal LMS color space.
//
// This only used as an intermediate space for conversions to and from OKLab
// and OKLCH. It's never used in a real color value and isn't returned by
// ColorSpaceFromName.
var LmsColorSpace ColorSpace = &lmsColorSpace{}

// --- Registry (matching Dart's ColorSpace.fromName) ---

// colorSpaceByName maps CSS color space names to their ColorSpace instances.
var colorSpaceByName = map[string]ColorSpace{
	"rgb":               RgbColorSpace,
	"hsl":               HslColorSpace,
	"hwb":               HwbColorSpace,
	"srgb":              SrgbColorSpace,
	"srgb-linear":       SrgbLinearColorSpace,
	"display-p3":        DisplayP3ColorSpace,
	"display-p3-linear": DisplayP3LinearColorSpace,
	"a98-rgb":           A98RgbColorSpace,
	"prophoto-rgb":      ProphotoRgbColorSpace,
	"rec2020":           Rec2020ColorSpace,
	"xyz":               XyzD65ColorSpace,
	"xyz-d65":           XyzD65ColorSpace,
	"xyz-d50":           XyzD50ColorSpace,
	"lab":               LabColorSpace,
	"lch":               LchColorSpace,
	"oklab":             OklabColorSpace,
	"oklch":             OklchColorSpace,
}

// ColorSpaceFromName returns the ColorSpace associated with the given CSS
// name, case-insensitively.
//
// Matches Dart: ColorSpace.fromName
func ColorSpaceFromName(name string, argumentName *string) (ColorSpace, error) {
	if cs, ok := colorSpaceByName[strings.ToLower(name)]; ok {
		return cs, nil
	}
	return nil, sasscommon.NewSassScriptException("Unknown color space \""+name+"\".", argumentName)
}
