// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/string.dart

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalStringFunctions returns the deprecated global aliases of the
// sass:string functions. Each entry wraps the module function with a
// deprecation warning (the wrapper names the original module function — see
// docs/ref/functions.md for the ordering rule); the length/insert/index/slice
// globals additionally rename to their str-* spellings.
// Matches Dart: global (string.dart).
func GlobalStringFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		unquoteFunction().WithDeprecationWarning("string", nil),
		quoteFunction().WithDeprecationWarning("string", nil),
		toUpperCaseFunction().WithDeprecationWarning("string", nil),
		toLowerCaseFunction().WithDeprecationWarning("string", nil),
		uniqueIDFunction().WithDeprecationWarning("string", nil),
		strLengthFunctionGlobal().WithDeprecationWarning("string", new("length")),
		strInsertFunctionGlobal().WithDeprecationWarning("string", new("insert")),
		strIndexFunctionGlobal().WithDeprecationWarning("string", new("index")),
		strSliceFunctionGlobal().WithDeprecationWarning("string", new("slice")),
	}
}

// StringModule returns the sass:string built-in module: unquote, quote, the
// case converters, unique-id, length, insert, index, slice, and the
// module-only split.
// Matches Dart: module (string.dart).
func StringModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		unquoteFunction(),
		quoteFunction(),
		toUpperCaseFunction(),
		toLowerCaseFunction(),
		uniqueIDFunction(),
		strLengthFunction(),
		strInsertFunction(),
		strIndexFunction(),
		strSliceFunction(),
		splitFunction(),
	}
	return sassmodule.NewBuiltInModule("string", fns, nil, nil)
}

// ---- Shared function implementations ----
//
// The constructors below port Dart's bare `_function(...)` closures, which
// carry no per-function docs; each note names the Sass signature and any
// behavior Dart documents at the call site. Every callable registers under
// the sass:string URL, porting Dart's _function URL helper. Helpers shared by
// the global and module spellings live here so both spellings stay identical.

// unquoteImpl drops quotes from s, returning s unchanged when already
// unquoted.
func unquoteImpl(s *value.SassString) *value.SassString {
	if !s.HasQuotes {
		return s
	}
	return &value.SassString{Text: s.Text, HasQuotes: false}
}

// quoteImpl adds quotes to s, returning s unchanged when already quoted.
func quoteImpl(s *value.SassString) *value.SassString {
	if s.HasQuotes {
		return s
	}
	return &value.SassString{Text: s.Text, HasQuotes: true}
}

// toUpperImpl uppercases s while preserving its quotedness.
func toUpperImpl(s *value.SassString) *value.SassString {
	return &value.SassString{Text: sassToUpper(s.Text), HasQuotes: s.HasQuotes}
}

// toLowerImpl lowercases s while preserving its quotedness.
func toLowerImpl(s *value.SassString) *value.SassString {
	return &value.SassString{Text: sassToLower(s.Text), HasQuotes: s.HasQuotes}
}

// sassToUpper folds ASCII lowercase to uppercase rune by rune, matching
// Dart's per-code-unit loop over the string.
func sassToUpper(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if r >= 'a' && r <= 'z' {
			runes[i] = r & ^0x20
		}
	}
	return string(runes)
}

// sassToLower folds ASCII uppercase to lowercase rune by rune, matching
// Dart's per-code-unit loop over the string.
func sassToLower(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			runes[i] = r | 0x20
		}
	}
	return string(runes)
}

// ---- Global-only functions ----

// unquoteFunction implements string.unquote($string).
func unquoteFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("unquote", "$string", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		s, err := value.AssertString(args[0], new("string"))
		if err != nil {
			return nil, err
		}
		return unquoteImpl(s), nil
	})
}

// quoteFunction implements string.quote($string).
func quoteFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("quote", "$string", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		s, err := value.AssertString(args[0], new("string"))
		if err != nil {
			return nil, err
		}
		return quoteImpl(s), nil
	})
}

// toUpperCaseFunction implements string.to-upper-case($string).
func toUpperCaseFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("to-upper-case", "$string", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		s, err := value.AssertString(args[0], new("string"))
		if err != nil {
			return nil, err
		}
		return toUpperImpl(s), nil
	})
}

