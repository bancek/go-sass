// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/function.dart

import (
	"fmt"
	"reflect"

	"github.com/bancek/go-sass/sasscommon"
)

// FunctionRef is a reference to a callable function.
//
// It is implemented structurally by sasscallable.Callable implementations
// (BuiltInCallable, UserDefinedCallable, PlainCssCallable, host callables)
// without importing those packages, so go/value stays a leaf and avoids a
// package cycle. Invocation is handled by the evaluator via type dispatch.
//
// Matches Dart: AsyncCallable (function.callable)
type FunctionRef interface {
	Name() string
}

// A SassScript function reference.
//
// A function reference captures a function from the local environment so that
// it may be passed between modules.
//
// Equality is the callable's identity: two references are equal only when
// they invoke the same callable.
type SassFunction struct {
	// FunctionRef is the actual callable reference, if available.
	// When set, call() uses this directly instead of looking up by name.
	FunctionRef FunctionRef

	// _compileContext tracks the compilation context for cross-compilation safety.
	compileContext any
}

// Name returns the callable's name, or "" if no callable is stored.
//
// Matches Dart: SassFunction.callable.name
func (f *SassFunction) Name() string {
	if f.FunctionRef == nil {
		return ""
	}
	return f.FunctionRef.Name()
}

// NewSassFunction creates a SassFunction with the given callable reference.
//
// The result carries no compile context, like Dart's plain constructor —
// cross-compilation checks only apply to references built with
// NewSassFunctionWithCompileContext.
//
// Matches Dart: SassFunction(this.callable)
func NewSassFunction(callable FunctionRef) *SassFunction {
	return &SassFunction{FunctionRef: callable}
}

// NewSassFunctionWithCompileContext creates a SassFunction with a compile context.
//
// The token identifies the compilation that owns the callable, so later
// AssertCompileContext calls can reject references leaked across compilations.
//
// Matches Dart: SassFunction.withCompileContext
func NewSassFunctionWithCompileContext(callable FunctionRef, compileContext any) *SassFunction {
	return &SassFunction{FunctionRef: callable, compileContext: compileContext}
}

// Equals reports whether other invokes the same callable (identity, not
// structural equality). A nil callable only equals another nil callable.
//
// Matches Dart: SassFunction.== (callable == other.callable).
func (f *SassFunction) Equals(other Value) bool {
	if of, ok := other.(*SassFunction); ok {
		return f.FunctionRef == of.FunctionRef
	}
	return false
}

func (f *SassFunction) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitFunction(f)
}

// AssertFunction returns this SassFunction, asserting it's a function.
//
// Matches Dart: SassFunction.assertFunction
func (f *SassFunction) AssertFunction(name ...string) *SassFunction { return f }

// AssertCompileContext asserts that this function belongs to the given
// compile context.
//
// The evaluator calls this before invoking a function reference so a value
// captured in one compilation can never execute in another. References with
// no context (such as host-defined callables) pass unconditionally.
//
// Matches Dart: SassFunction.assertCompileContext
func (f *SassFunction) AssertCompileContext(compileContext any) (*SassFunction, error) {
	if f.compileContext != nil && f.compileContext != compileContext {
		fStr, err := f.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s does not belong to current compilation.", fStr), nil)
	}
	return f, nil
}
func (f *SassFunction) isValue() {}

// ToCssString renders this function for CSS output. Functions have no
// plain-CSS form, so this always errors; use String for the inspect rendering.
// (Dart: Value.toCssString, which throws for functions.)
func (f *SassFunction) ToCssString(quote bool) (string, error) { return SerializeValue(f, quote) }

// String returns the inspect rendering of this function (Dart toString): the
// callable's name, used in error messages such as the compile-context check.
func (f *SassFunction) String() (string, error) { return SerializeValueInspect(f) }

// IsTruthy reports whether this value counts as true in an @if test and other
// conditional contexts. Functions are always truthy.
func (f *SassFunction) IsTruthy() bool { return true }

// HashCode returns the callable's identity hash, matching Equals: references
// to the same callable hash alike. Dart hashes the callable object itself;
// Go folds the interface's pointer since callables are always pointers.
//
// Matches Dart: SassFunction.hashCode => callable.hashCode.
func (f *SassFunction) HashCode() int {
	if f.FunctionRef == nil {
		return 0
	}
	rv := reflect.ValueOf(f.FunctionRef)
	if rv.Kind() == reflect.Pointer {
		return int(rv.Pointer())
	}
	return 0
}

func (f *SassFunction) Separator() ListSeparator                { return ListSeparatorUndecided }
func (f *SassFunction) HasBrackets() bool                       { return false }
func (f *SassFunction) AsList() ([]Value, error)                { return []Value{f}, nil }
func (f *SassFunction) LengthAsList() int                       { return 1 }
func (f *SassFunction) IsBlank() bool                           { return false }
func (f *SassFunction) IsSpecialNumber() bool                   { return false }
func (f *SassFunction) IsSpecialVariable() bool                 { return false }
func (f *SassFunction) TryMap() *SassMap                        { return nil }
func (f *SassFunction) RealNull() Value                         { return DefaultRealNull(f) }
func (f *SassFunction) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(f, other) }
func (f *SassFunction) Plus(other Value) (Value, error)         { return DefaultPlus(f, other) }
func (f *SassFunction) Minus(other Value) (Value, error)        { return DefaultMinus(f, other) }
func (f *SassFunction) Times(other Value) (Value, error)        { return DefaultTimes(f, other) }
func (f *SassFunction) DividedBy(other Value) (Value, error)    { return DefaultDividedBy(f, other) }
func (f *SassFunction) Modulo(other Value) (Value, error)       { return DefaultModulo(f, other) }
func (f *SassFunction) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(f, other) }
func (f *SassFunction) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(f, other)
}
func (f *SassFunction) LessThan(other Value) (Value, error) { return DefaultLessThan(f, other) }
func (f *SassFunction) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(f, other)
}
func (f *SassFunction) UnaryPlus() (Value, error)   { return DefaultUnaryPlus(f) }
func (f *SassFunction) UnaryMinus() (Value, error)  { return DefaultUnaryMinus(f) }
func (f *SassFunction) UnaryDivide() (Value, error) { return DefaultUnaryDivide(f) }
func (f *SassFunction) UnaryNot() (Value, error)    { return DefaultUnaryNot(f) }
