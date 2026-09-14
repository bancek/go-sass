// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:string_scanner/lib/src/span_scanner.dart

import (
	"fmt"
	"net/url"
	"unicode/utf8"
)

// LineScannerState represents a saved scanner state for backtracking.
//
// Matches Dart: LineScannerState (string_scanner line_scanner.dart): the
// scanner position plus its zero-based line and column, used to save and
// restore the scanner when backtracking. Go shift: Dart's state holds a
// back-reference to the scanner that created it and SetState rejects
// foreign states; Go's is a plain value with no owner check. Match
// information is not included, matching Dart.
type LineScannerState struct {
	// Position is the byte offset of the scanner in this state. (Dart
	// counts characters/UTF-16 code units; Go counts bytes.)
	Position int
	// Line is the zero-based line number of the scanner in this state.
	Line int
	// Column is the zero-based column number of the scanner in this state.
	Column int
}

// SpanScanner scans a source with line/column tracking and FileSpan
// generation, matching Dart's SpanScanner: a LineScanner that exposes
// matched ranges as FileSpans, backed by a SourceFile that caches line
// breaks and mints the spans.
//
// It decodes full UTF-8 runes (code points) rather than
// individual bytes, while keeping [pos] as a byte offset so that
// Substring, SpanFromTo, and Rest work correctly on the underlying byte
// slice. (Dart positions count UTF-16 code units, with surrogate-pair
// handling for supplementary-plane characters; Go's UTF-8 decoding needs
// no surrogate logic.)
//
// Go shift: Dart's Pattern-based match API (scan/matches with lastMatch
// and lastSpan) is absent — Go offers exact-string Scan plus the
// character-level ScanChar/ReadChar loops the Sass parser actually uses.
// Dart's eager and within factories have no counterpart either.
type SpanScanner struct {
	source *FileSource
	pos    int
	line   int
	column int
}

// NewSpanScanner creates a new scanner that scans [source].
// [sourceURL] is used for source span URLs (may be nil).
//
// Matches Dart: SpanScanner(String string, {sourceUrl, position})
// (span_scanner.dart), which wraps the text in a SourceFile the same way;
// sourceUrl feeds both the emitted FileSpans and error reporting. Go
// always starts at position 0 (Dart's initial-position argument has no
// counterpart; callers use SetPosition instead).
func NewSpanScanner(source []byte, sourceURL *url.URL) *SpanScanner {
	return &SpanScanner{
		source: NewFileSource(source, sourceURL),
	}
}

// Text returns the full string being scanned.
// Matches Dart: StringScanner.string (string_scanner.dart).
func (s *SpanScanner) Text() string { return s.source.Text() }

// SourceURL returns the URL of the scanned source, used for error
// reporting. May be nil when the source URL is unknown or unavailable.
// Matches Dart: StringScanner.sourceUrl (string_scanner.dart).
func (s *SpanScanner) SourceURL() *url.URL { return s.source.URL() }

// IsDone reports whether the scanner has completely consumed the source.
// Matches Dart: StringScanner.isDone (string_scanner.dart).
func (s *SpanScanner) IsDone() bool { return s.pos >= s.source.Length() }

// Position returns the current scanner position as a byte offset. (Dart's
// StringScanner.position counts characters instead.)
func (s *SpanScanner) Position() int { return s.pos }

// Line returns the scanner's current zero-based line number.
// Matches Dart: LineScanner.line (line_scanner.dart).
func (s *SpanScanner) Line() int { return s.line }

// Column returns the scanner's current zero-based column number.
// Matches Dart: LineScanner.column (line_scanner.dart).
func (s *SpanScanner) Column() int { return s.column }

// Rest returns the un-scanned portion of the source.
// Matches Dart: StringScanner.rest (string_scanner.dart).
func (s *SpanScanner) Rest() string { return s.source.Text()[s.pos:] }

