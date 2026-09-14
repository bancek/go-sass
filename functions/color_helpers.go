// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/color.dart (helpers slice: _colorInSpace,
// _channelName, _angleValue, _checkPercent, _percentageOrUnitless,
// _channelFunction, _removedColorFunction, _function, _functionString,
// _forcePercent, _isNone, _microsoftFilterStart, _specialCommaSpaces;
// warnForGlobalBuiltIn informally follows lib/src/callable/async_built_in.dart)

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// microsoftFilterRegex matches the start of a proprietary Microsoft filter
// declaration (e.g. `alpha(opacity=...)`), porting Dart's
// _microsoftFilterStart. Such strings take the plain-CSS path instead of the
// Sass color logic.
var microsoftFilterRegex = regexp.MustCompile(`^[a-zA-Z]+\s*=`)

// isMicrosoftFilter reports whether val is an unquoted string starting a
// proprietary Microsoft filter declaration. Callers use it to take the
// plain-CSS function path instead of Sass color evaluation.
func isMicrosoftFilter(val value.Value) bool {
	s, ok := val.(*value.SassString)
	return ok && !s.HasQuotes && microsoftFilterRegex.MatchString(s.Text)
}

// specialCommaSpaces lists the spaces where a special-number argument still
// serializes through the comma-separated three-/four-argument syntax for
// wider browser compatibility, porting Dart's _specialCommaSpaces (rgb,
// hsl). The channel parsers in color_spaces.go consult it.
var specialCommaSpaces = map[value.ColorSpace]struct{}{
	value.RgbColorSpace: {},
	value.HslColorSpace: {},
}

// isSpecialVariable reports whether val is an unquoted string holding a CSS
// variable or environment reference. It delegates to the Value interface's
// own check (Dart's `isSpecialVariable` getter); color constructors use it
// to return plain-CSS calls that can only resolve at browse time.
func isSpecialVariable(val value.Value) bool {
	return val.IsSpecialVariable()
}

// colorInSpace converts a color into the space named by a $space argument,
// porting Dart's _colorInSpace. Null passes the color through unchanged;
// otherwise the space must name an unquoted color space (errors attribute to
// $space). legacyMissing defaults to true; callers like to-space pass false
// so legacy missing channels resolve to zero on conversion.
func colorInSpace(colorUntyped *value.SassColor, spaceUntyped value.Value, legacyMissing *bool) (*value.SassColor, error) {
	lm := true
	if legacyMissing != nil {
		lm = *legacyMissing
	}
	if spaceUntyped == value.Null {
		return colorUntyped, nil
	}
	// A $space value must be an unquoted string; the name attaches to the
	// error since Dart threads it through the cascade operator instead.
	spaceStr, err := value.AssertString(spaceUntyped, new("space"))
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
	result, err := colorUntyped.ToSpace(space, &lm)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// channelNameFromArg validates a $channel argument, porting Dart's
// _channelName: the value must be a quoted string, and errors attribute to
// $channel.
func channelNameFromArg(v value.Value) (string, error) {
	s, err := value.AssertString(v, new("channel"))
	if err != nil {
		return "", err
	}
	if err := s.AssertQuoted(); err != nil {
		return "", sasscommon.NewSassScriptException(err.Error(), new("channel"))
	}
	return s.Text, nil
}

// checkChannelExists verifies that channelName is a channel of the color's
// space (alpha always counts). Go-only helper with no Dart counterpart; the
// channel/is-missing paths use it to attribute unknown names before reading.
func checkChannelExists(col *value.SassColor, channelName, argName string) error {
	chs := value.SpaceChannels(col.Space())
	for _, ch := range chs {
		if ch.Name == channelName {
			return nil
		}
	}
	if channelName == "alpha" {
		return nil
	}
	colStr, err := col.String()
	if err != nil {
		return err
	}
	return sasscommon.NewSassScriptException(fmt.Sprintf("Color %s has no channel named %q.", colStr, channelName), new(argName))
}

// toXyzNoMissing converts a color to xyz-d65 with missing channels resolved
// to zero, porting the local helper inside Dart's `same()` closure. Colors
// already there without missing channels pass through; xyz-d65 colors with
// missing channels are rebuilt with zeros; other spaces convert explicitly
// so no intermediate missing-channel color is allocated.
func toXyzNoMissing(col *value.SassColor) (*value.SassColor, error) {
	// Already xyz-d65 without missing channels: return as-is.
	if col.Space() == value.XyzD65ColorSpace && !col.HasMissingChannel() {
		return col, nil
	}
	// Already xyz-d65 but with missing channels: rebuild with zeros.
	if col.Space() == value.XyzD65ColorSpace {
		return value.NewColorXYZD65(col.Channel0(), col.Channel1(), col.Channel2(), col.Alpha())
	}
	// Any other space: convert explicitly so missing channels become zero
	// without intermediate color objects.
	// Dart: space.convert(ColorSpace.xyzD65, channel0, channel1, channel2, alpha)
	c0, c1, c2, alpha := col.Channel0(), col.Channel1(), col.Channel2(), col.Alpha()
	return col.Space().Convert(value.XyzD65ColorSpace, &c0, &c1, &c2, &alpha)
}

// warnForGlobalBuiltIn warns that a global built-in should be called through
// its module instead. Pointer only: the message mirrors
// warnForGlobalBuiltIn in lib/src/callable/async_built_in.dart (Dart Sass
// 3.0.0 removal notice with the module.name replacement); the actual global
// wrappers here go through WithDeprecationWarning instead, and only a few
// plain-CSS fallthroughs call this directly.
func warnForGlobalBuiltIn(ec *evalcontext.EvaluationContext, module, name string) error {
	return ec.WarnDeprecation(
		"Global built-in functions are deprecated and will be removed in Dart "+
			"Sass 3.0.0.\n"+
			"Use "+module+"."+name+" instead.\n\n"+
			"More info and automated migrator: https://sass-lang.com/d/import",
		deprecation.GlobalBuiltin,
	)
}

// ---- Helpers ----

// functionString renders name with its arguments as a plain-CSS call,
// porting Dart's _functionString. Each argument serializes to CSS, so values
// that can only resolve at browse time (special numbers, var()/env())
// survive verbatim instead of evaluating.
func functionString(name string, arguments []value.Value) (*value.SassString, error) {
	strs := make([]string, len(arguments))
	for i, arg := range arguments {
		cssStr, err := arg.ToCssString(true)
		if err != nil {
			return nil, err
		}
		strs[i] = cssStr
	}
	return &value.SassString{Text: name + "(" + strings.Join(strs, ", ") + ")", HasQuotes: false}, nil
}

// clampLikeCSS clamps val to [min, max] with NaN preferring the lower bound,
// porting Dart's clampLikeCss (via util.ClampLikeCss; negative zero is
// normalized per sass/dart-sass#2840).
func clampLikeCSS(val, min, max float64) float64 {
	return util.ClampLikeCss(val, min, max)
}

// percentageOrUnitless normalizes n for channel math, porting Dart's
// _percentageOrUnitless. Unitless numbers pass through; percentages scale so
// 0% is 0 and 100% is max; any other unit fails naming name.
func percentageOrUnitless(n value.SassNumber, max float64, name string) (float64, error) {
	if !n.HasUnits() {
		return n.NumValue(), nil
	}
	if n.HasUnit("%") {
		return max * n.NumValue() / 100, nil
	}
	nStr, err := n.String()
	if err != nil {
		return 0, err
	}
	return 0, sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to have unit \"%%\" or no units.", nStr), new(name))
}

