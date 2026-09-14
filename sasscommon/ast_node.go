// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/ast/node.dart

// AstNode is implemented by every abstract-syntax-tree node.
//
// Each node carries the source span where it was defined, used for error
// reporting and source-map generation. Debugging string forms generally
// reflect source text but must never be reused to regenerate Sass source:
// they are untested and not stable across versions.
//
// It ports Dart's sealed AstNode interface; the Go marker method IsAstNode
// plays the role of Dart's sealing, keeping the implementor set closed.
//
// Matches Dart: AstNode (lib/src/ast/node.dart, AST category)
type AstNode interface {
	// Span returns the source span where the node was defined.
	Span() (FileSpan, error)
	// IsAstNode seals the interface: only types in this module set (and
	// the AST packages built on it) may implement AstNode, mirroring
	// Dart's @sealed annotation.
	IsAstNode()
}

// FakeAstNode is an AstNode whose span is produced by a callback on demand.
// APIs that take nodes instead of spans use it to accept lazily-computed
// locations: computing spans eagerly can be expensive, so the callback form
// lets callers pass arbitrary spans that materialize only when read.
//
// The constructor is internal in Dart (AstNode.fake is @nodoc/@internal);
// it stays exported here only so tests and span-accepting APIs can use it.
//
// Matches Dart: AstNode.fake / _FakeAstNode
type FakeAstNode struct {
	callback func() (FileSpan, error)
}

// NewFakeAstNode creates a FakeAstNode whose Span runs callback on each call.
// The callback is invoked lazily (never at construction), so expensive span
// work is deferred until the location is actually needed.
func NewFakeAstNode(callback func() (FileSpan, error)) *FakeAstNode {
	return &FakeAstNode{callback: callback}
}

// Span runs the lazy callback to produce the node's span.
func (n *FakeAstNode) Span() (FileSpan, error) { return n.callback() }

// IsAstNode marks FakeAstNode as an AstNode.
func (n *FakeAstNode) IsAstNode() {}
