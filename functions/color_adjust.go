// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/color.dart (adjust slice: _adjust/_scale/
// _change closures, _updateComponents, _changeColor/_channelForChange,
// _scaleColor/_scaleChannel, _adjustColor/_adjustChannel,
// _sniffLegacyColorSpace, _checkPercent)

import (
	"fmt"
	"math"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// ---- adjust() / scale() / change() ----
// These port Dart's _adjust/_scale/_change closures through the shared
// _updateComponents engine: exactly one of the change/adjust/scale flags is
// set, selecting the write mode for the keyword channels.

// adjustColorCallable builds adjust($color, $kwargs...), which adds each
// keyword amount to its channel.
func adjustColorCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("adjust", "$color, $kwargs...", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		return updateComponents(ec, args, false, true, false)
	})
}

// scaleColorCallable builds scale($color, $kwargs...), which scales each
// keyword channel toward its bounds by a percentage factor.
func scaleColorCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("scale", "$color, $kwargs...", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		return updateComponents(ec, args, false, false, true)
	})
}

// changeColorCallable builds change($color, $kwargs...), which replaces each
// keyword channel outright (unquoted "none" marks a channel missing).
func changeColorCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("change", "$color, $kwargs...", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		return updateComponents(ec, args, true, false, false)
	})
}

// updateComponents implements color.change/adjust/scale. Exactly one of
// change, adjust, or scale selects the write mode. Only one positional
// argument is allowed (the color); everything else arrives by keyword in the
// trailing argument list. A legacy color with no explicit $space works in the
// legacy space its keywords sniff to; otherwise the color moves into the
// requested space first and the result converts back to the original space
// with legacy missing channels resolved to zero.
func updateComponents(ec *evalcontext.EvaluationContext, args []value.Value, change, adjust, scale bool) (value.Value, error) {
	// The second argument is the rest argument list holding the keywords.
	argumentList := args[1].(*value.SassArgumentList)

	// Positional rest arguments are rejected; channel updates pass by name only.
	argList, err := argumentList.AsList()
	if err != nil {
		return nil, err
	}
	if len(argList) > 0 {
		return nil, sasscommon.NewSassScriptException(
			"Only one positional argument is allowed. All other arguments must be passed by name.", nil)
	}

	// Copy the keyword map so $space and $alpha can be removed before the
	// remaining entries are matched to channels by name.
	keywords := orderedmap.New[string, value.Value]()
	for k, v := range argumentList.Keywords().Entries() {
		keywords.Put(k, v)
	}

	// Assert the color argument up front, before resolving the working space.
	originalColor, err := value.AssertColor(args[0], new("color"))
	if err != nil {
		return nil, err
	}

	// Pull $space out of the keywords; it selects the working space rather
	// than a channel, so it must not reach channel matching.
	var spaceKeyword value.Value
	if v, ok := keywords.Get("space"); ok {
		spaceKeyword = v
		keywords.Delete("space")
	}

	// Pull $alpha out likewise; it is handled separately from the space's
	// own channels.
	var alphaVal value.Value
	if v, ok := keywords.Get("alpha"); ok {
		alphaVal = v
		keywords.Delete("alpha")
	}

	if spaceKeyword != nil {
		spaceStr, err := value.AssertString(spaceKeyword, new("space"))
		if err != nil {
			return nil, err
		}
		if err := spaceStr.AssertUnquoted(); err != nil {
			return nil, sasscommon.NewSassScriptException(err.Error(), new("space"))
		}
	}

	// Determine the working color space
	var color *value.SassColor
	if spaceKeyword == nil && originalColor.IsLegacy() && keywords.Len() > 0 {
		// Sniff legacy space from keyword names
		sniffed := sniffLegacyColorSpace(keywords)
		if sniffed != nil {
			var csErr error
			color, csErr = originalColor.ToSpace(*sniffed, new(false))
			if csErr != nil {
				return nil, csErr
			}
		} else {
			color = originalColor
		}
	} else {
		var spaceOrNull value.Value = value.Null
		if spaceKeyword != nil {
			spaceOrNull = spaceKeyword
		}
		color, err = colorInSpace(originalColor, spaceOrNull, nil)
		if err != nil {
			return nil, err
		}
	}

	// Map keyword arguments to channel indices
	oldChannels := color.ChannelsOrNil()
	channelInfo := value.SpaceChannels(color.Space())
	channelArgs := make([]value.Value, len(oldChannels))
	for key, val := range keywords.Entries() {
		found := false
		for i, ch := range channelInfo {
			if ch.Name == key {
				channelArgs[i] = val
				found = true
				break
			}
		}
		if !found {
			return nil, sasscommon.NewSassScriptException(
				fmt.Sprintf("Color space %s doesn't have a channel with this name.", color.Space().Name()), new(key))
		}
	}

	var result *value.SassColor
	if change {
		result, err = changeColor(ec, color, channelArgs, alphaVal)
		if err != nil {
			return nil, err
		}
	} else {
		channelNumbers := make([]value.SassNumber, len(oldChannels))
		for i := range len(oldChannels) {
			if channelArgs[i] != nil {
				channelNumbers[i], err = value.AssertNumber(channelArgs[i], &channelInfo[i].Name)
				if err != nil {
					return nil, err
				}
			}
		}
		var alphaNumber value.SassNumber
		if alphaVal != nil {
			alphaNumber, err = value.AssertNumber(alphaVal, new("alpha"))
			if err != nil {
				return nil, err
			}
		}
		if scale {
			result, err = scaleColor(color, channelNumbers, alphaNumber)
			if err != nil {
				return nil, err
			}
		} else {
			result, err = adjustColor(ec, color, channelNumbers, alphaNumber)
			if err != nil {
				return nil, err
			}
		}
	}

	backConverted, err := result.ToSpace(originalColor.Space(), new(false))
	if err != nil {
		return nil, err
	}
	return backConverted, nil
}

