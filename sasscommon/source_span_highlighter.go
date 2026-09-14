// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:source_span/lib/src/highlighter.dart

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bancek/go-sass/termglyph"
)

// ANSI terminal color escape codes used by the highlighter.
//
// These are the Go spellings of Dart's colors.dart palette entries consumed
// by Highlighter (highlighter.dart): red highlights the primary span, blue
// the secondary spans and the sidebar, and reset (Dart's colors.none) turns
// colorization back off.
const (
	ansiRed   = "\x1b[31m"
	ansiBlue  = "\x1b[34m"
	ansiReset = "\x1b[0m"
)

// HighlightOptions configures ANSI terminal color highlighting.
//
// This bundles the coloring arguments of Dart's Highlighter constructors
// (highlighter.dart) with the glyph set. Dart reads the ASCII/Unicode choice
// from the term_glyph global; Go threads it explicitly through Glyphs.
type HighlightOptions struct {
	// Color enables highlighting. true selects the default colors, a string
	// is used as the primary color escape, and false/nil disables color.
	// (Matches Dart's dynamic color parameter on Highlighter().)
	Color any

	// PrimaryColor is the ANSI escape to use for the primary span highlight.
	// It overrides the Color default, matching Dart's primaryColor argument
	// on Highlighter.multiple (honored only when color is enabled).
	PrimaryColor string

	// SecondaryColor is the ANSI escape to use for secondary span highlights.
	// It overrides the Color default, matching Dart's secondaryColor
	// argument on Highlighter.multiple (honored only when color is enabled).
	SecondaryColor string

	// Glyphs selects ASCII or Unicode box-drawing characters.
	// A zero value defaults to UnicodeGlyphs.
	Glyphs termglyph.GlyphSet
}

// spacesPerTab is the number of spaces rendered for each hard tab in span
// text. Raw tabs would break caret alignment, so they are expanded.
// Matches Dart: Highlighter._spacesPerTab (highlighter.dart).
const spacesPerTab = 4

// highlighter is a class for writing a chunk of text with a particular span
// highlighted. Ported from Dart source_span Highlighter.
//
// Matches Dart: Highlighter (highlighter.dart). lines holds the display
// lines with their covering highlights (Dart's _lines); primaryColor colors
// the primary highlight within its context and secondaryColor the secondary
// ones, each empty when color is off (Dart's nullable _primaryColor /
// _secondaryColor); paddingBeforeSidebar is the character width before the
// sidebar bar; maxMultilineSpans sizes the per-line multiline indicator
// columns (Dart's _maxMultilineSpans); multipleFiles switches the file
// header style when lines come from different files (Dart's _multipleFiles);
// buf accumulates the result (Dart's _buffer); glyphs selects the
// box-drawing characters (Dart reads the term_glyph global instead).
type highlighter struct {
	lines                []_line
	primaryColor         string // empty if no color
	secondaryColor       string // empty if no color
	paddingBeforeSidebar int
	maxMultilineSpans    int
	multipleFiles        bool
	buf                  strings.Builder
	glyphs               termglyph.GlyphSet
}

// NewHighlighter creates a highlighter for a single span.
//
// Matches Dart: Highlighter(SourceSpan span, {Object? color})
// (highlighter.dart), which highlights the span within its file's text when
// highlight is called. Color may be a string (an ANSI escape used for the
// span's text, e.g. red), true (the default color), or false/nil (no
// color). The span is normalized before its lines are collated, matching
// Dart's _Highlight normalization.
func NewHighlighter(span FileSpan, opts HighlightOptions) (*highlighter, error) {
	var primaryColor string
	c := opts.Color
	if b, ok := c.(bool); ok && b {
		primaryColor = ansiRed
	} else if s, ok := c.(string); ok {
		primaryColor = s
	}
	normalized, err := normalizeSpan(span)
	if err != nil {
		return nil, err
	}
	lines, err := collateLines([]_highlight{{span: normalized, isPrimary: true}})
	if err != nil {
		return nil, err
	}
	return newHighlighter(
		lines,
		primaryColor,
		"",
		opts,
	)
}

