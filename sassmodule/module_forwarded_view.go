// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassmodule

// dart-source: lib/src/module/forwarded_view.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// forwardedMap wraps m so it only shows the members a @forward rule lets
// through, layering prefix, then safelist, then blocklist.
//
// A nil prefix with no effective filters returns m untouched. Only one of
// safelist or blocklist may apply — passing both is a caller bug.
//
// Matches Dart: ForwardedModuleView._forwardedMap
func forwardedMap[V any](m orderedmap.Map[string, V], prefix *string, safelist, blocklist map[string]struct{}) orderedmap.Map[string, V] {
	// Dart: assert(safelist == null || blocklist == null); — forwarded_view.dart
	if safelist != nil && blocklist != nil && len(blocklist) > 0 {
		panic("BUG: safelist and blocklist cannot both be set")
	}
	if prefix == nil && safelist == nil && len(blocklist) == 0 {
		return m
	}
	var result orderedmap.Map[string, V] = m
	if prefix != nil {
		result = orderedmap.NewPrefixedMapView[V](result, *prefix)
	}
	if safelist != nil {
		result = orderedmap.NewLimitedMapViewSafelist(result, safelist)
	} else if len(blocklist) > 0 {
		result = orderedmap.NewLimitedMapViewBlocklist(result, blocklist)
	}
	return result
}

// ForwardedModuleView is a Module that exposes an inner module's members
// according to a @forward rule's prefix and show/hide filters.
//
// The four member maps are precomputed once: variables and variableNodes
// share the rule's variable filters, while functions and mixins share the
// rule's mixin-and-function filters. Everything else — URL, upstream,
// extensions, CSS, comments, transitivity — delegates straight to the inner
// module, and variable identity resolves through it so @forward conflict
// detection sees the original definitions.
//
// Matches Dart: ForwardedModuleView (module/forwarded_view.dart)
type ForwardedModuleView struct {
	module        Module
	rule          *value.ForwardRule
	variables     orderedmap.Map[string, value.Value]
	variableNodes orderedmap.Map[string, sasscommon.AstNode]
	functions     orderedmap.Map[string, sasscallable.Callable]
	mixins        orderedmap.Map[string, sasscallable.Callable]
}

// NewForwardedModuleView wraps module so its members show through rule.
//
// Use ForwardedModuleViewIfNecessary when the rule may filter nothing — it
// skips the wrapper entirely in that case.
//
// Matches Dart: ForwardedModuleView constructor
func NewForwardedModuleView(module Module, rule *value.ForwardRule) *ForwardedModuleView {
	return &ForwardedModuleView{
		module:        module,
		rule:          rule,
		variables:     forwardedMap(module.Variables(), rule.Prefix, rule.ShownVariables.Map(), rule.HiddenVariables.Map()),
		variableNodes: forwardedMap(module.VariableNodes(), rule.Prefix, rule.ShownVariables.Map(), rule.HiddenVariables.Map()),
		functions:     forwardedMap(module.Functions(), rule.Prefix, rule.ShownMixinsAndFunctions.Map(), rule.HiddenMixinsAndFunctions.Map()),
		mixins:        forwardedMap(module.Mixins(), rule.Prefix, rule.ShownMixinsAndFunctions.Map(), rule.HiddenMixinsAndFunctions.Map()),
	}
}

// ForwardedModuleViewIfNecessary wraps module in a ForwardedModuleView,
// but returns module as-is when rule filters nothing (no prefix, no show
// lists, empty hide lists), avoiding a no-op view layer.
//
// Matches Dart: ForwardedModuleView.ifNecessary
func ForwardedModuleViewIfNecessary(module Module, rule *value.ForwardRule) Module {
	if rule.Prefix == nil &&
		rule.ShownMixinsAndFunctions == nil &&
		rule.ShownVariables == nil &&
		(rule.HiddenMixinsAndFunctions == nil || rule.HiddenMixinsAndFunctions.Len() == 0) &&
		(rule.HiddenVariables == nil || rule.HiddenVariables.Len() == 0) {
		return module
	}
	return NewForwardedModuleView(module, rule)
}

// URL, Upstream, ExtensionStore, CSS, PreModuleComments, and the
// transitivity queries all delegate to the inner module: forwarding only
// filters member visibility, never CSS or extension behavior.
func (v *ForwardedModuleView) URL() (string, error) { return v.module.URL() }
func (v *ForwardedModuleView) Upstream() []Module   { return v.module.Upstream() }

