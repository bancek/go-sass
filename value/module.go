// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/module.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassurl"
)

// ModuleRef is the part of a Sass module that a first-class module value
// needs. It is implemented structurally by sassmodule.Module without
// importing sassmodule here (which would create a package cycle).
//
// Matches Dart: Module.url
type ModuleRef interface {
	URL() (string, error)
	Equal(other ModuleRef) bool
	HashCode() int
}

// SassModule is a SassScript module reference.
//
// A module reference allows SassScript to reflect on and interact with
// modules.
//
// Matches Dart: SassModule
type SassModule struct {
	// Module is the wrapped module.
	Module ModuleRef
	// compileContext identifies the compilation that owns this reference.
	compileContext any
}

// NewSassModule creates a first-class module value with no compile context.
//
// Matches Dart: SassModule(this.module)
func NewSassModule(module ModuleRef) *SassModule {
	return &SassModule{Module: module}
}

// NewSassModuleWithCompileContext creates a first-class module value tied to
// the given compilation context.
//
// Matches Dart: SassModule.withCompileContext
func NewSassModuleWithCompileContext(module ModuleRef, compileContext any) *SassModule {
	return &SassModule{Module: module, compileContext: compileContext}
}

func (m *SassModule) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) {
	return v.VisitModule(m)
}

func (m *SassModule) isValue() {}

// AssertModule returns this SassModule.
//
// Matches Dart: SassModule.assertModule
func (m *SassModule) AssertModule(name ...string) *SassModule { return m }

// AssertModule returns v as a *SassModule, or an error if it isn't a module
// reference.
//
// Matches Dart: Value.assertModule
func AssertModule(v Value, name *string) (*SassModule, error) {
	m, ok := v.(*SassModule)
	if !ok {
		vStr, err := v.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a module reference.", vStr), name)
	}
	return m, nil
}

// AssertCompileContext asserts that this module belongs to compileContext.
//
// Matches Dart: SassModule.assertCompileContext
func (m *SassModule) AssertCompileContext(compileContext any) (*SassModule, error) {
	if m.compileContext != nil && m.compileContext != compileContext {
		mStr, err := m.String()
		if err != nil {
			return nil, err
		}
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("%s does not belong to current compilation.", mStr), nil)
	}
	return m, nil
}

// Equals compares the compile context and module identity.
//
// Matches Dart: SassModule.==
func (m *SassModule) Equals(other Value) bool {
	om, ok := other.(*SassModule)
	if !ok {
		return false
	}
	return m.compileContext == om.compileContext && m.Module.Equal(om.Module)
}

// HashCode returns the module's identity hash, matching Equals.
//
// Matches Dart: SassModule.hashCode => module.hashCode
func (m *SassModule) HashCode() int {
	if m.Module == nil {
		return 0
	}
	return m.Module.HashCode()
}

// ToCssString renders a module for CSS output, which always errors; use
// String for the inspect form.
func (m *SassModule) ToCssString(quote bool) (string, error) { return SerializeValue(m, quote) }

// String returns the inspect form, get-module("namespace").
func (m *SassModule) String() (string, error) { return SerializeValueInspect(m) }

func (m *SassModule) IsTruthy() bool                          { return true }
func (m *SassModule) Separator() ListSeparator                { return ListSeparatorUndecided }
func (m *SassModule) HasBrackets() bool                       { return false }
func (m *SassModule) AsList() ([]Value, error)                { return []Value{m}, nil }
func (m *SassModule) LengthAsList() int                       { return 1 }
func (m *SassModule) IsBlank() bool                           { return false }
func (m *SassModule) IsSpecialNumber() bool                   { return false }
func (m *SassModule) IsSpecialVariable() bool                 { return false }
func (m *SassModule) TryMap() *SassMap                        { return nil }
func (m *SassModule) RealNull() Value                         { return DefaultRealNull(m) }
func (m *SassModule) SingleEquals(other Value) (Value, error) { return DefaultSingleEquals(m, other) }
func (m *SassModule) Times(other Value) (Value, error)        { return DefaultTimes(m, other) }
func (m *SassModule) Modulo(other Value) (Value, error)       { return DefaultModulo(m, other) }
func (m *SassModule) GreaterThan(other Value) (Value, error)  { return DefaultGreaterThan(m, other) }
func (m *SassModule) GreaterThanOrEquals(other Value) (Value, error) {
	return DefaultGreaterThanOrEquals(m, other)
}
func (m *SassModule) LessThan(other Value) (Value, error) { return DefaultLessThan(m, other) }
func (m *SassModule) LessThanOrEquals(other Value) (Value, error) {
	return DefaultLessThanOrEquals(m, other)
}
func (m *SassModule) UnaryNot() (Value, error) { return DefaultUnaryNot(m) }

func (m *SassModule) Plus(other Value) (Value, error) {
	return moduleUndefinedBinary(m, other, "+")
}
func (m *SassModule) Minus(other Value) (Value, error) {
	return moduleUndefinedBinary(m, other, "-")
}
func (m *SassModule) DividedBy(other Value) (Value, error) {
	return moduleUndefinedBinary(m, other, "/")
}
func (m *SassModule) UnaryPlus() (Value, error) {
	return moduleUndefinedUnary(m, "+", false)
}
func (m *SassModule) UnaryMinus() (Value, error) {
	return moduleUndefinedUnary(m, "-", true)
}
func (m *SassModule) UnaryDivide() (Value, error) {
	return moduleUndefinedUnary(m, "/", false)
}

func moduleUndefinedBinary(left *SassModule, right Value, operator string) (Value, error) {
	leftStr, err := left.String()
	if err != nil {
		return nil, err
	}
	rightStr, err := right.String()
	if err != nil {
		return nil, err
	}
	return nil, sasscommon.NewSassScriptException(
		fmt.Sprintf("Undefined operation \"%s %s %s\".", leftStr, operator, rightStr), nil)
}

func moduleUndefinedUnary(value *SassModule, operator string, parenthesize bool) (Value, error) {
	repr, err := value.String()
	if err != nil {
		return nil, err
	}
	operand := repr
	if parenthesize {
		operand = "(" + repr + ")"
	}
	return nil, sasscommon.NewSassScriptException(
		fmt.Sprintf("Undefined operation \"%s%s\".", operator, operand), nil)
}

// moduleDefaultNamespace returns the default namespace for a module URL,
// matching Dart's defaultNamespace and the serializer's get-module() form.
func moduleDefaultNamespace(urlString string) (string, bool) {
	if urlString == "" {
		return "", false
	}
	u, err := sassurl.Parse(urlString)
	if err != nil {
		return "", false
	}
	return DefaultNamespace(u)
}
