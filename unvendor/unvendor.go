// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package unvendor

// dart-source: lib/src/utils.dart

// Unvendor strips the vendor prefix from a CSS name: the text after the
// second dash.
//
// For example, Unvendor("-webkit-keyframes") returns "keyframes".
// Names with no vendor prefix are returned unchanged, including
// "--custom" properties (whose second character is also a dash) and
// names with no second dash at all.
//
// Matches Dart: unvendor (utils.dart)
func Unvendor(name string) string {
	if len(name) < 2 || name[0] != '-' || name[1] == '-' {
		return name
	}
	for i := 2; i < len(name); i++ {
		if name[i] == '-' {
			return name[i+1:]
		}
	}
	return name
}
