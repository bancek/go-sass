// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/parameter_list.dart

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ParseParameterList parses a parameter list declaration from contents,
// which should be of the form "@rule name(args) {".
//
// If passed, url is the name of the file from which contents comes. A parse
// failure returns a SassFormatException.
//
// Matches Dart: ParameterList.parse
func ParseParameterList(contents string, url *url.URL) (*ParameterList, error) {
	p := NewStylesheetParser([]byte(contents), url, false, nil)
	return p.parseParameterList()
}

// ParameterList is a parameter declaration, as for a function or mixin
// definition.
//
// Matches Dart: ParameterList
type ParameterList struct {
	// Parameters are the parameters that are taken.
	Parameters []*Parameter
	// RestParameter is the name of the rest parameter (as in `$args...`),
	// or nil when none was declared.
	RestParameter *string
	span          sasscommon.FileSpan
}

// NewParameterList creates a parameter declaration, copying parameters.
//
// Matches Dart: ParameterList.new
func NewParameterList(parameters []*Parameter, span sasscommon.FileSpan, restParameter *string) *ParameterList {
	params := make([]*Parameter, len(parameters))
	copy(params, parameters)
	return &ParameterList{
		Parameters:    params,
		span:          span,
		RestParameter: restParameter,
	}
}

// NewParameterListEmpty creates a declaration that declares no parameters.
//
// Matches Dart: ParameterList.empty
func NewParameterListEmpty(span sasscommon.FileSpan) *ParameterList {
	return &ParameterList{
		Parameters: []*Parameter{},
		span:       span,
	}
}

func (p *ParameterList) Span() (sasscommon.FileSpan, error) { return p.span, nil }

// SpanWithName returns span expanded to include an identifier immediately
// before the declaration, when one is present: it walks back over whitespace
// and name characters and keeps the extension only for valid identifier
// starts. Empty spans (such as a mixin declared without parentheses) are
// trimmed.
//
// Matches Dart: ParameterList.spanWithName
func (p *ParameterList) SpanWithName() sasscommon.FileSpan {
	if p.span == nil {
		return nil
	}
	sourceFile, err := p.span.File()
	if err != nil {
		return p.span
	}
	sourceText := ""
	if sourceFile != nil {
		sourceText = sourceFile.Text()
	}
	if sourceText == "" {
		return p.span
	}
	startLoc, err := p.span.StartLocation()
	if err != nil {
		return p.span
	}
	i := startLoc.Offset - 1
	for i > 0 && isWhitespaceByte(sourceText[i]) {
		i--
	}
	if i < 0 || !isNameByte(sourceText[i]) {
		return p.span
	}
	i--
	for i >= 0 && isNameByte(sourceText[i]) {
		i--
	}
	nameStart := i + 1
	ch := sourceText[nameStart]
	if !(ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch >= 0x80) {
		return p.span
	}
	endLoc, err := p.span.EndLocation()
	if err != nil {
		return p.span
	}
	endOffset := endLoc.Offset
	for endOffset > nameStart && isWhitespaceByte(sourceText[endOffset-1]) {
		endOffset--
	}
	return sasscommon.NewFileSpanInFile(
		sourceFile,
		nameStart,
		endOffset,
	)
}

// IsEmpty returns whether this declaration takes no parameters: no fixed
// parameters and no rest parameter.
//
// Matches Dart: ParameterList.isEmpty
func (p *ParameterList) IsEmpty() bool {
	return len(p.Parameters) == 0 && p.RestParameter == nil
}

func (p *ParameterList) IsSassNode() {}
func (p *ParameterList) IsAstNode()  {}

