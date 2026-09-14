package sasslogger

import (
	"testing"

	"github.com/bancek/go-sass/deprecation"
)

func TestQuietAllMethods(t *testing.T) {
	Quiet.Warn("test", nil, nil)
	Quiet.Debug("test", nil)
	if err := Quiet.WarnDeprecation("test", nil, deprecation.CallString, nil); err != nil {
		t.Errorf("Quiet.WarnDeprecation should return nil, got %v", err)
	}
}

func TestQuietWarnDeprecationNoError(t *testing.T) {
	if err := Quiet.WarnDeprecation("msg", nil, deprecation.SlashDiv, testTrace("trace")); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
