// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/import_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ImportRule is an @import rule.
//
// It holds one or more static or dynamic imports processed in source order.
type ImportRule struct {
	// Imports are the targets imported by this statement.
	Imports []Import
	span    sasscommon.FileSpan
}

// NewImportRule creates an @import rule for the given imports.
func NewImportRule(imports []Import, span sasscommon.FileSpan) *ImportRule {
	imps := make([]Import, len(imports))
	copy(imps, imports)
	return &ImportRule{Imports: imps, span: span}
}

func (r *ImportRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ImportRule) IsStatement()                       {}
func (r *ImportRule) IsSassNode()                        {}
func (r *ImportRule) IsAstNode()                         {}
func (r *ImportRule) String() (string, error) {
	parts := make([]string, len(r.Imports))
	for i, imp := range r.Imports {
		s, err := imp.String()
		if err != nil {
			return "", err
		}
		parts[i] = s
	}
	return "@import " + strings.Join(parts, ", ") + ";", nil
}
