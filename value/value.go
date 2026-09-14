// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value.dart

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// Value is the base interface for all SassScript values.
//
// All SassScript values are immutable. New values are built with the concrete
// types' constructors, and untyped values are narrowed to a particular type
// with the Assert* functions below, which report a user-friendly
// SassScriptException when the value has the wrong type.
//
// Matches Dart: Value (lib/src/value.dart)
type Value interface {
	// AcceptVoid dispatches to the visitor's method for this value's type,
	// for effect visitors such as serialization that return no result.
	// Matches Dart: Value.accept
	AcceptVoid(ValueVisitor[struct{}]) (struct{}, error)
	// isValue seals the interface so only types in this package implement it.
	isValue()
	// Equals reports deep value equality, mirroring Dart's operator== which
	// ignores spans and metadata. SassFunction and SassMixin are the
	// exception: they compare by callable identity. Never use == on values.
	Equals(Value) bool
	// HashCode returns a hash consistent with Equals, for use as a map key.
	HashCode() int
	// IsTruthy reports whether the value counts as true in @if and other
	// conditional contexts. Only SassFalse and SassNull are falsy.
	// Matches Dart: Value.isTruthy
	IsTruthy() bool
	// Separator returns this value's separator when used as a list.
	//
	// Every value counts as a list: maps count as lists of pairs and all
	// other values as single-value lists.
	// Matches Dart: Value.separator
	Separator() ListSeparator
	// HasBrackets reports whether this value as a list has brackets.
	//
	// Every value counts as a list: maps count as lists of pairs and all
	// other values as single-value lists.
	// Matches Dart: Value.hasBrackets
	HasBrackets() bool
	// AsList returns this value as a list.
	//
	// Every value counts as a list: maps count as lists of pairs and all
	// other values as single-value lists.
	// Matches Dart: Value.asList
	AsList() ([]Value, error)
	// LengthAsList returns the length of AsList without allocating it, so
	// index bounds can be checked cheaply.
	// Matches Dart: Value.lengthAsList
	LengthAsList() int
	// IsBlank reports whether the value renders as the empty string in CSS.
	// Matches Dart: Value.isBlank
	IsBlank() bool
	// IsSpecialNumber reports whether CSS may treat this value as a number,
	// such as a calc() or var() call. Functions that shadow plain CSS
	// functions use it to decide when to pass arguments through untouched.
	// Matches Dart: Value.isSpecialNumber
	IsSpecialNumber() bool
	// IsSpecialVariable returns whether this is a call to `var()`, which may
	// be substituted in CSS for a custom property value.
	// Matches Dart: Value.isSpecialVariable
	IsSpecialVariable() bool

	// RealNull returns Dart's `null` if this is SassNull, and returns this
	// value otherwise.
	// Matches Dart: Value.realNull
	RealNull() Value

	// ToCssString returns a valid CSS representation of this value.
	//
	// It returns an error when the value can't be represented in plain CSS
	// (maps, functions, mixins, empty unbracketed lists). Use String instead
	// to get a representation even for values that aren't valid CSS. If quote
	// is false, quoted strings are emitted without quotes.
	// Matches Dart: Value.toCssString
	ToCssString(quote bool) (string, error)

	// String returns the inspect representation of this value (Dart's
	// toString): it always succeeds but doesn't reflect the user's output
	// settings. Use ToCssString to convert the value to CSS.
	// Matches Dart: Value.toString
	String() (string, error)

	// SassScript binary operators. Defaults below throw SassScriptException
	// for unsupported operand combinations; concrete types override the
	// combinations they support.

	// SingleEquals is the SassScript `=` operation.
	// The default renders both sides as CSS and joins them with "=" into an
	// unquoted string.
	// Matches Dart: Value.singleEquals
	SingleEquals(other Value) (Value, error)

	// GreaterThan is the SassScript `>` operation.
	// The default throws an Undefined operation error; ordered types such as
	// numbers override it.
	// Matches Dart: Value.greaterThan
	GreaterThan(other Value) (Value, error)

	// GreaterThanOrEquals is the SassScript `>=` operation.
	// The default throws an Undefined operation error; ordered types such as
	// numbers override it.
	// Matches Dart: Value.greaterThanOrEquals
	GreaterThanOrEquals(other Value) (Value, error)

	// LessThan is the SassScript `<` operation.
	// The default throws an Undefined operation error; ordered types such as
	// numbers override it.
	// Matches Dart: Value.lessThan
	LessThan(other Value) (Value, error)

	// LessThanOrEquals is the SassScript `<=` operation.
	// The default throws an Undefined operation error; ordered types such as
	// numbers override it.
	// Matches Dart: Value.lessThanOrEquals
	LessThanOrEquals(other Value) (Value, error)

	// Times is the SassScript `*` operation.
	// The default throws an Undefined operation error; numbers override it.
	// Matches Dart: Value.times
	Times(other Value) (Value, error)

	// Modulo is the SassScript `%` operation.
	// The default throws an Undefined operation error; numbers override it.
	// Matches Dart: Value.modulo
	Modulo(other Value) (Value, error)

	// Plus is the SassScript `+` operation.
	// The default concatenates: a string operand keeps its quotes, a
	// calculation operand is an Undefined operation error, and anything else
	// concatenates both sides' CSS forms into an unquoted string.
	// Matches Dart: Value.plus
	Plus(other Value) (Value, error)

	// Minus is the SassScript `-` operation.
	// A calculation operand is an Undefined operation error; otherwise the
	// default joins both sides' CSS forms around "-" into an unquoted string.
	// Matches Dart: Value.minus
	Minus(other Value) (Value, error)

	// DividedBy is the SassScript `/` operation.
	// The default joins both sides' CSS forms around "/" into an unquoted
	// string; this is slash-separation, never numeric division.
	// Matches Dart: Value.dividedBy
	DividedBy(other Value) (Value, error)

	// Unary operators. The defaults prefix the value's CSS form, so they
	// succeed for any value that is representable in CSS.

	// UnaryPlus is the SassScript unary `+` operation.
	// Matches Dart: Value.unaryPlus
	UnaryPlus() (Value, error)

	// UnaryMinus is the SassScript unary `-` operation.
	// Matches Dart: Value.unaryMinus
	UnaryMinus() (Value, error)

	// UnaryDivide is the SassScript unary `/` operation.
	// Matches Dart: Value.unaryDivide
	UnaryDivide() (Value, error)

	// UnaryNot is the SassScript unary `not` operation.
	// The default returns false; SassBoolean and SassNull override it.
	// Matches Dart: Value.unaryNot
	UnaryNot() (Value, error)

	// TryMap returns this value as a *SassMap if it is one, or nil otherwise.
	// An empty SassList counts as an empty map and reports one here.
	// Matches Dart: Value.tryMap
	TryMap() *SassMap
}