// sniffLegacyColorSpace guesses which legacy space the keywords target for
// a legacy color without an explicit $space. Any red/green/blue keyword
// means rgb, saturation/lightness means hsl, whiteness/blackness means hwb,
// and a lone hue falls back to hsl. It returns nil when no keyword names a
// legacy channel, leaving the color in its own space.
func sniffLegacyColorSpace(keywords *orderedmap.LinkedMap[string, value.Value]) *value.ColorSpace {
	for key := range keywords.Keys() {
		switch key {
		case "red", "green", "blue":
			s := value.RgbColorSpace
			return &s
		case "saturation", "lightness":
			s := value.HslColorSpace
			return &s
		case "whiteness", "blackness":
			s := value.HwbColorSpace
			return &s
		}
	}
	if keywords.Has("hue") {
		s := value.HslColorSpace
		return &s
	}
	return nil
}

// changeColor replaces channels outright, porting Dart's _changeColor. Each
// channel resolves through channelForChange below; alpha keeps the current
// value when absent, goes missing for unquoted "none", reads unitless 0-1 or
// % 0-100 directly, and warns before accepting any other unit. The result is
// built without clamping so explicit out-of-range writes survive.
func changeColor(ec *evalcontext.EvaluationContext, color *value.SassColor, channelArgs []value.Value, alphaVal value.Value) (*value.SassColor, error) {
	// Resolves one channel for change(): no argument keeps the current
	// channel (with hsl/hwb saturation-like channels expressed as %), while
	// an explicit argument must be a number or unquoted "none".
	channelForChange := func(channelArg value.Value, channel int) (value.SassNumber, error) {
		if channelArg == nil {
			chOrNil := color.ChannelsOrNil()
			if chOrNil[channel] == nil {
				return nil, nil
			}
			v := *chOrNil[channel]
			if (color.Space() == value.HslColorSpace || color.Space() == value.HwbColorSpace) && channel > 0 {
				return value.NewSingleUnitNumber(v, "%"), nil
			}
			return value.NewUnitlessNumber(v), nil
		}
		if isNone(channelArg) {
			return nil, nil
		}
		if n, ok := channelArg.(value.SassNumber); ok {
			return n, nil
		}
		channelArgStr, err := channelArg.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("%s is not a number or unquoted \"none\".", channelArgStr),
			new(value.SpaceChannels(color.Space())[channel].Name),
		)
	}

	c0, err := channelForChange(channelArgs[0], 0)
	if err != nil {
		return nil, err
	}
	c1, err := channelForChange(channelArgs[1], 1)
	if err != nil {
		return nil, err
	}
	c2, err := channelForChange(channelArgs[2], 2)
	if err != nil {
		return nil, err
	}

	// Resolve alpha per the switch above: absent keeps, "none" drops, numbers
	// clamp through their range checks, and anything else errors.
	var alpha *float64
	if alphaVal == nil {
		v := color.Alpha()
		alpha = &v
	} else if s, ok := alphaVal.(*value.SassString); ok && !s.HasQuotes && strings.ToLower(s.Text) == "none" {
		// alpha stays nil (missing)
	} else if n, ok := alphaVal.(value.SassNumber); ok {
		if !n.HasUnits() {
			v, err := n.ValueInRange(0, 1, new("alpha"))
			if err != nil {
				return nil, err
			}
			alpha = &v
		} else if n.HasUnit("%") {
			v, err := n.ValueInRangeWithUnit(0, 100, "alpha", "%")
			if err != nil {
				return nil, err
			}
			v = v / 100
			alpha = &v
		} else {
			nStr, strErr := n.String()
			if strErr != nil {
				return nil, strErr
			}
			if err := ec.WarnDeprecation(fmt.Sprintf(
				"$alpha: Passing a unit other than %% (%s) is deprecated.\n"+
					"\n"+
					"To preserve current behavior: %s\n"+
					"\n"+
					"See https://sass-lang.com/d/function-units",
				nStr, n.UnitSuggestion("alpha", nil)), deprecation.FunctionUnits); err != nil {
				return nil, err
			}
			v, err := n.ValueInRange(0, 1, new("alpha"))
			if err != nil {
				return nil, err
			}
			alpha = &v
		}
	} else {
		alphaValStr, strErr := alphaVal.String()
		if strErr != nil {
			return nil, strErr
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a number or unquoted \"none\".", alphaValStr), new("alpha"))
	}

	result, err := colorFromChannels(ec, color.Space(), c0, c1, c2, alpha, false, false)
	if err != nil {
		return nil, err
	}
	return result.(*value.SassColor), nil
}

