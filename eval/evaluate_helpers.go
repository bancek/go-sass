// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/visitor/evaluate.dart (helper sections: _serialize,
// _expressionNode, _addRestMap, _exception/_multiSpanException, _stackTrace /
// _stackFrame, _withStackFrame/_withEnvironment, _addExceptionSpan /
// _addErrorSpan/_addExceptionTrace, _evaluateArguments /
// _evaluateMacroArguments, _verifyArguments, _runUserDefinedCallable,
// _withParent/_addChild/_copyParentAfterSibling, _withStyleRule /
// _withMediaQueries/_mergeMediaQueries, _hasCssNesting/_styleRule,
// _performInterpolation*)

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/linkedhashmap"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassenv"
	"github.com/bancek/go-sass/sassurl"
	"github.com/bancek/go-sass/util"
	"github.com/bancek/go-sass/value"
)

// argumentResults holds the evaluated values of a function argument list.
//
// Matches Dart: the _ArgumentResults record returned by _evaluateArguments.
// positionalNodes and namedNodes keep the resolved declaration-site nodes
// (not prebuilt spans) so slash-division warnings can point at the
// declaration while spans that are expensive to manufacture are only built
// when needed. separator carries the rest list's separator, undecided when
// no rest argument contributed one, for rest-argument SassArgumentList
// assembly.
type argumentResults struct {
	positional      []value.Value
	positionalNodes []sasscommon.AstNode
	named           *orderedmap.LinkedMap[string, value.Value]
	namedNodes      map[string]sasscommon.AstNode
	separator       value.ListSeparator
}

// serialize calls val's CSS serialization and wraps a script failure so it
// reports nodeWithSpan's source span with the current stack trace.
//
// Matches Dart: _serialize. Already-spanned errors pass through unchanged.
// quote selects quoted output. Takes the node rather than a prebuilt span so
// spans that are expensive to manufacture are only built when an error needs
// one.
// Matches Dart: _serialize
func (v *EvaluateVisitor) serialize(val value.Value, nodeWithSpan sasscommon.AstNode, quote bool) (string, error) {
	return addExceptionSpan(v, nodeWithSpan, func() (string, error) {
		return val.ToCssString(quote)
	}, nil)
}

// expressionNode returns the node whose span should back warnings for
// expression.
//
// Matches Dart: _expressionNode. A variable reference resolves through the
// environment to the variable's declaration node, so slash-division warnings
// point at the declaration rather than the use. Anything else keeps its own
// node. Falls back to expression itself when the variable is undefined (the
// later lookup reports it). Span tracking is unconditional, kept for the
// slash-division warnings. Takes and returns nodes rather than spans so
// expensive span manufacture is deferred until a span is actually needed.
// Matches Dart: _expressionNode
func (v *EvaluateVisitor) expressionNode(expression sasscommon.AstNode) (sasscommon.AstNode, error) {
	if ve, ok := expression.(*value.VariableExpression); ok {
		// Look the declaration node up under an exception span so a failure
		// resolving it still reports the use site, then fall back to the
		// expression itself when there is no declaration node.
		// Dart: _addExceptionSpan(expression, () => getVariableNode(...)) ?? expression
		node, err := addExceptionSpan(v, expression, func() (sasscommon.AstNode, error) {
			return v.env.GetVariableNode(ve.Name(), ve.Namespace())
		}, nil)
		if err != nil {
			return nil, err
		}
		if node != nil {
			return node, nil
		}
	}
	return expression, nil
}

// addRestMap merges the entries of map m into values, converting each value
// with convert.
//
// Matches Dart: _addRestMap. String keys insert slash-stripped at the
// resolved declaration-site node (used for the deprecation warning span). A
// non-string key returns a runtime error at nodeWithSpan's own span naming
// the offending key and map. Takes a node rather than a span so the span is
// only manufactured when the error path needs it.
// Matches Dart: _addRestMap
func addRestMap[T any](v *EvaluateVisitor, values *orderedmap.LinkedMap[string, T], m *value.SassMap, nodeWithSpan sasscommon.AstNode, convert func(value.Value) T) error {
	exprNode, err := v.expressionNode(nodeWithSpan)
	if err != nil {
		return err
	}
	for key, val := range m.Entries() {
		if s, ok := key.(*value.SassString); ok {
			cleanedVal, err := v.withoutSlash(val, exprNode)
			if err != nil {
				return err
			}
			values.Put(s.Text, convert(cleanedVal))
		} else {
			sp, err := nodeWithSpan.Span()
			if err != nil {
				return err
			}
			keyStr, err := key.String()
			if err != nil {
				return err
			}
			mStr, err := m.String()
			if err != nil {
				return err
			}
			return v.exception(
				fmt.Sprintf("Variable keyword argument map must have string keys.\n%s is not a string in %s.", keyStr, mStr),
				&sp,
			)
		}
	}
	return nil
}

// exception builds a spanned runtime error with the current trace.
//
// Matches Dart: _exception. Uses span when non-nil, else the top stack
// frame's span. The trace is built from the original (possibly nil) span
// argument, not the resolved one, so an error raised inside a stack frame
// does not duplicate the innermost frame.
// Matches Dart: _exception
func (v *EvaluateVisitor) exception(message string, span *sasscommon.FileSpan) *sasscommon.SassRuntimeException {
	var s sasscommon.FileSpan
	if span != nil {
		s = *span
	} else if v.stack != nil {
		// Fall back to the top stack frame's span when the caller has none.
		// Dart: span ?? _stack.last.$2.span
		s = v.stack.Span
	}
	// Pass the original (possibly nil) span, not the resolved one, so the
	// innermost stack frame isn't duplicated when the error occurs inside a
	// stack frame. Dart: _stackTrace(span).
	var traceSpan sasscommon.FileSpan
	if span != nil {
		traceSpan = *span
	}
	return &sasscommon.SassRuntimeException{
		Message: message,
		Span:    s,
		Trace:   v.stackTrace(traceSpan),
	}
}

// performInterpolation evaluates interpolation contents to plain text.
//
// Matches Dart: _performInterpolation, the string-only twin of
// performInterpolationWithMap. Both delegate to performInterpolationHelper
// with source-map tracking off. warnForColor enables the named-color warning
// for color values used in interpolation.
// Matches Dart: _performInterpolation
func (v *EvaluateVisitor) performInterpolation(in *value.Interpolation, warnForColor *bool) (string, error) {
	wc := false
	if warnForColor != nil {
		wc = *warnForColor
	}
	result, _, err := v.performInterpolationHelper(in, false, wc)
	return result, err
}

// performInterpolationWithMap evaluates interpolation contents like
// performInterpolation but additionally records per-part target offsets and
// returns the InterpolationMap that maps spans from the resulting string back
// to the original interpolation, so callers can locate errors at the
// interpolation site.
//
// Matches Dart: _performInterpolationWithMap.
// Matches Dart: _performInterpolationWithMap
func (v *EvaluateVisitor) performInterpolationWithMap(in *value.Interpolation, warnForColor *bool) (string, *value.InterpolationMap, error) {
	wc := false
	if warnForColor != nil {
		wc = *warnForColor
	}
	return v.performInterpolationHelper(in, true, wc)
}

