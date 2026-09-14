// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/a98_rgb.dart

// a98RgbColorSpace is the A98 RGB color space.
//
// Matches Dart: A98RgbColorSpace
type a98RgbColorSpace struct{}

func (s *a98RgbColorSpace) Name() string               { return "a98-rgb" }
func (s *a98RgbColorSpace) Channels() [3]LinearChannel { return RGBChannels }
func (s *a98RgbColorSpace) IsBounded() bool            { return true }
func (s *a98RgbColorSpace) IsLegacy() bool             { return false }
func (s *a98RgbColorSpace) IsPolar() bool              { return false }

func (s *a98RgbColorSpace) ToLinear(ch float64) float64   { return a98ToLinear(ch) }
func (s *a98RgbColorSpace) FromLinear(ch float64) float64 { return a98FromLinear(ch) }

func (s *a98RgbColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return convertLinear(A98RgbColorSpace, dest, c0, c1, c2, alpha, nil)
}

func (s *a98RgbColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return linearA98RgbToLinearSrgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return linearA98RgbToLinearDisplayP3[:]
	case ProphotoRgbColorSpace:
		return linearA98RgbToLinearProphotoRgb[:]
	case Rec2020ColorSpace:
		return linearA98RgbToLinearRec2020[:]
	case XyzD65ColorSpace:
		return linearA98RgbToXyzD65[:]
	case XyzD50ColorSpace:
		return linearA98RgbToXyzD50[:]
	case LmsColorSpace:
		return linearA98RgbToLms[:]
	default:
		return nil
	}
}
