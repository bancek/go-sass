// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/stylesheet.dart (parseImportUrl section)

import (
	goUrl "net/url"
	"strings"

	"github.com/bancek/go-sass/util"
)

// Parses [rawURL] as an import URL.
//
// Matches Dart's parseImportUrl exactly:
//   - Absolute Windows paths (drive-letter, UNC, and backslash-rooted like
//     \foo) are converted to file:// URLs, EXCEPT those that are root-relative
//     in URL context (only `/` counts, per p.url.isRootRelative).
//   - Otherwise the URL is validated but returned UNCHANGED (no re-encoding).
//
// In Dart this is a protected method on StylesheetParser; Go lifts it to a
// package-level function so both the stylesheet @import rule and the
// indented-syntax importer share it (the StylesheetParser.parseImportUrl
// method is a thin wrapper). Its sibling isPlainImportUrl lives alongside
// the @import rule in parse_stylesheet_atrule.go, not here; validation that
// Dart gets from Uri.parse throwing FormatException happens at the call
// sites, which report "Invalid URL".
func ParseImportUrl(rawURL string) string {
	if looksLikeWindowsAbsolutePath(rawURL) && !isRootRelative(rawURL) {
		// UNC path: \\server\share\... → file://server/share/...
		if len(rawURL) >= 2 && rawURL[0] == '\\' && rawURL[1] == '\\' {
			afterPrefix := strings.ReplaceAll(rawURL[2:], "\\", "/")
			before, after, ok := strings.Cut(afterPrefix, "/")
			host := afterPrefix
			path := ""
			if ok {
				host = before
				path = "/" + after
			}
			u := &goUrl.URL{
				Scheme: "file",
				Host:   host,
				Path:   path,
			}
			return u.String()
		}
		// Backslash-rooted path: \foo\bar.scss → file:///foo/bar.scss
		if rawURL[0] == '\\' {
			return "file:///" + strings.ReplaceAll(rawURL[1:], "\\", "/")
		}
		// Drive-letter path: C:\foo → file:///C:/foo
		return "file:///" + strings.ReplaceAll(rawURL, "\\", "/")
	}
	// Throw a FormatException if url is invalid (validated by the caller).
	// Return url unchanged on success (matches Dart: Uri.parse(url); return url).
	return rawURL
}

// looksLikeWindowsAbsolutePath returns whether [path] is an absolute Windows
// path: a drive-letter path (C:\foo), a UNC path (\\server\share), or a
// backslash- or slash-rooted path (\foo, /foo). This matches Dart's
// p.windows.isAbsolute.
//
// Backwards compatibility for stylesheets that allow absolute Windows paths
// in imports is why this check exists at all.
func looksLikeWindowsAbsolutePath(path string) bool {
	if len(path) >= 1 && (path[0] == '\\' || path[0] == '/') {
		return true
	}
	// Drive-letter path (e.g. C:\foo or D:/bar)
	return len(path) >= 3 &&
		util.IsAlphabetic(int(path[0])) &&
		path[1] == ':' &&
		(path[2] == '\\' || path[2] == '/')
}

// isRootRelative returns whether [path] is root-relative in URL context.
//
// Matches Dart's p.url.isRootRelative: only `/` counts as a root separator —
// a backslash-rooted path like \foo is NOT root-relative in URL context.
func isRootRelative(path string) bool {
	return len(path) >= 1 && path[0] == '/'
}
