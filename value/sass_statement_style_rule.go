// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/style_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// StyleRule is a style rule.
//
// It applies its declarations and nested rules to elements matching its
// selector.
type StyleRule struct {
	parent ParentStatement
	// Selector is the interpolated selector the declarations apply to.
	// Exactly one of Selector and ParsedSelector is set: plain parses fill
	// Selector, while parseSelectors fills ParsedSelector instead.
	Selector *Interpolation
	// ParsedSelector holds the selector parsed as far as possible without
	// resolving interpolation. It is only set when the stylesheet is parsed
	// with selector parsing enabled and is unused by evaluation itself.
	ParsedSelector *InterpolatedSelectorList
	span           sasscommon.FileSpan
}

// NewStyleRule creates a style rule with an interpolated selector.
func NewStyleRule(selector *Interpolation, children []Statement, span sasscommon.FileSpan) *StyleRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &StyleRule{
		parent:         NewParentStatement(c),
		Selector:       selector,
		ParsedSelector: nil,
		span:           span,
	}
}

// NewStyleRuleWithParsedSelector creates a style rule whose selector was
// already parsed as far as interpolation allows. Selector stays nil.
func NewStyleRuleWithParsedSelector(parsedSelector *InterpolatedSelectorList, children []Statement, span sasscommon.FileSpan) *StyleRule {
	var c []Statement
	if children != nil {
		c = make([]Statement, len(children))
		copy(c, children)
	}
	return &StyleRule{
		parent:         NewParentStatement(c),
		Selector:       nil,
		ParsedSelector: parsedSelector,
		span:           span,
	}
}

func (r *StyleRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *StyleRule) IsStatement()                       {}
func (r *StyleRule) IsSassNode()                        {}
func (r *StyleRule) IsAstNode()                         {}
func (r *StyleRule) GetChildren() []Statement           { return r.parent.Children }
func (r *StyleRule) HasDeclarations() bool              { return r.parent.HasDeclarations() }
func (r *StyleRule) String() (string, error) {
	var selStr string
	if r.Selector != nil {
		var err error
		selStr, err = r.Selector.String()
		if err != nil {
			return "", err
		}
	} else {
		var err error
		selStr, err = r.ParsedSelector.String()
		if err != nil {
			return "", err
		}
	}
	var builder strings.Builder
	builder.WriteString(selStr)
	builder.WriteString(" {")
	parts := make([]string, len(r.parent.Children))
	for i, child := range r.parent.Children {
		s, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		parts[i] = s
	}
	builder.WriteString(strings.Join(parts, " "))
	builder.WriteString("}")
	return builder.String(), nil
}
