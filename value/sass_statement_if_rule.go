// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/if_rule.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// IfRule is an @if rule.
//
// It runs the first clause whose condition is true, falling back to the
// unconditional @else clause when no condition matches.
type IfRule struct {
	// Clauses holds the @if and @else if branches in source order.
	// The first clause whose expression is true runs its statements.
	Clauses []*IfClause
	// LastClause is the final unconditional @else branch,
	// or nil when there is none.
	LastClause *ElseClause
	span       sasscommon.FileSpan
}

// NewIfRule creates an @if rule from clauses with an optional trailing
// LastClause for the plain @else branch.
func NewIfRule(clauses []*IfClause, span sasscommon.FileSpan, lastClause *ElseClause) *IfRule {
	c := make([]*IfClause, len(clauses))
	copy(c, clauses)
	return &IfRule{
		Clauses:    c,
		LastClause: lastClause,
		span:       span,
	}
}

func (r *IfRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *IfRule) IsStatement()                       {}
func (r *IfRule) IsSassNode()                        {}
func (r *IfRule) IsAstNode()                         {}

func (r *IfRule) String() (string, error) {
	var builder strings.Builder
	for i, clause := range r.Clauses {
		if i > 0 {
			builder.WriteString(" ")
		}
		if i == 0 {
			builder.WriteString("@if ")
		} else {
			builder.WriteString("@else if ")
		}
		exprStr, err := clause.Expression.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(exprStr)
		builder.WriteString(" {")
		parts := make([]string, 0, len(clause.Children()))
		for _, child := range clause.Children() {
			s, err := statementChildString(child)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		builder.WriteString(strings.Join(parts, " "))
		builder.WriteString("}")
	}
	if r.LastClause != nil {
		builder.WriteString(" ")
		lastStr, err := r.LastClause.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(lastStr)
	}
	return builder.String(), nil
}

// IfRuleClause is the shared interface of the @if and @else clause types.
// Each clause carries the statements to run when it matches.
type IfRuleClause interface {
	isIfRuleClause()
	Children() []Statement
	HasDeclarations() bool
}

// IfRuleClauseBase provides shared state for IfRuleClause types.
type IfRuleClauseBase struct {
	children        []Statement
	hasDeclarations bool
}

func newIfRuleClauseBase(children []Statement) *IfRuleClauseBase {
	c := make([]Statement, len(children))
	copy(c, children)
	return &IfRuleClauseBase{
		children:        c,
		hasDeclarations: hasDeclarations(c),
	}
}

func (b *IfRuleClauseBase) Children() []Statement { return b.children }
func (b *IfRuleClauseBase) HasDeclarations() bool { return b.hasDeclarations }

// IfClause is an @if or @else if branch: a condition plus the statements
// to run when that condition is the first to hold.
type IfClause struct {
	IfRuleClauseBase
	// Expression decides whether this branch runs.
	Expression Expression
}

// NewIfClause creates a conditional branch running children when expression
// evaluates to true.
func NewIfClause(expression Expression, children []Statement) *IfClause {
	return &IfClause{
		IfRuleClauseBase: *newIfRuleClauseBase(children),
		Expression:       expression,
	}
}

func (c *IfClause) isIfRuleClause() {}
func (c *IfClause) String() (string, error) {
	exprStr, err := c.Expression.String()
	if err != nil {
		return "", err
	}
	childStr, err := joinStatements(c.Children())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@if %v {%s}", exprStr, childStr), nil
}

// ElseClause is the final unconditional @else branch of an @if rule.
type ElseClause struct {
	IfRuleClauseBase
}

// NewElseClause creates an unconditional @else branch running children.
func NewElseClause(children []Statement) *ElseClause {
	return &ElseClause{IfRuleClauseBase: *newIfRuleClauseBase(children)}
}
func (c *ElseClause) isIfRuleClause() {}
func (c *ElseClause) String() (string, error) {
	childStr, err := joinStatements(c.Children())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@else {%s}", childStr), nil
}

func joinStatements(children []Statement) (string, error) {
	parts := make([]string, 0, len(children))
	for _, child := range children {
		s, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " "), nil
}
