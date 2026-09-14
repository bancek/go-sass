package sasslogger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

func TestDPLSilence(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet,
		[]*deprecation.Deprecation{deprecation.SlashDiv}, nil, nil, false)
	if err := dp.WarnDeprecation("test", nil, deprecation.SlashDiv, nil); err != nil {
		t.Errorf("silenced deprecation should not error: %v", err)
	}
	if err := dp.WarnDeprecation("test", nil, deprecation.CallString, nil); err != nil {
		t.Errorf("non-silenced deprecation should not error: %v", err)
	}
}

func TestDPLFatal(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet, nil,
		[]*deprecation.Deprecation{deprecation.CallString}, nil, false)
	if err := dp.WarnDeprecation("test", nil, deprecation.CallString, nil); err == nil {
		t.Error("fatal deprecation should return an error")
	}
}

func TestDPLFatalErrorMessage(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet, nil,
		[]*deprecation.Deprecation{deprecation.CallString}, nil, false)
	err := dp.WarnDeprecation("test", nil, deprecation.CallString, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "This is only an error because you've set the call-string deprecation to be fatal.") {
		t.Errorf("unexpected error message: %q", errStr)
	}
}

func TestDPLFatalWithSpan(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet, nil,
		[]*deprecation.Deprecation{deprecation.CallString}, nil, false)
	span := testFileSpan("body { }\n", 0, 8)
	err := dp.WarnDeprecation("test", span, deprecation.CallString, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := err.(*sasscommon.SassException); !ok {
		t.Errorf("expected *sasscommon.SassException, got %T: %v", err, err)
	}
}

func TestDPLFatalWithSpanAndStack(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet, nil,
		[]*deprecation.Deprecation{deprecation.CallString}, nil, false)
	span := testFileSpan("body { }\n", 0, 8)
	err := dp.WarnDeprecation("test", span, deprecation.CallString, testTrace("stack trace"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := err.(*sasscommon.SassRuntimeException); !ok {
		t.Errorf("expected *sasscommon.SassRuntimeException, got %T: %v", err, err)
	}
}

func TestDPLFuture(t *testing.T) {
	dp := NewDeprecationProcessingLogger(Quiet, nil, nil, nil, false)
	if err := dp.WarnDeprecation("test", nil, deprecation.CssFunctionMixin, nil); err != nil {
		t.Errorf("CssFunctionMixin is not a future deprecation: %v", err)
	}
	if err := dp.WarnDeprecation("test", nil, deprecation.TypeFunction, nil); err != nil {
		t.Errorf("TypeFunction is not a future deprecation: %v", err)
	}
}

func TestDPLLimitRepetition(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	dp := NewDeprecationProcessingLogger(tl, nil, nil, nil, true)
	for i := 0; i < 5; i++ {
		if err := dp.WarnDeprecation("test", nil, deprecation.CallString, nil); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	if err := dp.WarnDeprecation("test", nil, deprecation.CallString, nil); err != nil {
		t.Errorf("suppressed deprecation should not error: %v", err)
	}
}

func TestDPLPassThrough(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	dp := NewDeprecationProcessingLogger(tl, nil, nil, nil, false)
	dp.Warn("test", nil, nil)
	if !tl.EmittedWarning() {
		t.Error("Warn should pass through to inner logger")
	}
	dp.Debug("test", nil)
	if !tl.EmittedDebug() {
		t.Error("Debug should pass through to inner logger")
	}
}

func TestDPLSummarize(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	dp := NewDeprecationProcessingLogger(tl, nil, nil, nil, true)
	for i := 0; i < 7; i++ {
		_ = dp.WarnDeprecation("test", nil, deprecation.CallString, nil)
	}
	dp.Summarize(false)
	if !tl.EmittedWarning() {
		t.Error("Summarize should emit warning about omitted messages")
	}
}

func TestDPLSummarizeExact(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner, nil, nil, nil, true)
	for i := 0; i < 7; i++ {
		_ = dp.WarnDeprecation("test", nil, deprecation.CallString, nil)
	}
	dp.Summarize(false)
	out := buf.String()
	if !strings.Contains(out, "2 repetitive deprecation warnings omitted.") {
		t.Errorf("expected '2 repetitive deprecation warnings omitted.', got %q", out)
	}
	if !strings.Contains(out, "Run in verbose mode to see all warnings.") {
		t.Errorf("expected 'Run in verbose mode to see all warnings.', got %q", out)
	}
}

func TestDPLSummarizeJS(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner, nil, nil, nil, true)
	for i := 0; i < 7; i++ {
		_ = dp.WarnDeprecation("test", nil, deprecation.CallString, nil)
	}
	dp.Summarize(true)
	out := buf.String()
	if !strings.Contains(out, "2 repetitive deprecation warnings omitted.") {
		t.Errorf("expected '2 repetitive deprecation warnings omitted.', got %q", out)
	}
	if strings.Contains(out, "Run in verbose mode") {
		t.Errorf("should NOT contain 'Run in verbose mode' in JS mode, got %q", out)
	}
}