// toLowerCaseFunction implements string.to-lower-case($string).
func toLowerCaseFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("to-lower-case", "$string", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		s, err := value.AssertString(args[0], new("string"))
		if err != nil {
			return nil, err
		}
		return toLowerImpl(s), nil
	})
}

// uniqueIDFunction implements string.unique-id(): an unquoted "u" followed by
// a zero-padded base-36 counter. The counter advances by a random step so
// successive IDs are hard to guess, wraps within the 36^6 ID space, and uses
// base 36 (alphabet plus digits) with a leading "u" so the result is always a
// valid identifier. Ports Dart's _uniqueId with its _previousUniqueId state.
func uniqueIDFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("unique-id", "", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		prevUniqueID += stringRandom.Int63n(36) + 1
		maxVal := int64(36 * 36 * 36 * 36 * 36 * 36)
		if prevUniqueID > maxVal {
			prevUniqueID %= maxVal
		}
		// Use base-36 encoding and zero-pad to 6 chars, matching Dart's
		// toRadixString(36).padLeft(6, '0')
		return &value.SassString{
			Text:      "u" + padLeft(strconv.FormatInt(prevUniqueID, 36), 6, '0'),
			HasQuotes: false,
		}, nil
	})
}

// padLeft is Go-only glue for the base-36 zero-padding above (Dart chains
// toRadixString with padLeft).
func padLeft(s string, length int, padChar byte) string {
	if len(s) >= length {
		return s
	}
	result := make([]byte, length)
	for i := 0; i < length-len(s); i++ {
		result[i] = padChar
	}
	copy(result[length-len(s):], s)
	return string(result)
}

// ---- str-length / length ----

// strLengthFunctionGlobal implements the deprecated global str-length($string)
// spelling over strLengthImpl.
func strLengthFunctionGlobal() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("str-length", "$string", "sass:string", strLengthImpl)
}

// strLengthFunction implements string.length($string): the rune count.
func strLengthFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("length", "$string", "sass:string", strLengthImpl)
}

// strLengthImpl asserts $string and returns its Sass (codepoint) length as a
// unitless number.
func strLengthImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	s, err := value.AssertString(args[0], new("string"))
	if err != nil {
		return nil, err
	}
	return value.NewUnitlessNumber(float64(s.SassLength())), nil
}

// ---- str-insert / insert ----

// strInsertFunctionGlobal implements the deprecated global str-insert($string,
// $insert, $index) spelling over strInsertImpl.
func strInsertFunctionGlobal() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("str-insert", "$string, $insert, $index", "sass:string", strInsertImpl)
}

// strInsertFunction implements string.insert($string, $insert, $index).
func strInsertFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("insert", "$string, $insert, $index", "sass:string", strInsertImpl)
}

// strInsertImpl splices $insert into $string at the Sass index, keeping the
// original quotedness. Negative indexes have unusual semantics: the insert is
// guaranteed to sit at $index in the result, so a negative $index inserts
// *after* that position — hence the +2 (+1 because negative indexes count
// from -1, another +1 for insert-after) clamped at zero.
func strInsertImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	s, err := value.AssertString(args[0], new("string"))
	if err != nil {
		return nil, err
	}
	insert, err := value.AssertString(args[1], new("insert"))
	if err != nil {
		return nil, err
	}
	index, err := value.AssertNumber(args[2], new("index"))
	if err != nil {
		return nil, err
	}
	if err := index.AssertNoUnits(new("index")); err != nil {
		return nil, err
	}
	indexInt64, err := index.AssertInt(new("index"))
	if err != nil {
		return nil, err
	}
	indexInt := int(indexInt64)

	lengthInCodepoints := s.SassLength()
	if indexInt < 0 {
		indexInt = maxInt(lengthInCodepoints+indexInt+2, 0)
	}

	codepointIdx := codepointForIndex(indexInt, lengthInCodepoints)
	codeUnitIdx := codepointIndexToCodeUnitIndex(s.Text, codepointIdx)

	result := s.Text[:codeUnitIdx] + insert.Text + s.Text[codeUnitIdx:]
	return &value.SassString{Text: result, HasQuotes: s.HasQuotes}, nil
}

