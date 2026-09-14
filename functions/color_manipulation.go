// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/color.dart (manipulation section: grayscale, saturate/desaturate, adjust-hue, lighten/darken, alpha/opacity, invert, complement, mix, opacify/transparentize, ie-hex-str)

import (
	"fmt"
	"math"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// ---- grayscale() ----

// grayscaleCallable creates the global grayscale() callable ($color).
//
// A number or special-number argument echoes as the plain-CSS grayscale()
// filter; otherwise it warns for the global built-in and grayscales the
// color via grayscaleImpl. Matches Dart: global _function("grayscale", ...)
// (color.dart:130).
func grayscaleCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("grayscale", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if _, ok := args[0].(value.SassNumber); ok || args[0].IsSpecialNumber() {
			result, err := functionString("grayscale", []value.Value{args[0]})
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		if err := warnForGlobalBuiltIn(ec, "color", "grayscale"); err != nil {
			return nil, err
		}
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		return grayscaleImpl(color)
	})
}

// grayscaleModuleCallable creates the color.grayscale() module callable ($color).
//
// A number argument echoes as plain CSS plus a color-module-compat
// deprecation carrying the plain-CSS recommendation; colors delegate to
// grayscaleImpl with no global warning. Matches Dart: module
// _function("grayscale", ...) (color.dart:452).
func grayscaleModuleCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("grayscale", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if _, ok := args[0].(value.SassNumber); ok {
			result, err := functionString("grayscale", []value.Value{args[0]})
			if err != nil {
				return nil, err
			}
			argStr, err := args[0].String()
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				"Passing a number ("+argStr+") to color.grayscale() is deprecated.\n"+
					"\n"+
					"Recommendation: "+result.Text,
				deprecation.ColorModuleCompat,
			); err != nil {
				return nil, err
			}
			return result, nil
		}
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		return grayscaleImpl(color)
	})
}

// grayscaleImpl grayscales color with no plain-CSS handling.
//
// Legacy colors zero the HSL saturation while modern colors zero the OKLCH
// chroma, converting back into the color's original space in both cases.
// Matches Dart: _grayscale (color.dart:900).
func grayscaleImpl(color *value.SassColor) (value.Value, error) {
	if color.IsLegacy() {
		hsl, err := color.ToSpace(value.HslColorSpace, nil)
		if err != nil {
			return nil, err
		}
		zero := 0.0
		a := hsl.Alpha()
		c, err := value.NewColorForSpaceInternal(value.HslColorSpace,
			hsl.Channel0OrNil(), &zero, hsl.Channel2OrNil(), &a,
		)
		if err != nil {
			return nil, err
		}
		backConverted, err := c.ToSpace(color.Space(), new(false))
		if err != nil {
			return nil, err
		}
		return backConverted, nil
	}
	oklch, err := color.ToSpace(value.OklchColorSpace, nil)
	if err != nil {
		return nil, err
	}
	c, err := newColorForSpaceInternal(value.OklchColorSpace,
		oklch.Channel0OrNil(),
		func() *float64 { v := 0.0; return &v }(),
		oklch.Channel2OrNil(),
		oklch.AlphaOrNil(),
	)
	if err != nil {
		return nil, err
	}
	backConverted, err := c.ToSpace(color.Space(), nil)
	if err != nil {
		return nil, err
	}
	return backConverted, nil
}

// ---- saturate() ----

// saturateCallable creates the overloaded global saturate() callable.
//
// The $amount overload echoes numbers and special numbers as the plain-CSS
// saturate() filter (a non-number $amount re-emits saturate(<css>)); the
// $color, $amount overload requires a legacy color, clamps
// saturation + amount to [0, 100], and warns color-functions with a
// scale/adjust suggestion. Matches Dart:
// BuiltInCallable.overloadedFunction("saturate", ...) (color.dart:221).
func saturateCallable() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("saturate", "sass:color",
		OverloadDef{"$amount", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if _, ok := args[0].(value.SassNumber); ok || args[0].IsSpecialNumber() {
				result, err := functionString("saturate", args)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			n, err := value.AssertNumber(args[0], new("amount"))
			if err != nil {
				return nil, err
			}
			str, err := n.String()
			if err != nil {
				return nil, err
			}
			return &value.SassString{Text: fmt.Sprintf("saturate(%s)", str), HasQuotes: false}, nil
		}},
		OverloadDef{"$color, $amount", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if err := warnForGlobalBuiltIn(ec, "color", "adjust"); err != nil {
				return nil, err
			}
			color, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			amount, err := value.AssertNumber(args[1], new("amount"))
			if err != nil {
				return nil, err
			}
			if !color.IsLegacy() {
				return nil, sasscommon.NewSassScriptException(
					"saturate() is only supported for legacy colors. Please use "+
						"color.adjust() instead with an explicit $space argument.",
					nil)
			}
			if _, err := amount.ValueInRange(0, 100, new("amount")); err != nil {
				return nil, err
			}
			saturation, err := color.Saturation()
			if err != nil {
				return nil, err
			}
			v := clampLikeCSS(saturation+amount.NumValue(), 0, 100)
			result, err := color.ChangeHSL(nil, &v, nil, nil)
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				"saturate() is deprecated. "+
					suggestScaleAndAdjust(color, amount.NumValue(), "saturation")+"\n"+
					"\n"+
					"More info: https://sass-lang.com/d/color-functions",
				deprecation.ColorFunctions,
			); err != nil {
				return nil, err
			}
			return result, nil
		}},
	)
}

