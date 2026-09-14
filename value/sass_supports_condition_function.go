// Copyright 2020 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/function.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SupportsFunction is a function-syntax condition such as selector(...)
// or font-tech(...).
type SupportsFunction struct {
	// Name is the function name under test.
	Name *Interpolation
	// Arguments is the parenthesized payload passed to the function.
	Arguments *Interpolation
	span      sasscommon.FileSpan
}

// NewSupportsFunction creates a function-syntax condition.
func NewSupportsFunction(name, arguments *Interpolation, span sasscommon.FileSpan) *SupportsFunction {
	return &SupportsFunction{Name: name, Arguments: arguments, span: span}
}

func (f *SupportsFunction) Span() (sasscommon.FileSpan, error) { return f.span, nil }
func (f *SupportsFunction) IsAstNode()                         {}
func (f *SupportsFunction) IsSassNode()                        {}
func (f *SupportsFunction) IsSupportsCondition()               {}

// ToInterpolation flattens the condition into source-equivalent text,
// preserving the separator between the name and its arguments.
func (f *SupportsFunction) ToInterpolation() (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	buf.AddInterpolation(f.Name)
	nameSpan, err := f.Name.Span()
	if err != nil {
		return nil, err
	}
	argsSpan, err := f.Arguments.Span()
	if err != nil {
		return nil, err
	}
	betweenSpan, err := nameSpan.Between(argsSpan)
	if err != nil {
		return nil, err
	}
	between, err := betweenSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(between)
	buf.AddInterpolation(f.Arguments)
	afterSpan, err := f.span.After(argsSpan)
	if err != nil {
		return nil, err
	}
	after, err := afterSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(after)
	return buf.Interpolation(f.span)
}

// WithSpan returns a copy of this condition covering span.
func (f *SupportsFunction) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsFunction(f.Name, f.Arguments, span)
}
func (f *SupportsFunction) String() (string, error) {
	nameStr, err := f.Name.String()
	if err != nil {
		return "", err
	}
	argsStr, err := f.Arguments.String()
	if err != nil {
		return "", err
	}
	return nameStr + "(" + argsStr + ")", nil
}
