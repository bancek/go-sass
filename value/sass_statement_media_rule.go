// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/media_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// MediaRule is a @media rule.
//
// It nests its children under a media query that is resolved once any
// interpolation it carries has been evaluated.
type MediaRule struct {
	parent ParentStatement
	// Query is the interpolated media query selecting the target platforms.
	// It is parsed into concrete queries only after interpolation resolves.
	Query *Interpolation
	span  sasscommon.FileSpan
}

// NewMediaRule creates a @media rule applying children under query.
func NewMediaRule(query *Interpolation, children []Statement, span sasscommon.FileSpan) *MediaRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &MediaRule{
		parent: NewParentStatement(c),
		Query:  query,
		span:   span,
	}
}

func (r *MediaRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *MediaRule) IsStatement()                       {}
func (r *MediaRule) IsSassNode()                        {}
func (r *MediaRule) IsAstNode()                         {}
func (r *MediaRule) GetChildren() []Statement           { return r.parent.Children }
func (r *MediaRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *MediaRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@media ")
	queryStr, err := r.Query.String()
	if err != nil {
		return "", err
	}
	builder.WriteString(queryStr)
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
