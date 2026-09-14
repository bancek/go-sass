// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/interpolation.dart

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// Interpolation is plain text interpolated with Sass expressions.
//
// Contents alternates text and unevaluated expressions: it holds strings and
// Expressions but never two adjacent strings, so every boundary between plain
// text and code stays unambiguous.
type Interpolation struct {
	// Contents holds the interpolation's text (string) and code (Expression)
	// elements, with no two adjacent strings.
	Contents []any // string or Expression
	// spans holds the source span of each element in Contents: nil for string
	// elements, and the span covering the whole `#{...}` (not just the
	// expression text) for Expression elements. Dart marks this @internal.
	spans []*sasscommon.FileSpan
	span  sasscommon.FileSpan
}

// NewInterpolationPlain creates an Interpolation with a single plain text
// element and no interpolated expressions.
//
// This is the fast path for text that needs an Interpolation wrapper without
// any code inside; the single string keeps a nil expression span.
// (Dart: Interpolation.plain.)
func NewInterpolationPlain(text string, span sasscommon.FileSpan) *Interpolation {
	return &Interpolation{
		Contents: []any{text},
		spans:    []*sasscommon.FileSpan{nil},
		span:     span,
	}
}

// NewInterpolation creates a new Interpolation with the given contents and
// spans. The spans must include a FileSpan for each Expression in contents.
//
// Contents may only hold strings or Expressions, never adjacent strings, and
// spans must parallel contents exactly: nil for every string element and the
// `#{...}`-covering span for every Expression. span must cover the whole
// interpolation. Violations return an ArgumentError.
// (Dart: Interpolation.new, with the same three validations in order.)
func NewInterpolation(contents []any, spans []*sasscommon.FileSpan, span sasscommon.FileSpan) (*Interpolation, error) {
	c := make([]any, len(contents))
	copy(c, contents)
	s := make([]*sasscommon.FileSpan, len(spans))
	copy(s, spans)

	in := &Interpolation{Contents: c, spans: s, span: span}

	if len(s) != len(c) {
		return nil, &sasscommon.ArgumentError{Name: "spans", Message: fmt.Sprintf("spans length (%d) must match contents length (%d)", len(s), len(c))}
	}

	for i, content := range c {
		if _, ok := content.(string); ok {
			if i > 0 {
				if _, ok := c[i-1].(string); ok {
					return nil, &sasscommon.ArgumentError{Message: "contents may not contain adjacent strings"}
				}
			}
			if in.spans[i] != nil {
				return nil, &sasscommon.ArgumentError{Name: "spans", Message: fmt.Sprintf("spans may not have a value for string elements (at index %d)", i)}
			}
		} else if _, ok := content.(Expression); ok {
			if in.spans[i] == nil {
				return nil, &sasscommon.ArgumentError{Name: "spans", Message: fmt.Sprintf("spans must have a value for expression elements (at index %d)", i)}
			}
		} else {
			return nil, &sasscommon.ArgumentError{Name: "contents", Message: fmt.Sprintf("contents may only contain strings or expressions, got %T", content)}
		}
	}

	return in, nil
}

func (in *Interpolation) Span() (sasscommon.FileSpan, error) { return in.span, nil }
func (in *Interpolation) IsSassNode()                        {}
func (in *Interpolation) IsAstNode()                         {}

// Spans returns the span for each element in Contents. The span is nil for
// string elements and non-nil for Expression elements, where it covers the
// whole `#{...}` rather than just the expression text. Dart marks this
// @internal; callers needing a covering span should prefer SpanForElement.
func (in *Interpolation) Spans() []*sasscommon.FileSpan { return in.spans }

// IsPlain returns whether this contains no interpolated expressions.
func (in *Interpolation) IsPlain() bool {
	return in.AsPlain() != nil
}

// AsPlain returns the text contents if this contains no interpolated
// expressions, or nil otherwise.
//
// An empty interpolation counts as plain and yields the empty string; a
// single string element yields its text. Anything containing an expression
// yields nil, which is what IsPlain tests.
// (Dart: Interpolation.asPlain.)
func (in *Interpolation) AsPlain() *string {
	switch {
	case len(in.Contents) == 0:
		s := ""
		return &s
	case len(in.Contents) == 1:
		if s, ok := in.Contents[0].(string); ok {
			return &s
		}
	}
	return nil
}

// InitialPlain returns the plain text before the first interpolation, or the
// empty string.
//
// Only a leading string element contributes; text after the first expression
// is not included. Dart marks this @internal.
func (in *Interpolation) InitialPlain() string {
	if len(in.Contents) > 0 {
		if s, ok := in.Contents[0].(string); ok {
			return s
		}
	}
	return ""
}

// SpanForElement returns the FileSpan covering the element at index.
//
// For expressions this is the stored `#{...}` span, which may in constructed
// (non-source) cases collapse to just the expression span. For strings there
// is no stored span, so one is derived from the neighbors: from the overall
// start (or the previous expression's end) to the next expression's start (or
// the overall end). That derived range includes a surrounding quote for text
// at the edges of quoted strings; quotes are never included for expressions.
// (Dart: Interpolation.spanForElement.)
func (in *Interpolation) SpanForElement(index int) (sasscommon.FileSpan, error) {
	switch in.Contents[index].(type) {
	case string:
		var start, end sasscommon.SourceLocation
		var err error
		if index == 0 {
			start, err = in.span.StartLocation()
			if err != nil {
				return nil, err
			}
		} else {
			start, err = (*in.spans[index-1]).EndLocation()
			if err != nil {
				return nil, err
			}
		}
		if index+1 == len(in.spans) {
			end, err = in.span.EndLocation()
			if err != nil {
				return nil, err
			}
		} else {
			end, err = (*in.spans[index+1]).StartLocation()
			if err != nil {
				return nil, err
			}
		}
		file, err := in.span.File()
		if err != nil {
			return nil, err
		}
		span := sasscommon.NewFileSpan(file, start.Offset, end.Offset)
		return span, nil
	default:
		return *in.spans[index], nil
	}
}

// String renders the interpolation back to source form: plain text verbatim
// and each expression wrapped in `#{...}`. (Dart: Interpolation.toString.)
func (in *Interpolation) String() (string, error) {
	parts := make([]string, len(in.Contents))
	for i, content := range in.Contents {
		switch v := content.(type) {
		case string:
			parts[i] = v
		default:
			s, err := SprintAny(v)
			if err != nil {
				return "", err
			}
			parts[i] = "#{" + s + "}"
		}
	}
	return strings.Join(parts, ""), nil
}
