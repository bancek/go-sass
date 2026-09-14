// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/value"
)

// dart-source: lib/src/visitor/async_evaluate.dart (EvaluateResult record;
// re-exported from the generated evaluate.dart)

// EvaluateResult is the result of evaluating a Sass stylesheet: the frozen
// CSS tree plus the canonical URLs of every stylesheet loaded during
// compilation.
//
// Matches Dart: the EvaluateResult record ({CssStylesheet stylesheet, Set<Uri>
// loadedUrls}). Stylesheet is the combined output tree; LoadedUrls collects
// every canonical URL seen during evaluation. Go holds the URLs in an
// insertion-ordered set of strings rather than Dart's URI set.
type EvaluateResult struct {
	// Stylesheet is the CSS syntax tree produced by evaluation.
	Stylesheet *value.CssStylesheet

	// LoadedUrls holds the canonical URLs of all stylesheets loaded during
	// compilation.
	LoadedUrls *orderedset.LinkedSet[string]
}
