// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassenv

// dart-source: lib/src/environment.dart

import (
	"fmt"
	"maps"
	"net/url"

	"github.com/bancek/go-sass/configuration"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sassclonecss"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// Environment tracks lexically-scoped information: variables, functions,
// mixins, and modules during evaluation.
type Environment struct {
	// The modules visible in the current scope, indexed by namespace.
	modules map[string]sassmodule.Module

	// A map from module namespaces to the AST nodes indicating where those
	// modules were loaded.
	namespaceNodes map[string]sasscommon.AstNode

	// Namespaceless modules (those loaded with @use ... as *).
	globalModules map[sassmodule.Module]sasscommon.AstNode

	// Modules imported into the current scope.
	importedModules map[sassmodule.Module]sasscommon.AstNode

	// Modules forwarded by this module, mapped to their @forward rule nodes.
	forwardedModules map[sassmodule.Module]sasscommon.AstNode

	// Modules forwarded by nested imports at each lexical scope level beneath
	// the global scope. Nil until needed.
	nestedForwardedModules [][]sassmodule.Module

	// All modules in the order they were @use'd.
	allModules []sassmodule.Module

	// Variables defined at each lexical scope level (first = global).
	variables []*orderedmap.LinkedMap[string, value.Value]

	variableNodes []*orderedmap.LinkedMap[string, sasscommon.AstNode]

	// Cached indices of where each variable is defined. Lazily filled.
	variableIndices map[string]int

	// Functions defined at each lexical scope level (first = global).
	functions []*orderedmap.LinkedMap[string, sasscallable.Callable]

	// Cached indices of where each function is defined. Lazily filled.
	functionIndices map[string]int

	// Mixins defined at each lexical scope level (first = global).
	mixins []*orderedmap.LinkedMap[string, sasscallable.Callable]

	// Cached indices of where each mixin is defined. Lazily filled.
	mixinIndices map[string]int

	// The content block passed to the enclosing mixin, or nil.
	content sasscallable.Callable

	// The set of variable names that could be configured when loading the module.
	configurableVariables map[string]struct{}

	// Whether the environment is lexically within a mixin.
	inMixin bool

	// Whether the environment is currently in a global or semi-global scope.
	inSemiGlobalScope bool

	// The name of the last variable accessed (cache for repeated access).
	lastVariableName string

	// The index in variables of the last variable accessed.
	lastVariableIndex int

	// Whether we have a valid cached lastVariableIndex.
	hasLastVariableIndex bool
}

// NewEnvironment creates a new Environment with an initial global scope.
func NewEnvironment() *Environment {
	return &Environment{
		modules:               make(map[string]sassmodule.Module),
		namespaceNodes:        make(map[string]sasscommon.AstNode),
		globalModules:         make(map[sassmodule.Module]sasscommon.AstNode),
		importedModules:       make(map[sassmodule.Module]sasscommon.AstNode),
		allModules:            make([]sassmodule.Module, 0),
		variables:             []*orderedmap.LinkedMap[string, value.Value]{orderedmap.New[string, value.Value]()},
		variableNodes:         []*orderedmap.LinkedMap[string, sasscommon.AstNode]{orderedmap.New[string, sasscommon.AstNode]()},
		variableIndices:       make(map[string]int),
		functions:             []*orderedmap.LinkedMap[string, sasscallable.Callable]{orderedmap.New[string, sasscallable.Callable]()},
		functionIndices:       make(map[string]int),
		mixins:                []*orderedmap.LinkedMap[string, sasscallable.Callable]{orderedmap.New[string, sasscallable.Callable]()},
		mixinIndices:          make(map[string]int),
		configurableVariables: make(map[string]struct{}),
		inSemiGlobalScope:     true,
	}
}

// environmentConstructor creates an Environment with the given fields.
// This is the Go equivalent of the private Environment._ constructor,
// used by Closure() and forImport().
func newEnvironmentWithFields(
	modules map[string]sassmodule.Module,
	namespaceNodes map[string]sasscommon.AstNode,
	globalModules map[sassmodule.Module]sasscommon.AstNode,
	importedModules map[sassmodule.Module]sasscommon.AstNode,
	forwardedModules map[sassmodule.Module]sasscommon.AstNode,
	nestedForwardedModules [][]sassmodule.Module,
	allModules []sassmodule.Module,
	variables []*orderedmap.LinkedMap[string, value.Value],
	variableNodes []*orderedmap.LinkedMap[string, sasscommon.AstNode],
	functions []*orderedmap.LinkedMap[string, sasscallable.Callable],
	mixins []*orderedmap.LinkedMap[string, sasscallable.Callable],
	content sasscallable.Callable,
	configurableVariables map[string]struct{},
) *Environment {
	return &Environment{
		modules:                modules,
		namespaceNodes:         namespaceNodes,
		globalModules:          globalModules,
		importedModules:        importedModules,
		forwardedModules:       forwardedModules,
		nestedForwardedModules: nestedForwardedModules,
		allModules:             allModules,
		variables:              variables,
		variableNodes:          variableNodes,
		variableIndices:        make(map[string]int),
		functions:              functions,
		functionIndices:        make(map[string]int),
		mixins:                 mixins,
		mixinIndices:           make(map[string]int),
		content:                content,
		configurableVariables:  configurableVariables,
		inSemiGlobalScope:      true,
	}
}

// Modules returns the modules visible in the current scope, indexed by namespace.
//
// The copy is a snapshot: callers can read it freely but mutating it never
// affects the environment. Use AddModule to register a module.
//
// Matches Dart: Environment.modules
func (e *Environment) Modules() map[string]sassmodule.Module {
	result := make(map[string]sassmodule.Module, len(e.modules))
	maps.Copy(result, e.modules)
	return result
}

// AtRoot returns true if the environment is lexically at the root of the document.
func (e *Environment) AtRoot() bool {
	return len(e.variables) == 1
}

// InMixin returns true if the environment is lexically within a mixin.
func (e *Environment) InMixin() bool {
	return e.inMixin
}

// SetInMixin sets whether we're currently evaluating a mixin body.
func (e *Environment) SetInMixin(v bool) {
	e.inMixin = v
}

// Content returns the content block passed to the enclosing mixin, or nil.
func (e *Environment) Content() sasscallable.Callable {
	return e.content
}

// Closure creates a copy of this environment for use as a closure.
// Scope changes in this environment will not affect the closure.
// However, any new declarations or assignments in scopes that are visible
// when the closure was created will be reflected.
//
// Matches Dart: Environment.closure
func (e *Environment) Closure() *Environment {
	vars := make([]*orderedmap.LinkedMap[string, value.Value], len(e.variables))
	copy(vars, e.variables)
	varNodes := make([]*orderedmap.LinkedMap[string, sasscommon.AstNode], len(e.variableNodes))
	copy(varNodes, e.variableNodes)
	fns := make([]*orderedmap.LinkedMap[string, sasscallable.Callable], len(e.functions))
	copy(fns, e.functions)
	mix := make([]*orderedmap.LinkedMap[string, sasscallable.Callable], len(e.mixins))
	copy(mix, e.mixins)

	return newEnvironmentWithFields(
		e.modules,
		e.namespaceNodes,
		e.globalModules,
		e.importedModules,
		e.forwardedModules,
		e.nestedForwardedModules,
		e.allModules,
		vars,
		varNodes,
		fns,
		mix,
		e.content,
		map[string]struct{}{}, // closures are always in nested contexts
	)
}

// ForImport creates a new environment for an imported file.
// It shares variables, functions, and mixins but excludes most modules
// (except for global modules that result from importing a file with forwards).
func (e *Environment) ForImport() *Environment {
	vars := make([]*orderedmap.LinkedMap[string, value.Value], len(e.variables))
	copy(vars, e.variables)
	varNodes := make([]*orderedmap.LinkedMap[string, sasscommon.AstNode], len(e.variableNodes))
	copy(varNodes, e.variableNodes)
	fns := make([]*orderedmap.LinkedMap[string, sasscallable.Callable], len(e.functions))
	copy(fns, e.functions)
	mix := make([]*orderedmap.LinkedMap[string, sasscallable.Callable], len(e.mixins))
	copy(mix, e.mixins)

	return newEnvironmentWithFields(
		make(map[string]sassmodule.Module),
		make(map[string]sasscommon.AstNode),
		make(map[sassmodule.Module]sasscommon.AstNode),
		e.importedModules,
		nil,
		e.nestedForwardedModules,
		make([]sassmodule.Module, 0),
		vars,
		varNodes,
		fns,
		mix,
		e.content,
		e.configurableVariables,
	)
}

// AddModule adds module to the set of modules visible in this environment.
// If namespace is set, the module is available under that namespace.
// If namespace is nil, the module is a global (namespaceless) module.
//
// The node's span reports conflicts: a duplicate namespace blames the
// original @use, and a global module colliding on a global variable name
// fails fast. Global modules also join allModules for CSS collation.
//
// Matches Dart: Environment.addModule
func (e *Environment) AddModule(module sassmodule.Module, nodeWithSpan sasscommon.AstNode, namespace *string) error {
	if namespace == nil {
		e.globalModules[module] = nodeWithSpan
		e.allModules = append(e.allModules, module)

		// Namespaceless modules share one variable scope with the global
		// frame, so a collision on any global variable is an error now
		// rather than an ambiguity later.
		for name := range e.variables[0].Keys() {
			if module.Variables().Has(name) {
				return sasscommon.NewSassScriptException(
					fmt.Sprintf("This module and the new module both define a variable named \"$%s\".", name),
					nil)
			}
		}
	} else {
		if _, ok := e.modules[*namespace]; ok {
			span, err := e.namespaceNodes[*namespace].Span()
			if err != nil {
				return err
			}
			secondary := map[sasscommon.FileSpan]string{}
			secondary[span] = "original @use"
			return &sasscommon.MultiSpanSassScriptException{
				Message:        fmt.Sprintf("There's already a module with namespace \"%s\".", *namespace),
				PrimaryLabel:   "new @use",
				SecondarySpans: secondary,
			}
		}

		e.modules[*namespace] = module
		e.namespaceNodes[*namespace] = nodeWithSpan
		e.allModules = append(e.allModules, module)
	}
	return nil
}

// ForwardModule exposes the members in module to downstream modules as though
// they were defined in this module, according to the modifications defined by rule.
//
// Matches Dart: Environment.forwardModule
func (e *Environment) ForwardModule(module sassmodule.Module, rule *value.ForwardRule, nodeWithSpan sasscommon.AstNode) error {
	if e.forwardedModules == nil {
		e.forwardedModules = make(map[sassmodule.Module]sasscommon.AstNode)
	}

	var view sassmodule.Module
	if rule != nil {
		view = sassmodule.ForwardedModuleViewIfNecessary(module, rule)
	} else {
		view = module
	}

	// Every pair of forwarded member sets must agree: a name defined by two
	// different origins is ambiguous downstream, so each kind (variables,
	// functions, mixins) is checked against every already-forwarded module.
	for other := range e.forwardedModules {
		if err := assertNoConflicts(
			view.Variables(),
			other.Variables(),
			view,
			other,
			"variable",
			e.forwardedModules,
		); err != nil {
			return err
		}
		if err := assertNoConflicts(
			view.Functions(),
			other.Functions(),
			view,
			other,
			"function",
			e.forwardedModules,
		); err != nil {
			return err
		}
		if err := assertNoConflicts(
			view.Mixins(),
			other.Mixins(),
			view,
			other,
			"mixin",
			e.forwardedModules,
		); err != nil {
			return err
		}
	}

	// Add the original module to _allModules (rather than the ForwardedModuleView)
	// so that we can de-duplicate upstream modules using ==.
	e.allModules = append(e.allModules, module)
	e.forwardedModules[view] = nodeWithSpan
	return nil
}

// assertNoConflicts returns a MultiSpanSassScriptException if newMembers has any
// keys that overlap with oldMembers from different origins.
//
// Iteration runs over the smaller map for speed. Variables compare by
// origin identity (one definition forwarded twice is fine); functions and
// mixins compare by reference. The error blames the new @forward and points
// at the original @forward of the older module.
//
// Matches Dart: _Environment._assertNoConflicts
func assertNoConflicts[V any](
	newMembers orderedmap.Map[string, V],
	oldMembers orderedmap.Map[string, V],
	newModule sassmodule.Module,
	oldModule sassmodule.Module,
	memberType string,
	forwardedModules map[sassmodule.Module]sasscommon.AstNode,
) error {
	var smaller, larger orderedmap.Map[string, V]
	var smallerModule, largerModule sassmodule.Module
	if newMembers.Len() < oldMembers.Len() {
		smaller = newMembers
		larger = oldMembers
		smallerModule = newModule
		largerModule = oldModule
	} else {
		smaller = oldMembers
		larger = newMembers
		smallerModule = oldModule
		largerModule = newModule
	}

	for name := range smaller.Keys() {
		if !larger.Has(name) {
			continue
		}
		if memberType == "variable" {
			smallerIdentity, err := smallerModule.VariableIdentity(name)
			if err != nil {
				continue
			}
			largerIdentity, err := largerModule.VariableIdentity(name)
			if err != nil {
				continue
			}
			if smallerIdentity == largerIdentity {
				continue
			}
		} else {
			smallVal, _ := smaller.Get(name)
			largeVal, _ := larger.Get(name)
			if any(smallVal) == any(largeVal) {
				continue
			}
		}

		displayName := name
		if memberType == "variable" {
			displayName = "$" + name
		}

		secondary := make(map[sasscommon.FileSpan]string)
		if node, ok := forwardedModules[oldModule]; ok {
			span, err := node.Span()
			if err != nil {
				return err
			}
			secondary[span] = "original @forward"
		}

		return &sasscommon.MultiSpanSassScriptException{
			Message:        fmt.Sprintf("Two forwarded modules both define a %s named %s.", memberType, displayName),
			PrimaryLabel:   "new @forward",
			SecondarySpans: secondary,
		}
	}
	return nil
}

// ImportForwards makes members forwarded by module available in the current
// environment. Called when module is @imported.
//
// Only environment-backed modules forward anything, so other module shapes
// are a no-op. At the root the forwarded members merge into the imported and
// forwarded sets (shadowing same-named locals out of the way first); in a
// nested scope they queue onto the current frame's nested-forwarded list,
// and in both cases same-named local definitions are dropped since the
// forwarded members now win.
//
// Matches Dart: Environment.importForwards
func (e *Environment) ImportForwards(module sassmodule.Module) error {
	envMod, ok := module.(*environmentModule)
	if !ok {
		return nil
	}
	forwarded := envMod.environment.forwardedModules
	if forwarded == nil {
		return nil
	}

	// Omit modules from forwarded that are already globally available and
	// forwarded in this module: re-adding them would duplicate members and
	// re-trigger conflict checks that already passed.
	if e.forwardedModules != nil {
		filtered := make(map[sassmodule.Module]sasscommon.AstNode)
		for mod, node := range forwarded {
			if _, inForwarded := e.forwardedModules[mod]; inForwarded {
				if _, inGlobal := e.globalModules[mod]; inGlobal {
					continue
				}
			}
			filtered[mod] = node
		}
		forwarded = filtered
	} else {
		e.forwardedModules = make(map[sassmodule.Module]sasscommon.AstNode)
	}

	forwardedVariableNames := make(map[string]struct{})
	for mod := range forwarded {
		for name := range mod.Variables().Keys() {
			forwardedVariableNames[name] = struct{}{}
		}
	}
	forwardedFunctionNames := make(map[string]struct{})
	for mod := range forwarded {
		for name := range mod.Functions().Keys() {
			forwardedFunctionNames[name] = struct{}{}
		}
	}
	forwardedMixinNames := make(map[string]struct{})
	for mod := range forwarded {
		for name := range mod.Mixins().Keys() {
			forwardedMixinNames[name] = struct{}{}
		}
	}

	if e.AtRoot() {
		// Hide members from modules that have already been imported or
		// forwarded that would otherwise conflict with the @imported members.
		// A view that ends up with no members and no CSS is dropped rather
		// than kept as dead weight.
		importedMods := make([]sassmodule.Module, 0, len(e.importedModules))
		importedNodes := make([]sasscommon.AstNode, 0, len(e.importedModules))
		for mod, node := range e.importedModules {
			importedMods = append(importedMods, mod)
			importedNodes = append(importedNodes, node)
		}
		for i, mod := range importedMods {
			if shadowed := sassmodule.NewShadowedModuleViewIfNecessary(mod, forwardedVariableNames, forwardedFunctionNames, forwardedMixinNames); shadowed != nil {
				delete(e.importedModules, mod)
				empty, emptyErr := shadowed.IsEmpty()
				if emptyErr != nil {
					return emptyErr
				}
				if !empty {
					e.importedModules[shadowed] = importedNodes[i]
				}
			}
		}

		forwardedMods := make([]sassmodule.Module, 0, len(e.forwardedModules))
		forwardedNodes := make([]sasscommon.AstNode, 0, len(e.forwardedModules))
		for mod, node := range e.forwardedModules {
			forwardedMods = append(forwardedMods, mod)
			forwardedNodes = append(forwardedNodes, node)
		}
		for i, mod := range forwardedMods {
			if shadowed := sassmodule.NewShadowedModuleViewIfNecessary(mod, forwardedVariableNames, forwardedFunctionNames, forwardedMixinNames); shadowed != nil {
				delete(e.forwardedModules, mod)
				empty, emptyErr := shadowed.IsEmpty()
				if emptyErr != nil {
					return emptyErr
				}
				if !empty {
					e.forwardedModules[shadowed] = forwardedNodes[i]
				}
			}
		}

		maps.Copy(e.importedModules, forwarded)
		maps.Copy(e.forwardedModules, forwarded)
	} else {
		if e.nestedForwardedModules == nil {
			e.nestedForwardedModules = make([][]sassmodule.Module, len(e.variables)-1)
			for i := range e.nestedForwardedModules {
				e.nestedForwardedModules[i] = make([]sassmodule.Module, 0)
			}
		}
		last := len(e.nestedForwardedModules) - 1
		for mod := range forwarded {
			e.nestedForwardedModules[last] = append(e.nestedForwardedModules[last], mod)
		}
	}

	// Remove existing member definitions that are now shadowed by the
	// forwarded modules, and drop their cached scope indices so later
	// lookups re-resolve instead of hitting stale frames.
	for name := range forwardedVariableNames {
		delete(e.variableIndices, name)
		e.variables[len(e.variables)-1].Delete(name)
		e.variableNodes[len(e.variableNodes)-1].Delete(name)
	}
	for name := range forwardedFunctionNames {
		delete(e.functionIndices, name)
		e.functions[len(e.functions)-1].Delete(name)
	}
	for name := range forwardedMixinNames {
		delete(e.mixinIndices, name)
		e.mixins[len(e.mixins)-1].Delete(name)
	}
	return nil
}

// GetVariable returns the value of the variable named name, optionally with
// the given namespace. Returns nil if no such variable is declared.
//
// Unnamespaced lookups run three tiers: the last-access cache, the lazily
// filled scope-index cache, then a full scope scan — each falling back to
// the namespaceless modules. It fails when the namespace is unknown or when
// several global modules expose the name.
//
// Matches Dart: Environment.getVariable
func (e *Environment) GetVariable(name string, namespace *string) (value.Value, error) {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return nil, err
		}
		val, _ := mod.Variables().Get(name)
		return val, nil
	}

	if e.lastVariableName == name && e.hasLastVariableIndex {
		if v, ok := e.variables[e.lastVariableIndex].Get(name); ok {
			return v, nil
		}
		return e.getVariableFromGlobalModule(name)
	}

	if index, ok := e.variableIndices[name]; ok {
		e.lastVariableName = name
		e.lastVariableIndex = index
		e.hasLastVariableIndex = true
		if v, ok := e.variables[index].Get(name); ok {
			return v, nil
		}
		return e.getVariableFromGlobalModule(name)
	} else if index := e.variableIndex(name); index != -1 {
		e.lastVariableName = name
		e.lastVariableIndex = index
		e.hasLastVariableIndex = true
		e.variableIndices[name] = index
		if v, ok := e.variables[index].Get(name); ok {
			return v, nil
		}
		return e.getVariableFromGlobalModule(name)
	} else {
		return e.getVariableFromGlobalModule(name)
	}
}

