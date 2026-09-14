// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/color.dart (constructor section: rgb/rgba, hsl/hsla, hwb, lab/lch/oklab/oklch, color(), channel parsing)

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// ---- rgb() / rgba() ----

// rgbCallable creates the overloaded rgb() callable.
//
// Signatures: $red, $green, $blue, $alpha / $red, $green, $blue /
// $color, $alpha / $channels. Each overload delegates to rgbImpl,
// rgbTwoArg, or parseChannels in the RGB space. Matches Dart:
// BuiltInCallable.overloadedFunction("rgb", ...) (color.dart:52).
func rgbCallable() *BuiltInCallable {
	rgbSpace := value.RgbColorSpace

	return MustNewBuiltInCallableOverloadedFunction("rgb", "sass:color",
		OverloadDef{"$red, $green, $blue, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbImpl(ec, "rgb", args)
		}},
		OverloadDef{"$red, $green, $blue", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbImpl(ec, "rgb", args)
		}},
		OverloadDef{"$color, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbTwoArg("rgb", args)
		}},
		OverloadDef{"$channels", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return parseChannels(ec, "rgb", "channels", args[0], &rgbSpace)
		}},
	)
}

// rgbaCallable creates the overloaded rgba() callable.
//
// Same four overloads as rgbCallable, with "rgba" as the plain-CSS fallback
// name. Matches Dart: BuiltInCallable.overloadedFunction("rgba", ...)
// (color.dart:60).
func rgbaCallable() *BuiltInCallable {
	rgbSpace := value.RgbColorSpace

	return MustNewBuiltInCallableOverloadedFunction("rgba", "sass:color",
		OverloadDef{"$red, $green, $blue, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbImpl(ec, "rgba", args)
		}},
		OverloadDef{"$red, $green, $blue", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbImpl(ec, "rgba", args)
		}},
		OverloadDef{"$color, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return rgbTwoArg("rgba", args)
		}},
		OverloadDef{"$channels", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return parseChannels(ec, "rgba", "channels", args[0], &rgbSpace)
		}},
	)
}