// ---- str-index / index ----

// strIndexFunctionGlobal implements the deprecated global str-index($string,
// $substring) spelling over strIndexImpl.
func strIndexFunctionGlobal() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("str-index", "$string, $substring", "sass:string", strIndexImpl)
}

// strIndexFunction implements string.index($string, $substring).
func strIndexFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("index", "$string, $substring", "sass:string", strIndexImpl)
}

// strIndexImpl finds $substring with a code-unit search, then converts the
// hit to a 1-based codepoint index; a miss returns null.
func strIndexImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	s, err := value.AssertString(args[0], new("string"))
	if err != nil {
		return nil, err
	}
	substring, err := value.AssertString(args[1], new("substring"))
	if err != nil {
		return nil, err
	}

	codeUnitIndex := strings.Index(s.Text, substring.Text)
	if codeUnitIndex == -1 {
		return value.Null, nil
	}
	codepointIdx := codeUnitIndexToCodepointIndex(s.Text, codeUnitIndex)
	return value.NewUnitlessNumber(float64(codepointIdx + 1)), nil
}

// ---- str-slice / slice ----

// strSliceFunctionGlobal implements the deprecated global
// str-slice($string, $start-at, $end-at: -1) spelling over strSliceImpl.
func strSliceFunctionGlobal() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("str-slice", "$string, $start-at, $end-at: -1", "sass:string", strSliceImpl)
}

// strSliceFunction implements string.slice($string, $start-at, $end-at: -1).
func strSliceFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("slice", "$string, $start-at, $end-at: -1", "sass:string", strSliceImpl)
}

// strSliceImpl extracts the inclusive [$start-at, $end-at] codepoint range,
// keeping the original quotedness. An $end-at of 0 always yields the empty
// string; an end pointing past the final codepoint steps back one, and an end
// before the start likewise yields the empty string.
func strSliceImpl(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	s, err := value.AssertString(args[0], new("string"))
	if err != nil {
		return nil, err
	}
	start, err := value.AssertNumber(args[1], new("start-at"))
	if err != nil {
		return nil, err
	}
	end, err := value.AssertNumber(args[2], new("end-at"))
	if err != nil {
		return nil, err
	}
	if err := start.AssertNoUnits(new("start-at")); err != nil {
		return nil, err
	}
	if err := end.AssertNoUnits(new("end-at")); err != nil {
		return nil, err
	}

	lengthInCodepoints := s.SassLength()

	// No matter what the start index is, an end index of 0 will produce an
	// empty string.
	endInt64, err := end.AssertInt(nil)
	if err != nil {
		return nil, err
	}
	endInt := int(endInt64)

	if endInt == 0 {
		return &value.SassString{Text: "", HasQuotes: s.HasQuotes}, nil
	}

	startInt64, err := start.AssertInt(nil)
	if err != nil {
		return nil, err
	}
	startCodepoint := codepointForIndex(int(startInt64), lengthInCodepoints)
	endCodepoint := codepointForIndexAllowNegative(endInt, lengthInCodepoints)
	if endCodepoint == lengthInCodepoints {
		endCodepoint--
	}
	if endCodepoint < startCodepoint {
		return &value.SassString{Text: "", HasQuotes: s.HasQuotes}, nil
	}

	startByte := codepointIndexToCodeUnitIndex(s.Text, startCodepoint)
	endByte := codepointIndexToCodeUnitIndex(s.Text, endCodepoint+1)
	result := s.Text[startByte:endByte]
	return &value.SassString{Text: result, HasQuotes: s.HasQuotes}, nil
}

// ---- split (module only) ----

