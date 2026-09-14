// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package evalcontext

import (
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// testLogger records the last call to each method for assertion.
type testLogger struct {
	lastWarnMsg   string
	lastWarnSpan  *sasscommon.FileSpan
	lastWarnStack *sasscommon.Trace

	lastDebugMsg  string
	lastDebugSpan *sasscommon.FileSpan

	lastWarnDepMsg   string
	lastWarnDepSpan  *sasscommon.FileSpan
	lastWarnDepDep   *deprecation.Deprecation
	lastWarnDepStack *sasscommon.Trace
	lastWarnDepErr   error
}

func (l *testLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	l.lastWarnMsg = message
	l.lastWarnSpan = span
	l.lastWarnStack = trace
}

func (l *testLogger) Debug(message string, span *sasscommon.FileSpan) {
	l.lastDebugMsg = message
	l.lastDebugSpan = span
}

func (l *testLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error {
	l.lastWarnDepMsg = message
	l.lastWarnDepSpan = span
	l.lastWarnDepDep = deprecation
	l.lastWarnDepStack = trace
	return l.lastWarnDepErr
}

func newTestFS() *sasscommon.FileSource {
	return sasscommon.NewFileSource([]byte("test source\n"), nil)
}

func newTestSpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(newTestFS(), 0, 5)
}

func newTestSpanWithText(text string) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(text), nil)
	return sasscommon.NewFileSpan(fs, 0, len(text))
}

func newEmptySpan() sasscommon.FileSpan {
	return sasscommon.NewFileSpan(nil, 0, 0)
}

// --- Construction ---

func TestNewEvalContextZeroState(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}

	if ec.CallableNode() != nil {
		t.Error("CallableNode should be nil initially")
	}
	if ec.ImportSpan() != nil {
		t.Error("ImportSpan should be nil initially")
	}
}

// --- CallableNode ---

func TestSetCallableNode(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpan()
	node := sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return span, nil
	})

	ec.SetCallableNode(node)

	if ec.CallableNode() != node {
		t.Error("CallableNode should return the set node")
	}
}

func TestSetCallableNodeNil(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}

	ec.SetCallableNode(nil)

	if ec.CallableNode() != nil {
		t.Error("CallableNode should be nil")
	}
}

func TestCurrentCallableSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpan()
	node := sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return span, nil
	})
	ec.SetCallableNode(node)

	got, err := ec.CurrentCallableSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != span {
		t.Error("CurrentCallableSpan should return the node's span")
	}
}

func TestCurrentCallableSpanNil(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}

	got, err := ec.CurrentCallableSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("CurrentCallableSpan should return nil when no callable")
	}
}

// --- ImportSpan ---

func TestSetImportSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpan()

	ec.SetImportSpan(span)

	if ec.ImportSpan() != span {
		t.Error("ImportSpan should return the set span")
	}
}

func TestImportSpanNilByDefault(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}

	if ec.ImportSpan() != nil {
		t.Error("ImportSpan should be nil by default")
	}
}

// --- DefaultWarnSpan ---

func TestSetDefaultWarnSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpan()

	ec.SetDefaultWarnSpan(span)

	if ec.DefaultWarnSpan() != span {
		t.Error("DefaultWarnSpan should return the set span")
	}
}

// --- warnSpan resolution ---

func TestWarnSpanImportSpanWithText(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	importSpan := newTestSpanWithText("some import text")
	ec.SetImportSpan(importSpan)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != importSpan {
		t.Error("warnSpan should return importSpan when it has text")
	}
}

func TestWarnSpanImportSpanEmptyText(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	emptySpan := newEmptySpan()
	ec.SetImportSpan(emptySpan)
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultSpan {
		t.Error("warnSpan should fall through to defaultWarnSpan when importSpan has empty text")
	}
}

func TestWarnSpanImportSpanEmptyTextWithCallable(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	emptySpan := newEmptySpan()
	ec.SetImportSpan(emptySpan)
	callableSpan := newTestSpanWithText("callable text")
	node := sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return callableSpan, nil
	})
	ec.SetCallableNode(node)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != callableSpan {
		t.Error("warnSpan should return callableNode.span when importSpan has empty text")
	}
}

func TestWarnSpanCallableNode(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	callableSpan := newTestSpanWithText("callable text")
	node := sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return callableSpan, nil
	})
	ec.SetCallableNode(node)
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != callableSpan {
		t.Error("warnSpan should return callableNode.span when no importSpan")
	}
}

func TestWarnSpanDefault(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultSpan {
		t.Error("warnSpan should return defaultWarnSpan when nothing else is set")
	}
}

