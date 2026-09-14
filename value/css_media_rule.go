// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/media_rule.dart

// CssMediaRule is a plain CSS @media rule.
//
// Matches Dart: CssMediaRule.
type CssMediaRule interface {
	CssParentNode
	// Queries returns the rule's media queries.
	Queries() []CssMediaQuery
}
