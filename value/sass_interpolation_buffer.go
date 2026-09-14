// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/interpolation_buffer.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// InterpolationBuffer builds an Interpolation iteratively.
// Add text via Write/WriteCharCode/Writeln/WriteAll, and expressions via Add.
// Once done, call Interpolation() to build the result.
//
// Plain text accumulates in an internal text buffer and is only flushed into
// the contents list when an expression arrives (or at build time), which is
// what keeps adjacent strings from ever appearing in contents.
type InterpolationBuffer struct {
	text     strings.Builder
	contents []any                  // strings and Expressions
	spans    []*sasscommon.FileSpan // nil for strings, non-nil for expressions
}

func (b *InterpolationBuffer) Write(obj string) {
	// Dart accepts Object? because toString() is infallible. Go takes string
	// because Value/Expression String() returns (string, error). Callers must
	// resolve String() errors before calling Write.
	b.text.WriteString(obj)
}

func (b *InterpolationBuffer) WriteAll(objects []string, separator *string) {
	// Dart accepts Iterable<Object?> because toString() is infallible. Go
	// takes []string — callers must resolve String() errors before calling
	// WriteAll.
	sep := ""
	if separator != nil {
		sep = *separator
	}
	for i, obj := range objects {
		if i > 0 {
			b.text.WriteString(sep)
		}
		b.text.WriteString(obj)
	}
}

func (b *InterpolationBuffer) WriteCharCode(ch int) {
	b.text.WriteRune(rune(ch))
}

func (b *InterpolationBuffer) Writeln(obj string) {
	// Dart accepts Object? because toString() is infallible. Go takes string
	// — callers must resolve String() errors before calling Writeln.
	b.text.WriteString(obj)
	b.text.WriteByte('\n')
}

// IsEmpty returns whether this buffer has no contents: neither flushed
// elements nor pending text. (Dart: InterpolationBuffer.isEmpty.)
func (b *InterpolationBuffer) IsEmpty() bool {
	return len(b.contents) == 0 && b.text.Len() == 0
}

// TrailingString returns the substring of the buffer after the last interpolation.
//
// This is the still-unflushed text: reading it does not flush, so a later Add
// still splits the text at the right boundary. (Dart: trailingString.)
func (b *InterpolationBuffer) TrailingString() string {
	return b.text.String()
}

// Clear empties this buffer.
//
// Contents, expression spans, and pending text are all reset, so unlike a
// fresh buffer nothing stale can pair with later additions.
func (b *InterpolationBuffer) Clear() {
	b.contents = nil
	b.spans = nil
	b.text.Reset()
}

// Add adds an expression with its span (cover from #{ through }).
//
// Pending text flushes first so the expression lands after it, preserving the
// no-adjacent-strings invariant. (Dart: InterpolationBuffer.add.)
func (b *InterpolationBuffer) Add(expression Expression, span sasscommon.FileSpan) {
	b.flushText()
	b.contents = append(b.contents, expression)
	b.spans = append(b.spans, &span)
}

// AddInterpolation merges the contents of an Interpolation into this buffer.
//
// A leading string folds back into the pending text and a trailing string is
// pulled back into the pending text after flushing, so merging never creates
// adjacent strings in contents. An empty interpolation is a no-op.
// (Dart: InterpolationBuffer.addInterpolation.)
func (b *InterpolationBuffer) AddInterpolation(interpolation *Interpolation) {
	if len(interpolation.Contents) == 0 {
		return
	}

	toAdd := interpolation.Contents
	spansToAdd := interpolation.Spans()

	if first, ok := toAdd[0].(string); ok {
		b.text.WriteString(first)
		toAdd = toAdd[1:]
		spansToAdd = spansToAdd[1:]
	}

	b.flushText()
	b.contents = append(b.contents, toAdd...)
	b.spans = append(b.spans, spansToAdd...)

	if len(b.contents) > 0 {
		if last, ok := b.contents[len(b.contents)-1].(string); ok {
			b.contents = b.contents[:len(b.contents)-1]
			b.spans = b.spans[:len(b.spans)-1]
			b.text.WriteString(last)
		}
	}
}

// flushText flushes text buffer to contents if non-empty.
//
// Empty text appends nothing (with a nil span for each real string), which is
// what keeps contents free of empty-string elements.
func (b *InterpolationBuffer) flushText() {
	if b.text.Len() == 0 {
		return
	}
	b.contents = append(b.contents, b.text.String())
	b.spans = append(b.spans, nil)
	b.text.Reset()
}

// Interpolation creates an Interpolation from accumulated contents.
//
// Pending text joins the contents before building, and a lone-string result
// takes the plain fast path (NewInterpolationPlain) rather than the general
// validated constructor. (Dart: InterpolationBuffer.interpolation.)
func (b *InterpolationBuffer) Interpolation(span sasscommon.FileSpan) (*Interpolation, error) {
	var contents []any
	var spans []*sasscommon.FileSpan

	contents = append(contents, b.contents...)
	spans = append(spans, b.spans...)
	if b.text.Len() > 0 {
		contents = append(contents, b.text.String())
		spans = append(spans, nil)
	}

	if len(contents) == 1 {
		if s, ok := contents[0].(string); ok {
			return NewInterpolationPlain(s, span), nil
		}
	}

	return NewInterpolation(contents, spans, span)
}

// String renders the buffered contents back to source form, like
// Interpolation.String: plain text verbatim, expressions wrapped in `#{...}`,
// plus any still-pending trailing text. (Dart: InterpolationBuffer.toString.)
func (b *InterpolationBuffer) String() (string, error) {
	var buf strings.Builder
	for _, element := range b.contents {
		if s, ok := element.(string); ok {
			buf.WriteString(s)
		} else {
			buf.WriteString("#{")
			s, err := SprintAny(element)
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
			buf.WriteByte('}')
		}
	}
	buf.WriteString(b.text.String())
	return buf.String(), nil
}
