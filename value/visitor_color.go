// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitColor section:
// _writeLegacyColor, _writeChannel, _tryHexOrNamedRgb, _canUseHex,
// _writeRgb, _tryIntegerRgbChannels, _writeHsl, _writeHwb,
// _writeColorFunction, _maybeWriteSlashAlpha, plus the gamut-mapped
// lab/lch/oklab/oklch and color-mix arms of visitColor)

import (
	"math"
	"strconv"

	"github.com/bancek/go-sass/sourcemapbuffer"
	"github.com/bancek/go-sass/util"
)

// writeLegacyColor emits a color in one of the three interchangeable
// legacy spaces (rgb, hsl, hwb), choosing the shortest spelling that all
// supported browsers accept.
//
// The order matters: out-of-gamut colors in normal mode can only be
// written accurately as HSL (no clamping at parse time), so they take that
// path first. Compressed output renders the rgb and hsl candidates and
// keeps the shorter one (with a two-character handicap for HSL's percent
// signs). Expanded output honors the color's own space and stored format:
// hsl space stays hsl, hwb space stays hwb in inspect mode, the
// rgb-function format forces rgb(), and preserved source text is replayed
// verbatim. Generated opaque colors prefer names, then hex; HWB colors
// with no hex spelling fall back to HSL as the clearer record of intent,
// and an IE-era workaround keeps generated transparent colors in rgba
// form. Everything else renders as rgb().
//
// Matches Dart: _SerializeVisitor._writeLegacyColor
func (sv *SerializeVisitor) writeLegacyColor(c *SassColor) error {
	alpha := c.Alpha()
	opaque := util.FuzzyEquals(alpha, 1)

	if !c.IsInGamut() && !sv.inspect {
		return sv.writeHsl(c)
	}

	if !sv.isCompressed() {
		if c.Space() == HslColorSpace {
			return sv.writeHsl(c)
		} else if sv.inspect && c.Space() == HwbColorSpace {
			return sv.writeHwb(c)
		}

		if c.Format() == ColorFormatRGBFunction {
			return sv.writeRgb(c)
		}

		switch f := c.Format().(type) {
		case *SpanColorFormat:
			orig, err := f.Original()
			if err != nil {
				return err
			}
			_, _ = sv.sb.WriteString(orig)
			return nil
		}
	} else {
		rgb, err := c.ToSpace(RgbColorSpace, nil)
		if err != nil {
			return err
		}
		if opaque {
			ok, err := sv.tryHexOrNamedRgb(rgb)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}

		// Render both candidates fully and emit the shorter spelling (plus
		// two characters of HSL handicap for its `%` signs), exactly like
		// Dart `_capture(() => _writeRgb/_writeHsl)`.
		rgbString, err := sv.capture(func() error { return sv.writeRgb(rgb) })
		if err != nil {
			return err
		}
		hsl, err := rgb.ToSpace(HslColorSpace, nil)
		if err != nil {
			return err
		}
		hslString, err := sv.capture(func() error { return sv.writeHsl(hsl) })
		if err != nil {
			return err
		}

		if len(rgbString) <= len(hslString)+2 {
			_, _ = sv.sb.WriteString(rgbString)
		} else {
			_, _ = sv.sb.WriteString(hslString)
		}
		return nil
	}

	if opaque {
		rgb, err := c.ToSpace(RgbColorSpace, nil)
		if err != nil {
			return err
		}
		if name := colorNameForSassColor(rgb); name != "" {
			_, _ = sv.sb.WriteString(name)
			return nil
		}
		if canUseHexForChannel(rgb.Channel0()) &&
			canUseHexForChannel(rgb.Channel1()) &&
			canUseHexForChannel(rgb.Channel2()) {
			_ = sv.sb.WriteByte('#')
			sv.writeHexComponent(int(math.Round(rgb.Channel0())))
			sv.writeHexComponent(int(math.Round(rgb.Channel1())))
			sv.writeHexComponent(int(math.Round(rgb.Channel2())))
			return nil
		}
	}

	if c.Space() == HwbColorSpace {
		return sv.writeHsl(c)
	}
	return sv.writeRgb(c)
}