func TestDPLValidateFatalObsolete(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner, nil,
		[]*deprecation.Deprecation{deprecation.CssFunctionMixin}, nil, false)
	dp.Validate()
	out := buf.String()
	if !strings.Contains(out, "css-function-mixin deprecation is obsolete, so does not need to be made fatal.") {
		t.Errorf("expected obsolete fatal warning, got %q", out)
	}
}

func TestDPLValidateFatalAndSilenced(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner,
		[]*deprecation.Deprecation{deprecation.CallString},
		[]*deprecation.Deprecation{deprecation.CallString}, nil, false)
	dp.Validate()
	out := buf.String()
	if !strings.Contains(out, "Ignoring setting to silence call-string deprecation, since it has also been made fatal.") {
		t.Errorf("expected fatal+silence conflict warning, got %q", out)
	}
}

func TestDPLValidateSilenceUserAuthored(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner,
		[]*deprecation.Deprecation{deprecation.UserAuthored}, nil, nil, false)
	dp.Validate()
	out := buf.String()
	if !strings.Contains(out, "User-authored deprecations should not be silenced.") {
		t.Errorf("expected user-authored silence warning, got %q", out)
	}
}

func TestDPLValidateSilenceObsolete(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner,
		[]*deprecation.Deprecation{deprecation.CssFunctionMixin}, nil, nil, false)
	dp.Validate()
	out := buf.String()
	if !strings.Contains(out, "css-function-mixin deprecation is obsolete. If you were previously silencing it, your code may now behave in unexpected ways.") {
		t.Errorf("expected silence+obsolete warning, got %q", out)
	}
}

func TestDPLValidateFutureNotFuture(t *testing.T) {
	var buf bytes.Buffer
	inner := NewStderrLoggerWithWriter(false, false, &buf)
	dp := NewDeprecationProcessingLogger(inner, nil, nil,
		[]*deprecation.Deprecation{deprecation.CallString}, false)
	dp.Validate()
	out := buf.String()
	if !strings.Contains(out, "call-string is not a future deprecation, so it does not need to be explicitly enabled.") {
		t.Errorf("expected not-future warning, got %q", out)
	}
}

func TestDPLValidate(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	dp := NewDeprecationProcessingLogger(tl,
		[]*deprecation.Deprecation{deprecation.CallString},
		[]*deprecation.Deprecation{deprecation.CallString},
		[]*deprecation.Deprecation{deprecation.CallString},
		false,
	)
	dp.Validate()
	if !tl.EmittedWarning() {
		t.Error("Validate should emit warnings for conflicts")
	}
}

func testTrace(member string) *sasscommon.Trace {
	return sasscommon.NewTrace([]sasscommon.Frame{{Member: member}})
}
