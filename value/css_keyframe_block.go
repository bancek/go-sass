// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/css/keyframe_block.dart

// CssKeyframeBlock is a block within a @keyframes rule.
// For example, "10% {opacity: 0.5}".
//
// Matches Dart: CssKeyframeBlock.
type CssKeyframeBlock interface {
	CssParentNode
	// Selector returns the keyframe selector list (offsets).
	Selector() sasscommon.CssValue[[]string]
}