// writeColorWithMissing emits a legacy-space color that has at least one
// missing channel or alpha, using the modern space-separated syntax with
// "none" placeholders and a slash alpha. HSL and HWB spell their hue in
// degrees (expanded only) and their other channels as percentages.
//
// Matches Dart: _SerializeVisitor.visitColor (rgb/hsl/hwb arms reached
// with missing channels)
func (sv *SerializeVisitor) writeColorWithMissing(c *SassColor) {
	space := c.Space()
	switch space {
	case RgbColorSpace:
		_, _ = sv.sb.WriteString("rgb(")
		sv.writeChannel(c.Channel0OrNil(), "")
		_ = sv.sb.WriteByte(' ')
		sv.writeChannel(c.Channel1OrNil(), "")
		_ = sv.sb.WriteByte(' ')
		sv.writeChannel(c.Channel2OrNil(), "")
		sv.maybeWriteSlashAlpha(c)
		_ = sv.sb.WriteByte(')')

	case HslColorSpace, HwbColorSpace:
		_, _ = sv.sb.WriteString(space.Name())
		_ = sv.sb.WriteByte('(')
		deg := "deg"
		if sv.isCompressed() {
			deg = ""
		}
		sv.writeChannel(c.Channel0OrNil(), deg)
		_ = sv.sb.WriteByte(' ')
		sv.writeChannel(c.Channel1OrNil(), "%")
		_ = sv.sb.WriteByte(' ')
		sv.writeChannel(c.Channel2OrNil(), "%")
		sv.maybeWriteSlashAlpha(c)
		_ = sv.sb.WriteByte(')')

	case LabColorSpace, LchColorSpace, OklabColorSpace, OklchColorSpace:
		sv.writeLabLchColor(c)

	default:
		sv.writeColorFunction(c)
	}
}

// writeColorMix renders an out-of-gamut lab/lch/oklab/oklch color through
// color-mix() so the declared space survives without clamping: the color
// is first converted to unrestricted XYZ, mixed at full weight against
// black (expanded) or red (compressed, the shorter name), keeping the
// target space in the mix descriptor. Per the spec no clamping applies,
// though browsers in practice still clamp values they cannot display.
//
// Matches Dart: _SerializeVisitor.visitColor (color-mix arms)
func (sv *SerializeVisitor) writeColorMix(c *SassColor) error {
	_, _ = sv.sb.WriteString("color-mix(in ")
	_, _ = sv.sb.WriteString(c.Space().Name())
	_, _ = sv.sb.WriteString(sv.commaSep())
	xyz, err := c.ToSpace(XyzD65ColorSpace, nil)
	if err != nil {
		return err
	}
	sv.writeColorFunction(xyz)
	sv.writeOptionalSpace()
	_, _ = sv.sb.WriteString("100%")
	_, _ = sv.sb.WriteString(sv.commaSep())
	if sv.isCompressed() {
		_, _ = sv.sb.WriteString("red")
	} else {
		_, _ = sv.sb.WriteString("black")
	}
	_ = sv.sb.WriteByte(')')
	return nil
}

// writeLabLchColor emits a lab, lch, oklab, or oklch color in its own
// function syntax. Out-of-bounds colors with missing channels cannot go
// through color-mix, so they use the less widely supported but more
// expressive relative-color form ("from red/black ...") which never clamps.
// The first channel renders as a percentage of its space maximum in
// expanded mode; polar hue gains a deg suffix outside compressed output.
//
// Matches Dart: _SerializeVisitor.visitColor (lab/oklab/lch/oklch arm)
func (sv *SerializeVisitor) writeLabLchColor(c *SassColor) {
	space := c.Space()
	_, _ = sv.sb.WriteString(space.Name())
	_ = sv.sb.WriteByte('(')

	chs := SpaceChannels(space)
	polar := chs[2].IsPolarAngle

	if !sv.inspect &&
		(!util.FuzzyInRange(c.Channel0(), 0, 100) ||
			(polar && util.FuzzyLessThan(c.Channel1(), 0))) {
		_, _ = sv.sb.WriteString("from ")
		if sv.isCompressed() {
			_, _ = sv.sb.WriteString("red")
		} else {
			_, _ = sv.sb.WriteString("black")
		}
		_ = sv.sb.WriteByte(' ')
	}

	if !sv.isCompressed() && !c.IsChannel0Missing() {
		max := chs[0].Max
		sv.writeNumber(c.Channel0() * 100 / max)
		_ = sv.sb.WriteByte('%')
	} else {
		sv.writeChannel(c.Channel0OrNil(), "")
	}
	_ = sv.sb.WriteByte(' ')
	sv.writeChannel(c.Channel1OrNil(), "")
	_ = sv.sb.WriteByte(' ')
	ch2Unit := ""
	if polar && !sv.isCompressed() {
		ch2Unit = "deg"
	}
	sv.writeChannel(c.Channel2OrNil(), ch2Unit)
	sv.maybeWriteSlashAlpha(c)
	_ = sv.sb.WriteByte(')')
}

