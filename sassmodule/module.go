// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassmodule

// dart-source: lib/src/module.dart

import (
	"net/url"

	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// Module is the interface for a Sass module: the compiled result of one
// stylesheet, exposing its members, CSS, and extension store.
//
// Identity is by reference — modules compare by pointer, never by value.
//
// Matches Dart: Module
type Module interface {
	// URL returns the canonical URL of the module's source file, or an
	// empty string when the module was loaded from a string with no URL.
	URL() (string, error)
	// Upstream returns the modules this module uses, in @use order. They
	// feed CSS collation, not member lookup.
	Upstream() []Module
	// Variables returns the module's variables, keyed without the $.
	Variables() orderedmap.Map[string, value.Value]
	// VariableNodes returns the nodes where each variable in Variables was
	// defined. Nodes (not spans) are stored so spans are only manufactured
	// when actually needed; keys always match Variables.
	VariableNodes() orderedmap.Map[string, sasscommon.AstNode]
	// Functions returns the module's functions, each stored under its own
	// name.
	Functions() orderedmap.Map[string, sasscallable.Callable]
	// Mixins returns the module's mixins, each stored under its own name.
	Mixins() orderedmap.Map[string, sasscallable.Callable]
	// ExtensionStore returns the module's extensions, able to update the
	// CSS tree in place for downstream @extend.
	ExtensionStore() extend.ExtensionStore
	// CSS returns the module's CSS tree.
	CSS() (*value.CssStylesheet, error)
	// PreModuleComments maps upstream modules to loud comments written in
	// this module that must be emitted before the given module.
	PreModuleComments() map[Module][]value.CssComment
	// TransitivelyContainsCss reports whether this module or any upstream
	// module contains CSS.
	TransitivelyContainsCss() bool
	// TransitivelyContainsExtensions reports whether this module or any
	// upstream module contains @extend rules.
	TransitivelyContainsExtensions() bool
	// SetVariable sets the variable named name to val, recording
	// nodeWithSpan for error spans. It takes the node rather than a span
	// so spans are only manufactured when needed. It fails when the module
	// defines no such variable.
	SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) error
	// VariableIdentity returns an opaque token that compares equal across
	// two modules exactly when both expose identical definitions of name,
	// per the Sass spec. It backs @forward conflict detection.
	VariableIdentity(name string) (any, error)
	// CouldHaveBeenConfigured reports whether the module exposes any of
	// the named variables that could have been configured when it loaded.
	CouldHaveBeenConfigured(variables map[string]struct{}) bool
	// CloneCss returns a copy of the module with a fresh CSS tree and
	// extension store, sharing the member definitions.
	CloneCss() (Module, error)
}

// dart-source: lib/src/module/built_in.dart
// BuiltInModule is a module provided by Sass, available under the special
// sass: URL space.
//
// Unlike evaluated modules it has no source: no upstream, no spans, no CSS,
// and no extensions. Its URL is always "sass:"+name.
//
// Matches Dart: BuiltInModule
type BuiltInModule struct {
	url       string
	variables *orderedmap.LinkedMap[string, value.Value]
	functions *orderedmap.LinkedMap[string, sasscallable.Callable]
	mixins    *orderedmap.LinkedMap[string, sasscallable.Callable]
}

// NewBuiltInModule creates a new BuiltInModule with the given name and
// optional members.
//
// Functions and mixins are indexed under their own names; a nil variables
// map becomes an empty one. The module URL is always "sass:"+name.
//
// Matches Dart: BuiltInModule constructor (name, functions, mixins, variables)
func NewBuiltInModule(name string, functions []sasscallable.Callable, mixins []sasscallable.Callable, variables *orderedmap.LinkedMap[string, value.Value]) *BuiltInModule {
	fnMap := orderedmap.New[string, sasscallable.Callable]()
	for _, fn := range functions {
		fnMap.Put(fn.Name(), fn)
	}
	mixinMap := orderedmap.New[string, sasscallable.Callable]()
	for _, m := range mixins {
		mixinMap.Put(m.Name(), m)
	}
	varMap := variables
	if varMap == nil {
		varMap = orderedmap.New[string, value.Value]()
	}
	return &BuiltInModule{
		url:       "sass:" + name,
		variables: varMap,
		functions: fnMap,
		mixins:    mixinMap,
	}
}

func (m *BuiltInModule) URL() (string, error)                           { return m.url, nil }
func (m *BuiltInModule) Upstream() []Module                             { return nil }
func (m *BuiltInModule) Variables() orderedmap.Map[string, value.Value] { return m.variables }

// Dart returns const {} (compile-time singleton) for the empty node map.
// Cache a single empty map so every built-in module shares one.
var emptyAstNodeMap = orderedmap.New[string, sasscommon.AstNode]()

// VariableNodes is always empty: built-in variables have no Sass source,
// so there are no definition nodes to point at.
func (m *BuiltInModule) VariableNodes() orderedmap.Map[string, sasscommon.AstNode] {
	return emptyAstNodeMap
}
func (m *BuiltInModule) Functions() orderedmap.Map[string, sasscallable.Callable] { return m.functions }
func (m *BuiltInModule) Mixins() orderedmap.Map[string, sasscallable.Callable]    { return m.mixins }

// ExtensionStore is always empty: built-ins define no style rules to extend.
func (m *BuiltInModule) ExtensionStore() extend.ExtensionStore { return &extend.EmptyExtensionStore{} }

// CSS is always an empty stylesheet at the module's sass: URL: built-ins
// contribute members only, never CSS.
func (m *BuiltInModule) CSS() (*value.CssStylesheet, error) {
	parsed, err := url.Parse(m.url)
	if err != nil {
		return nil, err
	}
	return value.NewCssStylesheetEmpty(parsed), nil
}
func (m *BuiltInModule) PreModuleComments() map[Module][]value.CssComment { return nil }
func (m *BuiltInModule) TransitivelyContainsCss() bool                    { return false }
func (m *BuiltInModule) TransitivelyContainsExtensions() bool             { return false }

// SetVariable always fails: unknown names are undefined, and built-in
// variables are immutable from Sass.
func (m *BuiltInModule) SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) error {
	if !m.variables.Has(name) {
		return sasscommon.NewSassScriptException("Undefined variable.", nil)
	}
	return sasscommon.NewSassScriptException("Cannot modify built-in variable.", nil)
}

// VariableIdentity returns the module itself: every reference to a
// built-in variable resolves to the same definition by construction.
func (m *BuiltInModule) VariableIdentity(name string) (any, error) {
	if !m.variables.Has(name) {
		panic("assertion failed: variable not found")
	}
	return m, nil
}

// AddFunction registers fn under its own name. It supports populating the
// module after construction; normal evaluation never calls it.
func (m *BuiltInModule) AddFunction(fn sasscallable.Callable) {
	m.functions.Put(fn.Name(), fn)
}

// AddMixin registers mx under its own name. Like AddFunction, it supports
// post-construction setup rather than evaluation.
func (m *BuiltInModule) AddMixin(mx sasscallable.Callable) {
	m.mixins.Put(mx.Name(), mx)
}

// CouldHaveBeenConfigured always returns false: built-in variables are
// never configurable through @use ... with.
func (m *BuiltInModule) CouldHaveBeenConfigured(variables map[string]struct{}) bool {
	return false
}

// CloneCss returns the module itself: with no CSS or extensions there is
// nothing to copy.
func (m *BuiltInModule) CloneCss() (Module, error) { return m, nil }