// scaleColor returns a copy of color with its channel values scaled toward
// their bounds, porting Dart's _scaleColor. Absent factors keep their
// channels; alpha scales through the same per-channel path.
func scaleColor(color *value.SassColor, channelArgs []value.SassNumber, alphaArg value.SassNumber) (*value.SassColor, error) {
	chs := value.SpaceChannels(color.Space())
	var scaledAlphaPtr *float64
	var err error
	if alphaArg != nil {
		scaledAlphaPtr, err = scaleChannel(color, value.AlphaChannel, color.AlphaOrNil(), alphaArg)
		if err != nil {
			return nil, err
		}
	} else {
		scaledAlphaPtr = color.AlphaOrNil()
	}
	s0, err := scaleChannel(color, chs[0], color.Channel0OrNil(), channelArgs[0])
	if err != nil {
		return nil, err
	}
	s1, err := scaleChannel(color, chs[1], color.Channel1OrNil(), channelArgs[1])
	if err != nil {
		return nil, err
	}
	s2, err := scaleChannel(color, chs[2], color.Channel2OrNil(), channelArgs[2])
	if err != nil {
		return nil, err
	}
	return newColorForSpaceInternal(color.Space(),
		s0, s1, s2, scaledAlphaPtr,
	)
}

// scaleChannel scales oldValue by factorArg within channel's range, porting
// Dart's _scaleChannel. A nil factor keeps the value; polar-angle channels
// are not scalable, and scaling a missing channel errors. The % factor in
// [-100, 100] pulls positive factors toward the max and negative factors
// toward the min, leaving values already at the bound in place.
func scaleChannel(color *value.SassColor, channel value.LinearChannel, oldValue *float64, factorArg value.SassNumber) (*float64, error) {
	if factorArg == nil {
		return oldValue, nil
	}
	if channel.IsPolarAngle {
		return nil, sasscommon.NewSassScriptException("Channel isn't scalable.", new(channel.Name))
	}

	if oldValue == nil {
		return nil, missingChannelError(color, channel.Name)
	}
	// The factor is a % in [-100, 100]; zero keeps the value.
	if err := factorArg.AssertUnit("%", &channel.Name); err != nil {
		return nil, err
	}
	// Normalize the % factor into [-1, 1] for the interpolation below.
	if _, err := factorArg.ValueInRange(-100, 100, &channel.Name); err != nil {
		return nil, err
	}
	factor := factorArg.NumValue() / 100
	if factor == 0 {
		return oldValue, nil
	}
	v := *oldValue
	if factor > 0 {
		if v >= channel.Max {
			return oldValue, nil
		}
		v = v + (channel.Max-v)*factor
	} else {
		if v <= channel.Min {
			return oldValue, nil
		}
		v = v + (v-channel.Min)*factor
	}
	return &v, nil
}

