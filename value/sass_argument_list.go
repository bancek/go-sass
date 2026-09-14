// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/argument_list.dart

import (
	"maps"
	"slices"
	"strings"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

// ArgumentList is the set of arguments passed to a function, mixin, or
// @content invocation, split into positional, named, and rest parts.
type ArgumentList struct {
	// Positional holds the arguments passed by position, in source order.
	Positional []Expression
	// Named holds the arguments passed by name, in insertion order.
	Named *orderedmap.LinkedMap[string, Expression]
	// NamedSpans covers each named argument including its name.
	// It always carries the same keys in the same order as Named.
	NamedSpans map[string]sasscommon.FileSpan
	// Rest is the first rest argument ($args...), or nil when absent.
	Rest Expression
	// KeywordRest is the second rest argument, expected to hold only a
	// keyword map, or nil when absent.
	KeywordRest Expression
	span        sasscommon.FileSpan
}

// NewArgumentList creates an invocation from its positional, named, and rest
// parts, copying each collection. Rest must be non-nil whenever keywordRest
// is set, matching the Dart assertion.
func NewArgumentList(
	positional []Expression,
	named *orderedmap.LinkedMap[string, Expression],
	namedSpans map[string]sasscommon.FileSpan,
	span sasscommon.FileSpan,
	rest Expression,
	keywordRest Expression,
) *ArgumentList {
	pos := make([]Expression, len(positional))
	copy(pos, positional)
	n := orderedmap.NewWithCapacity[string, Expression](named.Len())
	for k, v := range named.Entries() {
		n.Put(k, v)
	}
	ns := make(map[string]sasscommon.FileSpan, len(namedSpans))
	maps.Copy(ns, namedSpans)
	return &ArgumentList{
		Positional:  pos,
		Named:       n,
		NamedSpans:  ns,
		span:        span,
		Rest:        rest,
		KeywordRest: keywordRest,
	}
}

// NewArgumentListEmpty creates an invocation that passes no arguments.
func NewArgumentListEmpty(span sasscommon.FileSpan) *ArgumentList {
	return &ArgumentList{
		Positional: []Expression{},
		Named:      orderedmap.New[string, Expression](),
		NamedSpans: map[string]sasscommon.FileSpan{},
		span:       span,
	}
}

func (a *ArgumentList) Span() (sasscommon.FileSpan, error) { return a.span, nil }
func (a *ArgumentList) IsSassNode()                        {}
func (a *ArgumentList) IsAstNode()                         {}

// Matches Dart: ArgumentList.isEmpty
func (a *ArgumentList) IsEmpty() bool {
	return len(a.Positional) == 0 && a.Named.Len() == 0 && a.Rest == nil
}

// Matches Dart: ArgumentList._parenthesizeArgument
func parenthesizeArgument(arg Expression) (string, error) {
	if list, ok := arg.(*ListExpression); ok &&
		list.Separator == ListSeparatorComma &&
		!list.HasBrackets &&
		len(list.Contents) >= 2 {
		s, err := list.String()
		if err != nil {
			return "", err
		}
		return "(" + s + ")", nil
	}
	return arg.String()
}

func (a *ArgumentList) String() (string, error) {
	var components []string
	for _, arg := range a.Positional {
		s, err := parenthesizeArgument(arg)
		if err != nil {
			return "", err
		}
		components = append(components, s)
	}
	// Iterate in insertion order, matching Dart's Map.pairs.
	names := slices.Collect(a.Named.Keys())
	for _, name := range names {
		v, _ := a.Named.Get(name)
		s, err := parenthesizeArgument(v)
		if err != nil {
			return "", err
		}
		components = append(components, "$"+name+": "+s)
	}
	if a.Rest != nil {
		s, err := parenthesizeArgument(a.Rest)
		if err != nil {
			return "", err
		}
		components = append(components, s+"...")
	}
	if a.KeywordRest != nil {
		s, err := parenthesizeArgument(a.KeywordRest)
		if err != nil {
			return "", err
		}
		components = append(components, s+"...")
	}
	return "(" + strings.Join(components, ", ") + ")", nil
}