// getVariableFromGlobalModule returns the value of name from a namespaceless
// module, or nil if not found.
func (e *Environment) getVariableFromGlobalModule(name string) (value.Value, error) {
	return fromOneModule(e, name, "variable", func(m sassmodule.Module) value.Value {
		val, _ := m.Variables().Get(name)
		return val
	})
}

// GetVariableNode returns the AstNode for the variable named name, or nil
// if no such variable is declared.
//
// The node is a proxy for the definition span: nodes are stored instead of
// spans so spans are only manufactured when an error or warning needs one.
// It follows the same cache tiers as GetVariable.
//
// Matches Dart: Environment.getVariableNode
func (e *Environment) GetVariableNode(name string, namespace *string) (sasscommon.AstNode, error) {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return nil, err
		}
		val, _ := mod.VariableNodes().Get(name)
		return val, nil
	}

	if e.lastVariableName == name && e.hasLastVariableIndex {
		if n, ok := e.variableNodes[e.lastVariableIndex].Get(name); ok {
			return n, nil
		}
		return e.getVariableNodeFromGlobalModule(name)
	}

	if index, ok := e.variableIndices[name]; ok {
		e.lastVariableName = name
		e.lastVariableIndex = index
		e.hasLastVariableIndex = true
		if n, ok := e.variableNodes[index].Get(name); ok {
			return n, nil
		}
		return e.getVariableNodeFromGlobalModule(name)
	} else if index := e.variableIndex(name); index != -1 {
		e.lastVariableName = name
		e.lastVariableIndex = index
		e.hasLastVariableIndex = true
		e.variableIndices[name] = index
		if n, ok := e.variableNodes[index].Get(name); ok {
			return n, nil
		}
		return e.getVariableNodeFromGlobalModule(name)
	} else {
		return e.getVariableNodeFromGlobalModule(name)
	}
}