// ---- desaturate() ----

// desaturateCallable creates the global desaturate() callable ($color, $amount).
//
// Mirrors the two-argument saturate arm, subtracting the amount from
// saturation (clamped to [0, 100]) and suggesting with the negated
// adjustment. Matches Dart: _function("desaturate", ...) (color.dart:260).
func desaturateCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("desaturate", "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		amount, err := value.AssertNumber(args[1], new("amount"))
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				"desaturate() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		if _, err := amount.ValueInRange(0, 100, new("amount")); err != nil {
			return nil, err
		}
		saturation, err := color.Saturation()
		if err != nil {
			return nil, err
		}
		v := clampLikeCSS(saturation-amount.NumValue(), 0, 100)
		result, err := color.ChangeHSL(nil, &v, nil, nil)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			"desaturate() is deprecated. "+
				suggestScaleAndAdjust(color, -amount.NumValue(), "saturation")+"\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ---- adjust-hue() ----

// adjustHueCallable creates the global adjust-hue() callable ($color, $degrees).
//
// Parses $degrees as an angle, requires a legacy color, and shifts the HSL
// hue, warning color-functions with a color.adjust($color, $hue: ...) hint.
// Matches Dart: _function("adjust-hue", ...) (color.dart:141).
func adjustHueCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("adjust-hue", "$color, $degrees", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		degrees, err := angleValue(ec, args[1], "degrees")
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				"adjust-hue() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		suggestedValue := value.NewSingleUnitNumber(degrees, "deg")
		suggestedCSS, err := suggestedValue.ToCssString(true)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			"adjust-hue() is deprecated. Suggestion:\n"+
				"\n"+
				"color.adjust($color, $hue: "+suggestedCSS+")\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		newHue, err := color.Hue()
		if err != nil {
			return nil, err
		}
		newHue += degrees
		result, err := color.ChangeHSL(&newHue, nil, nil, nil)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ---- lighten() / darken() ----

// lightenCallable creates the global lighten() callable ($color, $amount).
//
// Requires a legacy color, clamps lightness + amount to [0, 100], and warns
// color-functions with a scale/adjust suggestion. Matches Dart:
// _function("lighten", ...) (color.dart:165).
func lightenCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("lighten", "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		amount, err := value.AssertNumber(args[1], new("amount"))
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				"lighten() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		if _, err := amount.ValueInRange(0, 100, new("amount")); err != nil {
			return nil, err
		}
		lightness, err := color.Lightness()
		if err != nil {
			return nil, err
		}
		v := clampLikeCSS(lightness+amount.NumValue(), 0, 100)
		result, err := color.ChangeHSL(nil, nil, &v, nil)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			"lighten() is deprecated. "+
				suggestScaleAndAdjust(color, amount.NumValue(), "lightness")+"\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// darkenCallable creates the global darken() callable ($color, $amount).
//
// Mirrors lightenCallable, subtracting the amount from lightness. Matches
// Dart: _function("darken", ...) (color.dart:193).
func darkenCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("darken", "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		amount, err := value.AssertNumber(args[1], new("amount"))
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				"darken() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		if _, err := amount.ValueInRange(0, 100, new("amount")); err != nil {
			return nil, err
		}
		lightness, err := color.Lightness()
		if err != nil {
			return nil, err
		}
		v := clampLikeCSS(lightness-amount.NumValue(), 0, 100)
		result, err := color.ChangeHSL(nil, nil, &v, nil)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			"darken() is deprecated. "+
				suggestScaleAndAdjust(color, -amount.NumValue(), "lightness")+"\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ---- alpha() / opacity() ----

// alphaCallable creates the overloaded global alpha() callable.
//
// The $color arm echoes proprietary Microsoft filters as plain CSS, rejects
// non-legacy colors (pointing at color.channel()), and otherwise returns the
// alpha after warning for the global built-in. The $args... arm only echoes
// when every argument is a Microsoft filter; an empty list throws Missing
// argument $color and longer lists throw Only 1 argument allowed. Matches
// Dart: global BuiltInCallable.overloadedFunction("alpha", ...)
// (color.dart:310).
func alphaCallable() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("alpha", "sass:color",
		OverloadDef{"$color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if isMicrosoftFilter(args[0]) {
				result, err := functionString("alpha", args)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			if col, ok := args[0].(*value.SassColor); ok && !col.IsLegacy() {
				return nil, sasscommon.NewSassScriptException(
					"alpha() is only supported for legacy colors. Please use color.channel() instead.", nil)
			}
			if err := warnForGlobalBuiltIn(ec, "color", "alpha"); err != nil {
				return nil, err
			}
			c, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			return value.NewUnitlessNumber(c.Alpha()), nil
		}},
		OverloadDef{"$args...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			restItems, err := args[0].AsList()
			if err != nil {
				return nil, err
			}
			if len(restItems) > 0 {
				allFilters := true
				for _, item := range restItems {
					if !isMicrosoftFilter(item) {
						allFilters = false
						break
					}
				}
				if allFilters {
					result, err := functionString("alpha", args)
					if err != nil {
						return nil, err
					}
					return result, nil
				}
			}
			if len(restItems) == 0 {
				return nil, sasscommon.NewSassScriptException("Missing argument $color.", nil)
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Only 1 argument allowed, but %d were passed.", len(restItems)), nil)
		}},
	)
}

