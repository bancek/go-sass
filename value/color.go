// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color.dart (SassColor type, channels, constructors;
// SassColor.interpolate lives in color_interpolation.go)

import (
	"fmt"
	"math"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ColorFormat records the syntax a color was originally written in, so
// expanded-mode serialization can preserve it.
// Matches Dart: ColorFormat
type ColorFormat interface {
	isColorFormat()
}

type colorFormatEnum string

func (c colorFormatEnum) isColorFormat() {}
func (c colorFormatEnum) String() string { return string(c) }

// ColorFormatRGBFunction marks a color defined through the rgb() or rgba()
// functions, which expanded output renders back in that form.
// Matches Dart: ColorFormat.rgbFunction
var ColorFormatRGBFunction ColorFormat = colorFormatEnum("rgbFunction")

// SpanColorFormat renders the color as the exact source text it was written
// with. The original text is tracked as a span rather than a copied string
// to avoid extra substring allocations.
// Matches Dart: SpanColorFormat
type SpanColorFormat struct {
	// Span locates the source text the color was originally written with.
	Span sasscommon.FileSpan
}

func (s *SpanColorFormat) isColorFormat() {}

// Original returns the source text the color was originally written with.
// Matches Dart: SpanColorFormat.original
func (s *SpanColorFormat) Original() (string, error) { return s.Span.SpanText() }

// SassColor is a SassScript color value: a color space, three channel values,
// an alpha channel, a missing-channel mask, and the original format for
// serialization. A missing channel reads back as 0 but serializes as `none`.
// Matches Dart: SassColor
type SassColor struct {
	space    ColorSpace
	channel0 float64
	channel1 float64
	channel2 float64
	alpha    float64
	missing  [4]bool
	format   ColorFormat
}

// --- Space and Channel getters ---

// Space returns this color's color space.
// Matches Dart: SassColor.space
func (c *SassColor) Space() ColorSpace { return c.space }

// Channel access: ChannelN returns the raw channel value (0 for a missing
// channel) while ChannelNOrNil returns nil for a missing channel. Channel
// semantics depend on the color space.

// Channel0 returns this color's first channel, or 0 when missing.
// Matches Dart: SassColor.channel0
func (c *SassColor) Channel0() float64 { return c.channel0 }

// Channel1 returns this color's second channel, or 0 when missing.
// Matches Dart: SassColor.channel1
func (c *SassColor) Channel1() float64 { return c.channel1 }

// Channel2 returns this color's third channel, or 0 when missing.
// Matches Dart: SassColor.channel2
func (c *SassColor) Channel2() float64 { return c.channel2 }

// Alpha returns this color's alpha channel (0 to 1), or 0 when missing. It
// is never NaN or negative zero.
// Matches Dart: SassColor.alpha
func (c *SassColor) Alpha() float64 { return c.alpha }

// Channel0OrNil returns this color's first channel, or nil when missing.
// Matches Dart: SassColor.channel0OrNull
func (c *SassColor) Channel0OrNil() *float64 {
	if c.missing[0] {
		return nil
	}
	return &c.channel0
}

// Channel1OrNil returns this color's second channel, or nil when missing.
// Matches Dart: SassColor.channel1OrNull
func (c *SassColor) Channel1OrNil() *float64 {
	if c.missing[1] {
		return nil
	}
	return &c.channel1
}

// Channel2OrNil returns this color's third channel, or nil when missing.
// Matches Dart: SassColor.channel2OrNull
func (c *SassColor) Channel2OrNil() *float64 {
	if c.missing[2] {
		return nil
	}
	return &c.channel2
}

// AlphaOrNil returns this color's alpha channel, or nil when missing.
// Matches Dart: SassColor.alphaOrNull
func (c *SassColor) AlphaOrNil() *float64 {
	if c.missing[3] {
		return nil
	}
	return &c.alpha
}

// Channels returns this color's three channel values, excluding alpha.
// Each channel's meaning varies with the color space.
// Matches Dart: SassColor.channels
func (c *SassColor) Channels() [3]float64 {
	return [3]float64{c.channel0, c.channel1, c.channel2}
}

// ChannelsOrNil returns this color's three channel values, excluding alpha,
// with nil for missing channels. Each channel's meaning varies with the
// color space.
// Matches Dart: SassColor.channelsOrNull
func (c *SassColor) ChannelsOrNil() [3]*float64 {
	return [3]*float64{c.Channel0OrNil(), c.Channel1OrNil(), c.Channel2OrNil()}
}

// Missing-channel reporters: each reports whether the corresponding channel
// was written as `none` (CSS missing) rather than as a number.

// IsChannel0Missing reports whether the first channel is missing.
// Matches Dart: SassColor.isChannel0Missing
func (c *SassColor) IsChannel0Missing() bool { return c.missing[0] }

// IsChannel1Missing reports whether the second channel is missing.
// Matches Dart: SassColor.isChannel1Missing
func (c *SassColor) IsChannel1Missing() bool { return c.missing[1] }

// IsChannel2Missing reports whether the third channel is missing.
// Matches Dart: SassColor.isChannel2Missing
func (c *SassColor) IsChannel2Missing() bool { return c.missing[2] }

// IsAlphaMissing reports whether the alpha channel is missing.
// Matches Dart: SassColor.isAlphaMissing
func (c *SassColor) IsAlphaMissing() bool { return c.missing[3] }

// Powerless-channel reporters: a powerless channel's value has no visual
// effect (for example hue at zero saturation), so interpolation and
// serialization treat it like a missing channel.

// IsChannel0Powerless reports whether the first channel is powerless: hue in
// HSL at zero saturation, or hue in HWB once whiteness plus blackness
// reaches 100.
// Matches Dart: SassColor.isChannel0Powerless
func (c *SassColor) IsChannel0Powerless() bool {
	switch c.space {
	case HslColorSpace:
		return util.FuzzyEquals(c.channel1, 0)
	case HwbColorSpace:
		return util.FuzzyGreaterThanOrEquals(c.channel1+c.channel2, 100)
	default:
		return false
	}
}

// IsChannel1Powerless reports whether the second channel is powerless. It
// is always false: no color space renders its middle channel powerless.
// Matches Dart: SassColor.isChannel1Powerless
func (c *SassColor) IsChannel1Powerless() bool {
	return false
}

// IsChannel2Powerless reports whether the third channel is powerless: hue in
// LCH and OKLCH at zero chroma.
// Matches Dart: SassColor.isChannel2Powerless
func (c *SassColor) IsChannel2Powerless() bool {
	switch c.space {
	case LchColorSpace, OklchColorSpace:
		return util.FuzzyEquals(c.channel1, 0)
	default:
		return false
	}
}

// --- Legacy channel getters ---
// Deprecated single-channel accessors for the pre-color-space syntax. Each
// converts a legacy color to the channel's space first and throws for modern
// spaces; prefer ChannelByName with an explicit space.

// Red returns the sRGB red channel from 0 to 255, rounded to the nearest
// integer (which may be lossy).
//
// Deprecated: use ChannelByName instead to get the exact channel value.
// Matches Dart: SassColor.red
func (c *SassColor) Red() (float64, error) {
	val, err := c.legacyChannel(RgbColorSpace, "red")
	if err != nil {
		return 0, err
	}
	return math.Round(val), nil
}

// Green returns the sRGB green channel from 0 to 255, rounded to the
// nearest integer (which may be lossy).
//
// Deprecated: use ChannelByName instead to get the exact channel value.
// Matches Dart: SassColor.green
func (c *SassColor) Green() (float64, error) {
	val, err := c.legacyChannel(RgbColorSpace, "green")
	if err != nil {
		return 0, err
	}
	return math.Round(val), nil
}

// Blue returns the sRGB blue channel from 0 to 255, rounded to the nearest
// integer (which may be lossy).
//
// Deprecated: use ChannelByName instead to get the exact channel value.
// Matches Dart: SassColor.blue
func (c *SassColor) Blue() (float64, error) {
	val, err := c.legacyChannel(RgbColorSpace, "blue")
	if err != nil {
		return 0, err
	}
	return math.Round(val), nil
}

// Hue returns the HSL hue from 0 to 360.
//
// Deprecated: use ChannelByName instead.
// Matches Dart: SassColor.hue
func (c *SassColor) Hue() (float64, error) {
	return c.legacyChannel(HslColorSpace, "hue")
}

// Saturation returns the HSL saturation as a percentage from 0 to 100.
//
// Deprecated: use ChannelByName instead.
// Matches Dart: SassColor.saturation
func (c *SassColor) Saturation() (float64, error) {
	return c.legacyChannel(HslColorSpace, "saturation")
}

// Lightness returns the HSL lightness as a percentage from 0 to 100.
//
// Deprecated: use ChannelByName instead.
// Matches Dart: SassColor.lightness
func (c *SassColor) Lightness() (float64, error) {
	return c.legacyChannel(HslColorSpace, "lightness")
}

// Whiteness returns the HWB whiteness as a percentage from 0 to 100.
//
// Deprecated: use ChannelByName instead.
// Matches Dart: SassColor.whiteness
func (c *SassColor) Whiteness() (float64, error) {
	return c.legacyChannel(HwbColorSpace, "whiteness")
}

// Blackness returns the HWB blackness as a percentage from 0 to 100.
//
// Deprecated: use ChannelByName instead.
// Matches Dart: SassColor.blackness
func (c *SassColor) Blackness() (float64, error) {
	return c.legacyChannel(HwbColorSpace, "blackness")
}

// legacyChannel converts a legacy color to space and reads channel from it.
// Modern-space colors throw, pointing at ChannelByName with an explicit
// space argument.
// Matches Dart: SassColor._legacyChannel
func (c *SassColor) legacyChannel(space ColorSpace, channel string) (float64, error) {
	if !c.IsLegacy() {
		return 0, sasscommon.NewSassScriptException(fmt.Sprintf("color.%s() is only supported for legacy colors. "+
			"Please use color.channel() instead with an explicit $space argument.", channel), nil)
	}
	converted, err := c.ToSpace(space, nil)
	if err != nil {
		return 0, err
	}
	return converted.ChannelByName(channel)
}

// ChannelByName returns the value of the named channel ("alpha" included),
// or a SassScriptException when the color has no such channel. The result is
// never NaN or negative zero, and polar channels never report infinities.
// Matches Dart: SassColor.channel
func (c *SassColor) ChannelByName(channel string) (float64, error) {
	switch channel {
	case c.space.Channels()[0].Name:
		return c.channel0, nil
	case c.space.Channels()[1].Name:
		return c.channel1, nil
	case c.space.Channels()[2].Name:
		return c.channel2, nil
	case "alpha":
		return c.alpha, nil
	default:
		s, err := c.String()
		if err != nil {
			return 0, err
		}
		return 0, sasscommon.NewSassScriptException(fmt.Sprintf("Color %s doesn't have a channel named %q", s, channel), nil)
	}
}

// IsChannelMissingByName reports whether the named channel ("alpha"
// included) was written as missing, or a SassScriptException when the color
// has no such channel.
// Matches Dart: SassColor.isChannelMissing
func (c *SassColor) IsChannelMissingByName(channel string) (bool, error) {
	switch channel {
	case c.space.Channels()[0].Name:
		return c.missing[0], nil
	case c.space.Channels()[1].Name:
		return c.missing[1], nil
	case c.space.Channels()[2].Name:
		return c.missing[2], nil
	case "alpha":
		return c.missing[3], nil
	default:
		s, err := c.String()
		if err != nil {
			return false, err
		}
		return false, sasscommon.NewSassScriptException(fmt.Sprintf("Color %s doesn't have a channel named %q.", s, channel), new("channel"))
	}
}

// IsChannelPowerlessByName reports whether the named channel ("alpha"
// included, always potent) has no visual effect, or a SassScriptException
// when the color has no such channel.
// Matches Dart: SassColor.isChannelPowerless
func (c *SassColor) IsChannelPowerlessByName(channel string) (bool, error) {
	switch channel {
	case c.space.Channels()[0].Name:
		return c.IsChannel0Powerless(), nil
	case c.space.Channels()[1].Name:
		return c.IsChannel1Powerless(), nil
	case c.space.Channels()[2].Name:
		return c.IsChannel2Powerless(), nil
	case "alpha":
		return false, nil
	default:
		s, err := c.String()
		if err != nil {
			return false, err
		}
		return false, sasscommon.NewSassScriptException(fmt.Sprintf("Color %s doesn't have a channel named %q.", s, channel), new("channel"))
	}
}

// --- Interpolation ---

// Interpolate returns a color partway between c and other in method's space,
// following the CSS Color 4 interpolation procedure: both sides convert to
// the method space, channels missing on one side take the other side's
// value, and the mix weights by premultiplied alpha. Weight selects how
// much of c survives (nil means 0.5); fuzzy 0 returns other and fuzzy 1
// returns c, while values outside 0..1 are an error. The result converts
// back to c's space, dropping legacy missing channels unless legacyMissing
// keeps them. The engine lives in interpolateColors.
// Matches Dart: SassColor.interpolate
func (c *SassColor) Interpolate(other *SassColor, method InterpolationMethod, legacyMissing bool, weight *float64) (*SassColor, error) {
	return interpolateColors(c, other, method, legacyMissing, weight)
}

// Format returns the original syntax of the color for serialization.
// Matches Dart: SassColor format getter
func (c *SassColor) Format() ColorFormat { return c.format }

// SetFormat records the original syntax of the color for serialization. It
// is only meaningful when the color lives in the rgb space.
// Matches Dart: SassColor format setter
func (c *SassColor) SetFormat(f ColorFormat) { c.format = f }

// --- Value interface ---
// Colors are single-value lists: undecided separator, no brackets, truthy,
// never blank, and never special numbers or variables.

func (c *SassColor) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitColor(c) }

