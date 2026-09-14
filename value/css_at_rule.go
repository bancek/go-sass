// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/at_rule.dart

// CssAtRule is an unknown plain CSS at-rule.
//
// Both the frozen node and ModifiableCssAtRule implement this. Name and
// Value carry spans; IsChildless distinguishes `@foo;` from `@foo {}` (an
// empty block is not childless). Matches Dart: CssAtRule.
type CssAtRule interface {
	CssParentNode
	// Name returns the at-rule name.
	Name() sasscommon.CssValue[string]
	// Value returns the at-rule value, or nil when absent.
	Value() *sasscommon.CssValue[string]
	// IsChildless reports whether the rule ends with ";" rather than a block.
	IsChildless() bool
}
