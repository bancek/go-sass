// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassio

// dart-source: lib/src/io.dart (realCasePath section)

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// realCasePath walks each component of path and corrects its casing to match
// the real filesystem. Results are cached in [DefaultIO.realCaseCache].
//
// Because correction matches directory entries (which name symlinks
// verbatim) instead of resolving links, symlink names survive — the same
// reason Canonicalize stays lexical. Callers other than Windows and macOS
// never reach here; those filesystems are treated as case-exact.
//
// Matches Dart: _realCasePath in lib/src/io.dart
func (d *DefaultIO) realCasePath(path string) string {
	// Fast path: read-lock cache lookup.
	d.realCaseCacheMu.RLock()
	cached, ok := d.realCaseCache[path]
	d.realCaseCacheMu.RUnlock()
	if ok {
		return cached
	}

	// On Windows, drive names are always case-insensitive — uppercase them.
	if runtime.GOOS == "windows" && len(path) >= 2 && path[1] == ':' {
		prefix := strings.ToUpper(path[:1])
		path = prefix + path[1:]
	}

	// Slow path: compute with write lock. realCaseHelperLocked checks the
	// cache first on every call (including the top-level one), so no
	// double-check is needed here — the race between RUnlock and Lock is
	// handled inside.
	d.realCaseCacheMu.Lock()
	defer d.realCaseCacheMu.Unlock()

	return d.realCaseHelperLocked(path)
}

// realCaseHelperLocked recursively resolves the case of each path component.
// Must be called with d.realCaseCacheMu write lock held.
//
// Each level reuses its parent's corrected directory, then picks the
// directory entry that matches the basename case-insensitively. A missing
// or unreadable directory (or no/ambiguous matches, meaning the volume is
// not actually case-insensitive) falls back to joining the basename as-is
// and lets the later use-time error speak, rather than failing here on
// symlink resolution. Every outcome is cached, including misses.
func (d *DefaultIO) realCaseHelperLocked(path string) string {
	dir := filepath.Dir(path)
	if dir == path {
		return path
	}

	if cached, ok := d.realCaseCache[path]; ok {
		return cached
	}

	realDir := d.realCaseHelperLocked(dir)
	base := filepath.Base(path)

	entries, err := os.ReadDir(realDir)
	if err != nil {
		result := filepath.Join(realDir, base)
		d.realCaseCache[path] = result
		return result
	}

	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), base) {
			result := filepath.Join(realDir, entry.Name())
			d.realCaseCache[path] = result
			return result
		}
	}

	result := filepath.Join(realDir, base)
	d.realCaseCache[path] = result
	return result
}
