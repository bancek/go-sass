// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/argument_list.dart

import (
	"github.com/bancek/go-sass/orderedmap"
)

// SassArgumentList is a SassScript argument list.
//
// An argument list comes from a rest argument. It's distinct from a normal
// SassList in that it may contain a keyword map as well as the positional
// arguments.
//
// Matches Dart: SassArgumentList (keywords: LinkedHashMap)
type SassArgumentList struct {
	list                 SassList
	keywords             *orderedmap.LinkedMap[string, Value]
	wereKeywordsAccessed bool
}

// NewSassArgumentList creates a new SassArgumentList with ordered keywords.
//
// Contents are validated exactly like a plain list (a multi-element rest
// argument needs an explicit separator); keywords keep insertion order in the
// provided map. Dart builds this from a rest parameter's evaluated positionals
// and named arguments.
// Matches Dart: SassArgumentList(rest, evaluated.named, separator)
func NewSassArgumentList(contents []Value, keywords *orderedmap.LinkedMap[string, Value], separator ListSeparator) (*SassArgumentList, error) {
	lst, err := NewSassList(contents, separator, false)
	if err != nil {
		return nil, err
	}
	return &SassArgumentList{
		list:     *lst,
		keywords: keywords,
	}, nil
}

// AcceptVoid dispatches to the visitor's list case: argument lists visit as
// lists, since keywords are invisible to most traversals.
// (Dart: SassArgumentList inherits the list accept path.)
func (a *SassArgumentList) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitList(a)
}

// ToCssString returns the CSS rendering of the positional elements, ignoring
// keywords — the same rendering the embedded list would produce. An empty
// unbracketed argument list has no CSS form and errors.
// (Dart: SassArgumentList inherits the list CSS rendering.)
func (a *SassArgumentList) ToCssString(quote bool) (string, error) { return SerializeValue(a, quote) }

// String returns the inspect rendering of the positional elements (Dart
// toString via the list case): parenthesized unless bracketed, empty, or a
// single comma-separated element.
func (a *SassArgumentList) String() (string, error) { return listString(a) }

func (a *SassArgumentList) isValue() {}

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Argument lists delegate to their positional list and
// are always truthy, like all lists.
func (a *SassArgumentList) IsTruthy() bool           { return a.list.IsTruthy() }
func (a *SassArgumentList) Separator() ListSeparator { return a.list.Separator() }
func (a *SassArgumentList) HasBrackets() bool        { return a.list.HasBrackets() }
func (a *SassArgumentList) AsList() ([]Value, error) { return a.list.AsList() }
func (a *SassArgumentList) LengthAsList() int        { return a.list.LengthAsList() }
func (a *SassArgumentList) IsBlank() bool            { return a.list.IsBlank() }
func (a *SassArgumentList) IsSpecialNumber() bool    { return false }
func (a *SassArgumentList) IsSpecialVariable() bool  { return false }
func (a *SassArgumentList) RealNull() Value          { return DefaultRealNull(a) }
func (a *SassArgumentList) TryMap() *SassMap         { return a.list.TryMap() }
func (a *SassArgumentList) SingleEquals(other Value) (Value, error) {
	return DefaultSingleEquals(a, other)
}
func (a *SassArgumentList) Plus(other Value) (Value, error)  { return DefaultPlus(a, other) }
func (a *SassArgumentList) Minus(other Value) (Value, error) { return DefaultMinus(a, other) }
func (a *SassArgumentList) Times(other Value) (Value, error) { return DefaultTimes(a, other) }
func (a *SassArgumentList) DividedBy(other Value) (Value, error) {
	return DefaultDividedBy(a, other)
}
func (a *SassArgumentList) Modulo(other Value) (Value, error) { return DefaultModulo(a, other) }
func (a *SassArgumentList) GreaterThan(other Value) (Value, error) {
	return DefaultGreaterThan(a, other)
}
func (a *SassArgumentList) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(a, other)
}
func (a *SassArgumentList) LessThan(other Value) (Value, error) {
	return DefaultLessThan(a, other)
}
func (a *SassArgumentList) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(a, other)
}
func (a *SassArgumentList) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(a) }
func (a *SassArgumentList) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(a) }
func (a *SassArgumentList) UnaryDivide() (Value, error) { return DefaultUnaryDivide(a) }
func (a *SassArgumentList) UnaryNot() (Value, error)    { return DefaultUnaryNot(a) }

// Equals reports whether other is a list equal to the positional elements.
// Keywords never participate: Dart inherits SassList.== unchanged, so two
// argument lists with the same positionals but different keywords are equal.
func (a *SassArgumentList) Equals(other Value) bool { return a.list.Equals(other) }

// HashCode delegates to the embedded SassList (no keywords — matches Equal/==).
// Matches Dart: SassArgumentList inherits SassList.hashCode.
func (a *SassArgumentList) HashCode() int { return a.list.HashCode() }

// Keywords returns the keyword arguments in insertion order.
//
// The argument names don't include $. Reading the keywords marks them
// accessed, which tells the caller that passing keywords to a rest argument
// was legitimate — use KeywordsWithoutMarking for a side-effect-free peek.
// Matches Dart: SassArgumentList.keywords getter (returns Map.unmodifiable).
func (a *SassArgumentList) Keywords() *orderedmap.LinkedMap[string, Value] {
	a.wereKeywordsAccessed = true
	return a.keywords
}

// KeywordsWithoutMarking returns the keyword arguments in insertion order
// without marking them as accessed.
//
// Use this when inspecting keywords that the callee may still reject: only a
// read through Keywords records that keywords were legitimately consumed.
// Matches Dart: SassArgumentList.keywordsWithoutMarking.
func (a *SassArgumentList) KeywordsWithoutMarking() *orderedmap.LinkedMap[string, Value] {
	return a.keywords
}

// Reports whether Keywords has been read. The evaluator uses this to decide
// whether passing unexpected keywords to a rest argument is an error: unread
// keywords mean the callee never accepted them. Dart marks this @internal.
//
// Matches Dart: SassArgumentList.wereKeywordsAccessed.
func (a *SassArgumentList) WereKeywordsAccessed() bool {
	return a.wereKeywordsAccessed
}