func (c *SassColor) isValue()         {}
func (c *SassColor) TryMap() *SassMap { return nil }

// IsTruthy reports true: colors always count as true.
// Matches Dart: Value.isTruthy
func (c *SassColor) IsTruthy() bool { return true }

// Separator returns ListSeparatorUndecided: a color is a single-value list.
// Matches Dart: Value.separator
func (c *SassColor) Separator() ListSeparator { return ListSeparatorUndecided }

// HasBrackets reports false: a color as a list never has brackets.
// Matches Dart: Value.hasBrackets
func (c *SassColor) HasBrackets() bool { return false }

// AsList returns the color as a single-value list.
// Matches Dart: Value.asList
func (c *SassColor) AsList() ([]Value, error) { return []Value{c}, nil }

// LengthAsList returns 1 without allocating the list.
// Matches Dart: Value.lengthAsList
func (c *SassColor) LengthAsList() int { return 1 }

// IsBlank reports false: colors never render as the empty string.
// Matches Dart: Value.isBlank
func (c *SassColor) IsBlank() bool { return false }

// IsSpecialNumber reports false: colors are never treated as numbers.
// Matches Dart: Value.isSpecialNumber
func (c *SassColor) IsSpecialNumber() bool { return false }

// IsSpecialVariable reports false: colors are never var() calls.
// Matches Dart: Value.isSpecialVariable
func (c *SassColor) IsSpecialVariable() bool { return false }