// rgbImpl implements the three- and four-argument rgb()/rgba() functions.
//
// Special numbers pass through as plain CSS; otherwise $red/$green/$blue
// assert as numbers and the optional $alpha clamps via %-or-unitless
// normalization, building an rgb-function-tagged color. Matches Dart: _rgb
// (color.dart:1295), including fromRgbFunction: true.
func rgbImpl(ec *evalcontext.EvaluationContext, name string, args []value.Value) (value.Value, error) {
	var alpha value.Value
	if len(args) > 3 {
		alpha = args[3]
	}
	if args[0].IsSpecialNumber() || args[1].IsSpecialNumber() || args[2].IsSpecialNumber() ||
		(alpha != nil && alpha.IsSpecialNumber()) {
		result, err := functionString(name, args)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	r, err := value.AssertNumber(args[0], new("red"))
	if err != nil {
		return nil, err
	}
	g, err := value.AssertNumber(args[1], new("green"))
	if err != nil {
		return nil, err
	}
	b, err := value.AssertNumber(args[2], new("blue"))
	if err != nil {
		return nil, err
	}

	var a float64 = 1.0
	if alpha != nil {
		aNum, err := value.AssertNumber(alpha, new("alpha"))
		if err != nil {
			return nil, err
		}
		p, err := percentageOrUnitless(aNum, 1, "alpha")
		if err != nil {
			return nil, err
		}
		a = clampLikeCSS(p, 0, 1)
	}

	return colorFromChannels(ec, value.RgbColorSpace, r, g, b, &a, true, true)
}

// rgbTwoArg implements the two-argument rgb()/rgba() functions.
//
// var()-ish inputs fall back to plain CSS because --foo may expand to
// "123, 456, 789" after substitution (functions parse after variables do).
// Non-legacy colors throw with a color.change() suggestion; a special-number
// alpha re-emits the numeric channels plus the raw alpha. Matches Dart:
// _rgbTwoArg (color.dart:1322).
func rgbTwoArg(name string, args []value.Value) (value.Value, error) {
	first := args[0]
	second := args[1]

	// rgba(var(--foo), 0.5) is valid CSS because --foo might expand to
	// `123, 456, 789`: functions parse after variable substitution
	// (color.dart:1323).
	if isSpecialVariable(first) || (!isColor(first) && isSpecialVariable(second)) {
		result, err := functionString(name, args)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	color, err := value.AssertColor(first, new("color"))
	if err != nil {
		return nil, err
	}
	if !color.IsLegacy() {
		colStr, err := color.String()
		if err != nil {
			return nil, err
		}
		arg1Str, err := args[1].String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("Expected %s to be in the legacy RGB, HSL, or HWB color space.\n\nRecommendation: color.change(%s, $alpha: %s)",
				colStr, colStr, arg1Str), new(name))
	}

	rgb, err := color.ToSpace(value.RgbColorSpace, nil)
	if err != nil {
		return nil, err
	}
	r := rgb.Channel0()
	g := rgb.Channel1()
	b := rgb.Channel2()
	if second.IsSpecialNumber() {
		result, err := functionString(name, []value.Value{
			value.NewUnitlessNumber(r),
			value.NewUnitlessNumber(g),
			value.NewUnitlessNumber(b),
			second,
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	s, err := value.AssertNumber(second, new("alpha"))
	if err != nil {
		return nil, err
	}
	p, err := percentageOrUnitless(s, 1, "alpha")
	if err != nil {
		return nil, err
	}
	return value.NewColorRGB(r, g, b, clampLikeCSS(p, 0, 1))
}

// isColor reports whether v is a Sass color. Go-only helper for the
// rgbTwoArg var() check, where Dart tests first is! SassColor inline
// (color.dart:1327).
func isColor(v value.Value) bool {
	_, ok := v.(*value.SassColor)
	return ok
}

// ---- hsl() / hsla() ----

// hslCallable creates the overloaded hsl() callable.
//
// Signatures: $hue, $saturation, $lightness, $alpha /
// $hue, $saturation, $lightness / $hue, $saturation / $channels. The two-arg
// overload only survives for special variables (hsl(123, var(--foo)) may
// gain channels post-substitution) and otherwise throws Missing argument
// $lightness. Matches Dart: BuiltInCallable.overloadedFunction("hsl", ...)
// (color.dart:98).
func hslCallable() *BuiltInCallable {
	hslSpace := value.HslColorSpace

	return MustNewBuiltInCallableOverloadedFunction("hsl", "sass:color",
		OverloadDef{"$hue, $saturation, $lightness, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return hslImpl(ec, "hsl", args)
		}},
		OverloadDef{"$hue, $saturation, $lightness", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return hslImpl(ec, "hsl", args)
		}},
		OverloadDef{"$hue, $saturation", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			// hsl(123, var(--foo)) is valid CSS because --foo might expand
			// to `10%, 20%`: functions parse after variable substitution
			// (color.dart:103).
			if isSpecialVariable(args[0]) || isSpecialVariable(args[1]) {
				return functionString("hsl", args)
			}
			return nil, sasscommon.NewSassScriptException("Missing argument $lightness.", nil)
		}},
		OverloadDef{"$channels", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return parseChannels(ec, "hsl", "channels", args[0], &hslSpace)
		}},
	)
}

// hslaCallable creates the overloaded hsla() callable.
//
// Same four overloads as hslCallable, with "hsla" as the plain-CSS fallback
// name. Matches Dart: BuiltInCallable.overloadedFunction("hsla", ...)
// (color.dart:115).
func hslaCallable() *BuiltInCallable {
	hslSpace := value.HslColorSpace

	return MustNewBuiltInCallableOverloadedFunction("hsla", "sass:color",
		OverloadDef{"$hue, $saturation, $lightness, $alpha", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return hslImpl(ec, "hsla", args)
		}},
		OverloadDef{"$hue, $saturation, $lightness", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return hslImpl(ec, "hsla", args)
		}},
		OverloadDef{"$hue, $saturation", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			// Same var() rationale as the hsl two-arg overload: --foo might
			// expand to `10%, 20%` post-substitution (color.dart:103).
			if isSpecialVariable(args[0]) || isSpecialVariable(args[1]) {
				return functionString("hsla", args)
			}
			return nil, sasscommon.NewSassScriptException("Missing argument $lightness.", nil)
		}},
		OverloadDef{"$channels", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return parseChannels(ec, "hsla", "channels", args[0], &hslSpace)
		}},
	)
}