// alphaModuleCallable creates the overloaded color.alpha() module callable.
//
// Same Microsoft-filter passthrough as alphaCallable, but each echo
// additionally warns color-module-compat with a Recommendation, and the
// color path returns the alpha with no global warning. Matches Dart: module
// BuiltInCallable.overloadedFunction("alpha", ...) (color.dart:491).
func alphaModuleCallable() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("alpha", "sass:color",
		OverloadDef{"$color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if isMicrosoftFilter(args[0]) {
				result, err := functionString("alpha", args)
				if err != nil {
					return nil, err
				}
				if err := ec.WarnDeprecation(
					"Using color.alpha() for a Microsoft filter is deprecated.\n"+
						"\n"+
						"Recommendation: "+result.Text,
					deprecation.ColorModuleCompat,
				); err != nil {
					return nil, err
				}
				return result, nil
			}
			if col, ok := args[0].(*value.SassColor); ok && !col.IsLegacy() {
				return nil, sasscommon.NewSassScriptException(
					"color.alpha() is only supported for legacy colors. Please use color.channel() instead.", nil)
			}
			c, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			return value.NewUnitlessNumber(c.Alpha()), nil
		}},
		OverloadDef{"$args...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			restItems, err := args[0].AsList()
			if err != nil {
				return nil, err
			}
			allFilters := true
			for _, item := range restItems {
				if !isMicrosoftFilter(item) {
					allFilters = false
					break
				}
			}
			if allFilters {
				result, err := functionString("alpha", args)
				if err != nil {
					return nil, err
				}
				if err := ec.WarnDeprecation(
					"Using color.alpha() for a Microsoft filter is deprecated.\n"+
						"\n"+
						"Recommendation: "+result.Text,
					deprecation.ColorModuleCompat,
				); err != nil {
					return nil, err
				}
				return result, nil
			}
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Only 1 argument allowed, but %d were passed.", len(args)), nil)
		}},
	)
}

// opacityCallable creates the global opacity() callable ($color).
//
// Numbers and special numbers echo as the plain-CSS opacity() filter;
// otherwise it warns for the global built-in and returns the color's alpha.
// Matches Dart: global _function("opacity", ...) (color.dart:351).
func opacityCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("opacity", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if _, ok := args[0].(value.SassNumber); ok || args[0].IsSpecialNumber() {
			result, err := functionString("opacity", args)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		if err := warnForGlobalBuiltIn(ec, "color", "opacity"); err != nil {
			return nil, err
		}
		c, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		return value.NewUnitlessNumber(c.Alpha()), nil
	})
}

// opacityModuleCallable creates the color.opacity() module callable ($color).
//
// A number argument echoes as plain CSS plus a color-module-compat
// deprecation; otherwise returns the color's alpha. Matches Dart: module
// _function("opacity", ...) (color.dart:541).
func opacityModuleCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("opacity", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if _, ok := args[0].(value.SassNumber); ok {
			result, err := functionString("opacity", args)
			if err != nil {
				return nil, err
			}
			argStr, err := args[0].String()
			if err != nil {
				return nil, err
			}
			resStr, err := result.String()
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				"Passing a number ("+argStr+" to color.opacity() is deprecated.\n"+
					"\n"+
					"Recommendation: "+resStr,
				deprecation.ColorModuleCompat,
			); err != nil {
				return nil, err
			}
			return result, nil
		}
		c, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		return value.NewUnitlessNumber(c.Alpha()), nil
	})
}

// ---- invert() ----

