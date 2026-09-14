// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/mixin_rule.dart

import (
	"strings"
	"unicode"

	"github.com/bancek/go-sass/sasscommon"
)

// MixinRule is a mixin declaration.
//
// It declares a mixin body invoked with @include.
type MixinRule struct {
	base               *callableDeclaration
	hasContent         bool
	hasContentComputed bool
}

// NewMixinRule creates a @mixin rule with the given name, parameters, body
// children, span, and preceding doc comment.
func NewMixinRule(originalName string, parameters *ParameterList, children []Statement, span sasscommon.FileSpan, comment *SilentComment) *MixinRule {
	base := newCallableDeclaration(originalName, parameters, children, span, comment)
	return &MixinRule{base: base}
}

// Declaration returns the rule as its callable-declaration view.
func (r *MixinRule) Declaration() CallableDeclaration   { return r }
func (r *MixinRule) Name() string                       { return r.base.Name() }
func (r *MixinRule) OriginalName() string               { return r.base.OriginalName() }
func (r *MixinRule) Parameters() *ParameterList         { return r.base.Parameters() }
func (r *MixinRule) Span() (sasscommon.FileSpan, error) { return r.base.Span() }
func (r *MixinRule) Comment() *SilentComment            { return r.base.Comment() }
func (r *MixinRule) IsStatement()                       {}
func (r *MixinRule) IsSassNode()                        {}
func (r *MixinRule) IsAstNode()                         {}
func (r *MixinRule) GetChildren() []Statement           { return r.base.GetChildren() }
func (r *MixinRule) HasDeclarations() bool              { return r.base.HasDeclarations() }

// HasContent returns whether this mixin contains a @content rule.
//
// Matches Dart: MixinRule.hasContent (late final, computed lazily via _HasContentVisitor).
func (r *MixinRule) HasContent() bool {
	if !r.hasContentComputed {
		r.hasContent = hasContentRule(r.GetChildren())
		r.hasContentComputed = true
	}
	return r.hasContent
}

// NameSpan returns the span covering just the mixin name, skipping a leading
// = in the indented syntax and the @mixin prefix otherwise.
func (r *MixinRule) NameSpan() (sasscommon.FileSpan, error) {
	text, err := r.base.span.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) > 0 && text[0] == '=' {
		start := 1
		for start < len(text) && unicode.IsSpace(rune(text[start])) {
			start++
		}
		length, err := r.base.span.Length()
		if err != nil {
			return nil, err
		}
		ss, err := r.base.span.Subspan(start, length)
		if err != nil {
			return nil, err
		}
		ns, err := ss.InitialIdentifier(0)
		if err != nil {
			return nil, err
		}
		return ns, nil
	}
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
func (r *MixinRule) IsSassDeclaration() {}

func hasContentRule(stmts []Statement) bool {
	v := &StatementSearchVisitor{
		ContentRuleFunc: func(_ *ContentRule) (bool, error) { return true, nil },
	}
	for _, stmt := range stmts {
		result, _ := stmt.AcceptBool(v)
		if result {
			return true
		}
	}
	return false
}

func (r *MixinRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@mixin ")
	builder.WriteString(r.Name())
	if len(r.base.parameters.Parameters) > 0 || r.base.parameters.RestParameter != nil {
		builder.WriteString("(")
		paramsStr, err := r.base.parameters.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(paramsStr)
		builder.WriteString(")")
	}
	builder.WriteString(" {")
	children := r.GetChildren()
	parts := make([]string, len(children))
	for i, child := range children {
		s, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		parts[i] = s
	}
	builder.WriteString(strings.Join(parts, " "))
	builder.WriteString("}")
	return builder.String(), nil
}
