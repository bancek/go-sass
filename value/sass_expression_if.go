// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/if.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// IfBranch is a single branch of an IfExpression: a condition with the
// expression it guards. A nil Condition means the "else" fallback branch.
type IfBranch struct {
	Condition  IfConditionExpression // nil means "else"
	Expression Expression
}

// IfExpression is a CSS if() expression.
//
// In addition to supporting the plain-CSS syntax, this supports a sass()
// condition that evaluates SassScript expressions.
//
// Matches Dart: IfExpression
type IfExpression struct {
	// Branches are the conditional branches of the if(). A nil condition
	// marks the else branch, which is always taken when reached.
	Branches []IfBranch
	span     sasscommon.FileSpan
}

// NewIfExpression creates a CSS if() with the given branches, copying the
// slice. An empty branch list is rejected, matching Dart's ArgumentError.
//
// Matches Dart: IfExpression.new
func NewIfExpression(branches []IfBranch, span sasscommon.FileSpan) (*IfExpression, error) {
	if len(branches) == 0 {
		return nil, &sasscommon.ArgumentError{Name: "branches", Message: "branches may not be empty"}
	}
	b := make([]IfBranch, len(branches))
	copy(b, branches)
	return &IfExpression{Branches: b, span: span}, nil
}

func (e *IfExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *IfExpression) SourceInterpolation() *Interpolation { return nil }
func (e *IfExpression) IsExpression()                       {}
func (e *IfExpression) IsSassNode()                         {}
func (e *IfExpression) IsAstNode()                          {}

func (e *IfExpression) String() (string, error) {
	var buf strings.Builder
	buf.WriteString("if(")
	for i, branch := range e.Branches {
		if i > 0 {
			buf.WriteString("; ")
		}
		if branch.Condition != nil {
			condStr, err := branch.Condition.String()
			if err != nil {
				return "", err
			}
			buf.WriteString(condStr)
		} else {
			buf.WriteString("else")
		}
		buf.WriteString(": ")
		branchExprStr, err := branch.Expression.String()
		if err != nil {
			return "", err
		}
		buf.WriteString(branchExprStr)
	}
	buf.WriteRune(')')
	return buf.String(), nil
}

// IfConditionExpression is the parent of conditions in an IfExpression,
// implemented by the six IfCondition* structs below.
//
// Matches Dart: IfConditionExpression (sealed class; Go keeps the set closed
// through the unexported isIfConditionExpression marker instead).
type IfConditionExpression interface {
	SassNode
	isIfConditionExpression()
	// ToInterpolation converts the condition into an interpolation producing
	// the same value. A sass() condition cannot be converted when an
	// arbitrary substitution is present, and reports a
	// MultiSourceSpanFormatError pointing at both spans.
	//
	// Matches Dart: IfConditionExpression.toInterpolation (internal)
	ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error)
	// IsArbitrarySubstitution reports whether the condition may be replaced
	// with multiple tokens at evaluation or render time.
	//
	// Matches Dart: IfConditionExpression.isArbitrarySubstitution (internal)
	IsArbitrarySubstitution() bool
	String() (string, error)
	// AcceptAny runs the untyped visitor over the condition.
	//
	// The four Accept methods are the collapsed owners of Dart's per-class
	// accept() docs ("calls the appropriate visit method"); each concrete
	// condition forwards to its matching Visit method.
	AcceptAny(v IfConditionExpressionVisitor[any]) (any, error)
	// AcceptBool runs the boolean-result visitor over the condition.
	AcceptBool(v IfConditionExpressionVisitor[bool]) (bool, error)
	// AcceptVoid runs the effect-only visitor over the condition.
	AcceptVoid(v IfConditionExpressionVisitor[struct{}]) (struct{}, error)
	// AcceptIfConditionExpression runs the rewriting visitor over the
	// condition.
	AcceptIfConditionExpression(v IfConditionExpressionVisitor[IfConditionExpression]) (IfConditionExpression, error)
}

// IfConditionParenthesized is a parenthesized condition.
//
// Matches Dart: IfConditionParenthesized
type IfConditionParenthesized struct {
	// Expression is the parenthesized expression.
	Expression IfConditionExpression
	span       sasscommon.FileSpan
}

// NewIfConditionParenthesized creates a parenthesized condition.
//
// Matches Dart: IfConditionParenthesized.new
func NewIfConditionParenthesized(expression IfConditionExpression, span sasscommon.FileSpan) *IfConditionParenthesized {
	return &IfConditionParenthesized{Expression: expression, span: span}
}

