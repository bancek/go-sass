// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/list.dart

import (
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

// GlobalListFunctions returns the deprecated global aliases of the sass:list
// functions. Each entry wraps the module function with a deprecation warning
// (the wrapper names the original module function — see
// docs/ref/functions.md for the ordering rule); the separator global
// additionally renames to its list-separator spelling.
// Matches Dart: global (list.dart).
func GlobalListFunctions() []sasscallable.Callable {
	return []sasscallable.Callable{
		lengthFunction().WithDeprecationWarning("list", nil),
		nthFunction().WithDeprecationWarning("list", nil),
		setNthFunction().WithDeprecationWarning("list", nil),
		joinFunction().WithDeprecationWarning("list", nil),
		appendFunction().WithDeprecationWarning("list", nil),
		zipFunction().WithDeprecationWarning("list", nil),
		indexFunction().WithDeprecationWarning("list", nil),
		isBracketedFunction().WithDeprecationWarning("list", nil),
		separatorFunction().WithDeprecationWarning("list", nil).WithName("list-separator"),
	}
}

// ListModule returns the sass:list built-in module: length, nth, set-nth,
// join, append, zip, index, is-bracketed, separator, and the module-only
// slash.
// Matches Dart: module (list.dart).
func ListModule() *sassmodule.BuiltInModule {
	fns := []sasscallable.Callable{
		lengthFunction(),
		nthFunction(),
		setNthFunction(),
		joinFunction(),
		appendFunction(),
		zipFunction(),
		indexFunction(),
		isBracketedFunction(),
		separatorFunction(),
		slashFunction(),
	}
	return sassmodule.NewBuiltInModule("list", fns, nil, nil)
}

// The constructors below port Dart's bare `_function(...)` closures, which
// carry no per-function docs; each note names the Sass signature. Every
// callable registers under the sass:list URL, porting Dart's _function URL
// helper.

// lengthFunction implements list.length($list): the element count as a
// unitless number.
func lengthFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("length", "$list", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		return value.NewUnitlessNumber(float64(args[0].LengthAsList())), nil
	})
}

// nthFunction implements list.nth($list, $n): the element at the 1-based Sass
// index (negative counts back from the end).
func nthFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("nth", "$list, $n", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list := args[0]
		index, err := value.SassIndexToListIndex(list, args[1], "n", ec.WarnDeprecation)
		if err != nil {
			return nil, err
		}
		l, err := list.AsList()
		if err != nil {
			return nil, err
		}
		return l[index], nil
	})
}

// setNthFunction implements list.set-nth($list, $n, $value): a copy of the
// list with the element at the Sass index replaced.
func setNthFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("set-nth", "$list, $n, $value", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list := args[0]
		index, err := value.SassIndexToListIndex(list, args[1], "n", ec.WarnDeprecation)
		if err != nil {
			return nil, err
		}
		l, err := list.AsList()
		if err != nil {
			return nil, err
		}
		newContents := make([]value.Value, len(l))
		copy(newContents, l)
		newContents[index] = args[2]
		return value.WithListContents(list, newContents, nil), nil
	})
}

// joinFunction implements list.join($list1, $list2, $separator: auto,
// $bracketed: auto). An auto separator prefers the first list's separator,
// falling back to the second's, and to space when both are undecided; an auto
// bracketed flag inherits the first list's brackets.
func joinFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("join", "$list1, $list2, $separator: auto, $bracketed: auto", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list1 := args[0]
		list2 := args[1]

		var separator value.ListSeparator
		if len(args) > 2 && args[2] != nil && args[2] != value.Null {
			sepStr, err := value.AssertString(args[2], new("separator"))
			if err != nil {
				return nil, err
			}
			switch sepStr.Text {
			case "auto":
				s1 := list1.Separator()
				s2 := list2.Separator()
				if s1 == value.ListSeparatorUndecided && s2 == value.ListSeparatorUndecided {
					separator = value.ListSeparatorSpace
				} else if s1 == value.ListSeparatorUndecided {
					separator = s2
				} else {
					separator = s1
				}
			case "space":
				separator = value.ListSeparatorSpace
			case "comma":
				separator = value.ListSeparatorComma
			case "slash":
				separator = value.ListSeparatorSlash
			default:
				return nil, sasscommon.NewSassScriptException("Must be \"space\", \"comma\", \"slash\", or \"auto\".", new("separator"))
			}
		} else {
			s1 := list1.Separator()
			s2 := list2.Separator()
			if s1 == value.ListSeparatorUndecided && s2 == value.ListSeparatorUndecided {
				separator = value.ListSeparatorSpace
			} else if s1 == value.ListSeparatorUndecided {
				separator = s2
			} else {
				separator = s1
			}
		}

		var bracketed bool
		s, ok := args[3].(*value.SassString)
		if ok && s.Text == "auto" {
			bracketed = list1.HasBrackets()
		} else {
			bracketed = args[3].IsTruthy()
		}

		newList := make([]value.Value, 0, list1.LengthAsList()+list2.LengthAsList())
		l1, err := list1.AsList()
		if err != nil {
			return nil, err
		}
		l2, err := list2.AsList()
		if err != nil {
			return nil, err
		}
		newList = append(newList, l1...)
		newList = append(newList, l2...)
		list, err := value.NewSassList(newList, separator, bracketed)
		if err != nil {
			return nil, err
		}
		return list, nil
	})
}

