// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span/lib/src/span_with_context.dart

import (
	"fmt"
	"net/url"
	"strings"
)

// SourceSpanWithContext is a span with explicit text and context strings,
// matching Dart's SourceSpanWithContext.
//
// Context is the text around the span, including the line containing it. It
// must contain the span text, and the text must start at start.Column within
// some line of the context (enforced by NewSourceSpanWithContext, matching
// Dart's constructor checks).
//
// Unlike SimpleFileSpan (which computes Context() from its FileSource),
// this type stores context string explicitly and returns nil from File().
// It is used by the highlighter's normalization pipeline: after
// normalizeContext, all spans are SourceSpanWithContext so they carry
// their own context string without a FileSource back-reference.
//
// Dart's _FileSpan implements SourceSpanWithContext directly (one type).
// Go needs two types because SimpleFileSpan's Context() is computed from
// FileSource (optimized for real spans), while SourceSpanWithContext stores
// a literal context string (used by the highlighter's synthetic spans).
type SourceSpanWithContext struct {
	start   SourceLocation
	end     SourceLocation
	text    string
	context string
	url     *url.URL
}

var _ FileSpan = SourceSpanWithContext{}

// NewSourceSpanWithContext creates a SourceSpanWithContext from start to end
// (exclusive) containing text in the given context.
//
// Matches Dart: the SourceSpanWithContext constructor
// (span_with_context.dart). The context must contain the text, and the text
// must start at start.Column in a line within the context (located with
// findLineStart, matching Dart's check). Go shift: Dart throws
// ArgumentError on violation; Go returns *ArgumentError. Dart's base-class
// same-URL check has no counterpart here because Go's SourceLocation
// carries no URL — the span URL travels in the separate sourceURL argument.
func NewSourceSpanWithContext(start, end SourceLocation, text, context string, sourceURL *url.URL) (SourceSpanWithContext, error) {
	if !strings.Contains(context, text) {
		return SourceSpanWithContext{}, &ArgumentError{Message: "The context line must contain the span text."}
	}
	if _, ok := findLineStart(context, text, start.Column); !ok {
		return SourceSpanWithContext{}, &ArgumentError{Message: "The span text must start at column " + fmt.Sprint(start.Column) + " in a line within the context."}
	}
	return SourceSpanWithContext{start: start, end: end, text: text, context: context, url: sourceURL}, nil
}

// File returns nil: a context span carries its own context string and has
// no FileSource backing, unlike SimpleFileSpan (whose File is Dart's
// FileSpan.file).
func (s SourceSpanWithContext) File() (*FileSource, error) { return nil, nil }

// SourceURL returns the URL of the source, which may be nil when unknown.
// Matches Dart: SourceSpan.sourceUrl (span.dart).
func (s SourceSpanWithContext) SourceURL() (*url.URL, error) { return s.url, nil }

// SpanText returns the source text for this span.
// Matches Dart: SourceSpan.text (span.dart).
func (s SourceSpanWithContext) SpanText() (string, error) { return s.text, nil }

// Context returns the text around the span, including its line.
// Matches Dart: SourceSpanWithContext.context (span_with_context.dart).
func (s SourceSpanWithContext) Context() (string, error) { return s.context, nil }

// StartLocation returns the start location of this span.
// Matches Dart: SourceSpan.start (span.dart).
func (s SourceSpanWithContext) StartLocation() (SourceLocation, error) { return s.start, nil }

// EndLocation returns the end location of this span, exclusive.
// Matches Dart: SourceSpan.end (span.dart).
func (s SourceSpanWithContext) EndLocation() (SourceLocation, error) { return s.end, nil }

// Length returns the length of this span: end offset minus start offset.
// Matches Dart: SourceSpanMixin.length (span_mixin.dart).
func (s SourceSpanWithContext) Length() (int, error) { return s.end.Offset - s.start.Offset, nil }

