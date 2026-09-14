// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span/lib/src/file.dart

import (
	"net/url"
	"unicode/utf8"
)

// FileSource represents a source file, matching Dart's SourceFile.
//
// Like SourceFile, this does not necessarily correspond to a file on disk,
// just a chunk of text usually with a URL associated with it. The URL may be
// nil, indicating that it is unknown or unavailable.
//
// Line starts are precomputed at construction: each entry is the offset of
// the first character after a newline, so a trailing newline leaves a final
// entry that is not actually in the file (matching Dart's _lineStarts).
//
// Key differences from Dart:
//   - Stores raw []byte (Dart uses Uint32List for code points). We convert
//     to string on demand via Text() for Text()/GetText() calls.
//   - No _cachedLine optimization. Dart caches the last GetLine result for
//     sequential-access patterns. Go's binary search is fast enough for
//     typical file sizes and avoids the complexity of mutable cache state.
//   - No decodedChars / codeUnits separation. Parsing is byte/UTF-8 based
//     (Go native), not code-point based.
//   - Offsets and Length count bytes, not characters. Dart's length counts
//     characters; user-facing columns convert via CharacterColumn.
type FileSource struct {
	url        *url.URL
	raw        []byte
	text       string
	textReady  bool
	lineStarts []int
}

// NewFileSource creates a new FileSource from raw bytes and a source URL.
//
// Matches Dart: SourceFile.fromString/_fromList (file.dart), which likewise
// precomputes the line-start table here rather than lazily. A carriage
// return not followed by a line feed counts as a newline, matching Dart's
// normalization in _fromList. A nil sourceURL marks the URL unknown,
// matching Dart's nullable url.
func NewFileSource(raw []byte, sourceURL *url.URL) *FileSource {
	lineStarts := []int{0}
	for i, b := range raw {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		} else if b == '\r' && (i+1 >= len(raw) || raw[i+1] != '\n') {
			// A carriage return not followed by a line feed counts as a
			// newline of its own. Matches Dart: the CR normalization in
			// SourceFile._fromList (file.dart).
			lineStarts = append(lineStarts, i+1)
		}
	}
	return &FileSource{
		url:        sourceURL,
		raw:        raw,
		lineStarts: lineStarts,
	}
}

// URL returns the source URL.
//
// Matches Dart: SourceFile.url (file.dart). May be nil, indicating that the
// URL is unknown or unavailable.
func (f *FileSource) URL() *url.URL { return f.url }

// Text returns the full source as a string, caching the conversion.
//
// This is a Go-only convenience with no direct Dart counterpart (Dart
// callers read SourceFile.getText with no range instead). The conversion
// from the raw bytes is performed once and cached.
func (f *FileSource) Text() string {
	if !f.textReady {
		f.text = string(f.raw)
		f.textReady = true
	}
	return f.text
}

// Length returns the length of the source in bytes.
//
// Matches Dart: SourceFile.length (file.dart), except that Dart counts
// characters while Go counts bytes (see the FileSource doc). Character-based
// widths for display and source maps come from CharacterColumn and the
// highlighter instead.
func (f *FileSource) Length() int { return len(f.raw) }

// Lines returns the number of lines in the source.
//
// Matches Dart: SourceFile.lines (file.dart): the number of line-start
// entries, so text ending in a newline counts a final empty line.
func (f *FileSource) Lines() int { return len(f.lineStarts) }

// GetLine returns the 0-based line containing offset.
//
// Matches Dart: SourceFile.getLine (file.dart), including its binary search
// over the line-start table (Dart's _binarySearch). Go shift: the
// _cachedLine sequential-access fast path is dropped (see the FileSource
// doc), and out-of-range offsets clamp to the first/last line instead of
// throwing RangeError.
func (f *FileSource) GetLine(offset int) int {
	// Binary search for the first line start past offset, then step back
	// one. Matches Dart: SourceFile._binarySearch (file.dart), whose result
	// getLine decrements the same way.
	lo, hi := 0, len(f.lineStarts)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if f.lineStarts[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo-1 < 0 {
		return 0
	}
	return lo - 1
}

// GetColumn returns the 0-based column for offset.
//
// If line is provided, it is used instead of computing from offset, which
// avoids a second binary search when the caller already knows the line.
// Matches Dart: SourceFile.getColumn (file.dart). Go shift: Dart validates
// the offset and line with RangeError; Go indexes the line-start table
// directly, so callers must pass a valid line.
func (f *FileSource) GetColumn(offset int, line ...int) int {
	l := f.GetLine(offset)
	if len(line) > 0 {
		l = line[0]
	}
	return offset - f.lineStarts[l]
}

// GetOffset returns the byte offset of the start of line, plus column.
// Column defaults to 0.
//
// Matches Dart: SourceFile.getOffset (file.dart). Go shift: Dart throws
// RangeError for a negative line/column or a column past the end of the
// line; Go performs no validation.
func (f *FileSource) GetOffset(line int, column ...int) int {
	col := 0
	if len(column) > 0 {
		col = column[0]
	}
	return f.lineStarts[line] + col
}

// GetText returns the substring from start to end (exclusive).
// If end is omitted, returns to end of file.
//
// Matches Dart: SourceFile.getText (file.dart), which builds the result
// from its decoded characters; Go slices the cached Text instead.
func (f *FileSource) GetText(start int, end ...int) string {
	e := len(f.raw)
	if len(end) > 0 {
		e = end[0]
	}
	return f.Text()[start:e]
}

// Span creates a new FileSpan from start to end (exclusive).
// If end is omitted, the span runs to the end of the file.
//
// Matches Dart: SourceFile.span (file.dart), which returns a lazily-computed
// _FileSpan; Go returns a SimpleFileSpan over this file.
func (f *FileSource) Span(start int, end ...int) FileSpan {
	e := len(f.raw)
	if len(end) > 0 {
		e = end[0]
	}
	return SimpleFileSpan{file: f, start: start, end: e}
}

// Location creates a SourceLocation at the given offset.
//
// Matches Dart: SourceFile.location (file.dart). Dart returns a FileLocation
// whose line and column are computed lazily from the file; Go returns a
// SourceLocation value with line and column filled in eagerly.
func (f *FileSource) Location(offset int) SourceLocation {
	line := f.GetLine(offset)
	return SourceLocation{
		Offset: offset,
		Line:   line,
		Column: f.GetColumn(offset, line),
	}
}

// CharacterColumn returns the 0-based column of offset counted in Unicode
// characters. Dart's SourceLocation columns count UTF-16 code units; Go's
// offsets are byte-based, so user-facing locations and source map columns must
// convert (multibyte characters count once).
//
// This is only for values that cross the port boundary (error locations,
// source maps); internal byte math should keep using GetColumn.
func (f *FileSource) CharacterColumn(offset int) int {
	line := f.GetLine(offset)
	if offset > len(f.raw) {
		offset = len(f.raw)
	}
	lineStart := f.lineStarts[line]
	seg := f.Text()[lineStart:offset]
	// Fast path: ASCII prefixes count one character per byte.
	for i := 0; i < len(seg); i++ {
		if seg[i] >= 0x80 {
			return utf8.RuneCountInString(seg)
		}
	}
	return offset - lineStart
}
