// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/function_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// FunctionRule is a user-defined function declaration.
//
// It declares a function invoked with normal CSS function syntax.
type FunctionRule struct {
	base *callableDeclaration
}

// NewFunctionRule creates a @function rule with the given name, parameters,
// body children, span, and preceding doc comment.
func NewFunctionRule(originalName string, parameters *ParameterList, children []Statement, span sasscommon.FileSpan, comment *SilentComment) *FunctionRule {
	base := newCallableDeclaration(originalName, parameters, children, span, comment)
	return &FunctionRule{base: base}
}

// Declaration returns the rule as its callable-declaration view.
func (r *FunctionRule) Declaration() CallableDeclaration   { return r }
func (r *FunctionRule) Name() string                       { return r.base.Name() }
func (r *FunctionRule) OriginalName() string               { return r.base.OriginalName() }
func (r *FunctionRule) Parameters() *ParameterList         { return r.base.Parameters() }
func (r *FunctionRule) Span() (sasscommon.FileSpan, error) { return r.base.Span() }
func (r *FunctionRule) Comment() *SilentComment            { return r.base.Comment() }
func (r *FunctionRule) IsStatement()                       {}
func (r *FunctionRule) IsSassNode()                        {}
func (r *FunctionRule) IsAstNode()                         {}
func (r *FunctionRule) GetChildren() []Statement           { return r.base.GetChildren() }
func (r *FunctionRule) HasDeclarations() bool              { return r.base.HasDeclarations() }

// NameSpan returns the span of the function name only.
//
// Matches Dart: FunctionRule.nameSpan
func (r *FunctionRule) NameSpan() (sasscommon.FileSpan, error) {
	ss, err := r.base.span.WithoutInitialAtRule()
	if err != nil {
		return nil, err
	}
	ns, err := ss.InitialIdentifier(0)
	if err != nil {
		return nil, err
	}
	return ns, nil
}
func (r *FunctionRule) IsSassDeclaration() {}
func (r *FunctionRule) String() (string, error) {
	children := r.GetChildren()
	parts := make([]string, len(children))
	for i, child := range children {
		s, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		parts[i] = s
	}
	paramsStr, err := r.base.parameters.String()
	if err != nil {
		return "", err
	}
	return "@function " + r.Name() + "(" + paramsStr + ") {" + strings.Join(parts, " ") + "}", nil
}
