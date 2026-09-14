// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"math"

	"github.com/bancek/go-sass/sassmath"
)

// dart-source: lib/src/value/color/space/oklch.dart

// OKLCHChannels are the channels for the OKLCH color space.
var OKLCHChannels = [3]LinearChannel{
	NewLinearChannel("lightness", 0, 1, WithLowerClamped, WithUpperClamped, WithConventionallyPercent),
	NewLinearChannel("chroma", 0, 0.4, WithLowerClamped),
	HueChannelInfo(),
}

// oklchColorSpace is the OKLCH color space.
//
// Matches Dart: OklchColorSpace
type oklchColorSpace struct{}

func (s *oklchColorSpace) Name() string               { return "oklch" }
func (s *oklchColorSpace) Channels() [3]LinearChannel { return OKLCHChannels }
func (s *oklchColorSpace) IsBounded() bool            { return false }
func (s *oklchColorSpace) IsLegacy() bool             { return false }
func (s *oklchColorSpace) IsPolar() bool              { return true }

func (s *oklchColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *oklchColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *oklchColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *oklchColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	chv, hv := 0.0, 0.0
	if c1 != nil {
		chv = *c1
	}
	if c2 != nil {
		hv = *c2
	}
	hueRadians := hv * math.Pi / 180
	av := chv * sassmath.Cos(hueRadians)
	bv := chv * sassmath.Sin(hueRadians)

	// Porting Dart bug: Dart's Convert always computes a/b as non-null
	// doubles even when both chroma and hue are null:
	//   var hueRadians = (hue ?? 0) * math.pi / 180;
	//   const OklabColorSpace().convert(dest, lightness,
	//     (chroma ?? 0) * math.cos(hueRadians),
	//     (chroma ?? 0) * math.sin(hueRadians),
	//     alpha);
	a := &av
	b := &bv
	return oklabSpace.convertInternal(dest, c0, a, b, alpha, &OklabConvertOpts{
		MissingChroma: c1 == nil,
		MissingHue:    c2 == nil,
	})
}
