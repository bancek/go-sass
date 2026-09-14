// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/supports_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// SupportsRule is a @supports rule.
//
// It includes its children only when the browser satisfies Condition.
type SupportsRule struct {
	parent ParentStatement
	// Condition selects which browsers the nested rules target.
	Condition SupportsCondition
	span      sasscommon.FileSpan
}

// NewSupportsRule creates a @supports rule gating children on condition.
func NewSupportsRule(condition SupportsCondition, children []Statement, span sasscommon.FileSpan) *SupportsRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &SupportsRule{
		parent:    NewParentStatement(c),
		Condition: condition,
		span:      span,
	}
}

func (r *SupportsRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *SupportsRule) IsStatement()                       {}
func (r *SupportsRule) IsSassNode()                        {}
func (r *SupportsRule) IsAstNode()                         {}
func (r *SupportsRule) GetChildren() []Statement           { return r.parent.Children }
func (r *SupportsRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *SupportsRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@supports ")
	condStr, err := r.Condition.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(condStr)
	builder.WriteString(" {")
	parts := make([]string, len(r.parent.Children))
	for i, child := range r.parent.Children {
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
