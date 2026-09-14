// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"github.com/bancek/go-sass/sassmath"
	"github.com/bancek/go-sass/util"
)

// dart-source: lib/src/value/color/gamut_map_method/local_minde.dart
// localMindeGamutMap implements the CSS Color 4 local-MINDE gamut mapping algorithm.
// Algorithm from https://www.w3.org/TR/2022/CRD-css-color-4-20221101/#css-gamut-mapping-algorithm
//
// It works in Oklch: white/black lightness extremes short-circuit to RGB
// white/black (legacy colors keep an RGB round-trip), a clipped color within
// the just-noticeable difference (JND 0.02) wins immediately, and otherwise a
// binary search on chroma narrows the in-gamut boundary to epsilon 0.0001.
// The in-gamut fast path deliberately skips the gamut check once minInGamut
// turns false, per the CSSWG clarification linked from the Dart source.
func localMindeGamutMap(color *SassColor) (*SassColor, error) {
	const jnd = 0.02
	const epsilon = 0.0001

	originOklch, err := color.ToSpace(OklchColorSpace, nil)
	if err != nil {
		return nil, err
	}
	var lightness *float64
	if !originOklch.missing[0] {
		lightness = &originOklch.channel0
	}
	var hue *float64
	if !originOklch.missing[2] {
		hue = &originOklch.channel2
	}
	var alpha *float64
	if !originOklch.missing[3] {
		alpha = &originOklch.alpha
	}
	var alphaOrNull *float64
	if !color.missing[3] {
		alphaOrNull = &color.alpha
	}

	if util.FuzzyGreaterThanOrEquals(derefOrZero(lightness), 1) {
		if color.IsLegacy() {
			c, err := NewColorForSpaceInternal(RgbColorSpace, new(255.0), new(255.0), new(255.0), alphaOrNull)
			if err != nil {
				return nil, err
			}
			result, err := c.ToSpace(color.space, nil)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		one := 1.0
		return newColorForSpaceInternalNoCheck(color.space, &one, &one, &one, alphaOrNull), nil
	} else if util.FuzzyLessThanOrEquals(derefOrZero(lightness), 0) {
		c, err := NewColorForSpaceInternal(RgbColorSpace, new(0.0), new(0.0), new(0.0), alphaOrNull)
		if err != nil {
			return nil, err
		}
		result, err := c.ToSpace(color.space, nil)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	clipped, err := color.ToGamut(GamutMapClip)
	if err != nil {
		return nil, err
	}
	if de, err := deltaEOK(clipped, color); err != nil {
		return nil, err
	} else if de < jnd {
		return clipped, nil
	}

	min := 0.0
	max := originOklch.channel1
	minInGamut := true
	for max-min > epsilon {
		chroma := (min + max) / 2

		current, err := convertColor(OklchColorSpace, color.space,
			derefOrZero(lightness), chroma, derefOrZero(hue), derefOrZero(alpha),
			[4]bool{originOklch.missing[0], false, originOklch.missing[2], originOklch.missing[3]}, nil)
		if err != nil {
			return nil, err
		}

		if minInGamut && current.IsInGamut() {
			min = chroma
			continue
		}

		clipped, err = current.ToGamut(GamutMapClip)
		if err != nil {
			return nil, err
		}
		e, err := deltaEOK(clipped, current)
		if err != nil {
			return nil, err
		}
		if e < jnd {
			if jnd-e < epsilon {
				return clipped, nil
			}
			minInGamut = false
			min = chroma
		} else {
			max = chroma
		}
	}
	return clipped, nil
}

// derefOrZero returns the pointed-to value or zero for a missing channel,
// matching Dart's `?? 0` fallbacks in the gamut-mapping search.
func derefOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// deltaEOK returns the ΔEOK color difference between two colors.
// Algorithm from https://www.w3.org/TR/css-color-4/#color-difference-OK
//
// Both colors are mapped to OKLab and their Euclidean distance is returned.
// Matches Dart: LocalMindeGamutMap._deltaEOK.
func deltaEOK(c1, c2 *SassColor) (float64, error) {
	lab1, err := c1.ToSpace(OklabColorSpace, nil)
	if err != nil {
		return 0, err
	}
	lab2, err := c2.ToSpace(OklabColorSpace, nil)
	if err != nil {
		return 0, err
	}
	d0 := lab1.channel0 - lab2.channel0
	d1 := lab1.channel1 - lab2.channel1
	d2 := lab1.channel2 - lab2.channel2
	return sassmath.Sqrt(d0*d0 + d1*d1 + d2*d2), nil
}