// invertCallable creates the global invert() callable ($color, $weight: 100%, $space: null).
//
// Warns for the global built-in on non-number first arguments, then
// delegates to invertImpl in global mode (where special numbers keep the
// plain-CSS path). Matches Dart: global _function("invert", ...)
// (color.dart:68).
func invertCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("invert", "$color, $weight: 100%, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if _, ok := args[0].(value.SassNumber); !ok && !args[0].IsSpecialNumber() {
			if err := warnForGlobalBuiltIn(ec, "color", "invert"); err != nil {
				return nil, err
			}
		}
		result, err := invertImpl(ec, args, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// invertModuleCallable creates the color.invert() module callable ($color, $weight: 100%, $space: null).
//
// Delegates to invertImpl in module mode; a plain-CSS string result
// additionally warns color-module-compat that a number was passed to
// color.invert(). Matches Dart: module _function("invert", ...)
// (color.dart:423).
func invertModuleCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("invert", "$color, $weight: 100%, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		result, err := invertImpl(ec, args, false)
		if err != nil {
			return nil, err
		}
		if strResult, ok := result.(*value.SassString); ok {
			argStr, err := args[0].String()
			if err != nil {
				return nil, err
			}
			if err := ec.WarnDeprecation(
				"Passing a number ("+argStr+") to color.invert() is deprecated.\n"+
					"\n"+
					"Recommendation: "+strResult.Text,
				deprecation.ColorModuleCompat,
			); err != nil {
				return nil, err
			}
		}
		return result, nil
	})
}

