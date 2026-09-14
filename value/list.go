// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/list.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// SassList is a SassScript list value.
//
// Dart notes a wish to move to persistent data structures (an RRB vector) or
// plain unmodifiable lists for small lists; Go holds a plain slice instead,
// treating Contents as immutable after construction.
type SassList struct {
	Contents    []Value
	separator   ListSeparator
	hasBrackets bool
	cachedHash  *int
}

// NewSassList creates a new SassList.
// A list with more than one element must have an explicit separator: only
// singletons and empty lists may stay undecided, so list functions can prefer
// another operand's separator when combining lists.
func NewSassList(contents []Value, separator ListSeparator, hasBrackets bool) (*SassList, error) {
	if separator == ListSeparatorUndecided && len(contents) > 1 {
		return nil, &sasscommon.ArgumentError{Message: "A list with more than one element must have an explicit separator."}
	}
	return &SassList{Contents: contents, separator: separator, hasBrackets: hasBrackets}, nil
}

// NewSassListEmpty creates an empty list with the given separator and brackets.
//
// Both arguments are optional: a nil separator defaults to undecided and nil
// brackets defaults to unbracketed, mirroring Dart's defaulted named
// parameters.
//
// Matches Dart: SassList.empty
func NewSassListEmpty(separator *ListSeparator, brackets *bool) *SassList {
	sep := ListSeparatorUndecided
	if separator != nil {
		sep = *separator
	}
	hasBrackets := false
	if brackets != nil {
		hasBrackets = *brackets
	}
	return &SassList{Contents: []Value{}, separator: sep, hasBrackets: hasBrackets}
}

func (l *SassList) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitList(l) }
func (l *SassList) isValue()                                              {}

// TryMap returns this list as a map when that is valid: an empty list counts
// as the empty map, anything else returns nil. It ports Dart's
// SassList.tryMap, which the evaluator uses where map-or-empty-list applies.
func (l *SassList) TryMap() *SassMap {
	if len(l.Contents) == 0 {
		return EmptySassMap()
	}
	return nil
}

// ToCssString returns the CSS rendering of this list's elements with their
// separator and brackets. An empty unbracketed list has no CSS form and
// errors, matching Dart's toCssString contract for `()`.
func (l *SassList) ToCssString(quote bool) (string, error) { return SerializeValue(l, quote) }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Lists are always truthy, even when empty.
func (l *SassList) IsTruthy() bool           { return true }
func (l *SassList) Separator() ListSeparator { return l.separator }
func (l *SassList) HasBrackets() bool        { return l.hasBrackets }
func (l *SassList) AsList() ([]Value, error) { return l.Contents, nil }
func (l *SassList) LengthAsList() int        { return len(l.Contents) }
func (l *SassList) IsBlank() bool {
	if l.hasBrackets {
		return false
	}
	for _, element := range l.Contents {
		if !element.IsBlank() {
			return false
		}
	}
	return true
}
func (l *SassList) IsSpecialNumber() bool                   { return false }
func (l *SassList) IsSpecialVariable() bool                 { return false }
func (l *SassList) RealNull() Value                         { return DefaultRealNull(l) }
func (l *SassList) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(l, other) }
func (l *SassList) Plus(other Value) (Value, error)         { return DefaultPlus(l, other) }
func (l *SassList) Minus(other Value) (Value, error)        { return DefaultMinus(l, other) }
func (l *SassList) Times(other Value) (Value, error)        { return DefaultTimes(l, other) }
func (l *SassList) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(l, other) }
func (l *SassList) Modulo(other Value) (Value, error)       { return DefaultModulo(l, other) }
func (l *SassList) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(l, other) }
func (l *SassList) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(l, other)
}
func (l *SassList) LessThan(other Value) (Value, error) { return DefaultLessThan(l, other) }
func (l *SassList) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(l, other)
}
func (l *SassList) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(l) }
func (l *SassList) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(l) }
func (l *SassList) UnaryDivide() (Value, error) { return DefaultUnaryDivide(l) }
func (l *SassList) UnaryNot() (Value, error)    { return DefaultUnaryNot(l) }