// checkPercent warns when n omits the % unit, porting Dart's _checkPercent.
// Callers normalize to % afterwards so behavior holds through the
// deprecation period.
func checkPercent(ec *evalcontext.EvaluationContext, n value.SassNumber, name string) error {
	if n.HasUnit("%") {
		return nil
	}
	nStr, err := n.String()
	if err != nil {
		return err
	}
	return ec.WarnDeprecation(
		fmt.Sprintf("$%s: Passing a number without unit %% (%s) is deprecated.\n"+
			"\n"+
			"To preserve current behavior: %s\n"+
			"\n"+
			"More info: https://sass-lang.com/d/function-units",
			name, nStr, n.UnitSuggestion(name, new("%"))),
		deprecation.FunctionUnits,
	)
}

// angleValue asserts its argument is a number and returns its degrees,
// porting Dart's _angleValue. Values compatible with deg coerce; anything
// else warns (with a unit suggestion) and reads as bare degrees through the
// deprecation period.
func angleValue(ec *evalcontext.EvaluationContext, angleValue value.Value, name string) (float64, error) {
	angle, err := value.AssertNumber(angleValue, &name)
	if err != nil {
		return 0, err
	}
	if angle.CompatibleWithUnit("deg") {
		return angle.CoerceValueToUnit("deg", &name)
	}

	angleStr, err := angle.String()
	if err != nil {
		return 0, err
	}
	if err := ec.WarnDeprecation(fmt.Sprintf(
		"$%s: Passing a unit other than deg (%s) is deprecated.\n"+
			"\n"+
			"To preserve current behavior: %s\n"+
			"\n"+
			"See https://sass-lang.com/d/function-units",
		name, angleStr, angle.UnitSuggestion(name, nil)), deprecation.FunctionUnits); err != nil {
		return 0, err
	}
	return angle.NumValue(), nil
}

// isNone reports whether val is the unquoted string "none"
// (case-insensitive), porting Dart's _isNone. Channel writers use it to mark
// a channel missing.
func isNone(val value.Value) bool {
	s, ok := val.(*value.SassString)
	return ok && !s.HasQuotes && strings.ToLower(s.Text) == "none"
}