// getVariableNodeFromGlobalModule returns the node for name from a
// namespaceless module, or nil if not found.
//
// Unlike the value lookup, there is no ambiguity error here: GetVariable
// already ruled out multiple definitions by the time nodes are requested.
// Imported modules shadow global ones by lookup order.
//
// Matches Dart: Environment._getVariableNodeFromGlobalModule
func (e *Environment) getVariableNodeFromGlobalModule(name string) (sasscommon.AstNode, error) {
	for _, list := range [][]sassmodule.Module{
		moduleKeys(e.importedModules),
		moduleKeys(e.globalModules),
	} {
		for _, module := range list {
			if n, ok := module.VariableNodes().Get(name); ok {
				return n, nil
			}
		}
	}
	return nil, nil
}

// VariableExists returns whether a variable named name exists.
func (e *Environment) VariableExists(name string) (bool, error) {
	val, err := e.GetVariable(name, nil)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// GlobalVariableExists returns whether a global variable named name exists.
func (e *Environment) GlobalVariableExists(name string, namespace *string) (bool, error) {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return false, err
		}
		return mod.Variables().Has(name), nil
	}
	if e.variables[0].Has(name) {
		return true, nil
	}
	val, err := e.getVariableFromGlobalModule(name)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// variableIndex returns the index of the last map in variables that has name,