// invertImpl implements invert() for both the global and module callables.
//
// When global is set, special-number first arguments take the plain-CSS
// path; otherwise number first arguments echo only when $weight is exactly
// 100%. With $space null the color must be legacy and the RGB inversion is
// mixed in by weight; with an explicit space the channels invert per space
// (HWB inverts whiteness while swapping the other two slots, HSL/LCH/OKLCH
// invert hue plus the last channel, every other space inverts all three)
// and partial weights interpolate toward the inversion. Matches Dart:
// _invert, including the global flag doc (color.dart:803).
func invertImpl(ec *evalcontext.EvaluationContext, args []value.Value, global bool) (value.Value, error) {
	// Dart: var weightNumber = arguments[1].assertNumber("weight");
	weightNumber, err := value.AssertNumber(args[1], new("weight"))
	if err != nil {
		return nil, err
	}

	// Dart: if (arguments[0] is SassNumber || (global && arguments[0].isSpecialNumber))
	if _, ok := args[0].(value.SassNumber); ok || (global && args[0].IsSpecialNumber()) {
		// Dart: if (weightNumber.value != 100 || !weightNumber.hasUnit("%"))
		if !util.FuzzyEquals(weightNumber.NumValue(), 100) || !weightNumber.HasUnit("%") {
			return nil, sasscommon.NewSassScriptException("Only one argument may be passed to the plain-CSS invert() function.", nil)
		}
		// Dart: return _functionString("invert", arguments.take(1));
		result, err := functionString("invert", args[:1])
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// Dart: var color = arguments[0].assertColor("color");
	color, err := value.AssertColor(args[0], new("color"))
	if err != nil {
		return nil, err
	}

	// Dart: if (arguments[2] == sassNull)
	if args[2] == value.Null {
		// Dart: if (!color.isLegacy) { throw ... }
		if !color.IsLegacy() {
			colStr, err := color.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(
				fmt.Sprintf("To use color.invert() with non-legacy color %s, you must provide a $space.", colStr),
				new("color"),
			)
		}

		if err := checkPercent(ec, weightNumber, "weight"); err != nil {
			return nil, err
		}

		// Dart: var rgb = color.toSpace(ColorSpace.rgb);
		rgb, err := color.ToSpace(value.RgbColorSpace, nil)
		if err != nil {
			return nil, err
		}
		// Dart: var [channel0, channel1, channel2] = ColorSpace.rgb.channels;
		chs := value.SpaceChannels(value.RgbColorSpace)
		// Dart: SassColor.rgb(_invertChannel(...), ...)
		c0, err := invertChannel(rgb, chs[0], rgb.Channel0OrNil())
		if err != nil {
			return nil, err
		}
		c1, err := invertChannel(rgb, chs[1], rgb.Channel1OrNil())
		if err != nil {
			return nil, err
		}
		c2, err := invertChannel(rgb, chs[2], rgb.Channel2OrNil())
		if err != nil {
			return nil, err
		}
		inv, err := value.NewColorRGBInternal(&c0, &c1, &c2, color.AlphaOrNil(), nil)
		if err != nil {
			return nil, err
		}
		// Dart: return _mixLegacy(...).toSpace(color.space);
		mixed, err := mixLegacy(inv, color, weightNumber)
		if err != nil {
			return nil, err
		}
		if mixedColor, ok := mixed.(*value.SassColor); ok {
			result, err := mixedColor.ToSpace(color.Space(), nil)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		return mixed, nil
	}

	// Dart: var space = ColorSpace.fromName(...)
	spaceStr, err := value.AssertString(args[2], new("space"))
	if err != nil {
		return nil, err
	}
	if err := spaceStr.AssertUnquoted(); err != nil {
		return nil, sasscommon.NewSassScriptException(err.Error(), new("space"))
	}
	space, err := value.ColorSpaceFromName(spaceStr.Text, new("space"))
	if err != nil {
		return nil, err
	}

	// Dart: var weight = weightNumber.valueInRangeWithUnit(0, 100, 'weight', '%') / 100;
	if err := weightNumber.AssertUnit("%", new("weight")); err != nil {
		return nil, err
	}
	if _, err := weightNumber.ValueInRange(0, 100, new("weight")); err != nil {
		return nil, err
	}
	weight := weightNumber.NumValue() / 100

	// Dart: if (fuzzyEquals(weight, 0)) return color;
	if util.FuzzyEquals(weight, 0) {
		return color, nil
	}

	// Dart: var inSpace = color.toSpace(space);
	inSpace, err := color.ToSpace(space, nil)
	if err != nil {
		return nil, err
	}

	// Dart: var inverted = switch (space) { ... };
	var inverted *value.SassColor

	switch space {
	case value.HwbColorSpace:
		// Dart: SassColor.hwb(_invertChannel(inSpace, space.channels[0], inSpace.channel0OrNull),
		//                    inSpace.channel2OrNull, inSpace.channel1OrNull, inSpace.alpha)
		chs := value.SpaceChannels(space)
		var hwbC0 float64
		hwbC0, err = invertChannel(inSpace, chs[0], inSpace.Channel0OrNil())
		if err != nil {
			return nil, err
		}
		c0 := &hwbC0
		inverted, err = newColorForSpaceInternal(space,
			c0,
			inSpace.Channel2OrNil(),
			inSpace.Channel1OrNil(),
			inSpace.AlphaOrNil(),
		)
		if err != nil {
			return nil, err
		}

	case value.HslColorSpace, value.LchColorSpace, value.OklchColorSpace:
		// Dart: SassColor.forSpaceInternal(space,
		//   _invertChannel(inSpace, space.channels[0], inSpace.channel0OrNull),
		//   inSpace.channel1OrNull,
		//   _invertChannel(inSpace, space.channels[2], inSpace.channel2OrNull),
		//   inSpace.alpha)
		chs := value.SpaceChannels(space)
		c0, err := invertChannel(inSpace, chs[0], inSpace.Channel0OrNil())
		if err != nil {
			return nil, err
		}
		c2, err := invertChannel(inSpace, chs[2], inSpace.Channel2OrNil())
		if err != nil {
			return nil, err
		}
		a := inSpace.Alpha()
		inverted, err = value.NewColorForSpaceInternal(space,
			&c0,
			inSpace.Channel1OrNil(),
			&c2,
			&a,
		)
		if err != nil {
			return nil, err
		}

	default:
		// Dart: SassColor.forSpaceInternal(space,
		//   _invertChannel(inSpace, channel0, inSpace.channel0OrNull),
		//   _invertChannel(inSpace, channel1, inSpace.channel1OrNull),
		//   _invertChannel(inSpace, channel2, inSpace.channel2OrNull),
		//   inSpace.alpha)
		chs := value.SpaceChannels(space)
		c0, err := invertChannel(inSpace, chs[0], inSpace.Channel0OrNil())
		if err != nil {
			return nil, err
		}
		c1, err := invertChannel(inSpace, chs[1], inSpace.Channel1OrNil())
		if err != nil {
			return nil, err
		}
		c2, err := invertChannel(inSpace, chs[2], inSpace.Channel2OrNil())
		if err != nil {
			return nil, err
		}
		a := inSpace.Alpha()
		inverted, err = value.NewColorForSpaceInternal(space,
			&c0, &c1, &c2, &a,
		)
		if err != nil {
			return nil, err
		}
	}

	// Dart: return fuzzyEquals(weight, 1)
	//   ? inverted.toSpace(color.space, legacyMissing: false)
	//   : color.interpolate(inverted, InterpolationMethod(space), weight: 1 - weight, legacyMissing: false);
	if util.FuzzyEquals(weight, 1) {
		backConverted, err := inverted.ToSpace(color.Space(), new(false))
		if err != nil {
			return nil, err
		}
		return backConverted, nil
	}
	method, methodErr := value.NewInterpolationMethod(space, nil)
	if methodErr != nil {
		return nil, methodErr
	}
	w := 1 - weight
	return color.Interpolate(inverted, method, false, &w)
}

// rgbFlagOrNil returns the alpha value if present, defaulting a missing
// (nil) alpha to 1.0 opaque. This mirrors how Dart's _invert legacy path
// passes color.alphaOrNull through (color.dart:833), where a missing alpha
// reads back as fully opaque.
func rgbFlagOrNil(v *float64) float64 {
	if v == nil {
		return 1.0
	}
	return *v
}

// invertChannel returns the inverse of val in linear channel channel of color.
//
// A missing channel reports missingChannelError; hues rotate by 180 degrees,
// channels with a negative minimum negate, and all other channels subtract
// from max. Matches Dart: _invertChannel (color.dart:888).
func invertChannel(color *value.SassColor, channel value.LinearChannel, val *float64) (float64, error) {
	if val == nil {
		return 0, missingChannelError(color, channel.Name)
	}
	// Dart: LinearChannel(min: < 0) => -value
	if channel.Min < 0 {
		return -(*val), nil
	}
	// Dart: LinearChannel(min: 0, :var max) => max - value
	// Dart: ColorChannel(isPolarAngle: true) => (value + 180) % 360
	// In Go, HueChannel is LinearChannel with Min=0, Max=0, IsPolarAngle=true.
	// Check polar angle first since HueChannel matches both conditions.
	if channel.IsPolarAngle {
		return math.Mod((*val)+180, 360), nil
	}
	return channel.Max - (*val), nil
}

// missingChannelError reports that a missing channel cannot be modified.
//
// The CSS working group is still deciding the behavior, so Sass throws
// rather than guessing, naming the offending channel and the color.
// Matches Dart: _missingChannelError (color.dart:1961).
func missingChannelError(color *value.SassColor, channel string) error {
	css, err := color.ToCssString(true)
	if err != nil {
		colStr, err := color.String()
		if err != nil {
			return err
		}
		return sasscommon.NewSassScriptException(
			fmt.Sprintf("Because the CSS working group is still deciding on the best behavior, Sass doesn't currently support modifying missing channels (color: %s).", colStr),
			new(channel),
		)
	}
	return sasscommon.NewSassScriptException(
		fmt.Sprintf("Because the CSS working group is still deciding on the best behavior, Sass doesn't currently support modifying missing channels (color: %s).", css),
		new(channel),
	)
}

// ---- complement() ----

// complementCallable creates the complement() callable ($color, $space: null).
//
// Legacy colors default to HSL while an explicit $space must name a polar
// space; the hue channel rotates 180 degrees (channel 0 for legacy spaces,
// channel 2 for modern ones) before converting back with legacyMissing
// disabled. Matches Dart: _complement (color.dart:753).
func complementCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("complement", "$color, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}

		// Matches Dart: color.isLegacy && arguments[1] == sassNull ? ColorSpace.hsl : ...
		var space value.ColorSpace
		if color.IsLegacy() && args[1] == value.Null {
			space = value.HslColorSpace
		} else {
			spaceStr, err := value.AssertString(args[1], new("space"))
			if err != nil {
				return nil, err
			}
			if err := spaceStr.AssertUnquoted(); err != nil {
				return nil, sasscommon.NewSassScriptException(err.Error(), new("space"))
			}
			space, err = value.ColorSpaceFromName(spaceStr.Text, new("space"))
			if err != nil {
				return nil, err
			}
		}

		if !space.IsPolar() {
			return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Color space %s doesn't have a hue channel.", space.Name()), new("space"))
		}

		// Matches Dart: legacyMissing: arguments[1] != sassNull
		legacyMissing := args[1] != value.Null
		inSpace, err := color.ToSpace(space, &legacyMissing)
		if err != nil {
			return nil, err
		}

		var result *value.SassColor
		if space.IsLegacy() {
			// Legacy: adjust hue (channel 0) by 180
			c0, err := adjustChannel(ec, inSpace, space.Channels()[0], inSpace.Channel0OrNil(), value.NewUnitlessNumber(180))
			if err != nil {
				return nil, err
			}
			result, err = value.NewColorForSpaceInternal(space,
				c0,
				inSpace.Channel1OrNil(),
				inSpace.Channel2OrNil(),
				inSpace.AlphaOrNil(),
			)
			if err != nil {
				return nil, err
			}
		} else {
			// Non-legacy: adjust hue (channel 2) by 180
			c2, err := adjustChannel(ec, inSpace, space.Channels()[2], inSpace.Channel2OrNil(), value.NewUnitlessNumber(180))
			if err != nil {
				return nil, err
			}
			result, err = value.NewColorForSpaceInternal(space,
				inSpace.Channel0OrNil(),
				inSpace.Channel1OrNil(),
				c2,
				inSpace.AlphaOrNil(),
			)
			if err != nil {
				return nil, err
			}
		}
		backConverted, err := result.ToSpace(color.Space(), new(false))
		if err != nil {
			return nil, err
		}
		return backConverted, nil
	})
}

