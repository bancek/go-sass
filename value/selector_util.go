// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector.dart

// IsParentSelector reports whether s is a *ParentSelector. Nesting
// resolution uses it to decide whether a compound's leading simple selector
// needs parent substitution.
func IsParentSelector(s SimpleSelector) bool {
	_, ok := s.(*ParentSelector)
	return ok
}