// hslImpl implements the three- and four-argument hsl()/hsla() functions.
//
// Mirrors rgbImpl: special numbers fall back to plain CSS, $alpha clamps via
// %-or-unitless normalization, and the HSL arm of colorFromChannels warns
// for unitless saturation/lightness. Matches Dart: _hsl (color.dart:1361).
func hslImpl(ec *evalcontext.EvaluationContext, name string, args []value.Value) (value.Value, error) {
	var alpha value.Value
	if len(args) > 3 {
		alpha = args[3]
	}
	if args[0].IsSpecialNumber() || args[1].IsSpecialNumber() || args[2].IsSpecialNumber() ||
		(alpha != nil && alpha.IsSpecialNumber()) {
		result, err := functionString(name, args)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	hue, err := value.AssertNumber(args[0], new("hue"))
	if err != nil {
		return nil, err
	}
	saturation, err := value.AssertNumber(args[1], new("saturation"))
	if err != nil {
		return nil, err
	}
	lightness, err := value.AssertNumber(args[2], new("lightness"))
	if err != nil {
		return nil, err
	}

	alphaVal := 1.0
	if alpha != nil {
		aNum, err := value.AssertNumber(alpha, new("alpha"))
		if err != nil {
			return nil, err
		}
		p, err := percentageOrUnitless(aNum, 1, "alpha")
		if err != nil {
			return nil, err
		}
		alphaVal = clampLikeCSS(p, 0, 1)
	}
	return colorFromChannels(ec, value.HslColorSpace,
		hue,
		saturation,
		lightness,
		&alphaVal, false, true)
}

// ---- hwb() ----

// hwbCallable creates the module hwb() overloads.
//
// The four-argument form synthesizes a slash-separated channel list and
// re-enters parseChannels with an empty attribution name (matching Dart's
// unnamed _parseChannels('hwb', ...) call); the one-argument form forwards
// $channels. Matches Dart: module
// BuiltInCallable.overloadedFunction("hwb", ...) (color.dart:469).
func hwbCallable() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("hwb", "sass:color",
		OverloadDef{"$hue, $whiteness, $blackness, $alpha: 1", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if len(args) >= 3 {
				alpha := args[3]
				innerList, err := value.NewSassList([]value.Value{args[0], args[1], args[2]}, value.ListSeparatorSpace, false)
				if err != nil {
					return nil, err
				}
				outerList, err := value.NewSassList([]value.Value{innerList, alpha}, value.ListSeparatorSlash, false)
				if err != nil {
					return nil, err
				}
				hwbSpace := value.HwbColorSpace
				return parseChannels(ec, "hwb", "", outerList, &hwbSpace)
			}
			panic("no matching overload")
		}},
		OverloadDef{"$channels", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			if len(args) == 1 {
				hwbSpace := value.HwbColorSpace
				return parseChannels(ec, "hwb", "channels", args[0], &hwbSpace)
			}
			panic("no matching overload")
		}},
	)
}

// hwbGlobalCallable creates the deprecated-global hwb() callable ($channels).
//
// Same body as the one-argument module overload; the global registry adds
// the color deprecation warning. Matches Dart: global _function("hwb", ...)
// (color.dart:369).
func hwbGlobalCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("hwb", "$channels", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if len(args) == 1 {
			hwbSpace := value.HwbColorSpace
			return parseChannels(ec, "hwb", "channels", args[0], &hwbSpace)
		}
		panic("no matching overload")
	})
}

// labCallable creates the lab() callable ($channels).
//
// Forwards to parseChannels in the lab space, throwing Missing argument
// $channels when called with no arguments. Matches Dart:
// _function("lab", ...) (color.dart:376).
func labCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("lab", "$channels", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if len(args) == 0 {
			return nil, sasscommon.NewSassScriptException("Missing argument $channels.", nil)
		}
		labSpace := value.LabColorSpace
		return parseChannels(ec, "lab", "channels", args[0], &labSpace)
	})
}

// lchCallable creates the lch() callable ($channels).
//
// Same shape as labCallable, in the lch space. Matches Dart:
// _function("lch", ...) (color.dart:383).
func lchCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("lch", "$channels", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if len(args) == 0 {
			return nil, sasscommon.NewSassScriptException("Missing argument $channels.", nil)
		}
		lchSpace := value.LchColorSpace
		return parseChannels(ec, "lch", "channels", args[0], &lchSpace)
	})
}

