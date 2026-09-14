// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/color.dart (core registries: global + module
// lists; module-only space, to-space, is-legacy, is-missing, is-in-gamut,
// to-gamut, channel, same, and is-powerless closures)

import (
	"fmt"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// GlobalColorFunctions returns all globally-available color functions.
//
// It ports Dart's `global` list: the RGB/HSL channel getters, mix, the
// shared rgb/rgba/hsl/hsla overload sets, the legacy-only adjust-hue/
// lighten/darken/saturate/desaturate and opacify/fade/transparentize forms
// (renamed toward `adjust` where Dart attaches a second deprecation name),
// the Microsoft-filter-aware alpha/opacity/grayscale/saturate fallthroughs,
// the color/hwb/lab/lch/oklab/oklch constructors, complement, ie-hex-str,
// and the adjust-color/scale-color/change-color renames — each wrapped in a
// `color` deprecation warning. The rgb/hsl-family constructors themselves
// live in color_spaces.go and the legacy adjusters in color_manipulation.go;
// this registry only wires them in.
func GlobalColorFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		// ### RGB
		channelFunctionGlobal("red", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Red() }).WithDeprecationWarning("color", nil),
		channelFunctionGlobal("green", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Green() }).WithDeprecationWarning("color", nil),
		channelFunctionGlobal("blue", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Blue() }).WithDeprecationWarning("color", nil),
		mixCallable().WithDeprecationWarning("color", nil),
		rgbCallable(),
		rgbaCallable(),
		invertCallable(),

		// ### HSL
		channelFunctionGlobalWithUnit("hue", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Hue() }, "deg").WithDeprecationWarning("color", nil),
		channelFunctionGlobalWithUnit("saturation", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Saturation() }, "%").WithDeprecationWarning("color", nil),
		channelFunctionGlobalWithUnit("lightness", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Lightness() }, "%").WithDeprecationWarning("color", nil),
		hslCallable(),
		hslaCallable(),
		grayscaleCallable(),
		adjustHueCallable().WithDeprecationWarning("color", new("adjust")),
		lightenCallable().WithDeprecationWarning("color", new("adjust")),
		darkenCallable().WithDeprecationWarning("color", new("adjust")),
		saturateCallable(),
		desaturateCallable().WithDeprecationWarning("color", new("adjust")),

		// ### Opacity
		opacifyCallable("opacify").WithDeprecationWarning("color", new("adjust")),
		opacifyCallable("fade-in").WithDeprecationWarning("color", new("adjust")),
		transparentizeCallable("transparentize").WithDeprecationWarning("color", new("adjust")),
		transparentizeCallable("fade-out").WithDeprecationWarning("color", new("adjust")),

		alphaCallable(),
		opacityCallable(),

		// ### Color Spaces
		MustNewBuiltInCallableFunction("color", "$description", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return parseChannels(ec, "color", "description", args[0], nil)
		}),
		hwbGlobalCallable(),
		labCallable(),
		lchCallable(),
		oklabCallable(),
		oklchCallable(),
		complementCallable().WithDeprecationWarning("color", nil),

		// ### Miscellaneous
		ieHexStrCallable(),
		adjustColorCallable().WithDeprecationWarning("color", nil).WithName("adjust-color"),
		scaleColorCallable().WithDeprecationWarning("color", nil).WithName("scale-color"),
		changeColorCallable().WithDeprecationWarning("color", nil).WithName("change-color"),
	}
}

