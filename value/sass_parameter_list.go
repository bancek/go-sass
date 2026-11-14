// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/parameter_list.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sassurl"

	"github.com/bancek/go-sass/orderedmap"
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
func ParseParameterList(contents string, url *sassurl.URL) (*ParameterList, error) {
	p := NewStylesheetParser([]byte(contents), url, false, nil)
	return p.parseParameterList()
}

// EmptyParameterList returns the declaration with no parameters, the Go
// spelling of Dart's `ParameterList.empty` for callables that declare none.
//
// Matches Dart: ParameterList.empty
func EmptyParameterList() *ParameterList {
	return NewParameterList(nil, sasscommon.BogusSpan, nil)
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

// Verify returns an error when positional and named are not valid for this
// parameter declaration: an argument passed both by position and by name, a
// missing required argument, too many positional arguments, or unknown named
// arguments. Errors point at the invocation with the declaration as a
// secondary span.
//
// named is the evaluated named-argument map, whose keys Dart reads through
// MapKeySet(evaluated.named); only the keys matter here, and they are reported
// in the order they were passed.
//
// Matches Dart: ParameterList.verify
func (p *ParameterList) Verify(positional int, named orderedmap.MapKeySet[string]) error {
	// Dart's MapKeySet is backed by the map itself, so a null argument map is
	// only reachable from Go callers; treat it as empty.
	if named == nil {
		named = orderedmap.New[string, Value]()
	}

	namedUsed := 0
	for i, param := range p.Parameters {
		nameIdent, err := util.ToCssIdentifier(param.Name())
		if err != nil {
			return err
		}
		if i < positional {
			if named.Has(nameIdent) {
				origName, err := param.OriginalName()
				if err != nil {
					return err
				}
				return sasscommon.NewSassScriptException(
					fmt.Sprintf("Argument %s was passed both by position and by name.", origName),
					nil)
			}
		} else if named.Has(nameIdent) {
			namedUsed++
		} else if param.DefaultValue == nil {
			origName, err := param.OriginalName()
			if err != nil {
				return err
			}
			return &sasscommon.MultiSpanSassScriptException{
				Message:        fmt.Sprintf("Missing argument %s.", origName),
				PrimaryLabel:   "invocation",
				SecondarySpans: sasscommon.NewSecondarySpans(sasscommon.SecondarySpan{Span: p.SpanWithName(), Label: "declaration"}),
			}
		}
	}

	if p.RestParameter != nil {
		return nil
	}

	if positional > len(p.Parameters) {
		posWord := ""
		if named.Len() > 0 {
			posWord = "positional "
		}
		return &sasscommon.MultiSpanSassScriptException{
			Message:        fmt.Sprintf("Only %d %s%s allowed, but %d %s passed.", len(p.Parameters), posWord, util.Pluralize("argument", len(p.Parameters), nil), positional, util.Pluralize("was", positional, new("were"))),
			PrimaryLabel:   "invocation",
			SecondarySpans: sasscommon.NewSecondarySpans(sasscommon.SecondarySpan{Span: p.SpanWithName(), Label: "declaration"}),
		}
	}

	if namedUsed < named.Len() {
		paramIdents := make([]string, 0, len(p.Parameters))
		for _, param := range p.Parameters {
			nameIdent, err := util.ToCssIdentifier(param.Name())
			if err != nil {
				return err
			}
			paramIdents = append(paramIdents, nameIdent)
		}

		unknownNames := make([]string, 0)
		for name := range named.Keys() {
			known := false
			for _, paramIdent := range paramIdents {
				if paramIdent == name {
					known = true
					break
				}
			}
			if !known {
				unknownNames = append(unknownNames, "$"+name)
			}
		}

		return &sasscommon.MultiSpanSassScriptException{
			Message:        fmt.Sprintf("No %s named %s.", util.Pluralize("parameter", len(unknownNames), nil), ToSentence(unknownNames, "or")),
			PrimaryLabel:   "invocation",
			SecondarySpans: sasscommon.NewSecondarySpans(sasscommon.SecondarySpan{Span: p.SpanWithName(), Label: "declaration"}),
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

// Matches returns whether positional and named are valid for this parameter
// declaration, using the same arity rules as Verify but answering true or
// false instead of raising an error.
//
// named is the evaluated named-argument map, as in Verify.
//
// Unlike Verify, the parameter names are compared as written, not in
// canonical form: Dart's matches still compares the raw parameter.name
// (c59636a0 changed only verify), so an invocation that spells a parameter
// name differently from its declaration does not match here.
//
// Matches Dart: ParameterList.matches
func (p *ParameterList) Matches(positional int, named orderedmap.MapKeySet[string]) bool {
	// Go-only guard: Dart's MapKeySet is always backed by a map, so a nil
	// interface can only come from a Go caller.
	if named == nil {
		named = orderedmap.New[string, Value]()
	}
	if p == nil {
		return positional == 0 && named.Len() == 0
	}

	namedUsed := 0
	for i, param := range p.Parameters {
		if i < positional {
			if named.Has(param.Name()) {
				return false
			}
		} else if named.Has(param.Name()) {
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
	if namedUsed < named.Len() {
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
		rest, err := util.ToCssIdentifier(*p.RestParameter)
		if err != nil {
			return "", err
		}
		parts = append(parts, "$"+rest+"...")
	}
	return strings.Join(parts, ", "), nil
}