// writeColorFunction emits colors from the remaining modern spaces with
// the generic color() function syntax: the space name, three channels,
// and a slash alpha.
//
// Matches Dart: _SerializeVisitor._writeColorFunction
func (sv *SerializeVisitor) writeColorFunction(c *SassColor) {
	_, _ = sv.sb.WriteString("color(")
	_, _ = sv.sb.WriteString(c.Space().Name())
	for i := range 3 {
		_ = sv.sb.WriteByte(' ')
		var ch *float64
		switch i {
		case 0:
			ch = c.Channel0OrNil()
		case 1:
			ch = c.Channel1OrNil()
		case 2:
			ch = c.Channel2OrNil()
		}
		sv.writeChannel(ch, "")
	}
	sv.maybeWriteSlashAlpha(c)
	_ = sv.sb.WriteByte(')')
}

// writeChannel emits one color channel that may be missing: "none" for a
// missing channel, the plain number plus unit for finite values, and the
// number-with-unit spelling (routing through VisitNumber so infinities
// become calc spellings) for non-finite values.
//
// Matches Dart: _SerializeVisitor._writeChannel
func (sv *SerializeVisitor) writeChannel(channel *float64, unit string) error {
	if channel == nil {
		_, err := sv.sb.WriteString("none")
		return err
	}
	return sv.writeChannelValue(*channel, unit)
}

// writeChannelValue emits a non-missing channel value with its unit.
// Non-finite values detour through VisitNumber (attaching the unit when
// one is present) so they gain the same calc spelling as standalone
// infinities; finite values take the plain number path.
//
// Matches Dart: _SerializeVisitor._writeChannel (finite/non-finite split)
func (sv *SerializeVisitor) writeChannelValue(v float64, unit string) error {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		if unit != "" {
			_, err := sv.VisitNumber(NewSingleUnitNumber(v, unit))
			return err
		}
		_, err := sv.VisitNumber(NewUnitlessNumber(v))
		return err
	}
	sv.writeNumber(v)
	if unit != "" {
		_, _ = sv.sb.WriteString(unit)
	}
	return nil
}

// maybeWriteSlashAlpha appends the slash-alpha component unless the color
// is fully opaque, using optional spaces around the slash so compressed
// output stays tight.
//
// Matches Dart: _SerializeVisitor._maybeWriteSlashAlpha
func (sv *SerializeVisitor) maybeWriteSlashAlpha(c *SassColor) {
	if util.FuzzyEquals(c.Alpha(), 1) {
		return
	}
	sv.writeOptionalSpace()
	_ = sv.sb.WriteByte('/')
	sv.writeOptionalSpace()
	sv.writeChannel(c.AlphaOrNil(), "")
}

