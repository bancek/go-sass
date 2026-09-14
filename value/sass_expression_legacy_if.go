// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/legacy_if.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// LegacyIfExpression is a ternary expression.
//
// This is defined as a separate syntactic construct rather than a normal
// function because only one of the $if-true and $if-false arguments are
// evaluated.
//
// Matches Dart: LegacyIfExpression
type LegacyIfExpression struct {
	args *ArgumentList
	span sasscommon.FileSpan
}

// LegacyIfDeclaration is the parameter list for if() as though it were a
// normal function: if($condition, $if-true, $if-false). It is parsed once at
// init for arity checking and error messages.
//
// Matches Dart: LegacyIfExpression.declaration
var LegacyIfDeclaration *ParameterList

func init() {
	_, params, err := ParseSignature("if($condition, $if-true, $if-false)", true)
	if err != nil {
		panic("BUG: failed to parse legacy if signature: " + err.Error())
	}
	LegacyIfDeclaration = params
}

// NewLegacyIfExpression creates a legacy ternary if() call.
//
// Matches Dart: LegacyIfExpression.new
func NewLegacyIfExpression(arguments *ArgumentList, span sasscommon.FileSpan) *LegacyIfExpression {
	return &LegacyIfExpression{
		args: arguments,
		span: span,
	}
}

func (e *LegacyIfExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *LegacyIfExpression) Arguments() *ArgumentList            { return e.args }
func (e *LegacyIfExpression) SourceInterpolation() *Interpolation { return nil }
func (e *LegacyIfExpression) IsExpression()                       {}
func (e *LegacyIfExpression) IsSassNode()                         {}
func (e *LegacyIfExpression) IsAstNode()                          {}
func (e *LegacyIfExpression) IsCallableInvocation()               {}

// ModernSuggestion returns a modern if() expression to use instead of this
// legacy call, or nil when the call shape is not the plain three-positional
// form. A null fallback collapses to a not-sass() guard so the suggestion
// stays minimal.
//
// Matches Dart: LegacyIfExpression.modernSuggestion (internal)
func (e *LegacyIfExpression) ModernSuggestion() (*string, error) {
	if len(e.args.Positional) == 3 && e.args.Named.Len() == 0 && e.args.Rest == nil {
		condition := e.args.Positional[0]
		ifTrue := e.args.Positional[1]
		ifFalse := e.args.Positional[2]
		condStr, err := condition.String()
		if err != nil {
			return nil, err
		}
		trueStr, err := ifTrue.String()
		if err != nil {
			return nil, err
		}
		falseStr, err := ifFalse.String()
		if err != nil {
			return nil, err
		}

		var suggestion string
		if _, ok := any(ifFalse).(*NullExpression); ok {
			suggestion = fmt.Sprintf("if(sass(%s): %s)", condStr, trueStr)
		} else if _, ok := any(ifTrue).(*NullExpression); ok {
			suggestion = fmt.Sprintf("if(not sass(%s): %s)", condStr, falseStr)
		} else {
			suggestion = fmt.Sprintf("if(sass(%s): %s; else: %s)", condStr, trueStr, falseStr)
		}
		return &suggestion, nil
	}
	return nil, nil
}

func (e *LegacyIfExpression) String() (string, error) {
	argsStr, err := e.args.String()
	if err != nil {
		return "", err
	}
	return "if" + argsStr, nil
}
