// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (StylesheetParser base class)

import (
	goUrl "net/url"

	"github.com/bancek/go-sass/sasscommon"
)

// StylesheetParser is the base parser shared by the SCSS and indented-syntax
// parsers, matching Dart's StylesheetParser in stylesheet.dart.
//
// Keeping the shared rules in one base type makes explicit which behaviors
// differ per syntax: editing this type rarely requires touching the
// subclasses, while editing one syntax parser usually needs a parallel change
// in the other. Per-syntax variance comes from constructor-assigned func
// fields (statement separators, child detection) and plain bool flags
// (indented, plainCss) rather than virtual dispatch.
//
// Only the members subclasses must reach are exposed; everything else stays
// unexported on Parser or here.
type StylesheetParser struct {
	Parser

	parseSelectors     bool
	isUseAllowed       bool
	inMixin            bool
	inContentBlock     bool
	inControlDirective bool
	inUnknownAtRule    bool
	inPlainCssFunction bool
	inStyleRule        bool
	inParentheses      bool
	inExpression       bool
	globalVariables    map[string]sasscommon.FileSpan

	warnings          []ParseTimeWarning
	lastSilentComment *SilentComment

	indented                 bool
	plainCss                 bool
	currentIndentation       func() int
	atRuleFn                 func(child func() (Statement, error), root bool) (Statement, error)
	styleRuleSelector        func() (*Interpolation, error)
	expectStatementSeparator func(name string) error
	atEndOfStatement         func() bool
	lookingAtChildren        func() (bool, error)
	scanElse                 func(ifIndentation int) (bool, error)
	children                 func(child func() (Statement, error)) ([]Statement, error)
	statements               func(statement func() (Statement, error)) ([]Statement, error)
	importArgumentFn         func() (Import, error)
	identifierLikeFn         func() (Expression, error)
	parenthesesFn            func() (Expression, error)
}

// NewStylesheetParser creates a StylesheetParser over contents, resolving
// relative URLs against url when provided. When parseSelectors is set, style
// rule selectors parse as interpolated selectors rather than raw
// interpolation; interpolationMap threads through so errors inside
// interpolation point back at the interpolation site. Use of @use-family
// rules is allowed until the parser consumes a rule that forbids them.
func NewStylesheetParser(contents []byte, url *goUrl.URL, parseSelectors bool, interpolationMap *InterpolationMap) *StylesheetParser {
	return &StylesheetParser{
		Parser:          *NewParser(contents, url, interpolationMap),
		parseSelectors:  parseSelectors,
		isUseAllowed:    true,
		globalVariables: make(map[string]sasscommon.FileSpan),
	}
}