// tryHexOrNamedRgb writes c in the shortest of the name/short-hex/long-hex
// spellings and reports true. It writes nothing and reports false when c
// cannot be a hex color. c must already be converted to sRGB. A name wins
// only when it is no longer than the applicable hex spelling (4 with short
// hex available, else 7).
//
// Matches Dart: _SerializeVisitor._tryHexOrNamedRgb (compressed path)
func (sv *SerializeVisitor) tryHexOrNamedRgb(c *SassColor) (bool, error) {
	ok, err := sv.canUseHex(c)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// Use raw channel values (Channel0/1/2) rather than the legacy Red/Green/Blue
	// getters, which math.Round internally. This matches Dart's _tryIntegerRgb
	// which uses rgb.channel0 etc. directly.
	redInt := int(math.Round(c.Channel0()))
	greenInt := int(math.Round(c.Channel1()))
	blueInt := int(math.Round(c.Channel2()))

	shortHex := isSymmetricalHex(redInt) && isSymmetricalHex(greenInt) && isSymmetricalHex(blueInt)
	maxLen := 7
	if shortHex {
		maxLen = 4
	}
	// Uses colorNameForSassColor (fuzzy equality verification) to match Dart's
	// namesByColor[rgb] lookup, rather than ColorNameForInts which does exact
	// integer matching and can produce false-positive color names when channels
	// are near but not at integer values.
	name := colorNameForSassColor(c)
	if name != "" && len(name) <= maxLen {
		_, _ = sv.sb.WriteString(name)
	} else if shortHex {
		_ = sv.sb.WriteByte('#')
		_ = sv.sb.WriteByte(byte(hexCharFor(redInt & 0xF)))
		_ = sv.sb.WriteByte(byte(hexCharFor(greenInt & 0xF)))
		_ = sv.sb.WriteByte(byte(hexCharFor(blueInt & 0xF)))
	} else {
		_ = sv.sb.WriteByte('#')
		sv.writeHexComponent(redInt)
		sv.writeHexComponent(greenInt)
		sv.writeHexComponent(blueInt)
	}
	return true, nil
}

// canUseHex reports whether c's raw channels can each be written as a
// two-character hex pair: near-integer values within [0, 256). It reads the
// raw channels rather than the rounded legacy getters.
//
// Matches Dart: _SerializeVisitor._canUseHex
func (sv *SerializeVisitor) canUseHex(c *SassColor) (bool, error) {
	return canUseHexForChannel(c.Channel0()) &&
		canUseHexForChannel(c.Channel1()) &&
		canUseHexForChannel(c.Channel2()), nil
}

// canUseHexForChannel reports whether one channel value fits in a
// two-character hex pair: a near-integer within [0, 256).
//
// Matches Dart: _SerializeVisitor._canUseHexForChannel
func canUseHexForChannel(channel float64) bool {
	return util.FuzzyIsInt(channel) &&
		util.FuzzyGreaterThanOrEquals(channel, 0) &&
		util.FuzzyLessThan(channel, 256)
}

// isSymmetricalHex reports whether a channel byte has identical nibbles
// (such as 0xFF), which is what allows the short three-digit hex spelling.
//
// Matches Dart: _SerializeVisitor._isSymmetricalHex
func isSymmetricalHex(color int) bool {
	return color&0xF == color>>4
}

// writeHexComponent emits color as a two-character lowercase hex pair.
// The caller must keep color below 0x100.
//
// Matches Dart: _SerializeVisitor._writeHexComponent
func (sv *SerializeVisitor) writeHexComponent(color int) {
	_ = sv.sb.WriteByte(byte(hexCharFor(color >> 4)))
	_ = sv.sb.WriteByte(byte(hexCharFor(color & 0xF)))
}

// capture runs emit against a throwaway plain buffer and returns the text
// it would have produced, leaving the real buffer untouched. The color
// serializer uses it to compare the rgb and hsl candidate spellings.
//
// Matches Dart: _SerializeVisitor._capture
func (sv *SerializeVisitor) capture(emit func() error) (string, error) {
	old := sv.sb
	tmp := sourcemapbuffer.NewNoSourceMapBuffer()
	sv.sb = tmp
	err := emit()
	sv.sb = old
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}

// writeRgb emits color as an rgb() or rgba() legacy function. Integer
// channels take the plain spelling; otherwise every channel renders as a
// percentage of 255, using the original color's channels exactly like
// Dart (older browsers only accept integers or percentages in legacy
// rgb()). The alpha travels as a trailing comma argument.
//
// Matches Dart: _SerializeVisitor._writeRgb
func (sv *SerializeVisitor) writeRgb(c *SassColor) error {
	alpha := c.Alpha()
	opaque := util.FuzzyEquals(alpha, 1)
	rgb, err := c.ToSpace(RgbColorSpace, nil)
	if err != nil {
		return err
	}
	if opaque {
		_, _ = sv.sb.WriteString("rgb(")
	} else {
		_, _ = sv.sb.WriteString("rgba(")
	}

	ok, err := sv.tryIntegerRgbChannels(rgb)
	if err != nil {
		return err
	}
	if !ok {
		if err := sv.writeChannelValue(c.Channel0()*100/255, "%"); err != nil {
			return err
		}
		_, _ = sv.sb.WriteString(sv.commaSep())
		if err := sv.writeChannelValue(c.Channel1()*100/255, "%"); err != nil {
			return err
		}
		_, _ = sv.sb.WriteString(sv.commaSep())
		if err := sv.writeChannelValue(c.Channel2()*100/255, "%"); err != nil {
			return err
		}
	}

	if !opaque {
		_, _ = sv.sb.WriteString(sv.commaSep())
		sv.writeNumber(alpha)
	}
	_ = sv.sb.WriteByte(')')
	return nil
}

