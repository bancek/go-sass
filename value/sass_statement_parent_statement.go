// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/parent.dart

// ParentStatement is a statement node that carries child statements.
//
// Concrete rules embed it to share the children list and the precomputed
// declaration scan. A nil Children means the rule ends with a semicolon;
// an empty non-nil slice means an empty block.
type ParentStatement struct {
	// Children are the nested statements, or nil when the rule has no block.
	Children        []Statement
	hasDeclarations bool
}

// NewParentStatement creates a ParentStatement and computes hasDeclarations.
//
// Matches Dart: ParentStatement constructor
func NewParentStatement(children []Statement) ParentStatement {
	return ParentStatement{
		Children:        children,
		hasDeclarations: hasDeclarations(children),
	}
}

// HasDeclarations returns whether any child is a variable, function, or mixin
// declaration, or a dynamic import rule.
//
// Matches Dart: ParentStatement.hasDeclarations
func (ps *ParentStatement) HasDeclarations() bool {
	return ps.hasDeclarations
}

func hasDeclarations(children []Statement) bool {
	for _, child := range children {
		switch child.(type) {
		case *VariableDeclaration, *FunctionRule, *MixinRule:
			return true
		case *ImportRule:
			rule := child.(*ImportRule)
			for _, imp := range rule.Imports {
				if _, ok := imp.(*DynamicImport); ok {
					return true
				}
			}
		}
	}
	return false
}
