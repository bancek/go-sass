// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/interpolation_method.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// HueInterpolationMethod represents how to interpolate hues between two colors.
//
// Each variant names an arc rule from CSS Color 4 for adjusting the hue delta
// θ₂ − θ₁ before blending; the surrounding InterpolationMethod selects which
// applies. Used by SassColor interpolation.
//
// Matches Dart: HueInterpolationMethod
type HueInterpolationMethod int

const (
	// HueInterpolationShorter takes the shorter arc, adjusting angles so that
	// θ₂ − θ₁ ∈ [-180, 180]. This is the default for polar spaces.
	//
	// https://www.w3.org/TR/css-color-4/#shorter
	// Matches Dart: HueInterpolationMethod.shorter
	HueInterpolationShorter HueInterpolationMethod = iota
	// HueInterpolationLonger takes the longer arc, adjusting angles so that
	// θ₂ − θ₁ ∈ {0, [180, 360)}.
	//
	// https://www.w3.org/TR/css-color-4/#hue-longer
	// Matches Dart: HueInterpolationMethod.longer
	HueInterpolationLonger
	// HueInterpolationIncreasing always goes up, adjusting angles so that
	// θ₂ − θ₁ ∈ [0, 360).
	//
	// https://www.w3.org/TR/css-color-4/#hue-increasing
	// Matches Dart: HueInterpolationMethod.increasing
	HueInterpolationIncreasing
	// HueInterpolationDecreasing always goes down, adjusting angles so that
	// θ₂ − θ₁ ∈ (-360, 0].
	//
	// https://www.w3.org/TR/css-color-4/#hue-decreasing
	// Matches Dart: HueInterpolationMethod.decreasing
	HueInterpolationDecreasing
)

// String returns the CSS keyword for a hue interpolation method
// ("shorter", "longer", "increasing", or "decreasing").
// (Dart: HueInterpolationMethod's enum name rendering.)
func (him HueInterpolationMethod) String() string {
	switch him {
	case HueInterpolationShorter:
		return "shorter"
	case HueInterpolationLonger:
		return "longer"
	case HueInterpolationIncreasing:
		return "increasing"
	case HueInterpolationDecreasing:
		return "decreasing"
	default:
		return fmt.Sprintf("HueInterpolationMethod(%d)", him)
	}
}

// InterpolationMethod represents how to interpolate between two colors.
//
// It pairs a color space with an optional hue rule. Used by SassColor
// interpolation.
//
// Matches Dart: InterpolationMethod
type InterpolationMethod struct {
	// Space is the color space in which to perform the interpolation.
	Space ColorSpace
	// Hue selects the hue-arc rule. It is non-nil if and only if Space is
	// polar; rectangular spaces interpolate without a hue method.
	Hue *HueInterpolationMethod
}

// NewInterpolationMethod creates an interpolation method.
//
// A polar space with no hue argument defaults to the shorter arc; a hue
// argument on a rectangular space is an error, since rectangular interpolation
// has no hue angle to adjust.
//
// Matches Dart: InterpolationMethod.new
func NewInterpolationMethod(space ColorSpace, hue *HueInterpolationMethod) (InterpolationMethod, error) {
	if !space.IsPolar() {
		if hue != nil {
			return InterpolationMethod{}, &sasscommon.ArgumentError{Message: fmt.Sprintf("Hue interpolation method may not be set for rectangular color space %s.", space.Name())}
		}
		return InterpolationMethod{Space: space}, nil
	}
	method := HueInterpolationShorter
	if hue != nil {
		method = *hue
	}
	return InterpolationMethod{Space: space, Hue: &method}, nil
}

