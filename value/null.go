// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/null.dart

// Null is the SassScript null value.
//
// Dart exposes this as the const sassNull value backed by an unconstructible
// private class; Go uses a shared singleton pointer instead. Always use this
// rather than constructing SassNull directly.
var Null = &SassNull{}

// SassNull represents a SassScript null value.
//
// This should not be constructed directly; use the [Null] singleton instead.
type SassNull struct{}

func (n *SassNull) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitNull() }
func (n *SassNull) isValue()                                              {}
func (n *SassNull) TryMap() *SassMap                                      { return nil }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Null is one of only two falsy values (the other is
// false), so this always returns false.
// Matches Dart: SassNull.isTruthy => false.
func (n *SassNull) IsTruthy() bool           { return false }
func (n *SassNull) Separator() ListSeparator { return ListSeparatorUndecided }
func (n *SassNull) HasBrackets() bool        { return false }
func (n *SassNull) AsList() ([]Value, error) { return []Value{n}, nil }
func (n *SassNull) LengthAsList() int        { return 1 }
func (n *SassNull) IsBlank() bool            { return true }
func (n *SassNull) IsSpecialNumber() bool    { return false }
func (n *SassNull) IsSpecialVariable() bool  { return false }
func (n *SassNull) RealNull() Value          { return nil }

// Equals reports whether other is also null. Null equals only null —
// unlike the empty list/map equivalence, there is no second value that
// compares equal to it.
//
// Matches Dart: SassNull has no custom ==, so const identity applies.
func (n *SassNull) Equals(other Value) bool {
	_, ok := other.(*SassNull)
	return ok
}

// HashCode returns the null hash. Every null is the shared singleton, so a
// constant hash always agrees with Equals.
//
// Matches Dart: SassNull inherits the constant Object hashCode.
func (n *SassNull) HashCode() int { return 0 }

// ToCssString returns the CSS rendering of null, which is the empty string:
// null silently vanishes from output. (Dart: Value.toCssString via the
// serializer's null case.)
func (n *SassNull) ToCssString(quote bool) (string, error) { return SerializeValue(n, quote) }

// String returns the inspect rendering of null (Dart toString): the text
// "null", which is how null appears in error messages and debug output.
func (n *SassNull) String() (string, error) { return SerializeValueInspect(n) }

func (n *SassNull) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(n, other) }
func (n *SassNull) Plus(other Value) (Value, error)         { return DefaultPlus(n, other) }
func (n *SassNull) Minus(other Value) (Value, error)        { return DefaultMinus(n, other) }
func (n *SassNull) Times(other Value) (Value, error)        { return DefaultTimes(n, other) }
func (n *SassNull) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(n, other) }
func (n *SassNull) Modulo(other Value) (Value, error)       { return DefaultModulo(n, other) }
func (n *SassNull) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(n, other) }
func (n *SassNull) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(n, other)
}
func (n *SassNull) LessThan(other Value) (Value, error) { return DefaultLessThan(n, other) }
func (n *SassNull) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(n, other)
}
func (n *SassNull) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(n) }
func (n *SassNull) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(n) }
func (n *SassNull) UnaryDivide() (Value, error) { return DefaultUnaryDivide(n) }

// UnaryNot implements SassScript `not`: null is falsy, so `not null` is true.
// Matches Dart: SassNull.unaryNot => sassTrue.
func (n *SassNull) UnaryNot() (Value, error) { return SassTrue, nil }