// String returns the span text. (Dart's SourceSpanMixin.toString renders a
// "<type: from start to end text" summary instead; Go's String serves the
// Span interface's text accessor.)
func (s SourceSpanWithContext) String() (string, error) { return s.text, nil }

// Expand returns a new span covering both this span and other.
//
// Matches Dart: FileSpan.expand (file.dart). Unlike union, other may be
// disjoint from this span; the text between the two is covered by the
// result. Both spans must share a source URL, else *ArgumentError. Go shift:
// with no file backing, the result carries empty text and context rather
// than the file text between the endpoints.
func (s SourceSpanWithContext) Expand(other FileSpan) (FileSpan, error) {
	otherURL, err := other.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(s.url, otherURL) {
		return nil, &ArgumentError{Message: fmt.Sprintf("Source URLs %q and %q don't match.", s.url, otherURL)}
	}
	start := s.start
	end := s.end
	otherStart, err := other.StartLocation()
	if err != nil {
		return nil, err
	}
	if otherStart.Offset < start.Offset {
		start = otherStart
	}
	otherEnd, err := other.EndLocation()
	if err != nil {
		return nil, err
	}
	if otherEnd.Offset > end.Offset {
		end = otherEnd
	}
	return SourceSpanWithContext{start: start, end: end, text: "", context: "", url: s.url}, nil
}

// Subspan returns a span from subStart (inclusive) to subEnd (exclusive)
// past the beginning of this span.
//
// Matches Dart: SourceSpanWithContextExtension.subspan
// (span_with_context.dart). Out-of-range bounds report *RangeError, matching
// Dart's RangeError.checkValidRange, and a full-range subspan returns this
// span unchanged. Line and column of the new endpoints advance past each
// newline in the skipped text, matching Dart's shared subspanLocations
// helper (utils.dart).
func (s SourceSpanWithContext) Subspan(subStart, subEnd int) (FileSpan, error) {
	if subStart < 0 || subEnd > len(s.text) || subStart > subEnd {
		return nil, &RangeError{Message: "Subspan out of bounds"}
	}
	if subStart == 0 && subEnd == len(s.text) {
		return s, nil
	}
	return SourceSpanWithContext{
		start: SourceLocation{
			Offset: s.start.Offset + subStart,
			Line:   s.start.Line + strings.Count(s.text[:subStart], "\n"),
			Column: columnAt(s.text, subStart),
		},
		end: SourceLocation{
			Offset: s.start.Offset + subEnd,
			Line:   s.start.Line + strings.Count(s.text[:subEnd], "\n"),
			Column: columnAt(s.text, subEnd),
		},
		text:    s.text[subStart:subEnd],
		context: s.context,
		url:     s.url,
	}, nil
}

// columnAt returns the column of offset within text: the distance past the
// last newline at or before offset, or offset itself when text has no
// newline. This folds Dart's subspanLocations line/column walk (utils.dart)
// into a direct computation for spans whose text is fully in hand.
func columnAt(text string, offset int) int {
	lastNewline := strings.LastIndex(text[:offset], "\n")
	if lastNewline < 0 {
		return offset
	}
	return offset - lastNewline - 1
}

// TrimRight returns this span with all trailing whitespace trimmed.
//
// Matches Dart: SpanExtensions.trimRight (dart-sass lib/src/util/span.dart).
// A span with no trailing whitespace returns itself unchanged. This mirrors
// SimpleFileSpan.TrimRight (util_span.go owns the FileSpan-backed logic);
// this copy serves highlighter-synthesized spans with no file.
func (s SourceSpanWithContext) TrimRight() (FileSpan, error) {
	text := s.text
	end := len(text)
	for end > 0 && isWhitespaceByte(text[end-1]) {
		end--
	}
	if end == len(text) {
		return s, nil
	}
	newEnd := SourceLocation{
		Offset: s.end.Offset - (len(text) - end),
		Line:   s.end.Line,
		Column: columnAt(s.text, end),
	}
	return SourceSpanWithContext{start: s.start, end: newEnd, text: text[:end], context: s.context, url: s.url}, nil
}

