// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/exception.dart

import (
	"fmt"
	"net/url"
	"strings"
)

// Frame represents a single stack frame in a Sass trace.
//
// It ports Dart's Frame from package:stack_trace: the library URI plus the
// 1-based line and column where the frame was entered, and the member
// (mixin, function, or stylesheet) under evaluation. A nil URI renders as
// "-" (unknown library).
//
// Matches Dart: Frame from package:stack_trace
type Frame struct {
	// URI is the canonical URL of the stylesheet that contributed the
	// frame, or nil when the library is unknown.
	URI *url.URL
	// Line is the 1-based line number of the frame.
	Line int
	// Column is the 1-based column number of the frame.
	Column int
	// Member names the Sass member under evaluation
	// (for example a function, mixin, or "root stylesheet").
	Member string
}

// String renders the frame as "library line:column  member", padding nothing;
// callers that align columns (Trace.String) pad around this form.
//
// Matches Dart: Frame.toString() => '$lib ${line}:${column}  $_member'
func (f Frame) String() string {
	var lib string
	if f.URI != nil {
		lib = PrettyUri(f.URI)
	} else {
		lib = "-"
	}
	return fmt.Sprintf("%s %d:%d  %s", lib, f.Line, f.Column, f.Member)
}

// location renders just the "library line:column" portion, shared by String
// and the trace-alignment width computation.
//
// Matches Dart: Frame.location
func (f Frame) location() string {
	var lib string
	if f.URI != nil {
		lib = PrettyUri(f.URI)
	} else {
		lib = "-"
	}
	return fmt.Sprintf("%s %d:%d", lib, f.Line, f.Column)
}

// Trace is a Sass stack trace: an ordered list of frames from the outermost
// call to the innermost, carried structurally through the evaluator and only
// stringified at the output boundary.
//
// It ports Dart's Trace from package:stack_trace, which the evaluator threads
// through each SassRuntimeException instead of capturing a host stack.
//
// Matches Dart: Trace from package:stack_trace
type Trace struct {
	// Frames holds the trace frames outermost first.
	Frames []Frame
}

// NewTrace creates a Trace holding the given frames. The slice is stored as
// provided, outermost frame first, matching Dart's Trace frame order.
func NewTrace(frames []Frame) *Trace {
	return &Trace{Frames: frames}
}

// String renders the trace one frame per line, padding each location to the
// width of the longest so member names align. A nil receiver renders empty,
// which lets callers stringify an absent trace without a nil check.
//
// Matches Dart: Trace.toString() — pads locations to align members.
func (t *Trace) String() string {
	if t == nil {
		return ""
	}

	longest := 0
	for _, frame := range t.Frames {
		loc := frame.location()
		if len(loc) > longest {
			longest = len(loc)
		}
	}

	var buf strings.Builder
	for i, frame := range t.Frames {
		if i > 0 {
			buf.WriteByte('\n')
		}
		loc := frame.location()
		buf.WriteString(loc)
		padding := longest - len(loc) + 2
		for range padding {
			buf.WriteByte(' ')
		}
		buf.WriteString(frame.Member)
	}
	return buf.String()
}

// FrameForSpan builds the single "root stylesheet"-style frame for a span,
// converting the span's 0-based start location to the 1-based line/column
// Dart reports. Columns count display characters (not bytes) so multi-byte
// source maps to the same columns Dart's UTF-16-based locations produce.
//
// Matches Dart: frameForSpan
func FrameForSpan(span FileSpan, member string) (Frame, error) {
	loc, err := span.StartLocation()
	if err != nil {
		return Frame{}, err
	}
	u, err := span.SourceURL()
	if err != nil {
		return Frame{}, err
	}
	column := loc.Column
	// Dart's SourceLocation columns count UTF-16 units, not bytes.
	if file, ferr := span.File(); ferr == nil && file != nil {
		column = file.CharacterColumn(loc.Offset)
	}
	return Frame{
		URI:    u,
		Line:   loc.Line + 1,
		Column: column + 1,
		Member: member,
	}, nil
}

