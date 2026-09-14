// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/map.dart

import (
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalMapFunctions returns the deprecated global aliases of the sass:map
// functions. Each entry wraps the module function with a deprecation warning
// (the wrapper names the original module function — see
// docs/ref/functions.md for the ordering rule) and renames it to its map-*
// spelling.
// Matches Dart: global (map.dart).
func GlobalMapFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		mapGetFunction().WithDeprecationWarning("map", nil).WithName("map-get"),
		mapMergeFunction().WithDeprecationWarning("map", nil).WithName("map-merge"),
		mapRemoveFunction().WithDeprecationWarning("map", nil).WithName("map-remove"),
		mapKeysFunction().WithDeprecationWarning("map", nil).WithName("map-keys"),
		mapValuesFunction().WithDeprecationWarning("map", nil).WithName("map-values"),
		mapHasKeyFunction().WithDeprecationWarning("map", nil).WithName("map-has-key"),
	}
}

// MapModule returns the sass:map built-in module: get, set, merge, remove,
// keys, values, has-key, and the module-only deep-merge and deep-remove.
// Matches Dart: module (map.dart).
func MapModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		mapGetFunction(),
		setFunction(),
		mapMergeFunction(),
		mapRemoveFunction(),
		mapKeysFunction(),
		mapValuesFunction(),
		mapHasKeyFunction(),
		deepMergeFunction(),
		deepRemoveFunction(),
	}
	return sassmodule.NewBuiltInModule("map", fns, nil, nil)
}

// The constructors below port Dart's bare `_function(...)` /
// `overloadedFunction(...)` closures, which carry no per-function docs; each
// note names the Sass signature and any behavior Dart documents at the call
// site. Every callable registers under the sass:map URL, porting Dart's
// _function URL helper.

// mapGetFunction implements map.get($map, $key, $keys...): the value at the
// nested key path, or null when any step misses or a midpoint is not a map.
func mapGetFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("get", "$map, $key, $keys...", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			keys := []value.Value{args[1]}
			if len(args) > 2 {
				kl, err := args[2].AsList()
				if err != nil {
					return nil, err
				}
				keys = append(keys, kl...)
			}
			for i, key := range keys {
				v, ok := m.Get(key)
				if i == len(keys)-1 {
					if !ok {
						return value.Null, nil
					}
					return v, nil
				}
				if !ok {
					return value.Null, nil
				}
				next, ok := v.(*value.SassMap)
				if !ok {
					return value.Null, nil
				}
				m = next
			}
			return value.Null, nil
		})
}

// setFunction implements the overloaded map.set: either set($map, $key,
// $value), or set($map, $args...) where the trailing rest argument is the
// value and the preceding ones form the nested key path. Updates go through
// modifyMap, creating intermediate maps as needed.
func setFunction() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("set", "sass:map",
		OverloadDef{"$map, $key, $value", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			return modifyMap(m, []value.Value{args[1]}, func(old value.Value) value.Value {
				return args[2]
			}, true), nil
		}},
		OverloadDef{"$map, $args...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			restItems, err := args[1].AsList()
			if err != nil {
				return nil, err
			}
			switch len(restItems) {
			case 0:
				return nil, sasscommon.NewSassScriptException("Expected $args to contain a key.", nil)
			case 1:
				return nil, sasscommon.NewSassScriptException("Expected $args to contain a value.", nil)
			default:
				keys := restItems[:len(restItems)-1]
				setVal := restItems[len(restItems)-1]
				return modifyMap(m, keys, func(old value.Value) value.Value {
					return setVal
				}, true), nil
			}
		}},
	)
}