// TrimLeft returns this span with all leading whitespace trimmed.
//
// Matches Dart: SpanExtensions.trimLeft (dart-sass lib/src/util/span.dart).
// A span with no leading whitespace returns itself unchanged. This mirrors
// SimpleFileSpan.TrimLeft (util_span.go owns the FileSpan-backed logic);
// this copy serves highlighter-synthesized spans with no file.
func (s SourceSpanWithContext) TrimLeft() (FileSpan, error) {
	text := s.text
	start := 0
	for start < len(text) && isWhitespaceByte(text[start]) {
		start++
	}
	if start == 0 {
		return s, nil
	}
	return s.Subspan(start, len(text))
}

// Trim returns this span with all whitespace trimmed from both sides.
//
// Matches Dart: SpanExtensions.trim (dart-sass lib/src/util/span.dart),
// which folds trimLeft/trimRight the same way.
func (s SourceSpanWithContext) Trim() (FileSpan, error) {
	result, err := s.TrimLeft()
	if err != nil {
		return nil, err
	}
	return result.TrimRight()
}

// Before returns a span covering the text from the beginning of this span
// to the beginning of sub.
//
// Matches Dart: SpanExtensions.before (dart-sass lib/src/util/span.dart).
// Reports *ArgumentError if sub is in a different file or is not fully
// within this span. A zero-width prefix yields the zero
// SourceSpanWithContext rather than a file-backed span, since there is no
// FileSource to slice.
func (s SourceSpanWithContext) Before(sub FileSpan) (FileSpan, error) {
	subURL, err := sub.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(s.url, subURL) {
		return nil, &ArgumentError{Message: "different files"}
	}
	subStart, err := sub.StartLocation()
	if err != nil {
		return nil, err
	}
	subEnd, err := sub.EndLocation()
	if err != nil {
		return nil, err
	}
	if subStart.Offset < s.start.Offset || subEnd.Offset > s.end.Offset {
		return nil, &ArgumentError{Message: "not contained"}
	}
	offset := subStart.Offset - s.start.Offset
	if offset <= 0 || offset > len(s.text) {
		return SourceSpanWithContext{}, nil
	}
	return SourceSpanWithContext{
		start:   s.start,
		end:     subStart,
		text:    s.text[:offset],
		context: s.context,
		url:     s.url,
	}, nil
}

// After returns a span covering the text from the end of sub to the end of
// this span.
//
// Matches Dart: SpanExtensions.after (dart-sass lib/src/util/span.dart).
// Reports *ArgumentError if sub is in a different file or is not fully
// within this span. A zero-width suffix yields the zero
// SourceSpanWithContext rather than a file-backed span, since there is no
// FileSource to slice.
func (s SourceSpanWithContext) After(sub FileSpan) (FileSpan, error) {
	subURL, err := sub.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(s.url, subURL) {
		return nil, &ArgumentError{Message: "different files"}
	}
	subStart, err := sub.StartLocation()
	if err != nil {
		return nil, err
	}
	subEnd, err := sub.EndLocation()
	if err != nil {
		return nil, err
	}
	if subStart.Offset < s.start.Offset || subEnd.Offset > s.end.Offset {
		return nil, &ArgumentError{Message: "not contained"}
	}
	offset := subEnd.Offset - s.start.Offset
	if offset < 0 || offset >= len(s.text) {
		return SourceSpanWithContext{}, nil
	}
	return SourceSpanWithContext{
		start:   subEnd,
		end:     s.end,
		text:    s.text[offset:],
		context: s.context,
		url:     s.url,
	}, nil
}

