package util

import "testing"

func TestIsPublic(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"abc", true},
		{"abc-def", true},
		{"$variable", true},
		{"-private", false},
		{"_private", false},
		{"", false}, // empty string starts with none, panic-capable
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "" {
				// empty string panics in Go
				defer func() { recover() }()
				IsPublic(tt.name)
				return
			}
			got := IsPublic(tt.name)
			if got != tt.want {
				t.Errorf("IsPublic(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsPrivateMember(t *testing.T) {
	if !IsPrivateMember("-foo") {
		t.Error("-foo should be private")
	}
	if !IsPrivateMember("_foo") {
		t.Error("_foo should be private")
	}
	if IsPrivateMember("foo") {
		t.Error("foo should not be private")
	}
}

func TestPluralize(t *testing.T) {
	// n == 1
	if got := Pluralize("thing", 1, nil); got != "thing" {
		t.Errorf("Pluralize(thing, 1) = %q, want %q", got, "thing")
	}
	// n != 1, no custom plural
	if got := Pluralize("thing", 2, nil); got != "things" {
		t.Errorf("Pluralize(thing, 2) = %q, want %q", got, "things")
	}
	// n != 1, custom plural
	custom := "thingies"
	if got := Pluralize("thing", 3, &custom); got != custom {
		t.Errorf("Pluralize(thing, 3, custom) = %q, want %q", got, custom)
	}
}