// RealNull returns the color itself: only SassNull reports otherwise.
// Matches Dart: Value.realNull
func (c *SassColor) RealNull() Value { return DefaultRealNull(c) }

// ToCssString returns the color's valid CSS representation; quote controls
// string quoting elsewhere and has no effect on colors.
// Matches Dart: Value.toCssString
func (c *SassColor) ToCssString(quote bool) (string, error) { return SerializeValue(c, quote) }

// String returns the inspect representation of the color.
// Matches Dart: Value.toString
func (c *SassColor) String() (string, error) { return SerializeValueInspect(c) }

// SingleEquals uses the Value default: both sides render as CSS around "=".
// Matches Dart: Value.singleEquals
func (c *SassColor) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(c, other) }

// Plus throws Undefined operation for number and color operands (Sass has no
// color arithmetic) and falls back to string concatenation otherwise.
// Matches Dart: SassColor.plus
func (c *SassColor) Plus(other Value) (Value, error) {
	if _, ok := other.(SassNumber); !ok {
		if _, ok := other.(*SassColor); !ok {
			return DefaultPlus(c, other)
		}
	}
	s, err := c.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s + %s\".", s, otherStr), nil)
}

// Minus throws Undefined operation for number and color operands and falls
// back to the `<css>-<css>` default otherwise.
// Matches Dart: SassColor.minus
func (c *SassColor) Minus(other Value) (Value, error) {
	if _, ok := other.(SassNumber); !ok {
		if _, ok := other.(*SassColor); !ok {
			return DefaultMinus(c, other)
		}
	}
	s, err := c.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s - %s\".", s, otherStr), nil)
}
func (c *SassColor) Times(other Value) (Value, error) { return DefaultTimes(c, other) }