// performInterpolationHelper implements the shared core of
// performInterpolation and performInterpolationWithMap.
//
// Matches Dart: _performInterpolationHelper. Clears inSupportsDeclaration
// while running (calculations must not simplify inside interpolation) and
// restores it afterwards. Literal string parts copy through verbatim;
// expression parts evaluate and then serialize unquoted. When sourceMap is
// set, the target offset of every non-first part is recorded for the
// InterpolationMap. When warnForColor is set, each named color value warns
// with a quoted-name alternative before serialization.
// Matches Dart: _performInterpolationHelper
func (v *EvaluateVisitor) performInterpolationHelper(in *value.Interpolation, sourceMap bool, warnForColor bool) (string, *value.InterpolationMap, error) {
	// A nil slice means offset tracking is off; only non-first parts record
	// offsets, matching Dart's targetOffsets?.add placement.
	var targetOffsets []int
	if sourceMap {
		targetOffsets = []int{}
	}

	// Calculations must not simplify inside interpolation; restored by defer
	// so error returns keep the flag consistent.
	oldInSupportsDeclaration := v.inSupportsDeclaration
	v.inSupportsDeclaration = false
	defer func() { v.inSupportsDeclaration = oldInSupportsDeclaration }()

	var buf strings.Builder
	first := true
	for _, part := range in.Contents {
		if !first && targetOffsets != nil {
			targetOffsets = append(targetOffsets, buf.Len())
		}
		first = false

		if s, ok := part.(string); ok {
			buf.WriteString(s)
			continue
		}

		expr := part.(value.Expression)
		val, err := v.eval(expr)
		if err != nil {
			return "", nil, err
		}

		if warnForColor {
			// Warn when a named color flows into interpolation unquoted: it
			// may serialize to a form that produces invalid CSS. Suggest the
			// quoted name, or "" + <source> to keep the color value.
			if c, ok := val.(*value.SassColor); ok {
				r, err := c.Red()
				if err != nil {
					return "", nil, err
				}
				g, err := c.Green()
				if err != nil {
					return "", nil, err
				}
				b, err := c.Blue()
				if err != nil {
					return "", nil, err
				}
				if name := value.ColorNameForInts(int(math.Round(r)), int(math.Round(g)), int(math.Round(b))); name != "" {
					s, err := expr.Span()
					if err != nil {
						return "", nil, err
					}
					st, err := s.SpanText()
					if err != nil {
						return "", nil, err
					}
					valStr, err := val.ToCssString(false)
					if err != nil {
						return "", nil, err
					}
					alternative := fmt.Sprintf(`"" + %s`, st)
					if err := v.warn(
						fmt.Sprintf(
							"You probably don't mean to use the color value "+
								"%s in interpolation here.\n"+
								"It may end up represented as %s, which will likely produce "+
								"invalid CSS.\n"+
								"Always quote color names when using them as strings or map keys "+
								`(for example, "%s").`+"\n"+
								"If you really want to use the color value here, use '%s'.",
							name, valStr, name, alternative,
						),
						s,
						nil,
					); err != nil {
						return "", nil, err
					}
				}
			}
		}

		result, err := v.serialize(val, expr, false)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString(result)
	}

	var im *value.InterpolationMap
	if sourceMap && targetOffsets != nil {
		var err error
		im, err = value.NewInterpolationMap(in, targetOffsets)
		if err != nil {
			return "", nil, err
		}
	}

	return buf.String(), im, nil
}

// evaluateArguments evaluates every argument of a function invocation.
//
// Matches Dart: _evaluateArguments. Each positional and named value is
// evaluated, then slash-stripped at its declaration-site span. Rest-argument
// dispatch on the evaluated rest value: a map merges into named via addRestMap,
// an argument list contributes positional values plus keywords and preserves
// its separator, a plain list contributes positional values and preserves its
// separator, and any other value becomes a single positional. (Dart tests the
// map arm first and then a single list arm with an argument-list sub-case;
// Go spells the argument-list arm out explicitly with identical behavior.)
// Keyword-rest must evaluate to a map and reports the use-site span otherwise.
// Span tracking is unconditional, kept for the slash-division warnings.
// Matches Dart: _evaluateArguments
func (v *EvaluateVisitor) evaluateArguments(args *value.ArgumentList) (*argumentResults, error) {
	positional := make([]value.Value, 0, len(args.Positional))
	positionalNodes := make([]sasscommon.AstNode, 0, len(args.Positional))
	for _, expr := range args.Positional {
		nodeForSpan, nserr := v.expressionNode(expr)
		if nserr != nil {
			return nil, nserr
		}
		val, err := v.eval(expr)
		if err != nil {
			return nil, err
		}
		cleaned, err := v.withoutSlash(val, nodeForSpan)
		if err != nil {
			return nil, err
		}
		positional = append(positional, cleaned)
		positionalNodes = append(positionalNodes, nodeForSpan)
	}

	named := orderedmap.NewWithCapacity[string, value.Value](args.Named.Len())
	namedNodes := make(map[string]sasscommon.AstNode, args.Named.Len())
	for name, expr := range args.Named.Entries() {
		nodeForSpan, nserr := v.expressionNode(expr)
		if nserr != nil {
			return nil, nserr
		}
		val, err := v.eval(expr)
		if err != nil {
			return nil, err
		}
		cleaned, err := v.withoutSlash(val, nodeForSpan)
		if err != nil {
			return nil, err
		}
		named.Put(name, cleaned)
		namedNodes[name] = nodeForSpan
	}

	separator := value.ListSeparatorUndecided

	if args.Rest != nil {
		rest, err := v.eval(args.Rest)
		if err != nil {
			return nil, err
		}
		restNodeForSpan, nserr := v.expressionNode(args.Rest)
		if nserr != nil {
			return nil, nserr
		}

		if m, ok := rest.(*value.SassMap); ok {
			// A map rest becomes keyword arguments; each key also records the
			// rest node for span purposes.
			if err := addRestMap(v, named, m, args.Rest, func(val value.Value) value.Value { return val }); err != nil {
				return nil, err
			}
			for key := range m.Entries() {
				if s, ok := key.(*value.SassString); ok {
					namedNodes[s.Text] = restNodeForSpan
				}
			}
		} else if argList, ok := rest.(*value.SassArgumentList); ok {
			// An argument-list rest splays into positional values plus its
			// keywords, keeping its separator for the eventual rest parameter.
			argItems, err := argList.AsList()
			if err != nil {
				return nil, err
			}
			for _, val := range argItems {
				cleaned, err := v.withoutSlash(val, restNodeForSpan)
				if err != nil {
					return nil, err
				}
				positional = append(positional, cleaned)
				positionalNodes = append(positionalNodes, restNodeForSpan)
			}
			separator = argList.Separator()
			for key, val := range argList.Keywords().Entries() {
				cleaned, err := v.withoutSlash(val, restNodeForSpan)
				if err != nil {
					return nil, err
				}
				named.Put(key, cleaned)
				namedNodes[key] = restNodeForSpan
			}
		} else if lst, ok := rest.(*value.SassList); ok {
			// A plain list rest splays into positional values, keeping its
			// separator; anything else is a single positional value.
			lstItems, err := lst.AsList()
			if err != nil {
				return nil, err
			}
			for _, val := range lstItems {
				cleaned, err := v.withoutSlash(val, restNodeForSpan)
				if err != nil {
					return nil, err
				}
				positional = append(positional, cleaned)
				positionalNodes = append(positionalNodes, restNodeForSpan)
			}
			separator = lst.Separator()
		} else {
			cleaned, err := v.withoutSlash(rest, restNodeForSpan)
			if err != nil {
				return nil, err
			}
			positional = append(positional, cleaned)
			positionalNodes = append(positionalNodes, restNodeForSpan)
		}
	}

	if args.KeywordRest != nil {
		kwRest, err := v.eval(args.KeywordRest)
		if err != nil {
			return nil, err
		}
		kwRestNodeForSpan, nserr := v.expressionNode(args.KeywordRest)
		if nserr != nil {
			return nil, nserr
		}

		if kwMap, ok := kwRest.(*value.SassMap); ok {
			if err := addRestMap(v, named, kwMap, args.KeywordRest, func(val value.Value) value.Value { return val }); err != nil {
				return nil, err
			}
			for key := range kwMap.Entries() {
				if s, ok := key.(*value.SassString); ok {
					namedNodes[s.Text] = kwRestNodeForSpan
				}
			}
		} else {
			// A non-map keyword-rest reports the use-site span (the raw
			// keyword-rest argument span), not the resolved value node.
			sp, err := kwRestNodeForSpan.Span()
			if err != nil {
				return nil, err
			}
			kwRestStr, err := kwRest.String()
			if err != nil {
				return nil, err
			}
			return nil, v.exception(
				fmt.Sprintf("Variable keyword arguments must be a map (was %s).", kwRestStr),
				&sp,
			)
		}
	}

	return &argumentResults{
		positional:      positional,
		positionalNodes: positionalNodes,
		named:           named,
		namedNodes:      namedNodes,
		separator:       separator,
	}, nil
}

