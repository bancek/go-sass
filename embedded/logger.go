// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/logger.dart

import (
	"strconv"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/termglyph"
	"google.golang.org/protobuf/proto"
)

// EmbeddedLogger is a Sass logger that sends log messages to the host as
// LogEvent messages through the compilation dispatcher.
//
// The formatted text honors the color and ascii flags: color enables ANSI
// highlighting and ascii selects the plain-ASCII glyph set, matching the
// stderr logger's rendering so hosts see identical text.
//
// Matches Dart: final class EmbeddedLogger extends LoggerWithDeprecationType
// in logger.dart.
type EmbeddedLogger struct {
	// dispatcher is the compilation dispatcher that log events are sent
	// through. It corresponds to Dart's _dispatcher field.
	dispatcher *CompilationDispatcher
	// color reports whether the formatted message embeds terminal colors.
	// It corresponds to Dart's _color field (default false).
	color bool
	// ascii reports whether the formatted message uses ASCII glyphs.
	// It corresponds to Dart's _ascii field (default false).
	ascii bool
}

// NewEmbeddedLogger creates an EmbeddedLogger that sends events through
// dispatcher, rendering formatted text with color and ascii styling.
//
// Dart exposes color/ascii as named constructor defaults; Go takes them as
// explicit parameters.
//
// Matches Dart: EmbeddedLogger constructor in logger.dart.
func NewEmbeddedLogger(dispatcher *CompilationDispatcher, color, ascii bool) *EmbeddedLogger {
	return &EmbeddedLogger{
		dispatcher: dispatcher,
		color:      color,
		ascii:      ascii,
	}
}

// Debug sends a DEBUG LogEvent for message at span.
//
// The formatted line names the source URL (or "-" when absent), the 1-based
// line, and the message, with "Debug" bolded when color is on. Dart requires
// a non-null span; Go accepts a nil span and falls back to the bare message
// so host debug calls without location still produce a valid event.
//
// Matches Dart: EmbeddedLogger.debug in logger.dart.
func (l *EmbeddedLogger) Debug(message string, span *sasscommon.FileSpan) {
	formatted := l.formatDebug(message, span)

	l.dispatcher.sendLog(&OutboundMessage_LogEvent{
		Type:      LogEventType_DEBUG,
		Message:   message,
		Span:      protofySpan(span),
		Formatted: formatted,
	})
}

// Warn sends a WARNING LogEvent for message.
//
// It is the non-deprecation entry into the shared warning path; deprecation
// warnings go through WarnDeprecation instead.
//
// Matches Dart: EmbeddedLogger.internalWarn (no-deprecation case) in
// logger.dart.
func (l *EmbeddedLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	l.internalWarn(message, span, trace, nil)
}

// WarnDeprecation sends message as a DEPRECATION_WARNING LogEvent tagged with
// the deprecation ID.
//
// Dart funnels both cases through the internalWarn override; Go splits them
// into Warn and WarnDeprecation to satisfy the host logger interface while
// sharing the same formatting and dispatch below.
//
// Matches Dart: EmbeddedLogger.internalWarn (deprecation case) in logger.dart.
func (l *EmbeddedLogger) WarnDeprecation(
	message string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace,
) error {
	l.internalWarn(message, span, trace, dep)
	return nil
}

// internalWarn formats message and sends it as a WARNING or
// DEPRECATION_WARNING LogEvent depending on whether dep is set.
//
// The event carries the protofied span only when present, the stack trace
// only when non-empty, and the deprecation ID only for deprecations,
// mirroring Dart's conditional field assignment.
//
// Matches Dart: EmbeddedLogger.internalWarn in logger.dart.
func (l *EmbeddedLogger) internalWarn(
	message string, span *sasscommon.FileSpan, trace *sasscommon.Trace, dep *deprecation.Deprecation,
) {
	stack := trace.String()
	formatted := l.formatWarning(message, span, stack, dep)

	eventType := LogEventType_WARNING
	if dep != nil {
		eventType = LogEventType_DEPRECATION_WARNING
	}

	event := &OutboundMessage_LogEvent{
		Type:      eventType,
		Message:   message,
		Formatted: formatted,
	}
	if span != nil {
		event.Span = protofySpan(span)
	}
	if stack != "" {
		event.StackTrace = stack
	}
	if dep != nil {
		event.DeprecationType = proto.String(dep.ID)
	}
	l.dispatcher.sendLog(event)
}

