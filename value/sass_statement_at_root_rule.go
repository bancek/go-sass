// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/at_root_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// AtRootRule is an @at-root rule.
//
// It moves its contents up the tree through parent nodes, past any style
// rules or at-rules that the query does not explicitly keep.
type AtRootRule struct {
	parent ParentStatement
	// Query selects which enclosing rules the contents move through.
	// A nil query means the default query, which excludes only style rules.
	Query *Interpolation
	span  sasscommon.FileSpan
}

// NewAtRootRule creates an AtRootRule holding children with span coverage.
// A nil query selects the default behavior of bubbling past style rules only.
func NewAtRootRule(children []Statement, span sasscommon.FileSpan, query *Interpolation) *AtRootRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &AtRootRule{
		parent: NewParentStatement(c),
		Query:  query,
		span:   span,
	}
}

func (r *AtRootRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *AtRootRule) IsStatement()                       {}
func (r *AtRootRule) IsSassNode()                        {}
func (r *AtRootRule) IsAstNode()                         {}
func (r *AtRootRule) GetChildren() []Statement           { return r.parent.Children }
func (r *AtRootRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *AtRootRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@at-root ")
	if r.Query != nil {
		queryStr, err := r.Query.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(queryStr)
		builder.WriteString(" ")
	}
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