// evaluateMacroArguments separates a macro invocation's arguments only as far
// as needed to tell positional from named, keeping them lazy.
//
// Matches Dart: _evaluateMacroArguments. Positional and named pass through as
// unevaluated expressions so macros such as if() can evaluate branches
// lazily; only the evaluated rest value is re-wrapped in ValueExpression
// nodes at the raw rest-argument span, slash-stripped at the resolved
// declaration-site node. Rest dispatch mirrors evaluateArguments (map into
// named, argument list into positional plus keywords, list into positional,
// other into a single positional), except map-rest errors report nodeWithSpan
// (the whole invocation) rather than the rest-argument span, and a non-map
// keyword-rest reports its own span.
// Matches Dart: _evaluateMacroArguments
func (v *EvaluateVisitor) evaluateMacroArguments(args *value.ArgumentList, nodeWithSpan sasscommon.AstNode) ([]value.Expression, *orderedmap.LinkedMap[string, value.Expression], error) {
	// Matches Dart: var restArgs_ = invocation.arguments.rest;
	//               if (restArgs_ == null) { return (invocation.arguments.positional, invocation.arguments.named); }
	restArgs_ := args.Rest
	if restArgs_ == nil {
		return args.Positional, args.Named, nil
	}

	// Matches Dart: var restArgs = restArgs_;
	restArgs := restArgs_

	// Matches Dart: var positional = invocation.arguments.positional.toList();
	//               var named = Map.of(invocation.arguments.named);
	positional := make([]value.Expression, len(args.Positional))
	copy(positional, args.Positional)
	named := orderedmap.New[string, value.Expression]()
	for name, e := range args.Named.Entries() {
		named.Put(name, e)
	}

	// Matches Dart: var rest = restArgs.accept(this);
	rest, err := v.eval(restArgs)
	if err != nil {
		return nil, nil, err
	}
	// Matches Dart: var restNodeForSpan = _expressionNode(restArgs);
	restNodeForSpan, nserr := v.expressionNode(restArgs)
	if nserr != nil {
		return nil, nil, nserr
	}

	restArgsSpan, err := restArgs.Span()
	if err != nil {
		return nil, nil, err
	}

	// Matches Dart: if (rest is SassMap) { _addRestMap(named, rest, invocation, (value) => ValueExpression(value, restArgs.span)); }
	if m, ok := rest.(*value.SassMap); ok {
		if err := addRestMap(v, named, m, nodeWithSpan, func(val value.Value) value.Expression {
			return value.NewValueExpression(val, restArgsSpan)
		}); err != nil {
			return nil, nil, err
		}
	} else if argList, ok := rest.(*value.SassArgumentList); ok {
		// Matches Dart: positional.addAll(rest.asList.map((value) => ValueExpression(_withoutSlash(value, restNodeForSpan), restArgs.span)));
		argItems, err := argList.AsList()
		if err != nil {
			return nil, nil, err
		}
		for _, val := range argItems {
			cleaned, err := v.withoutSlash(val, restNodeForSpan)
			if err != nil {
				return nil, nil, err
			}
			positional = append(positional, value.NewValueExpression(cleaned, restArgsSpan))
		}
		// Matches Dart: if (rest is SassArgumentList) { rest.keywords.forEach(...) }
		for key, val := range argList.Keywords().Entries() {
			cleaned, err := v.withoutSlash(val, restNodeForSpan)
			if err != nil {
				return nil, nil, err
			}
			named.Put(key, value.NewValueExpression(cleaned, restArgsSpan))
		}
	} else if lst, ok := rest.(*value.SassList); ok {
		// Matches Dart: positional.addAll(rest.asList.map((value) => ValueExpression(_withoutSlash(value, restNodeForSpan), restArgs.span)));
		lstItems, err := lst.AsList()
		if err != nil {
			return nil, nil, err
		}
		for _, val := range lstItems {
			cleaned, err := v.withoutSlash(val, restNodeForSpan)
			if err != nil {
				return nil, nil, err
			}
			positional = append(positional, value.NewValueExpression(cleaned, restArgsSpan))
		}
	} else {
		// Matches Dart: positional.add(ValueExpression(_withoutSlash(rest, restNodeForSpan), restArgs.span));
		cleaned, err := v.withoutSlash(rest, restNodeForSpan)
		if err != nil {
			return nil, nil, err
		}
		positional = append(positional, value.NewValueExpression(cleaned, restArgsSpan))
	}

	// Matches Dart: var keywordRestArgs_ = invocation.arguments.keywordRest;
	//               if (keywordRestArgs_ == null) return (positional, named);
	keywordRestArgs_ := args.KeywordRest
	if keywordRestArgs_ == nil {
		return positional, named, nil
	}
	// Matches Dart: var keywordRestArgs = keywordRestArgs_;
	keywordRestArgs := keywordRestArgs_

	// Matches Dart: var keywordRest = keywordRestArgs.accept(this);
	keywordRest, err := v.eval(keywordRestArgs)
	if err != nil {
		return nil, nil, err
	}
	// Matches Dart: var keywordRestNodeForSpan = _expressionNode(keywordRestArgs);
	keywordRestNodeForSpan, nserr := v.expressionNode(keywordRestArgs)
	if nserr != nil {
		return nil, nil, nserr
	}

	keywordRestArgsSpan, err := keywordRestArgs.Span()
	if err != nil {
		return nil, nil, err
	}

	// Matches Dart: if (keywordRest is SassMap) { _addRestMap(named, keywordRest, invocation, (value) => ValueExpression(value, keywordRestArgs.span)); }
	if kwMap, ok := keywordRest.(*value.SassMap); ok {
		if err := addRestMap(v, named, kwMap, nodeWithSpan, func(val value.Value) value.Expression {
			// The convert callback slash-strips a second time because
			// addRestMap already stripped once: this matches Dart, where the
			// keyword-rest convert wraps _withoutSlash around a value that
			// _addRestMap also strips.
			cleaned, err := v.withoutSlash(val, keywordRestNodeForSpan)
			if err != nil {
				panic(err)
			}
			return value.NewValueExpression(cleaned, keywordRestArgsSpan)
		}); err != nil {
			return nil, nil, err
		}
	} else {
		// Matches Dart: throw _exception(...)
		sp, err := keywordRestNodeForSpan.Span()
		if err != nil {
			return nil, nil, err
		}
		keywordRestStr, err := keywordRest.String()
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, v.exception(
			fmt.Sprintf("Variable keyword arguments must be a map (was %s).", keywordRestStr),
			&sp,
		)
	}

	return positional, named, nil
}

