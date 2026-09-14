// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/srgb_linear.dart

// srgbLinearColorSpace is the linear-light sRGB color space.
//
// Matches Dart: SrgbLinearColorSpace
type srgbLinearColorSpace struct{}

func (s *srgbLinearColorSpace) Name() string                  { return "srgb-linear" }
func (s *srgbLinearColorSpace) Channels() [3]LinearChannel    { return RGBChannels }
func (s *srgbLinearColorSpace) IsBounded() bool               { return true }
func (s *srgbLinearColorSpace) IsLegacy() bool                { return false }
func (s *srgbLinearColorSpace) IsPolar() bool                 { return false }
func (s *srgbLinearColorSpace) ToLinear(ch float64) float64   { return ch }
func (s *srgbLinearColorSpace) FromLinear(ch float64) float64 { return ch }

func (s *srgbLinearColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	switch dest {
	case SrgbColorSpace, RgbColorSpace, HslColorSpace, HwbColorSpace:
		var rLin, gLin, bLin *float64
		if c0 != nil {
			v := srgbAndDisplayP3FromLinear(*c0)
			rLin = &v
		}
		if c1 != nil {
			v := srgbAndDisplayP3FromLinear(*c1)
			gLin = &v
		}
		if c2 != nil {
			v := srgbAndDisplayP3FromLinear(*c2)
			bLin = &v
		}
		return srgbSpace.convertInternal(dest, rLin, gLin, bLin, alpha, nil)
	default:
		return convertLinear(SrgbLinearColorSpace, dest, c0, c1, c2, alpha, nil)
	}
}

func (s *srgbLinearColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
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