func TestWarnSpanImportWinsOverCallable(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	importSpan := newTestSpanWithText("import text")
	ec.SetImportSpan(importSpan)
	callableSpan := newTestSpanWithText("callable text")
	node := sasscommon.NewFakeAstNode(func() (sasscommon.FileSpan, error) {
		return callableSpan, nil
	})
	ec.SetCallableNode(node)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != importSpan {
		t.Error("warnSpan should prefer importSpan over callableSpan")
	}
}

func TestWarnSpanBothNil(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	got, err := ec.warnSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultSpan {
		t.Error("warnSpan should return defaultWarnSpan when both import and callable are nil")
	}
}

// --- Warn ---

func TestWarn(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpanWithText("test source")
	ec.Warn("test message", &span, testTrace("stack trace"))

	if logger.lastWarnMsg != "test message" {
		t.Errorf("Warn msg = %q, want %q", logger.lastWarnMsg, "test message")
	}
	if logger.lastWarnSpan == nil {
		t.Error("Warn span should not be nil")
	}
	if logger.lastWarnStack == nil || logger.lastWarnStack.String() != testTrace("stack trace").String() {
		t.Errorf("Warn stack = %q, want %q", logger.lastWarnStack, "stack trace")
	}
}

func TestWarnWithNilSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	ec.Warn("test message", nil, nil)

	if logger.lastWarnMsg != "test message" {
		t.Errorf("Warn msg = %q, want %q", logger.lastWarnMsg, "test message")
	}
	if logger.lastWarnSpan != nil {
		t.Error("Warn span should be nil")
	}
}

// --- Debug ---

func TestDebug(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	span := newTestSpanWithText("test source")
	ec.Debug("debug msg", &span)

	if logger.lastDebugMsg != "debug msg" {
		t.Errorf("Debug msg = %q, want %q", logger.lastDebugMsg, "debug msg")
	}
	if logger.lastDebugSpan == nil {
		t.Error("Debug span should not be nil")
	}
}

func TestDebugWithNilSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	ec.Debug("debug msg", nil)

	if logger.lastDebugMsg != "debug msg" {
		t.Errorf("Debug msg = %q, want %q", logger.lastDebugMsg, "debug msg")
	}
	if logger.lastDebugSpan != nil {
		t.Error("Debug span should be nil")
	}
}

// --- WarnDeprecation ---

func TestWarnDeprecationLogger(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}
	if logger.lastWarnDepMsg != "deprecated" {
		t.Errorf("WarnDeprecation msg = %q, want %q", logger.lastWarnDepMsg, "deprecated")
	}
	if logger.lastWarnDepDep != deprecation.CallString {
		t.Errorf("WarnDeprecation dep = %v, want %v", logger.lastWarnDepDep, deprecation.CallString)
	}
	if logger.lastWarnDepStack != nil {
		t.Errorf("WarnDeprecation stack = %q, want nil", logger.lastWarnDepStack)
	}
}

func TestWarnDeprecationWarnFn(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	var capturedMsg string
	var capturedDep *deprecation.Deprecation
	ec.SetWarnFn(func(message string, span sasscommon.FileSpan, dep *deprecation.Deprecation) error {
		capturedMsg = message
		capturedDep = dep
		return nil
	})

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}
	if capturedMsg != "deprecated" {
		t.Errorf("warnFn msg = %q, want %q", capturedMsg, "deprecated")
	}
	if capturedDep != deprecation.CallString {
		t.Errorf("warnFn dep = %v, want %v", capturedDep, deprecation.CallString)
	}
	if logger.lastWarnDepMsg != "" {
		t.Error("logger.WarnDeprecation should NOT have been called")
	}
}

