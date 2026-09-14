// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/supports_rule.dart

// CssSupportsRule is a plain CSS @supports rule.
//
// Matches Dart: CssSupportsRule.
type CssSupportsRule interface {
	CssParentNode
	// Condition returns the @supports condition.
	Condition() sasscommon.CssValue[string]
}