// Between returns a span covering the text after this span and before
// other.
//
// Matches Dart: SpanExtensions.between (dart-sass lib/src/util/span.dart).
// Reports *ArgumentError if other is in a different file or starts before
// this span ends. The result carries empty text with this span's context,
// since the gap text is not re-sliced from a file.
func (s SourceSpanWithContext) Between(other FileSpan) (FileSpan, error) {
	otherURL, err := other.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(s.url, otherURL) {
		return nil, &ArgumentError{Message: "different files"}
	}
	otherStart, err := other.StartLocation()
	if err != nil {
		return nil, err
	}
	if s.end.Offset > otherStart.Offset {
		return nil, &ArgumentError{Message: "s isn't before other"}
	}
	return SourceSpanWithContext{
		start:   s.end,
		end:     otherStart,
		text:    "",
		context: s.context,
		url:     s.url,
	}, nil
}

// Contains reports whether this span contains the target span.
//
// Matches Dart: SpanExtensions.contains (dart-sass lib/src/util/span.dart):
// both spans must be in the same file (a different file reports false, not
// an error) and target must lie within this span's inclusive start/end
// range.
func (s SourceSpanWithContext) Contains(target FileSpan) (bool, error) {
	targetURL, err := target.SourceURL()
	if err != nil {
		return false, err
	}
	if !sameURL(s.url, targetURL) {
		return false, nil
	}
	targetStart, err := target.StartLocation()
	if err != nil {
		return false, err
	}
	if s.start.Offset > targetStart.Offset {
		return false, nil
	}
	targetEnd, err := target.EndLocation()
	if err != nil {
		return false, err
	}
	return s.end.Offset >= targetEnd.Offset, nil
}

// WithoutInitialAtRule returns a subspan excluding an initial at-rule and
// any whitespace after it.
//
// Matches Dart: SpanExtensions.withoutInitialAtRule (dart-sass
// lib/src/util/span.dart). The text must begin with "@"; otherwise a
// *ScanError is returned, matching Dart's expectChar failure. This mirrors
// SimpleFileSpan.WithoutInitialAtRule (util_span.go owns the
// FileSpan-backed logic); this copy serves highlighter-synthesized spans
// with no file.
func (s SourceSpanWithContext) WithoutInitialAtRule() (FileSpan, error) {
	text := s.text
	if len(text) == 0 || text[0] != '@' {
		return nil, &ScanError{Message: "Expected @.", Span: s}
	}
	pos, err := scanIdent(text, 1, s)
	if err != nil {
		return nil, err
	}
	if pos >= len(text) {
		return s, nil
	}
	result, err := s.Subspan(pos, len(text))
	if err != nil {
		return nil, err
	}
	result2, err := trimLeftSpan(result)
	if err != nil {
		return nil, err
	}
	return result2, nil
}

