// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/style_rule.dart

// CssStyleRule is a plain CSS style rule.
//
// The selector may still hold placeholders pre-extension. OriginalSelector
// preserves the pre-extension selector and FromPlainCss marks plain-CSS
// inputs (internal metadata in Dart). Matches Dart: CssStyleRule.
type CssStyleRule interface {
	CssParentNode
	// Selector returns the current (possibly extended) selector.
	Selector() *SelectorList
	// OriginalSelector returns the pre-extension selector.
	OriginalSelector() *SelectorList
	// FromPlainCss reports whether the rule came from plain CSS.
	FromPlainCss() bool
}
