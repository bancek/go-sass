// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger.dart (_QuietLogger section)

import (
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// quietLogger silently ignores all messages. Every method is an empty body
// in Dart; Go mirrors that with no-ops.
//
// Matches Dart: _QuietLogger
type quietLogger struct{}

// Quiet is a logger that silently ignores all messages. Used for --quiet
// runs and anywhere output must be swallowed; deprecation warnings report
// success rather than going fatal.
//
// Matches Dart: Logger.quiet
var Quiet = &quietLogger{}

// Warn discards the warning. Matches Dart's _QuietLogger.warn (empty body).
func (q *quietLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {}

// Debug discards the debug message. Matches Dart's _QuietLogger.debug.
func (q *quietLogger) Debug(message string, span *sasscommon.FileSpan) {}

// WarnDeprecation discards the deprecation warning and always reports
// success, so quiet mode never fails a compilation on a fatal deprecation.
func (q *quietLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error {
	return nil
}