// DividedBy throws Undefined operation for number and color operands and
// falls back to the slash-separation default otherwise.
// Matches Dart: SassColor.dividedBy
func (c *SassColor) DividedBy(other Value) (Value, error) {
	if _, ok := other.(SassNumber); !ok {
		if _, ok := other.(*SassColor); !ok {
			return DefaultDividedBy(c, other)
		}
	}
	s, err := c.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s / %s\".", s, otherStr), nil)
}
func (c *SassColor) Modulo(other Value) (Value, error)      { return DefaultModulo(c, other) }
func (c *SassColor) GreaterThan(other Value) (Value, error) { return DefaultGreaterThan(c, other) }
func (c *SassColor) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(c, other)
}
func (c *SassColor) LessThan(other Value) (Value, error) { return DefaultLessThan(c, other) }
func (c *SassColor) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(c, other)
}
func (c *SassColor) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(c) }
func (c *SassColor) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(c) }
func (c *SassColor) UnaryDivide() (Value, error) { return DefaultUnaryDivide(c) }
func (c *SassColor) UnaryNot() (Value, error)    { return DefaultUnaryNot(c) }

// Equals reports Sass color equality: legacy colors compare in the rgb
// space (so different legacy spaces can still match), while modern colors
// must share a space. Channels compare fuzzily with missing-aware
// comparison, alpha included.
// Matches Dart: SassColor.operator==
func (c *SassColor) Equals(other Value) bool {
	if oc, ok := other.(*SassColor); ok {
		return c.ColorEquals(oc)
	}
	return false
}

