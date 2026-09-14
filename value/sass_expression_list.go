// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/list.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ListExpression is a list literal.
//
// Matches Dart: ListExpression
type ListExpression struct {
	// Contents are the elements of this list.
	Contents []Expression
	// Separator selects which separator this list uses.
	Separator ListSeparator
	// HasBrackets reports whether the list has square brackets.
	HasBrackets bool
	span        sasscommon.FileSpan
}

// NewListExpression creates a list literal, copying contents.
//
// Matches Dart: ListExpression.new
func NewListExpression(contents []Expression, separator ListSeparator, span sasscommon.FileSpan, hasBrackets bool) *ListExpression {
	c := make([]Expression, len(contents))
	copy(c, contents)
	return &ListExpression{
		Contents:    c,
		Separator:   separator,
		HasBrackets: hasBrackets,
		span:        span,
	}
}

func (e *ListExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *ListExpression) SourceInterpolation() *Interpolation { return nil }
func (e *ListExpression) IsExpression()                       {}
func (e *ListExpression) IsSassNode()                         {}
func (e *ListExpression) IsAstNode()                          {}

// String renders the list, adding parentheses for empty lists, lone trailing
// commas, and nested elements that would otherwise parse with a different
// grouping (see elementNeedsParens).
func (e *ListExpression) String() (string, error) {
	var buf strings.Builder

	if e.HasBrackets {
		buf.WriteRune('[')
	} else if len(e.Contents) == 0 || (len(e.Contents) == 1 && e.Separator == ListSeparatorComma) {
		buf.WriteRune('(')
	}

	for i, element := range e.Contents {
		if i > 0 {
			if e.Separator == ListSeparatorComma {
				buf.WriteString(", ")
			} else {
				buf.WriteRune(' ')
			}
		}
		elStr, err := element.String()
		if err != nil {
			return "", err
		}
		if e.elementNeedsParens(element) {
			buf.WriteRune('(')
			buf.WriteString(elStr)
			buf.WriteRune(')')
		} else {
			buf.WriteString(elStr)
		}
	}

	if e.HasBrackets {
		buf.WriteRune(']')
	} else if len(e.Contents) == 0 {
		buf.WriteRune(')')
	} else if len(e.Contents) == 1 && e.Separator == ListSeparatorComma {
		buf.WriteString(",)")
	}

	return buf.String(), nil
}

// elementNeedsParens reports whether expression, nested in this list, needs
// parentheses when printed: a bracketless nested list of two or more items
// whose separator would merge ambiguously, or a leading +/- unary operation
// in a space-separated list.
func (e *ListExpression) elementNeedsParens(expression Expression) bool {
	if listExpr, ok := any(expression).(*ListExpression); ok {
		if len(listExpr.Contents) >= 2 && !listExpr.HasBrackets {
			childSeparator := listExpr.Separator
			if e.Separator == ListSeparatorComma {
				return childSeparator == ListSeparatorComma
			}
			return childSeparator != ListSeparatorUndecided
		}
	}
	if unaryExpr, ok := any(expression).(*UnaryOperationExpression); ok {
		if unaryExpr.Operator == UnaryOperatorPlus || unaryExpr.Operator == UnaryOperatorMinus {
			return e.Separator == ListSeparatorSpace
		}
	}
	return false
}