// ColorModule returns the sass:color built-in module.
//
// It ports Dart's `module`: the channel getters, mix, the module invert and
// grayscale forms, the removed-function stubs that redirect toward
// color.adjust, the hwb constructor plus whiteness/blackness getters, the
// module alpha/opacity forms, the adjust/scale/change/ie-hex-str members,
// and the module-only closures defined below (space, to-space, is-legacy,
// is-missing, is-in-gamut, to-gamut, channel, same, is-powerless).
func ColorModule() *sassmodule.BuiltInModule {
	return sassmodule.NewBuiltInModule("color", []sasscallable.Callable{
		// ### RGB
		channelFunction("red", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Red() }),
		channelFunction("green", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Green() }),
		channelFunction("blue", value.RgbColorSpace, func(c *value.SassColor) (float64, error) { return c.Blue() }),
		mixCallable(),

		invertModuleCallable(),

		// ### HSL
		channelFunctionWithUnit("hue", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Hue() }, "deg"),
		channelFunctionWithUnit("saturation", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Saturation() }, "%"),
		channelFunctionWithUnit("lightness", value.HslColorSpace, func(c *value.SassColor) (float64, error) { return c.Lightness() }, "%"),

		// In the module these throw errors telling users to use color.adjust()
		removedColorFunction("adjust-hue", "hue"),
		removedColorFunction("lighten", "lightness"),
		removedColorFunctionNegative("darken", "lightness"),
		removedColorFunction("saturate", "saturation"),
		removedColorFunctionNegative("desaturate", "saturation"),

		grayscaleModuleCallable(),

		// ### HWB
		hwbCallable(),
		channelFunctionWithUnit("whiteness", value.HwbColorSpace, func(c *value.SassColor) (float64, error) { return c.Whiteness() }, "%"),
		channelFunctionWithUnit("blackness", value.HwbColorSpace, func(c *value.SassColor) (float64, error) { return c.Blackness() }, "%"),

		// ### Opacity
		removedColorFunction("opacify", "alpha"),
		removedColorFunction("fade-in", "alpha"),
		removedColorFunctionNegative("transparentize", "alpha"),
		removedColorFunctionNegative("fade-out", "alpha"),

		alphaModuleCallable(),
		opacityModuleCallable(),

		// ### Color Spaces
		// space($color) reports the color's space name as an unquoted string.
		MustNewBuiltInCallableFunction("space", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			return &value.SassString{Text: col.Space().Name(), HasQuotes: false}, nil
		}),

		// to-space($color, $space) converts into the named space. Missing
		// channels in legacy spaces resolve to zero on conversion, since
		// callers reaching for a legacy space generally want the most
		// compatible color rather than a missing-channel placeholder.
		MustNewBuiltInCallableFunction("to-space", "$color, $space", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			spaceVal := args[1]
			// The space must be an unquoted string naming a color space.
			spaceStr, err := value.AssertString(spaceVal, new("space"))
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
			result, err := col.ToSpace(space, new(false))
			if err != nil {
				return nil, err
			}
			return result, nil
		}),

		// is-legacy($color) reports whether the color lives in a legacy
		// (rgb/hsl/hwb) space.
		MustNewBuiltInCallableFunction("is-legacy", "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			if col.IsLegacy() {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		// is-missing($color, $channel) reports whether the named channel is
		// missing. The channel argument must be a quoted string; errors carry
		// the color/channel argument names.
		MustNewBuiltInCallableFunction("is-missing", "$color, $channel", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			channelName, err := channelNameFromArg(args[1])
			if err != nil {
				return nil, err
			}
			isMissing, err := col.IsChannelMissingByName(channelName)
			if err != nil {
				return nil, err
			}
			if isMissing {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		// is-in-gamut($color, $space: null) converts into the working space
		// (defaulting to the color's own) and reports whether it is in gamut.
		MustNewBuiltInCallableFunction("is-in-gamut", "$color, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			workingColor, err := colorInSpace(col, args[1], nil)
			if err != nil {
				return nil, err
			}
			if workingColor.IsInGamut() {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		// to-gamut($color, $space: null, $method: null) maps the color into the
		// gamut of the working space (defaulting to the color's own) and
		// converts back. The method argument is mandatory for forwards
		// compatibility with CSS spec changes; unbounded spaces return the
		// color untouched. The method name is validated before the bounded
		// check so bad names always error.
		MustNewBuiltInCallableFunction("to-gamut", "$color, $space: null, $method: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			var space value.ColorSpace
			if args[1] != value.Null {
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
			} else {
				space = col.Space()
			}
			if args[2] == value.Null {
				return nil, sasscommon.NewSassScriptException(
					"color.to-gamut() requires a $method argument for forwards-"+
						"compatibility with changes in the CSS spec. Suggestion:\n"+
						"\n"+
						"$method: local-minde",
					new("method"),
				)
			}
			methodStr, err := value.AssertString(args[2], new("method"))
			if err != nil {
				return nil, err
			}
			if err := methodStr.AssertUnquoted(); err != nil {
				return nil, sasscommon.NewSassScriptException(err.Error(), new("method"))
			}
			method, err := value.GamutMapMethodFromName(methodStr.Text)
			if err != nil {
				return nil, err
			}
			if !space.IsBounded() {
				return col, nil
			}
			spaceConverted, err := col.ToSpace(space, nil)
			if err != nil {
				return nil, err
			}
			gamutColor, err := spaceConverted.ToGamut(method)
			if err != nil {
				return nil, err
			}
			result, err := gamutColor.ToSpace(col.Space(), new(false))
			if err != nil {
				return nil, err
			}
			return result, nil
		}),

		// channel($color, $channel, $space: null) reads one channel in the
		// working space (defaulting to the color's own). Alpha returns
		// unitless; percentage-associated channels are rescaled to 0-100 with
		// "%", and an unknown channel name fails against $channel.
		MustNewBuiltInCallableFunction("channel", "$color, $channel, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			workingColor, err := colorInSpace(col, args[2], nil)
			if err != nil {
				return nil, err
			}
			channelName, err := channelNameFromArg(args[1])
			if err != nil {
				return nil, err
			}
			if channelName == "alpha" {
				return value.NewUnitlessNumber(workingColor.Alpha()), nil
			}
			chs := value.SpaceChannels(workingColor.Space())
			found := false
			var channelValue float64
			var unit string
			for i, ch := range chs {
				if ch.Name == channelName {
					found = true
					channelValue = workingColor.Channels()[i]
					unit = ch.AssociatedUnit
					if unit == "%" {
						channelValue = channelValue * 100 / ch.Max
					}
					break
				}
			}
			if !found {
				colStr, err := workingColor.String()
				if err != nil {
					return nil, err
				}
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Color %s has no channel named %s.", colStr, channelName), new("channel"))
			}
			if unit != "" {
				return value.NewSingleUnitNumber(channelValue, unit), nil
			}
			return value.NewUnitlessNumber(channelValue), nil
		}),

		// same($color1, $color2) reports visual equality. Colors sharing a space
		// compare channel-wise with fuzzy equality; colors in different spaces
		// are each converted to xyz-d65 with missing channels resolved to
		// zero (avoiding intermediate color objects) and then compared.
		MustNewBuiltInCallableFunction("same", "$color1, $color2", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col1, err := value.AssertColor(args[0], new("color1"))
			if err != nil {
				return nil, err
			}
			col2, err := value.AssertColor(args[1], new("color2"))
			if err != nil {
				return nil, err
			}
			var same bool
			if col1.Space() == col2.Space() {
				same = util.FuzzyEquals(col1.Channel0(), col2.Channel0()) &&
					util.FuzzyEquals(col1.Channel1(), col2.Channel1()) &&
					util.FuzzyEquals(col1.Channel2(), col2.Channel2()) &&
					util.FuzzyEquals(col1.Alpha(), col2.Alpha())
			} else {
				x1, err := toXyzNoMissing(col1)
				if err != nil {
					return nil, err
				}
				x2, err := toXyzNoMissing(col2)
				if err != nil {
					return nil, err
				}
				same = x1.Equals(x2)
			}
			if same {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		// is-powerless($color, $channel, $space: null) reports whether the named
		// channel is powerless in the working space (defaulting to the
		// color's own), with the same quoted-$channel validation as
		// is-missing.
		MustNewBuiltInCallableFunction("is-powerless", "$color, $channel, $space: null", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			col, err := value.AssertColor(args[0], new("color"))
			if err != nil {
				return nil, err
			}
			workingColor, err := colorInSpace(col, args[2], nil)
			if err != nil {
				return nil, err
			}
			channelName, err := channelNameFromArg(args[1])
			if err != nil {
				return nil, err
			}
			isPowerless, err := workingColor.IsChannelPowerlessByName(channelName)
			if err != nil {
				return nil, err
			}
			if isPowerless {
				return value.SassTrue, nil
			}
			return value.SassFalse, nil
		}),

		complementCallable(),
		adjustColorCallable(),
		scaleColorCallable(),
		changeColorCallable(),
		ieHexStrCallable(),
	}, nil, nil)
}
