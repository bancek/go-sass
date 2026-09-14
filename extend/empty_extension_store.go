// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package extend

// dart-source: lib/src/extend/empty_extension_store.dart

import (
	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// EmptyExtensionStore is a store that contains no extensions and can never
// gain any. Modules without @extend use it so downstream merging and
// cloning have a uniform target; every mutating method reports misuse
// instead.
type EmptyExtensionStore struct{}

// IsEmpty always reports true.
func (e *EmptyExtensionStore) IsEmpty() bool { return true }

// SimpleSelectors returns no selectors.
func (e *EmptyExtensionStore) SimpleSelectors() []value.SimpleSelector {
	return nil
}

// ExtensionsWhereTarget returns no extensions.
func (e *EmptyExtensionStore) ExtensionsWhereTarget(func(value.SimpleSelector) bool) []Extension {
	return nil
}

// AddSelector always fails: selectors cannot be added to the empty store.
func (e *EmptyExtensionStore) AddSelector(sel *value.SelectorList, mediaContext []*value.CssMediaQuery) (*box.Box[*value.SelectorList], error) {
	return nil, &sasscommon.UnsupportedError{Message: "addSelector() can't be called for a const ExtensionStore."}
}

// AddExtension always fails: extensions cannot be added to the empty store.
func (e *EmptyExtensionStore) AddExtension(extender *value.SelectorList, target value.SimpleSelector, extend *value.ExtendRule, mediaContext []*value.CssMediaQuery) error {
	return &sasscommon.UnsupportedError{Message: "addExtension() can't be called for a const ExtensionStore."}
}

// AddExtensions always fails: stores cannot be merged into the empty store.
func (e *EmptyExtensionStore) AddExtensions(extenders []ExtensionStore) error {
	return &sasscommon.UnsupportedError{Message: "addExtensions() can't be called for a const ExtensionStore."}
}

// Clone returns a fresh empty store and an empty selector map, matching the
// const empty clone in Dart.
func (e *EmptyExtensionStore) Clone() (ExtensionStore, map[*value.SelectorList]*box.Box[*value.SelectorList]) {
	return &EmptyExtensionStore{}, make(map[*value.SelectorList]*box.Box[*value.SelectorList])
}