// ColorEquals is the typed form of Equals, shared with the equality probe
// inside the legacy branch.
// Matches Dart: SassColor.operator==
func (c *SassColor) ColorEquals(other *SassColor) bool {
	if c.space.IsLegacy() {
		if !other.space.IsLegacy() {
			return false
		}
		if !util.FuzzyEqualsNullable(c.alpha, other.alpha, c.missing[3], other.missing[3]) {
			return false
		}
		if c.space == other.space {
			return util.FuzzyEqualsNullable(c.channel0, other.channel0, c.missing[0], other.missing[0]) &&
				util.FuzzyEqualsNullable(c.channel1, other.channel1, c.missing[1], other.missing[1]) &&
				util.FuzzyEqualsNullable(c.channel2, other.channel2, c.missing[2], other.missing[2])
		}
		cRGB, err := c.ToSpace(RgbColorSpace, nil)
		if err != nil {
			return false
		}
		otherRGB, err := other.ToSpace(RgbColorSpace, nil)
		if err != nil {
			return false
		}
		return cRGB.Equals(otherRGB)
	}

	return c.space == other.space &&
		util.FuzzyEqualsNullable(c.channel0, other.channel0, c.missing[0], other.missing[0]) &&
		util.FuzzyEqualsNullable(c.channel1, other.channel1, c.missing[1], other.missing[1]) &&
		util.FuzzyEqualsNullable(c.channel2, other.channel2, c.missing[2], other.missing[2]) &&
		util.FuzzyEqualsNullable(c.alpha, other.alpha, c.missing[3], other.missing[3])
}

// HashCode returns a hash consistent with Equals: legacy colors hash their
// rgb conversion (the space they compare in), modern colors fold the space
// hash in with the fuzzy-hashed channels and alpha.
// Matches Dart: SassColor.hashCode
func (c *SassColor) HashCode() int {
	if c.space.IsLegacy() {
		rgb, err := c.ToSpace(RgbColorSpace, nil)
		if err != nil {
			panic("BUG: failed to convert legacy color to RGB for hashing: " + err.Error())
		}
		return util.FuzzyHashCode(rgb.channel0) ^ util.FuzzyHashCode(rgb.channel1) ^ util.FuzzyHashCode(rgb.channel2) ^ util.FuzzyHashCode(rgb.alpha)
	}
	return hashColorSpace(c.space) ^ util.FuzzyHashCode(c.channel0) ^ util.FuzzyHashCode(c.channel1) ^ util.FuzzyHashCode(c.channel2) ^ util.FuzzyHashCode(c.alpha)
}

// hashColorSpace hashes a color space by name for HashCode. It is Go-only
// glue: Dart hashes the space object directly, which Go's interface values
// don't support.
func hashColorSpace(space ColorSpace) int {
	name := space.Name()
	hash := 0
	for _, ch := range name {
		hash = hash*31 + int(ch)
	}
	return hash
}

