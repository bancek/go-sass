// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/content_rule.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ContentRule is a @content rule.
//
// It appears inside a mixin body and includes the statement-level content
// block the caller passed to the mixin invocation.
type ContentRule struct {
	// Arguments are the values passed to the content block.
	// An empty invocation means plain @content with no arguments.
	Arguments *ArgumentList
	span      sasscommon.FileSpan
}

// NewContentRule creates a @content rule forwarding arguments to the caller's
// content block.
func NewContentRule(arguments *ArgumentList, span sasscommon.FileSpan) *ContentRule {
	return &ContentRule{Arguments: arguments, span: span}
}

func (r *ContentRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ContentRule) IsStatement()                       {}
func (r *ContentRule) IsSassNode()                        {}
func (r *ContentRule) IsAstNode()                         {}

// String returns the CSS representation of this content rule.
//
// Matches Dart: ContentRule.toString
func (r *ContentRule) String() (string, error) {
	if r.Arguments.IsEmpty() {
		return "@content;", nil
	}
	argsStr, err := r.Arguments.String()
	if err != nil {
		return "", err
	}
	return "@content" + argsStr + ";", nil
}
