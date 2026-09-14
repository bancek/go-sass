// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/parse/css.dart (_disallowedFunctionNames section)
// DisallowedFunctionNames returns the set of function names that may not be
// called in plain CSS: every global built-in name except those that are also
// valid native CSS functions (abs, rgb, max, and the like), which must still
// parse as plain CSS calls. The CSS parser rejects any other bare call whose
// name lands in this set.
//
// Matches Dart: _disallowedFunctionNames in lib/src/parse/css.dart, built
// from globalFunctions in lib/src/functions.dart
func DisallowedFunctionNames() map[string]bool {
	set := map[string]bool{}
	for _, fn := range GlobalFunctions() {
		set[fn.Name()] = true
	}
	// Each name removed here doubles as a native CSS function, so it stays
	// legal in plain CSS. Matches Dart: the ..remove() cascade on
	// _disallowedFunctionNames.
	delete(set, "abs")
	delete(set, "alpha")
	delete(set, "color")
	delete(set, "grayscale")
	delete(set, "hsl")
	delete(set, "hsla")
	delete(set, "hwb")
	delete(set, "invert")
	delete(set, "lab")
	delete(set, "lch")
	delete(set, "max")
	delete(set, "min")
	delete(set, "oklab")
	delete(set, "oklch")
	delete(set, "opacity")
	delete(set, "rgb")
	delete(set, "rgba")
	delete(set, "round")
	delete(set, "saturate")
	return set
}
