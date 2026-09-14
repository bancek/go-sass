// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/declaration.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// CssDeclaration is a plain CSS declaration (name: value pair).
//
// Matches Dart: CssDeclaration. Custom-property values are unparsed strings
// while SassScript values hold parsed values; ValueSpanForMap tracks the
// declaration site for source maps.
type CssDeclaration interface {
	CssNode
	// Name returns the declaration name.
	Name() sasscommon.CssValue[string]
	// Value returns the declaration value.
	Value() sasscommon.CssValue[Value]
	// ValueSpanForMap returns the span emitted to source maps.
	ValueSpanForMap() sasscommon.FileSpan
	// IsCustomProperty reports whether the name starts with "--".
	IsCustomProperty() bool
	// ParsedAsSassScript reports whether the value was parsed as SassScript.
	ParsedAsSassScript() bool
}
