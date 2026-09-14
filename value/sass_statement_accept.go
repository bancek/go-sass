// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/*.dart (Accept methods)

// This file holds the Accept dispatch for every statement node. Each node
// exposes one method per visitor result shape (Value, bool, struct{}), each
// forwarding to the matching Visit method. The per-node accept logic lives
// with its Dart class; it is gathered here only to keep the node files small.

func (r *AtRootRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitAtRootRule(r)
}

func (r *AtRootRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitAtRootRule(r)
}

func (r *AtRootRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitAtRootRule(r)
}

func (r *AtRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitAtRule(r)
}

func (r *AtRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitAtRule(r)
}

func (r *AtRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitAtRule(r)
}

func (r *ContentBlock) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitContentBlock(r)
}

func (r *ContentBlock) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitContentBlock(r)
}

func (r *ContentBlock) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitContentBlock(r)
}

func (r *ContentRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitContentRule(r)
}

func (r *ContentRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitContentRule(r)
}

func (r *ContentRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitContentRule(r)
}

func (r *DebugRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitDebugRule(r)
}

func (r *DebugRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitDebugRule(r)
}

func (r *DebugRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitDebugRule(r)
}

func (r *Declaration) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitDeclaration(r)
}

func (r *Declaration) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitDeclaration(r)
}

func (r *Declaration) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitDeclaration(r)
}

func (r *EachRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitEachRule(r)
}

func (r *EachRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitEachRule(r)
}

func (r *EachRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitEachRule(r)
}

func (r *ErrorRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitErrorRule(r)
}

func (r *ErrorRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitErrorRule(r)
}

func (r *ErrorRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitErrorRule(r)
}

func (r *ExtendRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitExtendRule(r)
}

func (r *ExtendRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitExtendRule(r)
}

func (r *ExtendRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitExtendRule(r)
}

func (r *ForRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitForRule(r)
}

func (r *ForRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitForRule(r)
}

func (r *ForRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitForRule(r)
}

func (r *ForwardRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitForwardRule(r)
}

func (r *ForwardRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitForwardRule(r)
}

func (r *ForwardRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitForwardRule(r)
}

func (r *FunctionRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitFunctionRule(r)
}

func (r *FunctionRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitFunctionRule(r)
}

func (r *FunctionRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitFunctionRule(r)
}

func (r *IfRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitIfRule(r)
}

func (r *IfRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitIfRule(r)
}

func (r *IfRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitIfRule(r)
}

func (r *ImportRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitImportRule(r)
}

func (r *ImportRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitImportRule(r)
}

func (r *ImportRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitImportRule(r)
}

func (r *IncludeRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitIncludeRule(r)
}

func (r *IncludeRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitIncludeRule(r)
}

func (r *IncludeRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitIncludeRule(r)
}

func (r *LoudComment) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitLoudComment(r)
}

func (r *LoudComment) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitLoudComment(r)
}

func (r *LoudComment) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitLoudComment(r)
}

func (r *MediaRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitMediaRule(r)
}

func (r *MediaRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitMediaRule(r)
}

func (r *MediaRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitMediaRule(r)
}

func (r *MixinRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitMixinRule(r)
}

func (r *MixinRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitMixinRule(r)
}

func (r *MixinRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitMixinRule(r)
}

func (r *ReturnRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitReturnRule(r)
}

func (r *ReturnRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitReturnRule(r)
}

func (r *ReturnRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitReturnRule(r)
}

func (r *SilentComment) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitSilentComment(r)
}

func (r *SilentComment) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitSilentComment(r)
}

func (r *SilentComment) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitSilentComment(r)
}

func (r *StyleRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitStyleRule(r)
}

func (r *StyleRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitStyleRule(r)
}

func (r *StyleRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitStyleRule(r)
}

func (r *Stylesheet) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitStylesheet(r)
}

func (r *Stylesheet) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitStylesheet(r)
}

func (r *Stylesheet) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitStylesheet(r)
}

func (r *SupportsRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitSupportsRule(r)
}

func (r *SupportsRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitSupportsRule(r)
}

func (r *SupportsRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitSupportsRule(r)
}

func (r *UseRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitUseRule(r)
}

func (r *UseRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitUseRule(r)
}

func (r *UseRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitUseRule(r)
}

func (r *VariableDeclaration) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitVariableDeclaration(r)
}

func (r *VariableDeclaration) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitVariableDeclaration(r)
}

func (r *VariableDeclaration) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitVariableDeclaration(r)
}

func (r *WarnRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitWarnRule(r)
}

func (r *WarnRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitWarnRule(r)
}

func (r *WarnRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitWarnRule(r)
}

func (r *WhileRule) AcceptValue(v StatementVisitor[Value]) (Value, error) {
	return v.VisitWhileRule(r)
}

func (r *WhileRule) AcceptBool(v StatementVisitor[bool]) (bool, error) {
	return v.VisitWhileRule(r)
}

func (r *WhileRule) AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error) {
	return v.VisitWhileRule(r)
}