// --- Helpers ---

// clamp01 clamps v to the 0..1 range. It is Go-only glue with no Dart
// counterpart.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// normalizeHue wraps h into [0, 360) with Dart's lossy double-modulo form.
// Zero and non-finite hues (including NaN) normalize to 0 instead.
// Matches Dart: SassColor._normalizeHue
func normalizeHue(h float64) float64 {
	if math.IsInf(h, 0) || math.IsNaN(h) {
		return 0
	}
	return math.Mod(math.Mod(h, 360)+360, 360)
}

// normalizeHueOptional is the nil-tolerant form of normalizeHue for
// channels that may be missing: nil stays nil (missing), non-finite stays 0.
// When the sibling channel (saturation or chroma) is negative, the hue
// shifts 180° to compensate for the negation applied there.
// Matches Dart: SassColor._normalizeHue (invert variant)
func normalizeHueOptional(hue, other *float64) float64 {
	if hue == nil {
		return 0
	}
	if math.IsInf(*hue, 0) || math.IsNaN(*hue) {
		return 0
	}
	offset := 0.0
	if other != nil && util.FuzzyLessThan(*other, 0) {
		offset = 180
	}
	return math.Mod(math.Mod(*hue, 360)+360+offset, 360)
}

// --- Factory constructors ---
// A nil channel means missing (CSS `none`); most factories take a concrete
// alpha that must lie between 0 and 1, while the Internal variants accept
// nil channels throughout.

// newColorForSpaceNoCheck builds a color in space with the given channels
// and no preprocessing of any channel: raw values land as-is, nils become
// missing.
// Matches Dart: SassColor._forSpace
func newColorForSpaceNoCheck(space ColorSpace, c0, c1, c2, alpha *float64, format *ColorFormat) *SassColor {
	c0v, c1v, c2v, av := float64(0), float64(0), float64(0), float64(0)
	if c0 != nil {
		c0v = *c0
	}
	if c1 != nil {
		c1v = *c1
	}
	if c2 != nil {
		c2v = *c2
	}
	if alpha != nil {
		av = *alpha
	}
	var f ColorFormat
	if format != nil {
		f = *format
	}
	return &SassColor{
		space:    space,
		channel0: c0v,
		channel1: c1v,
		channel2: c2v,
		alpha:    av,
		missing:  [4]bool{c0 == nil, c1 == nil, c2 == nil, alpha == nil},
		format:   f,
	}
}

// NewColorForSpaceInternal creates a color with per-space preprocessing
// (hue wrapping, negative saturation/chroma folding, linear normalization),
// range-checking alpha to 0..1 first. A nil channel means missing.
// Matches Dart: SassColor.forSpaceInternal
func NewColorForSpaceInternal(space ColorSpace, c0, c1, c2, alpha *float64) (*SassColor, error) {
	if alpha != nil {
		_, err := util.FuzzyAssertRange(*alpha, 0, 1, new("alpha"))
		if err != nil {
			return nil, err
		}
	}
	return newColorForSpaceInternalNoCheck(space, c0, c1, c2, alpha), nil
}

// newColorForSpaceInternalNoCheck is the unchecked tail of
// NewColorForSpaceInternal for callers whose alpha is already valid. HSL
// negates a negative saturation (shifting hue 180°), LCH/OKLCH do the same
// for chroma, and every other continuous channel normalizes NaN and negative
// zero to 0.
// Matches Dart: SassColor.forSpaceInternal switch body
func newColorForSpaceInternalNoCheck(space ColorSpace, c0, c1, c2, alpha *float64) *SassColor {
	switch space {
	case HslColorSpace:
		var s float64
		if c1 != nil {
			s = *c1
			if util.FuzzyLessThan(s, 0) {
				s = -s
			}
		}
		h := normalizeHueOptional(c0, c1)
		var hPtr, sPtr *float64
		if c0 != nil {
			hPtr = &h
		}
		if c1 != nil {
			sv := util.NormalizeLinear(s)
			sPtr = &sv
		}
		return newColorForSpaceNoCheck(space, hPtr, sPtr, normalizeLinearPtr(c2), normalizeLinearPtr(alpha), nil)
	case HwbColorSpace:
		var hPtr *float64
		if c0 != nil {
			h := normalizeHue(*c0)
			hPtr = &h
		}
		return newColorForSpaceNoCheck(space, hPtr, normalizeLinearPtr(c1), normalizeLinearPtr(c2), normalizeLinearPtr(alpha), nil)
	case LchColorSpace, OklchColorSpace:
		var ch float64
		if c1 != nil {
			ch = *c1
			if util.FuzzyLessThan(ch, 0) {
				ch = -ch
			}
		}
		h := normalizeHueOptional(c2, c1)
		var chPtr, hPtr *float64
		if c1 != nil {
			cv := util.NormalizeLinear(ch)
			chPtr = &cv
		}
		if c2 != nil {
			hPtr = &h
		}
		return newColorForSpaceNoCheck(space, normalizeLinearPtr(c0), chPtr, hPtr, normalizeLinearPtr(alpha), nil)
	default:
		return newColorForSpaceNoCheck(space, normalizeLinearPtr(c0), normalizeLinearPtr(c1), normalizeLinearPtr(c2), normalizeLinearPtr(alpha), nil)
	}
}

