// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package compile

// dart-source: lib/src/compile_result.dart

import (
	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sourcemap"
	"github.com/bancek/go-sass/value"
)

// CompileResult is the result of compiling a Sass document to CSS, along with
// metadata about the compilation process.
//
// Dart equivalent: class CompileResult { EvaluateResult _evaluate; SerializeResult _serialize; ... }
type CompileResult struct {
	evaluateResult  *eval.EvaluateResult
	serializeResult *value.SerializeResult
}

// NewCompileResult pairs an evaluation result with a serialization result.
//
// Matches Dart: the CompileResult constructor taking an EvaluateResult and
// a SerializeResult.
func NewCompileResult(evaluateResult *eval.EvaluateResult, serializeResult *value.SerializeResult) *CompileResult {
	return &CompileResult{
		evaluateResult:  evaluateResult,
		serializeResult: serializeResult,
	}
}

// CSS returns the compiled CSS.
func (r *CompileResult) CSS() string {
	return r.serializeResult.CSS
}

// SourceMap returns the source map indicating how the source files map to CSS,
// or nil if source mapping was disabled for this compilation.
//
// Matches Dart: CompileResult.sourceMap
func (r *CompileResult) SourceMap() *sourcemap.SingleMapping {
	return r.serializeResult.SourceMap
}

// LoadedUrls returns the canonical URLs of all stylesheets loaded during compilation.
//
// Matches Dart: CompileResult.loadedUrls
func (r *CompileResult) LoadedUrls() *orderedset.LinkedSet[string] {
	return r.evaluateResult.LoadedUrls
}
