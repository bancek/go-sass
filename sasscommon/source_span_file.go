// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span/lib/src/file.dart

import (
	"fmt"
	"net/url"
	"strings"
)

// FileSpan unifies three Dart types into one Go interface: FileSpan plus
// SourceSpanWithContext plus the SourceSpan base. Go interfaces are
// satisfied implicitly, so a single interface can carry the whole set
// (location accessors, text, highlighting, subspan algebra) that Dart
// spreads across its class hierarchy; both SimpleFileSpan and
// SourceSpanWithContext-shaped values satisfy it.
//
// File and Context below correspond to Dart's FileSpan.file and
// SourceSpanWithContext.context respectively.
//
// Matches Dart: FileSpan + SourceSpanWithContext + SourceSpan (package:source_span)
type FileSpan interface {
	Span
	SourceURL() (*url.URL, error)

	// File returns the FileSource this span belongs to.
	// Matches Dart: FileSpan.file
	File() (*FileSource, error)

	// Context returns the source text surrounding this span, trimmed to the
	// relevant line range.
	// Matches Dart: SourceSpanWithContext.context
	Context() (string, error)

	// Expand returns a span covering both this span and other. Unlike union,
	// the two spans may be disjoint, in which case the text between them is
	// covered too. Both spans must share a source URL.
	Expand(FileSpan) (FileSpan, error)
	// Subspan slices [start, end) bytes after the beginning of this span
	// into a new span. Out-of-range bounds report a RangeError, and a
	// full-range slice returns the span itself.
	Subspan(start, end int) (FileSpan, error)
	TrimRight() (FileSpan, error)
	Before(FileSpan) (FileSpan, error)
	After(FileSpan) (FileSpan, error)
	Between(FileSpan) (FileSpan, error)
	Contains(FileSpan) (bool, error)
	String() (string, error)

	// WithoutInitialAtRule returns a subspan excluding an initial at-rule and
	// any whitespace after it.
	// Matches Dart: SpanExtensions.withoutInitialAtRule
	WithoutInitialAtRule() (FileSpan, error)

	// InitialQuoted returns the span of the quoted text at the start of this
	// span. This span must start with " or '.
	// Matches Dart: SpanExtensions.initialQuoted
	InitialQuoted() (FileSpan, error)

	// InitialIdentifier returns the span of the identifier at the start of this
	// span. If includeLeading is greater than 0, that many additional characters
	// are included from the start before looking for an identifier.
	// Matches Dart: SpanExtensions.initialIdentifier
	InitialIdentifier(includeLeading int) (FileSpan, error)

	// Highlight highlights this span with optional ANSI terminal colors.
	// Matches Dart's SourceSpan.highlight(color:).
	Highlight(opts HighlightOptions) (string, error)

	// HighlightMultiple highlights this span as the primary span along with
	// secondarySpans. Matches Dart's SourceSpanExtension.highlightMultiple.
	HighlightMultiple(primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error)

	// MessageMultiple is like HighlightMultiple but prepends a message header.
	// Matches Dart's SourceSpanExtension.messageMultiple.
	MessageMultiple(message, primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error)
}

// SimpleFileSpan is the concrete FileSpan, porting Dart's private _FileSpan:
// a file plus start/end character offsets, with locations, text, and context
// derived lazily from the file so spans stay cheap to allocate.
//
// Offsets are the single source of truth, exactly as in Dart where start
// and end locations are generated from _start/_end on demand:
//
//	Dart _FileSpan        → Go SimpleFileSpan
//	file (SourceFile)     → file (*FileSource)
//	_start (int offset)   → start (int)
//	_end (int offset)     → end (int)
//	start → FileLocation  → StartLocation() → file.Location(start) (lazy, like Dart)
//	end → FileLocation    → EndLocation()   → file.Location(end)   (lazy, like Dart)
//	text → getText(...)   → SpanText()      → file.GetText(...)   (lazy, like Dart)
type SimpleFileSpan struct {
	file  *FileSource // Matches _FileSpan.file (SourceFile)
	start int         // Matches _FileSpan._start (int offset)
	end   int         // Matches _FileSpan._end (int offset)
}

var (
	_ Span     = SimpleFileSpan{}
	_ FileSpan = (*SimpleFileSpan)(nil)
)

// NewSimpleFileSpan creates a SimpleFileSpan covering [start, end) in file.
// Bounds are validated on use; prefer NewFileSpanInFile at call sites that
// want the range spelled out.
//
// Matches Dart: _FileSpan constructor (via SourceFile.span)
func NewSimpleFileSpan(file *FileSource, start, end int) SimpleFileSpan {
	return SimpleFileSpan{file: file, start: start, end: end}
}

