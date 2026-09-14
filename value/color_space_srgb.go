// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"math"

	"github.com/bancek/go-sass/sassmath"
	"github.com/bancek/go-sass/util"
)

// dart-source: lib/src/value/color/space/srgb.dart

// SrgbConvertOpts carries optional parameters matching Dart's
// SrgbColorSpace.convert named parameters.
//
// Matches Dart: SrgbColorSpace.convert named params. Flags record which
// analogous sets arrived missing so HSL/HWB destinations can propagate
// missingness (#2810).
type SrgbConvertOpts struct {
	// MissingLightness records a missing source lightness set.
	MissingLightness bool
	// MissingChroma records a missing source chroma/saturation set.
	MissingChroma bool
	// MissingHue records a missing source hue.
	MissingHue bool
}

// srgbColorSpace is the sRGB color space.
//
// Matches Dart: SrgbColorSpace. Conversions to HSL/HWB go through the
// max/min/delta hue wheel with negative-saturation folding (+180 degrees);
// the rgb case bypasses validation via the raw forSpace path.
type srgbColorSpace struct{}

// srgbSpace is the internal singleton used for conversion routing.
//
// The exported SrgbColorSpace in color_space_base.go is the public handle.
var srgbSpace = &srgbColorSpace{}

func (s *srgbColorSpace) Name() string               { return "srgb" }
func (s *srgbColorSpace) Channels() [3]LinearChannel { return RGBChannels }
func (s *srgbColorSpace) IsBounded() bool            { return true }
func (s *srgbColorSpace) IsLegacy() bool             { return false }
func (s *srgbColorSpace) IsPolar() bool              { return false }

func (s *srgbColorSpace) ToLinear(ch float64) float64   { return srgbAndDisplayP3ToLinear(ch) }
func (s *srgbColorSpace) FromLinear(ch float64) float64 { return srgbAndDisplayP3FromLinear(ch) }

func (s *srgbColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return s.convertInternal(dest, c0, c1, c2, alpha, nil)
}

// Matches Dart: SrgbColorSpace.convert
func (s *srgbColorSpace) convertInternal(dest ColorSpace, red, green, blue, alpha *float64, opts *SrgbConvertOpts) (*SassColor, error) {
	// Analogous missingness (#2810): an all-missing source converts to an
	// all-missing destination, as does a source missing a whole analogous
	// lightness+chroma+hue set.
	missingLightness := opts != nil && opts.MissingLightness
	missingChroma := opts != nil && opts.MissingChroma
	missingHue := opts != nil && opts.MissingHue
	if (red == nil && green == nil && blue == nil) || (missingLightness && missingChroma && missingHue) {
		return newColorForSpaceInternalNoCheck(dest, nil, nil, nil, alpha), nil
	}

	switch dest {
	case HslColorSpace, HwbColorSpace:
		rv, gv, bv := 0.0, 0.0, 0.0
		if red != nil {
			rv = *red
		}
		if green != nil {
			gv = *green
		}
		if blue != nil {
			bv = *blue
		}
		if red == nil {
			rv = 0
		}
		if green == nil {
			gv = 0
		}
		if blue == nil {
			bv = 0
		}

		max := math.Max(rv, math.Max(gv, bv))
		min := math.Min(rv, math.Min(gv, bv))
		delta := max - min

		var hue float64
		if max == min {
			hue = 0
		} else if max == rv {
			hue = 60*(gv-bv)/delta + 360
		} else if max == gv {
			hue = 60*(bv-rv)/delta + 120
		} else {
			hue = 60*(rv-gv)/delta + 240
		}

		if dest == HslColorSpace {
			lightness := (min + max) / 2
			var saturation float64
			if lightness == 0 || lightness == 1 {
				saturation = 0
			} else {
				saturation = float64(100*(max-lightness)) / math.Min(lightness, float64(1-lightness)) // prevent FMA: see matrixMul
			}
			if saturation < 0 {
				hue += 180
				saturation = sassmath.Abs(saturation)
			}
			hue = math.Mod(hue, 360)

			var hPtr, sPtr, lPtr *float64
			if !(missingHue || util.FuzzyEquals(saturation, 0)) {
				hPtr = &hue
			}
			if !missingChroma {
				sPtr = &saturation
			}
			if !missingLightness {
				lp := lightness * 100
				lPtr = &lp
			}
			return newColorForSpaceInternalNoCheck(HslColorSpace, hPtr, sPtr, lPtr, alpha), nil
		}

		whiteness := min * 100
		blackness := 100 - float64(max*100) // prevent FMA: see matrixMul
		hue = math.Mod(hue, 360)
		hMissing := missingHue || util.FuzzyGreaterThanOrEquals(whiteness+blackness, 100)

		var hPtr *float64
		if !hMissing {
			hPtr = &hue
		}
		// Analogous missingness (#2810): whiteness/blackness are not
		// analogous to any channels, so they go missing only with the whole
		// lightness+chroma set.
		wbPtr := func(v float64) *float64 {
			if missingChroma && missingLightness {
				return nil
			}
			return &v
		}
		return newColorForSpaceInternalNoCheck(HwbColorSpace, hPtr, wbPtr(whiteness), wbPtr(blackness), alpha), nil

	case RgbColorSpace:
		var r255, g255, b255 *float64
		if red != nil {
			v := *red * 255
			r255 = &v
		}
		if green != nil {
			v := *green * 255
			g255 = &v
		}
		if blue != nil {
			v := *blue * 255
			b255 = &v
		}
		// Dart: `SassColor.rgb(...)` → `rgbInternal` → raw `_forSpace`
		// (bypasses forSpaceInternal's preprocessing).
		return newColorForSpaceNoCheck(dest, r255, g255, b255, alpha, nil), nil

	case SrgbLinearColorSpace:
		var rLin, gLin, bLin *float64
		if red != nil {
			v := srgbAndDisplayP3ToLinear(*red)
			rLin = &v
		}
		if green != nil {
			v := srgbAndDisplayP3ToLinear(*green)
			gLin = &v
		}
		if blue != nil {
			v := srgbAndDisplayP3ToLinear(*blue)
			bLin = &v
		}
		return newColorForSpaceInternalNoCheck(dest, rLin, gLin, bLin, alpha), nil

	default:
		var linearOpts *ConvertLinearOpts
		if opts != nil {
			linearOpts = &ConvertLinearOpts{
				MissingLightness: opts.MissingLightness,
				MissingChroma:    opts.MissingChroma,
				MissingHue:       opts.MissingHue,
			}
		}
		return convertLinear(SrgbColorSpace, dest, red, green, blue, alpha, linearOpts)
	}
}

func (s *srgbColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return linearSrgbToLinearDisplayP3[:]
	case A98RgbColorSpace:
		return linearSrgbToLinearA98Rgb[:]
	case ProphotoRgbColorSpace:
		return linearSrgbToLinearProphotoRgb[:]
	case Rec2020ColorSpace:
		return linearSrgbToLinearRec2020[:]
	case XyzD65ColorSpace:
		return linearSrgbToXyzD65[:]
	case XyzD50ColorSpace:
		return linearSrgbToXyzD50[:]
	case LmsColorSpace:
		return linearSrgbToLms[:]
	default:
		return nil
	}
}
