package sassurl

import (
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input         string
		wantString    string
		wantScheme    string
		wantHost      string
		wantAuthority bool
		wantPath      string
		wantHasQuery  bool
		wantQuery     string
		wantHasFrag   bool
		wantFragment  string
		wantAbsolute  bool
	}{
		{"", "", "", "", false, "", false, "", false, "", false},
		{".", "./", "", "", false, "./", false, "", false, "", false},
		{"..", "../", "", "", false, "../", false, "", false, "", false},
		{"./", "./", "", "", false, "./", false, "", false, "", false},
		{"../", "../", "", "", false, "../", false, "", false, "", false},
		{"foo", "foo", "", "", false, "foo", false, "", false, "", false},
		{"foo/bar", "foo/bar", "", "", false, "foo/bar", false, "", false, "", false},
		{"foo/./bar", "foo/bar", "", "", false, "foo/bar", false, "", false, "", false},
		{"foo/../bar", "bar", "", "", false, "bar", false, "", false, "", false},
		{"http://example.com/foo/bar", "http://example.com/foo/bar", "http", "example.com", true, "/foo/bar", false, "", false, "", true},
		{"HTTP://EXAMPLE.COM/", "http://example.com/", "http", "example.com", true, "/", false, "", false, "", true},
		{"http://example.com:80/", "http://example.com/", "http", "example.com", true, "/", false, "", false, "", true},
		{"https://example.com:443/", "https://example.com/", "https", "example.com", true, "/", false, "", false, "", true},
		{"sass:color", "sass:color", "sass", "", false, "color", false, "", false, "", true},
		{"u:foo/bar", "u:foo/bar", "u", "", false, "foo/bar", false, "", false, "", true},
		{"pkg:%66oo", "pkg:foo", "pkg", "", false, "foo", false, "", false, "", true},
		{"u:%25percent", "u:%25percent", "u", "", false, "%25percent", false, "", false, "", true},
		{"file:///tmp/a.scss", "file:///tmp/a.scss", "file", "", true, "/tmp/a.scss", false, "", false, "", true},
		{"file:///tmp/a b.scss", "file:///tmp/a%20b.scss", "file", "", true, "/tmp/a%20b.scss", false, "", false, "", true},
		{"file:///tmp/a%20b.scss", "file:///tmp/a%20b.scss", "file", "", true, "/tmp/a%20b.scss", false, "", false, "", true},
		{"data:text/plain,hello", "data:text/plain,hello", "data", "", false, "text/plain,hello", false, "", false, "", true},
		{"data:;charset=utf-8,a%20b", "data:;charset=utf-8,a%20b", "data", "", false, ";charset=utf-8,a%20b", false, "", false, "", true},
		{"a b", "a%20b", "", "", false, "a%20b", false, "", false, "", false},
		{"a%20b", "a%20b", "", "", false, "a%20b", false, "", false, "", false},
		{"foo%2fbar", "foo%2Fbar", "", "", false, "foo%2Fbar", false, "", false, "", false},
		{"a?b#c", "a?b#c", "", "", false, "a", true, "b", true, "c", false},
		{"a#b", "a#b", "", "", false, "a", false, "", true, "b", false},
		{"a?", "a?", "", "", false, "a", true, "", false, "", false},
		{"http://user:pass@host:8080/p?q#f", "http://user:pass@host:8080/p?q#f", "http", "host", true, "/p", true, "q", true, "f", false},
		{"http://[::1]/", "http://[::1]/", "http", "::1", true, "/", false, "", false, "", true},
		{"http://[v1.a]/", "http://[v1.a]/", "http", "[v1.a]", true, "/", false, "", false, "", true},
		{"http://example.com/a/../b", "http://example.com/b", "http", "example.com", true, "/b", false, "", false, "", true},
		{"http://example.com/a/./b", "http://example.com/a/b", "http", "example.com", true, "/a/b", false, "", false, "", true},
		{"../a", "../a", "", "", false, "../a", false, "", false, "", false},
		{"./a", "a", "", "", false, "a", false, "", false, "", false},
		{"a/../b", "b", "", "", false, "b", false, "", false, "", false},
		{"a/./b", "a/b", "", "", false, "a/b", false, "", false, "", false},
		{"a//b", "a//b", "", "", false, "a//b", false, "", false, "", false},
		{`\foo\bar`, "/foo/bar", "", "", false, "/foo/bar", false, "", false, "", false},
		{`C:\foo\bar`, "c:/foo/bar", "c", "", false, "/foo/bar", false, "", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			u, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.input, err)
			}
			if got := u.String(); got != tt.wantString {
				t.Errorf("String() = %q, want %q", got, tt.wantString)
			}
			if u.Scheme != tt.wantScheme {
				t.Errorf("Scheme = %q, want %q", u.Scheme, tt.wantScheme)
			}
			if u.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", u.Host, tt.wantHost)
			}
			if u.HasAuthority != tt.wantAuthority {
				t.Errorf("HasAuthority = %v, want %v", u.HasAuthority, tt.wantAuthority)
			}
			if u.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", u.Path, tt.wantPath)
			}
			if u.HasQuery != tt.wantHasQuery {
				t.Errorf("HasQuery = %v, want %v", u.HasQuery, tt.wantHasQuery)
			}
			if u.Query != tt.wantQuery {
				t.Errorf("Query = %q, want %q", u.Query, tt.wantQuery)
			}
			if u.HasFragment != tt.wantHasFrag {
				t.Errorf("HasFragment = %v, want %v", u.HasFragment, tt.wantHasFrag)
			}
			if u.Fragment != tt.wantFragment {
				t.Errorf("Fragment = %q, want %q", u.Fragment, tt.wantFragment)
			}
			if u.IsAbs() != tt.wantAbsolute {
				t.Errorf("IsAbs = %v, want %v", u.IsAbs(), tt.wantAbsolute)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		base, ref string
		want      string
	}{
		{"u:foo/bar", "baz/qux", "u:foo/baz/qux"},
		{"u:orange", "midstream", "u:midstream"},
		{"u:entrypoint", "orange", "u:orange"},
		{"http://example.com/a/b", "", "http://example.com/a/b"},
		{"http://example.com/a/b", ".", "http://example.com/a/"},
		{"http://example.com/a/b", "..", "http://example.com/"},
		{"http://example.com/a/b", "../c", "http://example.com/c"},
		{"http://example.com/a/b", "/c", "http://example.com/c"},
		{"http://example.com/a/b", "c", "http://example.com/a/c"},
		{"http://example.com/a/b", "?q", "http://example.com/a/b?q"},
		{"http://example.com/a/b", "#f", "http://example.com/a/b#f"},
		{"file:///tmp/foo/bar.scss", "baz/qux.scss", "file:///tmp/foo/baz/qux.scss"},
		{"foo/bar", "baz/qux", "foo/baz/qux"},
		{"foo/bar", "../baz", "baz"},
		{"foo/bar", "./baz", "foo/baz"},
		{"foo/bar", "/baz", "/baz"},
		{"foo/bar", "?q", "foo/bar?q"},
		{"foo/bar", "#f", "foo/bar#f"},
	}
	for _, tt := range tests {
		t.Run(tt.base+" + "+tt.ref, func(t *testing.T) {
			base, err := Parse(tt.base)
			if err != nil {
				t.Fatal(err)
			}
			ref, err := Parse(tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			if got := base.Resolve(ref).String(); got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.base, tt.ref, got, tt.want)
			}
		})
	}
}