// NewFileSpan creates a FileSpan covering [start, end) in file. It is the
// interface-returning counterpart to NewSimpleFileSpan for call sites that
// trade on FileSpan.
//
// Matches Dart: SourceFile.span(start, [end])
func NewFileSpan(file *FileSource, start, end int) FileSpan {
	return SimpleFileSpan{file: file, start: start, end: end}
}

// NewFileSpanInFile creates a FileSpan spanning [startOffset, endOffset) in
// file. Offsets are character positions in the file text, matching the units
// Dart's SourceFile.span takes.
//
// Matches Dart: SourceFile.span(start, [end])
func NewFileSpanInFile(file *FileSource, startOffset, endOffset int) FileSpan {
	return SimpleFileSpan{file: file, start: startOffset, end: endOffset}
}

// File returns the source file this span was sliced from, or nil for the
// URL-less bogus span.
//
// Matches _FileSpan.file
func (s SimpleFileSpan) File() (*FileSource, error) { return s.file, nil }

// SourceURL returns the URL of the source file, or nil when the file (or
// its URL) is unknown.
//
// Matches _FileSpan.sourceUrl
func (s SimpleFileSpan) SourceURL() (*url.URL, error) {
	if s.file == nil {
		return nil, nil
	}
	return s.file.URL(), nil
}

// SpanText returns the source text covered by this span. A detached span
// (nil file) or an empty range yields "", so bogus spans render harmlessly.
//
// Matches _FileSpan.text
func (s SimpleFileSpan) SpanText() (string, error) {
	if s.file == nil || s.start >= s.end {
		return "", nil
	}
	return s.file.GetText(s.start, s.end), nil
}

// StartLocation resolves the span start to a line/column location via the
// file; a detached span yields the zero location. Locations are computed on
// demand rather than stored, as in Dart.
//
// Matches _FileSpan.start → FileLocation
func (s SimpleFileSpan) StartLocation() (SourceLocation, error) {
	if s.file == nil {
		return SourceLocation{}, nil
	}
	return s.file.Location(s.start), nil
}

// EndLocation resolves the span end (exclusive) to a line/column location
// via the file; a detached span yields the zero location.
//
// Matches _FileSpan.end → FileLocation
func (s SimpleFileSpan) EndLocation() (SourceLocation, error) {
	if s.file == nil {
		return SourceLocation{}, nil
	}
	return s.file.Location(s.end), nil
}

// Length returns the span length in characters (end minus start).
//
// Matches _FileSpan.length
func (s SimpleFileSpan) Length() (int, error) { return s.end - s.start, nil }

// Context returns the source lines surrounding this span: the full line the
// span starts on through the full line the span ends on.
//
// The end boundary needs three cases, ported from Dart: a span ending at
// column 0 of a later line covers that line's preceding newline, so context
// stops at the span end itself (except point spans, which instead show the
// following line, or "" at end of file); a span ending on the last line runs
// to end of file; otherwise context runs through the end line's newline.
//
// Matches _FileSpan.context
func (s SimpleFileSpan) Context() (string, error) {
	if s.file == nil {
		return "", nil
	}
	endLine := s.file.GetLine(s.end)
	endColumn := s.file.GetColumn(s.end)

	var endOffset int
	if endColumn == 0 && endLine != 0 {
		if s.end == s.start {
			if endLine == s.file.Lines()-1 {
				return "", nil
			}
			return s.file.GetText(
				s.file.GetOffset(endLine),
				s.file.GetOffset(endLine+1)), nil
		}
		endOffset = s.end
	} else if endLine == s.file.Lines()-1 {
		endOffset = s.file.Length()
	} else {
		endOffset = s.file.GetOffset(endLine + 1)
	}

	return s.file.GetText(
		s.file.GetOffset(s.file.GetLine(s.start)),
		endOffset), nil
}

// String returns the span text, satisfying the printable-span convention
// Dart's SourceSpan.toString follows.
func (s SimpleFileSpan) String() (string, error) { return s.SpanText() }

