// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/interpolated_function.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// InterpolatedFunctionExpression is an interpolated function invocation.
//
// This is always a plain CSS function.
//
// Matches Dart: InterpolatedFunctionExpression
type InterpolatedFunctionExpression struct {
	// Name is the name of the function being invoked.
	Name      *Interpolation
	arguments *ArgumentList
	span      sasscommon.FileSpan
}

// NewInterpolatedFunctionExpression creates an interpolated function call.
//
// Matches Dart: InterpolatedFunctionExpression.new
func NewInterpolatedFunctionExpression(name *Interpolation, arguments *ArgumentList, span sasscommon.FileSpan) *InterpolatedFunctionExpression {
	return &InterpolatedFunctionExpression{Name: name, arguments: arguments, span: span}
}

func (e *InterpolatedFunctionExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *InterpolatedFunctionExpression) SourceInterpolation() *Interpolation { return nil }
func (e *InterpolatedFunctionExpression) IsExpression()                       {}
func (e *InterpolatedFunctionExpression) IsSassNode()                         {}
func (e *InterpolatedFunctionExpression) IsAstNode()                          {}
func (e *InterpolatedFunctionExpression) IsCallableInvocation()               {}

// Arguments returns the arguments to pass to the function.
//
// Matches Dart: InterpolatedFunctionExpression.arguments
func (e *InterpolatedFunctionExpression) Arguments() *ArgumentList { return e.arguments }

func (e *InterpolatedFunctionExpression) String() (string, error) {
	argsStr, err := e.arguments.String()
	if err != nil {
		return "", err
	}
	nameStr, err := e.Name.String()
	if err != nil {
		return "", err
	}
	return nameStr + argsStr, nil
}