// --- Helpers ---

// addChild appends node to the current parent.
//
// Matches Dart: _addChild without a through predicate.
// Matches Dart: _addChild
func (v *EvaluateVisitor) addChild(node value.ModifiableCssNode) error {
	return v.parent.AddChild(node)
}

// addChildThrough appends node to the current parent, or — when through is
// given — to the first ancestor for which through returns false.
//
// Matches Dart: _addChild with a through predicate. Walks up past parents
// matching through, then, when that parent has a following sibling, reuses an
// already-made childless copy of it or clones one via CopyWithoutChildren so
// the sibling is not corrupted (the bubbling path). Panics when through never
// returns false; Dart throws ArgumentError for the same contract violation.
// Matches Dart: _addChild (with through parameter)
func (v *EvaluateVisitor) addChildThrough(node value.ModifiableCssNode, through func(value.CssNode) bool) error {
	parent := value.CssParentNode(v.parent)
	if through != nil {
		for through(parent) {
			gp := parent.Parent()
			if gp == nil {
				panic(fmt.Sprintf("through() must return false for at least one parent of %v.", node))
			}
			parent = gp
		}

		modParent, ok := parent.(value.ModifiableCssParentNode)
		if !ok {
			return v.parent.AddChild(node)
		}

		if modParent.HasFollowingSibling() {
			grandparent := modParent.Parent()
			if gp, ok := grandparent.(value.ModifiableCssParentNode); ok {
				if last, ok := gp.LastChild().(value.ModifiableCssParentNode); ok {
					equal, err := modParent.EqualsIgnoringChildren(last)
					if err != nil {
						return err
					}
					if equal {
						modParent = last
					} else {
						newParent, err := modParent.CopyWithoutChildren()
						if err != nil {
							return err
						}
						if err := gp.AddChild(newParent); err != nil {
							return err
						}
						modParent = newParent
					}
				}
			}
		}

		return modParent.AddChild(node)
	}

	return v.parent.AddChild(node)
}