// ValueEquals is the structural equality function for Value map keys: it
// delegates to Equals so keys compare by value rather than by identity.
// Matches Dart: Value.operator== (as used for map keys)
var ValueEquals = func(a, b Value) bool { return a.Equals(b) }

// --- Assertion functions ---
//
// Each Assert* function narrows an untyped Value to a concrete type,
// returning a SassScriptException naming the offending value when it has the
// wrong type. If the value came from a function argument, name is the
// argument name (without the `$`) used for error reporting.
//
// Matches Dart: Value.assert* family

// AssertBoolean returns v as a *SassBoolean, or an error if it isn't one.
//
// Callers that only need truthiness should use IsTruthy instead of requiring
// a literal boolean.
// Matches Dart: Value.assertBoolean
func AssertBoolean(v Value, name *string) (*SassBoolean, error) {
	b, ok := v.(*SassBoolean)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a boolean.", vStr), name)
	}
	return b, nil
}

// AssertNumber returns v as a SassNumber, or an error if it isn't one.
// Matches Dart: Value.assertNumber
func AssertNumber(v Value, name *string) (SassNumber, error) {
	n, ok := v.(SassNumber)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a number.", vStr), name)
	}
	return n, nil
}

// AssertString returns v as a *SassString, or an error if it isn't one.
// Matches Dart: Value.assertString
func AssertString(v Value, name *string) (*SassString, error) {
	s, ok := v.(*SassString)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a string.", vStr), name)
	}
	return s, nil
}

