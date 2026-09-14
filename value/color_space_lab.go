// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"github.com/bancek/go-sass/util"
)

// dart-source: lib/src/value/color/space/lab.dart

// LabConvertOpts carries optional parameters matching Dart's
// LabColorSpace.convert named parameters.
//
// Matches Dart: LabColorSpace.convert named params
type LabConvertOpts struct {
	// MissingChroma records a missing source chroma set for LCH routing.
	MissingChroma bool
	// MissingHue records a missing source hue for LCH routing.
	MissingHue bool
}

// LabChannels are the channels for the Lab color space.
var LabChannels = [3]LinearChannel{
	NewLinearChannel("lightness", 0, 100, WithLowerClamped, WithUpperClamped),
	NewLinearChannel("a", -125, 125),
	NewLinearChannel("b", -125, 125),
}

// labColorSpace is the CIE Lab color space.
//
// Matches Dart: LabColorSpace
type labColorSpace struct{}

// labSpace is the internal singleton used for conversion routing.
//
// The exported LabColorSpace in color_space_base.go is the public handle.
var labSpace = &labColorSpace{}

func (s *labColorSpace) Name() string               { return "lab" }
func (s *labColorSpace) Channels() [3]LinearChannel { return LabChannels }
func (s *labColorSpace) IsBounded() bool            { return false }
func (s *labColorSpace) IsLegacy() bool             { return false }
func (s *labColorSpace) IsPolar() bool              { return false }

func (s *labColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *labColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *labColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *labColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return s.convertInternal(dest, c0, c1, c2, alpha, nil)
}

// Matches Dart: LabColorSpace.convert
func (s *labColorSpace) convertInternal(dest ColorSpace, lightness, a, b, alpha *float64, opts *LabConvertOpts) (*SassColor, error) {
	// Analogous missingness: null a+b imply missing chroma+hue and vice
	// versa (#2810).
	missingChroma := opts != nil && opts.MissingChroma
	missingHue := opts != nil && opts.MissingHue
	if missingChroma && missingHue {
		a = nil
		b = nil
	} else if a == nil && b == nil {
		missingChroma = true
		missingHue = true
	}

	lv, av, bv := 0.0, 0.0, 0.0
	if lightness != nil {
		lv = *lightness
	}
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}

	switch dest {
	case LabColorSpace:
		powerlessAB := lightness == nil || util.FuzzyEquals(lv, 0)
		var newA, newB *float64
		if powerlessAB {
			newA, newB = nil, nil
		} else {
			if a != nil {
				newA = a
			}
			if b != nil {
				newB = b
			}
		}
		return NewColorForSpaceInternal(dest, lightness, newA, newB, alpha)

	case LchColorSpace:
		// Porting Dart bug: Dart's convertInternal does NOT pass
		// missingChroma/missingHue for the LCH destination path:
		//   case ColorSpace.lch:
		//     return labToLch(dest, lightness, a, b, alpha);
		// (missingChroma and missingHue default to false — they are not
		// forwarded from opts, unlike all other color spaces.)
		return labToLch(dest, lightness, a, b, alpha, false, false)

	default:
		missingLightness := lightness == nil
		lv2 := lv
		if lightness == nil {
			lv2 = 0
		}
		f1 := (lv2 + 16) / 116
		xv := labFToXZ(av/500+f1) * D50[0]
		var yVal float64
		if lv2 > LabKappa*LabEpsilon {
			f1cube := (lv2 + 16) / 116
			yVal = f1cube * f1cube * f1cube * 1.0
		} else {
			yVal = lv2 / LabKappa
		}
		yv := yVal * D50[1]
		zv := labFToXZ(f1-bv/200) * D50[2]

		return xyzD50Space.convertInternal(dest, &xv, &yv, &zv, alpha, &XyzD50ConvertOpts{
			MissingLightness: missingLightness,
			MissingChroma:    missingChroma,
			MissingHue:       missingHue,
			MissingA:         a == nil,
			MissingB:         b == nil,
		})
	}
}
