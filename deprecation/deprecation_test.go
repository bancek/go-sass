package deprecation

import "testing"

func TestDeprecationConstants(t *testing.T) {
	for _, d := range allDeprecations {
		if d.ID == "" {
			t.Error("deprecation has empty ID")
		}
		if d == nil {
			t.Error("deprecation is nil")
		}
	}
}

func TestFromID(t *testing.T) {
	d := FromID("call-string")
	if d == nil || d.ID != "call-string" {
		t.Error("FromID('call-string') should return the call-string deprecation")
	}
	if d := FromID("nonexistent"); d != nil {
		t.Error("FromID('nonexistent') should return nil")
	}
}

func TestString(t *testing.T) {
	if CallString.String() != "call-string" {
		t.Errorf("String() = %q, want %q", CallString.String(), "call-string")
	}
}

func TestForVersion(t *testing.T) {
	v := ForVersion("2.0.0")
	found := false
	for _, d := range v {
		if d.ID == "call-string" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ForVersion('2.0.0') should include call-string")
	}

	v = ForVersion("99.0.0")
	for _, d := range v {
		if d.ObsoleteIn != "" {
			t.Errorf("ForVersion('99.0.0') included %s which has obsolete_in=%q", d.ID, d.ObsoleteIn)
		}
	}
}

func TestParseVersion(t *testing.T) {
	major, minor, patch := parseVersion("1.23.45")
	if major != 1 || minor != 23 || patch != 45 {
		t.Errorf("parseVersion(1.23.45) = (%d, %d, %d), want (1, 23, 45)", major, minor, patch)
	}
	major, minor, patch = parseVersion("2.0")
	if major != 2 || minor != 0 || patch != 0 {
		t.Errorf("parseVersion(2.0) = (%d, %d, %d), want (2, 0, 0)", major, minor, patch)
	}
	major, minor, patch = parseVersion("")
	if major != 0 || minor != 0 || patch != 0 {
		t.Errorf("parseVersion(\"\") = (%d, %d, %d), want (0, 0, 0)", major, minor, patch)
	}
}

func TestVersionLessThan(t *testing.T) {
	if !versionLessThan(1, 0, 0, 2, 0, 0) {
		t.Error("1.0.0 < 2.0.0")
	}
	if versionLessThan(2, 0, 0, 1, 0, 0) {
		t.Error("2.0.0 should not be < 1.0.0")
	}
	if !versionLessThan(1, 5, 0, 1, 10, 0) {
		t.Error("1.5.0 < 1.10.0")
	}
	if versionLessThan(1, 0, 0, 1, 0, 0) {
		t.Error("1.0.0 should not be < 1.0.0")
	}
}