func TestWarnDeprecationWarnFnError(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	ec.SetWarnFn(func(message string, span sasscommon.FileSpan, dep *deprecation.Deprecation) error {
		return sasscommon.NewSassScriptException("warnFn error", nil)
	})

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err == nil {
		t.Fatal("expected error from warnFn")
	}
	want := "warnFn error"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestWarnDeprecationLoggerError(t *testing.T) {
	logger := &testLogger{}
	logger.lastWarnDepErr = sasscommon.NewSassScriptException("logger error", nil)
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err == nil {
		t.Fatal("expected error from logger")
	}
	want := "logger error"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestWarnDeprecationUsesWarnSpan(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	importSpan := newTestSpanWithText("import text")
	ec.SetImportSpan(importSpan)
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	var capturedSpanText string
	ec.SetWarnFn(func(message string, span sasscommon.FileSpan, dep *deprecation.Deprecation) error {
		text, _ := span.SpanText()
		capturedSpanText = text
		return nil
	})

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}
	if capturedSpanText != "import text" {
		t.Errorf("warnFn span text = %q, want %q", capturedSpanText, "import text")
	}
}

// --- WarnDeprecationFromApi ---

func TestWarnDeprecationFromApiDelegates(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnDeprecationFromApi("api dep", deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}
	if logger.lastWarnDepMsg != "api dep" {
		t.Errorf("WarnDeprecationFromApi msg = %q, want %q", logger.lastWarnDepMsg, "api dep")
	}
}

// --- WarnWithDeprecation ---

func TestWarnWithDeprecationTrueWarnFn(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	var capturedDep *deprecation.Deprecation
	ec.SetWarnFn(func(message string, span sasscommon.FileSpan, dep *deprecation.Deprecation) error {
		capturedDep = dep
		return nil
	})

	err := ec.WarnWithDeprecation("test msg", true)
	if err != nil {
		t.Fatal(err)
	}
	if capturedDep != deprecation.UserAuthored {
		t.Errorf("warnFn dep = %v, want %v", capturedDep, deprecation.UserAuthored)
	}
	if logger.lastWarnDepMsg != "" {
		t.Error("logger.WarnDeprecation should NOT have been called")
	}
}

func TestWarnWithDeprecationTrueLogger(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnWithDeprecation("test msg", true)
	if err != nil {
		t.Fatal(err)
	}
	if logger.lastWarnDepMsg != "test msg" {
		t.Errorf("WarnDeprecation msg = %q, want %q", logger.lastWarnDepMsg, "test msg")
	}
	if logger.lastWarnDepDep != deprecation.UserAuthored {
		t.Errorf("WarnDeprecation dep = %v, want %v", logger.lastWarnDepDep, deprecation.UserAuthored)
	}
}

func TestWarnWithDeprecationFalseWarnFn(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	var capturedDep *deprecation.Deprecation
	ec.SetWarnFn(func(message string, span sasscommon.FileSpan, dep *deprecation.Deprecation) error {
		capturedDep = dep
		return nil
	})

	err := ec.WarnWithDeprecation("test msg", false)
	if err != nil {
		t.Fatal(err)
	}
	if capturedDep != nil {
		t.Errorf("warnFn dep should be nil, got %v", capturedDep)
	}
	if logger.lastWarnMsg != "" {
		t.Error("logger.Warn should NOT have been called (warnFn was used)")
	}
}

func TestWarnWithDeprecationFalseLogger(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnWithDeprecation("test msg", false)
	if err != nil {
		t.Fatal(err)
	}
	if logger.lastWarnMsg != "test msg" {
		t.Errorf("Warn msg = %q, want %q", logger.lastWarnMsg, "test msg")
	}
	if logger.lastWarnStack != nil {
		t.Errorf("Warn stack = %q, want nil", logger.lastWarnStack)
	}
	if logger.lastWarnDepMsg != "" {
		t.Error("WarnDeprecation should NOT have been called")
	}
}

func TestWarnWithDeprecationFalseLoggerError(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnWithDeprecation("test msg", false)
	if err != nil {
		t.Fatal(err)
	}
}

// --- Error propagation ---

func TestWarnDeprecationErrorExactMessage(t *testing.T) {
	logger := &testLogger{}
	logger.lastWarnDepErr = sasscommon.NewSassScriptException("exact error message", nil)
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "exact error message"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestWarnDeprecationMultiSpanError(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	secondary := map[sasscommon.FileSpan]string{}
	secondarySpan := newTestSpanWithText("secondary")
	secondary[secondarySpan] = "secondary label"
	logger.lastWarnDepErr = &sasscommon.MultiSpanSassScriptException{
		Message:        "conflict error",
		PrimaryLabel:   "primary label",
		SecondarySpans: secondary,
	}

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "conflict error"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestWarnDeprecationSassScriptExceptionWithArgName(t *testing.T) {
	logger := &testLogger{}
	argName := "foo"
	logger.lastWarnDepErr = sasscommon.NewSassScriptException("bad argument", &argName)
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	err := ec.WarnDeprecation("deprecated", deprecation.CallString)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "$foo: bad argument"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// --- SetWarnFn nil ---

func TestSetWarnFnNil(t *testing.T) {
	logger := &testLogger{}
	ec := &EvaluationContext{Logger: logger}
	defaultSpan := newTestSpanWithText("default")
	ec.SetDefaultWarnSpan(defaultSpan)

	ec.SetWarnFn(nil)

	err := ec.WarnDeprecation("after nil", deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}
	if logger.lastWarnDepMsg != "after nil" {
		t.Errorf("msg = %q, want %q", logger.lastWarnDepMsg, "after nil")
	}
}

func testTrace(member string) *sasscommon.Trace {
	return sasscommon.NewTrace([]sasscommon.Frame{{Member: member}})
}
