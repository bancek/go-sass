// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// Package evalcontext provides types used during Sass evaluation that must be
// accessible from both the eval and value packages.
package evalcontext

// dart-source: lib/src/evaluation_context.dart (EvaluationContext interface,
// warn/warnForDeprecation helpers, withEvaluationContext) and
// lib/src/visitor/evaluate.dart (_EvaluationContext subclass)

import (
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sasslogger"
)

// EvaluationContext exposes information about the current Sass evaluation.
//
// Built-in callables receive it explicitly instead of reading ambient zone
// state as Dart does, which keeps concurrent compilations isolated. The
// logger serves plain warnings; deprecation and user-authored warnings are
// routed through the visitor hook when installed so they gain dedup,
// quiet-deps filtering, and stack traces.
type EvaluationContext struct {
	// Logger handles plain warn/debug output for this evaluation.
	Logger       sasslogger.Logger
	callableNode sasscommon.AstNode

	// importSpan is the span of the @import currently being evaluated, used
	// to attribute importer warnings.
	importSpan sasscommon.FileSpan

	// defaultWarnSpan is the fallback warning location, typically the root
	// stylesheet span, used when neither the import span nor the callable
	// node yields a span.
	defaultWarnSpan sasscommon.FileSpan

	// warnFn intercepts warning calls with the visitor's warn method. When
	// set, WarnDeprecation flows through dedup, quiet-deps, and trace
	// handling; when nil, warnings fall back to the logger directly.
	warnFn func(message string, span sasscommon.FileSpan, deprecation *deprecation.Deprecation) error
}

// CurrentCallableSpan returns the span of the innermost callable being
// invoked. It reports nil when no callable is active; Dart throws a
// StateError there, which Go callers handle as an absent span instead.
func (ec *EvaluationContext) CurrentCallableSpan() (sasscommon.FileSpan, error) {
	if ec.callableNode != nil {
		return ec.callableNode.Span()
	}
	return nil, nil
}

// SetCallableNode records the innermost callable node for warning spans.
// The node (rather than a resolved span) is stored so spans that take real
// work to manufacture are only computed when a warning actually needs one.
func (ec *EvaluationContext) SetCallableNode(node sasscommon.AstNode) {
	ec.callableNode = node
}

// CallableNode returns the innermost callable node, or nil outside a call.
func (ec *EvaluationContext) CallableNode() sasscommon.AstNode {
	return ec.callableNode
}

// SetImportSpan records the span of the @import currently being resolved.
func (ec *EvaluationContext) SetImportSpan(span sasscommon.FileSpan) {
	ec.importSpan = span
}

// ImportSpan returns the span of the @import currently being resolved.
func (ec *EvaluationContext) ImportSpan() sasscommon.FileSpan {
	return ec.importSpan
}

// SetDefaultWarnSpan installs the fallback warning span used when neither
// the import span nor the callable node applies.
func (ec *EvaluationContext) SetDefaultWarnSpan(span sasscommon.FileSpan) {
	ec.defaultWarnSpan = span
}

// DefaultWarnSpan returns the fallback warning span.
func (ec *EvaluationContext) DefaultWarnSpan() sasscommon.FileSpan {
	return ec.defaultWarnSpan
}

// SetWarnFn installs the visitor's warn method as the warning interceptor.
func (ec *EvaluationContext) SetWarnFn(fn func(message string, span sasscommon.FileSpan, deprecation *deprecation.Deprecation) error) {
	ec.warnFn = fn
}

// Warn emits a plain warning through the underlying logger.
func (ec *EvaluationContext) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	ec.Logger.Warn(message, span, trace)
}

// Debug emits a debug message through the underlying logger.
func (ec *EvaluationContext) Debug(message string, span *sasscommon.FileSpan) {
	ec.Logger.Debug(message, span)
}

// WarnDeprecation emits a warning tied to the current @import or call. The
// span follows the import span, then the callable span, then the default
// warn span. When the visitor hook is installed the warning flows through
// it for dedup, quiet-deps, and trace handling; otherwise it goes directly
// to the logger. A non-nil deprecation marks it as that deprecation kind.
func (ec *EvaluationContext) WarnDeprecation(message string, deprecation *deprecation.Deprecation) error {
	span, err := ec.warnSpan()
	if err != nil {
		return err
	}
	if ec.warnFn != nil {
		return ec.warnFn(message, span, deprecation)
	}
	return ec.Logger.WarnDeprecation(message, &span, deprecation, nil)
}

// WarnDeprecationFromApi emits a deprecation warning from host code running
// outside evaluation. When an evaluation is active it behaves like
// WarnDeprecation; otherwise Dart falls back to the default logger writing
// to standard error. The Go port routes through the same warn-span
// resolution, so callers should prefer a context-tied path when available.
func (ec *EvaluationContext) WarnDeprecationFromApi(message string, deprecation *deprecation.Deprecation) error {
	return ec.WarnDeprecation(message, deprecation)
}

// WarnWithDeprecation emits message as a plain warning, or as a
// user-authored deprecation warning when isDeprecation is true. This backs
// the host-facing warn() helper that runs inside custom function or
// importer callbacks.
func (ec *EvaluationContext) WarnWithDeprecation(message string, isDeprecation bool) error {
	span, err := ec.warnSpan()
	if err != nil {
		return err
	}
	var dep *deprecation.Deprecation
	if isDeprecation {
		dep = deprecation.UserAuthored
	}
	if ec.warnFn != nil {
		return ec.warnFn(message, span, dep)
	} else if isDeprecation {
		return ec.Logger.WarnDeprecation(message, &span, deprecation.UserAuthored, nil)
	} else {
		ec.Logger.Warn(message, &span, nil)
		return nil
	}
}

// warnSpan resolves the warning span in priority order: the current import
// span when it carries text, else the callable node's span, else the
// default warn span. The import-span text check avoids attributing warnings
// to an empty import location.
func (ec *EvaluationContext) warnSpan() (sasscommon.FileSpan, error) {
	if ec.importSpan != nil {
		importSpanText, err := ec.importSpan.SpanText()
		if err != nil {
			return nil, err
		}
		if importSpanText != "" {
			return ec.importSpan, nil
		}
	}
	if ec.callableNode != nil {
		return ec.callableNode.Span()
	}
	return ec.defaultWarnSpan, nil
}
