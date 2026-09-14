// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ChannelInfo returns metadata about a channel in the given color space.
//
// Out-of-range indexes fall back to the shared alpha channel. Matches Dart:
// SassColor channel metadata lookup via ColorSpace.channels.
//
// dart-source: lib/src/value/color.dart (channel metadata accessors)
func ChannelInfo(space ColorSpace, channel int) ColorChannel {
	channels := space.Channels()
	if channel >= 0 && channel < len(channels) {
		return channels[channel].ColorChannel
	}
	return AlphaChannel.ColorChannel
}

// SpaceChannels returns the 3 channel descriptors for the given color space.
//
// Matches Dart: ColorSpace.channels (SassApiColorSpace extension).
func SpaceChannels(space ColorSpace) [3]LinearChannel {
	return space.Channels()
}

// SpaceChannelNames returns the names of the 3 channels for the given space.
//
// Matches Dart: ColorSpace.channels names (used by channel() and serialization).
func SpaceChannelNames(space ColorSpace) [3]string {
	chs := space.Channels()
	return [3]string{chs[0].Name, chs[1].Name, chs[2].Name}
}

// --- Properties ---

// IsLegacy reports whether this color lives in a legacy space (rgb, hsl,
// hwb) with its compatibility semantics.
//
// Matches Dart: SassColor.isLegacy.
func (c *SassColor) IsLegacy() bool {
	return c.space.IsLegacy()
}

// IsInGamut reports whether every channel lies within its declared range.
//
// Unbounded spaces are always in gamut; polar-angle channels are skipped
// because hue wraps. Comparisons are fuzzy to absorb conversion rounding.
// Matches Dart: SassColor.isInGamut.
func (c *SassColor) IsInGamut() bool {
	if !c.space.IsBounded() {
		return true
	}
	chs := c.space.Channels()
	return isChannelInGamut(c.channel0, chs[0]) &&
		isChannelInGamut(c.channel1, chs[1]) &&
		isChannelInGamut(c.channel2, chs[2])
}

// ToGamut returns a copy of this color that's in-gamut in its color space.
//
// Matches Dart: SassColor.toGamut
func (c *SassColor) ToGamut(method GamutMapMethod) (*SassColor, error) {
	if c.IsInGamut() {
		return c, nil
	}
	switch method {
	case GamutMapClip:
		return clipGamutMap(c), nil
	case GamutMapLocalMinde:
		return localMindeGamutMap(c)
	default:
		return c, nil
	}
}

// isChannelInGamut reports whether one channel value is in gamut: always
// true for polar angles, otherwise a fuzzy min/max comparison.
//
// Matches Dart: SassColor._isChannelInGamut.
func isChannelInGamut(value float64, ch LinearChannel) bool {
	if ch.IsPolarAngle {
		return true
	}
	return util.FuzzyLessThanOrEquals(value, ch.Max) && util.FuzzyGreaterThanOrEquals(value, ch.Min)
}

// HasMissingChannel reports whether any color or alpha channel is missing.
//
// This is an internal helper in Dart (@internal); in Go it stays exported
// only because cross-file callers need it. Matches Dart:
// SassColor.hasMissingChannel.
func (c *SassColor) HasMissingChannel() bool {
	return c.missing[0] || c.missing[1] || c.missing[2] || c.missing[3]
}

// --- Deprecated change methods ---

// ChangeAlpha returns a copy of this color with alpha replaced.
//
// Matches Dart: SassColor.changeAlpha (deprecated alias kept for the
// color.change() built-in path).
func (c *SassColor) ChangeAlpha(alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(c.space, &c.channel0, &c.channel1, &c.channel2, &alpha)
}

// ChangeRGB returns a copy of this legacy color with the given 0-255 RGB
// channels replaced; nil entries keep the current channel.
//
// Only legacy colors are accepted; modern spaces must go through
// ChangeChannels with an explicit space. Matches Dart: SassColor.changeRgb.
func (c *SassColor) ChangeRGB(red, green, blue *int, alpha *float64) (*SassColor, error) {
	if !c.IsLegacy() {
		return nil, sasscommon.NewSassScriptException("color.changeRgb() is only supported for legacy colors. "+
			"Please use color.changeChannels() instead with an explicit $space argument.", nil)
	}
	a := c.alpha
	if alpha != nil {
		a = *alpha
	}
	var r float64
	if red != nil {
		r = float64(*red)
	} else {
		var err error
		r, err = c.ChannelByName("red")
		if err != nil {
			return nil, err
		}
	}
	var g float64
	if green != nil {
		g = float64(*green)
	} else {
		var err error
		g, err = c.ChannelByName("green")
		if err != nil {
			return nil, err
		}
	}
	var b float64
	if blue != nil {
		b = float64(*blue)
	} else {
		var err error
		b, err = c.ChannelByName("blue")
		if err != nil {
			return nil, err
		}
	}
	return NewColorRGB(r, g, b, a)
}