// or -1 if none exists.
func (e *Environment) variableIndex(name string) int {
	for i := len(e.variables) - 1; i >= 0; i-- {
		if e.variables[i].Has(name) {
			return i
		}
	}
	return -1
}

// SetVariable sets the variable named name to value, recording nodeWithSpan
// so later errors can point at the assignment.
//
// A namespace routes into that module. Otherwise a global write (or any
// write at the root) lands in frame 0, trying a same-named global module
// first; a nested write prefers the innermost scope that already defines
// the name, falls back to a nested-forwarded module holding it, and only
// then declares it in the current frame. Outside a semi-global scope, frame
// 0 is never written implicitly.
//
// The node (not a span) is stored so spans are manufactured lazily.
//
// Matches Dart: Environment.setVariable
func (e *Environment) SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode, namespace *string, global bool) error {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return err
		}
		return mod.SetVariable(name, val, nodeWithSpan)
	}

	if global || e.AtRoot() {
		// Don't set the index if there's already a variable with the given name,
		// since local accesses should still return the local variable.
		// (The cached global index must not shadow an inner declaration.)
		if _, ok := e.variableIndices[name]; !ok {
			e.lastVariableName = name
			e.lastVariableIndex = 0
			e.hasLastVariableIndex = true
			e.variableIndices[name] = 0
		}

		// If this module doesn't already contain a variable named name, try
		// setting it in a global module: a !global assignment targets the
		// module that owns the variable, not the local frame.
		if !e.variables[0].Has(name) {
			moduleWithName, err := fromOneModule(e, name, "variable", func(m sassmodule.Module) sassmodule.Module {
				if m.Variables().Has(name) {
					return m
				}
				return nil
			})
			if err != nil {
				return err
			}
			if moduleWithName != nil {
				return moduleWithName.SetVariable(name, val, nodeWithSpan)
			}
		}

		e.variables[0].Put(name, val)
		e.variableNodes[0].Put(name, nodeWithSpan)
		return nil
	}

	// A nested write with no cached or visible definition may still belong
	// to a nested-forwarded module (an @import inside a rule bringing in a
	// @forward), so those modules get first claim before a fresh local is
	// declared.
	if e.nestedForwardedModules != nil {
		if _, ok := e.variableIndices[name]; !ok {
			if e.variableIndex(name) == -1 {
				for i := len(e.nestedForwardedModules) - 1; i >= 0; i-- {
					modules := e.nestedForwardedModules[i]
					for j := len(modules) - 1; j >= 0; j-- {
						if modules[j].Variables().Has(name) {
							return modules[j].SetVariable(name, val, nodeWithSpan)
						}
					}
				}
			}
		}
	}

	index := 0
	if e.lastVariableName == name && e.hasLastVariableIndex {
		index = e.lastVariableIndex
	} else if idx, ok := e.variableIndices[name]; ok {
		index = idx
	} else if idx := e.variableIndex(name); idx != -1 {
		index = idx
		e.variableIndices[name] = idx
	} else {
		index = len(e.variables) - 1
		e.variableIndices[name] = index
	}

	// Outside a semi-global scope (a plain rule body), assignments must not
	// leak into frame 0: redirect them to the current frame.
	if !e.inSemiGlobalScope && index == 0 {
		index = len(e.variables) - 1
		e.variableIndices[name] = index
	}

	e.lastVariableName = name
	e.lastVariableIndex = index
	e.hasLastVariableIndex = true
	e.variables[index].Put(name, val)
	e.variableNodes[index].Put(name, nodeWithSpan)
	return nil
}

// SetLocalVariable sets the variable named name to value in the current scope.
// Unlike SetVariable, this will declare the variable in the current scope
// even if a declaration already exists in an outer scope.
func (e *Environment) SetLocalVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) {
	index := len(e.variables) - 1
	e.lastVariableName = name
	e.lastVariableIndex = index
	e.hasLastVariableIndex = true
	e.variableIndices[name] = index
	e.variables[index].Put(name, val)
	e.variableNodes[index].Put(name, nodeWithSpan)
}

// MarkVariableConfigurable records that name is a variable that could have been
// configured for this module, whether or not it actually was.
//
// The set backs the "already loaded, cannot configure" error: pushing a new
// configuration through @forward onto a loaded module fails only for names
// the module ever offered as configurable.
//
// Matches Dart: Environment.markVariableConfigurable
func (e *Environment) MarkVariableConfigurable(name string) {
	if e.configurableVariables == nil {
		e.configurableVariables = make(map[string]struct{})
	}
	e.configurableVariables[name] = struct{}{}
}

