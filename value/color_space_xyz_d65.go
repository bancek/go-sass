// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/xyz_d65.dart

// xyzD65ColorSpace is the CIE XYZ D65 color space.
//
// Matches Dart: XyzD65ColorSpace
type xyzD65ColorSpace struct{}

func (s *xyzD65ColorSpace) Name() string                  { return "xyz" }
func (s *xyzD65ColorSpace) Channels() [3]LinearChannel    { return XYZChannels }
func (s *xyzD65ColorSpace) IsBounded() bool               { return false }
func (s *xyzD65ColorSpace) IsLegacy() bool                { return false }
func (s *xyzD65ColorSpace) IsPolar() bool                 { return false }
func (s *xyzD65ColorSpace) ToLinear(ch float64) float64   { return ch }
func (s *xyzD65ColorSpace) FromLinear(ch float64) float64 { return ch }

func (s *xyzD65ColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return convertLinear(XyzD65ColorSpace, dest, c0, c1, c2, alpha, nil)
}

func (s *xyzD65ColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return xyzD65ToLinearSrgb[:]
	case A98RgbColorSpace:
		return xyzD65ToLinearA98Rgb[:]
	case ProphotoRgbColorSpace:
		return xyzD65ToLinearProphotoRgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return xyzD65ToLinearDisplayP3[:]
	case Rec2020ColorSpace:
		return xyzD65ToLinearRec2020[:]
	case XyzD50ColorSpace:
		return xyzD65ToXyzD50[:]
	case LmsColorSpace:
		return xyzD65ToLms[:]
	default:
		return nil
	}
}