// AssertColor returns v as a *SassColor, or an error if it isn't one.
// Matches Dart: Value.assertColor
func AssertColor(v Value, name *string) (*SassColor, error) {
	c, ok := v.(*SassColor)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a color.", vStr), name)
	}
	return c, nil
}

// AssertList returns v as a *SassList, or an error if it isn't one. A
// *SassArgumentList reports its underlying positional list.
func AssertList(v Value, name *string) (*SassList, error) {
	switch val := v.(type) {
	case *SassList:
		return val, nil
	case *SassArgumentList:
		return &val.list, nil
	}
	vStr, err := v.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a list.", vStr), name)
}

// AssertMap returns v as a *SassMap, or an error if it isn't one. An empty
// SassList counts as an empty map and reports a fresh one, since there is no
// inner map to borrow.
// Matches Dart: Value.assertMap
func AssertMap(v Value, name *string) (*SassMap, error) {
	switch val := v.(type) {
	case *SassMap:
		return val, nil
	case *SassList:
		if len(val.Contents) == 0 {
			return EmptySassMap(), nil
		}
	}
	vStr, err := v.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a map.", vStr), name)
}

// AssertFunction returns v as a *SassFunction, or an error if it isn't a
// function reference.
// Matches Dart: Value.assertFunction
func AssertFunction(v Value, name *string) (*SassFunction, error) {
	f, ok := v.(*SassFunction)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a function reference.", vStr), name)
	}
	return f, nil
}

// AssertMixin returns v as a *SassMixin, or an error if it isn't a mixin
// reference.
// Matches Dart: Value.assertMixin
func AssertMixin(v Value, name *string) (*SassMixin, error) {
	m, ok := v.(*SassMixin)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a mixin reference.", vStr), name)
	}
	return m, nil
}

// AssertCalculation returns v as a *SassCalculation, or an error if it
// isn't one.
// Matches Dart: Value.assertCalculation
func AssertCalculation(v Value, name *string) (*SassCalculation, error) {
	c, ok := v.(*SassCalculation)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a calculation.", vStr), name)
	}
	return c, nil
}

// --- Default operator implementations (standalone helpers) ---
// These carry the base-class behavior from Dart's Value: Go has no inherited
// method dispatch, so each concrete value type forwards its unsupported
// combinations here with self passed explicitly.

// DefaultSingleEquals implements the Value.SingleEquals default: both sides
// render as CSS and are joined with "=" into an unquoted string.
// Matches Dart: Value.singleEquals
func DefaultSingleEquals(self, other Value) (Value, error) {
	left, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: left + "=" + right, HasQuotes: false}, nil
}

// DefaultPlus implements the Value.Plus default: a string operand absorbs
// self's CSS form while keeping its quotes, a calculation operand is an
// Undefined operation error, and anything else concatenates both sides' CSS
// forms into an unquoted string.
// Matches Dart: Value.plus
func DefaultPlus(self, other Value) (Value, error) {
	if s, ok := other.(*SassString); ok {
		left, err := self.ToCssString(true)
		if err != nil {
			return nil, err
		}
		return &SassString{Text: left + s.Text, HasQuotes: s.HasQuotes}, nil
	}
	if _, ok := other.(*SassCalculation); ok {
		selfStr, err := self.String()
		if err != nil {
			return nil, err
		}
		otherStr, err := other.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s + %s\".", selfStr, otherStr), nil)
	}
	left, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: left + right, HasQuotes: false}, nil
}

