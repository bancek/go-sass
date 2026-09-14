// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "math"

// dart-source: lib/src/value/color/space/hwb.dart

// HWBChannels are the channels for the HWB color space.
var HWBChannels = [3]LinearChannel{
	HueChannelInfo(),
	NewLinearChannel("whiteness", 0, 100, WithRequiresPercent),
	NewLinearChannel("blackness", 0, 100, WithRequiresPercent),
}

// hwbColorSpace is the legacy HWB color space.
//
// Matches Dart: HwbColorSpace
type hwbColorSpace struct{}

func (s *hwbColorSpace) Name() string               { return "hwb" }
func (s *hwbColorSpace) Channels() [3]LinearChannel { return HWBChannels }
func (s *hwbColorSpace) IsBounded() bool            { return true }
func (s *hwbColorSpace) IsLegacy() bool             { return true }
func (s *hwbColorSpace) IsPolar() bool              { return true }

func (s *hwbColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *hwbColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *hwbColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *hwbColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	// Analogous missingness (#2810): whiteness/blackness are not analogous to
	// any channels, so handle a whiteness+blackness-missing source manually
	// rather than piping those flags everywhere.
	if c1 == nil && c2 == nil {
		if c0 == nil {
			return newColorForSpaceInternalNoCheck(dest, nil, nil, nil, alpha), nil
		}
		zero := 0.0
		converted, err := s.Convert(dest, c0, &zero, &zero, alpha)
		if err != nil {
			return nil, err
		}
		switch dest {
		case HslColorSpace:
			return newColorForSpaceInternalNoCheck(dest, &converted.channel0, nil, nil, &converted.alpha), nil
		case LchColorSpace, OklchColorSpace:
			return newColorForSpaceInternalNoCheck(dest, nil, nil, &converted.channel2, &converted.alpha), nil
		default:
			return converted, nil
		}
	}

	// From https://www.w3.org/TR/css-color-4/#hwb-to-rgb
	scaledHue := 0.0
	if c0 != nil {
		scaledHue = math.Mod(*c0, 360) / 360
	}
	scaledWhiteness := 0.0
	if c1 != nil {
		scaledWhiteness = *c1 / 100
	}
	scaledBlackness := 0.0
	if c2 != nil {
		scaledBlackness = *c2 / 100
	}

	sum := scaledWhiteness + scaledBlackness
	if sum > 1 {
		scaledWhiteness /= sum
		scaledBlackness /= sum
	}

	factor := 1 - scaledWhiteness - scaledBlackness
	toRgb := func(hue float64) float64 {
		return float64(hueToRgb(0, 1, hue)*factor) + scaledWhiteness // prevent FMA: see matrixMul
	}

	rv := toRgb(scaledHue + 1.0/3.0)
	gv := toRgb(scaledHue)
	bv := toRgb(scaledHue - 1.0/3.0)

	return srgbSpace.convertInternal(dest, &rv, &gv, &bv, alpha, &SrgbConvertOpts{
		MissingHue: c0 == nil,
	})
}
