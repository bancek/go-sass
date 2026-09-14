// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/configured_variable.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ConfiguredVariable is a variable configured by a `with` clause in a
// `@use` or `@forward` rule.
//
// Matches Dart: ConfiguredVariable
type ConfiguredVariable struct {
	// name is the name of the variable being configured.
	name string
	// Expression is the variable's value.
	Expression Expression
	// IsGuarded reports whether the variable can be further configured by
	// outer modules. This is always false for @use rules.
	IsGuarded bool
	span      sasscommon.FileSpan
}

// NewConfiguredVariable creates a configured variable. Guarded has no
// default in Go (Dart defaults it to false), so callers pass it explicitly.
//
// Matches Dart: ConfiguredVariable.new
func NewConfiguredVariable(
	name string,
	expression Expression,
	span sasscommon.FileSpan,
	guarded bool,
) *ConfiguredVariable {
	return &ConfiguredVariable{
		name:       name,
		Expression: expression,
		IsGuarded:  guarded,
		span:       span,
	}
}

func (v *ConfiguredVariable) Name() string                       { return v.name }
func (v *ConfiguredVariable) Span() (sasscommon.FileSpan, error) { return v.span, nil }

// NameSpan returns the span of the variable name, including the leading
// `$`.
func (v *ConfiguredVariable) NameSpan() (sasscommon.FileSpan, error) {
	span, err := v.span.InitialIdentifier(1)
	if err != nil {
		return nil, err
	}
	return span, nil
}
func (v *ConfiguredVariable) IsSassNode()        {}
func (v *ConfiguredVariable) IsAstNode()         {}
func (v *ConfiguredVariable) IsSassDeclaration() {}

func (v *ConfiguredVariable) String() (string, error) {
	exprStr, err := v.Expression.String()
	if err != nil {
		return "", err
	}
	if v.IsGuarded {
		return "$" + v.name + ": " + exprStr + " !default", nil
	}
	return "$" + v.name + ": " + exprStr, nil
}
