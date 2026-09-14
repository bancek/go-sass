// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/space/oklab.dart

// OklabConvertOpts carries optional parameters matching Dart's
// OklabColorSpace.convert named parameters.
//
// Matches Dart: OklabColorSpace.convert named params
type OklabConvertOpts struct {
	// MissingChroma records a missing source chroma set for OKLCH routing.
	MissingChroma bool
	// MissingHue records a missing source hue for OKLCH routing.
	MissingHue bool
}

// OKLabChannels are the channels for the OKLab color space.
var OKLabChannels = [3]LinearChannel{
	NewLinearChannel("lightness", 0, 1, WithLowerClamped, WithUpperClamped, WithConventionallyPercent),
	NewLinearChannel("a", -0.4, 0.4),
	NewLinearChannel("b", -0.4, 0.4),
}

// oklabColorSpace is the OKLab color space.
//
// Matches Dart: OklabColorSpace
type oklabColorSpace struct{}

// oklabSpace is the internal singleton used for conversion routing.
//
// The exported OklabColorSpace in color_space_base.go is the public handle.
var oklabSpace = &oklabColorSpace{}

func (s *oklabColorSpace) Name() string               { return "oklab" }
func (s *oklabColorSpace) Channels() [3]LinearChannel { return OKLabChannels }
func (s *oklabColorSpace) IsBounded() bool            { return false }
func (s *oklabColorSpace) IsLegacy() bool             { return false }
func (s *oklabColorSpace) IsPolar() bool              { return false }

func (s *oklabColorSpace) ToLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *oklabColorSpace) FromLinear(ch float64) float64 {
	panic("BUG: Color space doesn't support linear conversions")
}

func (s *oklabColorSpace) TransformationMatrix(dest ColorSpace) []float64 { return nil }

func (s *oklabColorSpace) Convert(dest ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	return s.convertInternal(dest, c0, c1, c2, alpha, nil)
}

// Matches Dart: OklabColorSpace.convert
func (s *oklabColorSpace) convertInternal(dest ColorSpace, lightness, a, b, alpha *float64, opts *OklabConvertOpts) (*SassColor, error) {
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

	if dest == OklchColorSpace {
		missingChroma := opts != nil && opts.MissingChroma
		missingHue := opts != nil && opts.MissingHue
		return labToLch(dest, lightness, a, b, alpha, missingChroma, missingHue)
	}

	// Analogous missingness: null a+b imply missing chroma+hue and vice
	// versa (#2810). Note the order (nulls first) differs from Lab.
	missingChroma := opts != nil && opts.MissingChroma
	missingHue := opts != nil && opts.MissingHue
	if a == nil && b == nil {
		missingChroma = true
		missingHue = true
	} else if missingChroma && missingHue {
		a = nil
		b = nil
	}

	missingLightness := lightness == nil
	missingA := a == nil
	missingB := b == nil

	longLms := float64(oklabToLms[0]*lv) + float64(oklabToLms[1]*av) + float64(oklabToLms[2]*bv)   // prevent FMA: see matrixMul
	mediumLms := float64(oklabToLms[3]*lv) + float64(oklabToLms[4]*av) + float64(oklabToLms[5]*bv) // prevent FMA: see matrixMul
	shortLms := float64(oklabToLms[6]*lv) + float64(oklabToLms[7]*av) + float64(oklabToLms[8]*bv)  // prevent FMA: see matrixMul
	long := longLms*longLms*longLms + 0.0
	medium := mediumLms*mediumLms*mediumLms + 0.0
	short := shortLms*shortLms*shortLms + 0.0

	return lmsSpace.convertInternal(dest, &long, &medium, &short, alpha, &LmsConvertOpts{
		MissingLightness: missingLightness,
		MissingChroma:    missingChroma,
		MissingHue:       missingHue,
		MissingA:         missingA,
		MissingB:         missingB,
	})
}
