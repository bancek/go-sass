// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger.dart

import (
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// Logger is used by the evaluator to emit warnings, debug messages, and
// deprecation warnings. It may be implemented by host code.
//
// A nil span or trace means absent (Dart's nullable FileSpan/Trace
// parameters). The trace is carried structurally, not pre-formatted;
// implementations stringify it only when rendering.
//
// This surface folds three Dart pieces into one: the Logger interface, the
// internal LoggerWithDeprecationType base (whose Deprecation-typed
// internalWarn becomes the first-class WarnDeprecation here), and the
// WarnForDeprecation extension (whose future-deprecation gate and plain-logger
// fallback live in DeprecationProcessingLogger). The Compile category marker
// is dropped. Matches Dart's Logger interface in lib/src/logger.dart.
type Logger interface {
	Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace)
	Debug(message string, span *sasscommon.FileSpan)
	// WarnDeprecation emits a deprecation warning for deprecation. It returns
	// an error when DeprecationProcessingLogger has marked that deprecation
	// fatal, failing the compilation through the normal error path; trace is
	// the Sass stack trace at issue time.
	WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error
}
