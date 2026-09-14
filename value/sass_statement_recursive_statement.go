// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/recursive_statement.dart

// RecursiveStatementVisitor walks every statement in a Sass AST.
//
// Leaf statements are no-ops; parents recurse into their children, with
// @if clauses and mixin content handled elementwise. Embed it and override
// individual Visit methods to add behavior for specific node types.
//
// The shared helpers visitCallableDeclaration and visitChildren recurse into
// callable bodies and child lists; overrides usually call them to keep the
// default traversal.
type RecursiveStatementVisitor struct{}

func (v *RecursiveStatementVisitor) VisitAtRootRule(node *AtRootRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitAtRule(node *AtRule) (struct{}, error) {
	if node.GetChildren() != nil {
		if err := v.visitChildren(node.GetChildren()); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitContentBlock(node *ContentBlock) (struct{}, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *RecursiveStatementVisitor) VisitContentRule(*ContentRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitDebugRule(*DebugRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitDeclaration(node *Declaration) (struct{}, error) {
	if node.GetChildren() != nil {
		if err := v.visitChildren(node.GetChildren()); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitEachRule(node *EachRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitErrorRule(*ErrorRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitExtendRule(*ExtendRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitForRule(node *ForRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitForwardRule(*ForwardRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitFunctionRule(node *FunctionRule) (struct{}, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *RecursiveStatementVisitor) VisitIfRule(node *IfRule) (struct{}, error) {
	for _, clause := range node.Clauses {
		for _, child := range clause.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return struct{}{}, err
			}
		}
	}
	if node.LastClause != nil {
		for _, child := range node.LastClause.Children() {
			if _, err := child.AcceptVoid(v); err != nil {
				return struct{}{}, err
			}
		}
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitImportRule(*ImportRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitIncludeRule(node *IncludeRule) (struct{}, error) {
	if node.Content != nil {
		return v.VisitContentBlock(node.Content)
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitLoudComment(*LoudComment) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitMediaRule(node *MediaRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitMixinRule(node *MixinRule) (struct{}, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *RecursiveStatementVisitor) VisitReturnRule(*ReturnRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitSilentComment(*SilentComment) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitStyleRule(node *StyleRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitStylesheet(node *Stylesheet) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitSupportsRule(node *SupportsRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitUseRule(*UseRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitVariableDeclaration(*VariableDeclaration) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitWarnRule(*WarnRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) VisitWhileRule(node *WhileRule) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) visitCallableDeclaration(node CallableDeclaration) (struct{}, error) {
	if err := v.visitChildren(node.GetChildren()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

func (v *RecursiveStatementVisitor) visitChildren(children []Statement) error {
	for _, child := range children {
		if _, err := child.AcceptVoid(v); err != nil {
			return err
		}
	}
	return nil
}
