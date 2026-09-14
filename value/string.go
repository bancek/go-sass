// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/string.dart

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bancek/go-sass/sasscommon"
)

var emptyQuotedSassString = &SassString{Text: "", HasQuotes: true}
var emptyUnquotedSassString = &SassString{Text: "", HasQuotes: false}

// EmptySassString returns a cached empty string, quoted or unquoted.
//
// Dart exposes this as the SassString.empty factory, which hands out one of
// two shared instances so empty strings never allocate. A nil quotes argument
// selects the quoted singleton; only an explicit false selects unquoted.
//
// Matches Dart: SassString.empty
func EmptySassString(quotes *bool) *SassString {
	if quotes != nil && !*quotes {
		return emptyUnquotedSassString
	}
	return emptyQuotedSassString
}

// SassString is a SassScript string.
//
// Strings can either be quoted or unquoted. Unquoted strings are usually CSS
// identifiers, but they may contain any text.
//
// Values are immutable: build new strings with a struct literal (Dart's
// SassString.new constructor) or reuse the EmptySassString singletons.
type SassString struct {
	// The contents of the string.
	//
	// For quoted strings, this is the semantic content—any escape sequences that
	// were written in the source text are resolved to their Unicode values.
	// For unquoted strings, though, escape sequences are preserved as literal
	// backslashes.
	//
	// This difference allows the evaluator to distinguish identifiers with
	// escapes, such as `url\u28 http://example.com\u29`, from unquoted strings
	// holding characters that aren't valid in identifiers, such as
	// `url(http://example.com)`. As a side effect, `foo` and `f\6F\6F` do not
	// compare equal.
	Text string
	// Whether this string has quotes.
	HasQuotes  bool
	cachedHash *int
}

func (s *SassString) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitString(s) }
func (s *SassString) isValue()                                              {}
func (s *SassString) TryMap() *SassMap                                      { return nil }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Strings are always truthy, including the empty string.
func (s *SassString) IsTruthy() bool           { return true }
func (s *SassString) Separator() ListSeparator { return ListSeparatorUndecided }
func (s *SassString) HasBrackets() bool        { return false }
func (s *SassString) AsList() ([]Value, error) { return []Value{s}, nil }
func (s *SassString) LengthAsList() int        { return 1 }
func (s *SassString) IsBlank() bool            { return !s.HasQuotes && s.Text == "" }

// SassLength returns Sass's notion of the length of this string.
//
// Sass treats strings as a series of Unicode code points while Go treats them
// as a series of bytes. This returns the number of Unicode code points (runes).
// Dart caches this count in a late field; Go recounts the runes on each call.
func (s *SassString) SassLength() int {
	return utf8.RuneCountInString(s.Text)
}

// AssertUnquoted throws an error if this is a quoted string.
//
// If the string came from a function argument, name is the argument name
// (without the `$`) used for error reporting.
func (s *SassString) AssertUnquoted() error {
	if s.HasQuotes {
		ss, err := s.String()
		if err != nil {
			return err
		}
		return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to be an unquoted string.", ss), nil)
	}
	return nil
}

// AssertQuoted throws an error if this is not a quoted string.
//
// If the string came from a function argument, name is the argument name
// (without the `$`) used for error reporting.
func (s *SassString) AssertQuoted() error {
	if !s.HasQuotes {
		ss, err := s.String()
		if err != nil {
			return err
		}
		return sasscommon.NewSassScriptException(fmt.Sprintf("Expected %s to be a quoted string.", ss), nil)
	}
	return nil
}

// Plus concatenates this string with another value.
//
// A string operand appends its raw text; any other value appends its CSS
// rendering. Either way the result keeps the receiver's quoting, so adding to
// a quoted string stays quoted.
//
// Matches Dart: SassString.plus
func (s *SassString) Plus(other Value) (Value, error) {
	if os, ok := other.(*SassString); ok {
		return &SassString{Text: s.Text + os.Text, HasQuotes: s.HasQuotes}, nil
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: s.Text + right, HasQuotes: s.HasQuotes}, nil
}

func (s *SassString) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(s, other) }
func (s *SassString) Minus(other Value) (Value, error)        { return DefaultMinus(s, other) }
func (s *SassString) Times(other Value) (Value, error)        { return DefaultTimes(s, other) }
func (s *SassString) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(s, other) }
func (s *SassString) Modulo(other Value) (Value, error)       { return DefaultModulo(s, other) }
func (s *SassString) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(s, other) }
func (s *SassString) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(s, other)
}
func (s *SassString) LessThan(other Value) (Value, error) { return DefaultLessThan(s, other) }
func (s *SassString) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(s, other)
}
func (s *SassString) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(s) }
func (s *SassString) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(s) }
func (s *SassString) UnaryDivide() (Value, error) { return DefaultUnaryDivide(s) }
func (s *SassString) UnaryNot() (Value, error)    { return DefaultUnaryNot(s) }