// HashCode returns listHash(asList) — elements only, no separator/brackets.
//
// Equality still checks separator and brackets, but the hash folds just the
// elements (equal lists always hash alike; unequal ones may collide). The hash
// is cached after first computation.
//
// Matches Dart: SassList.hashCode => listHash(asList).
func (l *SassList) HashCode() int {
	if l.cachedHash != nil {
		return *l.cachedHash
	}
	h := listHash(l.Contents)
	l.cachedHash = &h
	return h
}

// Equals reports whether other is a list with the same separator, brackets,
// and element-wise equal contents.
//
// An empty list also equals an empty map (in either direction), since `()`
// doubles as the empty map in Sass. Argument lists compare through their
// embedded list, ignoring keywords.
//
// Matches Dart: SassList.== (plus the empty-map equivalence).
func (l *SassList) Equals(other Value) bool {
	if len(l.Contents) == 0 {
		if _, ok := other.(*SassMap); ok {
			return other.LengthAsList() == 0
		}
	}
	var ol *SassList
	switch o := other.(type) {
	case *SassList:
		ol = o
	case *SassArgumentList:
		ol = &o.list
	default:
		return false
	}
	if l.separator != ol.separator || l.hasBrackets != ol.hasBrackets || len(l.Contents) != len(ol.Contents) {
		return false
	}
	for i := range l.Contents {
		if !l.Contents[i].Equals(ol.Contents[i]) {
			return false
		}
	}
	return true
}

// listString renders a list for inspect output, wrapping the serialized form
// in parentheses to make the list bounds clear. Bracketed, empty, and
// single-element comma-separated lists already delimit themselves, so they
// render bare. Dart documents this rationale on SassList.toString.
func listString(l ListValue) (string, error) {
	s, err := SerializeValueInspect(l)
	if err != nil {
		return "", err
	}
	if l.HasBrackets() ||
		l.LengthAsList() == 0 ||
		(l.LengthAsList() == 1 && l.Separator() == ListSeparatorComma) {
		return s, nil
	}
	return "(" + s + ")", nil
}

// String returns the inspect rendering of this list (Dart toString): the
// serialized elements wrapped in parentheses to mark the list bounds, except
// for bracketed, empty, and single-comma-element lists which delimit
// themselves. An empty unbracketed list renders as `()`.
func (l *SassList) String() (string, error) {
	return listString(l)
}

// ListSeparator is an enum of list separator types.
type ListSeparator int

const (
	// ListSeparatorSpace marks a space-separated list.
	ListSeparatorSpace ListSeparator = iota
	// ListSeparatorComma marks a comma-separated list.
	ListSeparatorComma
	// ListSeparatorSlash marks a slash-separated list.
	ListSeparatorSlash
	// ListSeparatorUndecided marks a list with no separator yet.
	//
	// Singleton and empty lists carry this; list functions prefer another
	// operand's decided separator when combining with them.
	// Matches Dart: ListSeparator.undecided
	ListSeparatorUndecided
)

// String returns the English name of the separator ("space", "comma",
// "slash", or "undecided"), mirroring Dart's ListSeparator.toString.
func (ls ListSeparator) String() string {
	switch ls {
	case ListSeparatorSpace:
		return "space"
	case ListSeparatorComma:
		return "comma"
	case ListSeparatorSlash:
		return "slash"
	case ListSeparatorUndecided:
		return "undecided"
	default:
		return fmt.Sprintf("ListSeparator(%d)", ls)
	}
}

// Separator returns the separator character for CSS output, or nil while the
// separator is still undecided. It ports Dart's ListSeparator.separator,
// which is null for undecided lists.
func (ls ListSeparator) Separator() *string {
	switch ls {
	case ListSeparatorSpace:
		s := " "
		return &s
	case ListSeparatorComma:
		s := ","
		return &s
	case ListSeparatorSlash:
		s := "/"
		return &s
	case ListSeparatorUndecided:
		return nil
	default:
		return nil
	}
}