// GetFunction returns the function named name, optionally with the given
// namespace, or nil if no such function is declared.
//
// Like variables, functions resolve through the lazily filled scope-index
// cache before the namespaceless modules. It fails for an unknown namespace
// or an ambiguous global.
//
// Matches Dart: Environment.getFunction
func (e *Environment) GetFunction(name string, namespace *string) (sasscallable.Callable, error) {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return nil, err
		}
		fn, _ := mod.Functions().Get(name)
		return fn, nil
	}

	if index, ok := e.functionIndices[name]; ok {
		if fn, ok := e.functions[index].Get(name); ok {
			return fn, nil
		}
		return e.getFunctionFromGlobalModule(name)
	} else if index := e.functionIndex(name); index != -1 {
		e.functionIndices[name] = index
		if fn, ok := e.functions[index].Get(name); ok {
			return fn, nil
		}
		return e.getFunctionFromGlobalModule(name)
	} else {
		return e.getFunctionFromGlobalModule(name)
	}
}

// getFunctionFromGlobalModule returns the function named name from a
// namespaceless module, or nil if not found.
func (e *Environment) getFunctionFromGlobalModule(name string) (sasscallable.Callable, error) {
	return fromOneModule(e, name, "function", func(m sassmodule.Module) sasscallable.Callable {
		fn, _ := m.Functions().Get(name)
		return fn
	})
}

// functionIndex returns the index of the last map in functions that has name,
// or -1 if none exists.
func (e *Environment) functionIndex(name string) int {
	for i := len(e.functions) - 1; i >= 0; i-- {
		if e.functions[i].Has(name) {
			return i
		}
	}
	return -1
}

// FunctionExists returns whether a function named name exists.
func (e *Environment) FunctionExists(name string, namespace *string) (bool, error) {
	fn, err := e.GetFunction(name, namespace)
	if err != nil {
		return false, err
	}
	return fn != nil, nil
}

// SetFunction sets the function named name in the current scope, recording
// the frame index so later lookups hit the cache.
//
// Matches Dart: Environment.setFunction
func (e *Environment) SetFunction(callable sasscallable.Callable) {
	index := len(e.functions) - 1
	e.functionIndices[callable.Name()] = index
	e.functions[index].Put(callable.Name(), callable)
}

// GetMixin returns the mixin named name, optionally with the given
// namespace, or nil if no such mixin is declared.
//
// Resolution mirrors GetFunction: scope-index cache first, then the
// namespaceless modules with ambiguity detection.
//
// Matches Dart: Environment.getMixin
func (e *Environment) GetMixin(name string, namespace *string) (sasscallable.Callable, error) {
	if namespace != nil {
		mod, err := e.getModule(*namespace)
		if err != nil {
			return nil, err
		}
		mix, _ := mod.Mixins().Get(name)
		return mix, nil
	}

	if index, ok := e.mixinIndices[name]; ok {
		if mix, ok := e.mixins[index].Get(name); ok {
			return mix, nil
		}
		return e.getMixinFromGlobalModule(name)
	} else if index := e.mixinIndex(name); index != -1 {
		e.mixinIndices[name] = index
		if mix, ok := e.mixins[index].Get(name); ok {
			return mix, nil
		}
		return e.getMixinFromGlobalModule(name)
	} else {
		return e.getMixinFromGlobalModule(name)
	}
}

// getMixinFromGlobalModule returns the mixin named name from a namespaceless
// module, or nil if not found.
func (e *Environment) getMixinFromGlobalModule(name string) (sasscallable.Callable, error) {
	return fromOneModule(e, name, "mixin", func(m sassmodule.Module) sasscallable.Callable {
		mix, _ := m.Mixins().Get(name)
		return mix
	})
}

// mixinIndex returns the index of the last map in mixins that has name,
// or -1 if none exists.
func (e *Environment) mixinIndex(name string) int {
	for i := len(e.mixins) - 1; i >= 0; i-- {
		if e.mixins[i].Has(name) {
			return i
		}
	}
	return -1
}

// MixinExists returns whether a mixin named name exists.
func (e *Environment) MixinExists(name string, namespace *string) (bool, error) {
	mix, err := e.GetMixin(name, namespace)
	if err != nil {
		return false, err
	}
	return mix != nil, nil
}

// SetMixin sets the mixin named name in the current scope, recording the
// frame index so later lookups hit the cache.
//
// Matches Dart: Environment.setMixin
func (e *Environment) SetMixin(callable sasscallable.Callable) {
	index := len(e.mixins) - 1
	e.mixinIndices[callable.Name()] = index
	e.mixins[index].Put(callable.Name(), callable)
}

// SetContent sets the content block for the current environment, swapped in
// around each mixin application and restored after.
//
// Matches Dart: Environment content setter (via withContent)
func (e *Environment) SetContent(content sasscallable.Callable) {
	e.content = content
}

// pushFrame creates a new lexical scope: one fresh frame each for variables,
// variable nodes, functions, and mixins, plus a nested-forwarded slot when
// the environment tracks them.
func (e *Environment) pushFrame() {
	e.variables = append(e.variables, orderedmap.New[string, value.Value]())
	e.variableNodes = append(e.variableNodes, orderedmap.New[string, sasscommon.AstNode]())
	e.functions = append(e.functions, orderedmap.New[string, sasscallable.Callable]())
	e.mixins = append(e.mixins, orderedmap.New[string, sasscallable.Callable]())
	if e.nestedForwardedModules != nil {
		e.nestedForwardedModules = append(e.nestedForwardedModules, make([]sassmodule.Module, 0))
	}
}

// popFrame discards the innermost lexical scope.
//
// Cached scope indices for every name defined in the popped frame are
// dropped so later lookups re-resolve, and the last-variable fast path is
// invalidated since its frame may be gone.
func (e *Environment) popFrame() {
	for name := range e.variables[len(e.variables)-1].Keys() {
		delete(e.variableIndices, name)
	}
	for name := range e.functions[len(e.functions)-1].Keys() {
		delete(e.functionIndices, name)
	}
	for name := range e.mixins[len(e.mixins)-1].Keys() {
		delete(e.mixinIndices, name)
	}
	e.variableNodes = e.variableNodes[:len(e.variableNodes)-1]
	e.variables = e.variables[:len(e.variables)-1]
	e.functions = e.functions[:len(e.functions)-1]
	e.mixins = e.mixins[:len(e.mixins)-1]
	if e.nestedForwardedModules != nil {
		e.nestedForwardedModules = e.nestedForwardedModules[:len(e.nestedForwardedModules)-1]
	}
	e.lastVariableName = ""
	e.hasLastVariableIndex = false
}

// WithContent sets content for the duration of callback, restoring the
// previous block after — even across nested mixin applications, since each
// level saves its own predecessor.
//
// Matches Dart: Environment.withContent
func (e *Environment) WithContent(content sasscallable.Callable, callback func()) {
	oldContent := e.content
	e.content = content
	callback()
	e.content = oldContent
}

// AsMixin sets inMixin to true for the duration of callback, restoring the
// previous value after. Content blocks are only valid inside this window.
//
// Matches Dart: Environment.asMixin
func (e *Environment) AsMixin(callback func()) {
	oldInMixin := e.inMixin
	e.inMixin = true
	callback()
	e.inMixin = oldInMixin
}

