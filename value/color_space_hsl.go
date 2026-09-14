// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "math"

// dart-source: lib/src/value/color/space/hsl.dart

// HSLChannels are the channels for the HSL color space.
var HSLChannels = [3]LinearChannel{
	HueChannelInfo(),
	NewLinearChannel("saturation", 0, 100, WithRequiresPercent, WithLowerClamped),
	NewLinearChannel("lightness", 0, 100, WithRequiresPercent),
}

// hslColorSpace is the legacy HSL color space.
//
// Matches Dart: HslColorSpace
type hslColorSpace struct{}

func (s *hslColorSpace) Name() string               { return "hsl" }
func (s *hslColorSpace) Channels() [3]LinearChannel { return HSLChannels }
func (s *hslColorSpace) IsBounded() bool            { return true }
func (s *hslColorSpace) IsLegacy() bool             { return true }
func (s *hslColorSpace) IsPolar() bool              { return true }

func (s *hslColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *hslColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *hslColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *hslColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	// Algorithm from the CSS3 spec: https://www.w3.org/TR/css3-color/#hsl-color.
	scaledHue := 0.0
	if c0 != nil {
		scaledHue = math.Mod(*c0/360, 1)
	}
	scaledSaturation := 0.0
	if c1 != nil {
		scaledSaturation = *c1 / 100
	}
	scaledLightness := 0.0
	if c2 != nil {
		scaledLightness = *c2 / 100
	}

	var m2 float64
	if scaledLightness <= 0.5 {
		m2 = scaledLightness * (scaledSaturation + 1)
	} else {
		m2 = scaledLightness + scaledSaturation - float64(scaledLightness*scaledSaturation) // prevent FMA: see matrixMul
	}
	m1 := float64(scaledLightness*2) - m2 // prevent FMA: see matrixMul

	rv := hueToRgb(m1, m2, scaledHue+1.0/3.0)
	gv := hueToRgb(m1, m2, scaledHue)
	bv := hueToRgb(m1, m2, scaledHue-1.0/3.0)

	return srgbSpace.convertInternal(dest, &rv, &gv, &bv, alpha, &SrgbConvertOpts{
		MissingLightness: c2 == nil,
		MissingChroma:    c1 == nil,
		MissingHue:       c0 == nil,
	})
}
