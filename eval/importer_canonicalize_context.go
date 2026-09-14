// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"net/url"
)

// dart-source: lib/src/importer/canonicalize_context.dart
//
// Internal in Dart (marked @internal there): contextual information threaded
// through importer canonicalization. Dart carries it in a Zone; Go passes the
// struct explicitly from the import cache into Canonicalize.
//
// CanonicalizeContext holds contextual information used by importers'
// canonicalize method.
type CanonicalizeContext struct {
	// Whether this is evaluating an @import rule.
	FromImport bool

	// URL of the stylesheet that triggered the current load, when known. Only
	// populated when the containing stylesheet has a canonical URL and the
	// load URL is relative or uses a non-canonical scheme, so canonical URLs
	// keep one meaning regardless of load site.
	containingURL *url.URL

	// Whether ContainingURL has been read. A read marks the in-flight
	// canonicalization as context-sensitive, which tells the import cache not
	// to share the result across load sites.
	wasContainingURLAccessed bool
}

// NewCanonicalizeContext creates a CanonicalizeContext for a load from
// containingURL, recording whether the load runs under an @import rule.
func NewCanonicalizeContext(containingURL *url.URL, fromImport bool) *CanonicalizeContext {
	return &CanonicalizeContext{
		containingURL: containingURL,
		FromImport:    fromImport,
	}
}

// ContainingURL returns the URL of the stylesheet that contains the current load.
//
// This marks the containing URL as accessed, which is used to determine whether
// the canonicalize result is cacheable: once read, the result is treated as
// load-site specific and is no longer shared across the whole importer chain.
func (ctx *CanonicalizeContext) ContainingURL() *url.URL {
	ctx.wasContainingURLAccessed = true
	return ctx.containingURL
}

// WasContainingURLAccessed returns whether ContainingURL has been accessed.
//
// This is used to determine whether the canonicalize result is cacheable: a
// result that observed its load site must not be reused for other sites.
func (ctx *CanonicalizeContext) WasContainingURLAccessed() bool {
	return ctx.wasContainingURLAccessed
}

// ContainingURLWithoutMarking returns ContainingURL without marking it accessed.
//
// Internal checks that must not affect cacheability use this; importer logic
// that genuinely depends on the load site reads ContainingURL instead.
func (ctx *CanonicalizeContext) ContainingURLWithoutMarking() *url.URL {
	return ctx.containingURL
}

// WithFromImport runs callback in a context with the given FromImport setting.
//
// The previous flag is restored afterwards, so a nested @import-scoped lookup
// cannot leak import-only resolution into the surrounding @use/@forward load.
// Dart additionally asserts the context is the ambient zone value; Go has no
// zones, so the context arrives as an explicit parameter instead.
//
// Matches Dart: CanonicalizeContext.withFromImport — canonicalize_context.dart
func WithFromImport[T any](ctx *CanonicalizeContext, fromImport bool, callback func() T) T {
	oldFromImport := ctx.FromImport
	ctx.FromImport = fromImport
	defer func() { ctx.FromImport = oldFromImport }()
	return callback()
}