// Scope runs callback in a new lexical scope and returns its result.
// Variables, functions, and mixins declared in a given scope are inaccessible
// outside of it. If semiGlobal is true, this scope can assign to global
// variables without a !global declaration. If when is true, a new scope is
// created; otherwise callback runs in the current scope.
//
// Matches Dart: Environment.scope
func Scope[T any](e *Environment, callback func() (T, error), semiGlobal bool, when bool) (T, error) {
	semiGlobal = semiGlobal && e.inSemiGlobalScope
	wasInSemiGlobalScope := e.inSemiGlobalScope
	e.inSemiGlobalScope = semiGlobal

	// Semi-globalness is tracked even when no frame opens, so that a rule
	// body assigning inside an @if doesn't leak into the global scope:
	// the flag, not the frame, decides.
	if !when {
		defer func() {
			e.inSemiGlobalScope = wasInSemiGlobalScope
		}()
		return callback()
	}

	e.variables = append(e.variables, orderedmap.New[string, value.Value]())
	e.variableNodes = append(e.variableNodes, orderedmap.New[string, sasscommon.AstNode]())
	e.functions = append(e.functions, orderedmap.New[string, sasscallable.Callable]())
	e.mixins = append(e.mixins, orderedmap.New[string, sasscallable.Callable]())
	if e.nestedForwardedModules != nil {
		e.nestedForwardedModules = append(e.nestedForwardedModules, make([]sassmodule.Module, 0))
	}

	defer func() {
		e.inSemiGlobalScope = wasInSemiGlobalScope
		e.lastVariableName = ""
		e.hasLastVariableIndex = false

		// Dart interleaves index cleaning with slice truncation per type
		// (environment.dart): variables.removeLast → clean indices
		// → variableNodes.removeLast → functions.removeLast → clean indices
		// → mixins.removeLast → clean indices → nestedForwardedModules.removeLast.
		for name := range e.variables[len(e.variables)-1].Keys() {
			delete(e.variableIndices, name)
		}
		e.variables = e.variables[:len(e.variables)-1]
		e.variableNodes = e.variableNodes[:len(e.variableNodes)-1]
		for name := range e.functions[len(e.functions)-1].Keys() {
			delete(e.functionIndices, name)
		}
		e.functions = e.functions[:len(e.functions)-1]
		for name := range e.mixins[len(e.mixins)-1].Keys() {
			delete(e.mixinIndices, name)
		}
		e.mixins = e.mixins[:len(e.mixins)-1]
		if e.nestedForwardedModules != nil {
			e.nestedForwardedModules = e.nestedForwardedModules[:len(e.nestedForwardedModules)-1]
		}
	}()

	return callback()
}

// ToImplicitConfiguration creates an implicit Configuration from the variables
// declared in this environment.
//
// Frame 0 contributes the imported modules' variables; deeper frames
// contribute their nested-forwarded modules' variables. Local definitions
// win over module ones at the same level since they apply second. Nodes
// travel alongside values for error spans.
//
// Matches Dart: Environment.toImplicitConfiguration
func (e *Environment) ToImplicitConfiguration() (*configuration.Configuration, error) {
	config := make(map[string]*configuration.ConfiguredValue)
	for i := 0; i < len(e.variables); i++ {
		var modules []sassmodule.Module
		if i == 0 {
			for module := range e.importedModules {
				modules = append(modules, module)
			}
		} else if e.nestedForwardedModules != nil && i-1 < len(e.nestedForwardedModules) {
			modules = e.nestedForwardedModules[i-1]
		}

		for _, module := range modules {
			for name, val := range module.Variables().Entries() {
				// Every module variable has a definition node by the
				// Module contract; a miss means the module broke it.
				// Dart: module.variableNodes[name]!
				node, ok := module.VariableNodes().Get(name)
				if !ok {
					url, err := module.URL()
					if err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("variable %q not found in module %s", name, url)
				}
				config[name] = configuration.NewConfiguredValueImplicit(val, node)
			}
		}

		values := e.variables[i]
		nodes := e.variableNodes[i]
		for name, val := range values.Entries() {
			node, _ := nodes.Get(name)
			config[name] = configuration.NewConfiguredValueImplicit(val, node)
		}
	}
	return configuration.NewConfigurationImplicit(config), nil
}

// ToModule returns a Module that represents the top-level members defined in
// this environment.
//
// The caller must be at the root: only a complete global scope converts to
// a module. The CSS, comments, and extension store become the module's own,
// while members merge frame-0 locals over the forwarded modules.
//
// Matches Dart: Environment.toModule
func (e *Environment) ToModule(css_ *value.CssStylesheet, preModuleComments map[sassmodule.Module][]value.CssComment, extensionStore extend.ExtensionStore) sassmodule.Module {
	if !e.AtRoot() {
		panic("assertion failed: atRoot must be true")
	}
	return newEnvironmentModule(e, css_, preModuleComments, extensionStore, e.forwardedModules)
}

// ToDummyModule returns a Module with the same members and upstream modules
// but an empty stylesheet and extension store.
//
// Imports resolve through dummy modules: they need the forwarded members in
// scope without any CSS. It is the only case where a nested environment
// becomes a module.
//
// Matches Dart: Environment.toDummyModule
func (e *Environment) ToDummyModule() sassmodule.Module {
	dummyStylesheet := value.NewCssStylesheet(
		[]value.CssNode{},
		sasscommon.NewSimpleFileSpan(sasscommon.NewFileSource([]byte{}, nil), 0, 0),
	)
	return newEnvironmentModule(e, dummyStylesheet, nil, &extend.EmptyExtensionStore{}, e.forwardedModules)
}

// getModule returns the module with the given namespace, failing when none
// was @use'd under it. The span-less message names the missing namespace;
// callers add spans.
//
// Matches Dart: Environment._getModule
func (e *Environment) getModule(namespace string) (sassmodule.Module, error) {
	if module, ok := e.modules[namespace]; ok {
		return module, nil
	}
	return nil, sasscommon.NewSassScriptException(
		fmt.Sprintf("There is no module with the namespace \"%s\".", namespace),
		nil)
}

