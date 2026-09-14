// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression.dart

import goUrl "net/url"

// Expression is a SassScript expression in a Sass syntax tree, implemented
// by the 18 expression structs in sass_expression_*.go.
//
// Matches Dart: Expression (sealed abstract class; Go keeps the set closed
// through the IsExpression marker instead).
type Expression interface {
	SassNode
	IsExpression()
	String() (string, error)
	// SourceInterpolation returns the equivalent of parsing this
	// expression's source as an interpolated plain-CSS value, or nil when
	// the expression is not valid interpolated plain CSS.
	//
	// This runs the expression through SourceInterpolationVisitor; a nil
	// buffer there surfaces here as a nil return.
	//
	// Matches Dart: Expression.sourceInterpolation (internal)
	SourceInterpolation() *Interpolation
	// IsCalculationSafe returns whether this expression can be used in a
	// calculation context.
	//
	// Matches Dart: Expression.isCalculationSafe
	IsCalculationSafe() (bool, error)
	// IsPlainCss returns whether this expression is valid plain CSS that will
	// produce the same result as it would in Sass. If allowInterpolation is
	// true, interpolated expressions are allowed even if they contain
	// SassScript.
	//
	// Matches Dart: Expression.isPlainCss
	IsPlainCss(allowInterpolation bool) (bool, error)
	AcceptValue(v ExpressionVisitor[Value]) (Value, error)
	// AcceptBool runs the boolean-result visitor over this expression.
	//
	// This is the collapsed owner of Dart's per-class accept() docs ("calls
	// the appropriate visit method"): every expression's AcceptBool forwards
	// to the matching Visit method, as do AcceptValue, AcceptVoid, and
	// AcceptExpr with their own result types.
	AcceptBool(v ExpressionVisitor[bool]) (bool, error)
	// AcceptVoid runs the effect-only visitor over this expression.
	AcceptVoid(v ExpressionVisitor[struct{}]) (struct{}, error)
	// AcceptExpr runs the expression-rewriting visitor over this expression.
	AcceptExpr(v ExpressionVisitor[Expression]) (Expression, error)
}

// ParseExpression parses an expression from contents.
//
// If passed, url is the name of the file from which contents comes. A parse
// failure returns a SassFormatException.
//
// Matches Dart: Expression.parse factory constructor
func ParseExpression(contents string, url *goUrl.URL) (Expression, error) {
	parser := NewScssParser([]byte(contents), url, false)
	expr, _, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}
	return expr, nil
}
