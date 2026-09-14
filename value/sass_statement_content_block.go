// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/content_block.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ContentBlock is an anonymous block of code passed to a mixin invocation
// and run for a ContentRule inside that mixin.
//
// It behaves like a callable declaration with the fixed name "@content".
type ContentBlock struct {
	base *callableDeclaration
}

// NewContentBlock creates a ContentBlock with the given @content parameters,
// body children, and span coverage.
func NewContentBlock(parameters *ParameterList, children []Statement, span sasscommon.FileSpan) *ContentBlock {
	base := newCallableDeclaration("@content", parameters, children, span, nil)
	return &ContentBlock{base: base}
}

// Declaration returns the block as its callable-declaration view.
func (r *ContentBlock) Declaration() CallableDeclaration   { return r }
func (r *ContentBlock) Name() string                       { return r.base.Name() }
func (r *ContentBlock) OriginalName() string               { return r.base.OriginalName() }
func (r *ContentBlock) Parameters() *ParameterList         { return r.base.Parameters() }
func (r *ContentBlock) Span() (sasscommon.FileSpan, error) { return r.base.Span() }
func (r *ContentBlock) Comment() *SilentComment            { return r.base.Comment() }
func (r *ContentBlock) IsStatement()                       {}
func (r *ContentBlock) IsSassNode()                        {}
func (r *ContentBlock) IsAstNode()                         {}
func (r *ContentBlock) GetChildren() []Statement           { return r.base.GetChildren() }
func (r *ContentBlock) HasDeclarations() bool              { return r.base.HasDeclarations() }

func (r *ContentBlock) String() (string, error) {
	var builder strings.Builder
	if !r.base.parameters.IsEmpty() {
		builder.WriteString(" using (")
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
