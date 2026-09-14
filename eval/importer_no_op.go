// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"net/url"
	"time"
)

// dart-source: lib/src/importer/no_op.dart
// NoOpImporter is an importer that never imports any stylesheets.
//
// It backs stylesheets that support no relative loads, such as those compiled
// from plain strings: canonicalization and loading both report "not
// recognized" so the import cache moves on to the next importer.
type NoOpImporter struct{}

// Canonicalize always reports the URL as unrecognized, returning nil without
// an error.
func (i *NoOpImporter) Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error) {
	return nil, nil
}

// Load always reports the stylesheet as not found here, returning nil without
// an error.
func (i *NoOpImporter) Load(canonicalURL *url.URL) (*ImporterResult, error) {
	return nil, nil
}

// CouldCanonicalize always returns false: no URL can resolve through this
// importer, so watch-mode invalidation never needs to consult it.
func (i *NoOpImporter) CouldCanonicalize(url *url.URL, canonicalURL *url.URL) bool {
	return false
}

// IsNonCanonicalScheme always returns false: this importer owns no schemes at
// all.
func (i *NoOpImporter) IsNonCanonicalScheme(scheme string) bool {
	return false
}

// ModificationTime reports the current time, marking any URL as always stale.
// In practice this importer never produces a canonical URL, so the value is
// only a fallback.
func (i *NoOpImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	return time.Now(), nil
}

// String returns a human-readable description of this importer.
func (i *NoOpImporter) String() string {
	return "(unknown)"
}