func (c *IfConditionParenthesized) Span() (sasscommon.FileSpan, error) { return c.span, nil }
func (c *IfConditionParenthesized) IsAstNode()                         {}
func (c *IfConditionParenthesized) IsSassNode()                        {}
func (c *IfConditionParenthesized) isIfConditionExpression()           {}
func (c *IfConditionParenthesized) IsArbitrarySubstitution() bool      { return false }
func (c *IfConditionParenthesized) ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	buf.WriteCharCode('(')
	inner, err := c.Expression.ToInterpolation(arbitrarySubstitution)
	if err != nil {
		return nil, err
	}
	buf.AddInterpolation(inner)
	buf.WriteCharCode(')')
	span, err := c.Span()
	if err != nil {
		return nil, err
	}
	return buf.Interpolation(span)
}
func (c *IfConditionParenthesized) String() (string, error) {
	exprStr, err := c.Expression.String()
	if err != nil {
		return "", err
	}
	return "(" + exprStr + ")", nil
}

// IfConditionNegation is a negated condition.
//
// Matches Dart: IfConditionNegation
type IfConditionNegation struct {
	// Expression is the expression negated by this condition.
	Expression IfConditionExpression
	span       sasscommon.FileSpan
}

// NewIfConditionNegation creates a negated condition.
//
// Matches Dart: IfConditionNegation.new
func NewIfConditionNegation(expression IfConditionExpression, span sasscommon.FileSpan) *IfConditionNegation {
	return &IfConditionNegation{Expression: expression, span: span}
}

func (c *IfConditionNegation) Span() (sasscommon.FileSpan, error) { return c.span, nil }
func (c *IfConditionNegation) IsAstNode()                         {}
func (c *IfConditionNegation) IsSassNode()                        {}
func (c *IfConditionNegation) isIfConditionExpression()           {}
func (c *IfConditionNegation) IsArbitrarySubstitution() bool      { return false }
func (c *IfConditionNegation) ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	buf.Write("not ")
	inner, err := c.Expression.ToInterpolation(arbitrarySubstitution)
	if err != nil {
		return nil, err
	}
	buf.AddInterpolation(inner)
	span, err := c.Span()
	if err != nil {
		return nil, err
	}
	return buf.Interpolation(span)
}
func (c *IfConditionNegation) String() (string, error) {
	exprStr, err := c.Expression.String()
	if err != nil {
		return "", err
	}
	return "not " + exprStr, nil
}

// IfConditionOperation is a sequence of "and"s or "or"s.
//
// Matches Dart: IfConditionOperation
type IfConditionOperation struct {
	// Expressions are the conditions conjoined or disjoined by this
	// operation.
	Expressions []IfConditionExpression
	// Op is the operator separating every expression.
	Op BooleanOperator
}

// NewIfConditionOperation creates an and/or chain over at least two
// conditions, copying the slice. Shorter lists are rejected, matching Dart's
// ArgumentError.
//
// Matches Dart: IfConditionOperation.new
func NewIfConditionOperation(expressions []IfConditionExpression, op BooleanOperator) (*IfConditionOperation, error) {
	if len(expressions) < 2 {
		return nil, &sasscommon.ArgumentError{Name: "expressions", Message: "expressions must have length >= 2"}
	}
	e := make([]IfConditionExpression, len(expressions))
	copy(e, expressions)
	return &IfConditionOperation{Expressions: e, Op: op}, nil
}

func (c *IfConditionOperation) Span() (sasscommon.FileSpan, error) {
	firstSpan, err := c.Expressions[0].Span()
	if err != nil {
		return nil, err
	}
	lastSpan, err := c.Expressions[len(c.Expressions)-1].Span()
	if err != nil {
		return nil, err
	}
	return firstSpan.Expand(lastSpan)
}
func (c *IfConditionOperation) IsAstNode()                    {}
func (c *IfConditionOperation) IsSassNode()                   {}
func (c *IfConditionOperation) isIfConditionExpression()      {}
func (c *IfConditionOperation) IsArbitrarySubstitution() bool { return false }
func (c *IfConditionOperation) ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	for i, expr := range c.Expressions {
		if i > 0 {
			buf.WriteCharCode(' ')
			buf.Write(c.Op.String())
			buf.WriteCharCode(' ')
		}
		inner, err := expr.ToInterpolation(arbitrarySubstitution)
		if err != nil {
			return nil, err
		}
		buf.AddInterpolation(inner)
	}
	span, err := c.Span()
	if err != nil {
		return nil, err
	}
	return buf.Interpolation(span)
}

func (c *IfConditionOperation) String() (string, error) {
	var parts []string
	for _, e := range c.Expressions {
		exprStr, err := e.String()
		if err != nil {
			return "", err
		}
		parts = append(parts, exprStr)
	}
	return strings.Join(parts, " "+c.Op.String()+" "), nil
}

// IfConditionFunction is a plain-CSS function-style condition.
//
// Matches Dart: IfConditionFunction
type IfConditionFunction struct {
	// Name is the name of the function being called.
	Name *Interpolation
	// Arguments are the arguments passed to the function call.
	Arguments *Interpolation
	span      sasscommon.FileSpan
}

