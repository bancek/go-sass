// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package extend

// dart-source: lib/src/extend/extension_store.dart (interface only; see
// default_extension_store.go for the DefaultExtensionStore subclass)

import (
	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/value"
)

// ExtensionStore tracks selectors and extensions, and applies the latter to
// the former. It is the shared contract satisfied by the tracking
// implementation and the empty sentinel: callers register selectors as they
// are defined, register @extend rules as they appear, merge downstream
// stores across module boundaries, and clone the store when a stylesheet
// must be copied without disturbing the original compilation.
type ExtensionStore interface {
	// IsEmpty reports whether the store holds no extensions.
	IsEmpty() bool
	// SimpleSelectors returns every simple selector seen so far, including
	// ones added by downstream extensions.
	SimpleSelectors() []value.SimpleSelector
	// ExtensionsWhereTarget returns the mandatory extensions whose targets
	// satisfy callback, unmerging merged extensions so only base rules come
	// back.
	ExtensionsWhereTarget(func(value.SimpleSelector) bool) []Extension
	// AddSelector extends sel with the registered extensions and returns a
	// sealed box for the result. The box updates automatically if later
	// extensions apply to it. MediaContext records where the selector was
	// defined, or nil at the top level.
	AddSelector(sel *value.SelectorList, mediaContext []*value.CssMediaQuery) (*box.Box[*value.SelectorList], error)
	// AddExtension registers extender {@extend target} for the given rule,
	// applying it to already-registered selectors and extenders. MediaContext
	// restricts the extension to that media context; nil means any context.
	AddExtension(extender *value.SelectorList, target value.SimpleSelector, extend *value.ExtendRule, mediaContext []*value.CssMediaQuery) error
	// AddExtensions folds every extension from extenders into the receiver,
	// extending already-present selectors and extenders but never extending
	// the incoming extensions against each other.
	AddExtensions(extenders []ExtensionStore) error
	// Clone copies the store and maps each original selector list to its
	// sealed copy, so a cloned stylesheet's style rules stay wired to the
	// cloned store.
	Clone() (ExtensionStore, map[*value.SelectorList]*box.Box[*value.SelectorList])
}

// mediaQueriesEqual reports whether two media-query contexts match query by
// query, treating two nil/empty contexts as equal by length.
func mediaQueriesEqual(a, b []*value.CssMediaQuery) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !value.MediaQueriesEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