// adjustPolarChannel rotates hue val by adjustment degrees, wrapping into
// [0, 360). A missing channel reports missingChannelError. This factors out
// the +180 rotation Dart performs inline via _adjustChannel in _complement
// (color.dart:775).
func adjustPolarChannel(color *value.SassColor, channel value.LinearChannel, val *float64, adjustment float64) (float64, error) {
	if val == nil {
		return 0, missingChannelError(color, channel.Name)
	}
	v := math.Mod((*val)+adjustment, 360)
	if v < 0 {
		v += 360
	}
	return v, nil
}

// ---- mix() ----

// mixCallable creates the mix() callable ($color1, $color2, $weight: 50%, $method: null).
//
// With $method the colors interpolate in that space (weight coerced to a %
// in [0, 100]); without it the weight must be a percent and both colors must
// be legacy, falling back to the Sass legacy mixing algorithm. Matches Dart:
// _mix (color.dart:715).
func mixCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("mix", "$color1, $color2, $weight: 50%, $method: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color1, err := value.AssertColor(args[0], new("color1"))
		if err != nil {
			return nil, err
		}
		color2, err := value.AssertColor(args[1], new("color2"))
		if err != nil {
			return nil, err
		}
		weight, err := value.AssertNumber(args[2], new("weight"))
		if err != nil {
			return nil, err
		}

		if args[3] != value.Null {
			method, err := value.InterpolationMethodFromValue(args[3], new("method"))
			if err != nil {
				return nil, err
			}
			if err := weight.AssertUnit("%", new("weight")); err != nil {
				return nil, err
			}
			if _, err := weight.ValueInRange(0, 100, new("weight")); err != nil {
				return nil, err
			}
			w := weight.NumValue() / 100
			result, err := color1.Interpolate(color2, method, false, &w)
			if err != nil {
				return nil, err
			}
			return result, nil
		}

		if err := checkPercent(ec, weight, "weight"); err != nil {
			return nil, err
		}
		if !color1.IsLegacy() {
			c1Str, err := color1.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(
				fmt.Sprintf("To use color.mix() with non-legacy color %s, you must provide a $method.", c1Str),
				new("color1"),
			)
		} else if !color2.IsLegacy() {
			c2Str, err := color2.String()
			if err != nil {
				return nil, err
			}
			return nil, sasscommon.NewSassScriptException(
				fmt.Sprintf("To use color.mix() with non-legacy color %s, you must provide a $method.", c2Str),
				new("color2"),
			)
		}

		return mixLegacy(color1, color2, weight)
	})
}