// SourceSpanFormatException pairs a message with the offending span plus the
// original source text, so the parser can adjust zero-length spans before
// they become public errors. It is the innermost layer of the parser's
// ScanError → SourceSpanFormatException → SassFormatException chain.
//
// It ports Dart's SourceSpanFormatException from package:source_span, which
// likewise exposes the failing span, the full source, and the error offset.
type SourceSpanFormatException struct {
	// Message describes the failure.
	Message string
	// Span marks the source range that failed to parse.
	Span FileSpan
	// Source holds the original source text the span was sliced from.
	Source string
	// Cause chains the underlying error, if any.
	Cause error
}

// Error renders "location: message", falling back to an unknown-location
// prefix when the span cannot supply one.
func (e *SourceSpanFormatException) Error() string {
	loc, err := e.Span.StartLocation()
	if err != nil {
		return fmt.Sprintf("unknown location: %s", e.Message)
	}
	return fmt.Sprintf("%v: %s", loc, e.Message)
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *SourceSpanFormatException) Unwrap() error { return e.Cause }

// SassFormatException is the parse-failure error surfaced to callers: a
// SassException that additionally behaves as a format error by exposing the
// source text and the error offset.
//
// It ports Dart's sealed SassFormatException, whose source getter reads the
// whole file text and whose offset getter reads the span start.
//
// Matches Dart: SassFormatException (parsing category)
type SassFormatException struct {
	// Message describes the failure.
	Message string
	// Span marks the source range that failed to parse.
	Span FileSpan
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure,
	// stamped at the evaluate boundary even on failure.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *SassFormatException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" followed by the span highlight
// and the default single-frame trace, honoring the caller's color settings.
func (e *SassFormatException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.Highlight(opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	if trace, err := e.Trace(); err == nil {
		for frame := range strings.SplitSeq(trace.String(), "\n") {
			if frame == "" {
				continue
			}
			buf.WriteByte('\n')
			buf.WriteString("  ")
			buf.WriteString(frame)
		}
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *SassFormatException) Unwrap() error { return e.Cause }

// Source returns the full source text the error span was sliced from,
// mirroring Dart's format-exception source getter.
func (e *SassFormatException) Source() (string, error) {
	f, err := e.Span.File()
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", nil
	}
	return f.Text(), nil
}

// Offset returns the byte offset of the error (the span start),
// mirroring Dart's format-exception offset getter. It reports 0 when the
// span cannot supply a location rather than failing the render.
func (e *SassFormatException) Offset() int {
	loc, err := e.Span.StartLocation()
	if err != nil {
		return 0
	}
	return loc.Offset
}

// Trace returns the default single-frame "root stylesheet" trace derived
// from the error span. Parse errors carry no evaluation stack, so the span
// itself is the trace.
//
// Matches Dart: SassException.trace (inherited via SassFormatException extends SassException)
func (e *SassFormatException) Trace() (*Trace, error) {
	frame, err := FrameForSpan(e.Span, "root stylesheet")
	if err != nil {
		return nil, err
	}
	return NewTrace([]Frame{frame}), nil
}

// WithLoadedUrls returns a copy carrying the given loaded URLs, leaving the
// receiver untouched so errors can be stamped at the evaluate boundary
// without mutating the original.
//
// Matches Dart: SassFormatException.withLoadedUrls
func (e *SassFormatException) WithLoadedUrls(urls []*url.URL) *SassFormatException {
	return &SassFormatException{
		Message:    e.Message,
		Span:       e.Span,
		Cause:      e.Cause,
		LoadedUrls: urls,
	}
}

// SassException is the general spanned Sass failure: a message plus the
// source span it points at, an optional chained cause, and the canonical
// stylesheet URLs loaded before the compilation failed.
//
// It ports Dart's sealed SassException. Rendering prints "Error: message",
// the span highlight, and an indented stack trace; the trace defaults to a
// single frame derived from the span unless WithTrace attached an evaluation
// trace.
//
// Matches Dart: SassException (compile category)
type SassException struct {
	// Message describes the failure.
	Message string
	// Span marks the primary source range the error points at.
	Span FileSpan
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure,
	// reported even on failure paths.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *SassException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" followed by the span highlight
// and the indented trace, honoring the caller's color settings. A span that
// cannot highlight degrades to a placeholder line rather than dropping the
// message.
func (e *SassException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.Highlight(opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	if trace, err := e.Trace(); err == nil {
		for frame := range strings.SplitSeq(trace.String(), "\n") {
			if frame == "" {
				continue
			}
			buf.WriteByte('\n')
			buf.WriteString("  ")
			buf.WriteString(frame)
		}
	} else {
		buf.WriteByte('\n')
		buf.WriteString("  [error generating stack trace]")
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *SassException) Unwrap() error { return e.Cause }

// Trace returns the default single-frame "root stylesheet" trace derived
// from the error span; evaluation failures replace this via WithTrace.
func (e *SassException) Trace() (*Trace, error) {
	frame, err := FrameForSpan(e.Span, "root stylesheet")
	if err != nil {
		return nil, err
	}
	return NewTrace([]Frame{frame}), nil
}

// Returns a copy of this error as a SassRuntimeException carrying trace as
// its evaluation stack. Internal in Dart (SassException.withTrace); kept
// exported only so other packages can attach traces.
//
// Matches Dart: SassException.withTrace
func (e *SassException) WithTrace(trace *Trace) *SassRuntimeException {
	return &SassRuntimeException{
		Message:    e.Message,
		Span:       e.Span,
		Trace:      trace,
		Cause:      e.Cause,
		LoadedUrls: e.LoadedUrls,
	}
}

// ToCssString returns a CSS stylesheet that displays this error message
// above the current page: the full render as a comment plus a body::before
// rule carrying the message as its content.
//
// Non-ASCII code points are emitted as escapes so the sheet survives
// misdeclared HTTP encodings, and comment-closing sequences in the message
// are neutralized so the error cannot break out of the comment.
//
// Matches Dart: SassException.toCssString
func (e *SassException) ToCssString() string {
	return exceptionToCssString(e.Error())
}

// Returns a copy carrying urls as the loaded-URL set, leaving the receiver
// untouched. Internal in Dart (SassException.withLoadedUrls); kept exported
// only so the evaluate boundary can stamp URLs without mutating the error.
//
// Matches Dart: SassException.withLoadedUrls
func (e *SassException) WithLoadedUrls(urls []*url.URL) *SassException {
	return &SassException{
		Message:    e.Message,
		Span:       e.Span,
		Cause:      e.Cause,
		LoadedUrls: urls,
	}
}

// MultiSpanSassFormatException is the parse-failure error with secondary
// spans attached as labeled points of reference alongside the primary span.
//
// It ports Dart's sealed MultiSpanSassFormatException: a
// MultiSpanSassException that also behaves as a format error (source text
// plus error offset), used when a parse failure wants to point at more than
// one location.
//
// Matches Dart: MultiSpanSassFormatException (parsing category)
type MultiSpanSassFormatException struct {
	// Message describes the failure.
	Message string
	// Span is the primary span the error points at.
	Span FileSpan
	// PrimaryLabel labels the primary span to distinguish it from Secondary.
	PrimaryLabel string
	// Secondary maps each extra span to the label shown beside it.
	Secondary map[FileSpan]string
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *MultiSpanSassFormatException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" with the primary span plus all
// secondary spans highlighted together, then the indented trace, honoring
// the caller's color settings.
func (e *MultiSpanSassFormatException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.HighlightMultiple(e.PrimaryLabel, e.Secondary, opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	if trace, err := e.Trace(); err == nil {
		for _, frame := range strings.Split(trace.String(), "\n") {
			if frame == "" {
				continue
			}
			buf.WriteByte('\n')
			buf.WriteString("  ")
			buf.WriteString(frame)
		}
	} else {
		buf.WriteByte('\n')
		buf.WriteString("  [error generating stack trace]")
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *MultiSpanSassFormatException) Unwrap() error { return e.Cause }

// Trace returns the default single-frame "root stylesheet" trace derived
// from the primary span; parse errors carry no evaluation stack.
//
// Matches Dart: SassException.trace (inherited via MultiSpanSassFormatException extends MultiSpanSassException)
func (e *MultiSpanSassFormatException) Trace() (*Trace, error) {
	frame, err := FrameForSpan(e.Span, "root stylesheet")
	if err != nil {
		return nil, err
	}
	return NewTrace([]Frame{frame}), nil
}

// Returns a copy carrying urls as the loaded-URL set, leaving the receiver
// untouched. Internal in Dart; kept exported only for the evaluate-boundary
// stamping pass.
//
// Matches Dart: MultiSpanSassFormatException.withLoadedUrls
func (e *MultiSpanSassFormatException) WithLoadedUrls(urls []*url.URL) *MultiSpanSassFormatException {
	return &MultiSpanSassFormatException{
		Message:      e.Message,
		Span:         e.Span,
		PrimaryLabel: e.PrimaryLabel,
		Secondary:    e.Secondary,
		Cause:        e.Cause,
		LoadedUrls:   urls,
	}
}

// MultiSpanSassException is a SassException that highlights secondary spans
// beside the primary one, giving the user extra labeled context (for example
// the other side of a conflict).
//
// It ports Dart's MultiSpanSassException, which is both a SassException and
// a MultiSourceSpanException: primaryLabel distinguishes the primary span
// while Secondary maps each extra span to its label.
//
// Matches Dart: MultiSpanSassException
type MultiSpanSassException struct {
	// Message describes the failure.
	Message string
	// Span is the primary span the error points at.
	Span FileSpan
	// PrimaryLabel labels the primary span to distinguish it from Secondary.
	PrimaryLabel string
	// Secondary maps each extra span to the label shown beside it.
	Secondary map[FileSpan]string
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *MultiSpanSassException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" with the primary span plus all
// secondary spans highlighted together, then the indented trace, honoring
// the caller's color settings.
func (e *MultiSpanSassException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.HighlightMultiple(e.PrimaryLabel, e.Secondary, opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	if trace, err := e.Trace(); err == nil {
		for frame := range strings.SplitSeq(trace.String(), "\n") {
			if frame == "" {
				continue
			}
			buf.WriteByte('\n')
			buf.WriteString("  ")
			buf.WriteString(frame)
		}
	} else {
		buf.WriteByte('\n')
		buf.WriteString("  [error generating stack trace]")
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *MultiSpanSassException) Unwrap() error { return e.Cause }

// WithAdditionalSpan returns a copy with span added to the secondary set
// under label, preserving every existing secondary. The copy shares nothing
// mutable with the receiver, so callers can accumulate context freely.
//
// Matches Dart: MultiSpanSassException.withAdditionalSpan
func (e *MultiSpanSassException) WithAdditionalSpan(span FileSpan, label string) *MultiSpanSassException {
	secondary := make(map[FileSpan]string, len(e.Secondary)+1)
	for k, v := range e.Secondary {
		secondary[k] = v
	}
	secondary[span] = label
	return &MultiSpanSassException{
		Message:      e.Message,
		Span:         e.Span,
		PrimaryLabel: e.PrimaryLabel,
		Secondary:    secondary,
		Cause:        e.Cause,
		LoadedUrls:   e.LoadedUrls,
	}
}

// ToCssString renders this error as a display-above-the-page stylesheet,
// sharing SassException's comment-plus-body::before encoding.
//
// Matches Dart: SassException.toCssString
func (e *MultiSpanSassException) ToCssString() string {
	return exceptionToCssString(e.Error())
}

// Trace returns the default single-frame "root stylesheet" trace derived
// from the primary span.
//
// Matches Dart: SassException.trace
func (e *MultiSpanSassException) Trace() (*Trace, error) {
	frame, err := FrameForSpan(e.Span, "root stylesheet")
	if err != nil {
		return nil, err
	}
	return NewTrace([]Frame{frame}), nil
}

// Returns a copy carrying urls as the loaded-URL set, leaving the receiver
// untouched. Internal in Dart; kept exported only for the evaluate-boundary
// stamping pass.
//
// Matches Dart: MultiSpanSassException.withLoadedUrls
func (e *MultiSpanSassException) WithLoadedUrls(urls []*url.URL) *MultiSpanSassException {
	return &MultiSpanSassException{
		Message:      e.Message,
		Span:         e.Span,
		PrimaryLabel: e.PrimaryLabel,
		Secondary:    e.Secondary,
		Cause:        e.Cause,
		LoadedUrls:   urls,
	}
}

// Converts this error into a MultiSpanSassException by keeping the primary
// span unlabeled and attaching span under label as the first secondary.
// Internal in Dart (SassException.withAdditionalSpan); kept exported only so
// evaluation sites can add reference spans without knowing the concrete type.
//
// Matches Dart: SassException.withAdditionalSpan
func (e *SassException) WithAdditionalSpan(span FileSpan, label string) *MultiSpanSassException {
	return &MultiSpanSassException{
		Message:      e.Message,
		Span:         e.Span,
		PrimaryLabel: "",
		Secondary:    map[FileSpan]string{span: label},
		Cause:        e.Cause,
		LoadedUrls:   e.LoadedUrls,
	}
}

// ScanError is the scanner-level failure: a message plus the span that
// could not be scanned, with an optional chained cause. The parser catches
// these (rather than public errors) so it can adjust zero-length spans
// before converting them up the error chain.
//
// It ports the error shape Dart's string_scanner surfaces for span-aware
// scan failures.
type ScanError struct {
	// Message describes what the scanner expected.
	Message string
	// Span marks the source range where scanning failed.
	Span FileSpan
	// Cause chains the underlying error, if any.
	Cause error
}

// Error returns the plain message without span rendering; span context is
// added when the parser converts this into a public error.
func (e *ScanError) Error() string { return e.Message }

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *ScanError) Unwrap() error { return e.Cause }

// ThrowWithTrace attaches cause as the chained cause of newErr and returns
// newErr, so each visit site can layer context onto the error propagating
// upward. Unknown error shapes cannot carry a cause, so they are wrapped in
// a new chained error instead.
//
// It ports Dart's throwWithTrace helper.
//
// Matches Dart: throwWithTrace
func ThrowWithTrace(newErr, cause error) error {
	switch e := newErr.(type) {
	case *SassException:
		e.Cause = cause
	case *SassFormatException:
		e.Cause = cause
	case *MultiSpanSassException:
		e.Cause = cause
	case *MultiSpanSassFormatException:
		e.Cause = cause
	case *SassRuntimeException:
		e.Cause = cause
	case *MultiSpanSassRuntimeException:
		e.Cause = cause
	case *SourceSpanFormatException:
		e.Cause = cause
	case *ScanError:
		e.Cause = cause
	default:
		return fmt.Errorf("%w: %v", cause, newErr)
	}
	return newErr
}

// SassRuntimeException is the evaluation-failure error: a SassException plus
// the Sass stack trace captured where the error was thrown.
//
// It ports Dart's SassRuntimeException. Unlike the base type, the trace is
// stored (not derived from the span), because only the evaluator knows the
// call stack; an empty trace means "derive a single frame from the span".
//
// Matches Dart: SassRuntimeException
type SassRuntimeException struct {
	// Message describes the failure.
	Message string
	// Span marks the source range under evaluation when the error was thrown.
	Span FileSpan
	// Trace is the Sass call stack at the throw point.
	Trace *Trace
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *SassRuntimeException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" with the span highlight and the
// stored trace's frames indented beneath, honoring the caller's color
// settings. A nil trace contributes no frames.
func (e *SassRuntimeException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.Highlight(opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	var traceStr string
	if e.Trace != nil {
		traceStr = e.Trace.String()
	}
	for frame := range strings.SplitSeq(traceStr, "\n") {
		if frame == "" {
			continue
		}
		buf.WriteByte('\n')
		buf.WriteString("  ")
		buf.WriteString(frame)
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *SassRuntimeException) Unwrap() error { return e.Cause }

// ToCssString renders this error as a display-above-the-page stylesheet,
// sharing SassException's comment-plus-body::before encoding.
//
// Matches Dart: SassException.toCssString
func (e *SassRuntimeException) ToCssString() string {
	return exceptionToCssString(e.Error())
}

// WithAdditionalSpan converts this error into a
// MultiSpanSassRuntimeException, keeping the primary span unlabeled and
// attaching span under label as the first secondary. The stored trace,
// cause, and loaded URLs carry over unchanged.
//
// Matches Dart: SassRuntimeException.withAdditionalSpan
func (e *SassRuntimeException) WithAdditionalSpan(span FileSpan, label string) *MultiSpanSassRuntimeException {
	return &MultiSpanSassRuntimeException{
		Message:        e.Message,
		Span:           e.Span,
		PrimaryLabel:   "",
		SecondarySpans: map[FileSpan]string{span: label},
		Trace:          e.Trace,
		Cause:          e.Cause,
		LoadedUrls:     e.LoadedUrls,
	}
}

// Returns a copy carrying urls as the loaded-URL set, leaving the receiver
// untouched. Internal in Dart (SassRuntimeException.withLoadedUrls); kept
// exported only for the evaluate-boundary stamping pass.
//
// Matches Dart: SassRuntimeException.withLoadedUrls
func (e *SassRuntimeException) WithLoadedUrls(urls []*url.URL) *SassRuntimeException {
	return &SassRuntimeException{
		Message:    e.Message,
		Span:       e.Span,
		Trace:      e.Trace,
		Cause:      e.Cause,
		LoadedUrls: urls,
	}
}

// MultiSpanSassRuntimeException is a SassRuntimeException that highlights
// secondary spans beside the primary one, each with its own label.
//
// It ports Dart's MultiSpanSassRuntimeException: both a
// MultiSpanSassException and a SassRuntimeException, storing the evaluation
// trace alongside the primary/secondary span set.
//
// Matches Dart: MultiSpanSassRuntimeException
type MultiSpanSassRuntimeException struct {
	// Message describes the failure.
	Message string
	// Span is the primary span the error points at.
	Span FileSpan
	// PrimaryLabel labels the primary span to distinguish it from SecondarySpans.
	PrimaryLabel string
	// SecondarySpans maps each extra span to the label shown beside it.
	SecondarySpans map[FileSpan]string
	// Trace is the Sass call stack at the throw point.
	Trace *Trace
	// Cause chains the underlying error, if any.
	Cause error
	// LoadedUrls lists canonical stylesheet URLs loaded before the failure.
	LoadedUrls []*url.URL
}

// Error renders the error with default (non-color) highlight options.
func (e *MultiSpanSassRuntimeException) Error() string {
	return e.ErrorWithOptions(HighlightOptions{})
}

// ErrorWithOptions renders "Error: message" with the primary span plus all
// secondary spans highlighted together and the stored trace's frames
// indented beneath, honoring the caller's color settings.
func (e *MultiSpanSassRuntimeException) ErrorWithOptions(opts HighlightOptions) string {
	var buf strings.Builder
	buf.WriteString("Error: ")
	buf.WriteString(e.Message)
	buf.WriteByte('\n')
	if highlight, err := e.Span.HighlightMultiple(e.PrimaryLabel, e.SecondarySpans, opts); err == nil {
		buf.WriteString(highlight)
	} else {
		buf.WriteString("[error highlighting source]\n")
	}
	var traceStr string
	if e.Trace != nil {
		traceStr = e.Trace.String()
	}
	for frame := range strings.SplitSeq(traceStr, "\n") {
		if frame == "" {
			continue
		}
		buf.WriteByte('\n')
		buf.WriteString("  ")
		buf.WriteString(frame)
	}
	return buf.String()
}

// Unwrap returns the chained cause, supporting errors.As/Is inspection.
func (e *MultiSpanSassRuntimeException) Unwrap() error { return e.Cause }

// ToCssString renders this error as a display-above-the-page stylesheet,
// sharing SassException's comment-plus-body::before encoding.
//
// Matches Dart: SassException.toCssString
func (e *MultiSpanSassRuntimeException) ToCssString() string {
	return exceptionToCssString(e.Error())
}

// Returns a copy carrying urls as the loaded-URL set, leaving the receiver
// untouched. Internal in Dart; kept exported only for the evaluate-boundary
// stamping pass.
//
// Matches Dart: MultiSpanSassRuntimeException.withLoadedUrls
func (e *MultiSpanSassRuntimeException) WithLoadedUrls(urls []*url.URL) *MultiSpanSassRuntimeException {
	return &MultiSpanSassRuntimeException{
		Message:        e.Message,
		Span:           e.Span,
		PrimaryLabel:   e.PrimaryLabel,
		SecondarySpans: e.SecondarySpans,
		Trace:          e.Trace,
		Cause:          e.Cause,
		LoadedUrls:     urls,
	}
}

// SassScriptException is the unspanned error built-in callables raise. It
// deliberately carries no span: Sass internals catch it at the evaluation
// boundary and convert it into a spanned SassRuntimeException with a stack
// trace, so throwing sites never need source locations in hand.
//
// It ports Dart's SassScriptException, whose message getter prefixes the
// argument name as "$name: message" when one is set.
//
// Matches Dart: SassScriptException
type SassScriptException struct {
	// Message is the error text without any argument-name prefix.
	Message string
	// ArgumentName names the Sass function argument that triggered the
	// error; Error renders it as a "$name: message" prefix when set.
	ArgumentName string
}

// Error renders the message, prefixed with "$argumentName: " when the error
// is tied to a specific Sass function argument.
func (e *SassScriptException) Error() string {
	if e.ArgumentName != "" {
		return fmt.Sprintf("$%s: %s", e.ArgumentName, e.Message)
	}
	return e.Message
}

// WithSpan converts this unspanned error into a SassException pointing at
// span, which is how the evaluator attaches a location at the boundary. The
// message carries the "$name: " prefix when ArgumentName is set, matching
// Dart's constructor-time message composition (withSpan forwards the
// already-composed message).
//
// Matches Dart: SassScriptException.withSpan
func (e *SassScriptException) WithSpan(span FileSpan) *SassException {
	return &SassException{
		Message: e.Error(),
		Span:    span,
	}
}

// NewSassScriptException creates an unspanned script error with the given
// message. A non-nil argumentName ties the error to that Sass function
// argument, so Error renders it as a "$name: message" prefix, matching
// Dart's constructor-time message composition without storing the prefix.
//
// It ports Dart's SassScriptException constructor, whose optional
// argumentName defaults to absent.
//
// Matches Dart: SassScriptException constructor
func NewSassScriptException(message string, argumentName *string) *SassScriptException {
	name := ""
	if argumentName != nil {
		name = *argumentName
	}
	return &SassScriptException{Message: message, ArgumentName: name}
}

// MultiSpanSassScriptException is the unspanned error with secondary spans
// attached as labeled points of reference. Like its single-span sibling it
// gains its primary span later, at the evaluation boundary.
//
// It ports Dart's MultiSpanSassScriptException: primaryLabel distinguishes
// the eventual primary span while SecondarySpans maps each extra span to
// its label.
//
// Matches Dart: MultiSpanSassScriptException
type MultiSpanSassScriptException struct {
	// Message is the error text without any argument-name prefix.
	Message string
	// PrimaryLabel labels the eventual primary span to distinguish it from
	// SecondarySpans.
	PrimaryLabel string
	// SecondarySpans maps each extra span to the label shown beside it.
	SecondarySpans map[FileSpan]string
}

// Error returns the plain message; span rendering happens once the boundary
// attaches the primary span.
func (e *MultiSpanSassScriptException) Error() string {
	return e.Message
}

// exceptionToCssString renders an error message as a CSS stylesheet that
// displays the error above the current page: the full render as a comment
// plus a body::before rule carrying the message as its content.
//
// The comment body neutralizes comment-closing sequences so the message
// cannot break out of the comment, and normalizes CRLF to LF to match the
// rest of the document. The content string escapes every non-US-ASCII rune
// so it survives misdeclared HTTP encodings.
//
// Matches Dart: SassException.toCssString
func exceptionToCssString(message string) string {
	// Keep comment-closers in the message from terminating the comment, and
	// normalize newlines before reflowing into "* "-prefixed lines.
	commentMessage := strings.ReplaceAll(message, "*/", "*∕")
	commentMessage = strings.ReplaceAll(commentMessage, "\r\n", "\n")

	var contentBuf strings.Builder
	for _, r := range message {
		if r > 0x7F {
			// Emit non-ASCII runes as CSS escapes; ASCII passes through.
			fmt.Fprintf(&contentBuf, "\\%x ", r)
		} else {
			contentBuf.WriteRune(r)
		}
	}

	var commentBuf strings.Builder
	commentBuf.WriteString("/* ")
	commentBuf.WriteString(strings.Join(strings.Split(commentMessage, "\n"), "\n * "))
	commentBuf.WriteString(" */")

	return fmt.Sprintf("%s\n\nbody::before {\n  font-family: \"Source Code Pro\", \"SF Mono\", Monaco, Inconsolata, \"Fira Mono\",\n      \"Droid Sans Mono\", monospace, monospace;\n  white-space: pre;\n  display: block;\n  padding: 1em;\n  margin-bottom: 1em;\n  border-bottom: 2px solid black;\n  content: %s;\n}",
		commentBuf.String(), contentBuf.String())
}
