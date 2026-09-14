// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/while_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// WhileRule is a @while rule.
//
// It re-evaluates Condition before each pass and runs the body for as long
// as the condition holds true.
type WhileRule struct {
	parent ParentStatement
	// Condition decides before each iteration whether the body runs again.
	Condition Expression
	span      sasscommon.FileSpan
}

// NewWhileRule creates a @while rule looping over children while condition
// evaluates to true.
func NewWhileRule(condition Expression, children []Statement, span sasscommon.FileSpan) *WhileRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &WhileRule{
		parent:    NewParentStatement(c),
		Condition: condition,
		span:      span,
	}
}

func (r *WhileRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *WhileRule) IsStatement()                       {}
func (r *WhileRule) IsSassNode()                        {}
func (r *WhileRule) IsAstNode()                         {}
func (r *WhileRule) GetChildren() []Statement           { return r.parent.Children }
func (r *WhileRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *WhileRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@while ")
	condStr, err := r.Condition.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(condStr)
	builder.WriteString(" {")
	parts := make([]string, 0, len(r.parent.Children))
	for _, child := range r.parent.Children {
		s, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	builder.WriteString(strings.Join(parts, " "))
	builder.WriteString("}")
	return builder.String(), nil
}
