// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span

import (
	"fmt"
)

// MultiSourceSpanFormatError pairs a primary span with secondary labeled
// spans for parse failures that need extra context, and also behaves as a
// format error by exposing the original source.
//
// It ports Dart's MultiSourceSpanFormatException from package:source_span:
// a MultiSourceSpanException (primaryLabel distinguishes the primary span,
// Secondary maps each extra span to its label) that is also a format error.
// The Sass-level MultiSpan*FormatException types in exception.go layer
// loaded URLs, traces, and CSS rendering on top of this shape.
//
// Matches Dart: MultiSourceSpanFormatException (package:source_span)
type MultiSourceSpanFormatError struct {
	// Message describes the failure.
	Message string
	// Span is the primary span the error points at.
	Span FileSpan
	// PrimaryLabel labels the primary span to distinguish it from Secondary.
	PrimaryLabel string
	// Source holds the original source text the spans were sliced from.
	Source string
	// Secondary maps secondary spans to labels.
	Secondary map[FileSpan]string
}

// Error renders "location: message" for the primary span's start; secondary
// spans surface through the HighlightMultiple/MessageMultiple paths rather
// than this plain rendering.
func (e *MultiSourceSpanFormatError) Error() string {
	loc, err := e.Span.StartLocation()
	if err != nil {
		return fmt.Sprintf("error getting start location: %v", err)
	}
	return fmt.Sprintf("%v: %s", loc, e.Message)
}