// PeekChar returns the Unicode code point [offset] bytes from the current
// position. Negative offsets (specifically -1) look backwards. Returns
// -1 if outside the string bounds.
//
// Matches Dart: StringScanner.peekChar (string_scanner.dart), which returns
// the character code at position+offset (null outside the string) without
// affecting the last match. Go shift: offsets are bytes, the missing
// character reports as -1 rather than null, only -1 looks backwards, and a
// full code point is decoded (Dart returns a UTF-16 code unit, with
// peekCodePoint for surrogate-pair decoding).
func (s *SpanScanner) PeekChar(offset int) int {
	if offset < 0 {
		if offset != -1 {
			return -1
		}
		r, _ := utf8.DecodeLastRuneInString(s.source.Text()[:s.pos])
		return int(r)
	}
	idx := s.pos + offset
	if idx < 0 || idx >= s.source.Length() {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(s.source.Text()[idx:])
	return int(r)
}

// ReadChar consumes and returns the current Unicode code point.
// Returns error if at end of input.
//
// Matches Dart: StringScanner.readChar plus LineScanner's override
// (string_scanner.dart, line_scanner.dart): consuming one character and
// adjusting the line/column after it. Go shift: Dart throws FormatException
// ("expected more input") at end of input; Go returns a *ScanError with an
// empty span. Invalid UTF-8 is a Go-only error with no Dart counterpart
// (Dart strings are always well-formed UTF-16).
func (s *SpanScanner) ReadChar() (int, error) {
	if s.pos >= s.source.Length() {
		return 0, s.errCurrent("expected more input.")
	}
	r, size := utf8.DecodeRuneInString(s.source.Text()[s.pos:])
	if r == utf8.RuneError && size == 1 {
		return 0, s.errCurrent("Invalid UTF-8.")
	}
	s.pos += size
	s.advanceLineCol(r)
	return int(r), nil
}

// ScanChar tries to consume [ch]. Returns true if consumed.
//
// Matches Dart: StringScanner.scanChar plus LineScanner's override
// (string_scanner.dart, line_scanner.dart). Dart's surrogate-pair branch
// for supplementary-plane characters collapses into the single UTF-8
// decode here; the line/column adjustment runs only on success, as in
// Dart.
func (s *SpanScanner) ScanChar(ch int) bool {
	r, size := utf8.DecodeRuneInString(s.source.Text()[s.pos:])
	if int(r) != ch {
		return false
	}
	s.pos += size
	s.advanceLineCol(r)
	return true
}

// ExpectChar consumes [ch] or returns an error.
//
// Matches Dart: StringScanner.expectChar (string_scanner.dart), including
// the special backslash name in the failure message. Go shift: Dart throws
// the FormatException; Go returns a *ScanError at the current position.
func (s *SpanScanner) ExpectChar(ch int) error {
	if s.ScanChar(ch) {
		return nil
	}
	var name string
	if ch == '\\' {
		name = `"\\"`
	} else {
		name = fmt.Sprintf(`"%c"`, ch)
	}
	return s.errCurrent("expected %s.", name)
}

// Scan tries to consume the exact string [str]. Returns true if consumed,
// false if any character didn't match (without consuming anything).
//
// This plays the role of Dart's StringScanner.scan/LineScanner.scan for the
// parser's needs, restricted to exact strings (Dart accepts any Pattern and
// records lastMatch). The save/restore around the character loop is a Go
// adaptation: Dart's prefix match is atomic, while the loop must rewind
// explicitly on a partial mismatch.
func (s *SpanScanner) Scan(str string) bool {
	state := s.State()
	for i := 0; i < len(str); i++ {
		if !s.ScanChar(int(str[i])) {
			s.SetState(state)
			return false
		}
	}
	return true
}

// Expect consumes the exact string [str] or returns an error.
//
// Matches Dart: StringScanner.expect (string_scanner.dart), which throws a
// FormatException naming the expected pattern (Dart renders the pattern
// itself when no name is given; Go always quotes the literal with %q). The
// error span starts where the attempt began, and consumed characters are
// not rewound — matching Dart, which likewise leaves the scanner after a
// failed expect.
func (s *SpanScanner) Expect(str string) error {
	start := s.pos
	for i := 0; i < len(str); i++ {
		if !s.ScanChar(int(str[i])) {
			return s.errAt(start, "expected %q.", str)
		}
	}
	return nil
}

// ExpectDone returns an error if there's remaining input.
//
// Matches Dart: StringScanner.expectDone (string_scanner.dart), which
// throws ("expected no more input") unless the string is fully consumed.
// Go returns the *ScanError instead of throwing.
func (s *SpanScanner) ExpectDone() error {
	if !s.IsDone() {
		return s.errCurrent("expected no more input.")
	}
	return nil
}

// Substring returns the source bytes from [start] to [end] as a string
// (defaults to current position).
//
// Matches Dart: StringScanner.substring (string_scanner.dart). Like Dart,
// end defaults to the current position rather than the end of the string;
// unlike Dart, offsets are bytes, not characters.
func (s *SpanScanner) Substring(start int, end *int) string {
	e := s.pos
	if end != nil {
		e = *end
	}
	return s.source.Text()[start:e]
}

// State returns the current scanner state for backtracking.
//
// Matches Dart: LineScanner.state (line_scanner.dart): position plus line
// and column, for efficiently saving and restoring the scanner. Match
// information is not included, matching Dart.
func (s *SpanScanner) State() LineScannerState {
	return LineScannerState{Position: s.pos, Line: s.line, Column: s.column}
}

// SetState restores a previously saved state.
//
// Matches Dart: LineScanner.state= (line_scanner.dart). Go shift: Dart
// rejects states created by a different scanner with ArgumentError; Go's
// state is a plain value and restores unconditionally.
func (s *SpanScanner) SetState(st LineScannerState) {
	s.pos = st.Position
	s.line = st.Line
	s.column = st.Column
}

// SetPosition sets the scanner position and recomputes line/column.
//
// Matches Dart: LineScanner.position= (line_scanner.dart), which likewise
// re-derives the line and column after a jump (scanning newlines forward,
// searching backward for the last newline). Go shift: Dart throws
// ArgumentError on an invalid position; Go returns a *ScanError carrying an
// empty span. The recompute here is a binary search over the FileSource's
// line starts (linecol) rather than Dart's regex newline walk.
func (s *SpanScanner) SetPosition(pos int) error {
	if pos < 0 || pos > s.source.Length() {
		msg := fmt.Sprintf("invalid position %d (source length %d)", pos, s.source.Length())
		return &ScanError{Message: msg, Span: s.EmptySpan()}
	}
	s.pos = pos
	s.line, s.column = s.linecol(pos)
	return nil
}

// Location returns the current location.
//
// Matches Dart: SpanScanner.location (span_scanner.dart). Dart returns a
// FileLocation whose line/column resolve lazily through the source file; Go
// returns a SourceLocation value computed eagerly.
func (s *SpanScanner) Location() SourceLocation {
	return s.locationFor(s.pos)
}

// EmptySpan returns a zero-length span at the current position.
//
// Matches Dart: SpanScanner.emptySpan (span_scanner.dart), a point span at
// the current location.
func (s *SpanScanner) EmptySpan() FileSpan {
	return s.SpanFromTo(s.pos, s.pos)
}

// SpanFrom creates a FileSpan from [start] state to the current position.
//
// Matches Dart: SpanScanner.spanFrom (span_scanner.dart). Go covers only
// the single-state form; Dart's optional end-state argument has no
// counterpart (callers needing both endpoints use SpanFromTo).
func (s *SpanScanner) SpanFrom(start LineScannerState) FileSpan {
	return s.SpanFromTo(start.Position, s.pos)
}

// SpanFromPosition creates a FileSpan from [startPos] to the current position.
//
// Matches Dart: SpanScanner.spanFromPosition (span_scanner.dart). Each
// position is an offset into the scanned source with StringScanner
// conventions — bytes here, code units in Dart. The span is built
// directly, so (as in Dart, which throws RangeError from the file) callers
// must pass positions within the source.
func (s *SpanScanner) SpanFromPosition(startPos int) FileSpan {
	return s.SpanFromTo(startPos, s.pos)
}

// SpanFromTo creates a FileSpan between two positions.
//
// This is the shared constructor behind SpanFrom and SpanFromPosition,
// matching the SourceFile.span call at the heart of Dart's spanFrom and
// spanFromPosition (span_scanner.dart).
func (s *SpanScanner) SpanFromTo(startPos, endPos int) FileSpan {
	return SimpleFileSpan{
		file:  s.source,
		start: startPos,
		end:   endPos,
	}
}

// Error creates a *ScanError at the given position/length.
// If position < 0, uses current position. If length < 0, uses 0.
//
// Matches Dart: SpanScanner.error/StringScanner.error (span_scanner.dart,
// string_scanner.dart), which throws a StringScannerException over a file
// span at the resolved position. Go shift: negative sentinels select the
// defaults (Dart uses nulls), there is no match argument (Go tracks no
// lastMatch), and the error is returned rather than thrown so the parser
// can adjust the span before it becomes public.
func (s *SpanScanner) Error(msg string, position, length int) error {
	if position < 0 {
		position = s.pos
	}
	if length < 0 {
		length = 0
	}
	return &ScanError{
		Message: msg,
		Span:    s.SpanFromTo(position, position+length),
	}
}

// errCurrent creates a *ScanError at the current position.
//
// This is the Go form of Dart's StringScanner._fail (string_scanner.dart):
// formatting "expected <name>." at the scanner's current spot.
func (s *SpanScanner) errCurrent(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return &ScanError{Message: msg, Span: s.EmptySpan()}
}

// errAt creates a *ScanError at the given position.
//
// Like errCurrent for errors attributed to an earlier offset (for example
// Expect's failure at the attempt's start), matching how Dart's error()
// resolves an explicit position into a file span.
func (s *SpanScanner) errAt(pos int, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return &ScanError{Message: msg, Span: s.SpanFromTo(pos, pos)}
}

// advanceLineCol updates line/column after consuming a rune.
//
// Matches Dart: LineScanner._adjustLineAndColumn (line_scanner.dart): a line
// feed — or a carriage return not immediately followed by a line feed —
// starts a new line, anything else advances the column. The CR lookahead
// (PeekChar(0), since the CR is already consumed) is Dart's _betweenCRLF
// guard. Go counts one column per rune; Dart counts supplementary-plane
// characters as two columns (UTF-16 units).
func (s *SpanScanner) advanceLineCol(r rune) {
	if r == '\n' || (r == '\r' && s.PeekChar(0) != '\n') {
		s.line++
		s.column = 0
	} else {
		s.column++
	}
}

// linecol returns the line and column for a given position.
//
// Binary search over the FileSource's line starts, matching what
// FileSource.GetLine/GetColumn compute; SetPosition and locationFor share
// it so jumps and span building agree. (Dart re-derives these with a regex
// newline walk in LineScanner.position= instead.)
func (s *SpanScanner) linecol(pos int) (int, int) {
	lo, hi := 0, len(s.source.lineStarts)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s.source.lineStarts[mid] <= pos {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	line := max(lo-1, 0)
	col := pos - s.source.lineStarts[line]
	return line, col
}

// locationFor returns a SourceLocation for any position in the source.
//
// Matches Dart: the _sourceFile.location(position) call behind
// SpanScanner.location (span_scanner.dart), with line/column filled in
// eagerly (see Location).
func (s *SpanScanner) locationFor(pos int) SourceLocation {
	line, col := s.linecol(pos)
	return SourceLocation{
		Offset: pos,
		Line:   line,
		Column: col,
	}
}
