// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/conversions.dart (ToSpace, ToRGB, ToHSL, ToHWB)
// --- Conversion ---

// ToSpace converts this color to space, preserving missing channels.
//
// An identity conversion returns the receiver. When legacyMissing is
// explicitly false, missing channels in a legacy result are filled with
// zeros instead of staying missing. Matches Dart: SassColor.toSpace.
func (c *SassColor) ToSpace(space ColorSpace, legacyMissing *bool) (*SassColor, error) {
	lm := true
	if legacyMissing != nil {
		lm = *legacyMissing
	}
	if c.space == space {
		return c, nil
	}

	converted, err := convertColor(c.space, space,
		c.channel0, c.channel1, c.channel2, c.alpha, c.missing, nil)
	if err != nil {
		return nil, err
	}

	if !lm && converted.IsLegacy() && converted.HasMissingChannel() {
		c0, c1, c2, a := converted.channel0, converted.channel1, converted.channel2, converted.alpha
		result, err := NewColorForSpaceInternal(converted.space, &c0, &c1, &c2, &a)
		if err != nil {
			return converted, nil
		}
		return result, nil
	}
	return converted, nil
}

// ToRGB returns the color in sRGB 0-255 range (legacy colors only).
//
// It converts to the legacy rgb space and unpacks the channels; there is no
// named Dart counterpart (the rgb/hsl/hwb helpers share the toSpace path).
// Matches Dart: SassColor.toSpace (legacy rgb section).
func (c *SassColor) ToRGB() (r, g, b, a float64, err error) {
	cc, err := c.ToSpace(RgbColorSpace, nil)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return cc.channel0, cc.channel1, cc.channel2, cc.alpha, nil
}

// ToHSL returns the color in HSL values (legacy colors only).
//
// Matches Dart: SassColor.toSpace (legacy hsl section).
func (c *SassColor) ToHSL() (h, s, l, a float64, err error) {
	cc, err := c.ToSpace(HslColorSpace, nil)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return cc.channel0, cc.channel1, cc.channel2, cc.alpha, nil
}

// ToHWB returns the color in HWB values (legacy colors only).
//
// Matches Dart: SassColor.toSpace (legacy hwb section).
func (c *SassColor) ToHWB() (h, w, b, a float64, err error) {
	cc, err := c.ToSpace(HwbColorSpace, nil)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return cc.channel0, cc.channel1, cc.channel2, cc.alpha, nil
}