// DefaultMinus implements the Value.Minus default: a calculation operand is
// an Undefined operation error, otherwise both sides' CSS forms are joined
// around "-" into an unquoted string.
// Matches Dart: Value.minus
func DefaultMinus(self, other Value) (Value, error) {
	if _, ok := other.(*SassCalculation); ok {
		selfStr, err := self.String()
		if err != nil {
			return nil, err
		}
		otherStr, err := other.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s - %s\".", selfStr, otherStr), nil)
	}
	left, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: left + "-" + right, HasQuotes: false}, nil
}

// DefaultTimes implements the Value.Times default, which always throws an
// Undefined operation error; only numbers override it.
// Matches Dart: Value.times
func DefaultTimes(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s * %s\".", selfStr, otherStr), nil)
}

// DefaultDividedBy implements the Value.DividedBy default: both sides' CSS
// forms are joined around "/" into an unquoted string. This is
// slash-separation, never numeric division.
// Matches Dart: Value.dividedBy
func DefaultDividedBy(self, other Value) (Value, error) {
	left, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	right, err := other.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: left + "/" + right, HasQuotes: false}, nil
}

// DefaultModulo implements the Value.Modulo default, which always throws an
// Undefined operation error; only numbers override it.
// Matches Dart: Value.modulo
func DefaultModulo(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s %% %s\".", selfStr, otherStr), nil)
}

// DefaultGreaterThan implements the Value.GreaterThan default, which always
// throws an Undefined operation error; ordered types such as numbers
// override it.
// Matches Dart: Value.greaterThan
func DefaultGreaterThan(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s > %s\".", selfStr, otherStr), nil)
}

// DefaultGreaterThanOrEquals implements the Value.GreaterThanOrEquals
// default, which always throws an Undefined operation error; ordered types
// such as numbers override it.
// Matches Dart: Value.greaterThanOrEquals
func DefaultGreaterThanOrEquals(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s >= %s\".", selfStr, otherStr), nil)
}

// DefaultLessThan implements the Value.LessThan default, which always
// throws an Undefined operation error; ordered types such as numbers
// override it.
// Matches Dart: Value.lessThan
func DefaultLessThan(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s < %s\".", selfStr, otherStr), nil)
}

// DefaultLessThanOrEquals implements the Value.LessThanOrEquals default,
// which always throws an Undefined operation error; ordered types such as
// numbers override it.
// Matches Dart: Value.lessThanOrEquals
func DefaultLessThanOrEquals(self, other Value) (Value, error) {
	selfStr, err := self.String()
	if err != nil {
		return nil, err
	}
	otherStr, err := other.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Undefined operation \"%s <= %s\".", selfStr, otherStr), nil)
}

// DefaultUnaryPlus implements the Value.UnaryPlus default: the value's CSS
// form prefixed with "+".
// Matches Dart: Value.unaryPlus
func DefaultUnaryPlus(self Value) (Value, error) {
	s, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: "+" + s, HasQuotes: false}, nil
}

// DefaultUnaryMinus implements the Value.UnaryMinus default: the value's CSS
// form prefixed with "-".
// Matches Dart: Value.unaryMinus
func DefaultUnaryMinus(self Value) (Value, error) {
	s, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: "-" + s, HasQuotes: false}, nil
}

// DefaultUnaryDivide implements the Value.UnaryDivide default: the value's
// CSS form prefixed with "/".
// Matches Dart: Value.unaryDivide
func DefaultUnaryDivide(self Value) (Value, error) {
	s, err := self.ToCssString(true)
	if err != nil {
		return nil, err
	}
	return &SassString{Text: "/" + s, HasQuotes: false}, nil
}

// DefaultUnaryNot implements the Value.UnaryNot default, which returns false
// for every value except the falsy ones (SassBoolean and SassNull override
// it).
// Matches Dart: Value.unaryNot
func DefaultUnaryNot(self Value) (Value, error) {
	return SassFalse, nil
}

// --- Helper functions ---

