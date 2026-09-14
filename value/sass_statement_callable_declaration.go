// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/callable_declaration.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// CallableDeclaration is a callable (function, mixin, or content block)
// declared in user code.
//
// Matches Dart: CallableDeclaration (abstract base class) — implemented by
// FunctionRule, MixinRule, and ContentBlock. Go mirrors Dart's abstract base
// with an interface so that "is this a MixinRule" is expressible via a type
// assertion (Dart: `declaration is MixinRule`; Rust: CallableDeclaration enum).
type CallableDeclaration interface {
	// Name is the callable name with underscores folded to hyphens.
	Name() string
	// OriginalName is the callable name as written, before folding.
	OriginalName() string
	// Parameters are the declared parameters the callable accepts.
	Parameters() *ParameterList
	Span() (sasscommon.FileSpan, error)
	// Comment is the doc comment immediately preceding the declaration,
	// or nil when there is none.
	Comment() *SilentComment
	// GetChildren returns the statements in the callable body.
	GetChildren() []Statement
	// HasDeclarations reports whether the body declares variables,
	// functions, mixins, or dynamic imports.
	HasDeclarations() bool
	IsStatement()
	IsSassNode()
	IsAstNode()
}

// callableDeclaration is the shared state for a callable declaration.
// It's the Go analog of Dart's CallableDeclaration abstract base class fields.
type callableDeclaration struct {
	parent       ParentStatement
	originalName string
	parameters   *ParameterList
	comment      *SilentComment
	name         string
	span         sasscommon.FileSpan
}

// newCallableDeclaration shares the common callable state, folding
// underscores in the original name to hyphens for the canonical name.
func newCallableDeclaration(originalName string, parameters *ParameterList, children []Statement, span sasscommon.FileSpan, comment *SilentComment) *callableDeclaration {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &callableDeclaration{
		parent:       NewParentStatement(c),
		originalName: originalName,
		parameters:   parameters,
		name:         strings.ReplaceAll(originalName, "_", "-"),
		comment:      comment,
		span:         span,
	}
}

func (d *callableDeclaration) Name() string                       { return d.name }
func (d *callableDeclaration) OriginalName() string               { return d.originalName }
func (d *callableDeclaration) Parameters() *ParameterList         { return d.parameters }
func (d *callableDeclaration) Span() (sasscommon.FileSpan, error) { return d.span, nil }
func (d *callableDeclaration) Comment() *SilentComment            { return d.comment }
func (d *callableDeclaration) IsStatement()                       {}
func (d *callableDeclaration) IsSassNode()                        {}
func (d *callableDeclaration) IsAstNode()                         {}
func (d *callableDeclaration) GetChildren() []Statement           { return d.parent.Children }
func (d *callableDeclaration) HasDeclarations() bool              { return d.parent.HasDeclarations() }

func (d *callableDeclaration) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	panic("BUG: callableDeclaration.AcceptValue called directly; use FunctionRule, MixinRule, or ContentBlock")
}

func (d *callableDeclaration) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	panic("BUG: callableDeclaration.AcceptBool called directly; use FunctionRule, MixinRule, or ContentBlock")
}

func (d *callableDeclaration) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	panic("BUG: callableDeclaration.AcceptVoid called directly; use FunctionRule, MixinRule, or ContentBlock")
}