// Expand returns a span covering both this span and other, filling any gap
// between them. The two spans must share a source URL; a mismatch reports
// an ArgumentError naming both URLs, as in Dart.
//
// Unlike union-style merges, disjoint spans are allowed here: the text
// between them is covered by the result.
//
// Matches _FileSpan.expand
func (s SimpleFileSpan) Expand(other FileSpan) (FileSpan, error) {
	otherURL, err := other.SourceURL()
	if err != nil {
		return nil, err
	}
	sURL, err := s.SourceURL()
	if err != nil {
		return nil, err
	}
	if !sameURL(sURL, otherURL) {
		return nil, &ArgumentError{Message: fmt.Sprintf("Source URLs %q and %q don't match.", sURL, otherURL)}
	}
	start := s.start
	end := s.end
	otherStart, err := other.StartLocation()
	if err != nil {
		return nil, err
	}
	if otherStart.Offset < start {
		start = otherStart.Offset
	}
	otherEnd, err := other.EndLocation()
	if err != nil {
		return nil, err
	}
	if otherEnd.Offset > end {
		end = otherEnd.Offset
	}
	return SimpleFileSpan{file: s.file, start: start, end: end}, nil
}

// sameURL reports whether a and b name the same source: two nil URLs count
// as the same (both unknown), while a nil/non-nil pair never matches.
// Comparison is by URL string, so distinct URL values with equal text agree.
func sameURL(a, b *url.URL) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.String() == b.String()
}

// Subspan slices [start, end) offsets after the beginning of this span.
// Bounds outside the span report a RangeError; a full-range slice returns
// the span itself, preserving any wrapper identity, as in Dart.
//
// Matches Dart: _FileSpan.subspan / FileSpanExtension.subspan
func (s SimpleFileSpan) Subspan(start, end int) (FileSpan, error) {
	length, err := s.Length()
	if err != nil {
		return nil, err
	}
	if start < 0 || end > length || start > end {
		return nil, &RangeError{Message: fmt.Sprintf("Invalid subspan range: [%d, %d) in span of length %d", start, end, length)}
	}
	if start == 0 && end == length {
		return s, nil
	}
	return SimpleFileSpan{
		file:  s.file,
		start: s.start + start,
		end:   s.start + end,
	}, nil
}

// Message formats message as "line L, column C [of url]: message" (1-based,
// with the human-readable URL form) and appends the span highlight when one
// renders. Location and URL lookup errors propagate instead of substituting
// placeholder text, so callers never report a bogus position.
//
// Matches Dart: SourceSpan.message
func (s SimpleFileSpan) Message(message string, opts HighlightOptions) (string, error) {
	var buf strings.Builder
	loc, err := s.StartLocation()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&buf, "line %d, column %d", loc.Line+1, loc.Column+1)
	sourceURL, err := s.SourceURL()
	if err != nil {
		return "", err
	}
	if sourceURL != nil {
		fmt.Fprintf(&buf, " of %s", PrettyUri(sourceURL))
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

// Highlight renders the span's source excerpt with optional ANSI color,
// delegating to the shared highlighter. It is Message without the
// "line/column/message" header.
//
// Matches Dart's SourceSpan.highlight(color:).
func (s SimpleFileSpan) Highlight(opts HighlightOptions) (string, error) {
	h, err := NewHighlighter(s, opts)
	if err != nil {
		return "", err
	}
	return h.Highlight()
}

// HighlightMultiple renders this span as the primary span together with
// secondarySpans, each shown under its label. This is the multi-span
// counterpart to Highlight; MessageMultiple adds the header on top.
//
// Matches Dart: SourceSpanExtension.highlightMultiple
func (s SimpleFileSpan) HighlightMultiple(primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	h, err := NewHighlighterMultiple(s, primaryLabel, secondarySpans, opts)
	if err != nil {
		return "", err
	}
	return h.Highlight()
}

// MessageMultiple renders a "line L, column C [of url]: message" header
// followed by the primary-plus-secondary highlight. It is HighlightMultiple
// with the same header Message prepends.
//
// Matches Dart: SourceSpanExtension.messageMultiple
func (s SimpleFileSpan) MessageMultiple(message, primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	var buf strings.Builder
	startLoc, err := s.StartLocation()
	if err != nil {
		return "", err
	}
	buf.WriteString(fmt.Sprintf("line %d, column %d", startLoc.Line+1, startLoc.Column+1))
	sourceURL, err := s.SourceURL()
	if err != nil {
		return "", err
	}
	if sourceURL != nil {
		fmt.Fprintf(&buf, " of %s", PrettyUri(sourceURL))
	}
	fmt.Fprintf(&buf, ": %s\n", message)
	highlight, err := s.HighlightMultiple(primaryLabel, secondarySpans, opts)
	if err != nil {
		return "", err
	}
	buf.WriteString(highlight)
	return buf.String(), nil
}