// fromOneModule returns the result of callback if it returns non-nil for
// exactly one module in globalModules or for any module in importedModules
// or nestedForwardedModules.
//
// Returns the zero value if callback returns nil for all modules. Returns a
// MultiSpanSassScriptException if callback returns non-nil for more than
// one global module (with different identities).
//
// This is a free function rather than a method because Go methods can't
// have type parameters; Dart declares it as AsyncEnvironment._fromOneModule<T>.
func fromOneModule[T comparable](e *Environment, name string, typeName string, callback func(sassmodule.Module) T) (T, error) {
	var zero T
	// Nested-forwarded modules win outright: the innermost definition
	// shadows everything, so the first hit returns with no ambiguity check.
	if e.nestedForwardedModules != nil {
		for i := len(e.nestedForwardedModules) - 1; i >= 0; i-- {
			modules := e.nestedForwardedModules[i]
			for j := len(modules) - 1; j >= 0; j-- {
				if val := callback(modules[j]); val != zero {
					return val, nil
				}
			}
		}
	}

	// Imported modules behave the same: first hit wins, since @import merges
	// into one scope where later imports already shadowed earlier ones.
	for module := range e.importedModules {
		if val := callback(module); val != zero {
			return val, nil
		}
	}

	var val T
	var identity any
	// Global modules coexist, so two hits must be the same definition:
	// callables compare by reference, variables by origin identity. A
	// genuine conflict blames every @use that includes the member.
	for module := range e.globalModules {
		valInModule := callback(module)
		if valInModule == zero {
			continue
		}

		var identityFromModule any
		if c, ok := any(valInModule).(sasscallable.Callable); ok {
			identityFromModule = c
		} else {
			var err error
			identityFromModule, err = module.VariableIdentity(name)
			if err != nil {
				return zero, err
			}
		}

		if identityFromModule == identity {
			continue
		}

		if val != zero {
			spans := make(map[sasscommon.FileSpan]string)
			for mod, node := range e.globalModules {
				if callback(mod) != zero {
					s, err := node.Span()
					if err != nil {
						return zero, err
					}
					spans[s] = "includes " + typeName
				}
			}
			return zero, &sasscommon.MultiSpanSassScriptException{
				Message:        fmt.Sprintf("This %s is available from multiple global modules.", typeName),
				PrimaryLabel:   typeName + " use",
				SecondarySpans: spans,
			}
		}

		val = valInModule
		identity = identityFromModule
	}

	return val, nil
}