// InterpolationMethodFromValue parses a SassScript value representing an
// interpolation method, not beginning with "in".
//
// The value must be an unbracketed space-separated list: the space name, then
// optionally a hue method followed by the unquoted word "hue" (at most three
// elements). Comma- or slash-separated and bracketed lists are rejected up
// front; each element is then validated in order with script errors naming the
// argument. A hue rule on a rectangular space is rejected.
//
// Matches Dart: InterpolationMethod.fromValue
func InterpolationMethodFromValue(val Value, name *string) (InterpolationMethod, error) {
	p := name
	if p == nil {
		p = new("value")
	}

	if val.Separator() == ListSeparatorComma || val.Separator() == ListSeparatorSlash || val.HasBrackets() {
		valStr, err := val.String()
		if err != nil {
			return InterpolationMethod{}, err
		}
		return InterpolationMethod{}, sasscommon.NewSassScriptException(fmt.Sprintf("Expected a space-separated list, was %s", valStr), p)
	}

	list, err := val.AsList()
	if err != nil {
		return InterpolationMethod{}, err
	}
	if len(list) == 0 {
		return InterpolationMethod{}, sasscommon.NewSassScriptException("Expected a color interpolation method, got an empty list.", p)
	}

	spaceStr, err := AssertString(list[0], name)
	if err != nil {
		return InterpolationMethod{}, err
	}
	if err := spaceStr.AssertUnquoted(); err != nil {
		return InterpolationMethod{}, sasscommon.NewSassScriptException(err.Error(), p)
	}
	space, err := ColorSpaceFromName(spaceStr.Text, name)
	if err != nil {
		return InterpolationMethod{}, err
	}

	if len(list) == 1 {
		return NewInterpolationMethod(space, nil)
	}

	hueMethod, err := hueInterpolationMethodFromValue(list[1], name)
	if err != nil {
		return InterpolationMethod{}, err
	}

	if len(list) == 2 {
		valStr, err := val.String()
		if err != nil {
			return InterpolationMethod{}, err
		}
		return InterpolationMethod{}, sasscommon.NewSassScriptException(fmt.Sprintf("Expected unquoted string \"hue\" after %s.", valStr), p)
	}

	thirdStr, err := AssertString(list[2], name)
	if err != nil {
		return InterpolationMethod{}, err
	}
	if err := thirdStr.AssertUnquoted(); err != nil {
		return InterpolationMethod{}, sasscommon.NewSassScriptException(err.Error(), p)
	}
	if strings.ToLower(thirdStr.Text) != "hue" {
		valStr, err := val.String()
		if err != nil {
			return InterpolationMethod{}, err
		}
		list2Str, err := list[2].String()
		if err != nil {
			return InterpolationMethod{}, err
		}
		return InterpolationMethod{}, sasscommon.NewSassScriptException(fmt.Sprintf("Expected unquoted string \"hue\" at the end of %s, was %s.", valStr, list2Str), p)
	}

	if len(list) > 3 {
		valStr, err := val.String()
		if err != nil {
			return InterpolationMethod{}, err
		}
		return InterpolationMethod{}, sasscommon.NewSassScriptException(fmt.Sprintf("Expected nothing after \"hue\" in %s.", valStr), p)
	}

	if !space.IsPolar() {
		return InterpolationMethod{}, sasscommon.NewSassScriptException(fmt.Sprintf("Hue interpolation method \"HueInterpolationMethod.%s hue\" may not be set for rectangular color space %s.", hueMethod, space.Name()), p)
	}

	return NewInterpolationMethod(space, &hueMethod)
}

// String returns a CSS representation of the interpolation method.
//
// A bare space renders as its name; a polar method appends the hue rule plus
// the "hue" keyword (for example "oklch shorter hue").
//
// Matches Dart: InterpolationMethod.toString
func (m InterpolationMethod) String() string {
	if m.Hue == nil {
		return m.Space.Name()
	}
	return fmt.Sprintf("%s %s hue", m.Space.Name(), m.Hue)
}

// hueInterpolationMethodFromValue parses a SassScript value representing a hue
// interpolation method, not ending with "hue".
//
// The value must be an unquoted "shorter", "longer", "increasing", or
// "decreasing" (case-insensitive); anything else is a script error naming the
// argument. (Dart: HueInterpolationMethod._fromValue.)
func hueInterpolationMethodFromValue(val Value, name *string) (HueInterpolationMethod, error) {
	p := name
	if p == nil {
		p = new("value")
	}
	s, err := AssertString(val, name)
	if err != nil {
		return 0, err
	}
	if err := s.AssertUnquoted(); err != nil {
		return 0, sasscommon.NewSassScriptException(err.Error(), p)
	}
	switch strings.ToLower(s.Text) {
	case "shorter":
		return HueInterpolationShorter, nil
	case "longer":
		return HueInterpolationLonger, nil
	case "increasing":
		return HueInterpolationIncreasing, nil
	case "decreasing":
		return HueInterpolationDecreasing, nil
	default:
		valStr, err := val.String()
		if err != nil {
			return 0, err
		}
		return 0, sasscommon.NewSassScriptException(fmt.Sprintf("Unknown hue interpolation method %s.", valStr), p)
	}
}
