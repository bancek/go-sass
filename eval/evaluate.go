// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (visitor core, config/state threading;
// evaluate() entry point, Evaluator, _EvaluateVisitor fields, run/_execute,
// _combineCss/_extendModules/_throwForUnsatisfiedExtension, _warn, _EvaluationContext)

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/configuration"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// cssMathFunctionNames contains the names of CSS math functions that are
// evaluated specially rather than as normal Sass function calls.
var cssMathFunctionNames = map[string]bool{
	"calc": true, "min": true, "max": true, "clamp": true,
	"round": true, "mod": true, "rem": true,
	"sin": true, "cos": true, "tan": true,
	"asin": true, "acos": true, "atan": true, "atan2": true,
	"pow": true, "sqrt": true, "hypot": true,
	"log": true, "exp": true,
	"abs": true, "sign": true,
	"calc-size": true,
}

// stackFrame is one frame of the dynamic call stack: function and mixin
// invocations plus imports surrounding the current context. Each frame pairs
// the human-readable member name with the node whose span marks where the
// trace starts and the callable being invoked, forming a parent-linked chain
// the evaluator walks to build Sass stack traces.
type stackFrame struct {
	Name     string
	Span     sasscommon.FileSpan
	Callable sasscallable.Callable
	Parent   *stackFrame
}

// warnKey identifies an emitted warning for dedup. The evaluator keeps one
// entry per message/location pair so a repeated warning does not flood the
// console; FileSpan is structurally comparable here, so it serves directly
// as the map key alongside the message.
type warnKey struct {
	message string
	span    sasscommon.FileSpan
}

