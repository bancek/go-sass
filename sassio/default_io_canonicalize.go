// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassio

// dart-source: lib/src/io.dart (canonicalize section)

import (
	"path/filepath"
	"runtime"
)

// Canonicalize returns the canonical form of path on disk: absolute,
// case-corrected on case-insensitive systems (macOS, Windows). It is LEXICAL —
// it never resolves symlinks (unlike filepath.EvalSymlinks), so e.g.
// `/var/...` stays `/var/...` rather than becoming `/private/var/...`.
//
// Dart normalizes the absolute path the same way and only case-corrects on
// possibly-case-insensitive systems; symlink names survive because
// correction matches directory entries rather than resolving links. Keeping
// symlinks unresolved matters for source map URLs, which must name the real
// filenames as the user addressed them.
//
// Matches Dart: io.canonicalize in lib/src/io.dart (p.normalize(p.absolute(path))
// + _realCasePath on case-insensitive filesystems)
func (d *DefaultIO) Canonicalize(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", newFSE(err, path)
	}

	// Case-sensitive systems need no case correction.
	if !caseInsensitiveFS() {
		return filepath.Clean(abs), nil
	}

	// Correct the case of each path component to match the real filesystem,
	// preserving symlink names (matching Dart's _realCasePath).
	return d.realCasePath(abs), nil
}

// caseInsensitiveFS returns true on macOS and Windows, which use
// case-insensitive (but case-preserving) filesystems by default. Dart frames
// the same gate as _couldBeCaseInsensitive and notes certainty is
// impossible, since individual macOS volumes are configured differently.
func caseInsensitiveFS() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}
