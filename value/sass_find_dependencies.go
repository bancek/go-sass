// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/find_dependencies.dart

import (
	"maps"
	"net/url"
)

// FindDependencies returns stylesheet's statically-declared dependencies:
// @use/@forward URLs, meta.load-css() URLs with static string arguments,
// and dynamic @import URLs.
//
// Matches Dart: findDependencies
func FindDependencies(stylesheet *Stylesheet) *DependencyReport {
	v := &findDependenciesVisitor{
		uses:           map[string]*url.URL{},
		forwards:       map[string]*url.URL{},
		metaLoadCss:    map[string]*url.URL{},
		imports:        map[string]*url.URL{},
		metaNamespaces: map[string]struct{}{},
	}
	v.VisitStylesheet(stylesheet)
	return &DependencyReport{
		Uses:        v.uses,
		Forwards:    v.forwards,
		MetaLoadCss: v.metaLoadCss,
		Imports:     v.imports,
	}
}

// DependencyReport holds the different kinds of dependencies a Sass
// stylesheet can contain. Each set is keyed by URL string, so duplicates
// collapse the way Dart's Set<Uri> does.
//
// Matches Dart: DependencyReport
type DependencyReport struct {
	// Uses holds every @use'd URL, excluding built-in modules.
	Uses map[string]*url.URL
	// Forwards holds every @forward'ed URL, excluding built-in modules.
	Forwards map[string]*url.URL
	// MetaLoadCss holds URLs from meta.load-css() calls with static string
	// arguments outside of mixins.
	MetaLoadCss map[string]*url.URL
	// Imports holds dynamically @import'ed URLs.
	Imports map[string]*url.URL
}

// Modules returns the union of Uses, Forwards, and MetaLoadCss.
//
// Matches Dart: DependencyReport.modules
func (r *DependencyReport) Modules() []*url.URL {
	return mergeURISets(r.Uses, r.Forwards, r.MetaLoadCss)
}

// All returns the union of Uses, Forwards, MetaLoadCss, and Imports.
//
// Matches Dart: DependencyReport.all
func (r *DependencyReport) All() []*url.URL {
	return mergeURISets(r.Uses, r.Forwards, r.MetaLoadCss, r.Imports)
}

// mergeURISets unions URL sets keyed by string, deduplicating repeats.
func mergeURISets(sets ...map[string]*url.URL) []*url.URL {
	seen := map[string]*url.URL{}
	for _, set := range sets {
		maps.Copy(seen, set)
	}
	result := make([]*url.URL, 0, len(seen))
	for _, u := range seen {
		result = append(result, u)
	}
	return result
}

// findDependenciesVisitor traverses a stylesheet and records every
// dependency on other stylesheets.
//
// Matches Dart: _FindDependenciesVisitor. Unlike Dart's RecursiveStatement
// mixin, this visitor implements StatementVisitor directly and recurses via
// visitChildren, so overrides dispatch to these methods.
type findDependenciesVisitor struct {
	uses        map[string]*url.URL
	forwards    map[string]*url.URL
	metaLoadCss map[string]*url.URL
	imports     map[string]*url.URL
	// The namespaces under which sass:meta has been @use'd in this stylesheet.
	// If this contains "", it means sass:meta was loaded without a namespace.
	metaNamespaces map[string]struct{}
}

// Compile-time assertion that findDependenciesVisitor implements
// StatementVisitor[struct{}] directly, so the recursive children walk
// dispatches to its overrides (unlike embedding RecursiveStatementVisitor,
// whose methods would shadow them by name).
var _ StatementVisitor[struct{}] = (*findDependenciesVisitor)(nil)

// visitChildren walks child statements through this visitor's own Visit
// methods.
func (v *findDependenciesVisitor) visitChildren(children []Statement) error {
	for _, child := range children {
		if _, err := child.AcceptVoid(v); err != nil {
			return err
		}
	}
	return nil
}

// VisitAtRootRule recurses into at-root children.
func (v *findDependenciesVisitor) VisitAtRootRule(node *AtRootRule) (struct{}, error) {
	return struct{}{}, v.visitChildren(node.GetChildren())
}

// VisitAtRule recurses into at-rule children when the rule has a block.
func (v *findDependenciesVisitor) VisitAtRule(node *AtRule) (struct{}, error) {
	if node.GetChildren() != nil {
		return struct{}{}, v.visitChildren(node.GetChildren())
	}
	return struct{}{}, nil
}

// The rules below can never contain imports, so they are visited as no-ops.

