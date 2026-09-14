package extend

import (
	"testing"
)

func TestExtendModeStringNormal(t *testing.T) {
	if ExtendModeNormal.String() != "normal" {
		t.Errorf("ExtendModeNormal.String() = %q, want %q", ExtendModeNormal.String(), "normal")
	}
}

func TestExtendModeStringReplace(t *testing.T) {
	if ExtendModeReplace.String() != "replace" {
		t.Errorf("ExtendModeReplace.String() = %q, want %q", ExtendModeReplace.String(), "replace")
	}
}

func TestExtendModeStringAllTargets(t *testing.T) {
	if ExtendModeAllTargets.String() != "allTargets" {
		t.Errorf("ExtendModeAllTargets.String() = %q, want %q", ExtendModeAllTargets.String(), "allTargets")
	}
}
