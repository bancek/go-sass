// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/extend_rule.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// ExtendRule is an @extend rule.
//
// It gives the enclosing selector all the styling of the extended selector.
type ExtendRule struct {
	// Selector is the interpolation for the selector being extended.
	Selector *Interpolation
	// IsOptional reports whether a missing match is tolerated.
	// A non-optional extension errors when it matches no selectors.
	IsOptional bool
	span       sasscommon.FileSpan
}

// NewExtendRule creates an @extend rule for selector.
// When optional is true, the extension silently matches nothing instead of
// reporting an error.
func NewExtendRule(selector *Interpolation, span sasscommon.FileSpan, optional bool) *ExtendRule {
	return &ExtendRule{Selector: selector, IsOptional: optional, span: span}
}

func (r *ExtendRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ExtendRule) IsStatement()                       {}
func (r *ExtendRule) IsSassNode()                        {}
func (r *ExtendRule) IsAstNode()                         {}
func (r *ExtendRule) String() (string, error) {
	selStr, err := r.Selector.String()
	if err != nil {
		return "", err
	}
	if r.IsOptional {
		return "@extend " + selStr + " !optional;", nil
	}
	return "@extend " + selStr + ";", nil
}
