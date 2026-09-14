// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/for_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ForRule is a @for rule.
//
// It counts from the From value to the To value, binding the index to
// Variable on each pass and evaluating the body that many times.
type ForRule struct {
	parent ParentStatement
	// Variable names the index variable without the leading dollar.
	Variable string
	// From is the expression for the start index.
	From Expression
	// To is the expression for the end index.
	To Expression
	// IsExclusive selects "to" (end excluded) over "through" (end included).
	IsExclusive bool
	span        sasscommon.FileSpan
}

// NewForRule creates a ForRule. Dart's ForRule(..., {bool exclusive = true})
// defaults exclusive to true. Go requires an explicit value; the sole caller
// (parse_stylesheet_atrule.go) always provides one.
func NewForRule(variable string, from, to Expression, children []Statement, span sasscommon.FileSpan, exclusive bool) *ForRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &ForRule{
		parent:      NewParentStatement(c),
		Variable:    variable,
		From:        from,
		To:          to,
		IsExclusive: exclusive,
		span:        span,
	}
}

func (r *ForRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ForRule) IsStatement()                       {}
func (r *ForRule) IsSassNode()                        {}
func (r *ForRule) IsAstNode()                         {}
func (r *ForRule) GetChildren() []Statement           { return r.parent.Children }
func (r *ForRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *ForRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@for $")
	builder.WriteString(r.Variable)
	builder.WriteString(" from ")
	fromStr, err := r.From.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(fromStr)
	builder.WriteString(" ")
	if r.IsExclusive {
		builder.WriteString("to")
	} else {
		builder.WriteString("through")
	}
	builder.WriteString(" ")
	toStr, err := r.To.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(toStr)
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