// Variables, VariableNodes, Functions, and Mixins return the precomputed
// filtered views built by NewForwardedModuleView — not the inner module's
// raw maps.
func (v *ForwardedModuleView) Variables() orderedmap.Map[string, value.Value] {
	return v.variables
}
func (v *ForwardedModuleView) VariableNodes() orderedmap.Map[string, sasscommon.AstNode] {
	return v.variableNodes
}
func (v *ForwardedModuleView) Functions() orderedmap.Map[string, sasscallable.Callable] {
	return v.functions
}
func (v *ForwardedModuleView) Mixins() orderedmap.Map[string, sasscallable.Callable] {
	return v.mixins
}
func (v *ForwardedModuleView) ExtensionStore() extend.ExtensionStore {
	return v.module.ExtensionStore()
}
func (v *ForwardedModuleView) CSS() (*value.CssStylesheet, error) { return v.module.CSS() }
func (v *ForwardedModuleView) PreModuleComments() map[Module][]value.CssComment {
	return v.module.PreModuleComments()
}
func (v *ForwardedModuleView) TransitivelyContainsCss() bool {
	return v.module.TransitivelyContainsCss()
}
func (v *ForwardedModuleView) TransitivelyContainsExtensions() bool {
	return v.module.TransitivelyContainsExtensions()
}

// SetVariable writes through the rule: names outside the show list, inside
// the hide list, or missing the prefix are reported undefined, and the
// prefix is stripped before delegating so the inner module sees the
// original name.
//
// Matches Dart: ForwardedModuleView.setVariable
func (v *ForwardedModuleView) SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) error {
	if v.rule.ShownVariables != nil {
		if !v.rule.ShownVariables.Contains(name) {
			return sasscommon.NewSassScriptException("Undefined variable.", nil)
		}
	} else if v.rule.HiddenVariables != nil {
		if v.rule.HiddenVariables.Contains(name) {
			return sasscommon.NewSassScriptException("Undefined variable.", nil)
		}
	}

	if v.rule.Prefix != nil {
		if !strings.HasPrefix(name, *v.rule.Prefix) {
			return sasscommon.NewSassScriptException("Undefined variable.", nil)
		}
		name = name[len(*v.rule.Prefix):]
	}

	return v.module.SetVariable(name, val, nodeWithSpan)
}

// VariableIdentity returns the inner module's identity for the
// prefix-stripped name, so two forwards of one definition compare equal.
// The caller must only pass names visible through this view.
//
// Matches Dart: ForwardedModuleView.variableIdentity
func (v *ForwardedModuleView) VariableIdentity(name string) (any, error) {
	if !v.variables.Has(name) {
		panic("assertion failed: variable not found")
	}

	if v.rule.Prefix != nil {
		if !strings.HasPrefix(name, *v.rule.Prefix) {
			panic("assertion failed: variable name does not have prefix")
		}
		name = name[len(*v.rule.Prefix):]
	}

	return v.module.VariableIdentity(name)
}

// CouldHaveBeenConfigured translates the query back through the rule
// before delegating: prefixed names shed their prefix, show lists intersect,
// hide lists subtract. A rule that filters nothing delegates unchanged.
//
// Matches Dart: ForwardedModuleView.couldHaveBeenConfigured
func (v *ForwardedModuleView) CouldHaveBeenConfigured(variables map[string]struct{}) bool {
	if v.rule.ShownVariables != nil && v.rule.HiddenVariables != nil {
		panic("assertion failed: shownVariables and hiddenVariables cannot both be set")
	}
	if v.rule.Prefix == nil &&
		v.rule.ShownVariables == nil &&
		(v.rule.HiddenVariables == nil || v.rule.HiddenVariables.Len() == 0) {
		return v.module.CouldHaveBeenConfigured(variables)
	}

	if v.rule.Prefix != nil {
		filtered := make(map[string]struct{})
		for name := range variables {
			if strings.HasPrefix(name, *v.rule.Prefix) {
				filtered[name[len(*v.rule.Prefix):]] = struct{}{}
			}
		}
		variables = filtered
	}

	if v.rule.ShownVariables != nil {
		filtered := make(map[string]struct{})
		for name := range variables {
			if v.rule.ShownVariables.Contains(name) {
				filtered[name] = struct{}{}
			}
		}
		return v.module.CouldHaveBeenConfigured(filtered)
	} else if v.rule.HiddenVariables != nil && v.rule.HiddenVariables.Len() > 0 {
		filtered := make(map[string]struct{})
		for name := range variables {
			if !v.rule.HiddenVariables.Contains(name) {
				filtered[name] = struct{}{}
			}
		}
		return v.module.CouldHaveBeenConfigured(filtered)
	}

	return v.module.CouldHaveBeenConfigured(variables)
}

// CloneCss clones the inner module's CSS tree and re-applies the same rule,
// so the copy filters exactly like the original.
//
// Matches Dart: ForwardedModuleView.cloneCss
func (v *ForwardedModuleView) CloneCss() (Module, error) {
	cloned, err := v.module.CloneCss()
	if err != nil {
		return nil, err
	}
	return NewForwardedModuleView(cloned, v.rule), nil
}

// String describes the view for debugging, prefixing the inner module
// with "forwarded".
//
// Matches Dart: ForwardedModuleView.toString
func (v *ForwardedModuleView) String() (string, error) {
	s, err := value.SprintAny(v.module)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("forwarded %s", s), nil
}
