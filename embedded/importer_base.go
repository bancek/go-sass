// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/importer/base.dart

import (
	supf "fmt"
	"net/url"
)

// ImporterBase is the shared base for importers that resolve through the
// host, carrying the dispatcher their requests are sent through.
//
// Dart declares the dispatcher @protected so only subclasses touch it; Go
// exports the field because the host and file importers live as separate
// structs composing (rather than extending) the base, and the field must
// stay reachable across those types.
//
// Matches Dart: abstract base class ImporterBase extends Importer in
// importer/base.dart.
type ImporterBase struct {
	// Dispatcher is the compilation dispatcher importer requests are sent
	// through. It corresponds to Dart's @protected dispatcher field.
	Dispatcher *CompilationDispatcher
}

// ParseAbsoluteURL parses urlStr and returns it, erroring when it is invalid
// or relative (including root-relative URLs, which carry no scheme).
//
// source names the caller for the error text (e.g. "The importer") so a bad
// host URL points at its origin. Dart throws a plain string; Go returns an
// error.
//
// Matches Dart: ImporterBase.parseAbsoluteUrl in importer/base.dart.
func (b *ImporterBase) ParseAbsoluteURL(source, urlStr string) (*url.URL, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, supf.Errorf("%s must return a URL, was \"%s\"", source, urlStr)
	}
	if parsedURL.Scheme == "" {
		return nil, supf.Errorf("%s must return an absolute URL, was \"%s\"", source, parsedURL.String())
	}
	return parsedURL, nil
}
