// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/channel.dart
// ColorChannel represents metadata about a single channel in a color space.
//
// This ports Dart's sealed channel class without its category/sealed
// annotations: Name identifies the channel, IsPolarAngle marks hue-style
// circular channels, and AssociatedUnit records the unit used when the value
// is serialized or returned from a Sass function.
type ColorChannel struct {
	// Name is the channel name (for example "red", "hue", or "lightness").
	Name string
	// IsPolarAngle reports whether the channel is a circular hue angle in degrees.
	IsPolarAngle bool
	// AssociatedUnit is the unit conventionally used when serializing the channel.
	AssociatedUnit string
}

// NewColorChannel creates channel metadata with the given name, polarity, and unit.
//
// Matches Dart: ColorChannel constructor.
func NewColorChannel(name string, isPolarAngle bool, associatedUnit string) ColorChannel {
	return ColorChannel{
		Name:           name,
		IsPolarAngle:   isPolarAngle,
		AssociatedUnit: associatedUnit,
	}
}

// IsAnalogous reports whether this channel is analogous to other.
//
// Matches Dart: ColorChannel.isAnalogous
func (c ColorChannel) IsAnalogous(other ColorChannel) bool {
	return analogousNames(c.Name, other.Name)
}

// analogousNames reports whether two channel names interpolate as the same
// conceptual channel (red/x, green/y, blue/z, chroma/saturation, lightness,
// hue), per https://www.w3.org/TR/css-color-4/#interpolation-missing.
func analogousNames(a, b string) bool {
	return (a == "red" || a == "x") && (b == "red" || b == "x") ||
		(a == "green" || a == "y") && (b == "green" || b == "y") ||
		(a == "blue" || a == "z") && (b == "blue" || b == "z") ||
		(a == "chroma" || a == "saturation") && (b == "chroma" || b == "saturation") ||
		a == "lightness" && b == "lightness" ||
		a == "hue" && b == "hue"
}

// LinearChannel represents metadata about a color channel with a linear value range.
//
// Min/Max double as the percentage reference range and, for bounded spaces,
// the in-gamut boundary; values may still fall outside unless the space is
// strictly bounded. Matches Dart: LinearChannel.
type LinearChannel struct {
	ColorChannel
	// Min is the channel's minimum reference value.
	Min float64
	// Max is the channel's maximum reference value.
	Max float64
	// RequiresPercent reports whether unitless values are rejected for this channel.
	RequiresPercent bool
	// LowerClamped reports whether the global-function syntax clamps the lower bound.
	LowerClamped bool
	// UpperClamped reports whether the global-function syntax clamps the upper bound.
	UpperClamped bool
}

// NewLinearChannel creates a linear channel; AssociatedUnit defaults to "%"
// exactly when the range is 0-100 unless an option overrides it.
//
// Matches Dart: LinearChannel constructor (conventionallyPercent logic).
func NewLinearChannel(name string, min, max float64, opts ...func(*LinearChannel)) LinearChannel {
	ch := LinearChannel{
		ColorChannel: ColorChannel{
			Name:           name,
			IsPolarAngle:   false,
			AssociatedUnit: percentIf(min, max),
		},
		Min: min,
		Max: max,
	}
	for _, opt := range opts {
		opt(&ch)
	}
	return ch
}

// AlphaChannel is the alpha channel shared across all colors.
//
// Matches Dart: ColorChannel.alpha.
var AlphaChannel = NewLinearChannel("alpha", 0, 1)

// percentIf returns "%" for a 0-100 range and "" otherwise; shared by the
// channel constructors for the default associated unit.
func percentIf(min, max float64) string {
	if min == 0 && max == 100 {
		return "%"
	}
	return ""
}

// WithRequiresPercent marks the channel as requiring "%" values.
//
// Matches Dart: LinearChannel.requiresPercent.
func WithRequiresPercent(ch *LinearChannel) {
	ch.RequiresPercent = true
}

// WithLowerClamped marks the channel's lower bound as clamped in the global
// function syntax.
//
// Matches Dart: LinearChannel.lowerClamped.
func WithLowerClamped(ch *LinearChannel) {
	ch.LowerClamped = true
}

// WithUpperClamped marks the channel's upper bound as clamped in the global
// function syntax.
//
// Matches Dart: LinearChannel.upperClamped.
func WithUpperClamped(ch *LinearChannel) {
	ch.UpperClamped = true
}

// WithConventionallyPercent forces the associated unit to "%".
//
// Matches Dart: LinearChannel conventionallyPercent=true.
func WithConventionallyPercent(ch *LinearChannel) {
	ch.AssociatedUnit = "%"
}

// WithNoPercent forces no associated unit.
//
// Matches Dart: LinearChannel conventionallyPercent=false.
func WithNoPercent(ch *LinearChannel) {
	ch.AssociatedUnit = ""
}

// HueChannelInfo returns the hue channel shared by all polar spaces.
//
// Matches Dart: hueChannel in space/utils.dart.
func HueChannelInfo() LinearChannel {
	return LinearChannel{
		ColorChannel: ColorChannel{
			Name:           "hue",
			IsPolarAngle:   true,
			AssociatedUnit: "deg",
		},
	}
}
