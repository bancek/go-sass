// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span/lib/src/span.dart

// SourceLocation is a byte position in a source file plus its resolved
// 0-based line and column. It ports Dart's SourceLocation from
// package:source_span, where file-backed locations compute line/column
// lazily from the offset and the file contents.
//
// Matches Dart: SourceLocation (package:source_span/lib/src/location.dart)
type SourceLocation struct {
	// Offset is the 0-based character offset into the source.
	Offset int
	// Line is the 0-based line number containing Offset.
	Line int
	// Column is the 0-based column number of Offset within Line.
	Column int
}

// Span is the base interface for source ranges, porting Dart's SourceSpan:
// a start/end pair plus the covered text, with message and highlight
// rendering. FileSpan extends this with file-backed context, URLs, and the
// subspan algebra.
//
// Matches Dart: SourceSpan (package:source_span/lib/src/span.dart)
type Span interface {
	// StartLocation returns the inclusive start of the span.
	StartLocation() (SourceLocation, error)
	// EndLocation returns the exclusive end of the span.
	EndLocation() (SourceLocation, error)
	// SpanText returns the source text covered by the span.
	SpanText() (string, error)
	// Length returns the span length in characters.
	Length() (int, error)

	// Message renders "line L, column C [of url]: message" (1-based) and
	// appends the span highlight when one renders. It is the header form of
	// the highlight-only rendering; FileSpan adds the multi-span variants.
	// Matches Dart: SourceSpan.message
	Message(message string, opts HighlightOptions) (string, error)
}