// oklabCallable creates the oklab() callable ($channels).
//
// Same shape as labCallable, in the oklab space. Matches Dart:
// _function("oklab", ...) (color.dart:390).
func oklabCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("oklab", "$channels", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if len(args) == 0 {
			return nil, sasscommon.NewSassScriptException("Missing argument $channels.", nil)
		}
		oklabSpace := value.OklabColorSpace
		return parseChannels(ec, "oklab", "channels", args[0], &oklabSpace)
	})
}

// oklchCallable creates the oklch() callable ($channels).
//
// Same shape as labCallable, in the oklch space. Matches Dart:
// _function("oklch", ...) (color.dart:397).
func oklchCallable() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("oklch", "$channels", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if len(args) == 0 {
			return nil, sasscommon.NewSassScriptException("Missing argument $channels.", nil)
		}

		oklchSpace := value.OklchColorSpace
		return parseChannels(ec, "oklch", "channels", args[0], &oklchSpace)
	})
}

// ---- channel parsing (_parseChannels family) ----

// parseChannels parses channel-list input into a color, echoing plain CSS when unresolvable.
//
// With a non-nil space it parses three channels in that space; with nil
// (the color() function) the first element names the space, except the
// rgb/hsl/hwb/lab/lch/oklab/oklch spaces, which color() rejects with a "use
// the <space>() function" error. var()/from-led, special-number, and
// slash-alpha inputs fall back to functionString; name attributes errors to
// the source argument. Matches Dart: _parseChannels (color.dart:1600).
func parseChannels(ec *evalcontext.EvaluationContext, functionName string, name string, input value.Value, space *value.ColorSpace) (value.Value, error) {
	return parseChannelsWithName(ec, functionName, name, input, space)
}

