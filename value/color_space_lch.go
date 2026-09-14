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

// dart-source: lib/src/value/color/space/lch.dart

// LCHChannels are the channels for the LCH color space.
var LCHChannels = [3]LinearChannel{
	NewLinearChannel("lightness", 0, 100, WithLowerClamped, WithUpperClamped),
	NewLinearChannel("chroma", 0, 150, WithLowerClamped),
	HueChannelInfo(),
}

// lchColorSpace is the CIE LCH color space.
//
// Matches Dart: LchColorSpace
type lchColorSpace struct{}

func (s *lchColorSpace) Name() string               { return "lch" }
func (s *lchColorSpace) Channels() [3]LinearChannel { return LCHChannels }
func (s *lchColorSpace) IsBounded() bool            { return false }
func (s *lchColorSpace) IsLegacy() bool             { return false }
func (s *lchColorSpace) IsPolar() bool              { return true }

func (s *lchColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *lchColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *lchColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *lchColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
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
	//   const LabColorSpace().convert(dest, lightness,
	//     (chroma ?? 0) * math.cos(hueRadians),  // non-null double
	//     (chroma ?? 0) * math.sin(hueRadians),  // non-null double
	//     alpha);
	a := &av
	b := &bv
	return labSpace.convertInternal(dest, c0, a, b, alpha, &LabConvertOpts{
		MissingChroma: c1 == nil,
		MissingHue:    c2 == nil,
	})
}