// ChangeHSL returns a copy of this legacy color with the given HSL channels
// replaced; nil entries keep the current channel. The HSL result is mapped
// back into the receiver's space.
//
// Matches Dart: SassColor.changeHsl.
func (c *SassColor) ChangeHSL(hue, saturation, lightness *float64, alpha *float64) (*SassColor, error) {
	if !c.IsLegacy() {
		return nil, sasscommon.NewSassScriptException("color.changeHsl() is only supported for legacy colors. "+
			"Please use color.changeChannels() instead with an explicit $space argument.", nil)
	}
	var h float64
	if hue != nil {
		h = *hue
	} else {
		var err error
		h, err = c.Hue()
		if err != nil {
			return nil, err
		}
	}
	var s float64
	if saturation != nil {
		s = *saturation
	} else {
		var err error
		s, err = c.Saturation()
		if err != nil {
			return nil, err
		}
	}
	var l float64
	if lightness != nil {
		l = *lightness
	} else {
		var err error
		l, err = c.Lightness()
		if err != nil {
			return nil, err
		}
	}
	a := c.alpha
	if alpha != nil {
		a = *alpha
	}
	result, err := NewColorHSL(h, s, l, a)
	if err != nil {
		return nil, err
	}
	converted, err := result.ToSpace(c.space, nil)
	if err != nil {
		return nil, err
	}
	return converted, nil
}

// ChangeHWB returns a copy of this legacy color with the given HWB channels
// replaced; nil entries keep the current channel. The HWB result is mapped
// back into the receiver's space.
//
// Matches Dart: SassColor.changeHwb.
func (c *SassColor) ChangeHWB(hue, whiteness, blackness *float64, alpha *float64) (*SassColor, error) {
	if !c.IsLegacy() {
		return nil, sasscommon.NewSassScriptException("color.changeHwb() is only supported for legacy colors. "+
			"Please use color.changeChannels() instead with an explicit $space argument.", nil)
	}
	var h float64
	if hue != nil {
		h = *hue
	} else {
		var err error
		h, err = c.Hue()
		if err != nil {
			return nil, err
		}
	}
	var w float64
	if whiteness != nil {
		w = *whiteness
	} else {
		var err error
		w, err = c.Whiteness()
		if err != nil {
			return nil, err
		}
	}
	var b float64
	if blackness != nil {
		b = *blackness
	} else {
		var err error
		b, err = c.Blackness()
		if err != nil {
			return nil, err
		}
	}
	a := c.alpha
	if alpha != nil {
		a = *alpha
	}
	a = a + 0.0
	result, err := NewColorHWB(h, w, b, a)
	if err != nil {
		return nil, err
	}
	converted, err := result.ToSpace(c.space, nil)
	if err != nil {
		return nil, err
	}
	return converted, nil
}

// ChangeChannels returns a copy of this color with the named channels
// replaced. An empty map returns the receiver unchanged; an explicit space
// converts there, applies the change, and converts back.
//
// Matches Dart: SassColor.changeChannels.
func (c *SassColor) ChangeChannels(newValues map[string]float64, space *ColorSpace) (*SassColor, error) {
	return c.changeChannels(newValues, nil, space)
}

// ChangeChannelsWithName is ChangeChannels with the originating color name
// attached for error messages.
//
// Matches Dart: SassColor.changeChannels (colorName parameter).
func (c *SassColor) ChangeChannelsWithName(newValues map[string]float64, colorName *string, space *ColorSpace) (*SassColor, error) {
	return c.changeChannels(newValues, colorName, space)
}

// changeChannels implements ChangeChannels: unknown names and duplicate
// assignments raise script errors, and untouched channels (including alpha)
// carry over unless they are missing.
func (c *SassColor) changeChannels(newValues map[string]float64, colorName *string, space *ColorSpace) (*SassColor, error) {
	if len(newValues) == 0 {
		return c, nil
	}

	if space != nil && *space != c.space {
		spaceConverted, err := c.ToSpace(*space, nil)
		if err != nil {
			return nil, err
		}
		converted, err := spaceConverted.changeChannels(newValues, colorName, space)
		if err != nil {
			return nil, err
		}
		result, err := converted.ToSpace(c.space, nil)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	chs := c.space.Channels()
	var new0, new1, new2 *float64
	var alpha *float64

	for name, val := range newValues {
		switch name {
		case chs[0].Name:
			if new0 != nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Multiple values supplied for %q: %g and %g", chs[0].Name, *new0, val), nil)
			}
			new0 = &val
		case chs[1].Name:
			if new1 != nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Multiple values supplied for %q: %g and %g", chs[1].Name, *new1, val), nil)
			}
			new1 = &val
		case chs[2].Name:
			if new2 != nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Multiple values supplied for %q: %g and %g", chs[2].Name, *new2, val), nil)
			}
			new2 = &val
		case "alpha":
			if alpha != nil {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Multiple values supplied for alpha: %g and %g", *alpha, val), nil)
			}
			alpha = &val
		default:
			s, err := c.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Color %s doesn't have a channel named %q", s, name), nil)
		}
	}

	c0 := new0
	if c0 == nil && !c.missing[0] {
		c0 = &c.channel0
	}
	c1 := new1
	if c1 == nil && !c.missing[1] {
		c1 = &c.channel1
	}
	c2 := new2
	if c2 == nil && !c.missing[2] {
		c2 = &c.channel2
	}
	a := alpha
	if a == nil && !c.missing[3] {
		a = &c.alpha
	}

	return NewColorForSpaceInternal(c.space, c0, c1, c2, a)
}
