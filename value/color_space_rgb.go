// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/rgb.dart

// LegacyRGBChannels are the channels for the legacy RGB space (0-255 range).
var LegacyRGBChannels = [3]LinearChannel{
	NewLinearChannel("red", 0, 255, WithLowerClamped, WithUpperClamped),
	NewLinearChannel("green", 0, 255, WithLowerClamped, WithUpperClamped),
	NewLinearChannel("blue", 0, 255, WithLowerClamped, WithUpperClamped),
}

// RGBChannels are the channels shared across all RGB-derived color spaces.
var RGBChannels = [3]LinearChannel{
	NewLinearChannel("red", 0, 1),
	NewLinearChannel("green", 0, 1),
	NewLinearChannel("blue", 0, 1),
}

// rgbColorSpace is the legacy RGB color space.
//
// Matches Dart: RgbColorSpace
type rgbColorSpace struct{}

func (s *rgbColorSpace) Name() string                  { return "rgb" }
func (s *rgbColorSpace) Channels() [3]LinearChannel    { return LegacyRGBChannels }
func (s *rgbColorSpace) IsBounded() bool               { return true }
func (s *rgbColorSpace) IsLegacy() bool                { return true }
func (s *rgbColorSpace) IsPolar() bool                 { return false }
func (s *rgbColorSpace) ToLinear(ch float64) float64   { return srgbAndDisplayP3ToLinear(ch / 255) }
func (s *rgbColorSpace) FromLinear(ch float64) float64 { return srgbAndDisplayP3FromLinear(ch) * 255 }

func (s *rgbColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	if dest == RgbColorSpace {
		// Dart: RgbColorSpace.convert delegates to SrgbColorSpace.convert,
		// whose `.rgb` case calls SassColor.rgb (raw `_forSpace`).
		return newColorForSpaceNoCheck(dest, c0, c1, c2, alpha, nil), nil
	}
	var r, g, b *float64
	if c0 != nil {
		v := *c0 / 255
		r = &v
	}
	if c1 != nil {
		v := *c1 / 255
		g = &v
	}
	if c2 != nil {
		v := *c2 / 255
		b = &v
	}
	return srgbSpace.convertInternal(dest, r, g, b, alpha, nil)
}

func (s *rgbColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }
