// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/boolean.dart

// SassBoolean is a SassScript boolean value.
type SassBoolean struct {
	// Whether this value is true or false.
	Value bool
}

// The SassScript true value.
var SassTrue = &SassBoolean{Value: true}

// The SassScript false value.
var SassFalse = &SassBoolean{Value: false}

// NewSassBoolean returns SassTrue or SassFalse corresponding to value.
//
// Matches Dart: SassBoolean factory constructor
func NewSassBoolean(value bool) *SassBoolean {
	if value {
		return SassTrue
	}
	return SassFalse
}

func (b *SassBoolean) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitBoolean(b)
}
func (b *SassBoolean) isValue()                 {}
func (b *SassBoolean) TryMap() *SassMap         { return nil }
func (b *SassBoolean) IsTruthy() bool           { return b.Value }
func (b *SassBoolean) Separator() ListSeparator { return ListSeparatorUndecided }
func (b *SassBoolean) HasBrackets() bool        { return false }
func (b *SassBoolean) AsList() ([]Value, error) { return []Value{b}, nil }
func (b *SassBoolean) LengthAsList() int        { return 1 }
func (b *SassBoolean) IsBlank() bool            { return false }
func (b *SassBoolean) IsSpecialNumber() bool    { return false }
func (b *SassBoolean) IsSpecialVariable() bool  { return false }
func (b *SassBoolean) RealNull() Value          { return DefaultRealNull(b) }

// AssertBoolean returns this value.
//
// Matches Dart: SassBoolean.assertBoolean
func (b *SassBoolean) AssertBoolean(name ...string) (*SassBoolean, error) {
	return b, nil
}

func (b *SassBoolean) HashCode() int {
	if b.Value {
		return 1
	}
	return 0
}

// String returns "true" or "false".
func (b *SassBoolean) Equals(other Value) bool {
	if ob, ok := other.(*SassBoolean); ok {
		return b.Value == ob.Value
	}
	return false
}

func (b *SassBoolean) ToCssString(quote bool) (string, error) { return SerializeValue(b, quote) }

func (b *SassBoolean) String() (string, error) { return SerializeValueInspect(b) }

// UnaryNot matches Dart: SassBoolean.unaryNot
func (b *SassBoolean) UnaryNot() (Value, error) {
	if b.Value {
		return SassFalse, nil
	}
	return SassTrue, nil
}

func (b *SassBoolean) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(b, other) }
func (b *SassBoolean) Plus(other Value) (Value, error)         { return DefaultPlus(b, other) }
func (b *SassBoolean) Minus(other Value) (Value, error)        { return DefaultMinus(b, other) }
func (b *SassBoolean) Times(other Value) (Value, error)        { return DefaultTimes(b, other) }
func (b *SassBoolean) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(b, other) }
func (b *SassBoolean) Modulo(other Value) (Value, error)       { return DefaultModulo(b, other) }
func (b *SassBoolean) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(b, other) }
func (b *SassBoolean) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(b, other)
}
func (b *SassBoolean) LessThan(other Value) (Value, error) { return DefaultLessThan(b, other) }
func (b *SassBoolean) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(b, other)
}
func (b *SassBoolean) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(b) }
func (b *SassBoolean) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(b) }
func (b *SassBoolean) UnaryDivide() (Value, error) { return DefaultUnaryDivide(b) }