// sassIndexToRuneIndex converts a Sass index (1-based, may be negative for
// end-relative) to a 0-based index into the string's code points.
//
// This step is O(1); the O(n) walk from a code-point index to a byte offset
// lives in sassIndexToStringIndex. An out-of-range or zero index raises a
// script error naming the argument when name is set.
//
// Matches Dart: SassString.sassIndexToRuneIndex
func (s *SassString) sassIndexToRuneIndex(sassIndex Value, name *string) (int, error) {
	indexNumber, err := AssertNumber(sassIndex, name)
	if err != nil {
		return 0, err
	}
	intIndex, err := indexNumber.AssertInt(name)
	if err != nil {
		return 0, err
	}
	if intIndex == 0 {
		return 0, sasscommon.NewSassScriptException("String index may not be 0.", name)
	}
	length := s.SassLength()
	if int64Abs(intIndex) > int64(length) {
		idxStr, err := sassIndex.String()
		if err != nil {
			return 0, err
		}
		return 0, sasscommon.NewSassScriptException(fmt.Sprintf("Invalid index %s for a string with %d characters.", idxStr, length), name)
	}
	if intIndex < 0 {
		return length + int(intIndex), nil
	}
	return int(intIndex) - 1, nil
}

// sassIndexToStringIndex converts a Sass index to a byte index into the Go
// string, accounting for multi-byte UTF-8 characters.
//
// Dart converts to UTF-16 code-unit offsets; Go converts to UTF-8 byte offsets
// instead, since Go strings are byte-indexed.
//
// Matches Dart: SassString.sassIndexToStringIndex
func (s *SassString) sassIndexToStringIndex(sassIndex Value, name *string) (int, error) {
	runeIndex, err := s.sassIndexToRuneIndex(sassIndex, name)
	if err != nil {
		return 0, err
	}
	return codepointIndexToCodeUnitIndex(s.Text, runeIndex), nil
}

func int64Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// codepointIndexToCodeUnitIndex converts a code point index to a byte index in
// a UTF-8 string.
//
// Dart's same-named helper in utils.dart targets UTF-16 code units; it is kept
// beside its caller here and adapted to byte offsets.
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

// Reports whether this is an unquoted attr(), if(), or var() call, which may
// be substituted for a custom property value. Quoted strings never qualify;
// matching is case-insensitive behind a length guard for the shortest
// candidate. Dart marks this getter @internal.
//
// Matches Dart: SassString.isSpecialVariable
func (s *SassString) IsSpecialVariable() bool {
	if s.HasQuotes {
		return false
	}
	if len(s.Text) < 6 {
		return false
	}
	lower := strings.ToLower(s.Text)
	return strings.HasPrefix(lower, "attr(") ||
		strings.HasPrefix(lower, "if(") ||
		strings.HasPrefix(lower, "var(")
}

// RealNull returns self, since only the null value collapses to Go nil.
// Matches Dart: SassString.realNull (Value.realNull default)
func (s *SassString) RealNull() Value { return DefaultRealNull(s) }

// Reports whether this is an unquoted string that CSS may treat as a number,
// such as a calc(), clamp(), env(), min(), max(), or var() call. Functions
// that shadow plain CSS functions use this to decide when to pass arguments
// through untouched. Quoted strings never qualify. Dart marks this @internal.
func (s *SassString) IsSpecialNumber() bool {
	if s.HasQuotes {
		return false
	}
	if len(s.Text) < 6 {
		return false
	}
	lower := strings.ToLower(s.Text)
	return strings.HasPrefix(lower, "attr(") ||
		strings.HasPrefix(lower, "calc(") ||
		strings.HasPrefix(lower, "clamp(") ||
		strings.HasPrefix(lower, "env(") ||
		strings.HasPrefix(lower, "if(") ||
		strings.HasPrefix(lower, "min(") ||
		strings.HasPrefix(lower, "max(") ||
		strings.HasPrefix(lower, "var(")
}

// Equals reports whether other is a string with identical text.
//
// Quoting is ignored: a quoted string equals its unquoted twin, matching
// Dart's operator== which compares text only (and HashCode hashes text only).
func (s *SassString) Equals(other Value) bool {
	if os, ok := other.(*SassString); ok {
		return s.Text == os.Text
	}
	return false
}

// HashCode returns text.hashCode (Text only, matches Equal which ignores HasQuotes).
// Matches Dart: SassString.hashCode => text.hashCode (cached).
func (s *SassString) HashCode() int {
	if s.cachedHash != nil {
		return *s.cachedHash
	}
	h := stringHashCode(s.Text)
	s.cachedHash = &h
	return h
}

// ToCssString returns the CSS rendering of this string: quoted strings keep
// their quotes unless quote is false, unquoted strings render as-is. Strings
// always have a CSS form, so this only errors if serialization itself fails.
func (s *SassString) ToCssString(quote bool) (string, error) { return SerializeValue(s, quote) }

// String returns the inspect rendering of this string (Dart toString): quoted
// strings keep their quotes so the result round-trips through evaluation,
// unlike ToCssString with quote=false which emits bare CSS text.
func (s *SassString) String() (string, error) { return SerializeValueInspect(s) }