// withParent attaches node as a child of the current parent (bubbling through
// matching parents when through is given), then runs callback with node as
// the current parent.
//
// Matches Dart: _withParent. Runs callback in a new environment scope unless
// scopeWhen is false (a nil scopeWhen means true). Restores the old parent
// afterwards even when callback fails.
// Matches Dart: _withParent
func (v *EvaluateVisitor) withParent(node value.ModifiableCssParentNode, callback func() error, through func(value.CssNode) bool, scopeWhen *bool) error {
	if err := v.addChildThrough(node, through); err != nil {
		return err
	}

	sw := true
	if scopeWhen != nil {
		sw = *scopeWhen
	}

	oldParent := v.parent
	v.parent = node
	_, err := sassenv.Scope(v.env, func() (struct{}, error) {
		if err := callback(); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, false, sw)
	v.parent = oldParent
	return err
}

// copyParentAfterSibling keeps later children from corrupting a parent that
// already has a following sibling.
//
// Matches Dart: _copyParentAfterSibling. When the current parent is not the
// last child of its own parent (for example a declaration wrote CSS into a
// rule that a later bubbled rule already left), replaces it with a childless
// copy appended to the grandparent so subsequent children land in a fresh
// node. A no-op without a grandparent or when already last.
// Mirrors Dart's _copyParentAfterSibling.
func (v *EvaluateVisitor) copyParentAfterSibling() error {
	grandparent := v.parent.Parent()
	if grandparent == nil {
		return nil
	}
	gp, ok := grandparent.(value.ModifiableCssParentNode)
	if !ok {
		return nil
	}
	if gp.ChildrenLen() == 0 {
		return nil
	}
	if gp.LastChild() != v.parent {
		newParent, err := v.parent.CopyWithoutChildren()
		if err != nil {
			return err
		}
		if err := gp.AddChild(newParent); err != nil {
			return err
		}
		v.parent = newParent
	}
	return nil
}

// WithStackFrame runs callback with member installed as the current member
// and a frame pairing the previous member name with nodeWithSpan's span
// pushed on the stack, then restores both.
//
// Matches Dart: _withStackFrame. The pushed frame carries the caller's member
// name (not the new one) together with the new call-site span. Takes the node
// rather than a span so expensive span manufacture is deferred; a span failure
// surfaces as a Go error before anything is pushed. Restoration runs on both
// success and error returns.
// Matches Dart: T _withStackFrame<T>(String member, AstNode nodeWithSpan, T callback())
func WithStackFrame[T any](v *EvaluateVisitor, member string, nodeWithSpan sasscommon.AstNode, callback func() (T, error)) (T, error) {
	sp, err := nodeWithSpan.Span()
	if err != nil {
		var zero T
		return zero, err
	}
	v.stack = &stackFrame{
		Name:   v.member,
		Span:   sp,
		Parent: v.stack,
	}
	oldMember := v.member
	v.member = member
	result, err := callback()
	v.member = oldMember
	v.stack = v.stack.Parent
	return result, err
}

// WithEnvironment runs callback with env as the current environment, then
// restores the previous one on both success and error returns.
//
// Matches Dart: _withEnvironment. Callers pass a callable's closure
// environment so mixin and content bodies see definition-site bindings.
// Matches Dart's T _withEnvironment<T>(Environment, T callback()).
func WithEnvironment[T any](v *EvaluateVisitor, env *sassenv.Environment, callback func() (T, error)) (T, error) {
	oldEnv := v.env
	v.env = env
	result, err := callback()
	v.env = oldEnv
	return result, err
}

// addExceptionSpan runs callback and converts spanless script failures into
// spanned runtime errors at nodeWithSpan's span.
//
// Matches Dart: _addExceptionSpan. A SassScriptException becomes a
// SassRuntimeException carrying the node's span and the current trace; a
// MultiSpanSassScriptException becomes a MultiSpanSassRuntimeException that
// preserves its secondary spans. Already-spanned runtime errors and non-Sass
// errors pass through unchanged. There is deliberately no helper that wraps
// an already-built error — every call site wraps the throwing callback
// itself. A nil addStackFrame defaults to true: with true the trace gains an
// innermost frame at the node's span, with false the existing stack stands
// as-is. Takes the node rather than a span so expensive span manufacture only
// happens on failure.
// Matches Dart: _addExceptionSpan
func addExceptionSpan[T any](v *EvaluateVisitor, nodeWithSpan sasscommon.AstNode, callback func() (T, error), addStackFrame *bool) (T, error) {
	stackFrame := true
	if addStackFrame != nil {
		stackFrame = *addStackFrame
	}
	result, err := callback()
	if err == nil {
		return result, nil
	}
	callErr := err
	nodeSpan, err := nodeWithSpan.Span()
	if err != nil {
		var zero T
		return zero, err
	}

	if _, ok := callErr.(*sasscommon.SassRuntimeException); ok {
		return result, callErr
	}
	if _, ok := callErr.(*sasscommon.MultiSpanSassRuntimeException); ok {
		return result, callErr
	}

	if sse, ok := errors.AsType[*sasscommon.SassScriptException](callErr); ok {
		var trace *sasscommon.Trace
		if stackFrame {
			trace = v.stackTrace(nodeSpan)
		} else {
			trace = v.stackTrace(nil)
		}
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.SassRuntimeException{
				Message: sse.Error(),
				Span:    nodeSpan,
				Trace:   trace,
			},
			sse,
		)
	}
	if msse, ok := errors.AsType[*sasscommon.MultiSpanSassScriptException](callErr); ok {
		var trace *sasscommon.Trace
		if stackFrame {
			trace = v.stackTrace(nodeSpan)
		} else {
			trace = v.stackTrace(nil)
		}
		// Preserve the multi-span shape: the error keeps its secondary spans
		// and primary label, gaining the node's span and the current trace.
		// Dart: error.withSpan(nodeWithSpan.span).withTrace(...).
		secondarySpans := make(map[sasscommon.FileSpan]string)
		for span, label := range msse.SecondarySpans {
			if span != nil {
				secondarySpans[span] = label
			}
		}
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.MultiSpanSassRuntimeException{
				Message:        msse.Message,
				Span:           nodeSpan,
				PrimaryLabel:   msse.PrimaryLabel,
				SecondarySpans: secondarySpans,
				Trace:          trace,
			},
			msse,
		)
	}
	return result, callErr
}

// addErrorSpan runs callback and re-spans @error failures at the call site.
//
// Matches Dart: _addErrorSpan. When callback returns a runtime error whose
// span text starts with "@error", it is rethrown with nodeWithSpan's span and
// a fresh current trace, so an @error deep inside a callable reports the
// call site rather than the @error rule. Any other error passes through
// unchanged. Applied at function-call and mixin-include sites.
// Matches Dart: _addErrorSpan
func addErrorSpan[T any](v *EvaluateVisitor, nodeWithSpan sasscommon.AstNode, callback func() (T, error)) (T, error) {
	result, err := callback()
	if err == nil {
		return result, nil
	}
	if sre, ok := errors.AsType[*sasscommon.SassRuntimeException](err); ok {
		sreSpanText, spanErr := sre.Span.SpanText()
		if spanErr != nil {
			var zero T
			return zero, err
		}
		if strings.HasPrefix(sreSpanText, "@error") {
			nodeSpan, nodeErr := nodeWithSpan.Span()
			if nodeErr != nil {
				var zero T
				return zero, err
			}
			return result, sasscommon.ThrowWithTrace(
				&sasscommon.SassRuntimeException{
					Message: sre.Message,
					Span:    nodeSpan,
					Trace:   v.stackTrace(nil),
				},
				sre,
			)
		}
	}
	return result, err
}

// addExceptionTrace runs callback and attaches the current stack trace to any
// spanned Sass failure that does not already carry one.
//
// Matches Dart: _addExceptionTrace. Runtime errors (single- and multi-span)
// pass through unchanged; other Sass failures (plain, format, and multi-span
// without a trace) are rethrown as runtime errors whose trace is built from
// the error's own span as the innermost frame plus the current stack.
// Non-Sass errors pass through. Callers wrap the load call while its frame
// is on the stack so traces point at the load site.
// Matches Dart: _addExceptionTrace
func addExceptionTrace[T any](v *EvaluateVisitor, callback func() (T, error)) (T, error) {
	result, err := callback()
	if err == nil {
		return result, nil
	}
	if _, ok := errors.AsType[*sasscommon.SassRuntimeException](err); ok {
		return result, err
	}
	if _, ok := errors.AsType[*sasscommon.MultiSpanSassRuntimeException](err); ok {
		return result, err
	}
	if se, ok := errors.AsType[*sasscommon.SassException](err); ok {
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.SassRuntimeException{
				Message:    se.Message,
				Span:       se.Span,
				Trace:      v.stackTrace(se.Span),
				LoadedUrls: se.LoadedUrls,
			},
			se,
		)
	}
	if mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.MultiSpanSassRuntimeException{
				Message:        mse.Message,
				Span:           mse.Span,
				PrimaryLabel:   mse.PrimaryLabel,
				SecondarySpans: mse.Secondary,
				Trace:          v.stackTrace(mse.Span),
				LoadedUrls:     mse.LoadedUrls,
			},
			mse,
		)
	}
	if sfe, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.SassRuntimeException{
				Message:    sfe.Message,
				Span:       sfe.Span,
				Trace:      v.stackTrace(sfe.Span),
				LoadedUrls: sfe.LoadedUrls,
			},
			sfe,
		)
	}
	if msfe, ok := errors.AsType[*sasscommon.MultiSpanSassFormatException](err); ok {
		return result, sasscommon.ThrowWithTrace(
			&sasscommon.SassRuntimeException{
				Message:    msfe.Message,
				Span:       msfe.Span,
				Trace:      v.stackTrace(msfe.Span),
				LoadedUrls: msfe.LoadedUrls,
			},
			msfe,
		)
	}
	return result, err
}

