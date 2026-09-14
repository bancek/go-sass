// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/silent_comment.dart

import (
	"strings"
	"unicode"

	"github.com/bancek/go-sass/sasscommon"
)

// SilentComment is a silent Sass-style comment.
//
// It is dropped during compilation and never reaches the output CSS, but it
// can carry SassDoc documentation read through DocComment.
type SilentComment struct {
	// Text is the comment source, including the comment markers.
	Text string
	span sasscommon.FileSpan
}

// NewSilentComment creates a silent comment holding text.
func NewSilentComment(text string, span sasscommon.FileSpan) *SilentComment {
	return &SilentComment{Text: text, span: span}
}

// DocComment returns the subset of lines in Text that are marked as part of
// the documentation comments by beginning with '///'.
func (c *SilentComment) DocComment() *string {
	var builder strings.Builder
	for line := range strings.SplitSeq(c.Text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "///") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "///")
		rest = strings.TrimPrefix(rest, " ")
		builder.WriteString(rest)
		builder.WriteByte('\n')
	}
	comment := strings.TrimRightFunc(builder.String(), unicode.IsSpace)
	if comment == "" {
		return nil
	}
	return &comment
}

func (c *SilentComment) Span() (sasscommon.FileSpan, error) { return c.span, nil }
func (c *SilentComment) IsStatement()                       {}
func (c *SilentComment) IsSassNode()                        {}
func (c *SilentComment) IsAstNode()                         {}
func (c *SilentComment) String() string                     { return c.Text }
