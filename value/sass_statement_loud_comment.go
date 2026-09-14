// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/statement/loud_comment.dart

// LoudComment is a loud CSS-style comment.
//
// It survives evaluation and is emitted into the compiled CSS.
type LoudComment struct {
	// Text is the interpolated comment text, including the comment markers.
	Text *Interpolation
}

// NewLoudComment creates a loud comment preserving text in the output CSS.
func NewLoudComment(text *Interpolation) *LoudComment {
	return &LoudComment{Text: text}
}

func (c *LoudComment) Span() (sasscommon.FileSpan, error) { return c.Text.Span() }
func (c *LoudComment) IsStatement()                       {}
func (c *LoudComment) IsSassNode()                        {}
func (c *LoudComment) IsAstNode()                         {}
func (c *LoudComment) String() (string, error)            { return c.Text.String() }