// parseChannelsWithName is parseChannels with an explicit argument name for
// error context ($channels, $description, or empty for the synthetic HWB
// call). Go-only split of Dart's optional name parameter in _parseChannels
// (color.dart:1600).
func parseChannelsWithName(ec *evalcontext.EvaluationContext, functionName string, name string, input value.Value, space *value.ColorSpace) (value.Value, error) {
	if isSpecialVariable(input) {
		result, err := functionString(functionName, []value.Value{input})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// Split the slash-separated alpha off first: color(space c0 c1 c2 / alpha).
	// A nil result means the shape cannot parse and the input echoes as
	// plain CSS (color.dart:1723).
	parsed, err := parseSlashChannelsWithName(input, name)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		result, err := functionString(functionName, []value.Value{input})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	components := parsed.components
	alphaValue := parsed.alpha

	chList, err := value.AssertCommonListStyle(components, name, false)
	if err != nil {
		return nil, err
	}
	if len(chList) == 0 {
		return nil, sasscommon.NewSassScriptException("Color component list may not be empty.", new(name))
	}

	// A leading from keyword means relative-color syntax, which Sass leaves
	// for the browser (color.dart:1618).
	if s, ok := chList[0].(*value.SassString); ok && !s.HasQuotes && strings.ToLower(s.Text) == "from" {
		result, err := functionString(functionName, []value.Value{input})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	var channels []value.Value

	if components.IsSpecialVariable() {
		channels = chList
	} else {
		if space == nil {
			// color() function: first element is the space name
			spaceName, err := value.AssertString(chList[0], new(name))
			if err != nil {
				return nil, err
			}
			if err := spaceName.AssertUnquoted(); err != nil {
				return nil, sasscommon.NewSassScriptException(err.Error(), new(name))
			}
			if isSpecialVariable(spaceName) {
				result, err := functionString(functionName, []value.Value{input})
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			parsedSpace, err := value.ColorSpaceFromName(spaceName.Text, new(name))
			if err != nil {
				return nil, err
			}
			// Check for unsupported spaces in color()
			switch parsedSpace {
			case value.RgbColorSpace, value.HslColorSpace, value.HwbColorSpace,
				value.LabColorSpace, value.LchColorSpace, value.OklabColorSpace, value.OklchColorSpace:
				return nil, sasscommon.NewSassScriptException(
					fmt.Sprintf("The color() function doesn't support the color space %s. Use the %s() function instead.", parsedSpace.Name(), parsedSpace.Name()), new(name))
			}
			space = &parsedSpace
			channels = chList[1:]
		} else {
			channels = chList
		}

		// Non-number, non-special-number channels must be none; anything
		// else names the offending channel in the error (color.dart:1645).
		for i, ch := range channels {
			if !ch.IsSpecialNumber() {
				if _, ok := ch.(value.SassNumber); !ok && !isNone(ch) {
					chName := fmt.Sprintf("channel %d", i+1)
					if space != nil {
						chs := value.SpaceChannels(*space)
						if i < len(chs) {
							chName = chs[i].Name + " channel"
						}
					}
					chStr, err := ch.String()
					if err != nil {
						return nil, err
					}
					return nil, sasscommon.NewSassScriptException(
						fmt.Sprintf("Expected %s to be a number, was %s.", chName, chStr), new(name))
				}
			}
		}
	}

	var alpha *float64
	if alphaValue != nil {
		if alphaValue.IsSpecialNumber() {
			inSpecialCommaSpaces := false
			if space != nil {
				_, inSpecialCommaSpaces = specialCommaSpaces[*space]
			}
			if len(channels) == 3 && inSpecialCommaSpaces {
				// Special numbers in rgb/hsl re-emit comma-separated for
				// broader browser compatibility (color.dart:24).
				allArgs := make([]value.Value, len(channels)+1)
				copy(allArgs, channels)
				allArgs[len(channels)] = alphaValue
				result, err := functionString(functionName, allArgs)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			result, err := functionString(functionName, []value.Value{input})
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		if isNone(alphaValue) {
			// alpha stays nil to represent missing
		} else if an, ok := alphaValue.(value.SassNumber); ok {
			a, err := percentageOrUnitless(an, 1, "alpha")
			if err != nil {
				return nil, err
			}
			v := clampLikeCSS(a, 0, 1)
			alpha = &v
		} else {
			_, err := value.AssertNumber(alphaValue, new(name))
			if err != nil {
				return nil, err
			}
		}
	} else {
		v := 1.0
		alpha = &v
	}

	// A nil space means a var() component or space name; the check waits
	// until here (rather than returning early) so the alpha still validates
	// even for colors that cannot fully parse (color.dart:1684).
	if space == nil {
		result, err := functionString(functionName, []value.Value{input})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// Check for special numbers in channels
	for _, ch := range channels {
		if ch.IsSpecialNumber() {
			_, inSpecialCommaSpaces := specialCommaSpaces[*space]
			if len(channels) == 3 && inSpecialCommaSpaces {
				allArgs := make([]value.Value, len(channels))
				copy(allArgs, channels)
				if alphaValue != nil {
					allArgs = append(allArgs, alphaValue)
				}
				result, err := functionString(functionName, allArgs)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			result, err := functionString(functionName, []value.Value{input})
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	if len(channels) != 3 {
		inputStr, err := input.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("The %s color space has 3 channels but %s has %d.", (*space).Name(), inputStr, len(channels)), new(name))
	}

	return colorFromChannels(ec, *space,
		channelAsSassNumber(channels[0]),
		channelAsSassNumber(channels[1]),
		channelAsSassNumber(channels[2]),
		alpha, *space == value.RgbColorSpace, true)
}

// slashResult holds parseSlashChannelsWithName's split of the input: the
// space-separated components plus an optional slash-separated alpha.
// Matches Dart's (components, alpha) record in _parseSlashChannels
// (color.dart:1723).
type slashResult struct {
	components value.Value
	alpha      value.Value
}

// parseSlashChannels splits input into space-separated components plus an
// optional slash-separated alpha. Go-only split of Dart's optional name
// parameter in _parseSlashChannels (color.dart:1723).
func parseSlashChannels(input value.Value) (*slashResult, error) {
	return parseSlashChannelsWithName(input, "")
}

// parseSlashChannelsWithName splits input into components plus an optional
// alpha: slash-separated lists, trailing channel/alpha strings, and
// as-slash numbers. It returns nil when the shape cannot parse and the
// caller should echo the input as plain CSS (multi-slash strings).
// Matches Dart: _parseSlashChannels (color.dart:1723).
func parseSlashChannelsWithName(input value.Value, name string) (*slashResult, error) {
	list, err := value.AssertCommonListStyle(input, name, true)
	if err != nil {
		return nil, err
	}

	switch input.Separator() {
	case value.ListSeparatorSlash:
		if len(list) == 2 {
			return &slashResult{components: list[0], alpha: list[1]}, nil
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("Only 2 slash-separated elements allowed, but %d %s passed.", len(list), util.Pluralize("was", len(list), new("were"))),
			&name,
		)
	default:
		if len(list) > 0 {
			last := list[len(list)-1]
			if s, ok := last.(*value.SassString); ok && !s.HasQuotes {
				parts := strings.Split(s.Text, "/")
				switch len(parts) {
				case 1:
					return &slashResult{components: input, alpha: nil}, nil
				case 2:
					channel, err := parseNumberOrString(strings.TrimSpace(parts[0]))
					if err != nil {
						return nil, err
					}
					alpha, err := parseNumberOrString(strings.TrimSpace(parts[1]))
					if err != nil {
						return nil, err
					}
					initial := list[:len(list)-1]
					newList := make([]value.Value, len(initial)+1)
					copy(newList, initial)
					newList[len(initial)] = channel
					sassList, err := value.NewSassList(newList, value.ListSeparatorSpace, false)
					if err != nil {
						return nil, err
					}
					return &slashResult{
						components: sassList,
						alpha:      alpha,
					}, nil
				default:
					return nil, nil
				}
			}
			if n, ok := last.(value.SassNumber); ok && n.HasSlash() {
				before, after := n.SlashPair()
				initial := list[:len(list)-1]
				newList := make([]value.Value, len(initial)+1)
				copy(newList, initial)
				newList[len(initial)] = before
				sassList, err := value.NewSassList(newList, value.ListSeparatorSpace, false)
				if err != nil {
					return nil, err
				}
				return &slashResult{
					components: sassList,
					alpha:      after,
				}, nil
			}
		}
		return &slashResult{components: input, alpha: nil}, nil
	}
}

// parseNumberOrString parses text as a number, falling back to an unquoted
// string when the SCSS number parse fails with a format error. Matches Dart:
// _parseNumberOrString (color.dart:1753).
func parseNumberOrString(text string) (value.Value, error) {
	p := value.NewScssParser([]byte(text), nil, false)
	n, err := p.ParseNumber()
	if err != nil {
		var sassErr *sasscommon.SassFormatException
		var multiErr *sasscommon.MultiSpanSassFormatException
		if errors.As(err, &sassErr) || errors.As(err, &multiErr) {
			return &value.SassString{Text: text, HasQuotes: false}, nil
		}
		return nil, err
	}
	if n == nil {
		return &value.SassString{Text: text, HasQuotes: false}, nil
	}
	return n, nil
}

// channelAsSassNumber narrows v to a SassNumber, yielding nil for the
// unquoted none placeholder. Go-only shim for Dart's
// castOrNull<SassNumber> in _parseChannels (color.dart:1705): a non-number
// channel there must be none, already validated above.
func channelAsSassNumber(v value.Value) value.SassNumber {
	if n, ok := v.(value.SassNumber); ok {
		return n
	}
	return nil
}

// channelFromValue converts SassNumber n to a float64 per channel ch,
// returning nil for missing channels.
//
// Polar angles coerce to deg and wrap mod 360; percent-requiring channels
// reject unitless numbers; otherwise values normalize via
// percentageOrUnitless and clamp only when asked and the channel is clamped
// on that side. Matches Dart: _channelFromValue (color.dart:1846).
func channelFromValue(ch value.LinearChannel, n value.SassNumber, clamp bool) (*float64, error) {
	if n == nil {
		return nil, nil
	}
	if ch.IsPolarAngle {
		v, err := n.CoerceValueToUnit("deg", &ch.Name)
		if err != nil {
			return nil, err
		}
		v = math.Mod(v, 360)
		return &v, nil
	}
	if ch.RequiresPercent && !n.HasUnit("%") {
		nStr, err := n.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have unit \"%%\".", nStr), new(ch.Name))
	}
	v, err := percentageOrUnitless(n, ch.Max, ch.Name)
	if err != nil {
		return nil, err
	}
	if clamp && (ch.LowerClamped || ch.UpperClamped) {
		min := ch.Min
		if !ch.LowerClamped {
			min = math.Inf(-1)
		}
		max := ch.Max
		if !ch.UpperClamped {
			max = math.Inf(1)
		}
		v = clampLikeCSS(v, min, max)
	}
	return &v, nil
}

// colorFromChannels builds a SassColor in space from channel values,
// throwing when a channel is invalid.
//
// HSL warns for unitless saturation/lightness and forces %; HWB asserts %
// whiteness/blackness and rescales when they sum past 100; RGB tags colors
// from rgb()/rgba() with the rgb-function format; other spaces convert
// generically. When clamp is set, clamped channels clamp. Matches Dart:
// _colorFromChannels (color.dart:1765).
func colorFromChannels(ec *evalcontext.EvaluationContext, space value.ColorSpace, channel0, channel1, channel2 value.SassNumber, alpha *float64, fromRgbFunction bool, clamp bool) (value.Value, error) {
	switch space {
	case value.HslColorSpace:
		if channel1 != nil {
			if err := checkPercent(ec, channel1, "saturation"); err != nil {
				return nil, err
			}
		}
		if channel2 != nil {
			if err := checkPercent(ec, channel2, "lightness"); err != nil {
				return nil, err
			}
		}
		var c0 *float64
		if channel0 != nil {
			v, err := angleValue(ec, channel0, "hue")
			if err != nil {
				return nil, err
			}
			c0 = &v
		}
		chs := value.SpaceChannels(value.HslColorSpace)
		c1, err := channelFromValue(chs[1], forcePercent(channel1), clamp)
		if err != nil {
			return nil, err
		}
		c2, err := channelFromValue(chs[2], forcePercent(channel2), clamp)
		if err != nil {
			return nil, err
		}
		return newColorForSpaceInternal(value.HslColorSpace, c0, c1, c2, alpha)

	case value.HwbColorSpace:
		if channel1 != nil {
			if err := channel1.AssertUnit("%", new("whiteness")); err != nil {
				return nil, err
			}
		}
		if channel2 != nil {
			if err := channel2.AssertUnit("%", new("blackness")); err != nil {
				return nil, err
			}
		}
		var whiteness, blackness *float64
		if channel1 != nil {
			v := channel1.NumValue()
			whiteness = &v
		}
		if channel2 != nil {
			v := channel2.NumValue()
			blackness = &v
		}
		// Whiteness and blackness rescale proportionally when they sum past
		// 100 (color.dart:1799).
		if whiteness != nil && blackness != nil && *whiteness+*blackness > 100 {
			oldW := *whiteness
			*whiteness = *whiteness / (*whiteness + *blackness) * 100
			*blackness = *blackness / (oldW + *blackness) * 100
		}
		var c0 *float64
		if channel0 != nil {
			v, err := angleValue(ec, channel0, "hue")
			if err != nil {
				return nil, err
			}
			c0 = &v
		}
		return newColorForSpaceInternal(value.HwbColorSpace, c0, whiteness, blackness, alpha)

	case value.RgbColorSpace:
		chs := value.SpaceChannels(value.RgbColorSpace)
		var c0, c1, c2 *float64
		if channel0 != nil {
			v, err := channelFromValue(chs[0], channel0, clamp)
			if err != nil {
				return nil, err
			}
			c0 = v
		}
		if channel1 != nil {
			v, err := channelFromValue(chs[1], channel1, clamp)
			if err != nil {
				return nil, err
			}
			c1 = v
		}
		if channel2 != nil {
			v, err := channelFromValue(chs[2], channel2, clamp)
			if err != nil {
				return nil, err
			}
			c2 = v
		}
		var format *value.ColorFormat
		if fromRgbFunction {
			f := value.ColorFormatRGBFunction
			format = &f
		}
		return value.NewColorRGBInternal(c0, c1, c2, alpha, format)

	default:
		chs := value.SpaceChannels(space)
		var c0, c1, c2 *float64
		v0, err := channelFromValue(chs[0], channel0, clamp)
		if err != nil {
			return nil, err
		}
		c0 = v0
		v1, err := channelFromValue(chs[1], channel1, clamp)
		if err != nil {
			return nil, err
		}
		c1 = v1
		v2, err := channelFromValue(chs[2], channel2, clamp)
		if err != nil {
			return nil, err
		}
		c2 = v2
		return newColorForSpaceInternal(space, c0, c1, c2, alpha)
	}
}