// verifyArguments checks that the evaluated positional count and named keys
// are valid for params, reporting failures at nodeWithSpan's span.
//
// Matches Dart: _verifyArguments. Delegates the checks to verifyParameterList,
// which emits spanless script failures; the addExceptionSpan wrapper converts
// them to spanned runtime errors with the current trace at the call site.
// named maps each passed name to whether it was passed (always true here).
// Matches Dart: _verifyArguments
func (v *EvaluateVisitor) verifyArguments(positional int, named map[string]bool, params *value.ParameterList, nodeWithSpan sasscommon.AstNode) error {
	_, err := addExceptionSpan(v, nodeWithSpan, func() (struct{}, error) {
		return struct{}{}, verifyParameterList(positional, named, params)
	}, nil)
	return err
}

// verifyParameterList checks that a positional count and named keys are valid
// for params, returning a spanless script failure for the caller to span.
//
// Matches Dart: ParameterList.verify. An argument passed both by position and
// by name fails; a missing required argument fails as a multi-span error with
// the declaration's span labeled "declaration" against the invocation. A rest
// parameter accepts any arity. Over-arity says "positional " when named
// arguments are present and otherwise counts plain arguments; unknown names
// are listed $-prefixed and joined with "or". Messages use the parameters'
// source spellings. A nil params expects zero arguments.
// Matches Dart: ParameterList.verify
func verifyParameterList(positional int, names map[string]bool, params *value.ParameterList) error {
	if params == nil {
		if positional > 0 || len(names) > 0 {
			return sasscommon.NewSassScriptException(
				fmt.Sprintf("Expected 0 arguments, got %d.", positional),
				nil)
		}
		return nil
	}

	paramsSpan, err := params.Span()
	if err != nil {
		return err
	}

	namedUsed := 0
	for i, param := range params.Parameters {
		if i < positional {
			if names[param.Name()] {
				return sasscommon.NewSassScriptException(
					fmt.Sprintf("Argument $%s was passed both by position and by name.", param.Name()),
					nil)
			}
		} else if names[param.Name()] {
			namedUsed++
		} else if param.DefaultValue == nil {
			return &sasscommon.MultiSpanSassScriptException{
				Message:      fmt.Sprintf("Missing argument $%s.", param.Name()),
				PrimaryLabel: "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{
					paramsSpan: "declaration",
				},
			}
		}
	}

	if params.RestParameter != nil {
		return nil
	}

	if positional > len(params.Parameters) {
		plural := "argument"
		if len(params.Parameters) != 1 {
			plural = "arguments"
		}
		wasPlural := "was"
		if positional != 1 {
			wasPlural = "were"
		}
		positionalPrefix := ""
		if len(names) > 0 {
			positionalPrefix = "positional "
		}
		return &sasscommon.MultiSpanSassScriptException{
			Message:      fmt.Sprintf("Only %d %s%s allowed, but %d %s passed.", len(params.Parameters), positionalPrefix, plural, positional, wasPlural),
			PrimaryLabel: "invocation",
			SecondarySpans: map[sasscommon.FileSpan]string{
				paramsSpan: "declaration",
			},
		}
	}

	if namedUsed < len(names) {
		unknownNames := make([]string, 0)
		for name := range names {
			found := false
			for _, param := range params.Parameters {
				if param.Name() == name {
					found = true
					break
				}
			}
			if !found {
				unknownNames = append(unknownNames, name)
			}
		}

		if len(unknownNames) > 0 {
			parameterWord := util.Pluralize("parameter", len(unknownNames), nil)

			dollarNames := make([]string, len(unknownNames))
			for i, name := range unknownNames {
				dollarNames[i] = "$" + name
			}

			var parameterNames string
			switch len(dollarNames) {
			case 1:
				parameterNames = dollarNames[0]
			case 2:
				parameterNames = dollarNames[0] + " or " + dollarNames[1]
			default:
				parameterNames = strings.Join(dollarNames[:len(dollarNames)-1], ", ") + ", or " + dollarNames[len(dollarNames)-1]
			}

			return &sasscommon.MultiSpanSassScriptException{
				Message:      fmt.Sprintf("No %s named %s.", parameterWord, parameterNames),
				PrimaryLabel: "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{
					paramsSpan: "declaration",
				},
			}
		}
	}

	return nil
}

// stackTrace builds the trace for the current point, innermost frame first.
//
// Matches Dart: _stackTrace. When span is non-nil it becomes the innermost
// frame under the current member; a nil span contributes no extra frame. The
// stack is a linked list from innermost to outermost, so frames are collected
// innermost-first, reversed to outermost-first, extended with the span frame,
// and reversed again — yielding [span frame, innermost, ..., outermost].
// Every frame's URL passes through humanizeFrame.
// Matches Dart: _stackTrace([FileSpan? span])
func (v *EvaluateVisitor) stackTrace(span sasscommon.FileSpan) *sasscommon.Trace {
	// The stack is a linked list from innermost to outermost (Dart appends,
	// so its first element is outermost).
	// Step 1: Collect stack frames
	var stackFrames []sasscommon.Frame
	for f := v.stack; f != nil; f = f.Parent {
		frame, err := sasscommon.FrameForSpan(f.Span, f.Name)
		if err != nil {
			panic(err)
		}
		v.humanizeFrame(&frame)
		stackFrames = append(stackFrames, frame)
	}
	// Step 2: Reverse to outermost→innermost (matches Dart's iteration order
	// over its append-grown stack)
	slices.Reverse(stackFrames)
	// Step 3: Append span frame at end (Dart: if (span != null) _stackFrame(_member, span))
	if span != nil {
		frame, err := sasscommon.FrameForSpan(span, v.member)
		if err != nil {
			panic(err)
		}
		v.humanizeFrame(&frame)
		stackFrames = append(stackFrames, frame)
	}
	// Step 4: Reverse to [span_frame, innermost, ..., outermost]
	slices.Reverse(stackFrames)
	return sasscommon.NewTrace(stackFrames)
}

// humanizeFrame rewrites frame's URL through the import cache's humanized
// form, leaving it unchanged when there is no cache or URI or the humanized
// form does not parse.
//
// Matches Dart: the URL humanization inside _stackFrame, which maps the span
// URL through importCache.humanize. Split out so stackTrace can apply it per
// frame.
// Matches Dart: _stackFrame URL humanization via _importCache.
func (v *EvaluateVisitor) humanizeFrame(frame *sasscommon.Frame) {
	if frame.URI == nil || v.importCache == nil {
		return
	}
	humanized, err := sassurl.Parse(v.importCache.Humanize(frame.URI))
	if err == nil {
		frame.URI = humanized
	}
}

// loadedUrlsList parses the loaded canonical URL strings back to URLs for
// error payloads and result population, silently dropping unparseable ones.
//
// Matches Dart: the _loadedUrls set of every stylesheet canonical URL seen
// during compilation.
func (v *EvaluateVisitor) loadedUrlsList() []*url.URL {
	urls := make([]*url.URL, 0, v.loadedUrls.Len())
	for u := range v.loadedUrls.Keys() {
		parsed, err := sassurl.Parse(u)
		if err == nil {
			urls = append(urls, parsed)
		}
	}
	return urls
}

