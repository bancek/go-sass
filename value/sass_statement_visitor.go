// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/statement.dart

// StatementVisitor traverses Sass statement nodes.
//
// Each method handles one statement type and returns a caller-chosen result,
// so the same traversal shape serves value-producing, predicate, and
// effectful passes. Nodes dispatch through their Accept methods.
type StatementVisitor[T any] interface {
	VisitAtRootRule(*AtRootRule) (T, error)
	VisitAtRule(*AtRule) (T, error)
	VisitContentBlock(*ContentBlock) (T, error)
	VisitContentRule(*ContentRule) (T, error)
	VisitDebugRule(*DebugRule) (T, error)
	VisitDeclaration(*Declaration) (T, error)
	VisitEachRule(*EachRule) (T, error)
	VisitErrorRule(*ErrorRule) (T, error)
	VisitExtendRule(*ExtendRule) (T, error)
	VisitForRule(*ForRule) (T, error)
	VisitForwardRule(*ForwardRule) (T, error)
	VisitFunctionRule(*FunctionRule) (T, error)
	VisitIfRule(*IfRule) (T, error)
	VisitImportRule(*ImportRule) (T, error)
	VisitIncludeRule(*IncludeRule) (T, error)
	VisitLoudComment(*LoudComment) (T, error)
	VisitMediaRule(*MediaRule) (T, error)
	VisitMixinRule(*MixinRule) (T, error)
	VisitReturnRule(*ReturnRule) (T, error)
	VisitSilentComment(*SilentComment) (T, error)
	VisitStyleRule(*StyleRule) (T, error)
	VisitStylesheet(*Stylesheet) (T, error)
	VisitSupportsRule(*SupportsRule) (T, error)
	VisitUseRule(*UseRule) (T, error)
	VisitVariableDeclaration(*VariableDeclaration) (T, error)
	VisitWarnRule(*WarnRule) (T, error)
	VisitWhileRule(*WhileRule) (T, error)
}