// Verify returns an error when positional and names are not valid for this
// parameter declaration: an argument passed both by position and by name, a
// missing required argument, too many positional arguments, or unknown named
// arguments. Errors point at the invocation with the declaration as a
// secondary span.
//
// Matches Dart: ParameterList.verify
func (p *ParameterList) Verify(positional int, names map[string]struct{}) error {
	namedUsed := 0
	for i, param := range p.Parameters {
		if i < positional {
			if _, ok := names[param.Name()]; ok {
				origName, err := param.OriginalName()
				if err != nil {
					return err
				}
				return sasscommon.NewSassScriptException(fmt.Sprintf("Argument %s was passed both by position and by name.", origName), nil)
			}
		} else if _, ok := names[param.Name()]; ok {
			namedUsed++
		} else if param.DefaultValue == nil {
			origName, err := param.OriginalName()
			if err != nil {
				return err
			}
			return &sasscommon.MultiSpanSassScriptException{
				Message:        fmt.Sprintf("Missing argument %s.", origName),
				PrimaryLabel:   "invocation",
				SecondarySpans: map[sasscommon.FileSpan]string{p.SpanWithName(): "declaration"},
			}
		}
	}

	if p.RestParameter != nil {
		return nil
	}

	if positional > len(p.Parameters) {
		posWord := ""
		if len(names) > 0 {
			posWord = "positional "
		}
		return &sasscommon.MultiSpanSassScriptException{
			Message:        fmt.Sprintf("Only %d %s%s allowed, but %d %s passed.", len(p.Parameters), posWord, util.Pluralize("argument", len(p.Parameters), nil), positional, util.Pluralize("was", positional, new("were"))),
			PrimaryLabel:   "invocation",
			SecondarySpans: map[sasscommon.FileSpan]string{p.SpanWithName(): "declaration"},
		}
	}

	if namedUsed < len(names) {
		unknownNames := make(map[string]struct{})
		for name := range names {
			unknownNames[name] = struct{}{}
		}
		for _, param := range p.Parameters {
			delete(unknownNames, param.Name())
		}
		nameList := make([]string, 0, len(unknownNames))
		for name := range unknownNames {
			origName, err := p.OriginalParameterName(name)
			if err != nil {
				nameList = append(nameList, "$"+name)
			} else {
				nameList = append(nameList, origName)
			}
		}
		return &sasscommon.MultiSpanSassScriptException{
			Message:        fmt.Sprintf("No %s named %s.", util.Pluralize("parameter", len(unknownNames), nil), toSentence(nameList, "or")),
			PrimaryLabel:   "invocation",
			SecondarySpans: map[sasscommon.FileSpan]string{p.SpanWithName(): "declaration"},
		}
	}

	return nil
}

// OriginalParameterName returns the parameter named name with a leading `$`
// and its original underscores (which are otherwise converted to hyphens).
// Rest parameters are re-sliced from source text after the last `$`.
//
// Matches Dart: ParameterList._originalParameterName
func (p *ParameterList) OriginalParameterName(name string) (string, error) {
	if p.RestParameter != nil && name == *p.RestParameter {
		text, err := p.span.SpanText()
		if err != nil {
			return "", err
		}
		fromDollar := text[strings.LastIndex(text, "$"):]
		dotIdx := strings.Index(fromDollar, ".")
		if dotIdx < 0 {
			return fromDollar, nil
		}
		return fromDollar[:dotIdx], nil
	}

	for _, param := range p.Parameters {
		if param.Name() == name {
			return param.OriginalName()
		}
	}

	return "", fmt.Errorf("This declaration has no parameter named \"$%s\".", name)
}

// Matches returns whether positional and names are valid for this
// parameter declaration, using the same arity rules as Verify but answering
// true or false instead of raising an error.
//
// Matches Dart: ParameterList.matches
func (p *ParameterList) Matches(positional int, names map[string]struct{}) bool {
	namedUsed := 0
	for i, param := range p.Parameters {
		if i < positional {
			if _, ok := names[param.Name()]; ok {
				return false
			}
		} else if _, ok := names[param.Name()]; ok {
			namedUsed++
		} else if param.DefaultValue == nil {
			return false
		}
	}

	if p.RestParameter != nil {
		return true
	}
	if positional > len(p.Parameters) {
		return false
	}
	if namedUsed < len(names) {
		return false
	}
	return true
}

// String renders the declaration as `$name` entries joined by commas, with
// a trailing `$rest...` when a rest parameter exists.
func (p *ParameterList) String() (string, error) {
	var parts []string
	for _, arg := range p.Parameters {
		s, err := arg.String()
		if err != nil {
			return "", err
		}
		parts = append(parts, "$"+s)
	}
	if p.RestParameter != nil {
		parts = append(parts, "$"+*p.RestParameter+"...")
	}
	return strings.Join(parts, ", "), nil
}
