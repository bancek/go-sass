// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/selector.dart

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalSelectorFunctions returns all globally-available selector functions.
//
// It ports Dart's `global` list: the same function objects as the module,
// with the parse/nest/append/extend/replace/unify entries renamed to their
// `selector-*` global names and every entry carrying a `selector`
// deprecation warning.
func GlobalSelectorFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		isSuperselectorFunction().WithDeprecationWarning("selector", nil),
		simpleSelectorsFunction().WithDeprecationWarning("selector", nil),
		selectorParseFunction().WithDeprecationWarning("selector", new("parse")),
		selectorNestFunction().WithDeprecationWarning("selector", new("nest")),
		selectorAppendFunction().WithDeprecationWarning("selector", new("append")),
		selectorExtendFunction().WithDeprecationWarning("selector", new("extend")),
		selectorReplaceFunction().WithDeprecationWarning("selector", new("replace")),
		selectorUnifyFunction().WithDeprecationWarning("selector", new("unify")),
	}
}

// SelectorModule returns the sass:selector built-in module.
//
// It ports Dart's `module` (`BuiltInModule("selector", ...)`): the eight
// callables under their short module names (parse, nest, append, extend,
// replace, unify plus is-superselector and simple-selectors).
func SelectorModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		isSuperselectorFunction(),
		simpleSelectorsFunction(),
		selectorParseModuleFunction(),
		selectorNestModuleFunction(),
		selectorAppendModuleFunction(),
		selectorExtendModuleFunction(),
		selectorReplaceModuleFunction(),
		selectorUnifyModuleFunction(),
	}
	return sassmodule.NewBuiltInModule("selector", fns, nil, nil)
}

// --- Helpers ---

// selectorListValue reports whether v has the list shape a selector argument
// allows. It accepts SassList and SassArgumentList alike, mirroring Dart's
// `is! SassList` check: there SassArgumentList extends SassList, so rest
// arguments pass the check too.
func selectorListValue(v value.Value) (value.Value, bool) {
	switch v.(type) {
	case *value.SassList, *value.SassArgumentList:
		return v, true
	}
	return nil, false
}

// selectorStringOrNil converts a SassScript value to the selector text it
// denotes, following Dart's Value._selectorStringOrNull. A SassString yields
// its text directly; a comma-separated list joins one entry per complex
// selector with ", ", any other separator joins simple-selector strings with
// " ". Nested lists must be space-separated, and slash-separated lists are
// never valid selectors. It returns ("", false) when the value has none of
// the accepted shapes (string, list of strings, list of lists of strings).
func selectorStringOrNil(v value.Value) (string, bool, error) {
	if s, ok := v.(*value.SassString); ok {
		return s.Text, true, nil
	} else if val, ok := selectorListValue(v); ok {
		items, err := val.AsList()
		if err != nil {
			return "", false, err
		}
		if len(items) == 0 {
			return "", false, nil
		}
		var parts []string
		switch val.Separator() {
		case value.ListSeparatorComma:
			for _, item := range items {
				if s, ok := item.(*value.SassString); ok {
					parts = append(parts, s.Text)
				} else if lst, ok := selectorListValue(item); ok {
					if lst.Separator() != value.ListSeparatorSpace {
						return "", false, nil
					}
					str, ok, err := selectorStringOrNil(lst)
					if err != nil {
						return "", false, err
					}
					if !ok {
						return "", false, nil
					}
					parts = append(parts, str)
				} else {
					return "", false, nil
				}
			}
		case value.ListSeparatorSlash:
			return "", false, nil
		default:
			for _, item := range items {
				if s, ok := item.(*value.SassString); ok {
					parts = append(parts, s.Text)
				} else {
					return "", false, nil
				}
			}
		}
		sep := " "
		if val.Separator() == value.ListSeparatorComma {
			sep = ", "
		}
		return strings.Join(parts, sep), true, nil
	}
	return "", false, nil
}

// selectorString converts a SassScript value to selector text ready for
// parsing. It wraps selectorStringOrNil: values without a valid selector
// shape fail with "$name is not a valid selector: it must be a string, a
// list of strings, or a list of lists of strings."
func selectorString(v value.Value, name string) (string, error) {
	str, ok, err := selectorStringOrNil(v)
	if err != nil {
		return "", err
	}
	if ok {
		return str, nil
	}
	vStr, err := v.String()
	if err != nil {
		return "", err
	}
	return "", sasscommon.NewSassScriptException(fmt.Sprintf("%s is not a valid selector: it must be a string,\na list of strings, or a list of lists of strings.", vStr), &name)
}

