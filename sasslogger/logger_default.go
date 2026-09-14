// Copyright 2026 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasslogger

// dart-source: lib/src/logger/default.dart

import (
	"os"
)

// supportsAnsiEscapes detects whether the current terminal supports ANSI
// escape codes for color output. A TERM of "dumb" always opts out;
// COLORTERM truecolor/24bit always opts in; otherwise stderr must be a
// character device. Stderr (not stdout) is probed here because that is
// where warnings go; Dart instead consults stdout.supportsAnsiEscapes.
func supportsAnsiEscapes() bool {
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// NewDefaultLogger creates a StderrLogger with ANSI color auto-detection,
// for use when no other logger is selected. Dart's DefaultLogger delegates
// to a lazily-initialized global StderrLogger; Go builds a fresh logger per
// call since there is no shared global to initialize.
// Matches Dart: Logger.defaultLogger / DefaultLogger.
func NewDefaultLogger(unicode bool) *StderrLogger {
	return NewStderrLogger(supportsAnsiEscapes(), unicode)
}