// EvaluateVisitor executes Sass code to produce a CSS tree.
//
// It holds every piece of state a compilation needs: the lexical
// environment, the CSS output parent chain, module and importer state,
// warning dedup, and the call stack used for traces. Where Dart keeps some
// of this in zones, Go threads it explicitly through the visitor and the
// evaluation context. One visitor serves one compilation and dispatches
// through typed visit/eval methods rather than implementing the Sass AST
// visitor interfaces.
type EvaluateVisitor struct {
	ec          *evalcontext.EvaluationContext
	env         *sassenv.Environment
	importCache *ImportCache
	logger      sasslogger.Logger

	// modules holds every module loaded and evaluated so far, indexed by
	// canonical URL.
	modules map[string]sassmodule.Module
	// moduleConfigurations records the first configuration used to load each
	// module URL, so a later conflicting `with` can report the original load.
	moduleConfigurations map[string]*configuration.Configuration
	// moduleNodes maps canonical module URLs to the nodes whose spans mark
	// where those modules were originally loaded. There is not guaranteed to
	// be an entry for every module: the entrypoint, for example, was not
	// loaded by a node.
	moduleNodes map[string]sasscommon.AstNode
	// builtInModules holds built-in modules indexed by URL. They are always
	// treated as already loaded since they need no execution and produce no CSS.
	builtInModules map[string]sassmodule.Module
	// builtInFunctions holds globally accessible built-ins, even under the
	// module system, keyed by hyphen-normalized name. User functions are
	// registered first so built-ins win on collision.
	builtInFunctions map[string]sasscallable.Callable

	// sourceMap records whether to track source locations of variable
	// declarations.
	sourceMap bool
	// quietDeps suppresses warnings from dependency stylesheets.
	quietDeps bool

	// stack is the dynamic call stack of function/mixin invocations and
	// imports surrounding the current context.
	stack *stackFrame
	// styleRuleIgnoringAtRoot is the style rule defining the current parent
	// selector, ignoring intermediate @at-root rules. Most callers want the
	// style-rule view that blanks this while directly inside an @at-root
	// rule excluding style rules.
	styleRuleIgnoringAtRoot value.CssStyleRule
	// atRootExcludingStyleRule reports whether evaluation is directly within
	// an @at-root rule that excludes style rules.
	atRootExcludingStyleRule bool
	// mediaQueries holds the current media queries, if any.
	mediaQueries []*value.CssMediaQuery
	// mediaQuerySources is the set of queries merged to create mediaQueries.
	// It is non-nil exactly when mediaQueries is non-nil, and empty when
	// mediaQueries is not the result of a merge.
	mediaQuerySources *linkedhashmap.LinkedHashSet[*value.CssMediaQuery]
	// parent is the current parent node in the output CSS tree.
	parent value.ModifiableCssParentNode
	// declarationName is the name of the current declaration parent.
	declarationName string
	// member is the human-readable name of the current stack frame,
	// starting at "root stylesheet".
	member string
	// importSpan is the span of the import currently being resolved, used
	// for importer warnings.
	importSpan sasscommon.FileSpan
	// inFunction reports whether a function body is executing.
	inFunction bool
	// inMixin reports whether a mixin body is executing.
	inMixin bool
	// inUnknownAtRule reports whether evaluation is building the output of
	// an unknown at-rule.
	inUnknownAtRule bool
	// inKeyframes reports whether evaluation is building a @keyframes rule.
	inKeyframes bool
	// inSupportsDeclaration reports whether a SupportsDeclaration is being
	// evaluated; while set, calculations are not simplified.
	inSupportsDeclaration bool
	// inDependency reports whether evaluation is inside a dependency: a
	// stylesheet imported by something other than the entrypoint. Nothing
	// counts as a dependency under Node importers.
	inDependency bool
	// warningsEmitted holds message/location pairs already warned about, so
	// each location warns at most once.
	warningsEmitted map[warnKey]struct{}

	// root is the root stylesheet node of the current module evaluation.
	root *value.ModifiableCssStylesheet
	// endOfImports is the first index in root children after the initial
	// block of CSS imports.
	endOfImports int
	// outOfOrderImports holds plain-CSS imports that appeared after the
	// initial import block; visitStylesheet splices them back into place
	// once the stylesheet is fully evaluated.
	outOfOrderImports []*value.ModifiableCssImport
	// extensionStore tracks extensions and style rules for the current module.
	extensionStore *extend.DefaultExtensionStore
	// activeModules maps canonical URLs of modules (or imported files)
	// currently being evaluated to the nodes marking their original loads.
	// A nil value marks an active module with no load span, such as the
	// entrypoint. This guards against infinite load loops.
	activeModules map[string]sasscommon.AstNode

	// importer resolves relative imports in the current stylesheet; nil
	// means relative imports are unsupported there.
	importer Importer
	// nodeImporter is the Node Sass-compatible importer used for new Sass
	// loads in Node.js compatibility mode.
	nodeImporter *NodePackageImporter
	// sourceURL is the canonical URL of the entrypoint stylesheet.
	sourceURL *url.URL
	// loadedUrls accumulates the canonical URLs of all stylesheets loaded
	// during compilation for error reporting.
	loadedUrls *orderedset.LinkedSet[string]

	// stylesheet is the stylesheet currently being evaluated.
	stylesheet *value.Stylesheet
	// preModuleComments maps modules loaded by the current module to loud
	// comments that should appear before the loaded module.
	preModuleComments map[sassmodule.Module][]value.CssComment
	// configuration is the configuration for the current module; empty
	// means the module is not configured.
	configuration *configuration.Configuration

	// compileContext is the unique per-compilation identity token. Function
	// and mixin values capture it at creation so cross-compilation calls
	// can be rejected, matching Dart's fresh Object per visitor.
	compileContext any
}

// NewEvaluateVisitor builds a visitor with a fresh environment, warning
// dedup set, and per-compilation identity token. The evaluation context
// starts with the given logger and routes warnings through the visitor so
// they pick up dedup, quiet-deps filtering, and stack traces; the import
// cache, when present, forwards its deprecation warnings through the
// current import span (falling back when the span has no source file).
func NewEvaluateVisitor(importCache *ImportCache, logger sasslogger.Logger) *EvaluateVisitor {
	v := &EvaluateVisitor{
		ec: &evalcontext.EvaluationContext{
			Logger: logger,
		},
		env:                  sassenv.NewEnvironment(),
		importCache:          importCache,
		logger:               logger,
		member:               "root stylesheet",
		configuration:        configuration.EmptyConfiguration(),
		modules:              make(map[string]sassmodule.Module),
		moduleConfigurations: make(map[string]*configuration.Configuration),
		moduleNodes:          make(map[string]sasscommon.AstNode),
		builtInModules:       make(map[string]sassmodule.Module),
		builtInFunctions:     make(map[string]sasscallable.Callable),
		stack:                nil,
		activeModules:        make(map[string]sasscommon.AstNode),
		loadedUrls:           nil,
		warningsEmitted:      make(map[warnKey]struct{}),
		compileContext:       &struct{}{},
	}
	v.ec.SetWarnFn(v.warn)
	if v.importCache != nil {
		v.importCache.SetWarnDeprecationFn(func(msg string, dep *deprecation.Deprecation) error {
			// Use the current import span, callable span, or default span.
			span := v.importSpan
			if span != nil {
				file, err := span.File()
				if err != nil {
					return err
				}
				if file == nil {
					span = nil
				}
			}
			return v.warn(msg, span, dep)
		})
	}
	return v
}

