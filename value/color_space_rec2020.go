// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/rec2020.dart

// rec2020ColorSpace is the Rec.2020 color space.
//
// Matches Dart: Rec2020ColorSpace
type rec2020ColorSpace struct{}

func (s *rec2020ColorSpace) Name() string               { return "rec2020" }
func (s *rec2020ColorSpace) Channels() [3]LinearChannel { return RGBChannels }
func (s *rec2020ColorSpace) IsBounded() bool            { return true }
func (s *rec2020ColorSpace) IsLegacy() bool             { return false }
func (s *rec2020ColorSpace) IsPolar() bool              { return false }

func (s *rec2020ColorSpace) ToLinear(ch float64) float64   { return rec2020ToLinear(ch) }
func (s *rec2020ColorSpace) FromLinear(ch float64) float64 { return rec2020FromLinear(ch) }

func (s *rec2020ColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return convertLinear(Rec2020ColorSpace, dest, c0, c1, c2, alpha, nil)
}

func (s *rec2020ColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return linearRec2020ToLinearSrgb[:]
	case A98RgbColorSpace:
		return linearRec2020ToLinearA98Rgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return linearRec2020ToLinearDisplayP3[:]
	case ProphotoRgbColorSpace:
		return linearRec2020ToLinearProphotoRgb[:]
	case XyzD65ColorSpace:
		return linearRec2020ToXyzD65[:]
	case XyzD50ColorSpace:
		return linearRec2020ToXyzD50[:]
	case LmsColorSpace:
		return linearRec2020ToLms[:]
	default:
		return nil
	}
}
