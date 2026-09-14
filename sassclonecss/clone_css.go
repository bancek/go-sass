// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassclonecss

// dart-source: lib/src/visitor/clone_css.dart

import (
	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/value"
)

// CloneCssStylesheet deep-copies sheet into a modifiable tree along with a
// copy of store. The cloned store supplies the selector mapping that keeps
// each copied style rule wired to its cloned extension entries; the two
// results must be used together. The store and stylesheet must come from
// the same compilation.
func CloneCssStylesheet(sheet *value.CssStylesheet, store extend.ExtensionStore) (*value.ModifiableCssStylesheet, extend.ExtensionStore, error) {
	newStore, oldToNewSelectors := store.Clone()
	v := &cloneCssVisitor{oldToNew: oldToNewSelectors}
	result, err := v.visitCssStylesheet(sheet)
	if err != nil {
		return nil, nil, err
	}
	return result, newStore, nil
}

// CloneCssNodeNoExt deep-copies one modifiable node without an extension
// store. Style rules keep their existing selector boxes instead of
// remapping them through a cloned store.
func CloneCssNodeNoExt(node value.ModifiableCssNode) (value.ModifiableCssNode, error) {
	v := &cloneCssVisitorNoExt{}
	return node.AcceptCloneModifiableCssNode(v)
}

// cloneCssVisitor deep-copies CSS nodes into mutable counterparts, mapping
// each original style-rule selector through the old-to-new table produced
// by ExtensionStore.Clone so the copy stays associated with the cloned
// store.
type cloneCssVisitor struct {
	oldToNew map[*value.SelectorList]*box.Box[*value.SelectorList]
}

// visitCssStylesheet copies the stylesheet node and re-homes every child,
// preserving group-end marks.
func (v *cloneCssVisitor) visitCssStylesheet(sheet *value.CssStylesheet) (*value.ModifiableCssStylesheet, error) {
	span, err := sheet.Span()
	if err != nil {
		return nil, err
	}
	result := value.NewModifiableCssStylesheet(span)
	for _, child := range sheet.Children() {
		if modChild, ok := child.(value.ModifiableCssNode); ok {
			newChild, err := modChild.AcceptCloneModifiableCssNode(v)
			if err != nil {
				return nil, err
			}
			newChild.SetIsGroupEnd(child.IsGroupEnd())
			if err := result.AddChild(newChild); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// VisitCloneCssStyleRule copies a style rule, swapping its selector for the
// cloned store's box. A selector missing from the map means the store and
// stylesheet came from different compilations, which panics.
func (v *cloneCssVisitor) VisitCloneCssStyleRule(node *value.ModifiableCssStyleRule) (value.ModifiableCssNode, error) {
	newSelectorBox, ok := v.oldToNew[node.Selector()]
	if !ok {
		panic("The extend.ExtensionStore and CssStylesheet passed to cloneCssStylesheet() must come from the same compilation.")
	}
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssStyleRule(newSelectorBox, span, node.OriginalSelector(), false)
	return v.visitChildren(rule, node)
}

// VisitCloneCssMediaRule copies a media rule and its children.
func (v *cloneCssVisitor) VisitCloneCssMediaRule(node *value.ModifiableCssMediaRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule, err := value.NewModifiableCssMediaRule(node.Queries(), span)
	if err != nil {
		return nil, err
	}
	return v.visitChildren(rule, node)
}

// VisitCloneCssAtRule copies an at-rule, preserving childlessness: childless
// rules copy as-is while rules with bodies gain cloned children.
func (v *cloneCssVisitor) VisitCloneCssAtRule(node *value.ModifiableCssAtRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssAtRule(node.Name(), span, node.IsChildless(), node.Value())
	if node.IsChildless() {
		return rule, nil
	}
	return v.visitChildren(rule, node)
}

// VisitCloneCssKeyframeBlock copies a keyframe block and its children.
func (v *cloneCssVisitor) VisitCloneCssKeyframeBlock(node *value.ModifiableCssKeyframeBlock) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	block := value.NewModifiableCssKeyframeBlock(node.Selector(), span)
	return v.visitChildren(block, node)
}

// VisitCloneCssSupportsRule copies a @supports rule and its children.
func (v *cloneCssVisitor) VisitCloneCssSupportsRule(node *value.ModifiableCssSupportsRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssSupportsRule(node.Condition(), span)
	return v.visitChildren(rule, node)
}

// VisitCloneCssComment copies a comment.
func (v *cloneCssVisitor) VisitCloneCssComment(node *value.ModifiableCssComment) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return value.NewModifiableCssComment(node.Text(), span), nil
}

// VisitCloneCssDeclaration copies a declaration, preserving its parsed form
// and value span.
func (v *cloneCssVisitor) VisitCloneCssDeclaration(node *value.ModifiableCssDeclaration) (value.ModifiableCssNode, error) {
	vs := node.ValueSpanForMap()
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	decl, err := value.NewModifiableCssDeclaration(node.Name(), node.Value(), span, node.ParsedAsSassScript(), &vs)
	if err != nil {
		return nil, err
	}
	return decl, nil
}

// VisitCloneCssImport copies a plain-CSS import with its URL and modifiers.
func (v *cloneCssVisitor) VisitCloneCssImport(node *value.ModifiableCssImport) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return value.NewModifiableCssImport(node.URL(), span, node.Modifiers()), nil
}