// DefaultRealNull implements the Value.RealNull default: every value reports
// itself, and only SassNull overrides this to report Go nil (Dart null).
// Matches Dart: Value.realNull
func DefaultRealNull(self Value) Value { return self }

// SassIndexToListIndex converts a one-based Sass index (negative from the
// end) into a zero-based Go index into v's AsList.
//
// A number carrying units warns through warn first, pointing at the
// unitSuggestion fix. Index 0 and out-of-range indexes throw a
// SassScriptException; name is the originating argument name (without the
// `$`) used for error reporting.
//
// Matches Dart: Value.sassIndexToListIndex
func SassIndexToListIndex(v Value, index Value, name string, warn func(string, *deprecation.Deprecation) error) (int, error) {
	indexNumber, err := AssertNumber(index, &name)
	if err != nil {
		return 0, err
	}
	if indexNumber.HasUnits() && warn != nil {
		// An empty name still warns under the generic "index" label so the
		// suggestion snippet always has a variable to hang onto.
		suggestionName := name
		if suggestionName == "" {
			suggestionName = "index"
		}
		if err := warn(fmt.Sprintf(
			"$%s: Passing a number with unit %s is deprecated.\n\n"+
				"To preserve current behavior: %s\n\n"+
				"More info: https://sass-lang.com/d/function-units",
			name, indexNumber.UnitString(), indexNumber.UnitSuggestion(suggestionName, nil)),
			deprecation.FunctionUnits); err != nil {
			return 0, err
		}
	}
	intIndex, err := indexNumber.AssertInt(&name)
	if err != nil {
		return 0, err
	}
	if intIndex == 0 {
		return 0, sasscommon.NewSassScriptException("List index may not be 0.", &name)
	}
	length := v.LengthAsList()
	// LengthAsList (not AsList) bounds-checks without allocating the list.
	absIndex := int(intIndex)
	if absIndex < 0 {
		absIndex = -absIndex
	}
	if absIndex > length {
		indexStr, err := index.String()
		if err != nil {
			return 0, err
		}
		return 0, sasscommon.NewSassScriptException(fmt.Sprintf("Invalid index %s for a list with %d elements.", indexStr, length), &name)
	}
	if intIndex < 0 {
		return length + int(intIndex), nil
	}
	return int(intIndex) - 1, nil
}

// WithListContents returns a new SassList holding contents that defaults to
// v's separator and brackets unless opts overrides them.
// Matches Dart: Value.withListContents
func WithListContents(v Value, contents []Value, opts *ListOptions) Value {
	separator := v.Separator()
	hasBrackets := v.HasBrackets()
	if opts != nil {
		if opts.Separator != nil {
			separator = *opts.Separator
		}
		if opts.Brackets != nil {
			hasBrackets = *opts.Brackets
		}
	}
	return &SassList{
		Contents:    contents,
		separator:   separator,
		hasBrackets: hasBrackets,
	}
}

// ListOptions carries optional separator/bracket overrides for
// WithListContents. A nil field keeps the source value's setting.
type ListOptions struct {
	Separator *ListSeparator
	Brackets  *bool
}

// ErrorMessage builds a "$name: <value> is not <description>." assertion
// message. It is Go-only glue sharing one format across the value package's
// typed error helpers.
func ErrorMessage(name string, value Value, description string) (string, error) {
	vStr, err := value.String()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("$%s: %s is not %s.", name, vStr, description), nil
}