// normalizeLinearPtr normalizes NaN and negative zero to 0, passing nil
// (missing) through. It is the Go spelling of Dart's _normalizeLinear.
// Matches Dart: SassColor._normalizeLinear
func normalizeLinearPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	n := util.NormalizeLinear(*v)
	return &n
}

// NewColorForSpace creates a color in space from three channel values and
// an alpha. The missing mask flags CSS-missing (`none`) channels; every
// space holds exactly three channels, so the fixed array replaces Dart's
// length check. Alpha must lie between 0 and 1.
// Matches Dart: SassColor.forSpace
func NewColorForSpace(space ColorSpace, channels [3]float64, alpha float64, missing [4]bool) (*SassColor, error) {
	var c0, c1, c2, a *float64
	if !missing[0] {
		c0 = &channels[0]
	}
	if !missing[1] {
		c1 = &channels[1]
	}
	if !missing[2] {
		c2 = &channels[2]
	}
	if !missing[3] {
		a = &alpha
	}
	return NewColorForSpaceInternal(space, c0, c1, c2, a)
}

// NewColorRGB creates an sRGB color from 0-255 channel values and an alpha
// between 0 and 1. It stores raw (no hue/chroma folding applies to rgb).
// Matches Dart: SassColor.rgb
func NewColorRGB(red, green, blue float64, alpha float64) (*SassColor, error) {
	a, err := util.FuzzyAssertRange(alpha, 0, 1, new("alpha"))
	if err != nil {
		return nil, err
	}
	return newColorForSpaceNoCheck(RgbColorSpace, &red, &green, &blue, &a, nil), nil
}

// NewColorRGBInternal creates an sRGB color that also records its original
// format (nil format means none). Any nil channel — alpha included — means
// missing; a non-nil alpha must lie between 0 and 1.
// Matches Dart: SassColor.rgbInternal
func NewColorRGBInternal(red, green, blue, alpha *float64, format *ColorFormat) (*SassColor, error) {
	if alpha != nil {
		_, err := util.FuzzyAssertRange(*alpha, 0, 1, new("alpha"))
		if err != nil {
			return nil, err
		}
	}
	return newColorForSpaceNoCheck(RgbColorSpace, red, green, blue, alpha, format), nil
}

// NewColorHSL creates a color in HSL space from hue degrees, saturation and
// lightness percentages, and an alpha between 0 and 1. The hue wraps to
// [0, 360); a negative saturation is negated with the hue shifted 180°.
// Matches Dart: SassColor.hsl
func NewColorHSL(hue, saturation, lightness float64, alpha float64) (*SassColor, error) {
	a, err := util.FuzzyAssertRange(alpha, 0, 1, new("alpha"))
	if err != nil {
		return nil, err
	}
	invert := util.FuzzyLessThan(saturation, 0)
	h := normalizeHue(hue)
	if invert {
		h = normalizeHue(hue + 180)
		saturation = -saturation
	}
	sat := saturation
	return NewColorForSpaceInternal(HslColorSpace, &h, &sat, &lightness, &a)
}