// visitChildren clones each child of oldParent into newParent, carrying
// over group-end marks, and returns newParent.
func (v *cloneCssVisitor) visitChildren(newParent value.ModifiableCssParentNode, oldParent value.CssParentNode) (value.ModifiableCssNode, error) {
	for _, child := range oldParent.Children() {
		if modChild, ok := child.(value.ModifiableCssNode); ok {
			newChild, err := modChild.AcceptCloneModifiableCssNode(v)
			if err != nil {
				return nil, err
			}
			newChild.SetIsGroupEnd(child.IsGroupEnd())
			if err := newParent.AddChild(newChild); err != nil {
				return nil, err
			}
		}
	}
	return newParent, nil
}

// cloneCssVisitorNoExt deep-copies CSS nodes without remapping selectors:
// style rules wrap their current selector in a fresh box instead of looking
// it up in a cloned store.
type cloneCssVisitorNoExt struct{}

// VisitCloneCssStyleRule copies a style rule, keeping its existing selector.
func (v *cloneCssVisitorNoExt) VisitCloneCssStyleRule(node *value.ModifiableCssStyleRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssStyleRule(box.NewModifiableBox(node.Selector()).Seal(), span, node.OriginalSelector(), false)
	return v.visitChildren(rule, node)
}

// VisitCloneCssMediaRule copies a media rule and its children.
func (v *cloneCssVisitorNoExt) VisitCloneCssMediaRule(node *value.ModifiableCssMediaRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule, err := value.NewModifiableCssMediaRule(node.Queries(), span)
	if err != nil {
		return nil, err
	}
	return v.visitChildren(rule, node)
}

// VisitCloneCssAtRule copies an at-rule, preserving childlessness.
func (v *cloneCssVisitorNoExt) VisitCloneCssAtRule(node *value.ModifiableCssAtRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssAtRule(node.Name(), span, node.IsChildless(), node.Value())
	if node.IsChildless() {
		return rule, nil
	}
	return v.visitChildren(rule, node)
}

// VisitCloneCssKeyframeBlock copies a keyframe block and its children.
func (v *cloneCssVisitorNoExt) VisitCloneCssKeyframeBlock(node *value.ModifiableCssKeyframeBlock) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	block := value.NewModifiableCssKeyframeBlock(node.Selector(), span)
	return v.visitChildren(block, node)
}

// VisitCloneCssSupportsRule copies a @supports rule and its children.
func (v *cloneCssVisitorNoExt) VisitCloneCssSupportsRule(node *value.ModifiableCssSupportsRule) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssSupportsRule(node.Condition(), span)
	return v.visitChildren(rule, node)
}

// VisitCloneCssComment copies a comment.
func (v *cloneCssVisitorNoExt) VisitCloneCssComment(node *value.ModifiableCssComment) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return value.NewModifiableCssComment(node.Text(), span), nil
}

// VisitCloneCssDeclaration copies a declaration, preserving its parsed form
// and value span.
func (v *cloneCssVisitorNoExt) VisitCloneCssDeclaration(node *value.ModifiableCssDeclaration) (value.ModifiableCssNode, error) {
	vs := node.ValueSpanForMap()
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	decl, err := value.NewModifiableCssDeclaration(node.Name(), node.Value(), span, node.ParsedAsSassScript(), &vs)
	if err != nil {
		return nil, err
	}
	return decl, nil
}

// VisitCloneCssImport copies a plain-CSS import with its URL and modifiers.
func (v *cloneCssVisitorNoExt) VisitCloneCssImport(node *value.ModifiableCssImport) (value.ModifiableCssNode, error) {
	span, err := node.Span()
	if err != nil {
		return nil, err
	}
	return value.NewModifiableCssImport(node.URL(), span, node.Modifiers()), nil
}

// visitChildren clones each child of oldParent into newParent, carrying
// over group-end marks, and returns newParent.
func (v *cloneCssVisitorNoExt) visitChildren(newParent value.ModifiableCssParentNode, oldParent value.CssParentNode) (value.ModifiableCssNode, error) {
	for _, child := range oldParent.Children() {
		if modChild, ok := child.(value.ModifiableCssNode); ok {
			newChild, err := modChild.AcceptCloneModifiableCssNode(v)
			if err != nil {
				return nil, err
			}
			newChild.SetIsGroupEnd(child.IsGroupEnd())
			if err := newParent.AddChild(newChild); err != nil {
				return nil, err
			}
		}
	}
	return newParent, nil
}
