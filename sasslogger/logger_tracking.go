// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger/tracking.dart

import (
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// TrackingLogger wraps a logger and keeps track of when it is used, so
// callers can detect whether any warning or debug output was emitted (for
// example when deciding exit codes). Like Dart, a deprecation warning
// counts as a warning, not as debug output.
//
// Matches Dart: TrackingLogger
type TrackingLogger struct {
	inner Logger
	// emittedWarning latches on the first Warn or WarnDeprecation call.
	emittedWarning bool
	// emittedDebug latches on the first Debug call.
	emittedDebug bool
}

// NewTrackingLogger creates a TrackingLogger that wraps inner. Both emitted
// flags start false.
//
// Matches Dart: TrackingLogger constructor
func NewTrackingLogger(inner Logger) *TrackingLogger {
	return &TrackingLogger{inner: inner}
}

// EmittedWarning reports whether Warn or WarnDeprecation has been called on
// this logger.
//
// Matches Dart: TrackingLogger.emittedWarning
func (t *TrackingLogger) EmittedWarning() bool { return t.emittedWarning }

// EmittedDebug reports whether Debug has been called on this logger.
//
// Matches Dart: TrackingLogger.emittedDebug
func (t *TrackingLogger) EmittedDebug() bool { return t.emittedDebug }

// Warn records the warning flag, then forwards to the wrapped logger.
//
// Matches Dart: TrackingLogger.warn
func (t *TrackingLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	t.emittedWarning = true
	t.inner.Warn(message, span, trace)
}

// Debug records the debug flag, then forwards to the wrapped logger.
//
// Matches Dart: TrackingLogger.debug
func (t *TrackingLogger) Debug(message string, span *sasscommon.FileSpan) {
	t.emittedDebug = true
	t.inner.Debug(message, span)
}

// WarnDeprecation records the warning flag, then forwards to the wrapped
// logger. Dart's TrackingLogger only overrides warn/debug (it implements
// the plain Logger, so deprecations arrive as warn calls); Go overrides the
// first-class deprecation entry point too, keeping the same "counts as a
// warning" semantics.
//
// Matches Dart: TrackingLogger.warn (deprecation path via warn)
func (t *TrackingLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error {
	t.emittedWarning = true
	return t.inner.WarnDeprecation(message, span, deprecation, trace)
}
