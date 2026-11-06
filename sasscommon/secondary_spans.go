// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

import "github.com/bancek/go-sass/orderedmap"

// SecondarySpan is one labeled secondary span in a multi-span error.
type SecondarySpan struct {
	// Span is the secondary source range.
	Span FileSpan
	// Label describes what the range represents.
	Label string
}

// SecondarySpans is an insertion-ordered map from secondary spans to labels,
// matching Dart's linked-hash-map behavior for multi-span errors.
//
// Assigning a label to an existing span updates it in place; a new span is
// appended. The highlighter sorts spans by source position, preserving this
// insertion order for spans that start at the same offset.
type SecondarySpans = orderedmap.LinkedMap[FileSpan, string]

// NewSecondarySpans creates an ordered secondary-span map from spans.
func NewSecondarySpans(spans ...SecondarySpan) *SecondarySpans {
	result := orderedmap.NewWithCapacity[FileSpan, string](len(spans))
	for _, span := range spans {
		result.Put(span.Span, span.Label)
	}
	return result
}