// assertSelector parses a SassScript value as a SelectorList, allowing parent
// selectors (&) only when allowParent is set (selector-nest passes true).
// Parse failures surface as SassScriptExceptions attributed to name, carrying
// just the format message — the span belongs to the caller.
func assertSelector(v value.Value, name string, allowParent bool, ec *evalcontext.EvaluationContext) (*value.SelectorList, error) {
	str, err := selectorString(v, name)
	if err != nil {
		return nil, err
	}
	parser := value.NewSelectorParser(str, nil, nil, &value.SelectorParserOptions{
		AllowParent:       &allowParent,
		Logger:            ec.Logger,
		WarnDeprecationFn: ec.WarnDeprecation,
	})
	result, err := parser.Parse()
	if err != nil {
		var msg string
		if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
			msg = sfe.Message
		} else {
			msg = err.Error()
		}
		return nil, &sasscommon.SassScriptException{Message: msg, ArgumentName: name}
	}
	return result, nil
}

// assertCompoundSelector parses a SassScript value and requires it to be a
// single compound selector, following Dart's assertCompoundSelector. Lists
// with several complexes, or a complex with several compounds, fail with
// "Selector ... must be a compound selector."
func assertCompoundSelector(v value.Value, name string, ec *evalcontext.EvaluationContext) (*value.CompoundSelector, error) {
	sel, err := assertSelector(v, name, false, ec)
	if err != nil {
		return nil, err
	}
	if len(sel.Components) > 1 {
		s, err := selectorListToString(sel)
		if err != nil {
			return nil, err
		}
		return nil, &sasscommon.SassScriptException{
			Message:      fmt.Sprintf("Selector %s must be a compound selector.", s),
			ArgumentName: name,
		}
	}
	compound := sel.Components[0].SingleCompound()
	if compound == nil {
		s, err := selectorListToString(sel)
		if err != nil {
			return nil, err
		}
		return nil, &sasscommon.SassScriptException{
			Message:      fmt.Sprintf("Selector %s must be a compound selector.", s),
			ArgumentName: name,
		}
	}
	return compound, nil
}

// selectorListToString renders a SelectorList as comma-separated text for
// error messages. Go-only helper with no Dart counterpart; the complexes
// already carry their own rendering.
func selectorListToString(sl *value.SelectorList) (string, error) {
	parts := make([]string, len(sl.Components))
	for i, c := range sl.Components {
		ss, err := c.String()
		if err != nil {
			return "", err
		}
		parts[i] = ss
	}
	return strings.Join(parts, ", "), nil
}

// prependParent adds a ParentSelector to the beginning of compound, returning
// nil where that would be invalid. A leading universal selector can never
// take a parent; a namespaced type selector can neither, while a plain type
// selector folds its name into the parent as a suffix. Anything else gets a
// bare parent prepended. The span labels the new nodes and comes from the
// current callable span, as in Dart's _prependParent.
func prependParent(compound *value.CompoundSelector, span sasscommon.FileSpan) (*value.CompoundSelector, error) {
	first := compound.Components[0]
	switch s := first.(type) {
	case *value.UniversalSelector:
		return nil, nil
	case *value.TypeSelector:
		if s.Name.Namespace != nil {
			return nil, nil
		}
		rest := compound.Components[1:]
		comps := make([]value.SimpleSelector, 0, 1+len(rest))
		comps = append(comps, value.NewParentSelector(span, &s.Name.Name))
		comps = append(comps, rest...)
		return value.NewCompoundSelector(comps, span)
	default:
		comps := make([]value.SimpleSelector, 0, 1+len(compound.Components))
		comps = append(comps, value.NewParentSelector(span, nil))
		comps = append(comps, compound.Components...)
		return value.NewCompoundSelector(comps, span)
	}
}

// --- Shared implementation closures ---
//
// Dart defines each selector function once and shares the object between the
// module and the renamed global; Go instead shares one implementation closure
// per function (parseImpl, nestImpl, ...) with thin module/global
// constructors around it.

// isSuperselectorFunction builds the is-superselector($super, $sub) callable.
// Both sides are parsed without parent selectors and rejected when bogus
// before delegating to SelectorList.IsSuperselector.
func isSuperselectorFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("is-superselector", "$super, $sub", "sass:selector", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		sel1, err := assertSelector(args[0], "super", false, ec)
		if err != nil {
			return nil, err
		}
		superName := "super"
		if err := sel1.AssertNotBogus(&superName, ec); err != nil {
			return nil, err
		}
		sel2, err := assertSelector(args[1], "sub", false, ec)
		if err != nil {
			return nil, err
		}
		subName := "sub"
		if err := sel2.AssertNotBogus(&subName, ec); err != nil {
			return nil, err
		}
		ok, err := sel1.IsSuperselector(sel2)
		if err != nil {
			return nil, err
		}
		if ok {
			return value.SassTrue, nil
		}
		return value.SassFalse, nil
	})
}