// splitFunction implements the module-only string.split($string, $separator,
// $limit: null): a bracketed comma-separated list of chunks keeping the
// input's quotedness. An empty input splits to the empty list, an empty
// separator splits into per-rune chunks, and $limit caps the number of
// separators consumed (so at most $limit cuts) with smaller limits rejected.
func splitFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("split", "$string, $separator, $limit: null", "sass:string", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		s, err := value.AssertString(args[0], new("string"))
		if err != nil {
			return nil, err
		}
		separator, err := value.AssertString(args[1], new("separator"))
		if err != nil {
			return nil, err
		}

		var limitVal int64
		hasLimit := false
		if args[2] != value.Null {
			limitNum, err := value.AssertNumber(args[2], new("limit"))
			if err != nil {
				return nil, err
			}
			limitVal, err = limitNum.AssertInt(new("limit"))
			if err != nil {
				return nil, err
			}
			if limitVal < 1 {
				return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Must be 1 or greater, was %d.", limitVal), new("limit"))
			}
			hasLimit = true
		}

		if len(s.Text) == 0 {
			list, err := value.NewSassList(nil, value.ListSeparatorComma, true)
			if err != nil {
				return nil, err
			}
			return list, nil
		}

		if len(separator.Text) == 0 {
			runes := []rune(s.Text)
			chunks := make([]value.Value, len(runes))
			for i, r := range runes {
				chunks[i] = &value.SassString{Text: string(r), HasQuotes: s.HasQuotes}
			}
			list, err := value.NewSassList(chunks, value.ListSeparatorComma, true)
			if err != nil {
				return nil, err
			}
			return list, nil
		}

		var chunks []string
		chunkCount := 0
		lastEnd := 0
		for {
			matchIndex := strings.Index(s.Text[lastEnd:], separator.Text)
			if matchIndex == -1 {
				break
			}
			matchIndex += lastEnd
			chunks = append(chunks, s.Text[lastEnd:matchIndex])
			lastEnd = matchIndex + len(separator.Text)
			chunkCount++
			if hasLimit && chunkCount == int(limitVal) {
				break
			}
		}
		chunks = append(chunks, s.Text[lastEnd:])

		result := make([]value.Value, len(chunks))
		for i, chunk := range chunks {
			result[i] = &value.SassString{Text: chunk, HasQuotes: s.HasQuotes}
		}
		list, err := value.NewSassList(result, value.ListSeparatorComma, true)
		if err != nil {
			return nil, err
		}
		return list, nil
	})
}

// ---- Codepoint helpers ----

// codepointForIndex converts a 1-based Sass index (negative counts back from
// the end) to a codepoint offset into a string of lengthInCodepoints
// codepoints. A negative index reaching past the start clamps to 0.
// Ports Dart's _codepointForIndex with allowNegative false.
func codepointForIndex(index int, lengthInCodepoints int) int {
	if index == 0 {
		return 0
	}
	if index > 0 {
		return minInt(index-1, lengthInCodepoints)
	}
	result := lengthInCodepoints + index
	if result < 0 {
		return 0
	}
	return result
}

// codepointForIndexAllowNegative is codepointForIndex for slice end indexes:
// a negative index reaching past the start stays negative so the caller can
// detect the empty range. Ports Dart's _codepointForIndex with allowNegative
// true.
func codepointForIndexAllowNegative(index int, lengthInCodepoints int) int {
	if index == 0 {
		return 0
	}
	if index > 0 {
		return minInt(index-1, lengthInCodepoints)
	}
	return lengthInCodepoints + index
}

// codepointIndexToCodeUnitIndex walks runes to translate a codepoint offset
// to a byte offset (Dart's codepointIndexToCodeUnitIndex from utils).
func codepointIndexToCodeUnitIndex(s string, codepointIndex int) int {
	if codepointIndex <= 0 {
		return 0
	}
	byteIndex := 0
	i := 0
	for byteIndex < len(s) && i < codepointIndex {
		_, size := utf8.DecodeRuneInString(s[byteIndex:])
		byteIndex += size
		i++
	}
	return byteIndex
}

// codeUnitIndexToCodepointIndex counts the runes before a byte offset,
// translating a code-unit hit back to codepoints (Dart's
// codeUnitIndexToCodepointIndex from utils).
func codeUnitIndexToCodepointIndex(s string, codeUnitIndex int) int {
	if codeUnitIndex <= 0 {
		return 0
	}
	s = s[:codeUnitIndex]
	return utf8.RuneCountInString(s)
}

// minInt is Go-only glue for the index clamping above.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt is Go-only glue for the negative-index clamping above.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
