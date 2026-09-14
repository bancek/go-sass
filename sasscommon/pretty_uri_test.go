package sasscommon

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrettyUriFileScheme(t *testing.T) {
	u, _ := url.Parse("file:///some/path/file.scss")
	got := PrettyUri(u)
	if got == "" {
		t.Error("PrettyUri returned empty string for file:// URI")
	}
	if strings.HasPrefix(got, "file://") {
		t.Errorf("PrettyUri should return path, not full URL: %q", got)
	}
}

func TestPrettyUriNonFileScheme(t *testing.T) {
	u, _ := url.Parse("https://example.com/path")
	got := PrettyUri(u)
	want := "https://example.com/path"
	if got != want {
		t.Errorf("PrettyUri(https://...) = %q, want %q", got, want)
	}
}

func TestPrettyUriEmptyScheme(t *testing.T) {
	u, _ := url.Parse("/relative/path/file.scss")
	got := PrettyUri(u)
	if got == "" {
		t.Error("PrettyUri returned empty string for empty-scheme URL")
	}
}

func TestPrettyUriRelativeShorter(t *testing.T) {
	// Verify that relative paths from CWD are used when they're shorter.
	cwd := "/some/very/deep/nested/path"
	abs := filepath.Join(cwd, "file.scss")
	u := &url.URL{Scheme: "file", Path: abs}
	got := PrettyUri(u)
	if strings.Count(got, string(filepath.Separator)) > strings.Count(abs, string(filepath.Separator)) {
		t.Errorf("PrettyUri returned a path with more segments than absolute: %q", got)
	}
}