// simpleSelectorsFunction builds the simple-selectors($selector) callable.
// The argument must be a single compound selector; each simple selector is
// returned as an unquoted string in a comma-separated list.
func simpleSelectorsFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("simple-selectors", "$selector", "sass:selector", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		compound, err := assertCompoundSelector(args[0], "selector", ec)
		if err != nil {
			return nil, err
		}
		items := make([]value.Value, len(compound.Components))
		for i, s := range compound.Components {
			str, err := s.String()
			if err != nil {
				return nil, err
			}
			items[i] = &value.SassString{Text: str, HasQuotes: false}
		}
		list, err := value.NewSassList(items, value.ListSeparatorComma, false)
		if err != nil {
			return nil, err
		}
		return list, nil
	})
}

// parseImpl parses $selector (no parent selectors) and returns its Sass
// list form. It backs both the `parse` module function and the
// `selector-parse` global.
var parseImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	sel, err := assertSelector(args[0], "selector", false, ec)
	if err != nil {
		return nil, err
	}
	return sel.AsSassList()
}

// selectorParseFunction builds the `selector-parse` global name for parseImpl.
func selectorParseFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-parse", "$selector", "sass:selector", parseImpl)
}

// selectorParseModuleFunction builds the `parse` module name for parseImpl.
func selectorParseModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("parse", "$selector", "sass:selector", parseImpl)
}

// nestImpl implements nest($selectors...): each rest argument is parsed with
// parent selectors allowed, then folded left through NestWithin starting
// from nil. An empty rest list fails with "$selectors: At least one selector
// must be passed." It backs both the `nest` module function and the
// `selector-nest` global.
var nestImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	selectors, err := args[0].AsList()
	if err != nil {
		return nil, err
	}
	if len(selectors) == 0 {
		return nil, &sasscommon.SassScriptException{Message: "$selectors: At least one selector must be passed."}
	}
	var result *value.SelectorList
	for _, v := range selectors {
		sel, err := assertSelector(v, "", true, ec)
		if err != nil {
			return nil, err
		}
		result, err = sel.NestWithin(result, true, false)
		if err != nil {
			return nil, err
		}
	}
	return result.AsSassList()
}

// selectorNestFunction builds the `selector-nest` global name for nestImpl.
func selectorNestFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-nest", "$selectors...", "sass:selector", nestImpl)
}

// selectorNestModuleFunction builds the `nest` module name for nestImpl.
func selectorNestModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("nest", "$selectors...", "sass:selector", nestImpl)
}

// appendImpl implements append($selectors...): the first selector is the
// parent into which each following child is nested. Every child's first
// compound gains a parent via prependParent (reusing the current callable
// span); a child with leading combinators, or one prependParent rejects,
// fails with "Can't append ... to ...". It backs both the `append` module
// function and the `selector-append` global.
var appendImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	selectors, err := args[0].AsList()
	if err != nil {
		return nil, err
	}
	if len(selectors) == 0 {
		return nil, &sasscommon.SassScriptException{Message: "$selectors: At least one selector must be passed."}
	}
	parsed := make([]*value.SelectorList, len(selectors))
	for i, v := range selectors {
		sel, err := assertSelector(v, "", false, ec)
		if err != nil {
			return nil, err
		}
		parsed[i] = sel
	}
	span, err := ec.CurrentCallableSpan()
	if err != nil {
		return nil, err
	}
	result := parsed[0]
	for i := 1; i < len(parsed); i++ {
		child := parsed[i]
		var newComponents []*value.ComplexSelector
		for _, complex := range child.Components {
			if len(complex.LeadingCombinators) > 0 {
				s, err := selectorListToString(result)
				if err != nil {
					return nil, err
				}
				complexStr, err := complex.String()
				if err != nil {
					return nil, err
				}
				return nil, &sasscommon.SassScriptException{
					Message: fmt.Sprintf("Can't append %s to %s.", complexStr, s),
				}
			}
			firstComp := complex.Components[0]
			var rest []*value.ComplexSelectorComponent
			if len(complex.Components) > 1 {
				rest = complex.Components[1:]
			}
			newCompound, err := prependParent(firstComp.Selector, span)
			if err != nil {
				return nil, err
			}
			if newCompound == nil {
				s, err := selectorListToString(result)
				if err != nil {
					return nil, err
				}
				complexStr, err := complex.String()
				if err != nil {
					return nil, err
				}
				return nil, &sasscommon.SassScriptException{
					Message: fmt.Sprintf("Can't append %s to %s.", complexStr, s),
				}
			}
			newComps := []*value.ComplexSelectorComponent{
				value.NewComplexSelectorComponent(newCompound, firstComp.Combinators, span),
			}
			newComps = append(newComps, rest...)
			cs, err := value.NewComplexSelector(nil, newComps, span, false)
			if err != nil {
				return nil, err
			}
			newComponents = append(newComponents, cs)
		}
		childList, err := value.NewSelectorList(newComponents, span)
		if err != nil {
			return nil, err
		}
		result, err = childList.NestWithin(result, true, false)
		if err != nil {
			return nil, err
		}
	}
	return result.AsSassList()
}