// NewColorHWB creates a color in HWB space from hue degrees, whiteness and
// blackness percentages, and an alpha between 0 and 1. The hue wraps to
// [0, 360).
// Matches Dart: SassColor.hwb
func NewColorHWB(hue, whiteness, blackness float64, alpha float64) (*SassColor, error) {
	a, err := util.FuzzyAssertRange(alpha, 0, 1, new("alpha"))
	if err != nil {
		return nil, err
	}
	h := normalizeHue(hue)
	return NewColorForSpaceInternal(HwbColorSpace, &h, &whiteness, &blackness, &a)
}

// NewColorSRGB creates a color in sRGB space from 0-1 channel values and an
// alpha between 0 and 1.
// Matches Dart: SassColor.srgb
func NewColorSRGB(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(SrgbColorSpace, &red, &green, &blue, &alpha)
}

// NewColorSRGBLinear creates a color in linear-light sRGB space from 0-1
// channel values and an alpha between 0 and 1.
// Matches Dart: SassColor.srgbLinear
func NewColorSRGBLinear(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(SrgbLinearColorSpace, &red, &green, &blue, &alpha)
}

// NewColorDisplayP3 creates a color in Display P3 space from 0-1 channel
// values and an alpha between 0 and 1.
// Matches Dart: SassColor.displayP3
func NewColorDisplayP3(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(DisplayP3ColorSpace, &red, &green, &blue, &alpha)
}

// NewColorDisplayP3Linear creates a color in linear-light Display P3 space
// from 0-1 channel values and an alpha between 0 and 1.
// Matches Dart: SassColor.displayP3Linear
func NewColorDisplayP3Linear(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(DisplayP3LinearColorSpace, &red, &green, &blue, &alpha)
}

// NewColorA98RGB creates a color in A98 RGB space from 0-1 channel values
// and an alpha between 0 and 1.
// Matches Dart: SassColor.a98Rgb
func NewColorA98RGB(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(A98RgbColorSpace, &red, &green, &blue, &alpha)
}

// NewColorProPhotoRGB creates a color in ProPhoto RGB space from 0-1 channel
// values and an alpha between 0 and 1.
// Matches Dart: SassColor.prophotoRgb
func NewColorProPhotoRGB(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(ProphotoRgbColorSpace, &red, &green, &blue, &alpha)
}

// NewColorRec2020 creates a color in Rec2020 space from 0-1 channel values
// and an alpha between 0 and 1.
// Matches Dart: SassColor.rec2020
func NewColorRec2020(red, green, blue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(Rec2020ColorSpace, &red, &green, &blue, &alpha)
}

// NewColorXYZD50 creates a color in XYZ D50 space from x/y/z values and an
// alpha between 0 and 1.
// Matches Dart: SassColor.xyzD50
func NewColorXYZD50(x, y, z, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(XyzD50ColorSpace, &x, &y, &z, &alpha)
}

// NewColorXYZD65 creates a color in XYZ D65 space from x/y/z values and an
// alpha between 0 and 1.
// Matches Dart: SassColor.xyzD65
func NewColorXYZD65(x, y, z, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(XyzD65ColorSpace, &x, &y, &z, &alpha)
}

// NewColorLab creates a color in CIE Lab space from lightness/a/b values
// and an alpha between 0 and 1.
// Matches Dart: SassColor.lab
func NewColorLab(lightness, a, b, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(LabColorSpace, &lightness, &a, &b, &alpha)
}

// NewColorLCH creates a color in CIE LCH space from lightness, chroma and hue
// values and an alpha between 0 and 1. A negative chroma is negated with the
// hue shifted 180°.
// Matches Dart: SassColor.lch
func NewColorLCH(lightness, chroma, hue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(LchColorSpace, &lightness, &chroma, &hue, &alpha)
}

// NewColorOKLab creates a color in OKLab space from lightness/a/b values and
// an alpha between 0 and 1.
// Matches Dart: SassColor.oklab
func NewColorOKLab(lightness, a, b, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(OklabColorSpace, &lightness, &a, &b, &alpha)
}

// NewColorOKLCH creates a color in OKLCH space from lightness, chroma and hue
// values and an alpha between 0 and 1. A negative chroma is negated with the
// hue shifted 180°.
// Matches Dart: SassColor.oklch
func NewColorOKLCH(lightness, chroma, hue, alpha float64) (*SassColor, error) {
	return NewColorForSpaceInternal(OklchColorSpace, &lightness, &chroma, &hue, &alpha)
}
