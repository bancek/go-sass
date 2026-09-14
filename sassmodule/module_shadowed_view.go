// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassmodule

// dart-source: lib/src/module/shadowed_view.dart

import (
	"fmt"

	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// shadowedMap returns a view of m omitting every key in blocklist, or m
// itself when the blocklist overlaps nothing — so shadow-free members cost
// no extra layer.
//
// Matches Dart: ShadowedModuleView._shadowedMap
func shadowedMap[V any](m orderedmap.Map[string, V], blocklist map[string]struct{}) orderedmap.Map[string, V] {
	if len(blocklist) == 0 || !needsBlocklist(m, blocklist) {
		return m
	}
	return orderedmap.NewLimitedMapViewBlocklist(m, blocklist)
}

// needsBlocklist reports whether any of m's keys appear in blocklist: the
// cheap overlap test that decides if a filtered view is even needed. Empty
// maps and empty blocklists never need one.
//
// Matches Dart: ShadowedModuleView._needsBlocklist
func needsBlocklist[V any](m orderedmap.Map[string, V], blocklist map[string]struct{}) bool {
	if blocklist == nil || m.Len() == 0 {
		return false
	}
	for k := range blocklist {
		if m.Has(k) {
			return true
		}
	}
	return false
}

// ShadowedModuleView is a Module that only exposes the inner module's
// members not shadowed away by blocklists of member names.
//
// The environment builds these when @imported forwarded members must hide
// same-named locals. variableNodes is filtered with the variable blocklist
// too, so namespaced node lookups agree with variable lookups. Everything
// besides member visibility — URL, upstream, extensions, CSS, comments,
// transitivity — delegates to the inner module.
//
// Matches Dart: ShadowedModuleView (module/shadowed_view.dart)
type ShadowedModuleView struct {
	module        Module
	variables     orderedmap.Map[string, value.Value]
	variableNodes orderedmap.Map[string, sasscommon.AstNode]
	functions     orderedmap.Map[string, sasscallable.Callable]
	mixins        orderedmap.Map[string, sasscallable.Callable]
}

// NewShadowedModuleView returns a view of module omitting the blocked
// variables, functions, and mixins.
//
// Prefer NewShadowedModuleViewIfNecessary, which skips the wrapper when the
// blocklists shadow nothing.
//
// Matches Dart: ShadowedModuleView constructor
func NewShadowedModuleView(module Module, variables, functions, mixins map[string]struct{}) *ShadowedModuleView {
	return &ShadowedModuleView{
		module:        module,
		variables:     shadowedMap(module.Variables(), variables),
		variableNodes: shadowedMap(module.VariableNodes(), variables),
		functions:     shadowedMap(module.Functions(), functions),
		mixins:        shadowedMap(module.Mixins(), mixins),
	}
}

// NewShadowedModuleViewIfNecessary wraps module in a ShadowedModuleView,
// but returns nil when the blocklists shadow nothing so the caller keeps
// the inner module as-is. (Dart returns null the same way.)
//
// Matches Dart: ShadowedModuleView.ifNecessary
func NewShadowedModuleViewIfNecessary(module Module, variables, functions, mixins map[string]struct{}) *ShadowedModuleView {
	if needsBlocklist(module.Variables(), variables) ||
		needsBlocklist(module.Functions(), functions) ||
		needsBlocklist(module.Mixins(), mixins) {
		return NewShadowedModuleView(module, variables, functions, mixins)
	}
	return nil
}

// URL, Upstream, ExtensionStore, CSS, PreModuleComments, and the
// transitivity queries all delegate to the inner module: shadowing only
// hides members, never CSS or extension behavior.
func (v *ShadowedModuleView) URL() (string, error) { return v.module.URL() }
func (v *ShadowedModuleView) Upstream() []Module   { return v.module.Upstream() }

// Variables, VariableNodes, Functions, and Mixins return the filtered
// views built by NewShadowedModuleView — each either the inner map itself
// or a blocklist view over it.
func (v *ShadowedModuleView) Variables() orderedmap.Map[string, value.Value] {
	return v.variables
}
func (v *ShadowedModuleView) VariableNodes() orderedmap.Map[string, sasscommon.AstNode] {
	return v.variableNodes
}
func (v *ShadowedModuleView) Functions() orderedmap.Map[string, sasscallable.Callable] {
	return v.functions
}
func (v *ShadowedModuleView) Mixins() orderedmap.Map[string, sasscallable.Callable] {
	return v.mixins
}
func (v *ShadowedModuleView) ExtensionStore() extend.ExtensionStore { return v.module.ExtensionStore() }
func (v *ShadowedModuleView) CSS() (*value.CssStylesheet, error)    { return v.module.CSS() }
func (v *ShadowedModuleView) PreModuleComments() map[Module][]value.CssComment {
	return v.module.PreModuleComments()
}
func (v *ShadowedModuleView) TransitivelyContainsCss() bool {
	return v.module.TransitivelyContainsCss()
}
func (v *ShadowedModuleView) TransitivelyContainsExtensions() bool {
	return v.module.TransitivelyContainsExtensions()
}

// SetVariable writes through to the inner module, but fails for names the
// view shadows away — from the outside those variables don't exist.
//
// Matches Dart: ShadowedModuleView.setVariable
func (v *ShadowedModuleView) SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) error {
	if !v.variables.Has(name) {
		return sasscommon.NewSassScriptException("Undefined variable.", nil)
	}
	return v.module.SetVariable(name, val, nodeWithSpan)
}

