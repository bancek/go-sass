// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/expression/variable.dart

// VariableExpression is a Sass variable reference.
//
// Matches Dart: VariableExpression
type VariableExpression struct {
	namespace *string
	// name is the variable name with underscores converted to hyphens.
	name string
	span sasscommon.FileSpan
}

// NewVariableExpression creates a variable reference. Underscores in name
// are the caller's responsibility to normalize; namespace is nil for
// unqualified references.
//
// Matches Dart: VariableExpression.new
func NewVariableExpression(name string, span sasscommon.FileSpan, namespace *string) *VariableExpression {
	return &VariableExpression{name: name, span: span, namespace: namespace}
}

func (e *VariableExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *VariableExpression) SourceInterpolation() *Interpolation { return nil }
func (e *VariableExpression) IsExpression()                       {}
func (e *VariableExpression) IsSassNode()                         {}
func (e *VariableExpression) IsAstNode()                          {}
func (e *VariableExpression) isSassReference()                    {}

// Namespace returns the namespace of the variable, or nil when referenced
// without one.
func (e *VariableExpression) Namespace() *string { return e.namespace }

// Name returns the variable name with underscores converted to hyphens,
// without the leading `$`.
func (e *VariableExpression) Name() string { return e.name }

// NameSpan returns the span of the variable name, without the namespace.
//
// Matches Dart: VariableExpression.nameSpan
func (e *VariableExpression) NameSpan() (sasscommon.FileSpan, error) {
	if e.namespace == nil {
		return e.span, nil
	}
	result, err := withoutNamespace(e.span)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// NamespaceSpan returns the span of the namespace, or nil if there is none.
//
// Matches Dart: VariableExpression.namespaceSpan
func (e *VariableExpression) NamespaceSpan() (*sasscommon.FileSpan, error) {
	if e.namespace == nil {
		return nil, nil
	}
	result, err := initialIdentifier(e.span)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// String returns the variable's source text, preserving the original
// underscores and namespace spelling.
//
// Matches Dart: VariableExpression.toString (returns span text rather than
// the normalized name)
func (e *VariableExpression) String() (string, error) {
	return e.span.SpanText()
}
