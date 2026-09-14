// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/importer/utils.dart (resolveImportPath section)

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
)

// resolveImportPath resolves an imported path using the same logic as the
// filesystem importer.
//
// This tries to fill in extensions and partial prefixes and check for a
// directory default. If no file can be found, it returns ("", nil).
//
// If multiple files match the same path, it returns an error indicating the
// import is ambiguous.
//
// The fromImport flag selects @import semantics (`.import` midway files and
// `index.import` defaults are considered); Dart threads the same flag through
// an ambient canonicalization context, while Go passes it explicitly.
func resolveImportPath(io sassio.IO, path string, fromImport bool) (string, error) {
	ext := filepath.Ext(path)
	if ext == ".sass" || ext == ".scss" || ext == ".css" {
		if fromImport {
			importPath := withoutExtension(path) + ".import" + ext
			result, err := exactlyOne(tryPath(io, importPath))
			if err != nil {
				return "", err
			}
			if result != "" {
				return result, nil
			}
		}
		return exactlyOne(tryPath(io, path))
	}

	if fromImport {
		result, err := exactlyOne(tryPathWithExtensions(io, path+".import"))
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
	}

	result, err := exactlyOne(tryPathWithExtensions(io, path))
	if err != nil {
		return "", err
	}
	if result != "" {
		return result, nil
	}

	return tryPathAsDirectory(io, path, fromImport)
}

// tryPath returns the partial ("_" + basename) and/or the path itself,
// whichever exist on disk, partial first. If neither exists, it returns an
// empty list; if both exist, exactlyOne reports the ambiguity.
func tryPath(io sassio.IO, path string) []string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	partial := filepath.Join(dir, "_"+base)
	var result []string
	if fileExists(io, partial) {
		result = append(result, partial)
	}
	if fileExists(io, path) {
		result = append(result, path)
	}
	return result
}

// tryPathWithExtensions tries the path with .sass and .scss extensions via
// tryPath, collecting both so a .sass/.scss pair reports an ambiguity. .css
// is only tried when neither Sass extension matches anything.
func tryPathWithExtensions(io sassio.IO, path string) []string {
	var result []string
	result = append(result, tryPath(io, path+".sass")...)
	result = append(result, tryPath(io, path+".scss")...)
	if len(result) > 0 {
		return result
	}
	return tryPath(io, path+".css")
}

// tryPathAsDirectory resolves path as a directory default: index.import.*
// under @import semantics, then index.*. A non-directory returns ("", nil).
func tryPathAsDirectory(io sassio.IO, path string, fromImport bool) (string, error) {
	if !dirExists(io, path) {
		return "", nil
	}

	if fromImport {
		result, err := exactlyOne(tryPathWithExtensions(io, filepath.Join(path, "index.import")))
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
	}

	return exactlyOne(tryPathWithExtensions(io, filepath.Join(path, "index")))
}

// exactlyOne returns the single path from paths, or ("", nil) when empty. On
// multiple matches it returns an ambiguity error listing each candidate as a
// file: URL.
func exactlyOne(paths []string) (string, error) {
	switch len(paths) {
	case 0:
		return "", nil
	case 1:
		return paths[0], nil
	default:
		var lines []string
		lines = append(lines, "It's not clear which file to import. Found:")
		for _, p := range paths {
			// Candidates render as file: URLs, matching the human-readable
			// listing format.
			u := &url.URL{Scheme: "file", Path: filepath.ToSlash(p)}
			lines = append(lines, "  "+sasscommon.PrettyUri(u))
		}
		return "", sasscommon.NewSassScriptException(strings.Join(lines, "\n"), nil)
	}
}

// fileExists returns whether a non-directory file exists at path. Lookup
// failures count as absent.
func fileExists(io sassio.IO, path string) bool {
	info, err := io.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists returns whether a directory exists at path. Lookup failures
// count as absent.
func dirExists(io sassio.IO, path string) bool {
	info, err := io.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// withoutExtension strips the trailing extension from path so @import-only
// candidates can insert the `.import` infix before it.
func withoutExtension(path string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)]
}