// NewIfConditionFunction creates a function-style condition.
//
// Matches Dart: IfConditionFunction.new
func NewIfConditionFunction(name, arguments *Interpolation, span sasscommon.FileSpan) *IfConditionFunction {
	return &IfConditionFunction{Name: name, Arguments: arguments, span: span}
}

func (c *IfConditionFunction) Span() (sasscommon.FileSpan, error) { return c.span, nil }
func (c *IfConditionFunction) IsAstNode()                         {}
func (c *IfConditionFunction) IsSassNode()                        {}
func (c *IfConditionFunction) isIfConditionExpression()           {}

func (c *IfConditionFunction) IsArbitrarySubstitution() bool {
	// Only if(), var(), attr(), and custom-property functions may expand to
	// multiple tokens at render time; anything else is fixed syntax.
	plain := c.Name.AsPlain()
	if plain == nil {
		return false
	}
	lower := strings.ToLower(*plain)
	if lower == "if" || lower == "var" || lower == "attr" {
		return true
	}
	return strings.HasPrefix(lower, "--")
}

func (c *IfConditionFunction) ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	buf.AddInterpolation(c.Name)
	buf.WriteCharCode('(')
	buf.AddInterpolation(c.Arguments)
	buf.WriteCharCode(')')
	span, err := c.Span()
	if err != nil {
		return nil, err
	}
	return buf.Interpolation(span)
}

func (c *IfConditionFunction) String() (string, error) {
	nameStr, err := c.Name.String()
	if err != nil {
		return "", err
	}
	argsStr, err := c.Arguments.String()
	if err != nil {
		return "", err
	}
	return nameStr + "(" + argsStr + ")", nil
}

// IfConditionSass is a SassScript condition in an if() expression: the
// sass() form evaluated at compile time to true or false.
//
// Matches Dart: IfConditionSass
type IfConditionSass struct {
	// Expression determines whether this condition matches.
	Expression Expression
	span       sasscommon.FileSpan
}

// NewIfConditionSass creates a Sass-evaluated condition.
//
// Matches Dart: IfConditionSass.new
func NewIfConditionSass(expression Expression, span sasscommon.FileSpan) *IfConditionSass {
	return &IfConditionSass{Expression: expression, span: span}
}

func (c *IfConditionSass) Span() (sasscommon.FileSpan, error) { return c.span, nil }
func (c *IfConditionSass) IsAstNode()                         {}
func (c *IfConditionSass) IsSassNode()                        {}
func (c *IfConditionSass) isIfConditionExpression()           {}
func (c *IfConditionSass) IsArbitrarySubstitution() bool      { return false }
func (c *IfConditionSass) ToInterpolation(arbitrarySubstitution sasscommon.AstNode) (*Interpolation, error) {
	subSpan, err := arbitrarySubstitution.Span()
	if err != nil {
		return nil, err
	}
	cSpan, err := c.Span()
	if err != nil {
		return nil, err
	}
	return nil, &sasscommon.MultiSourceSpanFormatError{
		Message:      "if() conditions with arbitrary substitutions may not contain sass() expressions.",
		Span:         subSpan,
		PrimaryLabel: "arbitrary substitution",
		Secondary: map[sasscommon.FileSpan]string{
			cSpan: "sass() expression",
		},
	}
}
func (c *IfConditionSass) String() (string, error) {
	exprStr, err := c.Expression.String()
	if err != nil {
		return "", err
	}
	return "sass(" + exprStr + ")", nil
}

// IfConditionRaw is a chunk of raw text, possibly with interpolations.
//
// This represents explicit interpolation, as well as whole expressions where
// arbitrary substitutions stand in for operators.
//
// Matches Dart: IfConditionRaw
type IfConditionRaw struct {
	// Text encompasses this condition.
	Text *Interpolation
}

// NewIfConditionRaw creates a raw-text condition.
//
// Matches Dart: IfConditionRaw.new
func NewIfConditionRaw(text *Interpolation) *IfConditionRaw {
	return &IfConditionRaw{Text: text}
}

func (c *IfConditionRaw) Span() (sasscommon.FileSpan, error) { return c.Text.Span() }
func (c *IfConditionRaw) IsAstNode()                         {}
func (c *IfConditionRaw) IsSassNode()                        {}
func (c *IfConditionRaw) isIfConditionExpression()           {}
func (c *IfConditionRaw) IsArbitrarySubstitution() bool      { return true }
func (c *IfConditionRaw) ToInterpolation(sasscommon.AstNode) (*Interpolation, error) {
	return c.Text, nil
}
func (c *IfConditionRaw) String() (string, error) { return c.Text.String() }
