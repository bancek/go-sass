// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"fmt"
	"net/url"
)

// dart-source: lib/src/importer/result.dart
// ImporterResult is the result of importing a Sass stylesheet, as returned by
// Importer.Load.
//
// The deprecated indented flag and isIndented getter from Dart have no Go
// counterpart: syntax selection always travels in the Syntax field.
type ImporterResult struct {
	// The contents of the stylesheet.
	Contents string

	// The syntax to use to parse the stylesheet.
	Syntax Syntax

	// An absolute, browser-accessible URL indicating the resolved location of
	// the imported stylesheet. If no URL was originally supplied, a data: URL
	// is generated automatically from Contents.
	//
	// Matches Dart: ImporterResult._sourceMapUrl
	sourceMapURL *url.URL
}

// NewImporterResult creates an ImporterResult holding contents parsed with
// syntax and attributed to sourceMapURL in source maps.
//
// The source-map URL must be absolute when supplied. Dart's deprecated
// indented flag has no counterpart here: callers always pass the syntax
// explicitly.
func NewImporterResult(contents string, syntax Syntax, sourceMapURL *url.URL) (*ImporterResult, error) {
	if sourceMapURL != nil && sourceMapURL.Scheme == "" {
		return nil, fmt.Errorf("sourceMapURL must be absolute")
	}
	return &ImporterResult{
		Contents:     contents,
		Syntax:       syntax,
		sourceMapURL: sourceMapURL,
	}, nil
}

// SourceMapURL returns the source map URL, auto-generating a data: URL
// from Contents if none was provided.
//
// The synthesized fallback encodes the stylesheet text itself (UTF-8
// text/plain), so every imported stylesheet stays attributable even when the
// importer supplies no URL.
//
// Matches Dart: ImporterResult.sourceMapUrl
func (r *ImporterResult) SourceMapURL() *url.URL {
	if r.sourceMapURL != nil {
		return r.sourceMapURL
	}
	// Dart: Uri.dataFromString defaults to text/plain MIME type — result.dart
	u, _ := url.Parse("data:text/plain;charset=utf-8," + url.PathEscape(r.Contents))
	return u
}
