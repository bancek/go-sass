// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (statement visitors)

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/configuration"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/sassurl"
	"github.com/bancek/go-sass/unvendor"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// --- Statement dispatch helper ---

// visitStatement evaluates a single Sass statement and returns the @return
// value when the statement (or a nested block) produces one, else nil.
//
// Matches Dart: Statement.accept(this) dispatch into the visit* family
// (evaluate.dart visitStylesheet and following statement visitors).
func (v *EvaluateVisitor) visitStatement(stmt value.Statement) (value.Value, error) {
	return stmt.AcceptValue(v)
}

// evaluateBlock evaluates a block of statements in order inside an
// exception-trace wrapper, returning the first error encountered.
//
// Unlike handleReturn it does not short-circuit on @return values; callers
// that need @return propagation iterate via handleReturn themselves.
func (v *EvaluateVisitor) evaluateBlock(children []value.Statement) (value.Value, error) {
	return addExceptionTrace(v, func() (value.Value, error) {
		for _, child := range children {
			if _, err := v.visitStatement(child); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
}

// handleReturn runs callback for each element of list until one returns a
// non-nil Value, which is then returned immediately so @return unwinds
// through nested blocks. Errors abort the iteration. Returns (nil, nil) when
// every callback returns nil.
//
// Matches Dart: _EvaluateVisitor._handleReturn (evaluate.dart _handleReturn
// section); the generic parameter lets one helper serve both Value lists and
// Statement blocks.
func handleReturn[T any](v *EvaluateVisitor, list []T, callback func(T) (value.Value, error)) (value.Value, error) {
	for _, val := range list {
		result, err := callback(val)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// interpolateSelector evaluates interpolation in and returns the resulting
// text without color-name warnings.
//
// Matches Dart: _EvaluateVisitor._performInterpolation (evaluate.dart
// interpolation section) with warnForColor false.
func (v *EvaluateVisitor) interpolateSelector(in *value.Interpolation) (string, error) {
	return v.interpolateSelectorWarn(in, false)
}

// interpolateSelectorWarn evaluates interpolation in and returns the
// resulting text, optionally warning when a named color flows into the
// interpolation (which usually renders as invalid CSS).
//
// Matches Dart: _performInterpolation with warnForColor
// (evaluate.dart interpolation section); nil interpolation yields "".
func (v *EvaluateVisitor) interpolateSelectorWarn(in *value.Interpolation, warnForColor bool) (string, error) {
	if in == nil {
		return "", nil
	}
	if in.IsPlain() {
		return *in.AsPlain(), nil
	}
	return v.performInterpolation(in, &warnForColor)
}

// interpolationToValue evaluates in and wraps the result in a CssValue
// carrying the interpolation span. When trim is true surrounding ASCII
// whitespace (excluding escapes) is removed; when warnForColor is true named
// colors interpolated as strings warn.
//
// Matches Dart: _interpolationToValue (evaluate.dart interpolation section);
// trim maps to trimAscii(excludeEscape) and the Go error return replaces
// Dart's span-wrapped throw.
func (v *EvaluateVisitor) interpolationToValue(in *value.Interpolation, trim bool, warnForColor bool) (sasscommon.CssValue[string], error) {
	result, err := v.performInterpolation(in, &warnForColor)
	if err != nil {
		return sasscommon.CssValue[string]{}, err
	}
	if trim {
		result = util.TrimAscii(result, true)
	}
	span, err := in.Span()
	if err != nil {
		return sasscommon.CssValue[string]{}, err
	}
	return sasscommon.NewCssValue(result, span), nil
}

// evaluateModule evaluates a parsed stylesheet as a module in a fresh
// environment and returns the resulting Module. Already-loaded URLs
// short-circuit, erroring when an explicit with-configuration conflicts with
// the original load; otherwise the visitor swaps in a fresh importer,
// stylesheet, CSS root, extension store, and scope flags, visits the
// stylesheet, splices out-of-order imports into the CSS, builds the module,
// caches it with its configuration and load node, and restores everything.
//
// Matches Dart: _EvaluateVisitor._execute (evaluate.dart _execute section);
// the caller's ambient configuration applies when config is nil, and
// namesInErrors controls whether module URLs appear in with-errors.
func (v *EvaluateVisitor) evaluateModule(sheet *value.Stylesheet, config *configuration.Configuration, importer Importer, nodeWithSpan sasscommon.AstNode, namesInErrors bool) (sassmodule.Module, error) {
	sheetSpan, err := sheet.Span()
	if err != nil {
		return nil, err
	}
	srcURL, err := sheetSpan.SourceURL()
	if err != nil {
		return nil, err
	}

	var urlStr string
	if srcURL != nil {
		urlStr = srcURL.String()
	}

	currentConfiguration := config
	if currentConfiguration == nil {
		currentConfiguration = v.configuration
	}

	if urlStr != "" {
		if alreadyLoaded, ok := v.modules[urlStr]; ok {
			prevConfig := v.moduleConfigurations[urlStr]
			if prevConfig != nil && !prevConfig.SameOriginal(currentConfiguration) &&
				currentConfiguration.IsExplicit() &&
				alreadyLoaded.CouldHaveBeenConfigured(configuredVariableKeys(currentConfiguration)) {
				var message string
				if namesInErrors {
					// Name the module with a pretty URI when the span alone
					// would not identify it.
					message = fmt.Sprintf("%s was already loaded, so it can't be configured using \"with\".", sasscommon.PrettyUri(srcURL))
				} else {
					message = "This module was already loaded, so it can't be configured using \"with\"."
				}

				secondarySpans := make(map[sasscommon.FileSpan]string)
				if prevNode, ok := v.moduleNodes[urlStr]; ok {
					prevSpan, spanErr := prevNode.Span()
					if spanErr == nil {
						secondarySpans[prevSpan] = "original load"
					}
				}
				if config == nil {
					if currentConfiguration.NodeWithSpan != nil {
						confSpan, spanErr := currentConfiguration.NodeWithSpan.Span()
						if spanErr == nil {
							secondarySpans[confSpan] = "configuration"
						}
					}
				}

				if len(secondarySpans) > 0 && nodeWithSpan != nil {
					nwsSpan, err := nodeWithSpan.Span()
					if err != nil {
						return nil, err
					}
					return nil, &sasscommon.MultiSpanSassRuntimeException{
						Message:        message,
						Span:           nwsSpan,
						PrimaryLabel:   "new load",
						SecondarySpans: secondarySpans,
						Trace:          v.stackTrace(nwsSpan),
					}
				}
				return nil, v.exception(message, nil)
			}
			return alreadyLoaded, nil
		}
	}

	oldEnv := v.env
	oldConfig := v.configuration

	v.env = sassenv.NewEnvironment()
	defer func() { v.env = oldEnv }()
	defer func() { v.configuration = oldConfig }()

	if config != nil {
		v.configuration = config
	}

	oldImporter := v.importer
	oldSourceURL := v.sourceURL
	oldStylesheet := v.stylesheet
	oldRoot := v.root
	oldParent := v.parent
	oldEndOfImports := v.endOfImports
	oldOutOfOrderImports := v.outOfOrderImports
	oldExtensionStore := v.extensionStore
	oldPreModuleComments := v.preModuleComments
	oldStyleRule := v.styleRuleIgnoringAtRoot
	oldMediaQueries := v.mediaQueries
	oldDeclarationName := v.declarationName
	oldInUnknownAtRule := v.inUnknownAtRule
	oldAtRootExcludingStyleRule := v.atRootExcludingStyleRule
	oldInKeyframes := v.inKeyframes
	defer func() {
		v.importer = oldImporter
		v.sourceURL = oldSourceURL
		v.stylesheet = oldStylesheet
		v.root = oldRoot
		v.parent = oldParent
		v.endOfImports = oldEndOfImports
		v.outOfOrderImports = oldOutOfOrderImports
		v.extensionStore = oldExtensionStore
		v.preModuleComments = oldPreModuleComments
		v.styleRuleIgnoringAtRoot = oldStyleRule
		v.mediaQueries = oldMediaQueries
		v.declarationName = oldDeclarationName
		v.inUnknownAtRule = oldInUnknownAtRule
		v.atRootExcludingStyleRule = oldAtRootExcludingStyleRule
		v.inKeyframes = oldInKeyframes
	}()
	v.importer = importer
	v.stylesheet = sheet
	v.root = value.NewModifiableCssStylesheet(sheetSpan)
	v.parent = v.root
	v.endOfImports = 0
	v.outOfOrderImports = nil
	v.extensionStore = extend.NewDefaultExtensionStore()
	v.preModuleComments = nil
	v.styleRuleIgnoringAtRoot = nil
	v.mediaQueries = nil
	v.declarationName = ""
	v.inUnknownAtRule = false
	v.atRootExcludingStyleRule = false
	v.inKeyframes = false
	if srcURL != nil {
		v.sourceURL = srcURL
	}

	if _, err := v.VisitStylesheet(sheet); err != nil {
		return nil, err
	}

	var css *value.CssStylesheet
	if len(v.outOfOrderImports) > 0 {
		children := v.root.Children()
		newChildren := make([]value.CssNode, 0, len(children)+len(v.outOfOrderImports))
		for i := 0; i < v.endOfImports && i < len(children); i++ {
			newChildren = append(newChildren, children[i])
		}
		for _, imp := range v.outOfOrderImports {
			newChildren = append(newChildren, imp)
		}
		for i := v.endOfImports; i < len(children); i++ {
			newChildren = append(newChildren, children[i])
		}
		sheetSpan2, err := sheet.Span()
		if err != nil {
			return nil, err
		}
		css = value.NewCssStylesheet(newChildren, sheetSpan2)
	} else {
		css = v.root.ToCssStylesheet()
	}
	mod := v.env.ToModule(css, v.preModuleComments, v.extensionStore)

	if urlStr != "" {
		v.modules[urlStr] = mod
		v.moduleConfigurations[urlStr] = currentConfiguration
		if nodeWithSpan != nil {
			v.moduleNodes[urlStr] = nodeWithSpan
		}
	}

	return mod, nil
}

// configuredVariableKeys returns the set of variable names defined in config.
//
// Used by evaluateModule's already-loaded check to decide whether a module
// could have been affected by a conflicting configuration.
//
// Matches Dart: MapKeySet(configuration.values) at the _execute
// already-loaded guard (evaluate.dart _execute section).
func configuredVariableKeys(config *configuration.Configuration) map[string]struct{} {
	vals := config.Values()
	keys := make(map[string]struct{}, len(vals))
	for k := range vals {
		keys[k] = struct{}{}
	}
	return keys
}

// removeUsedConfiguration drops upstream values already consumed downstream,
// unless pinned in except (the unguarded with names this @forward manages).
//
// Matches Dart: _EvaluateVisitor._removeUsedConfiguration (evaluate.dart
// visitForwardRule section).
func removeUsedConfiguration(upstream *configuration.Configuration, downstream *configuration.Configuration, except map[string]struct{}) {
	for name := range upstream.Values() {
		if _, ok := except[name]; ok {
			continue
		}
		if _, found := downstream.Get(name); !found {
			upstream.Remove(name)
		}
	}
}

// emptyStylesheet returns an empty CSS stylesheet.
func emptyStylesheet() *value.CssStylesheet {
	return value.NewCssStylesheetEmpty(nil)
}

// parseMediaQueryText converts a media query string to CssMediaQuery objects.
func parseMediaQueryText(text string) []value.CssMediaQuery {
	return parseMediaQueryTextWithMap(text, nil)
}

// parseMediaQueryTextWithMap converts a media query string to CssMediaQuery
// objects, using the interpolation map for span mapping.
func parseMediaQueryTextWithMap(text string, im *value.InterpolationMap) []value.CssMediaQuery {
	if text == "" {
		return nil
	}
	p := value.NewParser([]byte(text), nil, im)
	parser := &value.CssMediaQueryParser{Parser: *p}
	queries, err := parser.Parse()
	if err != nil {
		return nil
	}
	result := make([]value.CssMediaQuery, len(queries))
	for i, q := range queries {
		if q != nil {
			result[i] = *q
		}
	}
	return result
}

// withoutSlash strips /-as-division slash structure from val, emitting the
// slash-div deprecation warning at node's span first. Non-slash numbers pass
// through unchanged.
//
// Matches Dart: _EvaluateVisitor._withoutSlash applied to variable, @each,
// @return, and with-clause values (evaluate.dart expression-node section);
// the Go error return replaces Dart's throw.
func (v *EvaluateVisitor) withoutSlash(val value.Value, node sasscommon.AstNode) (value.Value, error) {
	if num, ok := val.(value.SassNumber); ok && num.HasSlash() {
		span, err := node.Span()
		if err != nil {
			return nil, err
		}
		rec, recErr := slashDivisionRecommendation(num)
		if recErr != nil {
			return nil, recErr
		}
		if err := v.warn(
			"Using / for division is deprecated and will be removed in Dart Sass 2.0.0.\n\n"+
				"Recommendation: "+rec+"\n\n"+
				"More info and automated migrator: https://sass-lang.com/d/slash-div",
			span,
			deprecation.SlashDiv,
		); err != nil {
			return nil, err
		}
		return num.WithoutSlash(), nil
	}
	return val, nil
}

// slashDivisionRecommendation builds a math.div(...) suggestion for a
// slash-separated number, recursing through nested slash pairs.
//
// Matches Dart: the recommendation closure inside _slash (evaluate.dart slash
// section); parenthesized operands render via their source text there, via
// String here.
func slashDivisionRecommendation(num value.SassNumber) (string, error) {
	if num.HasSlash() {
		n, d := num.SlashPair()
		nStr, err := slashDivisionRecommendation(n)
		if err != nil {
			return "", err
		}
		dStr, err := slashDivisionRecommendation(d)
		if err != nil {
			return "", err
		}
		return "math.div(" + nStr + ", " + dStr + ")", nil
	}
	return num.String()
}

// --- Statement visitors ---

// VisitStylesheet evaluates a whole stylesheet: parse-time warnings first,
// then each child in order. Global variables declared anywhere in the module
// are pre-declared as guarded nil so they appear in the module definition
// even when their assignments are never reached.
//
// Matches Dart: _EvaluateVisitor.visitStylesheet (evaluate.dart statement
// visitors); returns nil Value on success.
func (v *EvaluateVisitor) VisitStylesheet(stmt *value.Stylesheet) (value.Value, error) {
	for _, warning := range stmt.ParseTimeWarnings() {
		if err := v.warn(warning.Message, warning.Span, warning.Deprecation); err != nil {
			return nil, err
		}
	}

	for _, child := range stmt.GetChildren() {
		if _, err := v.visitStatement(child); err != nil {
			return nil, err
		}
	}

	for name, span := range stmt.GlobalVariables() {
		decl, err := value.NewVariableDeclaration(
			name,
			value.NewNullExpression(span),
			span,
			nil,
			true,  // guarded
			false, // global
			nil,   // comment
		)
		if err != nil {
			return nil, err
		}
		if _, err := v.VisitVariableDeclaration(decl); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// VisitVariableDeclaration assigns a variable, honoring guarded (!default)
// semantics and module-configuration overrides. Guarded declarations at the
// module root first consult the pending configuration (a non-nil override
// wins and is set global); otherwise an existing non-nil value is kept. New
// !global declarations warn (new-global deprecation). The right-hand side
// evaluates bare and slash-stripped; only the SetVariable call is
// span-wrapped.
//
// Matches Dart: _EvaluateVisitor.visitVariableDeclaration (evaluate.dart
// statement visitors).
func (v *EvaluateVisitor) VisitVariableDeclaration(stmt *value.VariableDeclaration) (value.Value, error) {
	if stmt.IsGuarded {
		if stmt.Namespace() == nil && v.env.AtRoot() {
			v.env.MarkVariableConfigurable(stmt.Name())
			if override := v.configuration.Remove(stmt.Name()); override != nil && override.Value != value.Null {
				_, err := addExceptionSpan(v, stmt, func() (value.Value, error) {
					return nil, v.env.SetVariable(stmt.Name(), override.Value, override.AssignmentNode, nil, true)
				}, nil)
				return nil, err
			}
		}

		existing, err := addExceptionSpan(v, stmt, func() (value.Value, error) {
			return v.env.GetVariable(stmt.Name(), stmt.Namespace())
		}, nil)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing != value.Null {
			return nil, nil
		}
	}

	if stmt.IsGlobal() {
		exists, err := v.env.GlobalVariableExists(stmt.Name(), nil)
		if err != nil {
			return nil, err
		}
		if !exists {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			message := "As of Dart Sass 2.0.0, !global assignments won't be able to declare new variables.\n\n"
			if v.env.AtRoot() {
				message += "Since this assignment is at the root of the stylesheet, the !global flag is\nunnecessary and can safely be removed."
			} else {
				originalName, err := stmt.OriginalName()
				if err != nil {
					return nil, err
				}
				message += "Recommendation: add `" + originalName + ": null` at the stylesheet root."
			}
			if err := v.warn(message, span, deprecation.NewGlobal); err != nil {
				return nil, err
			}
		}
	}

	val, err := v.eval(stmt.Expression)
	if err != nil {
		return nil, err
	}
	val, err = v.withoutSlash(val, stmt.Expression)
	if err != nil {
		return nil, err
	}

	exprNode, nserr := v.expressionNode(stmt.Expression)
	if nserr != nil {
		return nil, nserr
	}
	_, err = addExceptionSpan(v, stmt, func() (value.Value, error) {
		return nil, v.env.SetVariable(stmt.Name(), val, exprNode, stmt.Namespace(), stmt.IsGlobal())
	}, nil)
	return nil, err
}

// VisitStyleRule evaluates a style rule: selector resolution plus scoped
// child evaluation. Guards reject nested-declaration and keyframe-block
// parents first; in keyframes the selector parses as keyframe selectors,
// otherwise as a selector list that is nested within the enclosing rule
// unless merging is disabled (top level, from-plain-CSS parent, or plain CSS
// with a parent selector). The resolved selector registers with the extension
// store, children evaluate inside the new rule with style-rule bubbling, then
// bogus combinators warn and the last top-level child is marked group end.
//
// Matches Dart: _EvaluateVisitor.visitStyleRule (evaluate.dart statement
// visitors); scopeWhen follows node.hasDeclarations.
func (v *EvaluateVisitor) VisitStyleRule(stmt *value.StyleRule) (value.Value, error) {
	if v.declarationName != "" {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Style rules may not be used within nested declarations.", &span)
	} else if v.inKeyframes {
		if _, ok := v.parent.(*value.ModifiableCssKeyframeBlock); ok {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Style rules may not be used within keyframe blocks.", &span)
		}
	}

	var selectorText string
	var selectorMap *value.InterpolationMap
	var err error
	if stmt.Selector != nil {
		selectorText, selectorMap, err = v.performInterpolationWithMap(stmt.Selector, &[]bool{true}[0])
		if err != nil {
			return nil, err
		}
	}

	if v.inKeyframes {
		ksp := &value.KeyframeSelectorParser{
			Parser: *value.NewParser([]byte(selectorText), nil, selectorMap),
		}
		parsedSelector, err := ksp.Parse()
		if err != nil {
			return nil, err
		}
		selectorSpan, err := stmt.Selector.Span()
		if err != nil {
			return nil, err
		}
		stmtSpan, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		rule := value.NewModifiableCssKeyframeBlock(
			sasscommon.NewCssValue(parsedSelector, selectorSpan),
			stmtSpan,
		)
		return nil, v.withParent(rule, func() error {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		}, func(n value.CssNode) bool {
			_, ok := n.(value.CssStyleRule)
			return ok
		}, new(stmt.HasDeclarations()))
	}

	var parsedSelector *value.SelectorList
	if selectorText != "" {
		parsedSelector, err = value.NewSelectorParser(selectorText, nil, selectorMap, &value.SelectorParserOptions{
			AllowParent: new(true),
			PlainCss:    new(v.stylesheet.IsPlainCss()),
			Logger:      v.logger,
		}).Parse()
		if err != nil {
			return nil, err
		}
	}

	sr := v.styleRule()
	var merge bool
	if sr == nil {
		merge = true
	} else if sr.FromPlainCss() {
		merge = false
	} else {
		containsParent := false
		if parsedSelector != nil {
			var err error
			containsParent, err = parsedSelector.ContainsParentSelector()
			if err != nil {
				return nil, err
			}
		}
		merge = !v.stylesheet.IsPlainCss() || (parsedSelector != nil && !containsParent)
	}
	if merge && parsedSelector != nil {
		if v.stylesheet.IsPlainCss() {
			for _, complex := range parsedSelector.Components {
				if len(complex.LeadingCombinators) > 0 {
					firstSpan, err := complex.LeadingCombinators[0].Span()
					if err != nil {
						return nil, err
					}
					return nil, v.exception("Top-level leading combinators aren't allowed in plain CSS.", &firstSpan)
				}
			}
		}
		var parent *value.SelectorList
		if v.styleRuleIgnoringAtRoot != nil {
			parent = v.styleRuleIgnoringAtRoot.OriginalSelector()
		}
		parsedSelector, err = parsedSelector.NestWithin(
			parent,
			!v.atRootExcludingStyleRule,
			v.stylesheet.IsPlainCss(),
		)
		if err != nil {
			return nil, err
		}
	}

	var selectorBox *box.Box[*value.SelectorList]
	if parsedSelector != nil {
		var extErr error
		selectorBox, extErr = v.extensionStore.AddSelector(parsedSelector, v.mediaQueries)
		if extErr != nil {
			return nil, extErr
		}
	}

	var fromPlainCss bool
	if v.stylesheet != nil {
		fromPlainCss = v.stylesheet.IsPlainCss()
	}
	stmtSpan2, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssStyleRule(selectorBox, stmtSpan2, parsedSelector, fromPlainCss)

	oldAtRootExcludingStyleRule := v.atRootExcludingStyleRule
	v.atRootExcludingStyleRule = false

	var through func(value.CssNode) bool
	if merge {
		through = func(n value.CssNode) bool {
			_, ok := n.(value.CssStyleRule)
			return ok
		}
	}
	// Scope the child environment only when the rule can declare members.
	scopeWhen := stmt.HasDeclarations()
	err = v.withParent(rule, func() error {
		return v.withStyleRule(rule, func() error {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		})
	}, through, &scopeWhen)

	v.atRootExcludingStyleRule = oldAtRootExcludingStyleRule
	if err != nil {
		return nil, err
	}

	if err := v.warnForBogusCombinators(rule); err != nil {
		return nil, err
	}

	if v.styleRule() == nil && v.parent.ChildrenLen() > 0 {
		if lastChild, ok := v.parent.LastChild().(value.ModifiableCssNode); ok {
			lastChild.SetIsGroupEnd(true)
		}
	}

	return nil, nil
}

// loadModule loads the module at url and runs callback with the loaded
// module under a stack frame, reporting whether this is the first load.
// Built-in modules never execute and cannot be configured; other URLs load
// via loadStylesheet with module-loop detection through activeModules, an
// in-dependency flag swap, evaluation via evaluateModule, and an
// exception-span wrapper without an extra stack frame around the callback.
//
// Matches Dart: _EvaluateVisitor._loadModule (evaluate.dart module-loading
// section); baseURL defaults to the current stylesheet URL and namesInErrors
// controls whether module names appear in loop/configure errors.
func (v *EvaluateVisitor) loadModule(url *url.URL, stackFrame string, nodeWithSpan sasscommon.AstNode, configuration *configuration.Configuration, namesInErrors bool, baseURL *url.URL, callback func(sassmodule.Module, bool) error) error {
	if builtIn, ok := v.builtInModules[url.String()]; ok {
		if configuration != nil && configuration.IsExplicit() {
			var message string
			if namesInErrors {
				message = fmt.Sprintf("Built-in module %s can't be configured.", url.String())
			} else {
				message = "Built-in modules can't be configured."
			}
			var confSpan *sasscommon.FileSpan
			if configuration.NodeWithSpan != nil {
				s, err := configuration.NodeWithSpan.Span()
				if err == nil {
					confSpan = &s
				}
			}
			return v.exception(message, confSpan)
		}
		_, err := addExceptionSpan(v, nodeWithSpan, func() (struct{}, error) {
			return struct{}{}, callback(builtIn, false)
		}, nil)
		return err
	}

	_, err := WithStackFrame(v, stackFrame, nodeWithSpan, func() (struct{}, error) {
		return addExceptionTrace(v, func() (struct{}, error) {
			nwsSpan, err := nodeWithSpan.Span()
			if err != nil {
				return struct{}{}, err
			}
			ls, err := v.loadStylesheet(url, nwsSpan, baseURL, false)
			if err != nil {
				return struct{}{}, err
			}

			lsSpan3, err := ls.stylesheet.Span()
			if err != nil {
				return struct{}{}, err
			}
			canonicalUrlObj, err := lsSpan3.SourceURL()
			if err != nil {
				return struct{}{}, err
			}
			canonicalUrl := canonicalUrlObj.String()
			if canonicalUrl != "" {
				if prev, ok := v.activeModules[canonicalUrl]; ok {
					var message string
					if namesInErrors {
						// Name the looping module with a pretty URI, as above.
						message = fmt.Sprintf("Module loop: %s is already being loaded.", sasscommon.PrettyUri(canonicalUrlObj))
					} else {
						message = "Module loop: this module is already being loaded."
					}
					if prev != nil {
						prevSpan, spanErr := prev.Span()
						if spanErr == nil {
							nSpan, nSpanErr := nodeWithSpan.Span()
							if nSpanErr == nil {
								return struct{}{}, &sasscommon.MultiSpanSassRuntimeException{
									Message:        message,
									Span:           nSpan,
									PrimaryLabel:   "new load",
									SecondarySpans: map[sasscommon.FileSpan]string{prevSpan: "original load"},
									// Build the trace without a span: the new-load node is
									// already the innermost stack frame, so passing nSpan
									// would duplicate it.
									Trace: v.stackTrace(nil),
								}
							}
						}
					}
					nwsSpan, err := nodeWithSpan.Span()
					if err != nil {
						return struct{}{}, err
					}
					return struct{}{}, v.exception(message, &nwsSpan)
				}
				v.activeModules[canonicalUrl] = nodeWithSpan
			}

			firstLoad := canonicalUrl == "" || v.modules[canonicalUrl] == nil
			oldInDependency := v.inDependency
			v.inDependency = ls.isDependency
			module, err := v.evaluateModule(ls.stylesheet, configuration, ls.importer, nodeWithSpan, namesInErrors)
			if canonicalUrl != "" {
				delete(v.activeModules, canonicalUrl)
			}
			v.inDependency = oldInDependency
			if err != nil {
				return struct{}{}, err
			}

			_, err = addExceptionSpan(v, nodeWithSpan, func() (struct{}, error) {
				return struct{}{}, callback(module, firstLoad)
			}, new(bool))
			return struct{}{}, err
		})
	})
	return err
}

// VisitUseRule evaluates @use: builds an explicit configuration from the
// with clause (values slash-stripped) and loads the module under a @use
// stack frame, registering leading comments on first load and adding the
// module under its namespace, then asserting leftover configuration is
// empty.
//
// Matches Dart: _EvaluateVisitor.visitUseRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitUseRule(stmt *value.UseRule) (value.Value, error) {
	var config *configuration.Configuration
	if len(stmt.Configuration) > 0 {
		values := make(map[string]*configuration.ConfiguredValue)
		for _, cv := range stmt.Configuration {
			val, err := v.eval(cv.Expression)
			if err != nil {
				return nil, err
			}
			val, err = v.withoutSlash(val, cv.Expression)
			if err != nil {
				return nil, err
			}
			span, err := cv.Span()
			if err != nil {
				return nil, err
			}
			values[cv.Name()] = configuration.NewConfiguredValueExplicit(val, &span, cv.Expression)
		}
		config = configuration.NewExplicitConfiguration(values, stmt).Configuration
	}

	// Load under a @use frame; on first load stash leading comments, then add
	// the module under its namespace.
	err := v.loadModule(stmt.URL(), "@use", stmt, config, false, nil, func(loadedModule sassmodule.Module, firstLoad bool) error {
		if firstLoad {
			v.registerCommentsForModule(loadedModule)
		}
		if stmt.Namespace == nil {
			return v.env.AddModule(loadedModule, stmt, nil)
		}
		return v.env.AddModule(loadedModule, stmt, stmt.Namespace)
	})
	if err != nil {
		return nil, err
	}
	if err := v.assertConfigurationIsEmpty(config, false); err != nil {
		return nil, err
	}
	return nil, nil
}

// assertConfigurationIsEmpty reports an error when config is a non-empty
// explicit configuration, meaning a with-clause variable was never consumed
// because the @used module declares no matching !default. Implicit
// configurations always pass since subsets are allowed; the first leftover
// reports at its configuration span, naming the variable only when
// nameInError is set (load-css call sites where the span alone is unclear).
//
// Matches Dart: _EvaluateVisitor._assertConfigurationIsEmpty (evaluate.dart
// module-configuration section).
func (v *EvaluateVisitor) assertConfigurationIsEmpty(config *configuration.Configuration, nameInError bool) error {
	if config == nil {
		return nil
	}
	if !config.IsExplicit() {
		return nil
	}
	if config.IsEmpty() {
		return nil
	}
	for name, cv := range config.Values() {
		var message string
		if nameInError {
			message = fmt.Sprintf("$%s was not declared with !default in the @used module.", name)
		} else {
			message = "This variable was not declared with !default in the @used module."
		}
		return v.exception(message, cv.ConfigurationSpan)
	}
	return nil
}

// VisitForwardRule evaluates @forward: the ambient configuration passes
// through the rule first; a non-empty with clause builds an explicit config,
// loads inside the @forward frame, forwards the module, then prunes consumed
// keys and asserts the remainder is empty (the emptiness check runs after
// the frame exits so its trace carries no @forward frame). The empty-with
// path swaps the adjusted configuration in for the load and restores the
// original after.
//
// Matches Dart: _EvaluateVisitor.visitForwardRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitForwardRule(stmt *value.ForwardRule) (value.Value, error) {
	oldConfiguration := v.configuration
	adjustedConfiguration := oldConfiguration.ThroughForward(stmt)

	if len(stmt.Configuration) > 0 {
		newConfig, err := v.addForwardConfiguration(adjustedConfiguration, stmt)
		if err != nil {
			return nil, err
		}

		err = v.loadModule(stmt.URL(), "@forward", stmt, newConfig, false, nil, func(loadedModule sassmodule.Module, firstLoad bool) error {
			if firstLoad {
				v.registerCommentsForModule(loadedModule)
			}
			return v.env.ForwardModule(loadedModule, stmt, stmt)
		})
		if err != nil {
			return nil, err
		}

		except := make(map[string]struct{})
		for _, cv := range stmt.Configuration {
			if !cv.IsGuarded {
				except[cv.Name()] = struct{}{}
			}
		}
		removeUsedConfiguration(adjustedConfiguration, newConfig, except)

		configuredVariables := make(map[string]struct{})
		for _, cv := range stmt.Configuration {
			configuredVariables[cv.Name()] = struct{}{}
		}
		for name := range newConfig.Values() {
			if _, ok := configuredVariables[name]; !ok {
				newConfig.Remove(name)
			}
		}

		if newConfig.IsExplicit() && !newConfig.IsEmpty() {
			vals := newConfig.Values()
			for _, cv := range vals {
				return nil, v.exception(
					"This variable was not declared with !default in the @used module.",
					cv.ConfigurationSpan)
			}
		}
	} else {
		v.configuration = adjustedConfiguration
		err := v.loadModule(stmt.URL(), "@forward", stmt, nil, false, nil, func(loadedModule sassmodule.Module, firstLoad bool) error {
			if firstLoad {
				v.registerCommentsForModule(loadedModule)
			}
			return v.env.ForwardModule(loadedModule, stmt, stmt)
		})
		v.configuration = oldConfiguration
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// addForwardConfiguration folds a @forward with clause into the ambient
// configuration and returns the result. Guarded entries keep the existing
// non-nil value; others evaluate slash-stripped at the value node into
// explicit configured values. Explicitness is sticky: an explicit or empty
// base stays explicit, otherwise the result is implicit.
//
// Matches Dart: _EvaluateVisitor._addForwardConfiguration (evaluate.dart
// visitForwardRule section).
func (v *EvaluateVisitor) addForwardConfiguration(adjustedConfig *configuration.Configuration, stmt *value.ForwardRule) (*configuration.Configuration, error) {
	if len(stmt.Configuration) == 0 {
		return nil, nil
	}

	existingValues := adjustedConfig.Values()
	newValues := make(map[string]*configuration.ConfiguredValue, len(existingValues))
	for k, val := range existingValues {
		newValues[k] = val
	}

	for _, cv := range stmt.Configuration {
		if cv.IsGuarded {
			if oldVal := adjustedConfig.Remove(cv.Name()); oldVal != nil && oldVal.Value != value.Null {
				newValues[cv.Name()] = oldVal
				continue
			}
		}

		val, err := v.eval(cv.Expression)
		if err != nil {
			return nil, err
		}
		val, err = v.withoutSlash(val, cv.Expression)
		if err != nil {
			return nil, err
		}
		span, err := cv.Span()
		if err != nil {
			return nil, err
		}
		newValues[cv.Name()] = configuration.NewConfiguredValueExplicit(val, &span, cv.Expression)
	}

	if adjustedConfig.IsExplicit() || adjustedConfig.IsEmpty() {
		return configuration.NewExplicitConfiguration(newValues, stmt).Configuration, nil
	}
	return configuration.NewConfigurationImplicit(newValues), nil
}

// loadedStylesheet is the result of loadStylesheet: the parsed stylesheet
// plus the importer that resolved it and whether it counts as a dependency.
//
// Matches Dart: the _LoadedStylesheet record (evaluate.dart loading
// section).
type loadedStylesheet struct {
	stylesheet   *value.Stylesheet
	importer     Importer
	isDependency bool
}

// loadStylesheet canonicalizes parsedURL and loads the stylesheet, or
// returns a SassRuntimeException when it cannot be found. It tries the import
// cache relative to baseURL (defaulting to the current stylesheet's source
// URL), warning on relative canonical URLs and recording the canonical URL
// as loaded even when the load itself fails so watchers still watch it, then
// falls back to the node importer and the package-URL platform error. Every
// failure surfaces spanned at span; the import span is cleared on exit.
//
// Matches Dart: _EvaluateVisitor._loadStylesheet (evaluate.dart loading
// section); SassException values rethrow while other errors are wrapped with
// a trace at the call span.
func (v *EvaluateVisitor) loadStylesheet(parsedURL *url.URL, span sasscommon.FileSpan, baseURL *url.URL, forImport bool) (*loadedStylesheet, error) {
	v.ec.SetImportSpan(span)
	defer v.ec.SetImportSpan(sasscommon.SimpleFileSpan{})

	// Default the base URL to the current stylesheet's source URL.
	if baseURL == nil && v.stylesheet != nil {
		if sspan, err := v.stylesheet.Span(); err == nil {
			baseURL, _ = sspan.SourceURL()
		}
	}

	result, err := v.importCache.Canonicalize(parsedURL, v.importer, baseURL, forImport)
	if err != nil {
		// Spanned Sass errors rethrow as-is; argument errors and other
		// failures wrap with a trace at the call span.
		if _, ok := err.(*sasscommon.SassRuntimeException); ok {
			return nil, err
		}
		if _, ok := err.(*sasscommon.SassException); ok {
			return nil, err
		}
		if _, ok := err.(*sasscommon.SassFormatException); ok {
			return nil, err
		}
		// Argument errors wrap with their message; anything else wraps with
		// the importer error text.
		if ae, ok := err.(*sasscommon.ArgumentError); ok {
			return nil, sasscommon.ThrowWithTrace(v.exception(ae.Error(), nil), err)
		}
		return nil, sasscommon.ThrowWithTrace(v.exception(err.Error(), nil), err)
	}
	if result != nil {
		// Relative canonical URLs are deprecated; warn but continue loading.
		if result.CanonicalURL.Scheme == "" {
			if err := v.ec.WarnDeprecation(
				fmt.Sprintf("Importer %v canonicalized %s to %s.\n"+
					"Relative canonical URLs are deprecated and will eventually be disallowed.",
					result.Importer, parsedURL, result.CanonicalURL),
				deprecation.RelativeCanonical,
			); err != nil {
				return nil, err
			}
		}

		// Record the canonical URL as loaded even if the actual load fails,
		// because watchers should watch it to see if it changes in a way
		// that allows the load to succeed.
		canonicalKey := result.CanonicalURL.String()
		v.loadedUrls.Add(canonicalKey)

		isDependency := v.inDependency || result.Importer != v.importer

		sheet, err := v.importCache.ImportCanonical(result.Importer, result.CanonicalURL, result.OriginalURL)
		if err != nil {
			// Same span-preserving rethrow/wrap as canonicalization above.
			if _, ok := err.(*sasscommon.SassRuntimeException); ok {
				return nil, err
			}
			if _, ok := err.(*sasscommon.SassException); ok {
				return nil, err
			}
			if _, ok := err.(*sasscommon.SassFormatException); ok {
				return nil, err
			}
			if ae, ok := err.(*sasscommon.ArgumentError); ok {
				return nil, sasscommon.ThrowWithTrace(v.exception(ae.Error(), nil), err)
			}
			return nil, sasscommon.ThrowWithTrace(v.exception(err.Error(), nil), err)
		}
		// The importer may canonicalize a URL but fail to load it (nil sheet
		// without error). This falls through to the node-importer fallback
		// or the "Can't find stylesheet" error below.
		if sheet != nil {
			return &loadedStylesheet{
				stylesheet:   sheet,
				importer:     result.Importer,
				isDependency: isDependency,
			}, nil
		}
	}

	// Node importer fallback: try the Node.js-compatible importer before
	// reporting the stylesheet as missing.
	if v.nodeImporter != nil {
		urlStr := parsedURL.String()
		ls, err := v.importLikeNode(urlStr, baseURL, forImport)
		if err != nil {
			// Same span-preserving rethrow/wrap as above.
			if _, ok := err.(*sasscommon.SassRuntimeException); ok {
				return nil, err
			}
			if _, ok := err.(*sasscommon.SassException); ok {
				return nil, err
			}
			if _, ok := err.(*sasscommon.SassFormatException); ok {
				return nil, err
			}
			if ae, ok := err.(*sasscommon.ArgumentError); ok {
				return nil, sasscommon.ThrowWithTrace(v.exception(ae.Error(), nil), err)
			}
			return nil, sasscommon.ThrowWithTrace(v.exception(err.Error(), nil), err)
		}
		if ls != nil {
			lsSpan2, err := ls.stylesheet.Span()
			if err != nil {
				return nil, err
			}
			srcURL, err := lsSpan2.SourceURL()
			if err != nil {
				return nil, err
			}
			if srcURL != nil {
				v.loadedUrls.Add(srcURL.String())
			}
			return ls, nil
		}
	}

	// Package URLs on the JS platform get a dedicated error; elsewhere they
	// fall through to the generic missing-stylesheet error below.
	if parsedURL.Scheme == "package" && IsJS {
		return nil, &sasscommon.SassRuntimeException{
			Message: "\"package:\" URLs aren't supported on this platform.",
			Span:    span,
			Trace:   v.stackTrace(nil),
		}
	}

	return nil, &sasscommon.SassRuntimeException{
		Message: "Can't find stylesheet to import.",
		Span:    span,
		Trace:   v.stackTrace(nil),
	}
}

// registerCommentsForModule stashes root-level loud comments for a freshly
// loaded module so leading comments travel with the module through CSS
// combination. No-op outside a module root or when the module carries no
// CSS; otherwise moves the current root children into
// preModuleComments[module] and resets the import-boundary counters.
//
// Matches Dart: _EvaluateVisitor._registerCommentsForModule (evaluate.dart
// module-configuration section).
func (v *EvaluateVisitor) registerCommentsForModule(module sassmodule.Module) {
	if v.root == nil {
		return
	}
	children := v.root.Children()
	if len(children) == 0 || !module.TransitivelyContainsCss() {
		return
	}
	if v.preModuleComments == nil {
		v.preModuleComments = make(map[sassmodule.Module][]value.CssComment)
	}
	var comments []value.CssComment
	for _, child := range children {
		if c, ok := child.(value.CssComment); ok {
			comments = append(comments, c)
		}
	}
	v.preModuleComments[module] = append(v.preModuleComments[module], comments...)
	v.root.ClearChildren()
	v.endOfImports = 0
	v.outOfOrderImports = nil
}

// importLikeNode imports a stylesheet through the node-package importer,
// returning nil when the import fails. Syntax comes from the result URL
// (file URLs map by path, others default to SCSS).
//
// Matches Dart: _EvaluateVisitor._importLikeNode (evaluate.dart loading
// section); the two-step relative-then-absolute load collapses here into a
// single Load call with the original URL text.
func (v *EvaluateVisitor) importLikeNode(originalURL string, previous *url.URL, forImport bool) (*loadedStylesheet, error) {
	// Unparseable URLs simply miss; the caller falls through to the
	// missing-stylesheet error.
	parsedURL, parseErr := sassurl.Parse(originalURL)
	if parseErr != nil {
		return nil, nil
	}
	result, err := v.nodeImporter.Load(parsedURL)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	isDependency := true

	// Syntax follows the result URL: file URLs map by path, others parse as
	// SCSS.
	var syntax Syntax
	if result.SourceMapURL() != nil {
		urlStr := result.SourceMapURL().String()
		if strings.HasPrefix(urlStr, "file") {
			syntax = SyntaxForPath(urlStr)
		} else {
			syntax = SyntaxSCSS
		}
	} else {
		syntax = SyntaxSCSS
	}

	stylesheet, err := parseStylesheet(
		result.Contents,
		result.SourceMapURL(),
		syntax,
		v.importCache.parseSelectors,
	)
	if err != nil {
		return nil, err
	}

	return &loadedStylesheet{
		stylesheet:   stylesheet,
		importer:     nil,
		isDependency: isDependency,
	}, nil
}

// VisitImportRule evaluates legacy @import by dispatching each import to
// dynamic (stylesheet inlining) or static (plain-CSS @import node) handling.
//
// Matches Dart: _EvaluateVisitor.visitImportRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitImportRule(stmt *value.ImportRule) (value.Value, error) {
	for _, imp := range stmt.Imports {
		if di, ok := imp.(*value.DynamicImport); ok {
			if err := v.visitDynamicImport(di); err != nil {
				return nil, err
			}
		} else {
			if err := v.visitStaticImport(imp.(*value.StaticImport)); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// visitDynamicImport inlines the stylesheet referenced by imp under an
// @import stack frame with loop detection. Module-free stylesheets evaluate
// directly in the current environment; stylesheets that @use/@forward get an
// isolated ForImport environment and root so @extend resolves hermetically
// before CSS is injected, with forwarded members imported and user-module CSS
// combined through combineCss plus the imported-CSS visitor.
//
// Matches Dart: _EvaluateVisitor._visitDynamicImport (evaluate.dart @import
// section); the fast path (no uses/forwards) and the
// built-in-only-modules shortcut share one loadsUserDefinedModules flag.
func (v *EvaluateVisitor) visitDynamicImport(imp *value.DynamicImport) error {
	u := imp.URL()
	if u == nil {
		return nil
	}

	_, err := WithStackFrame(v, "@import", imp, func() (struct{}, error) {
		// Resolve through the import cache as a for-import load.
		impSpan, err := imp.Span()
		if err != nil {
			return struct{}{}, err
		}
		ls, err := v.loadStylesheet(u, impSpan, nil, true)
		if err != nil {
			return struct{}{}, err
		}

		// Report a loop with new-load/original-load spans when the target is
		// already being loaded.
		lsSpan, err := ls.stylesheet.Span()
		if err != nil {
			return struct{}{}, err
		}
		canonicalUrlObj, err := lsSpan.SourceURL()
		if err != nil {
			return struct{}{}, err
		}
		canonicalUrl := canonicalUrlObj.String()
		if canonicalUrl != "" {
			if prev, ok := v.activeModules[canonicalUrl]; ok {
				if prev != nil {
					prevSpan, err := prev.Span()
					if err != nil {
						return struct{}{}, err
					}
					impSpan, err := imp.Span()
					if err != nil {
						return struct{}{}, err
					}
					// Label the new load against the original load span.
					return struct{}{}, &sasscommon.MultiSpanSassRuntimeException{
						Message:      "This file is already being loaded.",
						Span:         impSpan,
						PrimaryLabel: "new load",
						SecondarySpans: map[sasscommon.FileSpan]string{
							prevSpan: "original load",
						},
						Trace: v.stackTrace(impSpan),
					}
				}
				impSpan, err := imp.Span()
				if err != nil {
					return struct{}{}, err
				}
				return struct{}{}, v.exception("This file is already being loaded.", &impSpan)
			}
			v.activeModules[canonicalUrl] = imp
			defer delete(v.activeModules, canonicalUrl)
		}

		// Fast path: module-free stylesheets inject their CSS directly in the
		// current environment; module users need the isolated path below.
		if len(ls.stylesheet.Uses()) == 0 && len(ls.stylesheet.Forwards()) == 0 {
			oldStylesheet := v.stylesheet
			oldInDependency := v.inDependency
			oldImporter := v.importer
			v.stylesheet = ls.stylesheet
			v.inDependency = ls.isDependency
			v.importer = ls.importer
			_, err := v.VisitStylesheet(ls.stylesheet)
			v.stylesheet = oldStylesheet
			v.inDependency = oldInDependency
			v.importer = oldImporter
			return struct{}{}, err
		}

		// Only built-in modules still need a separate environment (so their
		// namespaces stay hidden), but without a fresh root since @extend
		// needs no hermetic resolution.
		loadsUserDefinedModules := false
		for _, use := range ls.stylesheet.Uses() {
			if use.URL().Scheme != "sass" {
				loadsUserDefinedModules = true
				break
			}
		}
		if !loadsUserDefinedModules {
			for _, fwd := range ls.stylesheet.Forwards() {
				if fwd.URL().Scheme != "sass" {
					loadsUserDefinedModules = true
					break
				}
			}
		}

		// Create a ForImport sub-environment.
		env := v.env.ForImport()
		var children []value.CssNode

		_, err = WithEnvironment(v, env, func() (struct{}, error) {
			// Save ambient visitor state for the imported stylesheet.
			oldImporter := v.importer
			oldStylesheet := v.stylesheet
			oldRoot := v.root
			oldParent := v.parent
			oldEndOfImports := v.endOfImports
			oldOutOfOrderImports := v.outOfOrderImports
			oldConfig := v.configuration
			oldInDependency := v.inDependency

			// Point the visitor at the imported stylesheet, with a fresh root
			// only when user modules require hermetic @extend handling.
			v.importer = ls.importer
			v.stylesheet = ls.stylesheet
			if loadsUserDefinedModules {
				lsSpan4, err := ls.stylesheet.Span()
				if err != nil {
					return struct{}{}, err
				}
				v.root = value.NewModifiableCssStylesheet(lsSpan4)
				v.parent = v.root
				v.endOfImports = 0
				v.outOfOrderImports = nil
			}
			v.inDependency = ls.isDependency

			// Lazily build an implicit configuration only when a @forward
			// may consume it.
			if len(ls.stylesheet.Forwards()) > 0 {
				config, err := env.ToImplicitConfiguration()
				if err != nil {
					return struct{}{}, err
				}
				v.configuration = config
			}

			_, err := v.VisitStylesheet(ls.stylesheet)

			// Snapshot spliced children before restoring the ambient root.
			if loadsUserDefinedModules {
				children = buildOutOfOrderChildren(v)
			}

			// Restore ambient visitor state.
			v.importer = oldImporter
			v.stylesheet = oldStylesheet
			v.configuration = oldConfig
			v.inDependency = oldInDependency
			if loadsUserDefinedModules {
				v.root = oldRoot
				v.parent = oldParent
				v.endOfImports = oldEndOfImports
				v.outOfOrderImports = oldOutOfOrderImports
			}

			return struct{}{}, err
		})
		if err != nil {
			return struct{}{}, err
		}

		// Expose forwarded members, then inject combined CSS plus any
		// directly imported CSS through the imported-CSS path.
		module := env.ToDummyModule()
		if err := v.env.ImportForwards(module); err != nil {
			return struct{}{}, err
		}

		// Inject CSS only when user modules were loaded: combine upstream
		// CSS (cloning when extensions require it) and replay children
		// through the imported-CSS visitor.
		if loadsUserDefinedModules {
			if module.TransitivelyContainsCss() {
				combined, err := v.combineCss(module, module.TransitivelyContainsExtensions())
				if err != nil {
					return struct{}{}, err
				}
				if _, err := combined.AcceptVoid(v); err != nil {
					return struct{}{}, err
				}
			}
			vis := &importedCssVisitor{v: v}
			for _, child := range children {
				if _, err := child.AcceptVoid(vis); err != nil {
					return struct{}{}, err
				}
			}
		}

		return struct{}{}, nil
	})
	return err
}

// visitStaticImport emits a plain-CSS @import node for imp, interpolating
// URL and modifiers. Nested imports attach under the current parent (copying
// it after siblings first); root-level imports join the leading-import block
// via the endOfImports/outOfOrderImports bookkeeping.
//
// Matches Dart: _EvaluateVisitor._visitStaticImport (evaluate.dart @import
// section); mirrors visitCssImport logic.
func (v *EvaluateVisitor) visitStaticImport(imp *value.StaticImport) error {
	urlStr, err := v.interpolateSelector(imp.URL)
	if err != nil {
		return err
	}
	urlSpan, err := imp.URL.Span()
	if err != nil {
		return err
	}
	urlVal := sasscommon.NewCssValue(urlStr, urlSpan)
	var modifiers *sasscommon.CssValue[string]
	if imp.Modifiers != nil {
		modStr, err := v.interpolateSelector(imp.Modifiers)
		if err != nil {
			return err
		}
		modSpan, err := imp.Modifiers.Span()
		if err != nil {
			return err
		}
		mv := sasscommon.NewCssValue(modStr, modSpan)
		modifiers = &mv
	}
	impSpan2, err := imp.Span()
	if err != nil {
		return err
	}
	cssImport := value.NewModifiableCssImport(urlVal, impSpan2, modifiers)
	if v.parent != v.root {
		if err := v.copyParentAfterSibling(); err != nil {
			return err
		}
		if err := v.parent.AddChild(cssImport); err != nil {
			return err
		}
	} else if v.endOfImports == len(v.root.Children()) {
		if err := v.root.AddChild(cssImport); err != nil {
			return err
		}
		v.endOfImports++
	} else {
		v.outOfOrderImports = append(v.outOfOrderImports, cssImport)
	}
	return nil
}

// buildOutOfOrderChildren returns a copy of the root children with
// out-of-order imports spliced after endOfImports. With no out-of-order
// imports it returns the live children slice.
//
// Matches Dart: _EvaluateVisitor._addOutOfOrderImports (evaluate.dart module
// section).
func buildOutOfOrderChildren(v *EvaluateVisitor) []value.CssNode {
	if v.outOfOrderImports == nil {
		return v.root.Children()
	}
	children := v.root.Children()
	result := make([]value.CssNode, 0, len(v.outOfOrderImports)+len(children))
	for i := 0; i < v.endOfImports && i < len(children); i++ {
		result = append(result, children[i])
	}
	for _, imp := range v.outOfOrderImports {
		result = append(result, imp)
	}
	for i := v.endOfImports; i < len(children); i++ {
		result = append(result, children[i])
	}
	return result
}

// VisitMixinRule registers a mixin definition as a closure-capturing
// user-defined callable in the current environment.
//
// Matches Dart: _EvaluateVisitor.visitMixinRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitMixinRule(stmt *value.MixinRule) (value.Value, error) {
	callable := NewUserDefinedCallable(stmt, v.env.Closure(), v.inDependency)
	v.env.SetMixin(callable)
	return nil, nil
}

// VisitIncludeRule evaluates @include: resolves the mixin (span-wrapped so a
// missing namespace reports at the call site), rejects --named mixins
// defined without the prefix for plain-CSS future-proofing, rejects calling
// a @function as a mixin, wraps any content block in a closure-capturing
// callable, and runs the body through applyMixin with the raw argument list
// (evaluation happens inside).
//
// Matches Dart: _EvaluateVisitor.visitIncludeRule (evaluate.dart statement
// visitors); the content-less invocation span collapses into stmt here.
func (v *EvaluateVisitor) VisitIncludeRule(stmt *value.IncludeRule) (value.Value, error) {
	var mix sasscallable.Callable

	if ns := stmt.Namespace(); ns != nil {
		mod := v.env.Modules()[*ns]
		if mod == nil {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Undefined module.", &span)
		}
		var ok bool
		mix, ok = mod.Mixins().Get(stmt.Name())
		if !ok {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Undefined mixin.", &span)
		}
	} else {
		var err error
		// Span-wrap the lookup so a missing name reports at the call site.
		mix, err = addExceptionSpan(v, stmt, func() (sasscallable.Callable, error) {
			return v.env.GetMixin(stmt.Name(), nil)
		}, nil)
		if err != nil {
			return nil, err
		}
		if mix == nil {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Undefined mixin.", &span)
		}
	}

	if stmt.OriginalName != "" && strings.HasPrefix(stmt.OriginalName, "--") {
		if udMix, ok := mix.(*functions.UserDefinedCallable); ok && !strings.HasPrefix(udMix.Declaration().OriginalName(), "--") {
			span, err := stmt.NameSpan()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Sass @mixin names beginning with -- are forbidden for forward-compatibility with plain CSS mixins.\n\nFor details, see https://sass-lang.com/d/css-function-mixin", &span)
		}
	}

	if udMix, ok := mix.(*functions.UserDefinedCallable); ok {
		if _, isMixin := udMix.Declaration().(*value.MixinRule); !isMixin {
			span, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			return nil, v.exception("Function used as mixin.", &span)
		}
	}

	// Wrap an attached content block in a closure-capturing callable.
	var contentCallable *functions.UserDefinedCallable
	if stmt.Content != nil {
		contentCallable = NewUserDefinedCallable(stmt.Content, v.env.Closure(), v.inDependency)
	}

	// Pass the raw argument list; evaluation happens inside applyMixin and
	// runBuiltInCallable.
	return nil, v.applyMixin(mix, contentCallable, stmt, stmt.Arguments())
}

// runBuiltInCallable evaluates arguments and invokes a built-in callable,
// verifying parameters, filling named/default parameters, packing excess
// positionals plus leftover named arguments into a trailing SassArgumentList,
// and invoking the overload inside the call-site span. Unused keywords after
// the call report "No parameter named $x" as a multi-span error.
//
// Matches Dart: _EvaluateVisitor._runBuiltInCallable (evaluate.dart callable
// section); overload selection errors propagate with the callable node set on
// the evaluation context, and non-Sass callback failures are trace-wrapped at
// the invocation span.
func (v *EvaluateVisitor) runBuiltInCallable(callable *functions.BuiltInCallable, nodeWithSpan sasscommon.AstNode, arguments *value.ArgumentList) (value.Value, error) {
	oldCallableNode := v.ec.CallableNode()
	v.ec.SetCallableNode(nodeWithSpan)

	// Evaluate positional/named arguments once; overload selection and
	// default filling below consume the results.
	results, err := v.evaluateArguments(arguments)
	if err != nil {
		v.ec.SetCallableNode(oldCallableNode)
		return nil, err
	}
	positional := results.positional
	named := results.named
	separator := results.separator

	names := make(map[string]struct{})
	for k := range named.Keys() {
		names[k] = struct{}{}
	}
	overload, err := callable.CallbackFor(len(positional), names)
	if err != nil {
		v.ec.SetCallableNode(oldCallableNode)
		return nil, err
	}

	if overload.Params != nil {
		_, err := addExceptionSpan(v, nodeWithSpan, func() (struct{}, error) {
			return struct{}{}, overload.Params.Verify(len(positional), names)
		}, nil)
		if err != nil {
			v.ec.SetCallableNode(oldCallableNode)
			return nil, err
		}
		for i := len(positional); i < len(overload.Params.Parameters); i++ {
			param := overload.Params.Parameters[i]
			if val, ok := named.Get(param.Name()); ok {
				positional = append(positional, val)
				named.Delete(param.Name())
			} else if param.DefaultValue != nil {
				def, defErr := v.eval(param.DefaultValue)
				if defErr != nil {
					v.ec.SetCallableNode(oldCallableNode)
					return nil, defErr
				}
				defNode, varErr := v.expressionNode(param.DefaultValue)
				if varErr != nil {
					v.ec.SetCallableNode(oldCallableNode)
					return nil, varErr
				}
				def, defErr = v.withoutSlash(def, defNode)
				if defErr != nil {
					v.ec.SetCallableNode(oldCallableNode)
					return nil, defErr
				}
				positional = append(positional, def)
			}
		}
	}

	var restArgList *value.SassArgumentList
	if overload.Params != nil && overload.Params.RestParameter != nil {
		var rest []value.Value
		if len(positional) > len(overload.Params.Parameters) {
			rest = make([]value.Value, len(positional)-len(overload.Params.Parameters))
			copy(rest, positional[len(overload.Params.Parameters):])
			positional = positional[:len(overload.Params.Parameters)]
		}
		sep := separator
		if sep == value.ListSeparatorUndecided {
			sep = value.ListSeparatorComma
		}
		restArgList, err = value.NewSassArgumentList(rest, named, sep)
		if err != nil {
			v.ec.SetCallableNode(oldCallableNode)
			return nil, err
		}
		positional = append(positional, restArgList)
	}

	result, err := addExceptionSpan(v, nodeWithSpan, func() (value.Value, error) {
		return overload.Fn(v.ec, positional)
	}, nil)
	if err != nil {
		// Spanned Sass errors rethrow as-is below.
		if _, ok := err.(*sasscommon.SassRuntimeException); ok {
			return result, err
		}
		if _, ok := err.(*sasscommon.MultiSpanSassRuntimeException); ok {
			return result, err
		}
		if _, ok := err.(*sasscommon.MultiSpanSassException); ok {
			return result, err
		}
		if _, ok := err.(*sasscommon.SassException); ok {
			return result, err
		}
		if _, ok := err.(*sasscommon.SassFormatException); ok {
			return result, err
		}
		// Non-Sass callback failures wrap with a trace at the invocation span.
		sp, spanErr := nodeWithSpan.Span()
		if spanErr != nil {
			v.ec.SetCallableNode(oldCallableNode)
			return nil, spanErr
		}
		return nil, sasscommon.ThrowWithTrace(
			&sasscommon.SassRuntimeException{
				Message: err.Error(),
				Span:    sp,
				Trace:   v.stackTrace(sp),
			}, err)
	}
	v.ec.SetCallableNode(oldCallableNode)

	if restArgList == nil {
		return result, nil
	}
	kw := restArgList.KeywordsWithoutMarking()
	if kw.Len() == 0 || restArgList.WereKeywordsAccessed() {
		return result, nil
	}
	for name := range kw.Keys() {
		sp, spanErr := nodeWithSpan.Span()
		if spanErr != nil {
			return nil, spanErr
		}
		return nil, &sasscommon.MultiSpanSassRuntimeException{
			Message:        fmt.Sprintf("No parameter named $%s.", name),
			Span:           sp,
			PrimaryLabel:   "invocation",
			SecondarySpans: map[sasscommon.FileSpan]string{},
			Trace:          v.stackTrace(sp),
		}
	}
	return result, nil
}

// applyMixin evaluates mixin mix with arguments and optional content
// contentCallable. Nil mixin reports "Undefined mixin"; built-ins that
// reject content report a multi-span invocation/declaration error, otherwise
// run with content and in-mixin flags swapped in; user-defined mixins run
// through runUserDefinedCallable with each body statement wrapped in an
// error span so @content failures point at the @include.
//
// Matches Dart: _EvaluateVisitor._applyMixin (evaluate.dart @include
// section); the two nodeWithSpan arguments (with and without content)
// collapse into one here.
func (v *EvaluateVisitor) applyMixin(mix sasscallable.Callable, contentCallable *functions.UserDefinedCallable, nodeWithSpan sasscommon.AstNode, arguments *value.ArgumentList) error {
	if mix == nil {
		span, err := nodeWithSpan.Span()
		if err != nil {
			return err
		}
		return v.exception("Undefined mixin.", &span)
	}

	switch c := mix.(type) {
	case *functions.BuiltInCallable:
		if !c.AcceptsContent() && contentCallable != nil {
			// Evaluate arguments only to pick the overload for the
			// declaration span in the content-rejection error.
			results, err := v.evaluateArguments(arguments)
			if err != nil {
				return err
			}
			names := make(map[string]struct{})
			for name := range results.named.Keys() {
				names[name] = struct{}{}
			}
			overload, err := c.CallbackFor(len(results.positional), names)
			if err != nil {
				return err
			}
			nwsSpan, err := nodeWithSpan.Span()
			if err != nil {
				return err
			}
			return &sasscommon.MultiSpanSassRuntimeException{
				Message:      "Mixin doesn't accept a content block.",
				Span:         nwsSpan,
				PrimaryLabel: "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{
					overload.Params.SpanWithName(): "declaration",
				},
				Trace: v.stackTrace(nwsSpan),
			}
		}
		oldContent := v.env.Content()
		if contentCallable != nil {
			v.env.SetContent(contentCallable)
		} else {
			v.env.SetContent(nil)
		}
		oldInMixin := v.env.InMixin()
		v.env.SetInMixin(true)
		_, err := v.runBuiltInCallable(c, nodeWithSpan, arguments)
		v.env.SetInMixin(oldInMixin)
		v.env.SetContent(oldContent)
		return err

	case *functions.UserDefinedCallable:
		// Mixins declared without content reject a content block with a
		// multi-span invocation/declaration error.
		if mr, isMixin := c.Declaration().(*value.MixinRule); contentCallable != nil && c.Declaration() != nil && isMixin && !mr.HasContent() {
			nwsSpan, err := nodeWithSpan.Span()
			if err != nil {
				return err
			}
			// Point at the invocation with the declaration as context.
			return &sasscommon.MultiSpanSassRuntimeException{
				Message:      "Mixin doesn't accept a content block.",
				Span:         nwsSpan,
				PrimaryLabel: "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{
					mr.Parameters().SpanWithName(): "declaration",
				},
				Trace: v.stackTrace(nwsSpan),
			}
		}
		_, err := v.runUserDefinedCallable(c, arguments, nodeWithSpan, func() (value.Value, error) {
			oldContent := v.env.Content()
			if contentCallable != nil {
				v.env.SetContent(contentCallable)
			} else {
				v.env.SetContent(nil)
			}
			oldInMixin := v.env.InMixin()
			v.env.SetInMixin(true)

			for _, child := range c.Declaration().GetChildren() {
				if _, err := addErrorSpan(v, nodeWithSpan, func() (value.Value, error) {
					return v.visitStatement(child)
				}); err != nil {
					v.env.SetInMixin(oldInMixin)
					v.env.SetContent(oldContent)
					return nil, err
				}
			}

			v.env.SetInMixin(oldInMixin)
			v.env.SetContent(oldContent)
			return nil, nil
		})
		return err

	default:
		panic(fmt.Sprintf("Unknown callable type %T.", mix))
	}
}

// VisitFunctionRule registers a function definition as a closure-capturing
// user-defined callable in the current environment.
//
// Matches Dart: _EvaluateVisitor.visitFunctionRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitFunctionRule(stmt *value.FunctionRule) (value.Value, error) {
	callable := NewUserDefinedCallable(stmt, v.env.Closure(), v.inDependency)
	v.env.SetFunction(callable)
	return nil, nil
}

// VisitReturnRule evaluates @return and yields its slash-stripped value to
// the enclosing block; the non-nil return is what handleReturn
// short-circuits on.
//
// Matches Dart: _EvaluateVisitor.visitReturnRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitReturnRule(stmt *value.ReturnRule) (value.Value, error) {
	val, err := v.eval(stmt.Expression)
	if err != nil {
		return nil, err
	}
	val, err = v.withoutSlash(val, stmt.Expression)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// VisitEachRule evaluates @each: binds each item (destructuring into
// multiple variables when declared) at the list expression node with
// per-item and per-sub-item slash-stripping, runs the body per item, and
// propagates @return out of the whole loop. Missing destructured slots bind
// nil.
//
// Matches Dart: _EvaluateVisitor.visitEachRule plus _setMultipleVariables
// (evaluate.dart statement visitors); runs in a semi-global scope.
func (v *EvaluateVisitor) VisitEachRule(stmt *value.EachRule) (value.Value, error) {
	listVal, err := v.eval(stmt.List)
	if err != nil {
		return nil, err
	}

	items, err := listVal.AsList()
	if err != nil {
		return nil, err
	}
	nodeWithSpan, nserr := v.expressionNode(stmt.List)
	if nserr != nil {
		return nil, nserr
	}

	return sassenv.Scope(v.env, func() (value.Value, error) {
		return handleReturn(v, items, func(item value.Value) (value.Value, error) {
			if len(stmt.Variables) == 1 {
				cleaned, err := v.withoutSlash(item, nodeWithSpan)
				if err != nil {
					return nil, err
				}
				v.env.SetLocalVariable(stmt.Variables[0], cleaned, nodeWithSpan)
			} else {
				subList, err := item.AsList()
				if err != nil {
					return nil, err
				}
				minLen := min(len(stmt.Variables), len(subList))
				for vi := range minLen {
					cleaned, err := v.withoutSlash(subList[vi], nodeWithSpan)
					if err != nil {
						return nil, err
					}
					v.env.SetLocalVariable(stmt.Variables[vi], cleaned, nodeWithSpan)
				}
				for vi := minLen; vi < len(stmt.Variables); vi++ {
					v.env.SetLocalVariable(stmt.Variables[vi], value.Null, nodeWithSpan)
				}
			}
			return handleReturn(v, stmt.GetChildren(), func(child value.Statement) (value.Value, error) {
				return v.visitStatement(child)
			})
		})
	}, true, true)
}

// VisitForRule evaluates @for from from through/to to, binding an integer
// per step. Both bounds evaluate inside their expression spans with the to
// bound coerced to the from units; the loop runs inclusive (through) or
// exclusive (to), counting down when from exceeds to. The variable is defined
// at the from expression node in a semi-global scope, and @return in the body
// unwinds the loop.
//
// Matches Dart: _EvaluateVisitor.visitForRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitForRule(stmt *value.ForRule) (value.Value, error) {
	fromVal, err := v.eval(stmt.From)
	if err != nil {
		return nil, err
	}
	toVal, err := v.eval(stmt.To)
	if err != nil {
		return nil, err
	}

	fromNum, err := addExceptionSpan(v, stmt.From, func() (value.SassNumber, error) {
		return value.AssertNumber(fromVal, nil)
	}, nil)
	if err != nil {
		return nil, err
	}
	toNum, err := addExceptionSpan(v, stmt.To, func() (value.SassNumber, error) {
		return value.AssertNumber(toVal, nil)
	}, nil)
	if err != nil {
		return nil, err
	}

	from, err := addExceptionSpan(v, stmt.From, func() (int64, error) {
		return fromNum.AssertInt(nil)
	}, nil)
	if err != nil {
		return nil, err
	}
	coercedTo, err := addExceptionSpan(v, stmt.To, func() (value.SassNumber, error) {
		return toNum.Coerce(fromNum.NumNumeratorUnits(), fromNum.NumDenominatorUnits(), nil)
	}, nil)
	if err != nil {
		return nil, err
	}
	to, err := addExceptionSpan(v, stmt.To, func() (int64, error) {
		return coercedTo.AssertInt(nil)
	}, nil)
	if err != nil {
		return nil, err
	}

	dir := int64(1)
	if from > to {
		dir = -1
	}
	if !stmt.IsExclusive {
		to += dir
	}
	if from == to {
		return nil, nil
	}

	// Loop variables live at the from-expression node for slash-division
	// warnings.
	fromNode, err := v.expressionNode(stmt.From)
	if err != nil {
		return nil, err
	}
	return sassenv.Scope(v.env, func() (value.Value, error) {
		for i := from; i != to; i += dir {
			v.env.SetLocalVariable(stmt.Variable, value.SassNumberWithUnits(float64(i), fromNum.NumNumeratorUnits(), fromNum.NumDenominatorUnits()), fromNode)
			result, err := handleReturn(v, stmt.GetChildren(), func(child value.Statement) (value.Value, error) {
				return v.visitStatement(child)
			})
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
		}
		return nil, nil
	}, true, true)
}

// VisitWhileRule evaluates @while: re-tests the condition each iteration in
// one shared semi-global scope (gated on the body's declarations) so
// variables persist across iterations; @return in the body unwinds the loop.
//
// Matches Dart: _EvaluateVisitor.visitWhileRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitWhileRule(stmt *value.WhileRule) (value.Value, error) {
	return sassenv.Scope(v.env, func() (value.Value, error) {
		for {
			cond, err := v.eval(stmt.Condition)
			if err != nil {
				return nil, err
			}
			if !cond.IsTruthy() {
				return nil, nil
			}
			childResult, err := handleReturn(v, stmt.GetChildren(), func(child value.Statement) (value.Value, error) {
				return v.visitStatement(child)
			})
			if err != nil {
				return nil, err
			}
			if childResult != nil {
				return childResult, nil
			}
		}
	}, true, stmt.HasDeclarations())
}

// VisitIfRule evaluates @if/@else-if/@else: the first clause with a truthy
// condition wins, falling back to the final else clause when present. The
// chosen children run in a semi-global scope gated on that clause's
// declarations, with @return propagating out.
//
// Matches Dart: _EvaluateVisitor.visitIfRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitIfRule(stmt *value.IfRule) (value.Value, error) {
	// Default to the else clause; the loop below overwrites it on the first
	// truthy condition.
	var clause value.IfRuleClause
	if stmt.LastClause != nil {
		clause = stmt.LastClause
	}
	// Walk clauses in order, stopping at the first truthy condition.
	for _, clauseToCheck := range stmt.Clauses {
		cond, err := v.eval(clauseToCheck.Expression)
		if err != nil {
			return nil, err
		}
		if cond.IsTruthy() {
			clause = clauseToCheck
			break
		}
	}

	// No truthy branch and no else: emit nothing.
	if clause == nil {
		return nil, nil
	}

	return sassenv.Scope(v.env, func() (value.Value, error) {
		result, err := handleReturn(v, clause.Children(), func(child value.Statement) (value.Value, error) {
			return v.visitStatement(child)
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}, true, clause.HasDeclarations())
}

// VisitMediaRule evaluates @media: interpolates the query (color warnings
// on) and parses it with the interpolation map, merges with enclosing
// queries (an empty merge drops the rule), then evaluates children with
// style-rule/media bubbling. Inside a style rule the rule copies inward so
// bare declarations have a home; plain-CSS nesting short-circuits to flat
// scoped evaluation.
//
// Matches Dart: _EvaluateVisitor.visitMediaRule plus _visitMediaQueries and
// _mergeMediaQueries (evaluate.dart statement visitors); children scope on
// hasDeclarations.
func (v *EvaluateVisitor) VisitMediaRule(stmt *value.MediaRule) (value.Value, error) {
	if v.declarationName != "" {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Media rules may not be used within nested declarations.", &span)
	}

	queryText, im, err := v.performInterpolationWithMap(stmt.Query, &[]bool{true}[0])
	if err != nil {
		return nil, err
	}

	cssQueries := parseMediaQueryTextWithMap(queryText, im)

	// If the user has already opted into plain CSS nesting, don't bother with
	// any merging or bubbling.
	if v.hasCssNesting() {
		stmtSpan4, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		rule, err := value.NewModifiableCssMediaRule(cssQueries, stmtSpan4)
		if err != nil {
			return nil, err
		}
		err = v.withParent(rule, func() error {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(false))
		return nil, err
	}

	var mergedQueries []*value.CssMediaQuery
	if v.mediaQueries != nil {
		mergedQueries = v.mergeMediaQueries(v.mediaQueries, queriesToPtrs(cssQueries))
	}
	if mergedQueries != nil && len(mergedQueries) == 0 {
		return nil, nil
	}

	var ruleQueries []value.CssMediaQuery
	var mergedSources *linkedhashmap.LinkedHashSet[*value.CssMediaQuery]
	if mergedQueries != nil {
		ruleQueries = queriesFromPtrs(mergedQueries)
		mergedSources = linkedhashmap.NewLinkedHashSet[*value.CssMediaQuery](value.MediaQueryHashEqual)
		for _, q := range v.mediaQueries {
			mergedSources.Add(q)
		}
		if v.mediaQuerySources != nil {
			for q := range v.mediaQuerySources.Keys() {
				mergedSources.Add(q)
			}
		}
		for _, q := range queriesToPtrs(cssQueries) {
			mergedSources.Add(q)
		}
	} else {
		ruleQueries = cssQueries
	}
	var rule *value.ModifiableCssMediaRule
	stmtSpan5, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	rule, err = value.NewModifiableCssMediaRule(ruleQueries, stmtSpan5)
	if err != nil {
		return nil, err
	}

	through := func(n value.CssNode) bool {
		if _, ok := n.(value.CssStyleRule); ok {
			return true
		}
		if mergedSources.Len() > 0 {
			if mr, ok := n.(value.CssMediaRule); ok {
				mrQueries := mr.Queries()
				allContained := true
				for i := range mrQueries {
					if !mergedSources.Contains(&mrQueries[i]) {
						allContained = false
						break
					}
				}
				if allContained {
					return true
				}
			}
		}
		return false
	}

	err = v.withParent(rule, func() error {
		return v.withMediaQueries(ruleQueries, mergedSources, func() error {
			if sr := v.styleRule(); sr != nil {
				modSr, ok := sr.(value.ModifiableCssParentNode)
				if ok {
					newParent, err := modSr.CopyWithoutChildren()
					if err != nil {
						return err
					}
					return v.withParent(newParent, func() error {
						for _, child := range stmt.GetChildren() {
							if _, err := v.visitStatement(child); err != nil {
								return err
							}
						}
						return nil
					}, nil, new(false))
				}
			}
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		})
	}, through, new(stmt.HasDeclarations()))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// warnForBogusCombinators emits bogus-combinators deprecation warnings for
// rule. Rules invisible for other reasons are skipped; otherwise useless
// selectors, leading-combinator selectors (non-plain-CSS only), and
// nesting-only selectors with non-style-rule children each get their message
// shape, the last as a multi-span pointing at the offending child.
//
// Matches Dart: _EvaluateVisitor._warnForBogusCombinators (evaluate.dart
// statement visitors).
func (v *EvaluateVisitor) warnForBogusCombinators(rule value.CssStyleRule) error {
	// Skip rules invisible for other reasons (selector invisible or all
	// children invisible).
	if rule.IsInvisibleOtherThanBogusCombinators() {
		return nil
	}
	sel := rule.Selector()
	if sel == nil {
		return nil
	}
	for _, complex := range sel.Components {
		if !complex.IsBogus() {
			continue
		}
		compSpan, err := complex.Span()
		if err != nil {
			return err
		}
		cs, err := complex.String()
		if err != nil {
			return err
		}
		trimmed := strings.TrimSpace(cs)
		if complex.IsUseless() {
			trimmedSpan, err := compSpan.TrimRight()
			if err != nil {
				return err
			}
			if err := v.warn(
				fmt.Sprintf(`The selector "%s" is invalid CSS. It will be omitted from the generated CSS.
This will be an error in Dart Sass 2.0.0.

More info: https://sass-lang.com/d/bogus-combinators`, trimmed),
				trimmedSpan,
				deprecation.BogusCombinators,
			); err != nil {
				return err
			}
		} else if len(complex.LeadingCombinators) > 0 {
			if v.stylesheet == nil || !v.stylesheet.IsPlainCss() {
				trimmedSpan, err := compSpan.TrimRight()
				if err != nil {
					return err
				}
				if err := v.warn(
					fmt.Sprintf(`The selector "%s" is invalid CSS.
This will be an error in Dart Sass 2.0.0.

More info: https://sass-lang.com/d/bogus-combinators`, trimmed),
					trimmedSpan,
					deprecation.BogusCombinators,
				); err != nil {
					return err
				}
			}
		} else {
			msg := fmt.Sprintf(`The selector "%s" is only valid for nesting and shouldn't
have children other than style rules.`, trimmed)
			if complex.IsBogusOtherThanLeadingCombinator() {
				msg += " It will be omitted from the generated CSS."
			}
			msg += "\nThis will be an error in Dart Sass 2.0.0.\n\nMore info: https://sass-lang.com/d/bogus-combinators"
			children := rule.Children()
			var secondarySpans map[sasscommon.FileSpan]string
			if len(children) > 0 {
				childSpan, err := children[0].Span()
				if err != nil {
					return err
				}
				secondarySpans = map[sasscommon.FileSpan]string{childSpan: "this is not a style rule"}
			}
			trimmedSpan3, err := compSpan.TrimRight()
			if err != nil {
				return err
			}
			span := sasscommon.NewMultiSpanFileSpan(trimmedSpan3, "invalid selector", secondarySpans)
			if err := v.warn(msg, span, deprecation.BogusCombinators); err != nil {
				return err
			}
		}
	}
	return nil
}

// VisitSupportsRule evaluates @supports: renders the condition to plain CSS
// and evaluates children with style-rule bubbling. Nested-declaration
// context is rejected; plain-CSS nesting short-circuits to scoped
// evaluation, otherwise the enclosing style rule copies inward so bare
// declarations have a home.
//
// Matches Dart: _EvaluateVisitor.visitSupportsRule (evaluate.dart statement
// visitors, mirrored in visitCssSupportsRule); children scope on
// hasDeclarations.
func (v *EvaluateVisitor) VisitSupportsRule(stmt *value.SupportsRule) (value.Value, error) {
	if v.declarationName != "" {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Supports rules may not be used within nested declarations.", &span)
	}

	conditionText, err := v.visitSupportsCondition(stmt.Condition)
	if err != nil {
		return nil, err
	}

	condSpan, err := stmt.Condition.Span()
	if err != nil {
		return nil, err
	}
	stmtSpan6, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	rule := value.NewModifiableCssSupportsRule(
		sasscommon.NewCssValue(conditionText, condSpan),
		stmtSpan6,
	)

	if v.hasCssNesting() {
		err = v.withParent(rule, func() error {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(stmt.HasDeclarations()))
		return nil, err
	}

	// Scope the child environment only when the rule can declare members.
	scopeWhen := stmt.HasDeclarations()
	err = v.withParent(rule, func() error {
		if sr := v.styleRule(); sr != nil {
			modSr, ok := sr.(value.ModifiableCssParentNode)
			if ok {
				newParent, err := modSr.CopyWithoutChildren()
				if err != nil {
					return err
				}
				return v.withParent(newParent, func() error {
					for _, child := range stmt.GetChildren() {
						if _, err := v.visitStatement(child); err != nil {
							return err
						}
					}
					return nil
				}, nil, nil)
			}
		}
		for _, child := range stmt.GetChildren() {
			if _, err := v.visitStatement(child); err != nil {
				return err
			}
		}
		return nil
	}, func(n value.CssNode) bool {
		_, ok := n.(value.CssStyleRule)
		return ok
	}, &scopeWhen)

	return nil, err
}

// VisitDeclaration evaluates a declaration: interpolates the property name
// (color warnings on, prefixed by any enclosing nested-declaration name),
// then evaluates and serializes the value. Blank values are dropped unless
// they are empty lists (preserved so the error surfaces) or custom
// properties (which allow empty values per spec). Source-map spans come from
// the declaration expression node; nested children evaluate in a scope gated
// on their declarations.
//
// Matches Dart: _EvaluateVisitor.visitDeclaration (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitDeclaration(stmt *value.Declaration) (value.Value, error) {
	sr := v.styleRule()
	if sr == nil && !v.inUnknownAtRule && !v.inKeyframes {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("Declarations may only be used within style rules.", &span)
	}

	if v.declarationName != "" && !stmt.ParsedAsSassScript() {
		var msg string
		if strings.HasPrefix(stmt.Name.InitialPlain(), "--") {
			msg = "Declarations whose names begin with \"--\" may not be nested."
		} else {
			msg = "Declarations parsed as raw CSS may not be nested."
		}
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception(msg, &span)
	}

	// Interpolate the property name with color warnings; nested
	// declarations later prefix it with the enclosing name.
	name, err := v.interpolationToValue(stmt.Name, false, true)
	if err != nil {
		return nil, err
	}

	// Handle declarationName prefix for nested declarations
	if v.declarationName != "" {
		nsp, err := name.Span()
		if err != nil {
			return nil, err
		}
		name = sasscommon.NewCssValue(v.declarationName+"-"+name.Value, nsp)
	}

	// If there's a value, evaluate and serialize it
	if stmt.Value != nil {
		val, err := v.eval(stmt.Value)
		if err != nil {
			return nil, err
		}

		// Blank value preservation: don't emit the declaration if the value
		// is blank, unless it's an empty list (preserved so the error is seen)
		// or a custom property (per spec, custom properties allow empty values).
		empty, err := isEmptyList(val)
		if err != nil {
			return nil, err
		}
		if !val.IsBlank() || empty || strings.HasPrefix(name.Value, "--") {
			if err := v.copyParentAfterSibling(); err != nil {
				return nil, err
			}

			valueSpan, err := stmt.Value.Span()
			if err != nil {
				return nil, err
			}
			stmtSpan7, err := stmt.Span()
			if err != nil {
				return nil, err
			}
			var valueSpanForMap *sasscommon.FileSpan
			if v.sourceMap {
				exprNode, nserr := v.expressionNode(stmt.Value)
				if nserr != nil {
					return nil, nserr
				}
				nsp, err := exprNode.Span()
				if err != nil {
					return nil, err
				}
				valueSpanForMap = &nsp
			}
			decl, err := value.NewModifiableCssDeclaration(
				name,
				sasscommon.NewCssValue[value.Value](val, valueSpan),
				stmtSpan7,
				stmt.ParsedAsSassScript(),
				valueSpanForMap,
			)
			if err != nil {
				return nil, err
			}
			if err := v.addChild(decl); err != nil {
				return nil, err
			}
		}
	}

	// Handle nested declarations (children)
	if len(stmt.GetChildren()) > 0 {
		oldDeclarationName := v.declarationName
		v.declarationName = name.Value

		_, err := sassenv.Scope(v.env, func() (struct{}, error) {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return struct{}{}, err
				}
			}
			return struct{}{}, nil
		}, false, stmt.HasDeclarations())

		v.declarationName = oldDeclarationName

		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// isEmptyList reports whether val stringifies as an empty list. Used by
// VisitDeclaration's blank-value guard so empty lists still emit (and error
// at serialization) instead of vanishing.
//
// Matches Dart: _EvaluateVisitor._isEmptyList (evaluate.dart statement
// visitors).
func isEmptyList(val value.Value) (bool, error) {
	l, err := val.AsList()
	if err != nil {
		return false, err
	}
	return len(l) == 0, nil
}

// VisitWarnRule emits @warn: evaluates the message inside the rule span and
// logs it with a stack trace. Strings log verbatim; other values serialize
// in inspect mode.
//
// Matches Dart: _EvaluateVisitor.visitWarnRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitWarnRule(stmt *value.WarnRule) (value.Value, error) {
	// Evaluate the message inside the rule span so errors point at the @warn.
	val, err := addExceptionSpan(v, stmt, func() (value.Value, error) {
		return v.eval(stmt.Expression)
	}, nil)
	if err != nil {
		return nil, err
	}
	var text string
	if sassStr, ok := val.(*value.SassString); ok {
		text = sassStr.Text
	} else {
		text, err = value.SerializeValue(val, true)
		if err != nil {
			return nil, err
		}
	}
	span, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	v.logger.Warn(text, nil, v.stackTrace(span))
	return nil, nil
}

// VisitDebugRule emits @debug: evaluates the expression and logs its
// inspect string on the debug channel. Plain strings log verbatim.
//
// Matches Dart: _EvaluateVisitor.visitDebugRule (evaluate.dart statement
// visitors); unlike @warn the expression is not exception-span wrapped.
func (v *EvaluateVisitor) VisitDebugRule(stmt *value.DebugRule) (value.Value, error) {
	val, err := v.eval(stmt.Expression)
	if err != nil {
		return nil, err
	}
	var text string
	if str, ok := val.(*value.SassString); ok {
		text = str.Text
	} else {
		text, err = value.SerializeValueInspect(val)
		if err != nil {
			return nil, err
		}
	}
	span, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	v.logger.Debug(text, &span)
	return nil, nil
}

// VisitErrorRule throws @error: evaluates the message and raises it at the
// rule span with the rule span as the innermost trace frame.
//
// Matches Dart: _EvaluateVisitor.visitErrorRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitErrorRule(stmt *value.ErrorRule) (value.Value, error) {
	val, err := v.eval(stmt.Expression)
	if err != nil {
		return nil, err
	}
	text, err := value.SerializeValue(val, true)
	if err != nil {
		return nil, err
	}
	span, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	// Report at the rule span with the rule span as the innermost trace
	// frame.
	return nil, &sasscommon.SassRuntimeException{
		Message: text,
		Span:    span,
		Trace:   v.stackTrace(span),
	}
}

// VisitAtRootRule evaluates @at-root: re-parents children outside the
// excluded ancestors. The query parses from interpolation with the map;
// ancestors not excluded collect innermost-first, then trimIncluded drops a
// trailing run already rooted so a no-exclusion rule just scopes in place.
// Otherwise copies of the included ancestors nest inside the root and
// scopeForAtRoot adjusts parent, style-rule, media, keyframes, and
// unknown-at-rule state for the children.
//
// Matches Dart: _EvaluateVisitor.visitAtRootRule (evaluate.dart statement
// visitors); children scope on hasDeclarations.
func (v *EvaluateVisitor) VisitAtRootRule(stmt *value.AtRootRule) (value.Value, error) {
	var query *value.AtRootQuery
	if stmt.Query != nil {
		resolved, im, err := v.performInterpolationWithMap(stmt.Query, &[]bool{true}[0])
		if err != nil {
			return nil, err
		}
		p := value.NewParser([]byte(resolved), nil, im)
		parser := &value.AtRootQueryParser{Parser: *p}
		query, err = parser.Parse()
		if err != nil {
			return nil, err
		}
	} else {
		query = value.DefaultQuery
	}

	cur := value.ModifiableCssParentNode(v.parent)
	included := make([]value.ModifiableCssParentNode, 0)
	for {
		if _, ok := any(cur).(*value.ModifiableCssStylesheet); ok {
			break
		}
		if !query.Excludes(cur) {
			included = append(included, cur)
		}
		gp := cur.Parent()
		if gp == nil {
			panic("CssNodes must have a CssStylesheet transitive parent node.")
		}
		cur = gp.(value.ModifiableCssParentNode)
	}
	root, included, trimErr := v.trimIncluded(included)
	if trimErr != nil {
		return nil, trimErr
	}

	if root == v.parent {
		_, err := sassenv.Scope(v.env, func() (struct{}, error) {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return struct{}{}, err
				}
			}
			return struct{}{}, nil
		}, false, stmt.HasDeclarations())
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	var innerCopy value.ModifiableCssParentNode = root
	if len(included) > 0 {
		first := included[0]
		rest := included[1:]
		var err error
		innerCopy, err = first.CopyWithoutChildren()
		if err != nil {
			return nil, err
		}
		outerCopy := innerCopy
		for _, node := range rest {
			copy, err := node.CopyWithoutChildren()
			if err != nil {
				return nil, err
			}
			if err := copy.AddChild(outerCopy); err != nil {
				return nil, err
			}
			outerCopy = copy
		}
		if err := root.AddChild(outerCopy); err != nil {
			return nil, err
		}
	}

	scope := v.scopeForAtRoot(stmt, innerCopy, query, included)
	if err := scope(func() error {
		for _, child := range stmt.GetChildren() {
			if _, err := v.visitStatement(child); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

// trimIncluded destructively trims a trailing ancestor run from nodes that
// matches the current parent chain. Nodes run innermost-first; when a
// trailing sublist is contiguous (each node the direct parent of the
// previous) and rooted as a direct child of the CSS root, it is removed and
// its innermost node returned — otherwise nodes stay intact and the root is
// returned.
//
// Matches Dart: _EvaluateVisitor._trimIncluded (evaluate.dart @at-root
// section); Dart's non-ancestor ArgumentError cases are unreachable here
// because the chain is built by walking real parents.
func (v *EvaluateVisitor) trimIncluded(nodes []value.ModifiableCssParentNode) (value.ModifiableCssParentNode, []value.ModifiableCssParentNode, error) {
	if len(nodes) == 0 {
		return v.root, nodes, nil
	}

	cur := value.CssParentNode(v.parent)
	var innermostContiguous int
	hasContiguous := false
	for i := range nodes {
		for cur != nodes[i] {
			hasContiguous = false
			grandparent := cur.Parent()
			if grandparent == nil {
				panic(fmt.Sprintf("Expected %v to be an ancestor of %v.", nodes[i], v.parent))
			}
			cur = grandparent
		}
		if !hasContiguous {
			innermostContiguous = i
			hasContiguous = true
		}

		grandparent := cur.Parent()
		if grandparent == nil {
			panic(fmt.Sprintf("Expected %v to be an ancestor of %v.", nodes[i], v.parent))
		}
		cur = grandparent
	}

	if cur != v.root {
		return v.root, nodes, nil
	}
	root := nodes[innermostContiguous]
	nodes = nodes[:innermostContiguous]
	return root, nodes, nil
}

// scopeForAtRoot returns a scope callback that adjusts visitor state for the
// @at-root body based on which rules query excludes, always pointing parent
// at newParent. Style-rule exclusion sets atRootExcludingStyleRule, media
// exclusion clears media queries, keyframe exclusion clears inKeyframes, and
// the unknown-at-rule flag clears only when no included parent is an at-rule.
// State restores after the callback.
//
// Matches Dart: _EvaluateVisitor._scopeForAtRoot (evaluate.dart @at-root
// section); withParent is inlined here because the node must attach at the
// precomputed root position.
func (v *EvaluateVisitor) scopeForAtRoot(node *value.AtRootRule, newParent value.ModifiableCssParentNode, query *value.AtRootQuery, included []value.ModifiableCssParentNode) func(func() error) error {
	scope := func(callback func() error) error {
		oldParent := v.parent
		v.parent = newParent
		_, err := sassenv.Scope(v.env, func() (struct{}, error) {
			if err := callback(); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, nil
		}, false, node.HasDeclarations())
		v.parent = oldParent
		return err
	}

	if query.ExcludesStyleRules() {
		innerScope := scope
		scope = func(callback func() error) error {
			oldAtRootExcludingStyleRule := v.atRootExcludingStyleRule
			v.atRootExcludingStyleRule = true
			err := innerScope(callback)
			v.atRootExcludingStyleRule = oldAtRootExcludingStyleRule
			return err
		}
	}

	if v.mediaQueries != nil && query.ExcludesName("media") {
		innerScope := scope
		scope = func(callback func() error) error {
			return v.withMediaQueries(nil, nil, func() error {
				return innerScope(callback)
			})
		}
	}

	if v.inKeyframes && query.ExcludesName("keyframes") {
		innerScope := scope
		scope = func(callback func() error) error {
			wasInKeyframes := v.inKeyframes
			v.inKeyframes = false
			err := innerScope(callback)
			v.inKeyframes = wasInKeyframes
			return err
		}
	}

	if v.inUnknownAtRule {
		hasCssAtRule := false
		for _, p := range included {
			if _, ok := p.(*value.ModifiableCssAtRule); ok {
				hasCssAtRule = true
				break
			}
		}
		if !hasCssAtRule {
			innerScope := scope
			scope = func(callback func() error) error {
				wasInUnknownAtRule := v.inUnknownAtRule
				v.inUnknownAtRule = false
				err := innerScope(callback)
				v.inUnknownAtRule = wasInUnknownAtRule
				return err
			}
		}
	}

	return scope
}

// VisitAtRule evaluates an unknown @rule: interpolates name and value
// (value trimmed, empty values dropped) and evaluates children with
// style-rule bubbling. Nested-declaration context is rejected; childless
// rules attach directly. Keyframes (after unvendoring) sets inKeyframes,
// anything else sets inUnknownAtRule, both restored after. Inside a style
// rule the rule copies inward so bare declarations have a home, except
// keyframes, font-face, and top-level children which evaluate flat;
// plain-CSS nesting short-circuits to flat scoped evaluation.
//
// Matches Dart: _EvaluateVisitor.visitAtRule (evaluate.dart statement
// visitors, mirrored in visitCssAtRule); children scope on
// hasDeclarations.
func (v *EvaluateVisitor) VisitAtRule(stmt *value.AtRule) (value.Value, error) {
	if v.declarationName != "" {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("At-rules may not be used within nested declarations.", &span)
	}

	name, err := v.interpolationToValue(stmt.Name, false, false)
	if err != nil {
		return nil, err
	}

	var cssValue *sasscommon.CssValue[string]
	if stmt.Value != nil {
		cv, err := v.interpolationToValue(stmt.Value, true, true)
		if err != nil {
			return nil, err
		}
		if cv.Value != "" {
			cssValue = &cv
		}
	}

	childless := stmt.GetChildren() == nil
	stmtSpan8, err := stmt.Span()
	if err != nil {
		return nil, err
	}

	if childless {
		if err := v.copyParentAfterSibling(); err != nil {
			return nil, err
		}
		if err := v.addChild(value.NewModifiableCssAtRule(
			name,
			stmtSpan8,
			true,
			cssValue,
		)); err != nil {
			return nil, err
		}
		return nil, nil
	}

	wasInKeyframes := v.inKeyframes
	wasInUnknownAtRule := v.inUnknownAtRule
	// Dart compares name.value directly without lowercasing
	if unvendor.Unvendor(name.Value) == "keyframes" {
		v.inKeyframes = true
	} else {
		v.inUnknownAtRule = true
	}

	rule := value.NewModifiableCssAtRule(
		name,
		stmtSpan8,
		false,
		cssValue,
	)

	if v.hasCssNesting() {
		err = v.withParent(rule, func() error {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
			return nil
		}, nil, new(stmt.HasDeclarations()))
		v.inUnknownAtRule = wasInUnknownAtRule
		v.inKeyframes = wasInKeyframes
		return nil, err
	}

	err = v.withParent(rule, func() error {
		sr := v.styleRule()
		if sr == nil || v.inKeyframes || name.Value == "font-face" {
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
		} else {
			modSr, ok := sr.(value.ModifiableCssParentNode)
			if ok {
				newParent, err := modSr.CopyWithoutChildren()
				if err != nil {
					return err
				}
				return v.withParent(newParent, func() error {
					for _, child := range stmt.GetChildren() {
						if _, err := v.visitStatement(child); err != nil {
							return err
						}
					}
					return nil
				}, nil, new(false))
			}
			for _, child := range stmt.GetChildren() {
				if _, err := v.visitStatement(child); err != nil {
					return err
				}
			}
		}
		return nil
	}, func(n value.CssNode) bool {
		_, ok := n.(value.CssStyleRule)
		return ok
	}, new(stmt.HasDeclarations()))

	v.inUnknownAtRule = wasInUnknownAtRule
	v.inKeyframes = wasInKeyframes
	return nil, err
}

// VisitExtendRule evaluates @extend: parses the target selector and records
// the extension with the current media context. Requires an enclosing style
// rule and no nested declaration name; bogus extender combinators warn,
// complex targets and multi-simple compounds fail with format errors.
//
// Matches Dart: _EvaluateVisitor.visitExtendRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitExtendRule(stmt *value.ExtendRule) (value.Value, error) {
	styleRule := v.styleRule()
	if styleRule == nil || v.declarationName != "" {
		span, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		return nil, v.exception("@extend may only be used within style rules.", &span)
	}

	for _, complex := range styleRule.OriginalSelector().Components {
		if !complex.IsBogus() {
			continue
		}
		compSpan2, err := complex.Span()
		if err != nil {
			return nil, err
		}
		stmtSpan9, err := stmt.Span()
		if err != nil {
			return nil, err
		}
		cs, err := complex.String()
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf(
			"The selector %q is invalid CSS and %s be an extender.\n"+
				"This will be an error in Dart Sass 2.0.0.\n\n"+
				"More info: https://sass-lang.com/d/bogus-combinators",
			strings.TrimSpace(cs),
			map[bool]string{true: "can't", false: "shouldn't"}[complex.IsUseless()],
		)
		trimmed, err := compSpan2.TrimRight()
		if err != nil {
			return nil, err
		}
		if err := v.warn(msg, sasscommon.NewMultiSpanFileSpan(trimmed, "invalid selector", map[sasscommon.FileSpan]string{stmtSpan9: "@extend rule"}), deprecation.BogusCombinators); err != nil {
			return nil, err
		}
	}

	wc := true
	targetText, im, err := v.performInterpolationWithMap(stmt.Selector, &wc)
	if err != nil {
		return nil, err
	}

	targetText = util.TrimAscii(targetText, false)
	list, err := value.NewSelectorParser(targetText, nil, im, &value.SelectorParserOptions{
		AllowParent:       new(false),
		Logger:            v.logger,
		WarnDeprecationFn: v.ec.WarnDeprecation,
	}).Parse()
	if err != nil {
		return nil, err
	}

	for _, complex := range list.Components {
		compound := complex.SingleCompound()
		if compound == nil {
			compSpan, err := complex.Span()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.SassFormatException{
				Message: "complex selectors may not be extended.",
				Span:    compSpan,
			}
		}

		simple := compound.SingleSimple()
		if simple == nil {
			parts := make([]string, len(compound.Components))
			for i, comp := range compound.Components {
				str, err := comp.String()
				if err != nil {
					return nil, err
				}
				parts[i] = str
			}
			cmpSpan, err := compound.Span()
			if err != nil {
				return nil, err
			}
			return nil, &sasscommon.SassFormatException{
				Message: fmt.Sprintf("compound selectors may no longer be extended.\n"+
					"Consider `@extend %s` instead.\n"+
					"See https://sass-lang.com/d/extend-compound for details.\n",
					strings.Join(parts, ", ")),
				Span: cmpSpan,
			}
		}

		if err := v.extensionStore.AddExtension(
			styleRule.Selector(),
			simple,
			stmt,
			v.mediaQueries,
		); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// VisitContentRule invokes the ambient @content block. No-op when no
// content is active; otherwise runs the content callable's children through
// runUserDefinedCallable with the raw argument list (evaluation happens
// inside).
//
// Matches Dart: _EvaluateVisitor.visitContentRule (evaluate.dart statement
// visitors).
func (v *EvaluateVisitor) VisitContentRule(stmt *value.ContentRule) (value.Value, error) {
	content := v.env.Content()
	udContent, ok := content.(*functions.UserDefinedCallable)
	if !ok || udContent == nil || udContent.Declaration() == nil {
		return nil, nil
	}

	// Pass the raw argument list; evaluation happens inside
	// runUserDefinedCallable.
	_, err := v.runUserDefinedCallable(udContent, stmt.Arguments, stmt, func() (value.Value, error) {
		for _, child := range udContent.Declaration().GetChildren() {
			if _, err := v.visitStatement(child); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return nil, err
}

// VisitContentBlock is a no-op: evaluation handles @include and its content
// block together, with registration in VisitIncludeRule and invocation in
// VisitContentRule.
//
// Matches Dart: _EvaluateVisitor.visitContentBlock, which throws
// UnsupportedError for direct evaluation (evaluate.dart statement visitors).
func (v *EvaluateVisitor) VisitContentBlock(stmt *value.ContentBlock) (value.Value, error) {
	// Content blocks are handled by VisitContentRule through the environment's
	// content field. Registration happens in VisitIncludeRule.
	return nil, nil
}

// VisitLoudComment evaluates a loud (/* */) comment: interpolates the text
// and appends it to the CSS. No-op inside functions; comments at the root
// import boundary advance endOfImports; indented-syntax text without a
// closing */ gets one appended.
//
// Matches Dart: _EvaluateVisitor.visitLoudComment (evaluate.dart statement
// visitors, mirrored in visitCssComment).
func (v *EvaluateVisitor) VisitLoudComment(stmt *value.LoudComment) (value.Value, error) {
	// Loud comments are not preserved within function arguments
	if v.inFunction {
		return nil, nil
	}

	// Comments are allowed to appear between CSS imports; track the end
	// of imports position when at root.
	if v.parent == v.root && v.endOfImports == len(v.root.Children()) {
		v.endOfImports++
	}

	text, err := v.interpolateSelector(stmt.Text)
	if err != nil {
		return nil, err
	}

	// Indented syntax doesn't require */
	if !strings.HasSuffix(text, "*/") {
		text += " */"
	}

	if err := v.copyParentAfterSibling(); err != nil {
		return nil, err
	}

	stmtSpan10, err := stmt.Span()
	if err != nil {
		return nil, err
	}
	comment := value.NewModifiableCssComment(text, stmtSpan10)
	if err := v.addChild(comment); err != nil {
		return nil, err
	}
	return nil, nil
}

// VisitSilentComment is a no-op: silent comments vanish with no CSS and no
// evaluation.
//
// Matches Dart: _EvaluateVisitor.visitSilentComment, which returns nil
// (evaluate.dart statement visitors).
func (v *EvaluateVisitor) VisitSilentComment(stmt *value.SilentComment) (value.Value, error) {
	// Silent comments are not preserved in CSS output
	return nil, nil
}
