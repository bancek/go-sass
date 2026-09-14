// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/statement_search.dart

// StatementSearchVisitor is a StatementVisitor that reports whether any
// visited node matches.
//
// Every visit method returns false by default and short-circuits to true on
// the first match while recursing through children. Extend it (or set
// ContentRuleFunc) to recognize particular nodes and find their first
// occurrence in the tree.
//
// It supports the same additional methods as RecursiveStatementVisitor.
//
// ContentRuleFunc, if non-nil, is called instead of the default VisitContentRule
// behavior. This allows sub-visitors to override the ContentRule result without
// needing to implement all StatementVisitor methods. Matches Dart's pattern of
// extending StatementSearchVisitor and overriding visitContentRule.
type StatementSearchVisitor struct {
	ContentRuleFunc func(*ContentRule) (bool, error)
}

func (v *StatementSearchVisitor) VisitAtRootRule(node *AtRootRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitAtRule(node *AtRule) (bool, error) {
	if node.GetChildren() == nil {
		return false, nil
	}
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitContentBlock(node *ContentBlock) (bool, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *StatementSearchVisitor) VisitContentRule(node *ContentRule) (bool, error) {
	if v.ContentRuleFunc != nil {
		return v.ContentRuleFunc(node)
	}
	return false, nil
}

func (v *StatementSearchVisitor) VisitDebugRule(*DebugRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitDeclaration(node *Declaration) (bool, error) {
	if node.GetChildren() == nil {
		return false, nil
	}
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitEachRule(node *EachRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitErrorRule(*ErrorRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitExtendRule(*ExtendRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitForRule(node *ForRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitForwardRule(*ForwardRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitFunctionRule(node *FunctionRule) (bool, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *StatementSearchVisitor) VisitIfRule(node *IfRule) (bool, error) {
	for _, clause := range node.Clauses {
		for _, child := range clause.Children() {
			result, err := child.AcceptBool(v)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
	}
	if node.LastClause != nil {
		for _, child := range node.LastClause.Children() {
			result, err := child.AcceptBool(v)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
	}
	return false, nil
}

func (v *StatementSearchVisitor) VisitImportRule(*ImportRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitIncludeRule(node *IncludeRule) (bool, error) {
	if node.Content == nil {
		return false, nil
	}
	return v.VisitContentBlock(node.Content)
}

func (v *StatementSearchVisitor) VisitLoudComment(*LoudComment) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitMediaRule(node *MediaRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitMixinRule(node *MixinRule) (bool, error) {
	return v.visitCallableDeclaration(node.Declaration())
}

func (v *StatementSearchVisitor) VisitReturnRule(*ReturnRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitSilentComment(*SilentComment) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitStyleRule(node *StyleRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitStylesheet(node *Stylesheet) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitSupportsRule(node *SupportsRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) VisitUseRule(*UseRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitVariableDeclaration(*VariableDeclaration) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitWarnRule(*WarnRule) (bool, error) {
	return false, nil
}

func (v *StatementSearchVisitor) VisitWhileRule(node *WhileRule) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) visitCallableDeclaration(node CallableDeclaration) (bool, error) {
	return v.visitChildren(node.GetChildren())
}

func (v *StatementSearchVisitor) visitChildren(children []Statement) (bool, error) {
	for _, child := range children {
		result, err := child.AcceptBool(v)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}