// mapMergeFunction implements the overloaded map.merge: either merge($map1,
// $map2), or merge($map1, $args...) where the trailing rest argument must be
// a map and the preceding ones form the nested key path whose value is merged
// (a non-map at the path is replaced outright). An empty rest list misses a
// key, a single-element one a map. (Dart spells the same dispatch as a switch
// with an unreachable default kept only for flow analysis; the Go switch
// needs no such workaround.)
func mapMergeFunction() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("merge", "sass:map",
		OverloadDef{"$map1, $map2", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			map1, err := value.AssertMap(args[0], new("map1"))
			if err != nil {
				return nil, err
			}
			map2, err := value.AssertMap(args[1], new("map2"))
			if err != nil {
				return nil, err
			}
			merged := map1.Copy()
			for k, v := range map2.Entries() {
				merged.Set(k, v)
			}
			return merged, nil
		}},
		OverloadDef{"$map1, $args...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			map1, err := value.AssertMap(args[0], new("map1"))
			if err != nil {
				return nil, err
			}
			restItems, err := args[1].AsList()
			if err != nil {
				return nil, err
			}
			switch len(restItems) {
			case 0:
				return nil, sasscommon.NewSassScriptException("Expected $args to contain a key.", nil)
			case 1:
				return nil, sasscommon.NewSassScriptException("Expected $args to contain a map.", nil)
			default:
				keys := restItems[:len(restItems)-1]
				last := restItems[len(restItems)-1]
				map2, err := value.AssertMap(last, new("map2"))
				if err != nil {
					return nil, err
				}
				return modifyMap(map1, keys, func(old value.Value) value.Value {
					nestedMap := old.TryMap()
					if nestedMap == nil {
						return map2
					}
					merged := nestedMap.Copy()
					for k, v := range map2.Entries() {
						merged.Set(k, v)
					}
					return merged
				}, true), nil
			}
		}},
	)
}

// mapRemoveFunction implements the overloaded map.remove. The explicit $map
// overload exists because the $map, $key, $keys... spelling cannot match zero
// keys; the latter copies the map and deletes every listed key. The first key
// argument keeps its $key name so it can be passed by name.
func mapRemoveFunction() *BuiltInCallable {
	return MustNewBuiltInCallableOverloadedFunction("remove", "sass:map",
		OverloadDef{"$map", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			return value.AssertMap(args[0], new("map"))
		}},
		OverloadDef{"$map, $key, $keys...", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			keys := []value.Value{args[1]}
			if len(args) > 2 {
				kl, err := args[2].AsList()
				if err != nil {
					return nil, err
				}
				keys = append(keys, kl...)
			}
			result := m.Copy()
			for _, key := range keys {
				result.Delete(key)
			}
			return result, nil
		}},
	)
}

// mapKeysFunction implements map.keys($map): the keys as a comma-separated
// list.
func mapKeysFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("keys", "$map", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			keys := make([]value.Value, 0, m.LengthAsList())
			for k := range m.Entries() {
				keys = append(keys, k)
			}
			list, err := value.NewSassList(keys, value.ListSeparatorComma, false)
			if err != nil {
				return nil, err
			}
			return list, nil
		})
}

// mapValuesFunction implements map.values($map): the values as a
// comma-separated list.
func mapValuesFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("values", "$map", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			vals := make([]value.Value, 0, m.LengthAsList())
			for _, v := range m.Entries() {
				vals = append(vals, v)
			}
			list, err := value.NewSassList(vals, value.ListSeparatorComma, false)
			if err != nil {
				return nil, err
			}
			return list, nil
		})
}

// mapHasKeyFunction implements map.has-key($map, $key, $keys...): whether the
// nested key path resolves, stopping false at any miss or non-map midpoint.
func mapHasKeyFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("has-key", "$map, $key, $keys...", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			keys := []value.Value{args[1]}
			if len(args) > 2 {
				kl, err := args[2].AsList()
				if err != nil {
					return nil, err
				}
				keys = append(keys, kl...)
			}
			for i, key := range keys {
				v, ok := m.Get(key)
				if i == len(keys)-1 {
					if ok {
						return value.SassTrue, nil
					}
					return value.SassFalse, nil
				}
				if !ok {
					return value.SassFalse, nil
				}
				next, ok := v.(*value.SassMap)
				if !ok {
					return value.SassFalse, nil
				}
				m = next
			}
			return value.SassFalse, nil
		})
}

