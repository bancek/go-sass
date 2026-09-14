// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sourcemapbuffer

// dart-source: lib/src/util/source_map_buffer.dart
// dart-source: lib/src/util/no_source_map_buffer.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sourcemap"
)

// SourceMapBuffer is the interface for writing serialized CSS with optional
// source map tracking: a text buffer that maps output positions back to
// source spans as the serializer writes.
//
// Matches Dart: SourceMapBuffer (abstract interface in
// util/source_map_buffer.dart); NoSourceMapBuffer in
// util/no_source_map_buffer.dart provides the tracking-free implementation.
type SourceMapBuffer interface {
	// WriteString appends s to the buffer, tracking newlines for line and
	// column positions.
	WriteString(s string) (int, error)
	// WriteByte appends a single byte to the buffer, tracking newlines.
	WriteByte(c byte) error
	// WriteRune appends a rune to the buffer, tracking newlines.
	WriteRune(r rune) (int, error)
	// Write appends p to the buffer, tracking newlines.
	Write(p []byte) (int, error)
	// String returns the buffered text written so far.
	String() string
	// Len returns the length of the buffered text.
	Len() int
	// ForSpan records a mapping at the current position for the start of
	// span, then runs cb. Text written inside cb stays covered by the
	// span: span ends are deliberately not mapped, since browsers only
	// look up sources from targets and skipping ends halves map size.
	ForSpan(span sasscommon.FileSpan, cb func() error) error
	// BuildSourceMap produces the source map for the buffered text,
	// shifting every entry forward past prefix. The tracking-free
	// implementation reports an error instead.
	BuildSourceMap(prefix string) (*sourcemap.SingleMapping, error)
}