// forcePercent returns n with unit % regardless of its original unit,
// porting Dart's _forcePercent. Nil stays nil and % numbers pass through,
// so hsl saturation/lightness always convert from percentages.
func forcePercent(n value.SassNumber) value.SassNumber {
	if n == nil {
		return nil
	}
	if len(n.NumNumeratorUnits()) == 1 && n.NumNumeratorUnits()[0] == "%" && len(n.NumDenominatorUnits()) == 0 {
		return n
	}
	return value.NewSingleUnitNumber(n.NumValue(), "%")
}

// ---- Channel Functions ----

// channelFunction builds a deprecated module channel getter (e.g. red,
// hue) reading one channel via getter, porting one arm of Dart's
// _channelFunction. The companions below select the unit and the
// global-vs-module call site; all four funnel into
// newBuiltInChannelFunction.
func channelFunction(name string, space value.ColorSpace, getter func(*value.SassColor) (float64, error)) *BuiltInCallable {
	return newBuiltInChannelFunction(name, space, getter, "", false)
}

// channelFunctionGlobal builds the deprecated global form of a channel
// getter (same _channelFunction arm with global set).
func channelFunctionGlobal(name string, space value.ColorSpace, getter func(*value.SassColor) (float64, error)) *BuiltInCallable {
	return newBuiltInChannelFunction(name, space, getter, "", true)
}

// channelFunctionWithUnit builds the module form with a result unit (deg
// for hue, % for saturation-like channels).
func channelFunctionWithUnit(name string, space value.ColorSpace, getter func(*value.SassColor) (float64, error), unit string) *BuiltInCallable {
	return newBuiltInChannelFunction(name, space, getter, unit, false)
}

// channelFunctionGlobalWithUnit builds the global form with a result unit.
func channelFunctionGlobalWithUnit(name string, space value.ColorSpace, getter func(*value.SassColor) (float64, error), unit string) *BuiltInCallable {
	return newBuiltInChannelFunction(name, space, getter, unit, true)
}

// newBuiltInChannelFunction builds the deprecated channel getter itself:
// it reads the channel, warns toward color.channel($color, "name", $space:
// ...), and returns the value with the channel's unit.
func newBuiltInChannelFunction(name string, space value.ColorSpace, getter func(*value.SassColor) (float64, error), unit string, global bool) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$color", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		c, err := value.AssertColor(args[0], new("color"))
		if err != nil {
			return nil, err
		}
		// The message suggests the color.channel replacement naming this
		// getter and space; globals omit the "color." prefix.
		var prefix string
		if global {
			prefix = ""
		} else {
			prefix = "color."
		}
		if err := ec.WarnDeprecation(fmt.Sprintf(
			"%s%s() is deprecated. Suggestion:\n"+
				"\n"+
				"color.channel($color, %q, $space: %s)\n"+
				"\n"+
				"More info: https://sass-lang.com/d/color-functions",
			prefix, name, name, space.Name()), deprecation.ColorFunctions); err != nil {
			return nil, err
		}
		var unitSlice []string
		if unit != "" {
			unitSlice = []string{unit}
		}
		v, err := getter(c)
		if err != nil {
			return nil, err
		}
		return value.SassNumberWithUnits(v, unitSlice, nil), nil
	})
}

// newColorForSpaceInternal builds a color in space from raw channels,
// delegating to the value package's internal constructor. The adjust/scale
// paths use it so missing channels survive the write.
func newColorForSpaceInternal(space value.ColorSpace, c0, c1, c2, alpha *float64) (*value.SassColor, error) {
	return value.NewColorForSpaceInternal(space, c0, c1, c2, alpha)
}

// removedColorFunction builds a module stub for a function deleted from
// sass:color (e.g. lighten), porting Dart's _removedColorFunction. It always
// errors, recommending the color.adjust call that passes $amount to the
// named channel.
func removedColorFunction(name string, argument string) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		arg0Str, err := args[0].String()
		if err != nil {
			return nil, err
		}
		arg1Str, err := args[1].String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("The function %s() isn't in the sass:color module.\n\nRecommendation: color.adjust(%s, $%s: %s)\n\nMore info: https://sass-lang.com/documentation/functions/color#%s",
				name, arg0Str, argument, arg1Str, name), nil)
	})
}

// removedColorFunctionNegative builds the same stub where the recommendation
// negates the amount (e.g. darken lowers lightness), porting Dart's
// `negative: true` arm.
func removedColorFunctionNegative(name string, argument string) *BuiltInCallable {
	return MustNewBuiltInCallableFunction(name, "$color, $amount", "sass:color", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		arg0Str, err := args[0].String()
		if err != nil {
			return nil, err
		}
		arg1Str, err := args[1].String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("The function %s() isn't in the sass:color module.\n\nRecommendation: color.adjust(%s, $%s: -%s)\n\nMore info: https://sass-lang.com/documentation/functions/color#%s",
				name, arg0Str, argument, arg1Str, name), nil)
	})
}

// roundInt rounds v to the nearest integer for hex formatting (the
// ie-hex-str path). Go-only helper with no Dart counterpart; Dart formats
// through fuzzyRound inline in the _ieHexStr closure.
func roundInt(v float64) int64 {
	return int64(math.Round(v))
}