// mixLegacy mixes two legacy colors by weight with Sass's alpha-aware algorithm.
//
// Both the user weight and the alpha distance normalize to [-1, 1] (1 keeps
// color1, -1 keeps color2) and combine as (w + a)/(1 + w*a), with the w*a == -1
// degenerate case keeping the raw weight; the combined weight then
// renormalizes to [0, 1] for the RGB average while alpha blends by the raw
// weight. Matches Dart: _mixLegacy, including the normalization rationale
// (color.dart:1444).
func mixLegacy(color1, color2 *value.SassColor, weight value.SassNumber) (value.Value, error) {
	if _, err := weight.ValueInRange(0, 100, new("weight")); err != nil {
		return nil, err
	}
	weightScale := weight.NumValue() / 100
	normalizedWeight := weightScale*2 - 1
	alphaDistance := color1.Alpha() - color2.Alpha()

	var combinedWeight1 float64
	// Degenerate case: w*a == -1 leaves the combination undefined, so the
	// raw weight wins (color.dart:1475).
	if normalizedWeight*alphaDistance == -1 {
		combinedWeight1 = normalizedWeight
	} else {
		combinedWeight1 = (normalizedWeight + alphaDistance) / (1 + normalizedWeight*alphaDistance)
	}

	w1 := (combinedWeight1 + 1) / 2
	w2 := 1 - w1

	rgb1, err := color1.ToSpace(value.RgbColorSpace, nil)
	if err != nil {
		return nil, err
	}
	rgb2, err := color2.ToSpace(value.RgbColorSpace, nil)
	if err != nil {
		return nil, err
	}
	r1, g1, b1 := rgb1.Channel0(), rgb1.Channel1(), rgb1.Channel2()
	r2, g2, b2 := rgb2.Channel0(), rgb2.Channel1(), rgb2.Channel2()

	return value.NewColorRGB(
		r1*w1+r2*w2,
		g1*w1+g2*w2,
		b1*w1+b2*w2,
		clampLikeCSS(rgb1.Alpha()*weightScale+rgb2.Alpha()*(1-weightScale), 0, 1),
	)
}

// ---- opacify() / fade-in() / transparentize() / fade-out() ----

// opacifyCallable creates the opacify()/fade-in() callable ($color, $amount).
//
// The name selects the entry point; both share this body. Requires a legacy
// color, adds the unitless [0, 1] amount to the alpha (clamped), and warns
// color-functions with a scale/adjust suggestion. Matches Dart: _opacify
// (color.dart:1491).
func opacifyCallable(name string) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		amount, err := value.AssertNumber(args[1], new("amount"))
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				name+"() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		if _, err := amount.ValueInRangeWithUnit(0, 1, "amount", ""); err != nil {
			return nil, err
		}
		result, err := color.ChangeAlpha(
			clampLikeCSS(color.Alpha()+amount.NumValue(), 0, 1),
		)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			name+"() is deprecated. "+
				suggestScaleAndAdjust(color, amount.NumValue(), "alpha")+"\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// transparentizeCallable creates the transparentize()/fade-out() callable ($color, $amount).