// defaultNamespace derives the default @use namespace from u: the URL path's
// basename without its extension, with underscores replaced by hyphens.
func defaultNamespace(u *url.URL) string {
	base := filepath.Base(u.Path)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return strings.ReplaceAll(name, "_", "-")
}

// bindArguments binds evaluated positional and keyword arguments to params,
// setting local variables in the current scope, and returns the SassArgumentList
// built for the rest parameter, or nil when there is none.
//
// Matches Dart: the binding half of _runUserDefinedCallable. Positional values
// bind in order using their declaration-site nodes. Remaining parameters fall
// back to named, then to evaluated slash-stripped defaults (stripped at the
// default's declaration-site node), then fail with a missing-argument
// multi-span error. Excess positionals plus leftover named pack into a
// SassArgumentList under the rest parameter — an undecided separator becomes
// comma — whose keyword-accessed flag the caller checks for unused keywords.
// A nil params with any arguments panics; callers verify arity first.
// Matches Dart: argument binding inside _runUserDefinedCallable
func (v *EvaluateVisitor) bindArguments(params *value.ParameterList, decl value.CallableDeclaration, args []value.Value, kwargs *orderedmap.LinkedMap[string, value.Value], positionalNodes []sasscommon.AstNode, namedNodes map[string]sasscommon.AstNode, separator value.ListSeparator) (*value.SassArgumentList, error) {
	if params == nil {
		if len(args) > 0 || kwargs.Len() > 0 {
			panic("no parameters defined but arguments provided.")
		}
		return nil, nil
	}

	paramsSpan, err := params.Span()
	if err != nil {
		return nil, err
	}

	parameters := params.Parameters
	minLength := min(len(args), len(parameters))

	// Bind positional arguments by index, each under its declaration-site node.
	// Dart: _environment.setLocalVariable(parameters[i].name, evaluated.positional[i], evaluated.positionalNodes[i])
	for i := range minLength {
		var node sasscommon.AstNode = decl
		if i < len(positionalNodes) {
			node = positionalNodes[i]
		}
		v.env.SetLocalVariable(parameters[i].Name(), args[i], node)
	}

	// Bind named arguments and defaults for remaining parameters: named wins,
	// then the evaluated slash-stripped default, then a missing-argument error.
	for i := len(args); i < len(parameters); i++ {
		param := parameters[i]
		if val, ok := kwargs.Get(param.Name()); ok {
			kwargs.Delete(param.Name())
			v.env.SetLocalVariable(param.Name(), val, decl)
		} else if param.DefaultValue != nil {
			val, err := v.eval(param.DefaultValue)
			if err != nil {
				return nil, err
			}
			// Slash-strip the default at its declaration-site node.
			// Dart: _withoutSlash(parameter.defaultValue!.accept(this), ...)
			node, nserr := v.expressionNode(param.DefaultValue)
			if nserr != nil {
				return nil, nserr
			}
			cleaned, err := v.withoutSlash(val, node)
			if err != nil {
				return nil, err
			}
			v.env.SetLocalVariable(param.Name(), cleaned, decl)
		} else {
			return nil, &sasscommon.MultiSpanSassScriptException{
				Message:      fmt.Sprintf("Missing argument $%s.", param.Name()),
				PrimaryLabel: "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{
					paramsSpan: "declaration",
				},
			}
		}
	}

	// Pack excess positionals plus leftover named into the rest parameter.
	if params.RestParameter != nil {
		var rest []value.Value
		if len(args) > len(parameters) {
			rest = args[len(parameters):]
		}
		// An undecided separator becomes comma.
		// Dart: SassArgumentList(rest, evaluated.named, separator == undecided ? comma : separator)
		sep := separator
		if sep == value.ListSeparatorUndecided {
			sep = value.ListSeparatorComma
		}
		argList, err := value.NewSassArgumentList(rest, kwargs, sep)
		if err != nil {
			return nil, err
		}
		v.env.SetLocalVariable(*params.RestParameter, argList, decl)
		return argList, nil
	}

	return nil, nil
}

// --- CSS nesting and media query helpers ---

// styleRule returns the current style rule, or nil while an @at-root rule
// excludes style rules.
//
// Matches Dart: get _styleRule. The raw styleRuleIgnoringAtRoot field
// deliberately ignores intermediate @at-root rules; this accessor is the one
// callers use in the common case where exclusion matters (selector
// resolution, declaration checks).
func (v *EvaluateVisitor) styleRule() value.CssStyleRule {
	if v.atRootExcludingStyleRule {
		return nil
	}
	return v.styleRuleIgnoringAtRoot
}

// hasCssNesting reports whether the current position in the output stylesheet
// uses plain CSS nesting.
//
// Matches Dart: get _hasCssNesting. Walks up from styleRule — never the raw
// ignoring-at-root field, so this is false while @at-root excludes style
// rules — and reports true on hitting an enclosing style rule. True means the
// user has already opted into native nesting, so other nesting-spec features
// are safe to use.
func (v *EvaluateVisitor) hasCssNesting() bool {
	sr := v.styleRule()
	if sr == nil {
		return false
	}
	parent := sr.Parent()
	for parent != nil {
		switch parent.(type) {
		case value.CssStyleRule:
			return true
		}
		parent = parent.Parent()
	}
	return false
}

// mergeMediaQueries cross-joins two media query sets, merging compatible pairs.
//
// Matches Dart: _mergeMediaQueries. An empty merge result for a pair skips
// just that pair; a single unrepresentable pair poisons the whole result to
// nil, in which case the caller falls back to the node's own queries. Note
// nil (unrepresentable) is distinct from an empty non-nil list (everything
// merged away, which emits nothing).
func (v *EvaluateVisitor) mergeMediaQueries(queries1, queries2 []*value.CssMediaQuery) []*value.CssMediaQuery {
	queries := []*value.CssMediaQuery{}
	for _, q1 := range queries1 {
		for _, q2 := range queries2 {
			switch result := q1.Merge(q2).(type) {
			case *value.MediaQuerySuccessfulMergeResult:
				queries = append(queries, result.Query)
			case value.MediaQueryMergeResult:
				if result == value.MediaQueryMergeResultEmpty {
					continue
				}
				if result == value.MediaQueryMergeResultUnrepresentable {
					return nil
				}
			}
		}
	}
	return queries
}

