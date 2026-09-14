// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/combinator.dart

// Combinator describes how two compound selectors in a ComplexSelector are
// related to each other in the DOM tree.
type Combinator int

const (
	// CombinatorNextSibling matches when the right-hand selector immediately
	// follows the left-hand selector: `+`.
	CombinatorNextSibling Combinator = iota // +
	// CombinatorChild matches when the right-hand selector is a direct child
	// of the left-hand selector: `>`.
	CombinatorChild // >
	// CombinatorFollowingSibling matches when the right-hand selector comes
	// anywhere after the left-hand selector among siblings: `~`.
	CombinatorFollowingSibling // ~
)

// String returns the combinator's token text.
func (c Combinator) String() string {
	switch c {
	case CombinatorNextSibling:
		return "+"
	case CombinatorChild:
		return ">"
	case CombinatorFollowingSibling:
		return "~"
	default:
		return "?"
	}
}
