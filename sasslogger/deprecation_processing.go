// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger/deprecation_processing.dart

import (
	"fmt"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// The maximum number of repetitions of the same warning
// DeprecationProcessingLogger will emit before hiding the rest. The
// remainder is tallied and reported once by Summarize.
const maxRepetitions = 5

// A logger that wraps an inner logger to have special handling for
// deprecation warnings, silencing, making fatal, enabling future, and/or
// limiting repetition based on its inputs. The compile pipeline always wraps
// the user logger in one of these: Validate runs before evaluation and
// Summarize runs after, even on failure.
//
// Matches Dart: DeprecationProcessingLogger in
// lib/src/logger/deprecation_processing.dart.
type DeprecationProcessingLogger struct {
	// warningCounts tallies how many times each deprecation has been seen,
	// so repetition limiting and the Summarize total have a basis.
	warningCounts map[*deprecation.Deprecation]int

	// inner receives every warning that survives deprecation handling.
	inner Logger
	// silenceDeprecations drops matching warnings entirely (unless also fatal).
	silenceDeprecations map[*deprecation.Deprecation]struct{}
	// fatalDeprecations turns matching warnings into compilation errors.
	fatalDeprecations map[*deprecation.Deprecation]struct{}
	// futureDeprecations gates not-yet-active deprecations the user opted into.
	futureDeprecations map[*deprecation.Deprecation]struct{}
	// limitRepetition caps repeats of one warning at maxRepetitions.
	limitRepetition bool
}

// NewDeprecationProcessingLogger creates a new DeprecationProcessingLogger
// that wraps inner.
//
// silenceDeprecations: deprecation warnings of these types will be ignored.
// fatalDeprecations: deprecation warnings of these types will cause an error.
//
//	Future deprecations in this list will still cause an error even if they
//	are not also in futureDeprecations.
//
// futureDeprecations: future deprecations that the user has explicitly opted into.
// limitRepetition: whether repetitions of the same warning should be limited
//
//	to no more than maxRepetitions.
//
// Matches Dart: DeprecationProcessingLogger constructor.
func NewDeprecationProcessingLogger(
	inner Logger,
	silenceDeprecations []*deprecation.Deprecation,
	fatalDeprecations []*deprecation.Deprecation,
	futureDeprecations []*deprecation.Deprecation,
	limitRepetition bool,
) *DeprecationProcessingLogger {
	silence := make(map[*deprecation.Deprecation]struct{}, len(silenceDeprecations))
	for _, d := range silenceDeprecations {
		silence[d] = struct{}{}
	}
	fatal := make(map[*deprecation.Deprecation]struct{}, len(fatalDeprecations))
	for _, d := range fatalDeprecations {
		fatal[d] = struct{}{}
	}
	future := make(map[*deprecation.Deprecation]struct{}, len(futureDeprecations))
	for _, d := range futureDeprecations {
		future[d] = struct{}{}
	}
	return &DeprecationProcessingLogger{
		warningCounts:       make(map[*deprecation.Deprecation]int),
		inner:               inner,
		silenceDeprecations: silence,
		fatalDeprecations:   fatal,
		futureDeprecations:  future,
		limitRepetition:     limitRepetition,
	}
}

// Warn emits a non-deprecation warning straight through to the inner
// logger, bypassing deprecation handling entirely.
// Matches Dart: internalWarn with a null deprecation forwarding to _inner.
func (d *DeprecationProcessingLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	d.inner.Warn(message, span, trace)
}

// Debug emits a debug message straight through to the inner logger;
// deprecation handling never applies to debug output.
// Matches Dart: debug forwarding to _inner.
func (d *DeprecationProcessingLogger) Debug(message string, span *sasscommon.FileSpan) {
	d.inner.Debug(message, span)
}

// WarnDeprecation emits a deprecation warning through the future, fatal,
// silence, and repetition gates. It returns an error only when the
// deprecation is marked fatal.
// Matches Dart: internalWarn dispatching to _handleDeprecation.
func (d *DeprecationProcessingLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error {
	return d.handleDeprecation(deprecation, message, span, trace)
}

// handleDeprecation processes a deprecation warning.
//
// A future deprecation the user has not opted into is dropped silently. A
// fatal deprecation becomes an error noting the fatal setting and shaped by
// what location is known: span plus trace yields a runtime exception, span
// alone a span exception, neither a plain script exception. A silenced
// deprecation is dropped next, and when repetition limiting is on only the
// first maxRepetitions sightings of one deprecation pass.
//
// Survivors reach the inner logger as deprecation warnings.
//
// Matches Dart: _handleDeprecation
func (d *DeprecationProcessingLogger) handleDeprecation(
	deprecation *deprecation.Deprecation,
	message string,
	span *sasscommon.FileSpan,
	trace *sasscommon.Trace,
) error {
	if deprecation.IsFuture {
		if _, ok := d.futureDeprecations[deprecation]; !ok {
			return nil
		}
	}

	if _, ok := d.fatalDeprecations[deprecation]; ok {
		message += "\n\nThis is only an error because you've set the " +
			deprecation.ID + " deprecation to be fatal.\n" +
			"Remove this setting if you need to keep using this feature."
		if span != nil {
			if trace != nil {
				return &sasscommon.SassRuntimeException{
					Message: message,
					Span:    *span,
					Trace:   trace,
				}
			}
			return &sasscommon.SassException{
				Message: message,
				Span:    *span,
			}
		}
		return &sasscommon.SassScriptException{Message: message}
	}

	if _, ok := d.silenceDeprecations[deprecation]; ok {
		return nil
	}

	if d.limitRepetition {
		count := d.warningCounts[deprecation]
		count++
		d.warningCounts[deprecation] = count
		if count > maxRepetitions {
			return nil
		}
	}

	return d.inner.WarnDeprecation(message, span, deprecation, trace)
}

// Validate warns if any of the deprecation options are incompatible or
// unnecessary: a future deprecation made fatal without being enabled, an
// obsolete deprecation made fatal or silenced, a deprecation both silenced
// and fatal (fatal wins), a silenced user-authored deprecation, conflicting
// silence-plus-enable of one future deprecation, a silenced-but-inactive
// future deprecation, or an explicitly enabled non-future deprecation. It
// runs before evaluation so misconfiguration surfaces even when the compile
// itself would stay quiet.
// Matches Dart: validate
func (d *DeprecationProcessingLogger) Validate() {
	for dep := range d.fatalDeprecations {
		_, inFuture := d.futureDeprecations[dep]
		_, inSilence := d.silenceDeprecations[dep]
		switch {
		case dep.IsFuture && !inFuture:
			d.Warn(fmt.Sprintf(
				"Future %s deprecation must be enabled before it can be made fatal.",
				dep.ID), nil, nil)
		case dep.ObsoleteIn != "":
			d.Warn(fmt.Sprintf(
				"%s deprecation is obsolete, so does not need to be made fatal.",
				dep.ID), nil, nil)
		case inSilence:
			d.Warn(fmt.Sprintf(
				"Ignoring setting to silence %s deprecation, since it has also been made fatal.",
				dep.ID), nil, nil)
		}
	}

	for dep := range d.silenceDeprecations {
		_, inFuture := d.futureDeprecations[dep]
		switch {
		case dep == deprecation.UserAuthored:
			d.Warn("User-authored deprecations should not be silenced.", nil, nil)
		case dep.ObsoleteIn != "":
			d.Warn(fmt.Sprintf(
				"%s deprecation is obsolete. If you were previously silencing it, "+
					"your code may now behave in unexpected ways.",
				dep.ID), nil, nil)
		case dep.IsFuture && inFuture:
			d.Warn(fmt.Sprintf(
				"Conflicting options for future %s deprecation cancel each other out.",
				dep.ID), nil, nil)
		case dep.IsFuture:
			d.Warn(fmt.Sprintf(
				"Future %s deprecation is not yet active, so silencing it is unnecessary.",
				dep.ID), nil, nil)
		}
	}

	for dep := range d.futureDeprecations {
		if !dep.IsFuture {
			d.Warn(fmt.Sprintf(
				"%s is not a future deprecation, so it does not need to be explicitly enabled.",
				dep.ID), nil, nil)
		}
	}
}

// Summarize prints a warning indicating the number of deprecation warnings
// that were omitted due to repetition, tallying only sightings past
// maxRepetitions.
//
// The js flag indicates whether this is running in JS mode, in which case
// it doesn't mention "verbose mode" because the JS API doesn't support that.
// Go keeps the flag for behavioral parity even though this host never sets it.
// Matches Dart: summarize
func (d *DeprecationProcessingLogger) Summarize(js bool) {
	total := 0
	for _, count := range d.warningCounts {
		if count > maxRepetitions {
			total += count - maxRepetitions
		}
	}
	if total > 0 {
		msg := fmt.Sprintf("%d repetitive deprecation warnings omitted.", total)
		if !js {
			msg += "\nRun in verbose mode to see all warnings."
		}
		d.inner.Warn(msg, nil, nil)
	}
}