// SetNodeImporter installs the Node Sass-compatible importer used to load
// new Sass files in Node.js compatibility mode. It is consulted as a
// fallback when the import cache cannot resolve a URL.
func (v *EvaluateVisitor) SetNodeImporter(imp *NodePackageImporter) {
	v.nodeImporter = imp
}

// asNodeSass reports whether the evaluator runs in Node Sass-compatibility
// mode, which is exactly when a Node importer is installed.
func (v *EvaluateVisitor) asNodeSass() bool {
	return v.nodeImporter != nil
}

// SetQuietDeps controls whether warnings from dependency stylesheets are
// suppressed.
func (v *EvaluateVisitor) SetQuietDeps(quietDeps bool) {
	v.quietDeps = quietDeps
}

// SetSourceMap controls whether source locations of variable declarations
// are tracked for source-map output.
func (v *EvaluateVisitor) SetSourceMap(sourceMap bool) {
	v.sourceMap = sourceMap
}

// Evaluate converts a stylesheet to a plain CSS tree.
//
// If importCache (or, in Node Sass-compatibility mode, nodeImporter) is
// passed, it resolves imports in the Sass files. If importer is passed, it
// resolves relative imports against the stylesheet's source URL. The given
// functions are available as globals while evaluating; warnings go through
// logger, defaulting to standard error. If sourceMap is true, variable
// declaration locations are tracked.
//
// This ports Dart's top-level evaluate(): it builds a fresh visitor,
// registers user functions first so built-ins take priority, applies the
// quiet-deps and source-map flags, installs the Node importer when present,
// and runs the stylesheet.
func Evaluate(
	stylesheet *value.Stylesheet,
	importCache *ImportCache,
	nodeImporter *NodePackageImporter,
	importer Importer,
	functions []sasscallable.Callable,
	logger sasslogger.Logger,
	quietDeps bool,
	sourceMap bool,
) (*EvaluateResult, error) {
	v := NewEvaluateVisitor(importCache, logger)
	if len(functions) > 0 {
		v.RegisterUserFunctions(functions)
	}
	RegisterBuiltInFunctions(v.Env(), v)
	v.SetQuietDeps(quietDeps)
	v.SetSourceMap(sourceMap)
	if nodeImporter != nil {
		v.SetNodeImporter(nodeImporter)
	}
	return v.Run(importer, stylesheet)
}

// RegisterUserFunctions installs user-defined callables for global lookup,
// normalizing underscores to hyphens. They are added before built-ins so
// that a built-in of the same name takes priority, matching the constructor
// behavior of Dart's _EvaluateVisitor.
func (v *EvaluateVisitor) RegisterUserFunctions(fns []sasscallable.Callable) {
	for _, fn := range fns {
		normalized := strings.ReplaceAll(fn.Name(), "_", "-")
		v.builtInFunctions[normalized] = fn
	}
}

// RegisterBuiltinModule indexes a built-in module by its URL so @use of that
// URL resolves without executing a stylesheet. Built-ins count as already
// loaded: they need no evaluation and produce no CSS.
func (v *EvaluateVisitor) RegisterBuiltinModule(mod *sassmodule.BuiltInModule) error {
	url, err := mod.URL()
	if err != nil {
		return err
	}
	v.builtInModules[url] = mod
	return nil
}

// Env exposes the evaluator's lexical environment so built-in functions can
// be registered before evaluation starts.
func (v *EvaluateVisitor) Env() *sassenv.Environment {
	return v.env
}