// AssertCommonListStyle returns v's list contents when v is a plain-CSS
// expression list: space-separated (or slash-separated too when allowSlash is
// true) and unbracketed. Anything else throws a SassScriptException spelling
// out the expected shape; name is the originating argument name (without the
// `$`), empty when there is none.
// Matches Dart: Value.assertCommonListStyle
func AssertCommonListStyle(v Value, name string, allowSlash bool) ([]Value, error) {
	invalidSeparator := v.Separator() == ListSeparatorComma ||
		(!allowSlash && v.Separator() == ListSeparatorSlash)
	if !invalidSeparator && !v.HasBrackets() {
		return v.AsList()
	}

	var buf strings.Builder
	// The message is assembled piece by piece so each offending aspect
	// (brackets, separator) names itself, mirroring Dart's buffer cascade.
	buf.WriteString("Expected")
	if v.HasBrackets() {
		buf.WriteString(" an unbracketed")
	}
	if invalidSeparator {
		if v.HasBrackets() {
			buf.WriteString(",")
		} else {
			buf.WriteString(" a")
		}
		buf.WriteString(" space-")
		if allowSlash {
			buf.WriteString(" or slash-")
		}
		buf.WriteString("separated")
	}
	buf.WriteString(" list, was ")
	s, err := v.String()
	if err != nil {
		return nil, err
	}
	buf.WriteString(s)
	if name != "" {
		return nil, sasscommon.NewSassScriptException(buf.String(), &name)
	}
	return nil, errors.New(buf.String())
}

// SelectorString converts a selector-parse()-style input into a string that
// can be parsed: a string, a list of strings, or a list of lists of strings.
// Anything else (including slash-separated lists) throws a
// SassScriptException; name is the originating argument name (without the
// `$`) used for error reporting.
// Matches Dart: Value._selectorString
func SelectorString(v Value, name string) (string, error) {
	s := selectorStringOrNil(v)
	if s != nil {
		return *s, nil
	}
	vStr, err := v.String()
	if err != nil {
		return "", err
	}
	return "", sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a valid selector: it must be a string,\na list of strings, or a list of lists of strings.", vStr), &name)
}

// AssertSelector parses v as a selector list, in the same manner as the
// selector-parse() function. It throws a SassScriptException when v isn't a
// selector-shaped value or when parsing fails; allowParent permits parent
// selectors, and name is the originating argument name (without the `$`).
// A parse failure is rethrown as a script error chained onto the original,
// with the "Error: " prefix stripped.
// Matches Dart: SassApiValue.assertSelector
func AssertSelector(v Value, name string, allowParent bool) (*SelectorList, error) {
	s, err := SelectorString(v, name)
	if err != nil {
		return nil, err
	}
	parser := NewSelectorParser(s, nil, nil, &SelectorParserOptions{AllowParent: &allowParent})
	result, err := parser.Parse()
	if err != nil {
		// A parse failure surfaces as a script error chained onto the parse
		// error for its trace. Dart's colorize TODO is dropped: Go renders
		// the stripped message without terminal colors.
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			formatMsg := sfe.Message
			formatArgName := ""
			if name != "" {
				formatArgName = name
			}
			return nil, sasscommon.ThrowWithTrace(
				sasscommon.NewSassScriptException(formatMsg, &formatArgName),
				err,
			)
		}
		return nil, err
	}
	return result, nil
}

// AssertSimpleSelector parses v as a simple selector, in the same manner as
// the selector-parse() function. It throws a SassScriptException when v
// isn't a selector-shaped value or when parsing fails; allowParent permits
// parent selectors, and name is the originating argument name (without the
// `$`). A parse failure is rethrown as a script error chained onto the
// original, with the "Error: " prefix stripped.
// Matches Dart: SassApiValue.assertSimpleSelector
func AssertSimpleSelector(v Value, name string, allowParent bool) (SimpleSelector, error) {
	s, err := SelectorString(v, name)
	if err != nil {
		return nil, err
	}
	parser := NewSelectorParser(s, nil, nil, &SelectorParserOptions{AllowParent: &allowParent})
	result, err := parser.ParseSimpleSelector()
	if err != nil {
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			formatMsg := sfe.Message
			formatArgName := ""
			if name != "" {
				formatArgName = name
			}
			return nil, sasscommon.ThrowWithTrace(
				sasscommon.NewSassScriptException(formatMsg, &formatArgName),
				err,
			)
		}
		return nil, err
	}
	return result, nil
}

