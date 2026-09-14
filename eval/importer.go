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

// dart-source: lib/src/importer.dart

// NoOp is an importer that never imports any stylesheets.
//
// It backs stylesheets that support no relative loads, such as those compiled
// from plain strings rather than files: every canonicalization and load
// attempt reports "not recognized".
//
// Matches Dart: Importer.noOp
var NoOp Importer = &NoOpImporter{}

// Importer resolves Sass load URLs to the contents of Sass files.
//
// Hosts implement this interface to teach the compiler new URL schemes or
// lookup strategies. Provide a human-readable String on implementations, in
// the same spirit as the filesystem importer reporting its load path.
//
// The Go port carries only the synchronous half of Dart's importer pair:
// Canonicalize and Load run inline and are usable from every compile entry
// point, so a synchronous importer is always preferred where one suffices.
// Custom types should embed or extend the built-in importer structs rather
// than reimplementing the contract from scratch where possible.
type Importer interface {
	// Canonicalize returns the canonical form of u, or nil when this importer
	// does not recognize u.
	//
	// Canonical URLs must be absolute (a file: URL is encouraged for
	// on-disk stylesheets) and stable: repeated canonicalizations of the
	// same URL agree, canonicalizing a canonical URL returns it unchanged,
	// and one canonical URL always denotes the same stylesheet even across
	// importers. When several on-disk candidates match (partial prefix,
	// added extension, index fallback), the importer reports the ambiguity
	// as an error instead of guessing; when nothing matches it returns nil.
	//
	// The ctx parameter provides contextual information about the current load,
	// including whether it's from an @import rule and the containing URL.
	Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error)

	// Load loads the stylesheet at canonicalURL and returns its contents.
	//
	// The URL comes from an earlier Canonicalize call on the same importer.
	// A nil result means the stylesheet cannot be found here; a load failure
	// for a URL this importer owns surfaces as an error that the load site
	// wraps with the importing rule's span and trace.
	Load(canonicalURL *url.URL) (*ImporterResult, error)

	// CouldCanonicalize returns whether this importer could potentially
	// canonicalize url to canonicalUrl.
	//
	// This must stay cheap and must never touch the filesystem. Over-reporting
	// is allowed where precision would be expensive, but under-reporting is
	// not: a false negative would leave a stale cache entry behind when the
	// file graph changes.
	CouldCanonicalize(url *url.URL, canonicalURL *url.URL) bool

	// IsNonCanonicalScheme returns whether the given scheme is a non-canonical
	// scheme for this importer.
	//
	// An importer never returns a non-canonical scheme from Canonicalize; in
	// exchange, the containing URL stays visible while canonicalizing absolute
	// URLs with that scheme, so resolution can depend on the load site. The
	// answer must be stable per scheme and cheap to compute.
	IsNonCanonicalScheme(scheme string) bool

	// ModificationTime returns the time that the Sass file at canonicalURL was
	// last modified.
	//
	// The URL comes from an earlier Canonicalize call on the same importer.
	// The default reports the current time, which marks the stylesheet as
	// always stale; precise times let the compiler skip needless recompiles.
	//
	// If an error is returned, the caller will use the current time as a
	// fallback (matching Dart's behavior of catching exceptions and defaulting
	// to DateTime.now()).
	ModificationTime(canonicalURL *url.URL) (time.Time, error)
}