//
// Mirrors opacifyCallable, subtracting the amount and suggesting with the
// negated adjustment. Matches Dart: _transparentize (color.dart:1520).
func transparentizeCallable(name string) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		amount, err := value.AssertNumber(args[1], new("amount"))
		if err != nil {
			return nil, err
		}
		if !color.IsLegacy() {
			return nil, sasscommon.NewSassScriptException(
				name+"() is only supported for legacy colors. Please use "+
					"color.adjust() instead with an explicit $space argument.",
				nil)
		}
		if _, err := amount.ValueInRangeWithUnit(0, 1, "amount", ""); err != nil {
			return nil, err
		}
		result, err := color.ChangeAlpha(
			clampLikeCSS(color.Alpha()-amount.NumValue(), 0, 1),
		)
		if err != nil {
			return nil, err
		}
		if err := ec.WarnDeprecation(
			name+"() is deprecated. "+
				suggestScaleAndAdjust(color, -amount.NumValue(), "alpha")+"\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			deprecation.ColorFunctions,
		); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// suggestScaleAndAdjust returns suggested translations for deprecated color
// modification functions in terms of both color.scale() and color.adjust().
//
// The scale factor is +-100% when the result would overshoot the channel
// bounds, otherwise the proportional distance to the bound; alpha
// suggestions stay unitless while other channels carry %. Matches Dart:
// _suggestScaleAndAdjust (color.dart:1913).
func suggestScaleAndAdjust(color *value.SassColor, adjustment float64, channelName string) string {
	var chMin, chMax float64
	var ch *value.LinearChannel
	if channelName == "alpha" {
		// Alpha channel: min=0, max=1
		chMin = 0
		chMax = 1
	} else {
		chs := value.SpaceChannels(value.HslColorSpace)
		for i := range chs {
			if chs[i].Name == channelName {
				ch = &chs[i]
				break
			}
		}
		if ch == nil {
			return ""
		}
		chMin = ch.Min
		chMax = ch.Max
	}

	var oldValue float64
	if channelName == "alpha" {
		oldValue = color.Alpha()
	} else {
		hsl, convErr := color.ToSpace(value.HslColorSpace, nil)
		if convErr != nil {
			return ""
		}
		v, err := hsl.ChannelByName(channelName)
		if err != nil {
			return ""
		}
		oldValue = v
	}
	newValue := oldValue + adjustment

	suggestion := "Suggestion"
	if adjustment != 0 {
		var factor float64
		if newValue > chMax {
			factor = 1
		} else if newValue < chMin {
			factor = -1
		} else if adjustment > 0 {
			factor = adjustment / (chMax - oldValue)
		} else {
			factor = (newValue - oldValue) / (oldValue - chMin)
		}
		factorNum := value.NewSingleUnitNumber(factor*100, "%")
		factorCSS, _ := factorNum.ToCssString(true)
		suggestion += "s:\n\ncolor.scale($color, $" + channelName + ": " + factorCSS + ")\n"
	} else {
		suggestion += ":\n\n"
	}

	difference := value.NewUnitlessNumber(adjustment)
	if channelName != "alpha" {
		difference = value.NewSingleUnitNumber(adjustment, "%")
	}
	diffCSS, _ := difference.ToCssString(true)
	return suggestion + "color.adjust($color, $" + channelName + ": " + diffCSS + ")"
}

// ---- ie-hex-str() ----

// ieHexStrCallable creates the ie-hex-str() callable ($color).
//
// Converts to RGB, gamut-maps with local-minde, then emits #AARRGGBB with
// each component fuzzy-rounded (not plain-rounded) and uppercased. Matches
// Dart: _ieHexStr (color.dart:942).
func ieHexStrCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("ie-hex-str", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		color, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		// Convert to RGB and gamut-map, matching Dart's
		// .toSpace(.rgb).toGamut(.localMinde) chain (color.dart:942).
		rgb, err := color.ToSpace(value.RgbColorSpace, nil)
		if err != nil {
			return nil, err
		}
		gamutColor, err := rgb.ToGamut(value.GamutMapLocalMinde)
		if err != nil {
			return nil, err
		}
		gamutAlpha := gamutColor.Alpha() * 255
		gamutR := gamutColor.Channel0()
		gamutG := gamutColor.Channel1()
		gamutB := gamutColor.Channel2()
		return &value.SassString{
			Text:      fmt.Sprintf("#%02X%02X%02X%02X", roundInt(gamutAlpha), roundInt(gamutR), roundInt(gamutG), roundInt(gamutB)),
			HasQuotes: false,
		}, nil
	})
}