// AssertCompoundSelector parses v as a compound selector, in the same manner
// as the selector-parse() function. It throws a SassScriptException when v
// isn't a selector-shaped value or when parsing fails; allowParent permits
// parent selectors, and name is the originating argument name (without the
// `$`). A parse failure is rethrown as a script error chained onto the
// original, with the "Error: " prefix stripped.
// Matches Dart: SassApiValue.assertCompoundSelector
func AssertCompoundSelector(v Value, name string, allowParent bool) (*CompoundSelector, error) {
	s, err := SelectorString(v, name)
	if err != nil {
		return nil, err
	}
	parser := NewSelectorParser(s, nil, nil, &SelectorParserOptions{AllowParent: &allowParent})
	result, err := parser.ParseCompoundSelector()
	if err != nil {
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			formatMsg := sfe.Message
			formatArgName := ""
			if name != "" {
				formatArgName = name
			}
			return nil, sasscommon.ThrowWithTrace(
				sasscommon.NewSassScriptException(formatMsg, &formatArgName),
				err,
			)
		}
		return nil, err
	}
	return result, nil
}

// AssertComplexSelector parses v as a complex selector, in the same manner
// as the selector-parse() function. It throws a SassScriptException when v
// isn't a selector-shaped value or when parsing fails; allowParent permits
// parent selectors, and name is the originating argument name (without the
// `$`). A parse failure is rethrown as a script error chained onto the
// original, with the "Error: " prefix stripped.
// Matches Dart: SassApiValue.assertComplexSelector
func AssertComplexSelector(v Value, name string, allowParent bool) (*ComplexSelector, error) {
	s, err := SelectorString(v, name)
	if err != nil {
		return nil, err
	}
	parser := NewSelectorParser(s, nil, nil, &SelectorParserOptions{AllowParent: &allowParent})
	result, err := parser.ParseComplexSelector()
	if err != nil {
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			formatMsg := sfe.Message
			formatArgName := ""
			if name != "" {
				formatArgName = name
			}
			return nil, sasscommon.ThrowWithTrace(
				sasscommon.NewSassScriptException(formatMsg, &formatArgName),
				err,
			)
		}
		return nil, err
	}
	return result, nil
}

// selectorStringOrNil converts a selector-parse()-style input into a string
// that can be parsed, returning nil when v isn't a selector-shaped value:
// a string, a comma-separated list of strings and space-separated lists, or
// a space-separated list of strings. Empty and slash-separated lists report
// nil, as does any nesting that mixes in non-string values.
// Matches Dart: Value._selectorStringOrNull
func selectorStringOrNil(v Value) *string {
	switch val := v.(type) {
	case *SassString:
		return &val.Text
	case *SassList:
		if len(val.Contents) == 0 {
			return nil
		}
		switch val.separator {
		case ListSeparatorComma:
			var result []string
			for _, complex := range val.Contents {
				switch c := complex.(type) {
				case *SassString:
					result = append(result, c.Text)
				case *SassList:
					if c.separator != ListSeparatorSpace {
						return nil
					}
					s := selectorStringOrNil(c)
					if s == nil {
						return nil
					}
					result = append(result, *s)
				default:
					return nil
				}
			}
			r := strings.Join(result, ", ")
			return &r
		case ListSeparatorSlash:
			return nil
		default:
			var result []string
			for _, compound := range val.Contents {
				s, ok := compound.(*SassString)
				if !ok {
					return nil
				}
				result = append(result, s.Text)
			}
			r := strings.Join(result, " ")
			return &r
		}
	default:
		return nil
	}
}

// SprintAny renders an arbitrary value for error messages and debugging. It
// prefers the fallible String() (string, error) form with error propagation,
// then the infallible String() string form, and falls back to fmt.Sprint.
// It is Go-only glue with no Dart counterpart.
func SprintAny(obj any) (string, error) {
	if s, ok := obj.(interface{ String() (string, error) }); ok {
		return s.String()
	}
	if s, ok := obj.(interface{ String() string }); ok {
		return s.String(), nil
	}
	return fmt.Sprint(obj), nil
}