// VariableIdentity returns the inner module's identity unchanged:
// shadowing only hides members, so a visible name means the same
// definition. The caller must only pass names the view exposes.
//
// Matches Dart: ShadowedModuleView.variableIdentity
func (v *ShadowedModuleView) VariableIdentity(name string) (any, error) {
	if !v.variables.Has(name) {
		panic("assertion failed: variable not found")
	}
	return v.module.VariableIdentity(name)
}

// CouldHaveBeenConfigured delegates unchanged when no blocklist applied,
// and otherwise restricts the query to this view's filtered keys before
// delegating — shadowed-away names must not count as configurable.
//
// Matches Dart: ShadowedModuleView.couldHaveBeenConfigured
func (v *ShadowedModuleView) CouldHaveBeenConfigured(variables map[string]struct{}) bool {
	innerVars := v.module.Variables()
	if v.variables.Len() == innerVars.Len() {
		allKeysMatch := true
		for k := range v.variables.Keys() {
			if !innerVars.Has(k) {
				allKeysMatch = false
				break
			}
		}
		if allKeysMatch {
			return v.module.CouldHaveBeenConfigured(variables)
		}
	}

	filtered := make(map[string]struct{})
	for name := range v.variables.Keys() {
		if _, ok := variables[name]; ok {
			filtered[name] = struct{}{}
		}
	}
	return v.module.CouldHaveBeenConfigured(filtered)
}

// CloneCss clones the inner module's CSS tree, then re-derives the shadow
// blocklists from the key difference between the cloned maps and this
// view's filtered maps.
//
// Dart: ShadowedModuleView.cloneCss stores _shadowed* fields and passes them
// through to reconstruct the shadowed view. Instead of storing dead fields,
// we reconstruct them from the diff of the cloned inner module vs. the view.
func (v *ShadowedModuleView) CloneCss() (Module, error) {
	cloned, err := v.module.CloneCss()
	if err != nil {
		return nil, err
	}
	shadowedVars := mapKeyDiff(cloned.Variables(), v.variables)
	shadowedFuncs := mapKeyDiff(cloned.Functions(), v.functions)
	shadowedMixins := mapKeyDiff(cloned.Mixins(), v.mixins)
	return NewShadowedModuleView(cloned, shadowedVars, shadowedFuncs, shadowedMixins), nil
}

// mapKeyDiff returns the keys in full that are absent from filtered: the
// shadow set that produced this view, recomputed after cloning.
func mapKeyDiff[V any](full, filtered orderedmap.Map[string, V]) map[string]struct{} {
	if full.Len() == filtered.Len() {
		return nil
	}
	var result map[string]struct{}
	for k := range full.Keys() {
		if !filtered.Has(k) {
			if result == nil {
				result = make(map[string]struct{})
			}
			result[k] = struct{}{}
		}
	}
	return result
}

// IsEmpty reports whether the view exposes no members and no CSS. A
// memberless view that still carries CSS is kept, never dropped — the
// environment's shadowing path would otherwise lose that CSS.
//
// Matches Dart: ShadowedModuleView.isEmpty
func (v *ShadowedModuleView) IsEmpty() (bool, error) {
	if v.variables.Len() != 0 || v.functions.Len() != 0 || v.mixins.Len() != 0 {
		return false, nil
	}
	css, err := v.module.CSS()
	if err != nil {
		return false, err
	}
	return len(css.Children()) == 0, nil
}

// String describes the view for debugging, prefixing the inner module
// with "shadowed".
//
// Matches Dart: ShadowedModuleView.toString
func (v *ShadowedModuleView) String() (string, error) {
	s, err := value.SprintAny(v.module)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("shadowed %s", s), nil
}
