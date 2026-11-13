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

func TestForVersionWarnsBeyondCurrentVersion(t *testing.T) {
	var warnings []struct {
		deprecation *Deprecation
		message     string
	}
	_ = ForVersionInternal("999.0.0", func(d *Deprecation, message string) error {
		warnings = append(warnings, struct {
			deprecation *Deprecation
			message     string
		}{d, message})
		return nil
	})

	if len(warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(warnings))
	}
	if warnings[0].deprecation != DeprecationVersion {
		t.Errorf("deprecation = %s, want %s", warnings[0].deprecation, DeprecationVersion)
	}
	want := "Version \"999.0.0\" is beyond the current version of Dart Sass. " +
		"Selecting deprecations from later versions is deprecated and will be " +
		"an error in Dart Sass 3.0.\n\n" +
		"More info: http://sass-lang.com/d/deprecation-version"
	if warnings[0].message != want {
		t.Errorf("message = %q, want %q", warnings[0].message, want)
	}
}

func TestForVersionDoesNotWarnAtOrBelowCurrentVersion(t *testing.T) {
	_ = ForVersionInternal(CurrentVersion, func(d *Deprecation, message string) error {
		t.Errorf("unexpected warning %s: %s", d, message)
		return nil
	})
}

func TestDartSass2(t *testing.T) {
	ids := map[string]bool{}
	for _, d := range DartSass2() {
		ids[d.ID] = true
	}

	if !ids["new-global"] {
		t.Error("expected new-global in dart-sass-2")
	}
	for _, excluded := range []string{"color-module-compat", "import", "global-builtin", "if-function"} {
		if ids[excluded] {
			t.Errorf("didn't expect %s in dart-sass-2", excluded)
		}
	}
}
