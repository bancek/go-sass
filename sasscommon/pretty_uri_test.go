package sasscommon

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sassurl"
)

func TestPrettyUriFileScheme(t *testing.T) {
	u, _ := sassurl.Parse("file:///some/path/file.scss")
	got := PrettyUri(u)
	if got == "" {
		t.Error("PrettyUri returned empty string for file:// URI")
	}
	if strings.HasPrefix(got, "file://") {
		t.Errorf("PrettyUri should return path, not full URL: %q", got)
	}
}

func TestPrettyUriNonFileScheme(t *testing.T) {
	u, _ := sassurl.Parse("https://example.com/path")
	got := PrettyUri(u)
	want := "https://example.com/path"
	if got != want {
		t.Errorf("PrettyUri(https://...) = %q, want %q", got, want)
	}
}

func TestPrettyUriEmptyScheme(t *testing.T) {
	u, _ := sassurl.Parse("/relative/path/file.scss")
	got := PrettyUri(u)
	if got == "" {
		t.Error("PrettyUri returned empty string for empty-scheme URL")
	}
}

func TestPrettyUriDecodesPercentEscapes(t *testing.T) {
	u, _ := sassurl.Parse("file:///tmp/bad%20file.scss")
	got := PrettyUri(u)
	if !strings.Contains(got, "bad file.scss") {
		t.Errorf("PrettyUri(%q) = %q, want decoded file name", u.String(), got)
	}
}

func TestPrettyUriRelativeShorter(t *testing.T) {
	// Verify that relative paths from CWD are used when they're shorter.
	cwd := "/some/very/deep/nested/path"
	abs := filepath.Join(cwd, "file.scss")
	u := &sassurl.URL{Scheme: "file", HasAuthority: true, Path: abs}
	got := PrettyUri(u)
	if strings.Count(got, string(filepath.Separator)) > strings.Count(abs, string(filepath.Separator)) {
		t.Errorf("PrettyUri returned a path with more segments than absolute: %q", got)
	}
}
