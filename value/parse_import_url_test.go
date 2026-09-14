package value

import "testing"

func TestLooksLikeWindowsAbsolutePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`C:\foo\bar.scss`, true},
		{`C:/foo/bar.scss`, true},
		{`D:\`, true},
		{`X:\a\b\c\d`, true},
		{`\foo\bar.scss`, true},
		{`/absolute/unix/path.scss`, true},
		{`relative/path.scss`, false},
		{`../relative/path.scss`, false},
		{`foo.scss`, false},
		{``, false},
		{`C:`, false},
		{`C:foo.scss`, false},
		{`:\,`, false},
		{`5:\path`, false},
		{`AB:\path`, false},
		{`C:\`, true},
		{`c:\path`, true},
		{`z:\path`, true},
		{`\\server\share`, true},
		{`\\server\share\file`, true},
	}

	for _, tt := range tests {
		got := looksLikeWindowsAbsolutePath(tt.input)
		if got != tt.want {
			t.Errorf("looksLikeWindowsAbsolutePath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseImportUrl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"windows backslash", `C:\path\to\file.scss`, `file:///C:/path/to/file.scss`},
		{"windows forward slash", `D:/other/style.sass`, `file:///D:/other/style.sass`},
		{"windows lowercase drive", `c:\Foo\bar.sass`, `file:///c:/Foo/bar.sass`},
		{"unc path", `\\server\share\file.scss`, `file://server/share/file.scss`},
		{"root relative backslash", `\foo\bar.scss`, `file:///foo/bar.scss`},
		{"backslash relative", `a\b.scss`, `a\b.scss`},
		{"unix absolute", `/srv/sass/style.scss`, `/srv/sass/style.scss`},
		{"relative path", `../theme/_vars.scss`, `../theme/_vars.scss`},
		{"simple name", `style.scss`, `style.scss`},
		{"http url", `https://example.com/a.css`, `https://example.com/a.css`},
		{"empty string", ``, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseImportUrl(tt.input)
			if got != tt.want {
				t.Errorf("ParseImportUrl(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
