// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package extend

// dart-source: lib/src/extend/mode.dart

// ExtendMode selects how extension applies to existing selectors.
type ExtendMode int

const (
	// ExtendModeNormal serves the @extend rule: existing selectors are
	// preserved and each target is extended individually.
	ExtendModeNormal ExtendMode = iota
	// ExtendModeReplace serves selector-replace(): existing selectors are
	// replaced and every target must match for a compound selector to extend.
	ExtendModeReplace
	// ExtendModeAllTargets serves selector-extend(): existing selectors are
	// preserved but every target must match for a compound selector to extend.
	ExtendModeAllTargets
)

var extendModeNames = [...]string{
	ExtendModeNormal:     "normal",
	ExtendModeReplace:    "replace",
	ExtendModeAllTargets: "allTargets",
}

// String returns the mode name.
func (m ExtendMode) String() string { return extendModeNames[m] }