// deepMergeFunction implements the module-only map.deep-merge($map1, $map2)
// via deepMergeImpl.
func deepMergeFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("deep-merge", "$map1, $map2", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			map1, err := value.AssertMap(args[0], new("map1"))
			if err != nil {
				return nil, err
			}
			map2, err := value.AssertMap(args[1], new("map2"))
			if err != nil {
				return nil, err
			}
			return deepMergeImpl(map1, map2), nil
		})
}

// deepRemoveFunction implements the module-only map.deep-remove($map, $key,
// $keys...): all but the last key form the path to the nested map losing the
// final key. modifyMap runs with addNesting false, so missing intermediate
// maps leave the input unchanged, and a nested value that is absent or not a
// map passes through untouched.
func deepRemoveFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("deep-remove", "$map, $key, $keys...", "sass:map",
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			m, err := value.AssertMap(args[0], new("map"))
			if err != nil {
				return nil, err
			}
			keys := []value.Value{args[1]}
			if len(args) > 2 {
				kl, err := args[2].AsList()
				if err != nil {
					return nil, err
				}
				keys = append(keys, kl...)
			}
			return modifyMap(m, keys[:len(keys)-1], func(val value.Value) value.Value {
				nestedMap := val.TryMap()
				if nestedMap != nil {
					if _, ok := nestedMap.Get(keys[len(keys)-1]); ok {
						result := nestedMap.Copy()
						result.Delete(keys[len(keys)-1])
						return result
					}
				}
				return val
			}, false), nil
		})
}

// deepMergeImpl merges map2 into a copy of map1, with map2 values taking
// precedence. When both sides hold a map under the same key the merge recurs
// down; a recursive result identical to the existing value keeps the original
// (preserving sharing). Either side empty returns the other side directly.
func deepMergeImpl(map1, map2 *value.SassMap) *value.SassMap {
	if map1.LengthAsList() == 0 {
		return map2
	}
	if map2.LengthAsList() == 0 {
		return map1
	}
	result := map1.Copy()
	for key, val := range map2.Entries() {
		existing, ok := result.Get(key)
		if ok {
			existingMap := existing.TryMap()
			valMap := val.TryMap()
			if existingMap != nil && valMap != nil {
				merged := deepMergeImpl(existingMap, valMap)
				if merged == existingMap {
					continue
				}
				result.Set(key, merged)
			} else {
				result.Set(key, val)
			}
		} else {
			result.Set(key, val)
		}
	}
	return result
}

// modifyMap updates the value in m selected by keys through the modify
// callback, then returns the resulting map.
//
// With several keys, the keys form a path of nested maps leading to the
// targeted value passed to modify. A non-map along the path (other than the
// last key) either grows fresh nested maps when addNesting is true — with
// null passed to modify for a missing leaf — or aborts and returns m
// unchanged when false. With no keys, m itself is passed to modify.
//
// Ports Dart's _modify (map.dart).
func modifyMap(m *value.SassMap, keys []value.Value, modify func(value.Value) value.Value, addNesting bool) value.Value {
	if len(keys) == 0 {
		return modify(m)
	}
	return modifyNestedMap(m, keys, 0, modify, addNesting)
}

// modifyNestedMap is the recursive worker behind modifyMap: it copies each
// map along the key path (so the input is never mutated), applies modify at
// the leaf — passing null for a missing key — and rebuilds the spine on the
// way out. Ports the modifyNestedMap closure nested in Dart's _modify.
func modifyNestedMap(m *value.SassMap, keys []value.Value, depth int, modify func(value.Value) value.Value, addNesting bool) *value.SassMap {
	result := m.Copy()
	key := keys[depth]
	if depth == len(keys)-1 {
		old, exists := result.Get(key)
		if !exists {
			old = value.Null
		}
		result.Set(key, modify(old))
		return result
	}
	next, exists := result.Get(key)
	var nestedMap *value.SassMap
	if exists {
		nestedMap = next.TryMap()
	}
	if nestedMap == nil && !addNesting {
		return result
	}
	if nestedMap == nil {
		nestedMap = value.EmptySassMap()
	}
	result.Set(key, modifyNestedMap(nestedMap, keys, depth+1, modify, addNesting))
	return result
}