// tryIntegerRgbChannels writes rgb's channels plainly and reports true
// when all three are integers, and writes nothing and reports false
// otherwise. rgb must already be in the RGB space.
//
// Matches Dart: _SerializeVisitor._tryIntegerRgbChannels
func (sv *SerializeVisitor) tryIntegerRgbChannels(rgb *SassColor) (bool, error) {
	channels := [3]float64{rgb.Channel0(), rgb.Channel1(), rgb.Channel2()}
	var ints [3]int64
	for i, ch := range channels {
		v, ok := util.AsIntForSerialize(ch, sv.inspect)
		if !ok {
			return false, nil
		}
		ints[i] = v
	}

	_, _ = sv.sb.WriteString(strconv.FormatInt(ints[0], 10))
	_, _ = sv.sb.WriteString(sv.commaSep())
	_, _ = sv.sb.WriteString(strconv.FormatInt(ints[1], 10))
	_, _ = sv.sb.WriteString(sv.commaSep())
	_, _ = sv.sb.WriteString(strconv.FormatInt(ints[2], 10))
	return true, nil
}

// writeHsl emits color as an hsl() or hsla() legacy function: hue plain,
// saturation and lightness as percentages, and alpha as a trailing comma
// argument when translucent.
//
// Matches Dart: _SerializeVisitor._writeHsl
func (sv *SerializeVisitor) writeHsl(c *SassColor) error {
	alpha := c.Alpha()
	opaque := util.FuzzyEquals(alpha, 1)
	hsl, err := c.ToSpace(HslColorSpace, nil)
	if err != nil {
		return err
	}
	if opaque {
		_, _ = sv.sb.WriteString("hsl(")
	} else {
		_, _ = sv.sb.WriteString("hsla(")
	}
	if err := sv.writeChannelValue(hsl.Channel0(), ""); err != nil {
		return err
	}
	_, _ = sv.sb.WriteString(sv.commaSep())
	if err := sv.writeChannelValue(hsl.Channel1(), "%"); err != nil {
		return err
	}
	_, _ = sv.sb.WriteString(sv.commaSep())
	if err := sv.writeChannelValue(hsl.Channel2(), "%"); err != nil {
		return err
	}
	if !opaque {
		_, _ = sv.sb.WriteString(sv.commaSep())
		sv.writeNumber(alpha)
	}
	_ = sv.sb.WriteByte(')')
	return nil
}

// writeHwb emits color in the modern space-separated hwb() syntax: hue
// plain, whiteness and blackness as percentages, and a slash alpha when
// translucent. It serves inspect mode only, which is why it never uses the
// legacy comma syntax.
//
// Matches Dart: _SerializeVisitor._writeHwb
func (sv *SerializeVisitor) writeHwb(c *SassColor) error {
	hwb, err := c.ToSpace(HwbColorSpace, nil)
	if err != nil {
		return err
	}
	alpha := c.Alpha()
	_, _ = sv.sb.WriteString("hwb(")
	if err := sv.writeChannelValue(hwb.Channel0(), ""); err != nil {
		return err
	}
	_ = sv.sb.WriteByte(' ')
	if err := sv.writeChannelValue(hwb.Channel1(), "%"); err != nil {
		return err
	}
	_ = sv.sb.WriteByte(' ')
	if err := sv.writeChannelValue(hwb.Channel2(), "%"); err != nil {
		return err
	}
	if !util.FuzzyEquals(alpha, 1) {
		_, _ = sv.sb.WriteString(" / ")
		sv.writeNumber(alpha)
	}
	_ = sv.sb.WriteByte(')')
	return nil
}