func (v *findDependenciesVisitor) VisitContentBlock(*ContentBlock) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitContentRule(*ContentRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitDebugRule(*DebugRule) (struct{}, error) {
	return struct{}{}, nil
}

// VisitDeclaration recurses into declaration children when the declaration
// has a block.
func (v *findDependenciesVisitor) VisitDeclaration(node *Declaration) (struct{}, error) {
	if node.GetChildren() != nil {
		return struct{}{}, v.visitChildren(node.GetChildren())
	}
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitEachRule(*EachRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitErrorRule(*ErrorRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitExtendRule(*ExtendRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitForRule(*ForRule) (struct{}, error) {
	return struct{}{}, nil
}

// VisitForwardRule records non-built-in forwarded URLs.
func (v *findDependenciesVisitor) VisitForwardRule(node *ForwardRule) (struct{}, error) {
	if node.URL().Scheme != "sass" {
		v.forwards[node.URL().String()] = node.URL()
	}
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitFunctionRule(*FunctionRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitIfRule(*IfRule) (struct{}, error) {
	return struct{}{}, nil
}

// VisitImportRule records the URLs of dynamic imports; static imports stay
// as plain CSS and are not dependencies.
func (v *findDependenciesVisitor) VisitImportRule(node *ImportRule) (struct{}, error) {
	for _, imp := range node.Imports {
		if d, ok := imp.(*DynamicImport); ok {
			v.imports[d.URL().String()] = d.URL()
		}
	}
	return struct{}{}, nil
}

// VisitIncludeRule records meta.load-css() URLs: only includes named
// load-css under a tracked sass:meta namespace, with a static plain-string
// first argument, count. Invalid URLs are ignored.
func (v *findDependenciesVisitor) VisitIncludeRule(node *IncludeRule) (struct{}, error) {
	if node.Name() != "load-css" {
		return struct{}{}, nil
	}
	ns := ""
	if node.Namespace() != nil {
		ns = *node.Namespace()
	}
	if _, ok := v.metaNamespaces[ns]; !ok {
		return struct{}{}, nil
	}
	args := node.Arguments()
	if len(args.Positional) == 0 {
		return struct{}{}, nil
	}
	if se, ok := args.Positional[0].(*StringExpression); ok {
		if plain := se.Text.AsPlain(); plain != nil {
			u, err := url.Parse(*plain)
			if err == nil {
				v.metaLoadCss[u.String()] = u
			}
		}
	}
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitLoudComment(*LoudComment) (struct{}, error) {
	return struct{}{}, nil
}

// VisitMediaRule recurses into media-rule children.
func (v *findDependenciesVisitor) VisitMediaRule(node *MediaRule) (struct{}, error) {
	return struct{}{}, v.visitChildren(node.GetChildren())
}

func (v *findDependenciesVisitor) VisitMixinRule(*MixinRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitReturnRule(*ReturnRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitSilentComment(*SilentComment) (struct{}, error) {
	return struct{}{}, nil
}

// VisitStyleRule recurses into style-rule children.
func (v *findDependenciesVisitor) VisitStyleRule(node *StyleRule) (struct{}, error) {
	return struct{}{}, v.visitChildren(node.GetChildren())
}

// VisitStylesheet recurses into the stylesheet's top-level children.
func (v *findDependenciesVisitor) VisitStylesheet(node *Stylesheet) (struct{}, error) {
	return struct{}{}, v.visitChildren(node.GetChildren())
}

// VisitSupportsRule recurses into supports-rule children.
func (v *findDependenciesVisitor) VisitSupportsRule(node *SupportsRule) (struct{}, error) {
	return struct{}{}, v.visitChildren(node.GetChildren())
}

// VisitUseRule records non-built-in @use URLs and tracks the namespaces
// under which sass:meta itself was loaded for later load-css matching.
func (v *findDependenciesVisitor) VisitUseRule(node *UseRule) (struct{}, error) {
	if node.URL().Scheme != "sass" {
		v.uses[node.URL().String()] = node.URL()
	} else if node.URL().String() == "sass:meta" {
		ns := ""
		if node.Namespace != nil {
			ns = *node.Namespace
		}
		v.metaNamespaces[ns] = struct{}{}
	}
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitVariableDeclaration(*VariableDeclaration) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitWarnRule(*WarnRule) (struct{}, error) {
	return struct{}{}, nil
}

func (v *findDependenciesVisitor) VisitWhileRule(*WhileRule) (struct{}, error) {
	return struct{}{}, nil
}
