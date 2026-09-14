// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger/stderr.dart

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/termglyph"
)

// ANSI terminal color escape sequences used by StderrLogger.
const (
	stderrAnsiBold   = "\x1b[1m"
	stderrAnsiYellow = "\x1b[33m"
	stderrAnsiBlue   = "\x1b[34m"
	stderrAnsiReset  = "\x1b[0m"
)

// StderrLogger writes warnings and debug messages to stderr with optional
// ANSI color support. It is the stderr factory target, backing both the
// explicit stderr constructor and the auto-detected default logger.
//
// Matches Dart: StderrLogger in lib/src/logger/stderr.dart (a
// LoggerWithDeprecationType writing via printError).
type StderrLogger struct {
	color bool
	// Unicode selects the glyph set for source-frame highlighting; false
	// renders the ASCII fallback (Dart reads the global glyph.ascii flag).
	Unicode bool
	// Writer is the output sink; when nil, os.Stderr is used. Tests inject
	// a buffer here. Go-only seam — Dart writes straight to printError.
	Writer io.Writer // if nil, defaults to os.Stderr
}

// NewStderrLogger creates a new StderrLogger that writes to os.Stderr.
// Matches Dart's stderr factory (color) plus an explicit unicode choice.
func NewStderrLogger(color bool, unicode bool) *StderrLogger {
	return &StderrLogger{color: color, Unicode: unicode}
}

// NewStderrLoggerWithWriter creates a new StderrLogger that writes to w.
// Go-only test seam with no Dart counterpart.
func NewStderrLoggerWithWriter(color bool, unicode bool, w io.Writer) *StderrLogger {
	return &StderrLogger{color: color, Unicode: unicode, Writer: w}
}

// Warn emits a plain warning: the internalWarn path with no deprecation,
// so the header reads "Warning" rather than "Deprecation Warning".
// Matches Dart: StderrLogger warn via internalWarn (deprecation: null).
func (l *StderrLogger) Warn(msg string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	l.internalWarn(msg, span, trace, nil)
}

// WarnDeprecation emits a deprecation warning and always reports success —
// a stderr logger never fails the compilation itself (fatal handling lives
// in DeprecationProcessingLogger).
// Matches Dart: StderrLogger warn via internalWarn (deprecation given).
func (l *StderrLogger) WarnDeprecation(msg string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace) error {
	l.internalWarn(msg, span, trace, dep)
	return nil
}

// internalWarn renders and prints one warning. The header is bold yellow
// "Warning" (color) or plain "WARNING", prefixed with "Deprecation " and
// suffixed with the deprecation id in blue — except user-authored
// deprecations, which show no id since they carry no catalog entry. When
// both span and trace are present the span only highlights (its location is
// duplicated in the trace); with a span alone the message is anchored onto
// the span text; highlight failures fall back to the bare header. The
// trace, trimmed and indented, closes the output.
// Matches Dart: StderrLogger.internalWarn.
func (l *StderrLogger) internalWarn(msg string, span *sasscommon.FileSpan, trace *sasscommon.Trace, dep *deprecation.Deprecation) {
	stack := trace.String()
	var result strings.Builder
	showDeprecation := dep != nil && dep != deprecation.UserAuthored

	if l.color {
		result.WriteString(stderrAnsiYellow)
		result.WriteString(stderrAnsiBold)
		if dep != nil {
			result.WriteString("Deprecation ")
		}
		result.WriteString("Warning")
		result.WriteString(stderrAnsiReset)
		if showDeprecation {
			fmt.Fprintf(&result, " [%s%s%s]", stderrAnsiBlue, dep.ID, stderrAnsiReset)
		}
	} else {
		if dep != nil {
			result.WriteString("DEPRECATION ")
		}
		result.WriteString("WARNING")
		if showDeprecation {
			fmt.Fprintf(&result, " [%s]", dep.ID)
		}
	}

	opts := sasscommon.HighlightOptions{Color: l.color}
	if !l.Unicode {
		opts.Glyphs = termglyph.AsciiGlyphs
	}

	if span == nil {
		fmt.Fprintf(&result, ": %s\n", msg)
	} else if stack != "" {
		// If there's a span and a trace, the span's location information is
		// probably duplicated in the trace, so we just use it for highlighting.
		fmt.Fprintf(&result, ": %s\n\n", msg)
		highlight, err := (*span).Highlight(opts)
		if err != nil {
			_, _ = fmt.Fprintln(l.writer(), result.String())
			return
		}
		result.WriteString(highlight)
		result.WriteString("\n")
	} else {
		message, err := (*span).Message("\n"+msg, opts)
		if err != nil {
			_, _ = fmt.Fprintln(l.writer(), result.String())
			return
		}
		fmt.Fprintf(&result, " on %s\n", message)
	}

	if stack != "" {
		result.WriteString(indent(stack, 4))
		result.WriteString("\n")
	}

	_, _ = fmt.Fprintln(l.writer(), result.String())
}

// Debug emits a one-line debug message of the form "url:line DEBUG: msg"
// (bold "Debug" under color), with "-" for a missing source URL and line 1
// when there is no span. A span whose URL or start location cannot be read
// emits nothing.
// Matches Dart: StderrLogger.debug.
func (l *StderrLogger) Debug(msg string, span *sasscommon.FileSpan) {
	w := l.writer()
	if span == nil {
		if l.color {
			_, _ = fmt.Fprintf(w, "-:1 \x1b[1mDebug\x1b[0m: %s\n", msg)
		} else {
			_, _ = fmt.Fprintf(w, "-:1 DEBUG: %s\n", msg)
		}
		return
	}

	s := *span
	url := "-"
	su, err := s.SourceURL()
	if err != nil {
		return
	}
	if su != nil {
		url = sasscommon.PrettyUri(su)
	}

	startLoc, err := s.StartLocation()
	if err != nil {
		return
	}
	if l.color {
		_, _ = fmt.Fprintf(w, "%s:%d \x1b[1mDebug\x1b[0m: %s\n", url, startLoc.Line+1, msg)
	} else {
		_, _ = fmt.Fprintf(w, "%s:%d DEBUG: %s\n", url, startLoc.Line+1, msg)
	}
}

// writer returns the configured sink, or os.Stderr when none is set —
// the Go stand-in for Dart's printError call ending internalWarn/debug.
func (l *StderrLogger) writer() io.Writer {
	if l.Writer != nil {
		return l.Writer
	}
	return os.Stderr
}

// indent indents each non-empty line of s with n spaces for trace output,
// matching Dart's indent helper from utils.dart.
func indent(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		} else {
			lines[i] = prefix
		}
	}
	return strings.Join(lines, "\n")
}