// InitialQuoted returns the span of the quoted text at the start of this
// span. This span must start with " or '.
//
// Matches Dart: SpanExtensions.initialQuoted (dart-sass
// lib/src/util/span.dart). Backslash escapes are skipped, so an escaped
// quote does not end the span; an unterminated quote returns a *ScanError,
// matching Dart's readChar failure. This mirrors
// SimpleFileSpan.InitialQuoted (util_span.go owns the FileSpan-backed
// logic); this copy serves highlighter-synthesized spans with no file.
func (s SourceSpanWithContext) InitialQuoted() (FileSpan, error) {
	text := s.text
	if len(text) == 0 || (text[0] != '"' && text[0] != '\'') {
		return nil, &ScanError{Message: "Expected quote.", Span: s}
	}
	quote := text[0]
	end := 1
	for end < len(text) {
		if text[end] == quote {
			end++
			result, err := s.Subspan(0, end)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		if text[end] == '\\' {
			end++
			if end < len(text) {
				end++
			}
			continue
		}
		end++
	}
	return nil, &ScanError{Message: fmt.Sprintf("Expected %s.", string(quote)), Span: s}
}

// InitialIdentifier returns the span of the identifier at the start of this
// span. If includeLeading is greater than 0, that many additional
// characters are included from the start before looking for an identifier.
//
// Matches Dart: SpanExtensions.initialIdentifier (dart-sass
// lib/src/util/span.dart), whose StringScanner walk over name characters
// and escapes is folded into the scanIdent helper here. This mirrors
// SimpleFileSpan.InitialIdentifier (util_span.go owns the FileSpan-backed
// logic); this copy serves highlighter-synthesized spans with no file.
func (s SourceSpanWithContext) InitialIdentifier(includeLeading int) (FileSpan, error) {
	text := s.text
	pos := 0
	for i := 0; i < includeLeading; i++ {
		if pos >= len(text) {
			result, err := s.Subspan(0, pos)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		pos++
	}
	var err error
	pos, err = scanIdent(text, pos, s)
	if err != nil {
		return nil, err
	}
	result, err := s.Subspan(0, pos)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Message formats message in a human-friendly way associated with this
// span: "line X, column Y[ of url]: message" (1-based) followed by the
// highlighted span when the highlight is non-empty.
//
// Matches Dart: SourceSpanMixin.message (span_mixin.dart), which builds the
// same header and appends highlight() only when it is non-empty.
func (s SourceSpanWithContext) Message(message string, opts HighlightOptions) (string, error) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "line %d, column %d", s.start.Line+1, s.start.Column+1)
	if s.url != nil {
		fmt.Fprintf(&buf, " of %s", s.url.String())
	}
	fmt.Fprintf(&buf, ": %s", message)
	highlight, err := s.Highlight(opts)
	if err != nil {
		return "", err
	}
	if highlight != "" {
		buf.WriteByte('\n')
		buf.WriteString(highlight)
	}
	return buf.String(), nil
}

// Highlight prints the text associated with this span in a user-friendly
// way: identical to Message, except it omits the file name, line number,
// column number, and message.
//
// Matches Dart: SourceSpanMixin.highlight (span_mixin.dart). Dart returns
// an empty string for a zero-length span that is not a
// SourceSpanWithContext; this type always is one (its normalization
// supplies context), so it always renders through the highlighter.
func (s SourceSpanWithContext) Highlight(opts HighlightOptions) (string, error) {
	h, err := NewHighlighter(s, opts)
	if err != nil {
		return "", err
	}
	return h.Highlight()
}

// HighlightMultiple is like Highlight, but also highlights secondarySpans
// to give the user additional context. Each span takes a label (primaryLabel
// for this span, the map values for the secondary spans) indicating what
// that span represents.
//
// Matches Dart: SourceSpanExtension.highlightMultiple (span.dart). Color
// travels on opts (Dart's color/primaryColor/secondaryColor arguments);
// nil secondary spans are skipped, a Go tolerance with no Dart counterpart
// (Dart map keys cannot be null).
func (s SourceSpanWithContext) HighlightMultiple(primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	h, err := NewHighlighterMultiple(s, primaryLabel, secondarySpans, opts)
	if err != nil {
		return "", err
	}
	return h.Highlight()
}

// MessageMultiple is like Message, but also highlights secondarySpans to
// give the user additional context. Each span takes a label (primaryLabel
// for this span, the map values for the secondary spans) indicating what
// that span represents.
//
// Matches Dart: SourceSpanExtension.messageMultiple (span.dart): the same
// "line X, column Y[ of url]: message" header as Message, followed by the
// multi-span highlight.
func (s SourceSpanWithContext) MessageMultiple(message, primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("line %d, column %d", s.start.Line+1, s.start.Column+1))
	if s.url != nil {
		buf.WriteString(fmt.Sprintf(" of %s", s.url.String()))
	}
	buf.WriteString(fmt.Sprintf(": %s\n", message))
	highlight, err := s.HighlightMultiple(primaryLabel, secondarySpans, opts)
	if err != nil {
		return "", err
	}
	buf.WriteString(highlight)
	return buf.String(), nil
}