// NewHighlighterMultiple creates a highlighter for multiple spans.
//
// Matches Dart: Highlighter.multiple (highlighter.dart), which highlights
// the primary span as well as every secondary span within their files'
// text. Each span carries a label written alongside it (primaryLabel for
// the primary span, the map values for the secondary spans) so the reader
// can tell what each highlight marks. Enabling Color highlights the primary
// span in red and the secondary spans in blue unless PrimaryColor /
// SecondaryColor override them; those overrides are ignored when color is
// off, matching Dart. Nil secondary spans are skipped, a Go tolerance with
// no Dart counterpart (Dart map keys cannot be null).
func NewHighlighterMultiple(primarySpan FileSpan, primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (*highlighter, error) {
	var primaryColor, secondaryColor string
	if b, ok := opts.Color.(bool); ok && b {
		primaryColor = ansiRed
		secondaryColor = ansiBlue
	}
	if opts.PrimaryColor != "" {
		primaryColor = opts.PrimaryColor
	}
	if opts.SecondaryColor != "" {
		secondaryColor = opts.SecondaryColor
	}

	primaryNormalized, err := normalizeSpan(primarySpan)
	if err != nil {
		return nil, err
	}
	highlights := []_highlight{{span: primaryNormalized, label: primaryLabel, isPrimary: true}}
	for span, label := range secondarySpans {
		if span == nil {
			continue
		}
		normalized, err := normalizeSpan(span)
		if err != nil {
			return nil, err
		}
		highlights = append(highlights, _highlight{span: normalized, label: label})
	}
	lines, err := collateLines(highlights)
	if err != nil {
		return nil, err
	}
	return newHighlighter(lines, primaryColor, secondaryColor, opts)
}

// newHighlighter finishes construction from collated lines and resolved
// colors, computing the derived layout.
//
// Matches Dart: Highlighter._ (highlighter.dart). The sidebar padding fits
// the widest 1-based line number, widened to 3 digits when the lines are
// not contiguous (room for the "..." gap marker). maxMultilineSpans is the
// largest number of multiline spans covering any single line, which sizes
// the per-line indicator columns. multipleFiles records whether the lines
// come from different files, which selects the file-header style.
func newHighlighter(lines []_line, primaryColor, secondaryColor string, opts HighlightOptions) (*highlighter, error) {
	glyphs := opts.Glyphs
	if glyphs.IsZero() {
		glyphs = termglyph.UnicodeGlyphs
	}
	h := &highlighter{
		lines:          lines,
		primaryColor:   primaryColor,
		secondaryColor: secondaryColor,
		glyphs:         glyphs,
	}

	maxLineNum := lines[len(lines)-1].number + 1
	// Digit count via formatting: Dart notes that floor(log10(n)) is
	// unreliable from floating-point error, so it converts to a string;
	// Go counts the same string's length directly.
	lineDigits := len(fmt.Sprint(maxLineNum))
	contiguous := contiguous(lines)
	pad := 1 + lineDigits
	if !contiguous && pad < 1+3 {
		pad = 1 + 3
	}
	h.paddingBeforeSidebar = pad

	for _, line := range lines {
		count := 0
		for _, hl := range line.highlights {
			multi, err := isMultiline(hl.span)
			if err != nil {
				return nil, err
			}
			if multi {
				count++
			}
		}
		if count > h.maxMultilineSpans {
			h.maxMultilineSpans = count
		}
	}

	// multipleFiles check: any line whose display URL differs from the
	// first means the highlight spans files. Matches Dart: the
	// _multipleFiles computation (!isAllTheSame of line urls).
	if len(lines) > 0 {
		first := lines[0].url
		for _, line := range lines[1:] {
			if line.url != first {
				h.multipleFiles = true
				break
			}
		}
	}

	return h, nil
}

// Highlight returns the highlighted span text. Should only be called once.
//
// Matches Dart: Highlighter.highlight (highlighter.dart). Each slot of
// highlightsByColumn is a column past the sidebar that draws one active
// multiline highlight: nil when the column is empty, otherwise the
// highlight drawn there.
func (h *highlighter) Highlight() (string, error) {
	h.writeFileStart(h.lines[0].url)

	highlightsByColumn := make([]*_highlight, h.maxMultilineSpans)

	for i, line := range h.lines {
		if i > 0 {
			lastLine := h.lines[i-1]
			if lastLine.url != line.url {
				h.writeSidebarEnd(h.glyphs.UpEnd)
				h.buf.WriteByte('\n')
				h.writeFileStart(line.url)
			} else if lastLine.number+1 != line.number {
				h.writeSidebarText("...")
				h.buf.WriteByte('\n')
			}
		}

		// If a highlight covers the entire first line other than initial
		// whitespace, don't bother pointing out exactly where it begins.
		// Iterate in reverse so that longer highlights (sorted after
		// shorter ones) take the outer columns, leaving fewer crossed
		// lines. Matches Dart: the reversed loop in highlight().
		for j := len(line.highlights) - 1; j >= 0; j-- {
			hl := &line.highlights[j]
			multi, err := isMultiline(hl.span)
			if err != nil {
				return "", err
			}
			startLoc, err := hl.span.StartLocation()
			if err != nil {
				return "", err
			}
			if multi &&
				startLoc.Line == line.number &&
				isOnlyWhitespace(line.text[:startLoc.Column]) {
				replaceFirstNull(highlightsByColumn, hl)
			}
		}

		h.writeSidebarLine(line.number)
		h.buf.WriteByte(' ')
		if err := h.writeMultilineHighlights(line, highlightsByColumn, nil); err != nil {
			return "", err
		}
		if len(highlightsByColumn) > 0 {
			h.buf.WriteByte(' ')
		}

		var primary *_highlight
		for j := range line.highlights {
			if line.highlights[j].isPrimary {
				primary = &line.highlights[j]
				break
			}
		}

		if primary != nil {
			startCol := 0
			primaryStartLoc, err := primary.span.StartLocation()
			if err != nil {
				return "", err
			}
			primaryEndLoc, err := primary.span.EndLocation()
			if err != nil {
				return "", err
			}
			if primaryStartLoc.Line == line.number {
				startCol = primaryStartLoc.Column
			}
			endCol := len(line.text)
			if primaryEndLoc.Line == line.number {
				endCol = primaryEndLoc.Column
			}
			h.writeHighlightedText(line.text, startCol, endCol, h.primaryColor)
		} else {
			h.writeText(line.text)
		}
		h.buf.WriteByte('\n')

		// Always write the primary span's indicator first so that it's right
		// next to the highlighted text. Matches Dart: highlight() writes
		// the primary indicator before the secondary ones.
		if primary != nil {
			if err := h.writeIndicator(line, *primary, highlightsByColumn); err != nil {
				return "", err
			}
		}
		for j := range line.highlights {
			if line.highlights[j].isPrimary {
				continue
			}
			if err := h.writeIndicator(line, line.highlights[j], highlightsByColumn); err != nil {
				return "", err
			}
		}
	}

	h.writeSidebarEnd(h.glyphs.UpEnd)
	return h.buf.String(), nil
}

// --- private helpers ---

// contiguous reports whether lines holds any adjacent lines from the same
// source file that are not adjacent in the original file (a gap renders as
// "..."). Matches Dart: Highlighter._contiguous (highlighter.dart).
func contiguous(lines []_line) bool {
	for i := 0; i < len(lines)-1; i++ {
		if lines[i].number+1 != lines[i+1].number && lines[i].url == lines[i+1].url {
			return false
		}
	}
	return true
}

// collateLines collects the source lines from every highlight's context
// and associates each line with the highlights covering it.
//
// Matches Dart: Highlighter._collateLines (highlighter.dart). Spans without
// URLs get opaque per-span group keys so lines from different spans never
// merge: Dart uses fresh Objects, Go uses nul-prefixed synthetic keys (a
// real display URL can never start with a nul byte). Within each file group
// the highlights sort by start position before lines are built.
func collateLines(highlights []_highlight) ([]_line, error) {
	// Group by URL. The primary's file group renders first, then the rest in
	// insertion order (Dart renders the primary span's file first). The map is
	// only used for lookup; group order is kept in slices so it's deterministic.
	groupOf := make(map[string]int)
	var groups [][]_highlight
	var groupURLs []string
	noURLIdx := 0
	primaryGroup := 0
	for i, hl := range highlights {
		u, err := hl.span.SourceURL()
		if err != nil {
			return nil, err
		}
		url := ""
		if u != nil {
			url = PrettyUri(u)
		}
		if url == "" {
			url = fmt.Sprintf("\x00%d", noURLIdx)
			noURLIdx++
		}
		idx, ok := groupOf[url]
		if !ok {
			idx = len(groups)
			groups = append(groups, nil)
			groupURLs = append(groupURLs, url)
			groupOf[url] = idx
		}
		if i == 0 {
			primaryGroup = idx
		}
		groups[idx] = append(groups[idx], hl)
	}

	if primaryGroup != 0 && len(groups) > 0 {
		group := groups[primaryGroup]
		url := groupURLs[primaryGroup]
		groups = append(groups[:primaryGroup], groups[primaryGroup+1:]...)
		groupURLs = append(groupURLs[:primaryGroup], groupURLs[primaryGroup+1:]...)
		groups = append([][]_highlight{group}, groups...)
		groupURLs = append([]string{url}, groupURLs...)
	}

	var allLines []_line
	for gi, hls := range groups {
		url := groupURLs[gi]
		// Sort by start position
		if err := sortHighlights(hls); err != nil {
			return nil, err
		}

		// Build lines from context. First, create every source line the
		// group has context for along with its line number; only add a
		// line if a previous span has not already added it.
		// Matches Dart's _collateLines which iterates highlights and
		// splits their span.context by '\n', adding unique non-duplicate lines.
		var lines []_line
		for _, hl := range hls {
			// Matches Dart's _collateLines: highlight.span.context.
			// Fallback to SpanText() is needed because Context() returns ""
			// for spans without a FileSource (e.g., BogusSpan). Dart's
			// _FileSpan always has a SourceFile, so it always has context.
			context, err := hl.span.Context()
			if err != nil {
				return nil, err
			}
			if context == "" {
				context, err = hl.span.SpanText()
				if err != nil {
					return nil, err
				}
			}
			spanText, err := hl.span.SpanText()
			if err != nil {
				return nil, err
			}
			startLoc, err := hl.span.StartLocation()
			if err != nil {
				return nil, err
			}
			lineStart, ok := findLineStart(context, spanText, startLoc.Column)
			if !ok {
				continue
			}
			linesBeforeSpan := strings.Count(context[:lineStart], "\n")
			lineNumber := startLoc.Line - linesBeforeSpan
			for _, lineText := range strings.Split(context, "\n") {
				if len(lines) == 0 || lineNumber > lines[len(lines)-1].number {
					lines = append(lines, _line{text: lineText, number: lineNumber, url: url})
				}
				lineNumber++
			}
		}

		// Associate highlights with lines. Next, walk the lines carrying
		// forward the active highlights: drop the ones that ended before
		// this line, then pick up every highlight starting on it.
		// Matches Dart: the activeHighlights sweep in _collateLines
		// (removeWhere on end.line, then add highlights starting here).
		var activeHighlights []_highlight
		hlIdx := 0
		for li := range lines {
			line := &lines[li]
			// Remove highlights that have ended
			filtered := activeHighlights[:0]
			for _, hl := range activeHighlights {
				endLoc, err := hl.span.EndLocation()
				if err != nil {
					return nil, err
				}
				if endLoc.Line >= line.number {
					filtered = append(filtered, hl)
				}
			}
			activeHighlights = filtered

			for hlIdx < len(hls) {
				startLoc, err := hls[hlIdx].span.StartLocation()
				if err != nil {
					return nil, err
				}
				if startLoc.Line > line.number {
					break
				}
				activeHighlights = append(activeHighlights, hls[hlIdx])
				hlIdx++
			}

			// Add ALL active highlights to this line (not just newly started ones).
			// Matches Dart: line.highlights.addAll(activeHighlights)
			line.highlights = append(line.highlights, activeHighlights...)
		}

		allLines = append(allLines, lines...)
	}

	return allLines, nil
}

// writeFileStart writes the beginning of a file's highlight block.
//
// Matches Dart: Highlighter._writeFileStart (highlighter.dart). A lone file
// (or a span with no URL, held under a nul-prefixed synthetic key) just
// closes the sidebar; with multiple files, a headed URL line names the new
// file via PrettyUri, matching Dart's prettyUri rendering.
func (h *highlighter) writeFileStart(url string) {
	if !h.multipleFiles || url == "" || url[0] == '\x00' {
		h.writeSidebarEnd(h.glyphs.DownEnd)
	} else {
		h.writeSidebarEnd(h.glyphs.TopLeftCorner)
		h.colorize(func() {
			h.buf.WriteString(h.glyphs.HorizontalLine)
			h.buf.WriteString(h.glyphs.HorizontalLine)
			h.buf.WriteString(">")
		}, ansiBlue)
		h.buf.WriteString(" ")
		h.buf.WriteString(url)
	}
	h.buf.WriteByte('\n')
}

// writeMultilineHighlights writes the post-sidebar highlight bars for
// line according to highlightsByColumn.
//
// Matches Dart: Highlighter._writeMultilineHighlights (highlighter.dart).
// When current is passed, an indicator is being written for it: its column
// draws a corner (top on its start line, bottom on its end line) and a
// horizontal line runs from its column to the rightmost column. Otherwise
// each occupied column draws the span's vertical bar, opening a new corner
// for spans that start on this line and remembering the opened color for
// the rightward line.
func (h *highlighter) writeMultilineHighlights(line _line, highlightsByColumn []*_highlight, current *_highlight) error {
	openedOnThisLine := false
	openedOnThisLineColor := ""
	currentColor := ""
	if current != nil {
		if current.isPrimary {
			currentColor = h.primaryColor
		} else {
			currentColor = h.secondaryColor
		}
	}
	foundCurrent := false

	for _, hl := range highlightsByColumn {
		startLine := -1
		endLine := -1
		if hl != nil {
			startLoc, err := hl.span.StartLocation()
			if err != nil {
				return err
			}
			startLine = startLoc.Line
			endLoc, err := hl.span.EndLocation()
			if err != nil {
				return err
			}
			endLine = endLoc.Line
		}
		eq, err := highlightEqual(current, hl)
		if err != nil {
			return err
		}
		if eq && current != nil {
			foundCurrent = true
			char := h.glyphs.BottomLeftCorner
			if startLine == line.number {
				char = h.glyphs.TopLeftCorner
			}
			h.colorize(func() { h.buf.WriteString(char) }, currentColor)
		} else if foundCurrent {
			char := h.glyphs.Cross
			if hl == nil {
				char = h.glyphs.HorizontalLine
			}
			h.colorize(func() { h.buf.WriteString(char) }, currentColor)
		} else if hl == nil {
			if openedOnThisLine {
				h.colorize(func() { h.buf.WriteString(h.glyphs.HorizontalLine) }, openedOnThisLineColor)
			} else {
				h.buf.WriteString(" ")
			}
		} else {
			hlColor := h.secondaryColor
			if hl.isPrimary {
				hlColor = h.primaryColor
			}
			vertical := h.glyphs.Cross
			if !openedOnThisLine {
				vertical = h.glyphs.VerticalLine
			}
			hlEndLoc, err := hl.span.EndLocation()
			if err != nil {
				return err
			}
			h.colorize(func() {
				if current != nil {
					h.buf.WriteString(vertical)
				} else if startLine == line.number {
					char := h.glyphs.GlyphOrAscii("┌", "/")
					if openedOnThisLine {
						char = h.glyphs.GlyphOrAscii("┬", "/")
					}
					h.colorize(func() { h.buf.WriteString(char) }, openedOnThisLineColor)
					openedOnThisLine = true
					if openedOnThisLineColor == "" {
						openedOnThisLineColor = hlColor
					}
				} else if endLine == line.number && hlEndLoc.Column == len(line.text) {
					if hl.label == "" {
						h.buf.WriteString(h.glyphs.GlyphOrAscii("└", "\\"))
					} else {
						h.buf.WriteString(vertical)
					}
				} else {
					h.colorize(func() { h.buf.WriteString(vertical) }, openedOnThisLineColor)
				}
			}, hlColor)
		}
	}
	return nil
}

// writeHighlightedText writes text with the [startColumn, endColumn) range
// colorized. Matches Dart: Highlighter._writeHighlightedText
// (highlighter.dart).
func (h *highlighter) writeHighlightedText(text string, startColumn, endColumn int, color string) {
	if startColumn > len(text) {
		startColumn = len(text)
	}
	if endColumn > len(text) {
		endColumn = len(text)
	}
	if startColumn > endColumn {
		startColumn = endColumn
	}
	h.writeText(text[:startColumn])
	h.colorize(func() { h.writeText(text[startColumn:endColumn]) }, color)
	h.writeText(text[endColumn:])
}

// writeIndicator writes an indicator for where highlight starts, ends, or
// both below line.
//
// Matches Dart: Highlighter._writeIndicator (highlighter.dart), which may
// either add or remove highlight from highlightsByColumn. A single-line
// span draws an underline (^ for primary, bold horizontal for secondary)
// plus its label; a span starting on this line claims a column and draws
// an opening arrow; a span ending here draws a closing arrow (or a short
// rule for a whole-line end) plus its label, then releases the column —
// unless it covers the whole line with no label, in which case it just
// releases the column silently.
func (h *highlighter) writeIndicator(line _line, highlight _highlight, highlightsByColumn []*_highlight) error {
	color := h.secondaryColor
	if highlight.isPrimary {
		color = h.primaryColor
	}
	multi, err := isMultiline(highlight.span)
	if err != nil {
		return err
	}
	if !multi {
		h.writeSidebar()
		h.buf.WriteByte(' ')
		if err := h.writeMultilineHighlights(line, highlightsByColumn, &highlight); err != nil {
			return err
		}
		if len(highlightsByColumn) > 0 {
			h.buf.WriteByte(' ')
		}
		ulStartLoc, err := highlight.span.StartLocation()
		if err != nil {
			return err
		}
		ulEndLoc, err := highlight.span.EndLocation()
		if err != nil {
			return err
		}
		underlineLength := h.colorizeInt(func() int {
			start := h.buf.Len()
			h.writeUnderline(line, ulStartLoc.Column, ulEndLoc.Column, func() string {
				if highlight.isPrimary {
					return "^"
				}
				return h.glyphs.HorizontalLineBold
			}())
			return h.buf.Len() - start
		}, color)
		h.writeLabel(highlight, highlightsByColumn, underlineLength)
		return nil
	}
	startLoc, err := highlight.span.StartLocation()
	if err != nil {
		return err
	}
	endLoc, err := highlight.span.EndLocation()
	if err != nil {
		return err
	}
	if startLoc.Line == line.number {
		inCol, err := highlightInColumn(highlightsByColumn, &highlight)
		if err != nil {
			return err
		}
		if inCol {
			return nil
		}
		replaceFirstNull(highlightsByColumn, &highlight)
		h.writeSidebar()
		h.buf.WriteByte(' ')
		if err := h.writeMultilineHighlights(line, highlightsByColumn, &highlight); err != nil {
			return err
		}
		h.colorize(func() { h.writeArrow(line, startLoc.Column, true) }, color)
		h.buf.WriteByte('\n')
		return nil
	}
	if endLoc.Line == line.number {
		coversWholeLine := endLoc.Column == len(line.text)
		if coversWholeLine && highlight.label == "" {
			if err := replaceWithNull(highlightsByColumn, &highlight); err != nil {
				return err
			}
			return nil
		}
		h.writeSidebar()
		h.buf.WriteByte(' ')
		if err := h.writeMultilineHighlights(line, highlightsByColumn, &highlight); err != nil {
			return err
		}
		underlineLength := h.colorizeInt(func() int {
			start := h.buf.Len()
			if coversWholeLine {
				h.buf.WriteString(strings.Repeat(h.glyphs.HorizontalLine, 3))
			} else {
				col := endLoc.Column - 1
				if col < 0 {
					col = 0
				}
				h.writeArrow(line, col, false)
			}
			return h.buf.Len() - start
		}, color)
		h.writeLabel(highlight, highlightsByColumn, underlineLength)
		if err := replaceWithNull(highlightsByColumn, &highlight); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// highlightInColumn checks if a highlight with the same span and primary flag
// is already present in the column list. Uses value comparison, not pointer
// comparison, because callers may pass different copies of the same struct.
//
// This is the Go spelling of Dart's highlightsByColumn.contains(highlight)
// identity check in _writeIndicator: Dart compares object identity, but Go
// copies _highlight values at every call site, so equality of the span
// endpoints plus the primary flag stands in for it.
func highlightInColumn(columns []*_highlight, hl *_highlight) (bool, error) {
	for _, col := range columns {
		if col != nil && col.isPrimary == hl.isPrimary {
			colStart, err := col.span.StartLocation()
			if err != nil {
				return false, err
			}
			hlStart, err := hl.span.StartLocation()
			if err != nil {
				return false, err
			}
			colEnd, err := col.span.EndLocation()
			if err != nil {
				return false, err
			}
			hlEnd, err := hl.span.EndLocation()
			if err != nil {
				return false, err
			}
			if colStart == hlStart && colEnd == hlEnd {
				return true, nil
			}
		}
	}
	return false, nil
}

// writeUnderline underlines the portion of line covered by the
// [startColumn, endColumn) range with repeated instances of character.
//
// Matches Dart: Highlighter._writeUnderline (highlighter.dart), which
// underlines a single-line span known to sit within the line text. Start
// and end shift right to account for tabs expanded to spacesPerTab, and a
// zero-width span still draws one character so it stays visible.
func (h *highlighter) writeUnderline(line _line, startColumn, endColumn int, character string) {
	if startColumn > len(line.text) {
		startColumn = len(line.text)
	}
	if endColumn > len(line.text) {
		endColumn = len(line.text)
	}
	if startColumn > endColumn {
		startColumn = endColumn
	}
	// Columns are byte offsets into line.text, but the underline must be sized
	// and positioned by display characters (Dart counts UTF-16 units). Convert
	// to rune counts so multibyte text doesn't stretch the caret.
	startByte := charBoundaryAtOrBefore(line.text, startColumn)
	endByte := charBoundaryAtOrBefore(line.text, endColumn)
	tabsBefore := countTabs(line.text[:startByte])
	tabsInside := countTabs(line.text[startByte:endByte])
	start := utf8.RuneCountInString(line.text[:startByte]) + tabsBefore*(spacesPerTab-1)
	end := utf8.RuneCountInString(line.text[:endByte]) + (tabsBefore+tabsInside)*(spacesPerTab-1)
	width := end - start
	if width < 1 {
		width = 1
	}
	h.buf.WriteString(strings.Repeat(" ", start))
	h.buf.WriteString(strings.Repeat(character, width))
}

// charBoundaryAtOrBefore clamps idx back to the start of a UTF-8 rune.
//
// Byte-offset columns can land mid-codepoint on multibyte text; callers
// pass columns through here before slicing or counting display characters
// so multibyte lines neither panic nor stretch the caret. This is a Go-only
// guard with no Dart counterpart (Dart indexes UTF-16 units, which cannot
// split observably here).
func charBoundaryAtOrBefore(text string, idx int) int {
	if idx > len(text) {
		idx = len(text)
	}
	for idx > 0 && idx < len(text) && !utf8.RuneStart(text[idx]) {
		idx--
	}
	return idx
}

// writeArrow writes an arrow pointing to column in line.
//
// Matches Dart: Highlighter._writeArrow (highlighter.dart). If the arrow
// points at a tab, beginning selects whether it points to the tab's start
// or end (the scan covers one extra character for a closing arrow).
func (h *highlighter) writeArrow(line _line, column int, beginning bool) {
	if column > len(line.text) {
		column = len(line.text)
	}
	// Columns are byte offsets; the arrow is positioned by display character.
	colByte := charBoundaryAtOrBefore(line.text, column)
	// For a non-beginning arrow the tab count scans one character past the
	// cursor; clamp so an end column at (or past) the end of the line doesn't
	// slice out of range. The clamp is a Go slice-safety guard with no Dart
	// counterpart (Dart's substring would throw there too, but its callers
	// never pass an end column).
	end := colByte
	if !beginning && colByte < len(line.text) {
		_, size := utf8.DecodeRuneInString(line.text[colByte:])
		end = colByte + size
	}
	tabCount := countTabs(line.text[:end])
	total := 1 + utf8.RuneCountInString(line.text[:colByte]) + tabCount*(spacesPerTab-1)
	h.buf.WriteString(strings.Repeat(h.glyphs.HorizontalLine, total))
	h.buf.WriteString("^")
}

// writeLabel writes highlight's label after the underline.
//
// Matches Dart: Highlighter._writeLabel (highlighter.dart). The buffer is
// assumed written up to where the label's first line goes after a space;
// continuation lines re-emit the sidebar, the still-active highlight
// columns, and underlineLength spaces of indentation so they align under
// the first line. underlineLength is the display width of the rule drawn
// between the highlights and the first label.
func (h *highlighter) writeLabel(highlight _highlight, highlightsByColumn []*_highlight, underlineLength int) {
	label := highlight.label
	if label == "" {
		h.buf.WriteByte('\n')
		return
	}
	labelLines := strings.Split(label, "\n")
	color := h.secondaryColor
	if highlight.isPrimary {
		color = h.primaryColor
	}
	h.colorize(func() {
		h.buf.WriteString(" ")
		h.buf.WriteString(labelLines[0])
	}, color)
	h.buf.WriteByte('\n')

	for _, text := range labelLines[1:] {
		h.writeSidebar()
		h.buf.WriteByte(' ')
		for _, colHl := range highlightsByColumn {
			if colHl == nil || colHl == &highlight {
				h.buf.WriteString(" ")
			} else {
				h.buf.WriteString(h.glyphs.VerticalLine)
			}
		}
		h.buf.WriteString(strings.Repeat(" ", underlineLength))
		h.colorize(func() {
			h.buf.WriteString(" ")
			h.buf.WriteString(text)
		}, color)
		h.buf.WriteByte('\n')
	}
}

// writeText writes a snippet of source text, converting hard tabs into
// plain indentation. Matches Dart: Highlighter._writeText
// (highlighter.dart).
func (h *highlighter) writeText(text string) {
	for _, ch := range text {
		if ch == '\t' {
			h.buf.WriteString(strings.Repeat(" ", spacesPerTab))
		} else {
			h.buf.WriteRune(ch)
		}
	}
}

// writeSidebar writes a blank sidebar (continuation/indicator rows).
// Matches Dart: Highlighter._writeSidebar() (highlighter.dart).
func (h *highlighter) writeSidebar() {
	h.writeSidebarInternal(-1, "", "")
}

// writeSidebarLine writes a sidebar showing the 0-based line number.
// Matches Dart: Highlighter._writeSidebar(line:) (highlighter.dart).
func (h *highlighter) writeSidebarLine(line int) {
	h.writeSidebarInternal(line, "", "")
}

// writeSidebarText writes a sidebar showing text (e.g. "..." for a gap)
// in place of the line number.
// Matches Dart: Highlighter._writeSidebar(text:) (highlighter.dart).
func (h *highlighter) writeSidebarText(text string) {
	h.writeSidebarInternal(-1, text, "")
}

// writeSidebarEnd writes a sidebar closed with end (a corner glyph) in
// place of the default vertical bar.
// Matches Dart: Highlighter._writeSidebar(end:) (highlighter.dart).
func (h *highlighter) writeSidebarEnd(end string) {
	h.writeSidebarInternal(-1, "", end)
}

// writeSidebarInternal writes a sidebar cell: text (a 1-based line number
// when line is given) padded to paddingBeforeSidebar, then the end glyph.
//
// Matches Dart: Highlighter._writeSidebar (highlighter.dart), which
// converts 0-indexed lines to human-friendly 1-indexed numbers the same
// way. Line and text are mutually exclusive; an empty end selects the
// vertical bar. The whole cell renders in the sidebar color.
func (h *highlighter) writeSidebarInternal(line int, text, end string) {
	if line >= 0 {
		text = fmt.Sprint(line + 1)
	}
	if end == "" {
		end = h.glyphs.VerticalLine
	}
	h.colorize(func() {
		padded := text
		for len(padded) < h.paddingBeforeSidebar {
			padded = padded + " "
		}
		h.buf.WriteString(padded)
		h.buf.WriteString(end)
	}, ansiBlue)
}

// colorize writes everything cb emits wrapped in color's ANSI escape,
// when colorization is enabled.
//
// Matches Dart: Highlighter._colorize (highlighter.dart). Color applies
// only when both the highlighter has a primary color (i.e. color was
// enabled at construction) and the fragment supplies a non-empty color —
// so sidebar blue and secondary blue stay dark together when color is off.
func (h *highlighter) colorize(cb func(), color string) {
	if h.primaryColor != "" && color != "" {
		h.buf.WriteString(color)
	}
	cb()
	if h.primaryColor != "" && color != "" {
		h.buf.WriteString(ansiReset)
	}
}

// colorizeInt is colorize for callbacks that measure what they wrote: it
// returns cb's byte count (used for the underline width in writeIndicator)
// rather than discarding it.
func (h *highlighter) colorizeInt(cb func() int, color string) int {
	if h.primaryColor != "" && color != "" {
		h.buf.WriteString(color)
	}
	result := cb()
	if h.primaryColor != "" && color != "" {
		h.buf.WriteString(ansiReset)
	}
	return result
}

// countTabs returns the number of hard tabs in text.
// Matches Dart: Highlighter._countTabs (highlighter.dart).
func countTabs(text string) int {
	count := 0
	for _, ch := range text {
		if ch == '\t' {
			count++
		}
	}
	return count
}

// isOnlyWhitespace reports whether text contains only spaces or tabs.
// Matches Dart: Highlighter._isOnlyWhitespace (highlighter.dart).
func isOnlyWhitespace(text string) bool {
	for _, ch := range text {
		if ch != ' ' && ch != '\t' {
			return false
		}
	}
	return true
}

// --- _highlight ---

// _highlight is how to highlight a single section of a source file.
//
// Matches Dart: _Highlight (highlighter.dart). span is the section to
// highlight, already normalized for the highlighter to consume; isPrimary
// marks the primary span, drawn with the primary color and (for
// single-line spans) the ^ character rather than the secondary style;
// label is written inline next to the span to say what it marks.
type _highlight struct {
	span      FileSpan
	isPrimary bool
	label     string
}

// --- _line ---

// _line is a single line of the highlighted source file.
//
// Matches Dart: _Line (highlighter.dart). text is the line without its
// trailing newline; number is the 0-based line number in the source file;
// url is the display URL of the file the line came from (for spans without
// a URL, an opaque nul-prefixed key that differs per span, matching Dart's
// opaque-Object keys); highlights holds every highlight covering any part
// of the line in source order, filled in after creation.
type _line struct {
	text       string
	number     int
	url        string
	highlights []_highlight
}

// --- span normalization (ported from _Highlight helpers) ---

// normalizeSpan runs the full normalization chain over span so the
// highlighter can assume context, newlines, and endpoints line up.
//
// Matches Dart: the _Highlight constructor body (highlighter.dart), which
// threads _normalizeContext, _normalizeNewlines, _normalizeTrailingNewline,
// and _normalizeEndOfLine in the same order.
func normalizeSpan(span FileSpan) (FileSpan, error) {
	s, err := normalizeContext(span)
	if err != nil {
		return nil, err
	}
	s, err = normalizeNewlines(s)
	if err != nil {
		return nil, err
	}
	s, err = normalizeTrailingNewline(s)
	if err != nil {
		return nil, err
	}
	return normalizeEndOfLine(s)
}

// normalizeContext ensures span is a SourceSpanWithContext whose context
// actually contains its text at the expected column.
//
// Matches Dart: _Highlight._normalizeContext (highlighter.dart). A span
// already carrying a consistent context passes through; otherwise the
// locations are re-based so the highlighter can assume they match the
// context (line 0, column 0 start with the line/column derived from the
// text itself). The middle branch is a Go adaptation for the two-type
// split — see its inline note.
func normalizeContext(span FileSpan) (FileSpan, error) {
	text, err := span.SpanText()
	if err != nil {
		return nil, err
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	column := startLoc.Column

	// Matches Dart's _Highlight._normalizeContext.
	//
	// Dart has one type check: span is SourceSpanWithContext — because
	// _FileSpan implements SourceSpanWithContext directly.
	//
	// Go has three branches because SimpleFileSpan and SourceSpanWithContext
	// are separate concrete types (both implement FileSpan, neither is the
	// other). The middle branch catches spans that have a valid Context()
	// (e.g., SimpleFileSpan backed by FileSource) but are not
	// SourceSpanWithContext instances. This is a necessary Go adaptation,
	// not a simplification — removing it would discard the FileSource-computed
	// context and fall back to using SpanText() as the context window.
	if ssc, ok := span.(SourceSpanWithContext); ok {
		sscCtx, err := ssc.Context()
		if err != nil {
			return nil, err
		}
		sscText, err := ssc.SpanText()
		if err != nil {
			return nil, err
		}
		if _, ok2 := findLineStart(sscCtx, sscText, column); ok2 {
			return span, nil
		}
	} else if context, err := span.Context(); err == nil && context != "" {
		if _, ok := findLineStart(context, text, column); ok {
			startLoc, err := span.StartLocation()
			if err != nil {
				return nil, err
			}
			endLoc, err := span.EndLocation()
			if err != nil {
				return nil, err
			}
			sourceURL, err := span.SourceURL()
			if err != nil {
				return nil, err
			}
			ssc, err := NewSourceSpanWithContext(
				startLoc,
				endLoc,
				text,
				context,
				sourceURL,
			)
			if err != nil {
				return nil, err
			}
			return ssc, nil
		}
	}

	startLoc2, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	endLoc2, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	sourceURL, err := span.SourceURL()
	if err != nil {
		return nil, err
	}
	ssc, err := NewSourceSpanWithContext(
		SourceLocation{Offset: startLoc2.Offset, Line: 0, Column: 0},
		SourceLocation{
			Offset: endLoc2.Offset,
			Line:   strings.Count(text, "\n"),
			Column: lastLineLength(text),
		},
		text,
		text,
		sourceURL,
	)
	if err != nil {
		return nil, err
	}
	return ssc, nil
}

// normalizeNewlines replaces Windows-style newlines with Unix-style ones
// in both the span text and its context.
//
// Matches Dart: _Highlight._normalizeNewlines (highlighter.dart). Each
// CRLF pair collapses to one character, so the end offset steps back one
// per pair to stay aligned with the shortened text.
func normalizeNewlines(span FileSpan) (FileSpan, error) {
	text, err := span.SpanText()
	if err != nil {
		return nil, err
	}
	if !strings.Contains(text, "\r\n") {
		return span, nil
	}
	endLoc, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	endOffset := endLoc.Offset
	for i := 0; i < len(text)-1; i++ {
		if text[i] == '\r' && text[i+1] == '\n' {
			endOffset--
		}
	}
	context, err := span.Context()
	if err != nil {
		return nil, err
	}
	if context == "" {
		context, err = span.SpanText()
		if err != nil {
			return nil, err
		}
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	endLoc2, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	sourceURL, err := span.SourceURL()
	if err != nil {
		return nil, err
	}
	ssc, err := NewSourceSpanWithContext(
		startLoc,
		SourceLocation{
			Offset: endOffset,
			Line:   endLoc2.Line,
			Column: endLoc2.Column,
		},
		strings.ReplaceAll(text, "\r\n", "\n"),
		strings.ReplaceAll(context, "\r\n", "\n"),
		sourceURL,
	)
	if err != nil {
		return nil, err
	}
	return ssc, nil
}

// normalizeTrailingNewline removes a trailing newline from the span
// context, adjusting the span end when it pointed past that newline.
//
// Matches Dart: _Highlight._normalizeTrailingNewline (highlighter.dart). A
// context ending in a full blank line is significant and left alone, as is
// a span text ending in a double newline; otherwise the span text loses a
// final newline that sits at the very end of the context, with the end
// location rewound to the previous line (a span reduced to empty collapses
// onto its start, and a point span follows its end).
func normalizeTrailingNewline(span FileSpan) (FileSpan, error) {
	context, err := span.Context()
	if err != nil {
		return nil, err
	}
	if context == "" {
		context, err = span.SpanText()
		if err != nil {
			return nil, err
		}
	}
	if !strings.HasSuffix(context, "\n") {
		return span, nil
	}
	text, err := span.SpanText()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(text, "\n\n") {
		return span, nil
	}
	context = context[:len(context)-1]
	newText := text
	startLoc, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	endLoc, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	newStart := startLoc
	newEnd := endLoc
	if strings.HasSuffix(text, "\n") {
		atEnd, err := isTextAtEndOfContext(span)
		if err != nil {
			return nil, err
		}
		if atEnd {
			newText = text[:len(text)-1]
			if len(newText) == 0 {
				newEnd = newStart
			} else {
				endLoc2, err := span.EndLocation()
				if err != nil {
					return nil, err
				}
				newEnd = SourceLocation{
					Offset: endLoc2.Offset - 1,
					Line:   endLoc2.Line - 1,
					Column: lastLineLength(context),
				}
				startLoc2, err := span.StartLocation()
				if err != nil {
					return nil, err
				}
				endLoc3, err := span.EndLocation()
				if err != nil {
					return nil, err
				}
				if startLoc2.Offset == endLoc3.Offset {
					newStart = newEnd
				}
			}
		}
	}
	sourceURL, err := span.SourceURL()
	if err != nil {
		return nil, err
	}
	ssc, err := NewSourceSpanWithContext(
		newStart,
		newEnd,
		newText,
		context,
		sourceURL,
	)
	if err != nil {
		return nil, err
	}
	return ssc, nil
}

// normalizeEndOfLine moves a span end that sits at column 0 of some line
// back to the end of the previous line.
//
// Matches Dart: _Highlight._normalizeEndOfLine (highlighter.dart). A span
// ending at the very start of a line covers that line's newline, so the
// highlight trims the trailing newline and ends on the prior line instead.
// Single-line spans are untouched. When the context itself ends with a
// newline its last line may be incomplete, so it is trimmed from the
// context rather than printed.
func normalizeEndOfLine(span FileSpan) (FileSpan, error) {
	endLoc, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	if endLoc.Column != 0 {
		return span, nil
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	if endLoc.Line == startLoc.Line {
		return span, nil
	}
	text, err := span.SpanText()
	if err != nil {
		return nil, err
	}
	text = text[:len(text)-1]
	context, err := span.Context()
	if err != nil {
		return nil, err
	}
	if context == "" {
		context = text
	}
	context = strings.TrimSuffix(context, "\n")
	startLoc2, err := span.StartLocation()
	if err != nil {
		return nil, err
	}
	endLoc2, err := span.EndLocation()
	if err != nil {
		return nil, err
	}
	sourceURL, err := span.SourceURL()
	if err != nil {
		return nil, err
	}
	ssc, err := NewSourceSpanWithContext(
		startLoc2,
		SourceLocation{
			Offset: endLoc2.Offset - 1,
			Line:   endLoc2.Line - 1,
			Column: lastLineLength(text),
		},
		text,
		context,
		sourceURL,
	)
	if err != nil {
		return nil, err
	}
	return ssc, nil
}

// lastLineLength returns the length of the last line in text, whether or
// not text ends in a newline. Matches Dart: _Highlight._lastLineLength
// (highlighter.dart).
func lastLineLength(text string) int {
	if len(text) == 0 {
		return 0
	}
	if text[len(text)-1] == '\n' {
		if len(text) == 1 {
			return 0
		}
		return len(text) - lastIndex(text, '\n', len(text)-2) - 1
	}
	return len(text) - lastIndex(text, '\n', len(text)-1) - 1
}

// isTextAtEndOfContext reports whether span's text runs all the way to the
// end of its context. Matches Dart: _Highlight._isTextAtEndOfContext
// (highlighter.dart), which normalizeTrailingNewline consults before
// trimming a final newline off the span text.
func isTextAtEndOfContext(span FileSpan) (bool, error) {
	context, err := span.Context()
	if err != nil {
		return false, err
	}
	if context == "" {
		context, err = span.SpanText()
		if err != nil {
			return false, err
		}
	}
	spanText, err := span.SpanText()
	if err != nil {
		return false, err
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		return false, err
	}
	ls, ok := findLineStart(context, spanText, startLoc.Column)
	if !ok {
		return false, nil
	}
	length, err := span.Length()
	if err != nil {
		return false, err
	}
	return ls+startLoc.Column+length == len(context), nil
}

// --- utilities ported from utils.dart ---

// findLineStart finds a line in context containing text at column.
//
// Matches Dart: findLineStart (source_span utils.dart). It returns the
// index in context where that line begins, or false when no line holds the
// text at that column. Empty text matches the first line with at least
// column characters; otherwise each occurrence is measured against its own
// line start (searching before the match in case text begins with a
// newline).
func findLineStart(context, text string, column int) (int, bool) {
	if len(text) == 0 {
		beginningOfLine := 0
		for {
			idx := strings.IndexByte(context[beginningOfLine:], '\n')
			if idx == -1 {
				return beginningOfLine, len(context)-beginningOfLine >= column
			}
			absIdx := beginningOfLine + idx
			if absIdx-beginningOfLine >= column {
				return beginningOfLine, true
			}
			beginningOfLine = absIdx + 1
		}
	}

	idx := 0
	for {
		found := strings.Index(context[idx:], text)
		if found == -1 {
			break
		}
		idx = idx + found
		lineStart := 0
		if idx > 0 {
			lineStart = lastIndex(context, '\n', idx-1) + 1
		}
		textColumn := idx - lineStart
		if column == textColumn {
			return lineStart, true
		}
		idx++
	}
	return 0, false
}

// isMultiline reports whether span covers multiple lines.
// Matches Dart: isMultiline (source_span utils.dart).
func isMultiline(span FileSpan) (bool, error) {
	startLoc, err := span.StartLocation()
	if err != nil {
		return false, err
	}
	endLoc, err := span.EndLocation()
	if err != nil {
		return false, err
	}
	return startLoc.Line != endLoc.Line, nil
}

// replaceFirstNull sets the first nil element of list to element.
//
// Matches Dart: replaceFirstNull (source_span utils.dart). Go shift: Dart
// throws ArgumentError when the list holds no nil; Go silently keeps the
// list unchanged (callers size the column list from maxMultilineSpans, so a
// missing slot cannot occur in practice).
func replaceFirstNull(list []*_highlight, element *_highlight) {
	for i, e := range list {
		if e == nil {
			list[i] = element
			return
		}
	}
}

// highlightEqual reports whether two column entries denote the same
// highlight: both nil, or equal primary flags with equal span endpoints.
//
// Go has no _Highlight.== (Dart compares by identity in the column lists),
// so endpoint comparison stands in — see highlightInColumn.
func highlightEqual(a, b *_highlight) (bool, error) {
	if a == nil || b == nil {
		return a == b, nil
	}
	aStart, err := a.span.StartLocation()
	if err != nil {
		return false, err
	}
	bStart, err := b.span.StartLocation()
	if err != nil {
		return false, err
	}
	aEnd, err := a.span.EndLocation()
	if err != nil {
		return false, err
	}
	bEnd, err := b.span.EndLocation()
	if err != nil {
		return false, err
	}
	return a.isPrimary == b.isPrimary &&
		aStart == bStart &&
		aEnd == bEnd, nil
}

// replaceWithNull sets the column entry matching element to nil, releasing
// its indicator column.
//
// Matches Dart: replaceWithNull (source_span utils.dart). Go shift: Dart
// throws ArgumentError when no entry matches; Go silently keeps the list
// unchanged.
func replaceWithNull(list []*_highlight, element *_highlight) error {
	for i, e := range list {
		eq, err := highlightEqual(e, element)
		if err != nil {
			return err
		}
		if eq {
			list[i] = nil
			return nil
		}
	}
	return nil
}

// containsHighlight reports whether list holds a highlight equal to
// element, by endpoint comparison (see highlightEqual).
func containsHighlight(list []*_highlight, element *_highlight) (bool, error) {
	for _, e := range list {
		eq, err := highlightEqual(e, element)
		if err != nil {
			return false, err
		}
		if eq {
			return true, nil
		}
	}
	return false, nil
}

// lenNonNil counts the occupied indicator columns in list.
func lenNonNil(list []*_highlight) int {
	count := 0
	for _, e := range list {
		if e != nil {
			count++
		}
	}
	return count
}

// sortHighlights orders highlights by start offset with a stable insertion
// sort. Matches Dart: the compareTo sort in _collateLines (highlighter.dart),
// except Dart breaks start ties by span length while the stable sort keeps
// the original order for equal starts.
func sortHighlights(highlights []_highlight) error {
	for i := 1; i < len(highlights); i++ {
		j := i
		for j > 0 {
			aLoc, err := highlights[j-1].span.StartLocation()
			if err != nil {
				return err
			}
			bLoc, err := highlights[j].span.StartLocation()
			if err != nil {
				return err
			}
			a := aLoc.Offset
			b := bLoc.Offset
			if a > b {
				highlights[j-1], highlights[j] = highlights[j], highlights[j-1]
			}
			j--
		}
	}
	return nil
}

// lastIndex returns the last index of byte b at or before from in s, or
// -1. This is the Go spelling of Dart's String.lastIndexOf with a start
// index, used by the line/column math throughout the highlighter.
func lastIndex(s string, b byte, from int) int {
	for i := from; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}
