// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sourcemapbuffer

// dart-source: lib/src/util/source_map_buffer.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sourcemap"
)

// DefaultSourceMapBuffer is a SourceMapBuffer that builds a source map while
// writing: it buffers the target text, tracks the current line and column,
// and records entries into the builder as spans open and lines pass.
//
// Matches Dart: SourceMapBuffer (util/source_map_buffer.dart)
type DefaultSourceMapBuffer struct {
	// buf holds the target text written so far.
	buf strings.Builder
	// line and col track the current target position: the index of the
	// current line and the column within it.
	line int
	col  int
	// bldr collects the source map entries; nil disables tracking so
	// ForSpan just runs its callback.
	bldr *sourcemap.Builder
	// inSpan reports whether the text currently being written falls
	// within a source span, in which case each newline maps the new
	// line's start back to that span's source.
	inSpan bool
}

// NewDefaultSourceMapBuffer creates a new DefaultSourceMapBuffer recording
// source map entries into bldr. A nil builder disables tracking while
// keeping the same write behavior.
//
// Matches Dart: SourceMapBuffer constructor (util/source_map_buffer.dart)
func NewDefaultSourceMapBuffer(bldr *sourcemap.Builder) *DefaultSourceMapBuffer {
	return &DefaultSourceMapBuffer{bldr: bldr}
}

// WriteString appends s to the buffer, advancing the column per character
// and recording a passed line per newline.
//
// Matches Dart: SourceMapBuffer.write
func (b *DefaultSourceMapBuffer) WriteString(s string) (int, error) {
	n, _ := b.buf.WriteString(s)
	for _, ch := range s {
		if ch == '\n' {
			if err := b.writeLine(); err != nil {
				return 0, err
			}
		} else {
			b.col++
		}
	}
	return n, nil
}

// WriteByte appends a single byte to the buffer, recording a passed line
// for a newline and advancing the column otherwise.
//
// Matches Dart: SourceMapBuffer.writeCharCode
func (b *DefaultSourceMapBuffer) WriteByte(c byte) error {
	b.buf.WriteByte(c)
	if c == '\n' {
		if err := b.writeLine(); err != nil {
			return err
		}
	} else {
		b.col++
	}
	return nil
}

// WriteRune appends a rune to the buffer, recording a passed line for a
// newline and advancing the column otherwise. Columns count characters,
// matching the serializer's character-based positions.
//
// Matches Dart: SourceMapBuffer.writeCharCode
func (b *DefaultSourceMapBuffer) WriteRune(r rune) (int, error) {
	n, _ := b.buf.WriteRune(r)
	if r == '\n' {
		if err := b.writeLine(); err != nil {
			return 0, err
		}
	} else {
		b.col++
	}
	return n, nil
}

// Write appends p to the buffer with the same position tracking as
// WriteString.
//
// Matches Dart: SourceMapBuffer.write
func (b *DefaultSourceMapBuffer) Write(p []byte) (int, error) {
	return b.WriteString(string(p))
}

// String returns the buffered target text.
//
// Matches Dart: SourceMapBuffer.toString
func (b *DefaultSourceMapBuffer) String() string { return b.buf.String() }

// Len returns the length of the buffered target text.
//
// Matches Dart: SourceMapBuffer.length
func (b *DefaultSourceMapBuffer) Len() int { return b.buf.Len() }

// ForSpan records a mapping at the current position for span.start, then runs
// callback. While inSpan, every newline auto-generates a mapping to keep every
// output line mapped, since source maps resolve each line separately. The
// span end is deliberately not mapped: browsers only look up sources from
// targets, so end mappings would double map size for no benefit.
//
// Matches Dart: SourceMapBuffer.forSpan (with the _addEntry dedup applied)
func (b *DefaultSourceMapBuffer) ForSpan(span sasscommon.FileSpan, cb func() error) error {
	if b.bldr == nil {
		return cb()
	}
	// Matches Dart's _addEntry (and Rust's for_span dedup): skip the entry if
	// the last one shares its source line AND target line (or the same target
	// offset). Browsers only look up source from target, so same-line entries
	// are redundant.
	startLoc, err := span.StartLocation()
	if err != nil {
		return err
	}
	if gl, gc, sl, _, ok := b.bldr.LastMapping(); !ok ||
		!((sl == startLoc.Line && gl == b.line) || (gl == b.line && gc == b.col)) {
		if err := b.bldr.AddMapping(b.line, b.col, span); err != nil {
			return err
		}
	}
	wasInSpan := b.inSpan
	b.inSpan = true
	defer func() { b.inSpan = wasInSpan }()
	return cb()
}

// writeLine records that a line has been passed: a trailing entry pointing
// exactly at the newline is useless (it maps zero width), so it is
// trimmed; then, while inside a span, the new line's start maps back to
// the same source so every output line stays covered.
//
// Matches Dart: SourceMapBuffer._writeLine
func (b *DefaultSourceMapBuffer) writeLine() error {
	// Matches Dart's _writeLine: trim a useless entry whose target is exactly
	// at the current position (before the newline).
	b.bldr.TrimLastIfAt(b.line, b.col)
	b.line++
	b.col = 0
	if b.inSpan {
		// Matches Dart's _writeLine continuation: an entry at the start of the
		// new line reusing the last entry's source.
		b.bldr.AddMappingLikeLast(b.line, b.col)
	}
	return nil
}

// BuildSourceMap produces a SingleMapping from the accumulated entries,
// adjusting target positions by the prefix string. Only entries on the
// prefix's last line shift columns; every entry shifts lines.
//
// Matches Dart: SourceMapBuffer.buildSourceMap
func (b *DefaultSourceMapBuffer) BuildSourceMap(prefix string) (*sourcemap.SingleMapping, error) {
	if prefix == "" {
		return b.bldr.ToMapping()
	}
	prefixLines := 0
	prefixCol := 0
	for _, ch := range prefix {
		if ch == '\n' {
			prefixLines++
			prefixCol = 0
		} else {
			prefixCol++
		}
	}
	return b.bldr.ToMappingWithPrefix(prefixLines, prefixCol)
}