// withMediaQueries runs callback with queries and sources as the current
// media-query context, restoring the old ones afterwards on both success and
// error returns.
//
// Matches Dart: _withMediaQueries. sources is the set of queries merged to
// produce queries (empty when queries are not a merge result); the bubbling
// predicate uses it to decide when one query may pass through another.
// queries arrives as values and is stored as pointers.
// Matches Dart: _withMediaQueries
func (v *EvaluateVisitor) withMediaQueries(
	queries []value.CssMediaQuery,
	sources *linkedhashmap.LinkedHashSet[*value.CssMediaQuery],
	callback func() error,
) error {
	oldQueries := v.mediaQueries
	oldSources := v.mediaQuerySources
	v.mediaQueries = make([]*value.CssMediaQuery, len(queries))
	for i := range queries {
		v.mediaQueries[i] = &queries[i]
	}
	v.mediaQuerySources = sources
	err := callback()
	v.mediaQueries = oldQueries
	v.mediaQuerySources = oldSources
	return err
}

// queriesToPtrs adapts a media-query value slice to the pointer slice the
// visitor stores. Go-only glue with no Dart counterpart.
func queriesToPtrs(queries []value.CssMediaQuery) []*value.CssMediaQuery {
	ptrs := make([]*value.CssMediaQuery, len(queries))
	for i := range queries {
		ptrs[i] = &queries[i]
	}
	return ptrs
}

// queriesFromPtrs adapts the stored media-query pointer slice back to values
// for rule construction. Go-only glue with no Dart counterpart.
func queriesFromPtrs(ptrs []*value.CssMediaQuery) []value.CssMediaQuery {
	vals := make([]value.CssMediaQuery, len(ptrs))
	for i, p := range ptrs {
		vals[i] = *p
	}
	return vals
}

// runUserDefinedCallable evaluates arguments and invokes a user-defined
// callable's body in a new stack frame, closure environment, and scope; run
// holds the body-specific logic.
//
// Matches Dart: _runUserDefinedCallable. Evaluates arguments, then nests:
// stack frame (the callable name with a "()" suffix, except "@content" which
// keeps its bare name) → closure environment (so bodies see definition-site
// bindings plus an extra scope level that shields the closure from
// modification) → arity verification inside the frame (so the trace includes
// it) → parameter binding → run. inDependency swaps to the callable's for the
// duration and is restored after. After run, leftover named arguments with an
// un-accessed rest parameter raise unusedKeywordsError. Only functions require
// a value: mixins and content blocks return nil, while a function that
// finishes without @return raises at the declaration's span.
// Matches Dart: _runUserDefinedCallable
// Matches Dart: _runUserDefinedCallable(ArgumentList arguments, UserDefinedCallable callable, AstNode nodeWithSpan, ...)
func (v *EvaluateVisitor) runUserDefinedCallable(
	c *functions.UserDefinedCallable,
	arguments *value.ArgumentList,
	nodeWithSpan sasscommon.AstNode,
	run func() (value.Value, error),
) (value.Value, error) {
	// Evaluate arguments before pushing the frame, so argument errors report
	// the call site rather than inside the callable.
	// Dart: var evaluated = _evaluateArguments(arguments)
	results, err := v.evaluateArguments(arguments)
	if err != nil {
		return nil, err
	}
	positional := results.positional
	named := results.named
	positionalNodes := results.positionalNodes
	namedNodes := results.namedNodes
	separator := results.separator

	name := c.Name()
	if name != "@content" {
		name += "()"
	}

	oldInDependency := v.inDependency
	v.inDependency = c.InDependency()
	defer func() { v.inDependency = oldInDependency }()

	callableNode := v.ec.CallableNode()
	if callableNode == nil {
		callableNode = nodeWithSpan
	}
	return WithStackFrame(v, name, callableNode, func() (value.Value, error) {
		return WithEnvironment(v, c.Env().Closure(), func() (value.Value, error) {
			return sassenv.Scope(v.env, func() (value.Value, error) {
				namedKeys := make(map[string]bool)
				for k := range named.Keys() {
					namedKeys[k] = true
				}
				if err := v.verifyArguments(len(positional), namedKeys, c.Arguments(), callableNode); err != nil {
					return nil, err
				}

				argList, err := v.bindArguments(c.Arguments(), c.Declaration(), positional, named, positionalNodes, namedNodes, separator)
				if err != nil {
					return nil, err
				}

				result, err := run()
				if err != nil {
					return result, err
				}

				// After the body, leftover named arguments with an un-accessed
				// rest parameter fail: the call passed keywords the callable
				// never consumed.
				// Dart: if argumentList != null && evaluated.named.isNotEmpty && !argumentList.wereKeywordsAccessed
				if argList != nil && named.Len() > 0 && !argList.WereKeywordsAccessed() {
					return nil, v.unusedKeywordsError(named, callableNode, c.Declaration().Parameters())
				}

				// Only functions require a @return value: a nil result from a
				// mixin or content block is success, while a function that
				// finishes without @return raises at its declaration span.
				// Dart throws this only from _runFunctionCallable, never for
				// mixin/content.
				if _, isFn := c.Declaration().(*value.FunctionRule); !isFn {
					return nil, nil
				}
				if result != nil {
					return result, nil
				}
				// A spanned runtime error (not script): the declaration span and
				// trace must survive outer wrappers.
				// Dart: throw _exception("Function finished without @return.", callable.declaration.span)
				declSpan, err := c.Declaration().Span()
				if err != nil {
					return nil, err
				}
				return nil, &sasscommon.SassRuntimeException{
					Message: "Function finished without @return.",
					Span:    declSpan,
					Trace:   v.stackTrace(nil),
				}
			}, false, true)
		})
	})
}

// unusedKeywordsError builds the multi-span error for keyword arguments left
// over after a call whose rest parameter never observed them.
//
// Matches Dart: the trailing check in _runUserDefinedCallable. Reports "No
// parameter(s) named $..." with "invocation" on the call-site span and
// "declaration" on the parameter list's named span, traced at the invocation.
// kwargs keys are sorted for deterministic output.
// Matches Dart: MultiSpanSassRuntimeException throw in _runUserDefinedCallable.
func (v *EvaluateVisitor) unusedKeywordsError(kwargs *orderedmap.LinkedMap[string, value.Value], nodeWithSpan sasscommon.AstNode, params *value.ParameterList) error {
	nwsSpan, err := nodeWithSpan.Span()
	if err != nil {
		return err
	}

	var names []string
	for name := range kwargs.Keys() {
		names = append(names, "$"+name)
	}
	sort.Strings(names)

	parameterWord := util.Pluralize("parameter", len(names), nil)

	var parameterNames string
	switch len(names) {
	case 1:
		parameterNames = names[0]
	case 2:
		parameterNames = names[0] + " or " + names[1]
	default:
		parameterNames = strings.Join(names[:len(names)-1], ", ") + ", or " + names[len(names)-1]
	}

	// Label the parameter list's named span as the declaration side, mirroring
	// Dart's {callable.declaration.parameters.spanWithName: "declaration"}.
	return &sasscommon.MultiSpanSassRuntimeException{
		Message:      fmt.Sprintf("No %s named %s.", parameterWord, parameterNames),
		Span:         nwsSpan,
		PrimaryLabel: "invocation",
		SecondarySpans: map[sasscommon.FileSpan]string{
			params.SpanWithName(): "declaration",
		},
		Trace: v.stackTrace(nwsSpan),
	}
}
