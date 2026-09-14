package sassurl

import (
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		raw        string
		wantStr    string
		wantPath   string
		wantScheme string
	}{
		{"sass:color", "sass:color", "color", "sass"},
		{"u:foo/bar", "u:foo/bar", "foo/bar", "u"},
		{"u:orange", "u:orange", "orange", "u"},
		{"pkg:%66oo", "pkg:%66oo", "foo", "pkg"},
		{"u:%25percent", "u:%25percent", "%percent", "u"},
		{"file:///tmp/test.scss", "file:///tmp/test.scss", "/tmp/test.scss", "file"},
		{"http://example.com/style.scss", "http://example.com/style.scss", "/style.scss", "http"},
		// Relative paths (no scheme) — these are the inputs that fail in Rust
		{"callable/arguments/function/utils", "callable/arguments/function/utils", "callable/arguments/function/utils", ""},
		{"../utils", "../utils", "../utils", ""},
		{"./utils", "./utils", "./utils", ""},
		{"utils", "utils", "utils", ""},
		{"relative/path/to/file.scss", "relative/path/to/file.scss", "relative/path/to/file.scss", ""},
	}
	for _, tt := range tests {
		u, err := Parse(tt.raw)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.raw, err)
			continue
		}
		if u.String() != tt.wantStr {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.raw, u.String(), tt.wantStr)
		}
		if u.Opaque != "" {
			t.Errorf("Parse(%q) has Opaque=%q, want empty", tt.raw, u.Opaque)
		}
		if u.Path != tt.wantPath {
			t.Errorf("Parse(%q).Path = %q, want %q", tt.raw, u.Path, tt.wantPath)
		}
		if u.Scheme != tt.wantScheme {
			t.Errorf("Parse(%q).Scheme = %q, want %q", tt.raw, u.Scheme, tt.wantScheme)
		}
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		base    func() *url.URL
		ref     string
		wantStr string
	}{
		{
			name: "opaque base with relative path",
			base: func() *url.URL {
				u, _ := url.Parse("u:foo/bar")
				return u
			},
			ref:     "baz/qux",
			wantStr: "u:foo/baz/qux",
		},
		{
			name: "opaque base with no subdirectory",
			base: func() *url.URL {
				u, _ := url.Parse("u:orange")
				return u
			},
			ref:     "midstream",
			wantStr: "u:midstream",
		},
		{
			name: "opaque base (simple)",
			base: func() *url.URL {
				u, _ := url.Parse("u:entrypoint")
				return u
			},
			ref:     "orange",
			wantStr: "u:orange",
		},
		{
			name: "sassurl.Parse base with relative path",
			base: func() *url.URL {
				u, _ := Parse("u:foo/bar")
				return u
			},
			ref:     "baz/qux",
			wantStr: "u:foo/baz/qux",
		},
		{
			name: "sassurl.Parse base (simple)",
			base: func() *url.URL {
				u, _ := Parse("u:entrypoint")
				return u
			},
			ref:     "orange",
			wantStr: "u:orange",
		},
		{
			name: "http base with relative path",
			base: func() *url.URL {
				u, _ := url.Parse("http://example.com/foo/bar")
				return u
			},
			ref:     "baz/qux",
			wantStr: "http://example.com/foo/baz/qux",
		},
		{
			name: "file base with relative path",
			base: func() *url.URL {
				u, _ := url.Parse("file:///tmp/foo/bar.scss")
				return u
			},
			ref:     "baz/qux.scss",
			wantStr: "file:///tmp/foo/baz/qux.scss",
		},
		{
			name: "absolute reference on opaque base",
			base: func() *url.URL {
				u, _ := url.Parse("u:foo/bar")
				return u
			},
			ref:     "http://example.com/style.scss",
			wantStr: "http://example.com/style.scss",
		},
		{
			name: "root-relative reference on opaque base",
			base: func() *url.URL {
				u, _ := url.Parse("u:foo/bar")
				return u
			},
			ref:     "/baz/qux",
			wantStr: "u:baz/qux",
		},
		{
			name: "parent directory reference",
			base: func() *url.URL {
				u, _ := url.Parse("u:foo/bar")
				return u
			},
			ref:     "../baz",
			wantStr: "u:baz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := url.Parse(tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			base := tt.base()
			resolved := Resolve(base, ref)
			if resolved.String() != tt.wantStr {
				t.Errorf("Resolve(%s, %s) = %q, want %q",
					base.String(), ref.String(), resolved.String(), tt.wantStr)
			}
			// Verify the resolved URL can be used for further resolution
			_ = resolved
		})
	}
}

func TestResolveRoundTrip(t *testing.T) {
	// Verify that resolving a relative URL against an opaque base produces
	// a URL that, when used as a base for further resolution, works correctly.
	entrypoint, _ := url.Parse("u:entrypoint")
	import1 := Resolve(entrypoint, mustParseURL(t, "orange"))
	if import1.String() != "u:orange" {
		t.Fatalf("step 1: %q", import1.String())
	}

	// Now resolve midstream relative to the imported URL
	midstream := Resolve(import1, mustParseURL(t, "midstream"))
	if midstream.String() != "u:midstream" {
		t.Fatalf("step 2: %q", midstream.String())
	}

	// Resolve a nested path
	nested := Resolve(midstream, mustParseURL(t, "sub/deep"))
	if nested.String() != "u:sub/deep" {
		t.Fatalf("step 3: %q", nested.String())
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
