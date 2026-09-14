// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/import.dart

// CssImport is a plain CSS @import.
//
// Matches Dart: CssImport.
type CssImport interface {
	CssNode
	// URL returns the imported URL.
	URL() sasscommon.CssValue[string]
	// Modifiers returns media/supports/layer modifiers, or nil when absent.
	Modifiers() *sasscommon.CssValue[string]
}
