package sasslogger

import (
	"testing"

	"github.com/bancek/go-sass/deprecation"
)

func TestTrackingLoggerInitialState(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	if tl.EmittedWarning() {
		t.Error("EmittedWarning should be false initially")
	}
	if tl.EmittedDebug() {
		t.Error("EmittedDebug should be false initially")
	}
}

func TestTrackingLoggerWarn(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	tl.Warn("test", nil, nil)
	if !tl.EmittedWarning() {
		t.Error("EmittedWarning should be true after Warn")
	}
	if tl.EmittedDebug() {
		t.Error("EmittedDebug should still be false")
	}
}

func TestTrackingLoggerDebug(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	tl.Debug("test", nil)
	if !tl.EmittedDebug() {
		t.Error("EmittedDebug should be true after Debug")
	}
	if tl.EmittedWarning() {
		t.Error("EmittedWarning should still be false")
	}
}

func TestTrackingLoggerWarnDeprecation(t *testing.T) {
	tl := NewTrackingLogger(Quiet)
	if err := tl.WarnDeprecation("dep", nil, deprecation.CallString, nil); err != nil {
		t.Errorf("WarnDeprecation should not error: %v", err)
	}
	if !tl.EmittedWarning() {
		t.Error("EmittedWarning should be true after WarnDeprecation")
	}
}