// formatDebug renders a debug line as "<url>:<line> DEBUG: <message>\n".
//
// The URL falls back to "-" when the span has no source, the line is 1-based,
// and the label is bolded under color. Dart builds this inline in debug with
// prettyUri and term_glyph scoping; Go factors it out so both Debug and tests
// share one renderer.
//
// Matches Dart: EmbeddedLogger.debug formatted string in logger.dart.
func (l *EmbeddedLogger) formatDebug(message string, span *sasscommon.FileSpan) string {
	if span == nil {
		return message + "\n"
	}
	s := *span
	sourceURL, _ := s.SourceURL()
	urlStr := "-"
	if sourceURL != nil {
		urlStr = sourceURL.String()
	}
	start, _ := s.StartLocation()
	label := "DEBUG"
	if l.color {
		label = "\x1b[1mDebug\x1b[0m"
	}
	return urlStr + ":" + strconv.Itoa(start.Line+1) + " " + label + ": " + message + "\n"
}

// formatWarning renders a warning or deprecation warning the way the stderr
// logger would print it.
//
// The header is "WARNING" (or "DEPRECATION WARNING") with the deprecation ID
// in brackets, colorized when enabled; user-authored deprecations omit the ID
// because they carry no actionable migration target. With no span only the
// header and message print; with a span and a trace the message is followed
// by the highlighted span; with a span but no trace the span's own message
// line carries it. A trailing stack trace is trimmed and indented four
// spaces. Dart scopes glyph selection through withGlyphs; Go passes the
// resolved glyph/color options directly.
//
// Matches Dart: EmbeddedLogger.internalWarn formatted string in logger.dart.
func (l *EmbeddedLogger) formatWarning(
	message string, span *sasscommon.FileSpan, stack string, dep *deprecation.Deprecation,
) string {
	var buf strings.Builder
	showDeprecation := dep != nil && dep != deprecation.UserAuthored
	if l.color {
		buf.WriteString("\x1b[33m\x1b[1m")
		if dep != nil {
			buf.WriteString("Deprecation ")
		}
		buf.WriteString("Warning\x1b[0m")
		if showDeprecation {
			buf.WriteString(" [\x1b[34m" + dep.ID + "\x1b[0m]")
		}
	} else {
		if dep != nil {
			buf.WriteString("DEPRECATION ")
		}
		buf.WriteString("WARNING")
		if showDeprecation {
			buf.WriteString(" [" + dep.ID + "]")
		}
	}

	opts := sasscommon.HighlightOptions{}
	if l.ascii {
		opts.Glyphs = termglyph.AsciiGlyphs
	}
	if l.color {
		opts.Color = true
	}

	if span == nil {
		buf.WriteString(": " + message + "\n")
	} else {
		s := *span
		if stack != "" {
			buf.WriteString(": " + message + "\n\n")
			if hl, err := s.Highlight(opts); err == nil {
				buf.WriteString(hl)
			}
		} else {
			msgStr, _ := s.Message("\n"+message, opts)
			buf.WriteString(" on ")
			buf.WriteString(msgStr)
		}
	}
	if stack != "" {
		buf.WriteString("\n")
		buf.WriteString(indent(strings.TrimRight(stack, " \t\n\r"), 4))
	}
	return buf.String()
}

// indent prefixes each line of s with count spaces.
//
// Go-local copy of Dart's utils indent helper: the embedded package cannot
// import the CLI text utilities, so the four-space stack-trace indent used by
// formatWarning lives here. An empty input stays empty rather than gaining a
// stray prefix.
func indent(s string, count int) string {
	if s == "" {
		return ""
	}
	prefix := strings.Repeat(" ", count)
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
