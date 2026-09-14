// Copyright (c) 2017, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// Package termglyph provides box-drawing glyphs in both ASCII and Unicode.
//
// It mirrors the subset of Dart's external package:term_glyph that the
// compiler uses: the GlyphSet selector plus the box-drawing fields needed
// by the span highlighter. Dart exposes GlyphSet as a class so individual
// chunks of code can choose between its asciiGlyphs and unicodeGlyphs, and
// ships a global glyphs/ascii switch with many more getters (bullets,
// arrows, corners, tees); only the fields the highlighter needs are
// ported, and the global switch is omitted.
//
// dart-source: package:term_glyph (term_glyph.dart + src/generated/glyph_set.dart)
package termglyph

// GlyphSet holds a set of glyph characters for drawing boxes. Callers pick
// AsciiGlyphs or UnicodeGlyphs the way Dart code picks asciiGlyphs or
// unicodeGlyphs.
//
// Matches Dart: GlyphSet abstract class in package:term_glyph.
type GlyphSet struct {
	// VerticalLine draws a vertical box line.
	VerticalLine string
	// HorizontalLine draws a horizontal box line.
	HorizontalLine string
	// DownEnd draws the bottom half of a vertical box line.
	DownEnd string
	// UpEnd draws the top half of a vertical box line.
	UpEnd string
	// TopLeftCorner draws the upper left-hand corner of a box.
	TopLeftCorner string
	// BottomLeftCorner draws the lower left-hand corner of a box.
	BottomLeftCorner string
	// Cross draws an intersection of vertical and horizontal box lines.
	Cross string
	// HorizontalLineBold draws a bold horizontal box line.
	HorizontalLineBold string
}

// AsciiGlyphs always returns plain-ASCII glyphs: the fallback used when
// Unicode output is disabled.
//
// Matches Dart: asciiGlyphs in package:term_glyph.
var AsciiGlyphs = GlyphSet{
	VerticalLine:       "|",
	HorizontalLine:     "-",
	DownEnd:            ",",
	UpEnd:              "'",
	TopLeftCorner:      ",",
	BottomLeftCorner:   "'",
	Cross:              "+",
	HorizontalLineBold: "=",
}

// UnicodeGlyphs always returns Unicode box-drawing glyphs: the default
// used for terminal output.
//
// Matches Dart: unicodeGlyphs in package:term_glyph.
var UnicodeGlyphs = GlyphSet{
	VerticalLine:       "│",
	HorizontalLine:     "─",
	DownEnd:            "╷",
	UpEnd:              "╵",
	TopLeftCorner:      "┌",
	BottomLeftCorner:   "└",
	Cross:              "┼",
	HorizontalLineBold: "━",
}

// IsZero reports whether g is the zero value: a GlyphSet with no glyphs
// set. This is a Go-only helper with no Dart counterpart; it lets callers
// detect an unset set before choosing glyphs.
//
// Go-only (no Dart counterpart).
func (g GlyphSet) IsZero() bool { return g.VerticalLine == "" }

// GlyphOrAscii returns glyph when this set supports Unicode glyphs, or
// alternative when it is the ASCII set.
//
// Matches Dart: GlyphSet.glyphOrAscii in package:term_glyph.
func (g GlyphSet) GlyphOrAscii(glyph, alternative string) string {
	if g == AsciiGlyphs {
		return alternative
	}
	return glyph
}
