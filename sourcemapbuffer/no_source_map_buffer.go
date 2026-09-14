// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sourcemapbuffer

// dart-source: lib/src/util/no_source_map_buffer.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sourcemap"
)

// NoSourceMapBuffer is a SourceMapBuffer that doesn't actually build a source
// map. It's used when source map generation is disabled: writes buffer text
// only, span callbacks run untracked, and building a map reports an error.
//
// Matches Dart: NoSourceMapBuffer (util/no_source_map_buffer.dart)
type NoSourceMapBuffer struct {
	// buf holds the target text written so far.
	buf strings.Builder
}

// NewNoSourceMapBuffer creates a new NoSourceMapBuffer.
//
// Matches Dart: NoSourceMapBuffer constructor (util/no_source_map_buffer.dart)
func NewNoSourceMapBuffer() *NoSourceMapBuffer {
	return &NoSourceMapBuffer{}
}

// WriteString appends s to the buffer without any position tracking.
//
// Matches Dart: NoSourceMapBuffer.write
func (b *NoSourceMapBuffer) WriteString(s string) (int, error) {
	return b.buf.WriteString(s)
}

// WriteByte appends a single byte to the buffer without any position tracking.
//
// Matches Dart: NoSourceMapBuffer.writeCharCode
func (b *NoSourceMapBuffer) WriteByte(c byte) error {
	return b.buf.WriteByte(c)
}

// WriteRune appends a rune to the buffer without any position tracking.
//
// Matches Dart: NoSourceMapBuffer.writeCharCode
func (b *NoSourceMapBuffer) WriteRune(r rune) (int, error) {
	return b.buf.WriteRune(r)
}

// Write appends p to the buffer without any position tracking.
//
// Matches Dart: NoSourceMapBuffer.write
func (b *NoSourceMapBuffer) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

// String returns the buffered text written so far.
//
// Matches Dart: NoSourceMapBuffer.toString
func (b *NoSourceMapBuffer) String() string {
	return b.buf.String()
}

// Len returns the length of the buffered text.
//
// Matches Dart: NoSourceMapBuffer.length
func (b *NoSourceMapBuffer) Len() int {
	return b.buf.Len()
}

// ForSpan runs cb without recording any mapping: with no map under
// construction there is nothing to associate the span with.
//
// Matches Dart: NoSourceMapBuffer.forSpan
func (b *NoSourceMapBuffer) ForSpan(span sasscommon.FileSpan, cb func() error) error {
	return cb()
}

// BuildSourceMap always reports an error: a buffer that tracks nothing
// has no map to build.
//
// Matches Dart: NoSourceMapBuffer.buildSourceMap (throwing UnsupportedError)
func (b *NoSourceMapBuffer) BuildSourceMap(prefix string) (*sourcemap.SingleMapping, error) {
	return nil, &sasscommon.UnsupportedError{Message: "NoSourceMapBuffer.buildSourceMap() is not supported."}
}
