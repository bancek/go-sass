// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/function.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// FunctionExpression is a function invocation.
//
// This may be a plain CSS function or a Sass function, but may not include
// interpolation.
//
// Matches Dart: FunctionExpression
type FunctionExpression struct {
	// Namespace is the namespace of the function being invoked, or nil when
	// invoked without a namespace.
	Namespace *string
	// Name is the function name with underscores converted to hyphens. For a
	// plain CSS function, use OriginalName instead.
	Name string
	// OriginalName is the function name as written, with underscores left
	// as-is.
	OriginalName string
	args         *ArgumentList
	span         sasscommon.FileSpan
}

// NewFunctionExpression creates a function invocation. Underscores in
// originalName are converted to hyphens for Name, matching Sass's
// underscore/hyphen equivalence; namespace is nil for unqualified calls.
//
// Matches Dart: FunctionExpression.new
func NewFunctionExpression(
	originalName string,
	arguments *ArgumentList,
	span sasscommon.FileSpan,
	namespace *string,
) *FunctionExpression {
	name := strings.ReplaceAll(originalName, "_", "-")
	return &FunctionExpression{
		OriginalName: originalName,
		Name:         name,
		args:         arguments,
		span:         span,
		Namespace:    namespace,
	}
}

func (e *FunctionExpression) Span() (sasscommon.FileSpan, error) { return e.span, nil }

// Arguments returns the arguments to pass to the function.
func (e *FunctionExpression) Arguments() *ArgumentList { return e.args }

// NameSpan returns the span of the function name: the leading identifier of
// the call, skipping a namespace qualifier when one is present.
func (e *FunctionExpression) NameSpan() (sasscommon.FileSpan, error) {
	if e.Namespace == nil {
		result, err := initialIdentifier(e.span)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	withoutNS, err := withoutNamespace(e.span)
	if err != nil {
		return nil, err
	}
	result, err := initialIdentifier(withoutNS)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// NamespaceSpan returns the span of the namespace qualifier, or nil when
// the call has none.
func (e *FunctionExpression) NamespaceSpan() (*sasscommon.FileSpan, error) {
	if e.Namespace == nil {
		return nil, nil
	}
	result, err := initialIdentifier(e.span)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (e *FunctionExpression) SourceInterpolation() *Interpolation { return nil }
func (e *FunctionExpression) IsExpression()                       {}
func (e *FunctionExpression) IsSassNode()                         {}
func (e *FunctionExpression) IsAstNode()                          {}
func (e *FunctionExpression) IsCallableInvocation()               {}
func (e *FunctionExpression) isSassReference()                    {}

func (e *FunctionExpression) String() (string, error) {
	var buf strings.Builder
	if e.Namespace != nil {
		buf.WriteString(*e.Namespace)
		buf.WriteString(".")
	}
	buf.WriteString(e.OriginalName)
	argsStr, err := e.args.String()
	if err != nil {
		return "", err
	}
	buf.WriteString(argsStr)
	return buf.String(), nil
}
