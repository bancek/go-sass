package eval

import (
	"net/url"
	"testing"

	"github.com/bancek/go-sass/sassurl"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := sassurl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// file: URL paths always carry a leading slash, so a Windows path arrives
// as "/C:/..." — a form the OS rejects. The drive-letter strip is what
// lets host-provided file URLs (file:///C:/...) resolve on Windows;
// without it TestFileImporter_* fail there. Pure string logic, so the
// cases run on every OS.
func TestFileURLToOSPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/C:/dir/file.scss", "C:/dir/file.scss"},
		{"/c:/dir/file.scss", "c:/dir/file.scss"},
		{"/C:", "C:"},
		{"/tmp/x.scss", "/tmp/x.scss"},
		{"/", "/"},
		{"", ""},
		{"C:/x.scss", "C:/x.scss"},
		{"relative/x.scss", "relative/x.scss"},
		{"/CC/x.scss", "/CC/x.scss"},
		{"/1:/x.scss", "/1:/x.scss"},
	}
	for _, tc := range cases {
		if got := fileURLToOSPath(tc.in); got != tc.want {
			t.Errorf("fileURLToOSPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// humanize ignores original URLs without a scheme: they can be ambiguous with
// file: URLs resolved relative to the current working directory (#2777, #2778).
// Here the schemeless dep/... original is shorter, so without the filter it
// would win over the file: URL.
func TestHumanizeIgnoresSchemelessOriginalURLs(t *testing.T) {
	cache := NewImportCache(nil, false)

	canonical := mustParseURL(t, "file:///app/dep/_lib.scss")
	schemeless := mustParseURL(t, "dep/_lib.scss")
	if schemeless.Scheme != "" {
		t.Fatalf("expected schemeless URL, got scheme %q", schemeless.Scheme)
	}
	withScheme := mustParseURL(t, "file:///app/dep/_lib.scss")
	if withScheme.Scheme == "" {
		t.Fatal("expected a scheme")
	}

	for _, original := range []*url.URL{schemeless, withScheme} {
		cache.canonicalizeCache[canonicalizeKey{url: original.String()}] = &CanonicalizeResult{
			CanonicalURL: canonical,
			OriginalURL:  original,
		}
	}

	if got := cache.Humanize(canonical); got != "file:///app/dep/_lib.scss" {
		t.Errorf("Humanize = %q, want %q", got, "file:///app/dep/_lib.scss")
	}
}

// Without any schemed original URL, Humanize falls back to the canonical URL
// itself (unchanged behavior, #2778).
func TestHumanizeSchemelessOnlyFallsBackToCanonical(t *testing.T) {
	cache := NewImportCache(nil, false)

	canonical := mustParseURL(t, "file:///app/dep/_lib.scss")
	schemeless := mustParseURL(t, "dep/_lib.scss")
	cache.canonicalizeCache[canonicalizeKey{url: schemeless.String()}] = &CanonicalizeResult{
		CanonicalURL: canonical,
		OriginalURL:  schemeless,
	}

	if got := cache.Humanize(canonical); got != "file:///app/dep/_lib.scss" {
		t.Errorf("Humanize = %q, want %q", got, "file:///app/dep/_lib.scss")
	}
}