// appendFunction implements list.append($list, $val, $separator: auto). An
// auto separator keeps the list's separator, defaulting to space for
// undecided lists; brackets always carry over from the input list.
func appendFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("append", "$list, $val, $separator: auto", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list := args[0]
		val := args[1]

		var separator value.ListSeparator
		if len(args) > 2 && args[2] != nil && args[2] != value.Null {
			sepStr, err := value.AssertString(args[2], new("separator"))
			if err != nil {
				return nil, err
			}
			switch sepStr.Text {
			case "auto":
				if list.Separator() == value.ListSeparatorUndecided {
					separator = value.ListSeparatorSpace
				} else {
					separator = list.Separator()
				}
			case "space":
				separator = value.ListSeparatorSpace
			case "comma":
				separator = value.ListSeparatorComma
			case "slash":
				separator = value.ListSeparatorSlash
			default:
				return nil, sasscommon.NewSassScriptException("Must be \"space\", \"comma\", \"slash\", or \"auto\".", new("separator"))
			}
		} else {
			if list.Separator() == value.ListSeparatorUndecided {
				separator = value.ListSeparatorSpace
			} else {
				separator = list.Separator()
			}
		}

		l, err := list.AsList()
		if err != nil {
			return nil, err
		}
		newContents := make([]value.Value, 0, list.LengthAsList()+1)
		newContents = append(newContents, l...)
		newContents = append(newContents, val)
		result, err := value.NewSassList(newContents, separator, list.HasBrackets())
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// zipFunction implements list.zip($lists...): space-separated tuples drawn
// positionally, stopping at the shortest list, gathered into a comma-separated
// result. No input lists yield the empty comma-separated list.
func zipFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("zip", "$lists...", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		argList := args[0]
		lists, err := argList.AsList()
		if err != nil {
			return nil, err
		}
		if len(lists) == 0 {
			list, err := value.NewSassList(nil, value.ListSeparatorComma, false)
			if err != nil {
				return nil, err
			}
			return list, nil
		}

		var results []value.Value
		i := 0
		for {
			var allHave bool = true
			for _, list := range lists {
				if i >= list.LengthAsList() {
					allHave = false
					break
				}
			}
			if !allHave {
				break
			}
			zipEntry := make([]value.Value, len(lists))
			for j, list := range lists {
				l, err := list.AsList()
				if err != nil {
					return nil, err
				}
				zipEntry[j] = l[i]
			}
			zipList, err := value.NewSassList(zipEntry, value.ListSeparatorSpace, false)
			if err != nil {
				return nil, err
			}
			results = append(results, zipList)
			i++
		}
		out, err := value.NewSassList(results, value.ListSeparatorComma, false)
		if err != nil {
			return nil, err
		}
		return out, nil
	})
}

// indexFunction implements list.index($list, $value): the 1-based position of
// the first equal element, or null when absent.
func indexFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("index", "$list, $value", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list, err := args[0].AsList()
		if err != nil {
			return nil, err
		}
		val := args[1]
		for i, item := range list {
			if item.Equals(val) {
				return value.NewUnitlessNumber(float64(i + 1)), nil
			}
		}
		return value.Null, nil
	})
}

// isBracketedFunction implements list.is-bracketed($list).
func isBracketedFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("is-bracketed", "$list", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		if args[0].HasBrackets() {
			return value.SassTrue, nil
		}
		return value.SassFalse, nil
	})
}

// separatorFunction implements list.separator($list): the unquoted "comma",
// "slash", or "space" name. An undecided separator reports "space".
func separatorFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("separator", "$list", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		switch args[0].Separator() {
		case value.ListSeparatorComma:
			return &value.SassString{Text: "comma", HasQuotes: false}, nil
		case value.ListSeparatorSlash:
			return &value.SassString{Text: "slash", HasQuotes: false}, nil
		default:
			return &value.SassString{Text: "space", HasQuotes: false}, nil
		}
	})
}

// slashFunction implements the module-only list.slash($elements...): the
// elements as a slash-separated list, requiring at least two.
func slashFunction() *BuiltInCallable {
	return MustNewBuiltInCallableFunction("slash", "$elements...", "sass:list", func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
		list, err := args[0].AsList()
		if err != nil {
			return nil, err
		}
		if len(list) < 2 {
			return nil, sasscommon.NewSassScriptException("At least two elements are required.", nil)
		}
		result, err := value.NewSassList(list, value.ListSeparatorSlash, false)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}
