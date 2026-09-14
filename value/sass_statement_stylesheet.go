// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/stylesheet.dart

import (
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// ParseTimeWarning is a warning discovered during parsing, held back for
// evaluation to emit once a logger is available.
//
// Matches Dart: ParseTimeWarning typedef in lib/src/ast/sass/statement/stylesheet.dart
type ParseTimeWarning struct {
	// Deprecation identifies the deprecation behind the warning, if any.
	Deprecation *deprecation.Deprecation
	// Message is the human-readable warning text.
	Message string
	// Span covers the source that triggered the warning.
	Span sasscommon.FileSpan
}

// Stylesheet is a Sass stylesheet, the root node of the AST.
//
// It holds the top-level statements plus the leading @use/@forward preamble
// and any warnings or global-variable spans collected while parsing.
type Stylesheet struct {
	parent            ParentStatement
	span              sasscommon.FileSpan
	plainCss          bool
	uses              []*UseRule
	forwards          []*ForwardRule
	parseTimeWarnings []ParseTimeWarning
	globalVariables   map[string]sasscommon.FileSpan
}

// NewStylesheet creates a stylesheet with no parse-time warnings and no
// plain-CSS flag.
func NewStylesheet(children []Statement, span sasscommon.FileSpan) *Stylesheet {
	return newStylesheetInternal(children, span, []ParseTimeWarning{}, false, nil)
}

// NewStylesheetDetailed creates a stylesheet carrying parser warnings, the
// plain-CSS marker, and the global-variable span index.
func NewStylesheetDetailed(children []Statement, span sasscommon.FileSpan, warnings []ParseTimeWarning, plainCss bool, globalVariables map[string]sasscommon.FileSpan) *Stylesheet {
	return newStylesheetInternal(children, span, warnings, plainCss, globalVariables)
}

func newStylesheetInternal(children []Statement, span sasscommon.FileSpan, parseTimeWarnings []ParseTimeWarning, plainCss bool, globalVariables map[string]sasscommon.FileSpan) *Stylesheet {
	if parseTimeWarnings == nil {
		parseTimeWarnings = []ParseTimeWarning{}
	}
	if globalVariables == nil {
		globalVariables = map[string]sasscommon.FileSpan{}
	}

	ch := make([]Statement, len(children))
	copy(ch, children)

	s := &Stylesheet{
		parent:            NewParentStatement(ch),
		span:              span,
		plainCss:          plainCss,
		parseTimeWarnings: parseTimeWarnings,
		globalVariables:   globalVariables,
	}

loop:
	for _, child := range s.parent.Children {
		switch c := any(child).(type) {
		case *UseRule:
			s.uses = append(s.uses, c)
		case *ForwardRule:
			s.forwards = append(s.forwards, c)
		case *SilentComment, *LoudComment, *VariableDeclaration:
		default:
			break loop
		}
	}

	if s.uses == nil {
		s.uses = []*UseRule{}
	}
	if s.forwards == nil {
		s.forwards = []*ForwardRule{}
	}

	return s
}

// Uses returns the leading @use rules in source order.
func (s *Stylesheet) Uses() []*UseRule {
	result := make([]*UseRule, len(s.uses))
	copy(result, s.uses)
	return result
}

// Forwards returns the leading @forward rules in source order.
func (s *Stylesheet) Forwards() []*ForwardRule {
	result := make([]*ForwardRule, len(s.forwards))
	copy(result, s.forwards)
	return result
}
func (s *Stylesheet) Span() (sasscommon.FileSpan, error) { return s.span, nil }

// ParseTimeWarnings returns the warnings collected while parsing, emitted
// during evaluation once a logger is available.
func (s *Stylesheet) ParseTimeWarnings() []ParseTimeWarning { return s.parseTimeWarnings }

// GlobalVariables maps normalized global variable names to their definition
// spans.
func (s *Stylesheet) GlobalVariables() map[string]sasscommon.FileSpan {
	return s.globalVariables
}

// IsPlainCss reports whether the stylesheet was parsed from plain CSS.
func (s *Stylesheet) IsPlainCss() bool         { return s.plainCss }
func (s *Stylesheet) IsStatement()             {}
func (s *Stylesheet) IsSassNode()              {}
func (s *Stylesheet) IsAstNode()               {}
func (s *Stylesheet) GetChildren() []Statement { return s.parent.Children }
func (s *Stylesheet) HasDeclarations() bool    { return s.parent.HasDeclarations() }
func (s *Stylesheet) String() (string, error) {
	var builder strings.Builder
	for i, child := range s.parent.Children {
		if i > 0 {
			builder.WriteString(" ")
		}
		childStr, err := statementChildString(child)
		if err != nil {
			return "", err
		}
		builder.WriteString(childStr)
	}
	return builder.String(), nil
}

// ParseScss parses SCSS contents into a Stylesheet.
// Matches Dart: Stylesheet.parseScss(contents, url: ..., parseSelectors: ...)
func ParseScss(contents []byte, url *goUrl.URL, parseSelectors bool) (*Stylesheet, error) {
	return NewScssParser(contents, url, parseSelectors).Parse()
}

// ParseSass parses indented-syntax contents into a Stylesheet.
// Matches Dart: Stylesheet.parseSass(contents, url: ..., parseSelectors: ...)
func ParseSass(contents []byte, url *goUrl.URL, parseSelectors bool) (*Stylesheet, error) {
	return NewSassParser(contents, url, parseSelectors).Parse()
}

// ParseCss parses plain CSS contents into a Stylesheet.
// Matches Dart: Stylesheet.parseCss(contents, url: ..., parseSelectors: ...)
func ParseCss(contents []byte, url *goUrl.URL, parseSelectors bool, disallowedFunctionNames map[string]bool) (*Stylesheet, error) {
	return NewCssParser(contents, url, parseSelectors, disallowedFunctionNames).Parse()
}