// withEvaluationContext runs callback with span as the fallback warn span,
// restoring the previous default afterwards. This is the explicit
// replacement for Dart's zone-scoped withEvaluationContext: run, expression,
// and statement entry points each install the node's span so warnings
// without a better location still point somewhere useful.
func (v *EvaluateVisitor) withEvaluationContext(span sasscommon.FileSpan, callback func() error) error {
	old := v.ec.DefaultWarnSpan()
	v.ec.SetDefaultWarnSpan(span)
	defer v.ec.SetDefaultWarnSpan(old)
	return callback()
}

// RunExpression evaluates one expression as if it appeared in a stylesheet,
// inside a temporary evaluation context rooted at the expression's span.
// A fake stylesheet/importer frame is installed so relative loads and error
// spans behave like a real run, then everything is restored.
func (v *EvaluateVisitor) RunExpression(importer Importer, expression value.Expression) (value.Value, error) {
	span, err := expression.Span()
	if err != nil {
		return nil, err
	}
	var result value.Value
	err = v.withEvaluationContext(span, func() error {
		var innerErr error
		result, innerErr = addExceptionTrace(v, func() (value.Value, error) {
			return withFakeStylesheet(v, importer, expression, func() (value.Value, error) {
				return v.eval(expression)
			})
		})
		return innerErr
	})
	return result, err
}

// RunStatement evaluates one statement as if it appeared in a stylesheet,
// inside a temporary evaluation context rooted at the statement's span.
// Like RunExpression it installs a fake stylesheet/importer frame with
// exception-trace wrapping, then restores the prior state.
func (v *EvaluateVisitor) RunStatement(importer Importer, statement value.Statement) error {
	span, err := statement.Span()
	if err != nil {
		return err
	}
	return v.withEvaluationContext(span, func() error {
		_, err := addExceptionTrace(v, func() (value.Value, error) {
			return withFakeStylesheet(v, importer, statement, func() (value.Value, error) {
				return v.visitStatement(statement)
			})
		})
		return err
	})
}

// withFakeStylesheet runs callback with importer installed and a synthetic
// stylesheet carrying nodeWithSpan's span, restoring both afterwards. This
// lets single-expression or single-statement evaluations reuse code that
// assumes a current stylesheet without leaking that frame to the caller.
func withFakeStylesheet[T any](v *EvaluateVisitor, importer Importer, nodeWithSpan sasscommon.AstNode, callback func() (T, error)) (T, error) {
	oldImporter := v.importer
	oldStylesheet := v.stylesheet

	v.importer = importer
	span, err := nodeWithSpan.Span()
	if err != nil {
		var zero T
		return zero, err
	}
	v.stylesheet = value.NewStylesheet(nil, span)

	result, callbackErr := callback()

	v.importer = oldImporter
	v.stylesheet = oldStylesheet

	return result, callbackErr
}

// warn emits message at span, attaching the current stack trace. Warnings
// from dependencies are dropped under quietDeps, and each message/location
// pair is emitted at most once. Plain warnings go to the logger; deprecation
// warnings go through the logger's deprecation path, and a resulting
// Sass error is rethrown with the same trace so the warning location
// survives.
func (v *EvaluateVisitor) warn(message string, span sasscommon.FileSpan, deprecation *deprecation.Deprecation) error {
	if v.quietDeps && v.inDependency {
		return nil
	}
	key := warnKey{message: message, span: span}
	if _, ok := v.warningsEmitted[key]; ok {
		return nil
	}
	v.warningsEmitted[key] = struct{}{}
	trace := v.stackTrace(span)
	if deprecation == nil {
		v.logger.Warn(message, &span, trace)
		return nil
	}
	err := v.logger.WarnDeprecation(message, &span, deprecation, trace)
	if err != nil {
		// In Dart, _warn passes the trace to _handleDeprecation, which includes
		// it in the SassRuntimeException.
		if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
			return sasscommon.ThrowWithTrace(
				&sasscommon.SassRuntimeException{
					Message:    se.Message,
					Span:       se.Span,
					Trace:      trace,
					LoadedUrls: se.LoadedUrls,
				},
				se,
			)
		}
		if sre, ok := errors.AsType[*sasscommon.SassRuntimeException](err); ok && sre.Trace == nil {
			sre.Trace = trace
		}
	}
	return err
}

