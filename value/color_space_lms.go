// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/lms.dart

// LmsConvertOpts carries optional parameters matching Dart's
// LmsColorSpace.convert named parameters.
//
// Matches Dart: LmsColorSpace.convert named params
type LmsConvertOpts struct {
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

// LMSChannels are the channels for the internal LMS color space.
var LMSChannels = [3]LinearChannel{
	NewLinearChannel("long", 0, 1),
	NewLinearChannel("medium", 0, 1),
	NewLinearChannel("short", 0, 1),
}

// lmsColorSpace is the internal LMS color space (never exposed to users).
//
// Matches Dart: LmsColorSpace
type lmsColorSpace struct{}

// lmsSpace is the internal singleton used for conversion routing.
//
// The exported LmsColorSpace in color_space_base.go is the public handle;
// LMS never appears in a real color value.
var lmsSpace = &lmsColorSpace{}

func (s *lmsColorSpace) Name() string                  { return "lms" }
func (s *lmsColorSpace) Channels() [3]LinearChannel    { return LMSChannels }
func (s *lmsColorSpace) IsBounded() bool               { return false }
func (s *lmsColorSpace) IsLegacy() bool                { return false }
func (s *lmsColorSpace) IsPolar() bool                 { return false }
func (s *lmsColorSpace) ToLinear(ch float64) float64   { return ch }
func (s *lmsColorSpace) FromLinear(ch float64) float64 { return ch }

func (s *lmsColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return s.convertInternal(dest, c0, c1, c2, alpha, nil)
}

// Matches Dart: LmsColorSpace.convert
func (s *lmsColorSpace) convertInternal(dest ColorSpace, long, medium, short, alpha *float64, opts *LmsConvertOpts) (*SassColor, error) {
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
	if (missingLightness && missingChroma && missingHue) || (long == nil && medium == nil && short == nil) {
		return newColorForSpaceInternalNoCheck(dest, nil, nil, nil, alpha), nil
	}

	lv, mv, sv := 0.0, 0.0, 0.0
	if long != nil {
		lv = *long
	}
	if medium != nil {
		mv = *medium
	}
	if short != nil {
		sv = *short
	}

	switch dest {
	case OklabColorSpace:
		longS := cubeRootPreservingSign(lv)
		mediumS := cubeRootPreservingSign(mv)
		shortS := cubeRootPreservingSign(sv)
		l := float64(lmsToOklab[0]*longS) + float64(lmsToOklab[1]*mediumS) + float64(lmsToOklab[2]*shortS) // prevent FMA: see matrixMul
		a := float64(lmsToOklab[3]*longS) + float64(lmsToOklab[4]*mediumS) + float64(lmsToOklab[5]*shortS) // prevent FMA: see matrixMul
		b := float64(lmsToOklab[6]*longS) + float64(lmsToOklab[7]*mediumS) + float64(lmsToOklab[8]*shortS) // prevent FMA: see matrixMul

		var lP, aP, bP *float64
		if !missingLightness {
			lP = &l
		}
		if !missingA {
			aP = &a
		}
		if !missingB {
			bP = &b
		}
		return NewColorForSpaceInternal(OklabColorSpace, lP, aP, bP, alpha)

	case OklchColorSpace:
		longS := cubeRootPreservingSign(lv)
		mediumS := cubeRootPreservingSign(mv)
		shortS := cubeRootPreservingSign(sv)
		var lPtr *float64
		if !missingLightness {
			l := float64(lmsToOklab[0]*longS) + float64(lmsToOklab[1]*mediumS) + float64(lmsToOklab[2]*shortS) // prevent FMA: see matrixMul
			lPtr = &l
		}
		a := float64(lmsToOklab[3]*longS) + float64(lmsToOklab[4]*mediumS) + float64(lmsToOklab[5]*shortS) // prevent FMA: see matrixMul
		b := float64(lmsToOklab[6]*longS) + float64(lmsToOklab[7]*mediumS) + float64(lmsToOklab[8]*shortS) // prevent FMA: see matrixMul

		return labToLch(OklchColorSpace, lPtr, &a, &b, alpha, missingChroma, missingHue)

	default:
		linearOpts := ConvertLinearOpts{
			MissingLightness: missingLightness,
			MissingChroma:    missingChroma,
			MissingHue:       missingHue,
			MissingA:         missingA,
			MissingB:         missingB,
		}
		return convertLinear(LmsColorSpace, dest, long, medium, short, alpha, &linearOpts)
	}
}

func (s *lmsColorSpace) TransformationMatrix(dest ColorSpace) []float64 {
	switch dest {
	case SrgbColorSpace, SrgbLinearColorSpace, RgbColorSpace:
		return lmsToLinearSrgb[:]
	case A98RgbColorSpace:
		return lmsToLinearA98Rgb[:]
	case ProphotoRgbColorSpace:
		return lmsToLinearProphotoRgb[:]
	case DisplayP3ColorSpace, DisplayP3LinearColorSpace:
		return lmsToLinearDisplayP3[:]
	case Rec2020ColorSpace:
		return lmsToLinearRec2020[:]
	case XyzD65ColorSpace:
		return lmsToXyzD65[:]
	case XyzD50ColorSpace:
		return lmsToXyzD50[:]
	default:
		return nil
	}
}
