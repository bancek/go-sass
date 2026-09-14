// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/at_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// AtRule is an unknown at-rule.
//
// It covers any at-rule Sass does not model with a dedicated node, such as
// `@unknown`. Known rules parse into their own statement types instead.
type AtRule struct {
	parent ParentStatement
	// Name is the interpolated at-rule name without the leading @.
	Name *Interpolation
	// Value is the interpolated rule value, or nil when the rule takes none.
	Value *Interpolation
	span  sasscommon.FileSpan
}

// NewAtRule creates an AtRule with the given name, optional value, and
// optional block children. A nil children slice means the rule ends with a
// semicolon rather than opening a block.
func NewAtRule(name *Interpolation, span sasscommon.FileSpan, value *Interpolation, children []Statement) *AtRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &AtRule{
		parent: NewParentStatement(c),
		Name:   name,
		Value:  value,
		span:   span,
	}
}

func (r *AtRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *AtRule) IsStatement()                       {}
func (r *AtRule) IsSassNode()                        {}
func (r *AtRule) IsAstNode()                         {}
func (r *AtRule) GetChildren() []Statement           { return r.parent.Children }
func (r *AtRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *AtRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@")
	nameStr, err := r.Name.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(nameStr)
	if r.Value != nil {
		builder.WriteString(" ")
		valStr, err := r.Value.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(valStr)
	}
	if r.parent.Children == nil {
		builder.WriteString(";")
	} else {
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
	}
	return builder.String(), nil
}
