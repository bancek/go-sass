// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/prophoto_rgb.dart

// prophotoRgbColorSpace is the ProPhoto RGB color space.
//
// Matches Dart: ProphotoRgbColorSpace
type prophotoRgbColorSpace struct{}

func (s *prophotoRgbColorSpace) Name() string               { return "prophoto-rgb" }
func (s *prophotoRgbColorSpace) Channels() [3]LinearChannel { return RGBChannels }
func (s *prophotoRgbColorSpace) IsBounded() bool            { return true }
func (s *prophotoRgbColorSpace) IsLegacy() bool             { return false }
func (s *prophotoRgbColorSpace) IsPolar() bool              { return false }

func (s *prophotoRgbColorSpace) ToLinear(ch float64) float64   { return prophotoToLinear(ch) }
func (s *prophotoRgbColorSpace) FromLinear(ch float64) float64 { return prophotoFromLinear(ch) }

func (s *prophotoRgbColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return convertLinear(ProphotoRgbColorSpace, dest, c0, c1, c2, alpha, nil)
}

func (s *prophotoRgbColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return linearProphotoRgbToLinearSrgb[:]
	case A98RgbColorSpace:
		return linearProphotoRgbToLinearA98Rgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return linearProphotoRgbToLinearDisplayP3[:]
	case Rec2020ColorSpace:
		return linearProphotoRgbToLinearRec2020[:]
	case XyzD65ColorSpace:
		return linearProphotoRgbToXyzD65[:]
	case XyzD50ColorSpace:
		return linearProphotoRgbToXyzD50[:]
	case LmsColorSpace:
		return linearProphotoRgbToLms[:]
	default:
		return nil
	}
}
