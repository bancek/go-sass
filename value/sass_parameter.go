// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/parameter.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// Parameter is a parameter declared as part of a ParameterList.
//
// Matches Dart: Parameter
type Parameter struct {
	// name is the parameter name.
	name string
	// DefaultValue is the default value, or nil when none was declared.
	DefaultValue Expression
	span         sasscommon.FileSpan
}

// NewParameter creates a parameter with an optional default value.
//
// Matches Dart: Parameter.new
func NewParameter(name string, span sasscommon.FileSpan, defaultValue Expression) *Parameter {
	return &Parameter{
		name:         name,
		DefaultValue: defaultValue,
		span:         span,
	}
}

func (p *Parameter) Name() string                       { return p.name }
func (p *Parameter) Span() (sasscommon.FileSpan, error) { return p.span, nil }

// NameSpan returns the span covering just the variable name (without type/annotation).
//
// Matches Dart: Parameter.nameSpan
func (p *Parameter) NameSpan() (sasscommon.FileSpan, error) {
	if p.DefaultValue == nil {
		return p.span, nil
	}
	return p.span.InitialIdentifier(1)
}

// OriginalName returns the variable name as written, without underscore to
// hyphen conversion and including the leading `$`. It re-slices source text,
// so it is best reserved for error messages.
//
// Matches Dart: Parameter.originalName
func (p *Parameter) OriginalName() (string, error) {
	if p.DefaultValue == nil {
		return p.span.SpanText()
	}
	return declarationName(p.span)
}
func (p *Parameter) IsSassNode()        {}
func (p *Parameter) IsAstNode()         {}
func (p *Parameter) IsSassDeclaration() {}

// String renders the parameter as `$name` or `$name: default`.
func (p *Parameter) String() (string, error) {
	if p.DefaultValue == nil {
		return p.name, nil
	}
	valStr, err := p.DefaultValue.String()
	if err != nil {
		return "", err
	}
	return p.name + ": " + valStr, nil
}
