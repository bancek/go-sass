// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/display_p3.dart

// displayP3ColorSpace is the Display P3 color space.
//
// Matches Dart: DisplayP3ColorSpace
type displayP3ColorSpace struct{}

func (s *displayP3ColorSpace) Name() string               { return "display-p3" }
func (s *displayP3ColorSpace) Channels() [3]LinearChannel { return RGBChannels }
func (s *displayP3ColorSpace) IsBounded() bool            { return true }
func (s *displayP3ColorSpace) IsLegacy() bool             { return false }
func (s *displayP3ColorSpace) IsPolar() bool              { return false }

func (s *displayP3ColorSpace) ToLinear(ch float64) float64   { return srgbAndDisplayP3ToLinear(ch) }
func (s *displayP3ColorSpace) FromLinear(ch float64) float64 { return srgbAndDisplayP3FromLinear(ch) }

func (s *displayP3ColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	if dest == DisplayP3LinearColorSpace {
		var rLin, gLin, bLin *float64
		if c0 != nil {
			v := srgbAndDisplayP3ToLinear(*c0)
			rLin = &v
		}
		if c1 != nil {
			v := srgbAndDisplayP3ToLinear(*c1)
			gLin = &v
		}
		if c2 != nil {
			v := srgbAndDisplayP3ToLinear(*c2)
			bLin = &v
		}
		return newColorForSpaceInternalNoCheck(dest, rLin, gLin, bLin, alpha), nil
	}
	return convertLinear(DisplayP3ColorSpace, dest, c0, c1, c2, alpha, nil)
}

func (s *displayP3ColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return linearDisplayP3ToLinearSrgb[:]
	case A98RgbColorSpace:
		return linearDisplayP3ToLinearA98Rgb[:]
	case ProphotoRgbColorSpace:
		return linearDisplayP3ToLinearProphotoRgb[:]
	case Rec2020ColorSpace:
		return linearDisplayP3ToLinearRec2020[:]
	case XyzD65ColorSpace:
		return linearDisplayP3ToXyzD65[:]
	case XyzD50ColorSpace:
		return linearDisplayP3ToXyzD50[:]
	case LmsColorSpace:
		return linearDisplayP3ToLms[:]
	default:
		return nil
	}
}