// Run evaluates stylesheet with importer and returns the combined CSS plus
// loaded URLs. It installs a fresh extension store, root stylesheet, and
// parent chain for the run, records the entrypoint URL (skipping the plain
// "stdin" marker in Node Sass-compatibility mode) and marks it active so
// recursive loads are detected. The stylesheet is visited under exception
// tracing; Sass errors are annotated with the loaded URLs before they
// propagate. Out-of-order plain-CSS imports are spliced after the leading
// import block, the root module is built and combined with its upstream
// modules, and the previous extension store is restored on return.
func (v *EvaluateVisitor) Run(importer Importer, stylesheet *value.Stylesheet) (*EvaluateResult, error) {
	v.stylesheet = stylesheet
	sspan, err := stylesheet.Span()
	if err != nil {
		return nil, err
	}

	extensionStore := extend.NewDefaultExtensionStore()
	oldExtensionStore := v.extensionStore
	defer func() {
		v.extensionStore = oldExtensionStore
	}()
	v.extensionStore = extensionStore

	var result *EvaluateResult
	err = v.withEvaluationContext(sspan, func() error {
		v.root = value.NewModifiableCssStylesheet(sspan)
		v.parent = v.root

		// Reset per-run state
		v.loadedUrls = orderedset.New[string]()

		// The entrypoint URL is recorded up front so error reports can list
		// every loaded stylesheet, except for the synthetic "stdin" marker
		// under Node Sass-compatibility mode.
		srcURL, err := sspan.SourceURL()
		if err != nil {
			return err
		}
		if srcURL != nil {
			v.sourceURL = srcURL
			// Dart: if (!(_asNodeSass && url.toString() == 'stdin')) _loadedUrls.add(url);
			// Go keeps the same guard: plain stdin in Node-compat mode is not a
			// real loaded file.
			if !(v.asNodeSass() && srcURL.String() == "stdin") {
				v.loadedUrls.Add(srcURL.String())
			}
			v.activeModules[srcURL.String()] = nil
		}

		// The active importer is installed before visiting, mirroring the
		// assignment inside Dart's _execute.
		v.importer = importer

		_, err = addExceptionTrace(v, func() (value.Value, error) {
			_, err := v.VisitStylesheet(stylesheet)
			return nil, err
		})
		if err != nil {
			// Failing evaluations still report what was loaded, so the
			// Sass error carries the accumulated URL list.
			urlList := v.loadedUrlsList()
			if sre, ok := errors.AsType[*sasscommon.SassRuntimeException](err); ok {
				err = sasscommon.ThrowWithTrace(sre.WithLoadedUrls(urlList), sre)
			} else if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
				err = sasscommon.ThrowWithTrace(se.WithLoadedUrls(urlList), se)
			}
			return err
		}

		var css *value.CssStylesheet
		if len(v.outOfOrderImports) > 0 {
			span, err := v.stylesheet.Span()
			if err != nil {
				return err
			}
			css = value.NewCssStylesheet(buildOutOfOrderChildren(v), span)
		} else {
			css = v.root.ToCssStylesheet()
		}
		rootModule := v.env.ToModule(css, v.preModuleComments, v.extensionStore)
		// The root module is cached by URL like any loaded module, together
		// with its configuration, so later loads see it as already evaluated.
		if srcURL != nil {
			urlStr := srcURL.String()
			v.modules[urlStr] = rootModule
			v.moduleConfigurations[urlStr] = v.configuration
			v.moduleNodes[urlStr] = nil
		}
		combined, err := v.combineCss(rootModule, false)
		if err != nil {
			return err
		}
		result = &EvaluateResult{
			Stylesheet: combined,
			LoadedUrls: v.loadedUrls,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// combineCss returns a new stylesheet containing root's CSS as well as the CSS
// of all modules transitively used by root.
//
// Each module's extensions are applied to its upstream modules along the
// way. When clone is true the modules are copied first so extending does
// not mutate root or its dependencies (used for re-entrant meta loads).
//
// The fast path handles a root with no upstream CSS: any unsatisfied
// mandatory extension is reported immediately and root's own CSS is
// returned. Otherwise imports and comments stay in the leading import
// block while remaining statements accumulate in document order, modules
// are visited depth-first into reverse-topological order, extensions are
// applied downstream-to-upstream, and the concatenated children form the
// final stylesheet.
func (v *EvaluateVisitor) combineCss(root sassmodule.Module, clone bool) (*value.CssStylesheet, error) {
	if !anyTransitivelyContainsCss(root.Upstream()) {
		selectors := root.ExtensionStore().SimpleSelectors()
		selectorsSet := linkedhashmap.NewLinkedHashSet[value.SimpleSelector](value.SimpleSelectorEquals)
		for _, s := range selectors {
			selectorsSet.Add(s)
		}
		extensions := root.ExtensionStore().ExtensionsWhereTarget(func(target value.SimpleSelector) bool {
			return !selectorsSet.Contains(target)
		})
		if len(extensions) > 0 {
			return nil, v.throwForUnsatisfiedExtension(extensions[0])
		}
		rootCSS, err := root.CSS()
		if err != nil {
			return nil, err
		}
		return rootCSS, nil
	}

	var imports []value.CssNode
	var css []value.CssNode
	var sorted []sassmodule.Module
	seen := make(map[sassmodule.Module]struct{})

	var visitModule func(sassmodule.Module) error
	visitModule = func(module sassmodule.Module) error {
		if _, ok := seen[module]; ok {
			return nil
		}
		seen[module] = struct{}{}
		if clone {
			var err error
			module, err = module.CloneCss()
			if err != nil {
				return err
			}
		}

		for _, upstream := range module.Upstream() {
			if upstream.TransitivelyContainsCss() {
				if comments, ok := module.PreModuleComments()[upstream]; ok {
					if len(css) == 0 {
						imports = append(imports, cssCommentSliceToCssNode(comments)...)
					} else {
						css = append(css, cssCommentSliceToCssNode(comments)...)
					}
				}
				if err := visitModule(upstream); err != nil {
					return err
				}
			}
		}

		sorted = append([]sassmodule.Module{module}, sorted...)
		modCSS, err := module.CSS()
		if err != nil {
			return err
		}
		statements := modCSS.Children()
		index := indexAfterImports(statements)
		imports = append(imports, statements[:index]...)
		css = append(css, statements[index:]...)
		return nil
	}

	if err := visitModule(root); err != nil {
		return nil, err
	}

	if root.TransitivelyContainsExtensions() {
		if err := v.extendModules(sorted); err != nil {
			return nil, err
		}
	}

	rootCSS, err := root.CSS()
	if err != nil {
		return nil, err
	}
	cssSpan, err := rootCSS.Span()
	if err != nil {
		return nil, err
	}
	return value.NewCssStylesheet(append(imports, css...), cssSpan), nil
}

// anyTransitivelyContainsCss reports whether any module in upstream
// transitively contains CSS, driving the fast path in combineCss.
func anyTransitivelyContainsCss(upstream []sassmodule.Module) bool {
	for _, module := range upstream {
		if module.TransitivelyContainsCss() {
			return true
		}
	}
	return false
}

// indexAfterImports returns the index of the first node in statements that comes
// after all static imports. Plain-CSS imports count; comments are skipped so
// loud comments between imports stay in the leading block; anything else ends
// the block.
func indexAfterImports(statements []value.CssNode) int {
	lastImport := -1
loop:
	for i, stmt := range statements {
		switch stmt.(type) {
		case value.CssImport:
			lastImport = i
		case value.CssComment:
			continue loop
		default:
			break loop
		}
	}
	return lastImport + 1
}

// extendModules destructively updates the selectors in each module with the
// extensions defined in downstream modules.
//
// Modules arrive in reverse-topological order, so by the time a module is
// processed every downstream extension store has already been collected for
// it. For each module the store snapshots its current simple selectors,
// records as-yet-unsatisfied mandatory extensions targeting anything else,
// folds in the downstream stores, forwards its own store to each upstream
// URL, and then drops extensions that the merged store now satisfies. Any
// extension still unsatisfied after all modules are processed is reported.
func (v *EvaluateVisitor) extendModules(sortedModules []sassmodule.Module) error {
	downstreamExtensionStores := make(map[string][]extend.ExtensionStore)
	unsatisfiedExtensions := make(map[extend.Extension]struct{})

	for _, module := range sortedModules {
		originalSelectors := linkedhashmap.NewLinkedHashSet[value.SimpleSelector](value.SimpleSelectorEquals)
		for _, s := range module.ExtensionStore().SimpleSelectors() {
			originalSelectors.Add(s)
		}

		extensions := module.ExtensionStore().ExtensionsWhereTarget(func(target value.SimpleSelector) bool {
			return !originalSelectors.Contains(target)
		})
		for _, ext := range extensions {
			unsatisfiedExtensions[ext] = struct{}{}
		}

		url, err := module.URL()
		if err != nil {
			return err
		}
		if stores, ok := downstreamExtensionStores[url]; ok {
			if err := module.ExtensionStore().AddExtensions(stores); err != nil {
				return err
			}
		}
		if module.ExtensionStore().IsEmpty() {
			continue
		}

		for _, upstream := range module.Upstream() {
			url, err := upstream.URL()
			if err != nil {
				return err
			}
			if url != "" {
				downstreamExtensionStores[url] = append(
					downstreamExtensionStores[url],
					module.ExtensionStore(),
				)
			}
		}

		nowSatisfied := module.ExtensionStore().ExtensionsWhereTarget(func(target value.SimpleSelector) bool {
			return originalSelectors.Contains(target)
		})
		for _, ext := range nowSatisfied {
			delete(unsatisfiedExtensions, ext)
		}
	}

	if len(unsatisfiedExtensions) > 0 {
		for ext := range unsatisfiedExtensions {
			return v.throwForUnsatisfiedExtension(ext)
		}
	}
	return nil
}

// throwForUnsatisfiedExtension reports a mandatory extension whose target
// selector was never found, suggesting !optional as the remedy.
func (v *EvaluateVisitor) throwForUnsatisfiedExtension(extension extend.Extension) error {
	target, err := extension.Target().String()
	if err != nil {
		return err
	}
	extSpan, err := extension.Span()
	if err != nil {
		return err
	}
	return &sasscommon.SassException{
		Message: "The target selector was not found.\n" +
			"Use \"@extend " + target + " !optional\" to avoid this error.",
		Span: extSpan,
	}
}

// cssCommentSliceToCssNode adapts a comment slice to a CSS node slice for
// splicing pre-module comments into the combined import/CSS streams.
func cssCommentSliceToCssNode(comments []value.CssComment) []value.CssNode {
	nodes := make([]value.CssNode, len(comments))
	for i, c := range comments {
		nodes[i] = c
	}
	return nodes
}

// eval evaluates expr by dispatching to its value-visitor accept method,
// the Go counterpart of Dart's expression.accept(this).
func (v *EvaluateVisitor) eval(expr value.Expression) (value.Value, error) {
	return expr.AcceptValue(v)
}

// Evaluator evaluates multiple independent statements and expressions in the
// context of a single module. It wraps one EvaluateVisitor plus the importer
// used to resolve @use rules, so hosts can drive a module piecemeal without
// re-running the whole pipeline.
type Evaluator struct {
	visitor  *EvaluateVisitor
	importer Importer
}

// NewEvaluator builds an Evaluator over a fresh visitor. Arguments mirror
// Evaluate: user functions are registered first so built-ins take priority,
// and built-ins are installed before any evaluation runs.
func NewEvaluator(importCache *ImportCache, importer Importer, functions []sasscallable.Callable, logger sasslogger.Logger) *Evaluator {
	v := NewEvaluateVisitor(importCache, logger)
	if len(functions) > 0 {
		v.RegisterUserFunctions(functions)
	}
	RegisterBuiltInFunctions(v.Env(), v)
	return &Evaluator{visitor: v, importer: importer}
}

// Use processes a @use rule through the evaluator's importer.
func (e *Evaluator) Use(use *value.UseRule) error {
	return e.visitor.RunStatement(e.importer, use)
}

// Evaluate evaluates a Sass expression in the evaluator's module context.
func (e *Evaluator) Evaluate(expression value.Expression) (value.Value, error) {
	return e.visitor.RunExpression(e.importer, expression)
}

// SetVariable processes a variable declaration in the evaluator's module context.
func (e *Evaluator) SetVariable(declaration *value.VariableDeclaration) error {
	return e.visitor.RunStatement(e.importer, declaration)
}

var _ value.IfConditionExpressionVisitor[any] = (*EvaluateVisitor)(nil)
