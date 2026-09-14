// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color.dart

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// formatDoubleLikeDart formats a float to match Dart's double string
// representation, which always includes a decimal point for whole values.
//
// Used for the out-of-range weight error so `0` renders as `0.0` like Dart's
// RangeError text.
func formatDoubleLikeDart(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return s
}

// interpolateColors returns a color partway between c and other according to
// method, as defined by the CSS Color 4 color interpolation procedure.
// weight is a number between 0 and 1 (default 0.5).
//
// Endpoints short-circuit on fuzzy equality (0 returns other, 1 returns c),
// then both colors move to the interpolation space, missing channels take
// the other side's value, and premultiplied-alpha mixing applies. Hues in
// polar spaces use interpolateHues instead of linear mixing. Matches Dart:
// SassColor.interpolate.
func interpolateColors(c *SassColor, other *SassColor, method InterpolationMethod, legacyMissing bool, weight *float64) (*SassColor, error) {
	w := 0.5
	if weight != nil {
		w = *weight
	}

	if util.FuzzyEquals(w, 0) {
		return other, nil
	}
	if util.FuzzyEquals(w, 1) {
		return c, nil
	}

	if w < 0 || w > 1 {
		return nil, &sasscommon.RangeError{Name: "weight", Message: fmt.Sprintf("Invalid value: Not in inclusive range 0..1: %s", formatDoubleLikeDart(w))}
	}

	color1, err := c.ToSpace(method.Space, nil)
	if err != nil {
		return nil, err
	}
	color2, err := other.ToSpace(method.Space, nil)
	if err != nil {
		return nil, err
	}

	// A channel missing in one color (or analogous to a missing channel —
	// conversions now propagate analogous missingness themselves, #2810)
	// takes the other color's value.
	ch1_0 := color1.Channel0OrNil()
	if ch1_0 == nil {
		ch1_0 = color2.Channel0OrNil()
	}
	ch1_1 := color1.Channel1OrNil()
	if ch1_1 == nil {
		ch1_1 = color2.Channel1OrNil()
	}
	ch1_2 := color1.Channel2OrNil()
	if ch1_2 == nil {
		ch1_2 = color2.Channel2OrNil()
	}
	ch2_0 := color2.Channel0OrNil()
	if ch2_0 == nil {
		ch2_0 = color1.Channel0OrNil()
	}
	ch2_1 := color2.Channel1OrNil()
	if ch2_1 == nil {
		ch2_1 = color1.Channel1OrNil()
	}
	ch2_2 := color2.Channel2OrNil()
	if ch2_2 == nil {
		ch2_2 = color1.Channel2OrNil()
	}

	alpha1 := c.alpha
	if c.missing[3] {
		alpha1 = other.alpha
	}
	alpha2 := other.alpha
	if other.missing[3] {
		alpha2 = c.alpha
	}

	thisMultiplier := w
	if !c.missing[3] {
		thisMultiplier = c.alpha * w
	}
	otherMultiplier := 1 - w
	if !other.missing[3] {
		otherMultiplier = other.alpha * (1 - w)
	}

	var mixedAlpha *float64
	if c.missing[3] && other.missing[3] {
		mixedAlpha = nil
	} else {
		ma := alpha1*w + alpha2*(1-w)
		mixedAlpha = &ma
	}

	maVal := float64(1)
	if mixedAlpha != nil {
		maVal = *mixedAlpha
	}

	mixed0 := mixChannels(ch1_0, ch2_0, thisMultiplier, otherMultiplier, maVal)
	mixed1 := mixChannels(ch1_1, ch2_1, thisMultiplier, otherMultiplier, maVal)
	mixed2 := mixChannels(ch1_2, ch2_2, thisMultiplier, otherMultiplier, maVal)

	var result *SassColor
	switch method.Space {
	case HslColorSpace, HwbColorSpace:
		var hVal *float64
		if ch1_0 != nil {
			h := interpolateHues(*ch1_0, *ch2_0, *method.Hue, w)
			hVal = &h
		}
		result = newColorForSpaceInternalNoCheck(method.Space,
			hVal, mixed1, mixed2, mixedAlpha)
	case LchColorSpace, OklchColorSpace:
		var hVal *float64
		if ch1_2 != nil {
			h := interpolateHues(*ch1_2, *ch2_2, *method.Hue, w)
			hVal = &h
		}
		result = newColorForSpaceInternalNoCheck(method.Space,
			mixed0, mixed1, hVal, mixedAlpha)
	default:
		result = newColorForSpaceInternalNoCheck(method.Space,
			mixed0, mixed1, mixed2, mixedAlpha)
	}

	resultConverted, err := result.ToSpace(c.space, &legacyMissing)
	if err != nil {
		return nil, err
	}
	return resultConverted, nil
}

// mixChannels returns the premultiplied-alpha mix of one channel pair; nil
// when the first side is missing (in which case both sides are missing, since
// ch1 falls back to ch2's value).
//
// Matches Dart: per-channel premultiplied mix inside SassColor.interpolate.
func mixChannels(ch1, ch2 *float64, m1, m2, ma float64) *float64 {
	if ch1 == nil {
		return nil
	}
	v := (*ch1*m1 + *ch2*m2) / ma
	return &v
}

// interpolateHues returns a hue partway between hue1 and hue2 according to method.
// Algorithms from https://www.w3.org/TR/css-color-4/#hue-interpolation
//
// Shorter takes the <180-degree arc, longer the >180-degree arc, and
// increasing/decreasing force a clockwise/counter-clockwise direction before
// the linear weight blend. Matches Dart: SassColor._interpolateHues.
func interpolateHues(hue1, hue2 float64, method HueInterpolationMethod, weight float64) float64 {
	switch method {
	case HueInterpolationShorter:
		diff := hue2 - hue1
		if diff > 180 {
			hue1 += 360
		} else if diff < -180 {
			hue2 += 360
		}
	case HueInterpolationLonger:
		diff := hue2 - hue1
		if diff > 0 && diff < 180 {
			hue2 += 360
		} else if diff > -180 && diff <= 0 {
			hue1 += 360
		}
	case HueInterpolationIncreasing:
		if hue2 < hue1 {
			hue2 += 360
		}
	case HueInterpolationDecreasing:
		if hue1 < hue2 {
			hue1 += 360
		}
	}
	return hue1*weight + hue2*(1-weight)
}
