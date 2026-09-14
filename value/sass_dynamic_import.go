// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/import/dynamic.dart

import (
	"net/url"

	"github.com/bancek/go-sass/sasscommon"
)

// DynamicImport is an import that will load a Sass file at runtime.
//
// Matches Dart: DynamicImport
type DynamicImport struct {
	// urlString is the URL of the file to import, kept as a string so a
	// leading ./ stays visible for Node Sass imports. Relative URLs resolve
	// against the containing file.
	//
	// Matches Dart: DynamicImport.urlString (internal).
	urlString string
	span      sasscommon.FileSpan
}

// NewDynamicImport creates a runtime Sass import.
//
// Matches Dart: DynamicImport.new
func NewDynamicImport(urlString string, span sasscommon.FileSpan) *DynamicImport {
	return &DynamicImport{urlString: urlString, span: span}
}

// URL returns the parsed URL. This never returns nil, matching Dart's
// Uri.parse() which always produces a Uri (never null).
//
// Matches Dart: DynamicImport.url
func (i *DynamicImport) URL() *url.URL {
	parsed, err := url.Parse(i.urlString)
	if err != nil {
		// url.Parse returns a partially parsed URL even on error, so we
		// return it rather than nil to match Dart's non-null guarantee.
		return parsed
	}
	return parsed
}

func (i *DynamicImport) URLSpan() (sasscommon.FileSpan, error) { return i.span, nil }
func (i *DynamicImport) Span() (sasscommon.FileSpan, error)    { return i.span, nil }
func (i *DynamicImport) IsImport()                             {}
func (i *DynamicImport) isSassDependency()                     {}
func (i *DynamicImport) IsSassNode()                           {}
func (i *DynamicImport) IsAstNode()                            {}

// String returns a quoted CSS string representation.
//
// Matches Dart: DynamicImport.toString()
func (i *DynamicImport) String() (string, error) {
	return QuoteText(i.urlString), nil
}
