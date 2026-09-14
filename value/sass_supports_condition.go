// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/sass/supports_condition.dart

// SupportsCondition is a test selecting which browsers a @supports rule
// targets.
//
// ToInterpolation flattens the condition back into source-equivalent text
// for resolution; WithSpan copies the condition under a new span.
type SupportsCondition interface {
	SassNode
	ToInterpolation() (*Interpolation, error)
	WithSpan(span sasscommon.FileSpan) SupportsCondition
	IsSupportsCondition()
	String() (string, error)
}
