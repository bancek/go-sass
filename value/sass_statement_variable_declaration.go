// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/variable_declaration.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// VariableDeclaration defines or sets a variable.
type VariableDeclaration struct {
	// Expression is the value assigned to the variable.
	Expression Expression
	// IsGuarded selects a guarded assignment, which only applies when the
	// variable is currently undefined or nil.
	IsGuarded bool
	comment   *SilentComment
	namespace *string
	isGlobal  bool
	name      string
	span      sasscommon.FileSpan
}

// NewVariableDeclaration creates a variable assignment for name.
// A non-nil namespace targets another module's member, which cannot combine
// with global; that combination reports an error.
func NewVariableDeclaration(name string, expression Expression, span sasscommon.FileSpan, namespace *string, guarded bool, global bool, comment *SilentComment) (*VariableDeclaration, error) {
	if namespace != nil && global {
		return nil, &sasscommon.ArgumentError{Message: "Other modules' members can't be defined with !global."}
	}
	return &VariableDeclaration{
		name:       name,
		Expression: expression,
		span:       span,
		namespace:  namespace,
		IsGuarded:  guarded,
		isGlobal:   global,
		comment:    comment,
	}, nil
}

// Name returns the variable name with underscores folded to hyphens.
func (d *VariableDeclaration) Name() string { return d.name }

// IsGlobal reports whether the assignment always targets the global scope.
func (d *VariableDeclaration) IsGlobal() bool { return d.isGlobal }

// Namespace returns the namespace of the variable being set,
// or nil when it is defined without a namespace.
func (d *VariableDeclaration) Namespace() *string { return d.namespace }

// OriginalName returns the variable name as written, including the leading $.
//
// Matches Dart: VariableDeclaration.originalName
func (d *VariableDeclaration) OriginalName() (string, error) {
	return declarationName(d.span)
}

// NameSpan returns the span covering the variable name itself,
// skipping any namespace qualifier.
func (d *VariableDeclaration) NameSpan() (sasscommon.FileSpan, error) {
	span := d.span
	if d.namespace != nil {
		var err error
		span, err = withoutNamespace(span)
		if err != nil {
			return nil, err
		}
	}
	return span.InitialIdentifier(1)
}

// NamespaceSpan returns the span covering the namespace qualifier,
// or nil when the declaration is unqualified.
func (d *VariableDeclaration) NamespaceSpan() (*sasscommon.FileSpan, error) {
	if d.namespace == nil {
		return nil, nil
	}
	result, err := initialIdentifier(d.span)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (d *VariableDeclaration) Span() (sasscommon.FileSpan, error) { return d.span, nil }
func (d *VariableDeclaration) IsStatement()                       {}
func (d *VariableDeclaration) IsSassNode()                        {}
func (d *VariableDeclaration) IsAstNode()                         {}
func (d *VariableDeclaration) IsSassDeclaration()                 {}

// Comment returns the doc comment immediately preceding the declaration,
// or nil when there is none.
func (d *VariableDeclaration) Comment() *SilentComment { return d.comment }
func (d *VariableDeclaration) String() (string, error) {
	var builder strings.Builder
	if d.namespace != nil {
		builder.WriteString(*d.namespace)
		builder.WriteString(".")
	}
	builder.WriteString("$")
	builder.WriteString(d.name)
	builder.WriteString(": ")
	exprStr, err := d.Expression.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(exprStr)
	builder.WriteString(";")
	return builder.String(), nil
}