// adjustColor returns a copy of color with each channel shifted by its
// adjustment, porting Dart's _adjustColor. Alpha adjusts through the same
// channel path (for its unit warnings and % handling) and clamps to [0, 1]
// like CSS.
func adjustColor(ec *evalcontext.EvaluationContext, color *value.SassColor, channelArgs []value.SassNumber, alphaArg value.SassNumber) (*value.SassColor, error) {
	chs := value.SpaceChannels(color.Space())
	// Adjust alpha through adjustChannel to get proper deprecation warnings and % normalization
	var adjustedAlphaPtr *float64
	if alphaArg != nil {
		a, err := adjustChannel(ec, color, value.AlphaChannel, color.AlphaOrNil(), alphaArg)
		if err != nil {
			return nil, err
		}
		if a != nil {
			v := clampLikeCSS(*a, 0, 1)
			adjustedAlphaPtr = &v
		}
	} else {
		adjustedAlphaPtr = color.AlphaOrNil()
	}

	a0, err := adjustChannel(ec, color, chs[0], color.Channel0OrNil(), channelArgs[0])
	if err != nil {
		return nil, err
	}
	a1, err := adjustChannel(ec, color, chs[1], color.Channel1OrNil(), channelArgs[1])
	if err != nil {
		return nil, err
	}
	a2, err := adjustChannel(ec, color, chs[2], color.Channel2OrNil(), channelArgs[2])
	if err != nil {
		return nil, err
	}
	return newColorForSpaceInternal(color.Space(),
		a0, a1, a2, adjustedAlphaPtr,
	)
}

// adjustChannel shifts oldValue by adjustmentArg per channel, porting Dart's
// _adjustChannel with clamping disabled in the value conversion itself. A
// nil adjustment keeps the value; adjusting a missing channel errors. Hue in
// hsl/hwb coerces through degrees (with its deprecation path), hsl
// saturation/lightness accept unitless numbers with a % warning, and alpha
// with units warns before dropping to unitless. Results clamp to the
// channel's clamped ends, except a value already out of range moves with the
// adjustment rather than snapping back.
func adjustChannel(ec *evalcontext.EvaluationContext, color *value.SassColor, channel value.LinearChannel, oldValue *float64, adjustmentArg value.SassNumber) (*float64, error) {
	if adjustmentArg == nil {
		return oldValue, nil
	}
	if oldValue == nil {
		return nil, missingChannelError(color, channel.Name)
	}
	// Normalize the adjustment ahead of the value conversion, mirroring the
	// deprecation-period handling for hue/saturation/lightness/alpha.
	switch {
	case (color.Space() == value.HslColorSpace || color.Space() == value.HwbColorSpace) && channel.IsPolarAngle:
		deg, err := angleValue(ec, adjustmentArg, "hue")
		if err != nil {
			return nil, err
		}
		adjustmentArg = value.NewUnitlessNumber(deg)
	case color.Space() == value.HslColorSpace && (channel.Name == "saturation" || channel.Name == "lightness"):
		if err := checkPercentDeprecation(ec, adjustmentArg, channel.Name); err != nil {
			return nil, err
		}
		adjustmentArg = value.NewSingleUnitNumber(adjustmentArg.NumValue(), "%")
	case channel.Name == "alpha" && adjustmentArg.HasUnits():
		if err := ec.WarnDeprecation(fmt.Sprintf(
			"$alpha: Passing a number with unit %s is deprecated.\n"+
				"\n"+
				"To preserve current behavior: %s\n"+
				"\n"+
				"More info: https://sass-lang.com/d/function-units",
			adjustmentArg.UnitString(), adjustmentArg.UnitSuggestion("alpha", nil)), deprecation.FunctionUnits); err != nil {
			return nil, err
		}
		adjustmentArg = value.NewUnitlessNumber(adjustmentArg.NumValue())
	}
	// Add the converted adjustment, then clamp to the channel's clamped ends.
	adjPtr, err := channelFromValue(channel, adjustmentArg, false)
	if err != nil {
		return nil, err
	}
	result := *oldValue + *adjPtr
	// Clamp toward the bound, but let an already out-of-range value move with
	// the adjustment instead of snapping back inside.
	if channel.LowerClamped && result < channel.Min {
		if *oldValue < channel.Min {
			result = math.Max(*oldValue, result)
		} else {
			result = channel.Min
		}
	}
	if channel.UpperClamped && result > channel.Max {
		if *oldValue > channel.Max {
			result = math.Min(*oldValue, result)
		} else {
			result = channel.Max
		}
	}
	return &result, nil
}

// checkPercentDeprecation warns when n omits the % unit, porting Dart's
// _checkPercent. Callers normalize the number to % after the warning so
// behavior is preserved through the deprecation period.
func checkPercentDeprecation(ec *evalcontext.EvaluationContext, n value.SassNumber, name string) error {
	if n.HasUnit("%") {
		return nil
	}
	nStr, err := n.String()
	if err != nil {
		return err
	}
	return ec.WarnDeprecation(fmt.Sprintf(
		"$%s: Passing a number without unit %% (%s) is deprecated.\n"+
			"\n"+
			"To preserve current behavior: %s\n"+
			"\n"+
			"More info: https://sass-lang.com/d/function-units",
		name, nStr, n.UnitSuggestion(name, new("%"))), deprecation.FunctionUnits)
}
