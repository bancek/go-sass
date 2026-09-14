// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/xyz_d50.dart

// XyzD50ConvertOpts carries optional parameters matching Dart's
// XyzD50ColorSpace.convert named parameters.
//
// Matches Dart: XyzD50ColorSpace.convert named params
type XyzD50ConvertOpts struct {
	// MissingLightness records a missing source lightness set.
	MissingLightness bool
	// MissingChroma records a missing source chroma set.
	MissingChroma bool
	// MissingHue records a missing source hue.
	MissingHue bool
	// MissingA records a missing source a channel.
	MissingA bool
	// MissingB records a missing source b channel.
	MissingB bool
}

// xyzD50Space is the internal singleton used for conversion routing.
//
// The exported XyzD50ColorSpace in color_space_base.go is the public handle.
var xyzD50Space = &xyzD50ColorSpace{}

// xyzD50ColorSpace is the CIE XYZ D50 color space.
//
// Matches Dart: XyzD50ColorSpace
type xyzD50ColorSpace struct{}

func (s *xyzD50ColorSpace) Name() string                  { return "xyz-d50" }
func (s *xyzD50ColorSpace) Channels() [3]LinearChannel    { return XYZChannels }
func (s *xyzD50ColorSpace) IsBounded() bool               { return false }
func (s *xyzD50ColorSpace) IsLegacy() bool                { return false }
func (s *xyzD50ColorSpace) IsPolar() bool                 { return false }
func (s *xyzD50ColorSpace) ToLinear(ch float64) float64   { return ch }
func (s *xyzD50ColorSpace) FromLinear(ch float64) float64 { return ch }

func (s *xyzD50ColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return s.convertInternal(dest, c0, c1, c2, alpha, nil)
}

// Matches Dart: XyzD50ColorSpace.convert
func (s *xyzD50ColorSpace) convertInternal(dest ColorSpace, x, y, z, alpha *float64, opts *XyzD50ConvertOpts) (*SassColor, error) {
	// Analogous missingness normalization (#2810): missing a+b imply missing
	// chroma+hue and vice versa; an all-missing source converts to an
	// all-missing destination.
	missingLightness := opts != nil && opts.MissingLightness
	missingChroma := opts != nil && opts.MissingChroma
	missingHue := opts != nil && opts.MissingHue
	missingA := opts != nil && opts.MissingA
	missingB := opts != nil && opts.MissingB
	if missingA && missingB {
		missingChroma = true
		missingHue = true
	} else if missingChroma && missingHue {
		missingA = true
		missingB = true
	}
	if (missingLightness && missingChroma && missingHue) || (x == nil && y == nil && z == nil) {
		return newColorForSpaceInternalNoCheck(dest, nil, nil, nil, alpha), nil
	}

	xv, yv, zv := 0.0, 0.0, 0.0
	if x != nil {
		xv = *x
	}
	if y != nil {
		yv = *y
	}
	if z != nil {
		zv = *z
	}

	switch dest {
	case LabColorSpace, LchColorSpace:
		f0 := labConvertComponentToF(xv / D50[0])
		f1 := labConvertComponentToF(yv / D50[1])
		f2 := labConvertComponentToF(zv / D50[2])

		lightness := float64(116*f1) - 16 // prevent FMA: see matrixMul
		a := 500 * (f0 - f1)
		b := 200 * (f1 - f2)

		if dest == LabColorSpace {
			var lP, aP, bP *float64
			if !missingLightness {
				lP = &lightness
			}
			if !missingA {
				aP = &a
			}
			if !missingB {
				bP = &b
			}
			return NewColorForSpaceInternal(dest, lP, aP, bP, alpha)
		}

		var lPtr *float64
		if !missingLightness {
			lPtr = &lightness
		}
		return labToLch(LchColorSpace, lPtr, &a, &b, alpha, missingChroma, missingHue)

	default:
		linearOpts := ConvertLinearOpts{
			MissingLightness: missingLightness,
			MissingChroma:    missingChroma,
			MissingHue:       missingHue,
			MissingA:         missingA,
			MissingB:         missingB,
		}
		return convertLinear(XyzD50ColorSpace, dest, x, y, z, alpha, &linearOpts)
	}
}

func (s *xyzD50ColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return xyzD50ToLinearSrgb[:]
	case A98RgbColorSpace:
		return xyzD50ToLinearA98Rgb[:]
	case ProphotoRgbColorSpace:
		return xyzD50ToLinearProphotoRgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return xyzD50ToLinearDisplayP3[:]
	case Rec2020ColorSpace:
		return xyzD50ToLinearRec2020[:]
	case XyzD65ColorSpace:
		return xyzD50ToXyzD65[:]
	case LmsColorSpace:
		return xyzD50ToLms[:]
	default:
		return nil
	}
}