// selectorAppendFunction builds the `selector-append` global name for appendImpl.
func selectorAppendFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-append", "$selectors...", "sass:selector", appendImpl)
}

// selectorAppendModuleFunction builds the `append` module name for appendImpl.
func selectorAppendModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("append", "$selectors...", "sass:selector", appendImpl)
}

// extendImpl implements extend($selector, $extendee, $extender): all three
// sides are parsed and rejected when bogus, then run through the static
// extension store with the current callable span. It backs both the `extend`
// module function and the `selector-extend` global.
var extendImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	sel, err := assertSelector(args[0], "selector", false, ec)
	if err != nil {
		return nil, err
	}
	selName := "selector"
	if err := sel.AssertNotBogus(&selName, ec); err != nil {
		return nil, err
	}
	target, err := assertSelector(args[1], "extendee", false, ec)
	if err != nil {
		return nil, err
	}
	targetName := "extendee"
	if err := target.AssertNotBogus(&targetName, ec); err != nil {
		return nil, err
	}
	source, err := assertSelector(args[2], "extender", false, ec)
	if err != nil {
		return nil, err
	}
	sourceName := "extender"
	if err := source.AssertNotBogus(&sourceName, ec); err != nil {
		return nil, err
	}
	currentCallableSpan, err := ec.CurrentCallableSpan()
	if err != nil {
		return nil, err
	}
	result, err := extend.ExtendStatic(sel, source, target, currentCallableSpan)
	if err != nil {
		return nil, err
	}
	return result.AsSassList()
}

// selectorExtendFunction builds the `selector-extend` global name for extendImpl.
func selectorExtendFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-extend", "$selector, $extendee, $extender", "sass:selector", extendImpl)
}

// selectorExtendModuleFunction builds the `extend` module name for extendImpl.
func selectorExtendModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("extend", "$selector, $extendee, $extender", "sass:selector", extendImpl)
}

// replaceImpl implements replace($selector, $original, $replacement):
// like extendImpl but through the store's static replace. It backs both the
// `replace` module function and the `selector-replace` global.
var replaceImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	sel, err := assertSelector(args[0], "selector", false, ec)
	if err != nil {
		return nil, err
	}
	selName := "selector"
	if err := sel.AssertNotBogus(&selName, ec); err != nil {
		return nil, err
	}
	target, err := assertSelector(args[1], "original", false, ec)
	if err != nil {
		return nil, err
	}
	targetName := "original"
	if err := target.AssertNotBogus(&targetName, ec); err != nil {
		return nil, err
	}
	source, err := assertSelector(args[2], "replacement", false, ec)
	if err != nil {
		return nil, err
	}
	sourceName := "replacement"
	if err := source.AssertNotBogus(&sourceName, ec); err != nil {
		return nil, err
	}
	currentCallableSpan, err := ec.CurrentCallableSpan()
	if err != nil {
		return nil, err
	}
	result, err := extend.ReplaceStatic(sel, source, target, currentCallableSpan)
	if err != nil {
		return nil, err
	}
	return result.AsSassList()
}

// selectorReplaceFunction builds the `selector-replace` global name for replaceImpl.
func selectorReplaceFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-replace", "$selector, $original, $replacement", "sass:selector", replaceImpl)
}

// selectorReplaceModuleFunction builds the `replace` module name for replaceImpl.
func selectorReplaceModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("replace", "$selector, $original, $replacement", "sass:selector", replaceImpl)
}

// unifyImpl implements unify($selector1, $selector2): both sides are parsed
// and rejected when bogus, then unified; a nil unification returns Sass
// null. It backs both the `unify` module function and the `selector-unify`
// global.
var unifyImpl = func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
	sel1, err := assertSelector(args[0], "selector1", false, ec)
	if err != nil {
		return nil, err
	}
	sel1Name := "selector1"
	if err := sel1.AssertNotBogus(&sel1Name, ec); err != nil {
		return nil, err
	}
	sel2, err := assertSelector(args[1], "selector2", false, ec)
	if err != nil {
		return nil, err
	}
	sel2Name := "selector2"
	if err := sel2.AssertNotBogus(&sel2Name, ec); err != nil {
		return nil, err
	}
	result, err := sel1.Unify(sel2)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return value.Null, nil
	}
	return result.AsSassList()
}

// selectorUnifyFunction builds the `selector-unify` global name for unifyImpl.
func selectorUnifyFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("selector-unify", "$selector1, $selector2", "sass:selector", unifyImpl)
}

// selectorUnifyModuleFunction builds the `unify` module name for unifyImpl.
func selectorUnifyModuleFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("unify", "$selector1, $selector2", "sass:selector", unifyImpl)
}