// moduleKeys extracts the keys of a map[Module]T as a []sassmodule.Module.
func moduleKeys[T any](m map[sassmodule.Module]T) []sassmodule.Module {
	keys := make([]sassmodule.Module, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// environmentModule is a private Module implementation representing the
// top-level members defined in an Environment: frame-0 locals filtered to
// public members, merged over the forwarded modules.
//
// Members travel by reference into lazy merged views (no eager copies), and
// modulesByVariable remembers which forwarded module owns each variable so
// writes route to the origin.
//
// Matches Dart: _EnvironmentModule
type environmentModule struct {
	environment *Environment

	upstream                       []sassmodule.Module
	css                            *value.CssStylesheet
	preModuleComments              map[sassmodule.Module][]value.CssComment
	extensionStore                 extend.ExtensionStore
	variables                      orderedmap.Map[string, value.Value]
	variableNodes                  orderedmap.Map[string, sasscommon.AstNode]
	functions                      orderedmap.Map[string, sasscallable.Callable]
	mixins                         orderedmap.Map[string, sasscallable.Callable]
	modulesByVariable              map[string]sassmodule.Module
	transitivelyContainsCss        bool
	transitivelyContainsExtensions bool
}

func newEnvironmentModule(
	env *Environment,
	css_ *value.CssStylesheet,
	preModuleComments map[sassmodule.Module][]value.CssComment,
	extensionStore extend.ExtensionStore,
	forwardedModules map[sassmodule.Module]sasscommon.AstNode,
) *environmentModule {
	// Pre-compute the modulesByVariable map and forwarded module list.
	modulesByVariable := makeModulesByVariable(forwardedModules)

	// Build member maps merging forwarded modules and local public members.
	// Forwarded maps come first so locals shadow them; memberMap wraps the
	// whole merge in lazy views.
	forwardedSlice := moduleKeys(forwardedModules)
	forwardedVarMaps := make([]orderedmap.Map[string, value.Value], len(forwardedSlice))
	forwardedVarNodeMaps := make([]orderedmap.Map[string, sasscommon.AstNode], len(forwardedSlice))
	forwardedFnMaps := make([]orderedmap.Map[string, sasscallable.Callable], len(forwardedSlice))
	forwardedMixinMaps := make([]orderedmap.Map[string, sasscallable.Callable], len(forwardedSlice))
	for i, fm := range forwardedSlice {
		forwardedVarMaps[i] = fm.Variables()
		forwardedVarNodeMaps[i] = fm.VariableNodes()
		forwardedFnMaps[i] = fm.Functions()
		forwardedMixinMaps[i] = fm.Mixins()
	}

	vars := memberMap(env.variables[0], forwardedVarMaps)
	varNodes := memberMap(env.variableNodes[0], forwardedVarNodeMaps)
	fns := memberMap(env.functions[0], forwardedFnMaps)
	mix := memberMap(env.mixins[0], forwardedMixinMaps)

	// Transitivity is sticky: CSS or extensions anywhere upstream (or in the
	// fresh tree itself) mark the whole module.
	transitiveCss := len(css_.Children()) > 0 || len(preModuleComments) > 0
	if !transitiveCss {
		for _, module := range env.allModules {
			if module.TransitivelyContainsCss() {
				transitiveCss = true
				break
			}
		}
	}

	transitiveExt := false
	if extensionStore != nil && !extensionStore.IsEmpty() {
		transitiveExt = true
	}
	if !transitiveExt {
		for _, module := range env.allModules {
			if module.TransitivelyContainsExtensions() {
				transitiveExt = true
				break
			}
		}
	}

	if preModuleComments == nil {
		preModuleComments = make(map[sassmodule.Module][]value.CssComment)
	}

	return &environmentModule{
		environment:                    env,
		upstream:                       env.allModules,
		css:                            css_,
		preModuleComments:              preModuleComments,
		extensionStore:                 extensionStore,
		variables:                      vars,
		variableNodes:                  varNodes,
		functions:                      fns,
		mixins:                         mix,
		modulesByVariable:              modulesByVariable,
		transitivelyContainsCss:        transitiveCss,
		transitivelyContainsExtensions: transitiveExt,
	}
}

// makeModulesByVariable creates the modulesByVariable map for forwarded modules.
//
// Nested environment modules flatten one level: each child's variables map
// to the child itself, while the child's own frame-0 keys map to the child
// module — private names included, since writes must still route correctly.
// This flattening keeps lookups flat instead of O(depth).
//
// Matches Dart: _EnvironmentModule._makeModulesByVariable
func makeModulesByVariable(forwardedModules map[sassmodule.Module]sasscommon.AstNode) map[string]sassmodule.Module {
	if len(forwardedModules) == 0 {
		return make(map[string]sassmodule.Module)
	}

	modulesByVariable := make(map[string]sassmodule.Module)
	for module := range forwardedModules {
		if envMod, ok := module.(*environmentModule); ok {
			// Flatten nested forwarded modules to avoid O(depth) overhead.
			for _, child := range envMod.modulesByVariable {
				for key := range child.Variables().Keys() {
					modulesByVariable[key] = child
				}
			}
			// Dart: setAll(modulesByVariable, module._environment._variables.first.keys, module)
			// includes ALL keys including private ones.
			for key := range envMod.environment.variables[0].Keys() {
				modulesByVariable[key] = module
			}
		} else {
			for key := range module.Variables().Keys() {
				modulesByVariable[key] = module
			}
		}
	}
	return modulesByVariable
}

// memberMap merges local and forwarded maps, filtering local to public members.
//
// Private locals (names starting with - or _) never leave the module,
// while forwarded maps pass through whole. Locals sit
// last so they shadow forwarded members; empty forwarded maps are skipped
// so a lone local needs no merge layer.
//
// Matches Dart: _EnvironmentModule._memberMap
func memberMap[V any](localMap *orderedmap.LinkedMap[string, V], otherMaps []orderedmap.Map[string, V]) orderedmap.Map[string, V] {
	localView := orderedmap.NewPublicMemberMapView[V](localMap)
	if len(otherMaps) == 0 {
		return localView
	}

	nonEmpty := make([]orderedmap.Map[string, V], 0, len(otherMaps))
	for _, m := range otherMaps {
		if m.Len() > 0 {
			nonEmpty = append(nonEmpty, m)
		}
	}
	if len(nonEmpty) == 0 {
		return localView
	}

	allMaps := make([]orderedmap.Map[string, V], 0, len(nonEmpty)+1)
	allMaps = append(allMaps, nonEmpty...)
	allMaps = append(allMaps, localView)
	return orderedmap.NewMergedMapView(allMaps)
}

// --- Module interface implementation ---
//
// Member accessors return the precomputed merged views; URL derives from
// the CSS tree's span. Writes route through modulesByVariable so a
// forwarded variable lands in its originating module.

// URL returns the module's source URL from its CSS span, or "" for
// string-compiled stylesheets with no URL.
func (m *environmentModule) URL() (string, error) {
	span, err := m.css.Span()
	if err != nil {
		return "", err
	}
	u, err := span.SourceURL()
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", nil
	}
	return u.String(), nil
}

func (m *environmentModule) Upstream() []sassmodule.Module {
	return m.upstream
}

func (m *environmentModule) Variables() orderedmap.Map[string, value.Value] {
	return m.variables
}

func (m *environmentModule) VariableNodes() orderedmap.Map[string, sasscommon.AstNode] {
	return m.variableNodes
}

func (m *environmentModule) Functions() orderedmap.Map[string, sasscallable.Callable] {
	return m.functions
}

func (m *environmentModule) Mixins() orderedmap.Map[string, sasscallable.Callable] {
	return m.mixins
}

func (m *environmentModule) ExtensionStore() extend.ExtensionStore {
	return m.extensionStore
}

func (m *environmentModule) CSS() (*value.CssStylesheet, error) {
	return m.css, nil
}

func (m *environmentModule) PreModuleComments() map[sassmodule.Module][]value.CssComment {
	return m.preModuleComments
}

func (m *environmentModule) TransitivelyContainsCss() bool {
	return m.transitivelyContainsCss
}

func (m *environmentModule) TransitivelyContainsExtensions() bool {
	return m.transitivelyContainsExtensions
}

// SetVariable writes through modulesByVariable when a forwarded module owns
// the name, and otherwise writes frame 0 directly — failing for names the
// module never defined.
//
// Matches Dart: _EnvironmentModule.setVariable
func (m *environmentModule) SetVariable(name string, val value.Value, nodeWithSpan sasscommon.AstNode) error {
	if module, ok := m.modulesByVariable[name]; ok {
		return module.SetVariable(name, val, nodeWithSpan)
	}

	if !m.environment.variables[0].Has(name) {
		return sasscommon.NewSassScriptException("Undefined variable.", nil)
	}

	m.environment.variables[0].Put(name, val)
	m.environment.variableNodes[0].Put(name, nodeWithSpan)
	return nil
}

// VariableIdentity delegates to the owning forwarded module when one is
// recorded, and otherwise returns the module itself: a directly defined
// variable's identity is its own module.
//
// Matches Dart: _EnvironmentModule.variableIdentity
func (m *environmentModule) VariableIdentity(name string) (any, error) {
	if !m.variables.Has(name) {
		panic(fmt.Sprintf("assertion failed: variable %s not found in module", name))
	}
	if module, ok := m.modulesByVariable[name]; ok {
		return module.VariableIdentity(name)
	}
	return m, nil
}

// CouldHaveBeenConfigured checks the module's own configurable set first,
// then the forwarded modules that overlap the query — each side iterating
// the smaller set, so tiny queries against huge modules (and vice versa)
// both stay cheap.
//
// Matches Dart: _EnvironmentModule.couldHaveBeenConfigured
func (m *environmentModule) CouldHaveBeenConfigured(variables map[string]struct{}) bool {
	// Check if this module defines a configurable variable with any of the given names.
	if len(variables) < len(m.environment.configurableVariables) {
		for name := range variables {
			if _, ok := m.environment.configurableVariables[name]; ok {
				return true
			}
		}
	} else {
		for name := range m.environment.configurableVariables {
			if _, ok := variables[name]; ok {
				return true
			}
		}
	}

	// Find forwarded modules whose variables overlap with variables and check them.
	checked := make(map[sassmodule.Module]struct{})
	if len(variables) < len(m.modulesByVariable) {
		for name := range variables {
			if module, ok := m.modulesByVariable[name]; ok {
				if _, done := checked[module]; !done {
					checked[module] = struct{}{}
					if module.CouldHaveBeenConfigured(variables) {
						return true
					}
				}
			}
		}
	} else {
		for name, module := range m.modulesByVariable {
			if _, ok := variables[name]; ok {
				if _, done := checked[module]; !done {
					checked[module] = struct{}{}
					if module.CouldHaveBeenConfigured(variables) {
						return true
					}
				}
			}
		}
	}

	return false
}

// CloneCss returns the module itself when nothing upstream contains CSS;
// otherwise it clones the CSS tree and extension store while sharing the
// member views and ownership map, which need no copying.
//
// Matches Dart: _EnvironmentModule.cloneCss
func (m *environmentModule) CloneCss() (sassmodule.Module, error) {
	if !m.transitivelyContainsCss {
		return m, nil
	}

	newCss, newExtensionStore, err := sassclonecss.CloneCssStylesheet(m.css, m.extensionStore)
	if err != nil {
		return nil, err
	}

	return &environmentModule{
		environment:                    m.environment,
		upstream:                       m.environment.allModules,
		css:                            newCss.ToCssStylesheet(),
		preModuleComments:              m.preModuleComments,
		extensionStore:                 newExtensionStore,
		modulesByVariable:              m.modulesByVariable,
		variables:                      m.variables,
		variableNodes:                  m.variableNodes,
		functions:                      m.functions,
		mixins:                         m.mixins,
		transitivelyContainsCss:        m.transitivelyContainsCss,
		transitivelyContainsExtensions: m.transitivelyContainsExtensions,
	}, nil
}

// String describes the module by its pretty source URL, or "<unknown url>"
// when it was compiled from a string without one.
//
// Matches Dart: _EnvironmentModule.toString
func (m *environmentModule) String() (string, error) {
	urlStr, err := m.URL()
	if err != nil {
		return "", err
	}
	if urlStr == "" {
		return "<unknown url>", nil
	}
	// Dart: url == null ? "<unknown url>" : p.prettyUri(url) — environment.dart:1151
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, nil
	}
	return sasscommon.PrettyUri(parsed), nil
}