func TestFilePaths(t *testing.T) {
	tests := []struct {
		path    string
		wantURL string
	}{
		{"/tmp/a.scss", "file:///tmp/a.scss"},
		{"/tmp/a b.scss", "file:///tmp/a%20b.scss"},
	}
	for _, tt := range tests {
		u, err := FileURLFromAbsPath(tt.path)
		if err != nil {
			t.Fatalf("FileURLFromAbsPath(%q): %v", tt.path, err)
		}
		if got := u.String(); got != tt.wantURL {
			t.Errorf("FileURLFromAbsPath(%q) = %q, want %q", tt.path, got, tt.wantURL)
		}
		got, ok := u.ToFilePathString()
		wantPath := filepath.FromSlash(tt.path)
		if !ok || got != wantPath {
			t.Errorf("ToFilePathString(%q) = %q, %v, want %q, true", tt.wantURL, got, ok, wantPath)
		}
	}
}

func TestPathToURI(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"a.scss", "a.scss"},
		{"dir/a.scss", "dir/a.scss"},
		{"./a.scss", "a.scss"},
		{"../a.scss", "../a.scss"},
		{`C:\a\b.scss`, "C%3A%5Ca%5Cb.scss"},
	}
	for _, tt := range tests {
		u, err := pathToURIWithOS(tt.path, false)
		if err != nil {
			t.Fatalf("pathToURIWithOS(%q, false): %v", tt.path, err)
		}
		if got := u.String(); got != tt.want {
			t.Errorf("pathToURIWithOS(%q, false) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestWindowsPathToURI(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{`C:\a\b.scss`, "file:///C:/a/b.scss"},
		{`\\server\share\a.scss`, "file://server/share/a.scss"},
	}
	for _, tt := range tests {
		u, err := pathToURIWithOS(tt.path, true)
		if err != nil {
			t.Fatalf("pathToURIWithOS(%q): %v", tt.path, err)
		}
		if got := u.String(); got != tt.want {
			t.Errorf("pathToURIWithOS(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestWindowsFilePaths(t *testing.T) {
	tests := []struct {
		path    string
		wantURL string
	}{
		{`C:\foo\bar.scss`, "file:///C:/foo/bar.scss"},
		{`\\server\share\file.scss`, "file://server/share/file.scss"},
	}
	for _, tt := range tests {
		u, err := windowsFileURL(normalizeFilePath(tt.path, true))
		if err != nil {
			t.Fatalf("windowsFileURL(%q): %v", tt.path, err)
		}
		if got := u.String(); got != tt.wantURL {
			t.Errorf("windowsFileURL(%q) = %q, want %q", tt.path, got, tt.wantURL)
		}
	}
	got, ok := uriToPath("/C:/foo/bar.scss", nil, true)
	if !ok || got != `C:\foo\bar.scss` {
		t.Errorf("uriToPath = %q, %v, want %q, true", got, ok, `C:\foo\bar.scss`)
	}
}

func TestDataURLFromString(t *testing.T) {
	tests := []struct {
		contents string
		want     string
	}{
		{"", "data:;charset=utf-8,"},
		{"a", "data:;charset=utf-8,a"},
		{"a b", "data:;charset=utf-8,a%20b"},
		{".x { color: red; }", "data:;charset=utf-8,.x%20%7B%20color:%20red;%20%7D"},
	}
	for _, tt := range tests {
		if got := DataURLFromString(tt.contents); got != tt.want {
			t.Errorf("DataURLFromString(%q) = %q, want %q", tt.contents, got, tt.want)
		}
	}
}

func TestEqual(t *testing.T) {
	a, _ := Parse("HTTP://EXAMPLE.COM/")
	b, _ := Parse("http://example.com/")
	if !a.Equal(b) {
		t.Error("expected equal URLs")
	}
	c, _ := Parse("http://example.com/x")
	if a.Equal(c) {
		t.Error("expected unequal URLs")
	}
}
