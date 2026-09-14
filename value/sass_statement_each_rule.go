// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/each_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// EachRule is an @each rule.
//
// It assigns each value (or key/value pair, for maps) from the iterated
// expression to Variables in turn and evaluates the body for each step.
type EachRule struct {
	parent ParentStatement
	// Variables holds the loop variable names without leading dollars.
	// Maps bind two variables per iteration; lists bind one.
	Variables []string
	// List is the expression whose value the rule iterates through.
	List Expression
	span sasscommon.FileSpan
}

// NewEachRule creates an @each rule binding variables to each value of list
// and evaluating children per iteration.
func NewEachRule(variables []string, list Expression, children []Statement, span sasscommon.FileSpan) *EachRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	v := make([]string, len(variables))
	copy(v, variables)
	return &EachRule{
		parent:    NewParentStatement(c),
		Variables: v,
		List:      list,
		span:      span,
	}
}

func (r *EachRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *EachRule) IsStatement()                       {}
func (r *EachRule) IsSassNode()                        {}
func (r *EachRule) IsAstNode()                         {}
func (r *EachRule) GetChildren() []Statement           { return r.parent.Children }
func (r *EachRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *EachRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@each ")
	vars := make([]string, len(r.Variables))
	for i, v := range r.Variables {
		vars[i] = "$" + v
	}
	builder.WriteString(strings.Join(vars, ", "))
	builder.WriteString(" in ")
	listStr, err := r.List.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(listStr)
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
