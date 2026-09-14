// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/utils.dart

import (
	"math"

	"github.com/bancek/go-sass/sassmath"
	"github.com/bancek/go-sass/util"
)

// XYZChannels are the channels shared across both XYZ color spaces.
//
// Matches Dart: xyzChannels in space/utils.dart.
var XYZChannels = [3]LinearChannel{
	NewLinearChannel("x", 0, 1),
	NewLinearChannel("y", 0, 1),
	NewLinearChannel("z", 0, 1),
}

// Lab constants used in Lab/XYZ conversions.
//
// LabKappa (29^3/3^3) and LabEpsilon (6^3/29^3) gate the linear vs cube-root
// branches. Matches Dart: labKappa/labEpsilon in space/utils.dart.
const (
	LabKappa   = 24389.0 / 27.0
	LabEpsilon = 216.0 / 24389.0
)

// hueToRgb converts a legacy HSL/HWB hue to an RGB channel.
// Algorithm from the CSS3 spec: http://www.w3.org/TR/css3-color/#hsl-color.
//
// Hue wraps into 0-1, then the m1/m2 ramp selects the rising edge, plateau,
// falling edge, or floor. Matches Dart: hueToRgb in space/utils.dart.
func hueToRgb(m1, m2, hue float64) float64 {
	if hue < 0 {
		hue += 1
	}
	if hue > 1 {
		hue -= 1
	}

	switch {
	case hue < 1.0/6.0:
		return m1 + float64(float64((m2-m1)*hue)*6) // prevent FMA: see matrixMul
	case hue < 1.0/2.0:
		return m2
	case hue < 2.0/3.0:
		return m1 + float64(float64((m2-m1)*(2.0/3.0-hue))*6) // prevent FMA: see matrixMul
	default:
		return m1
	}
}

// srgbAndDisplayP3ToLinear converts a single sRGB or Display-P3 channel to linear-light form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code
func srgbAndDisplayP3ToLinear(channel float64) float64 {
	abs := sassmath.Abs(channel)
	if abs <= 0.04045 {
		return channel / 12.92
	}
	return math.Copysign(sassmath.Pow((abs+0.055)/1.055, 2.4), channel)
}

// srgbAndDisplayP3FromLinear converts a single linear-light channel to sRGB or Display-P3 gamma form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code
func srgbAndDisplayP3FromLinear(channel float64) float64 {
	abs := sassmath.Abs(channel)
	if abs <= 0.0031308 {
		return channel * 12.92
	}
	return math.Copysign(float64(1.055*sassmath.Pow(abs, 1.0/2.4))-0.055, channel) // prevent FMA: see matrixMul
}

// labToLch converts a Lab or OKLab color to LCH or OKLCH, respectively.
// Dart: calls SassColor.forSpaceInternal which validates alpha via
// fuzzyAssertRange. Alpha is always valid in practice (derived from
// existing SassColor), but we call the validating constructor for
// line-by-line structural parity.
//
// A zero chroma (or pre-existing missing hue) yields a missing hue, since
// hue is powerless without chroma. Matches Dart: labToLch.
func labToLch(dest ColorSpace, lightness, a, b *float64, alpha *float64, missingChroma, missingHue bool) (*SassColor, error) {
	// Analogous missingness: null a+b imply missing chroma+hue (#2810).
	missingChroma = missingChroma || (a == nil && b == nil)
	missingHue = missingHue || (a == nil && b == nil)

	av, bv := 0.0, 0.0
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	chroma := sassmath.Sqrt(float64(av*av) + float64(bv*bv)) // prevent FMA: see matrixMul
	var hue *float64
	if !missingHue && !util.FuzzyEquals(chroma, 0) {
		h := sassmath.Atan2(bv, av) * 180 / math.Pi
		if h >= 0 {
			hue = &h
		} else {
			h = h + 360
			hue = &h
		}
	}

	var chPtr *float64
	if !missingChroma {
		ch := chroma
		chPtr = &ch
	}
	return NewColorForSpaceInternal(dest, lightness, chPtr, hue, alpha)
}

// a98ToLinear converts a single A98-RGB channel to linear-light form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code
//
// A98 uses a plain 563/256 gamma with sign preservation for out-of-gamut
// negatives.
func a98ToLinear(channel float64) float64 {
	return math.Copysign(sassmath.Pow(sassmath.Abs(channel), 563.0/256.0), channel)
}

// a98FromLinear converts a single linear-light channel to A98-RGB form.
//
// Inverse of a98ToLinear with the 256/563 exponent.
func a98FromLinear(channel float64) float64 {
	return math.Copysign(sassmath.Pow(sassmath.Abs(channel), 256.0/563.0), channel)
}

// prophotoToLinear converts a single ProPhoto RGB channel to linear-light form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code
//
// Values at or below 16/512 stay linear (divided by 16); larger values use
// the 1.8 gamma.
func prophotoToLinear(channel float64) float64 {
	abs := sassmath.Abs(channel)
	if abs <= 16.0/512.0 {
		return channel / 16.0
	}
	return math.Copysign(sassmath.Pow(abs, 1.8), channel)
}

// prophotoFromLinear converts a single linear-light channel to ProPhoto RGB form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code
//
// Mirror of prophotoToLinear with the 1/512 threshold.
func prophotoFromLinear(channel float64) float64 {
	abs := sassmath.Abs(channel)
	if abs >= 1.0/512.0 {
		return math.Copysign(sassmath.Pow(abs, 1.0/1.8), channel)
	}
	return 16.0 * channel
}

// rec2020ToLinear converts a single Rec2020 channel to linear-light form.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-conversion-code.
// Since dart-sass 1.102 (#2729) this is the plain 2.4-gamma power law (the old
// alpha/beta piecewise form is gone).
func rec2020ToLinear(channel float64) float64 {
	return math.Copysign(sassmath.Pow(sassmath.Abs(channel), 2.40), channel)
}

// rec2020FromLinear converts a single linear-light channel to Rec2020 form.
func rec2020FromLinear(channel float64) float64 {
	return math.Copysign(sassmath.Pow(sassmath.Abs(channel), 1.0/2.40), channel)
}

// cubeRootPreservingSign returns the cube root of the absolute value with the same sign.
//
// Used for the LMS cube-root legs of OKLab/OKLCH conversion so negative
// out-of-gamut values keep their sign instead of producing NaN.
func cubeRootPreservingSign(v float64) float64 {
	return math.Copysign(sassmath.Pow(sassmath.Abs(v), 1.0/3.0), v)
}
