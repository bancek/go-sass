// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import (
	"fmt"
	"reflect"

	"github.com/bancek/go-sass/sasscommon"
)

// dart-source: lib/src/value/mixin.dart

// MixinRef is a reference to a callable mixin.
//
// It is implemented structurally by sasscallable.Callable implementations
// without importing those packages, so go/value stays a leaf and avoids a
// package cycle. Whether the mixin accepts content is determined by the
// evaluator via type dispatch.
//
// Matches Dart: AsyncCallable (mixin.callable)
type MixinRef interface {
	Name() string
}

// A SassScript mixin reference.
//
// A mixin reference captures a mixin from the local environment so that
// it may be passed between modules.
//
// Equality is the callable's identity: two references are equal only when
// they invoke the same callable.
type SassMixin struct {
	// MixinRef is the actual callable reference, if available.
	MixinRef MixinRef

	// compileContext tracks the compilation context for cross-compilation safety.
	compileContext any
}

// Name returns the callable's name, or "" if no callable is stored.
//
// Matches Dart: SassMixin.callable.name
func (m *SassMixin) Name() string {
	if m.MixinRef == nil {
		return ""
	}
	return m.MixinRef.Name()
}

// NewSassMixin creates a SassMixin with the given callable reference.
//
// The result carries no compile context, like Dart's plain constructor —
// cross-compilation checks only apply to references built with
// NewSassMixinWithCompileContext.
//
// Matches Dart: SassMixin(this.callable)
func NewSassMixin(ref MixinRef) *SassMixin {
	return &SassMixin{MixinRef: ref}
}

// NewSassMixinWithCompileContext creates a SassMixin with a compile context.
//
// The token identifies the compilation that owns the callable, so later
// AssertCompileContext calls can reject references leaked across compilations.
//
// Matches Dart: SassMixin.withCompileContext
func NewSassMixinWithCompileContext(ref MixinRef, compileContext any) *SassMixin {
	return &SassMixin{MixinRef: ref, compileContext: compileContext}
}

// Equals reports whether other invokes the same callable (identity, not
// structural equality). A nil callable only equals another nil callable.
//
// Matches Dart: SassMixin.== (callable == other.callable).
func (m *SassMixin) Equals(other Value) bool {
	if om, ok := other.(*SassMixin); ok {
		return m.MixinRef == om.MixinRef
	}
	return false
}

func (m *SassMixin) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitMixin(m) }

// AssertMixin returns this SassMixin, asserting it's a mixin.
//
// Matches Dart: SassMixin.assertMixin
func (m *SassMixin) AssertMixin(name ...string) *SassMixin { return m }

// AssertCompileContext asserts that this mixin belongs to the given compile context.
//
// The evaluator calls this before invoking a mixin reference so a value
// captured in one compilation can never execute in another. References with
// no context pass unconditionally.
//
// Matches Dart: SassMixin.assertCompileContext
func (m *SassMixin) AssertCompileContext(compileContext any) (*SassMixin, error) {
	if m.compileContext != nil && m.compileContext != compileContext {
		mStr, err := m.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s does not belong to current compilation.", mStr), nil)
	}
	return m, nil
}

// HashCode returns the callable's identity hash, matching Equals: references
// to the same callable hash alike. Dart hashes the callable object itself;
// Go folds the interface's pointer since callables are always pointers.
//
// Matches Dart: SassMixin.hashCode => callable.hashCode.
func (m *SassMixin) HashCode() int {
	if m.MixinRef == nil {
		return 0
	}
	rv := reflect.ValueOf(m.MixinRef)
	if rv.Kind() == reflect.Pointer {
		return int(rv.Pointer())
	}
	return 0
}

func (m *SassMixin) isValue() {}

// ToCssString renders this mixin for CSS output. Mixins have no plain-CSS
// form, so this always errors; use String for the inspect rendering.
// (Dart: Value.toCssString, which throws for mixins.)
func (m *SassMixin) ToCssString(quote bool) (string, error) { return SerializeValue(m, quote) }

// String returns the inspect rendering of this mixin (Dart toString): the
// callable's name, used in error messages such as the compile-context check.
func (m *SassMixin) String() (string, error) { return SerializeValueInspect(m) }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Mixins are always truthy.
func (m *SassMixin) IsTruthy() bool                          { return true }
func (m *SassMixin) Separator() ListSeparator                { return ListSeparatorUndecided }
func (m *SassMixin) HasBrackets() bool                       { return false }
func (m *SassMixin) AsList() ([]Value, error)                { return []Value{m}, nil }
func (m *SassMixin) LengthAsList() int                       { return 1 }
func (m *SassMixin) IsBlank() bool                           { return false }
func (m *SassMixin) IsSpecialNumber() bool                   { return false }
func (m *SassMixin) IsSpecialVariable() bool                 { return false }
func (m *SassMixin) TryMap() *SassMap                        { return nil }
func (m *SassMixin) RealNull() Value                         { return DefaultRealNull(m) }
func (m *SassMixin) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(m, other) }
func (m *SassMixin) Plus(other Value) (Value, error)         { return DefaultPlus(m, other) }
func (m *SassMixin) Minus(other Value) (Value, error)        { return DefaultMinus(m, other) }
func (m *SassMixin) Times(other Value) (Value, error)        { return DefaultTimes(m, other) }
func (m *SassMixin) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(m, other) }
func (m *SassMixin) Modulo(other Value) (Value, error)       { return DefaultModulo(m, other) }
func (m *SassMixin) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(m, other) }
func (m *SassMixin) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(m, other)
}
func (m *SassMixin) LessThan(other Value) (Value, error) { return DefaultLessThan(m, other) }
func (m *SassMixin) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(m, other)
}
func (m *SassMixin) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(m) }
func (m *SassMixin) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(m) }
func (m *SassMixin) UnaryDivide() (Value, error) { return DefaultUnaryDivide(m) }
func (m *SassMixin) UnaryNot() (Value, error)    { return DefaultUnaryNot(m) }
