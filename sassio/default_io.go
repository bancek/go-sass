// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassio

// dart-source: lib/src/io/vm.dart

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/term"
)

// DefaultIO is the default [IO] implementation backed by the OS kernel
// filesystem.
//
// This file holds the Platform-query, stdout/stderr, UTF-8, and trace-shape
// mechanics, matching how Dart keeps the contract in io/interface.dart and
// the VM behavior in io/vm.dart: contract docs live on [IO], behavior notes
// live here. Case-corrected canonicalization lives alongside in
// default_io_canonicalize.go plus the default_io_real_case_path.go
// case-corrector. Dart's watchDir has no counterpart (watch mode is out of
// scope); the UTF-8 "Invalid UTF-8" span error Dart raises inside readFile
// is raised by the compile entry instead, since ReadFile returns raw bytes.
type DefaultIO struct {
	// realCaseCacheMu guards realCaseCache for concurrent compilations.
	realCaseCacheMu sync.RWMutex
	// realCaseCache memoizes realCasePath results per directory chain,
	// matching Dart's _realCaseCache.
	realCaseCache map[string]string
	// exitCode is the process exit code, kept per instance (Dart re-exports
	// the process-wide dart:io exitCode instead).
	exitCode int
}

// NewDefaultIO creates a DefaultIO with an empty case-correction cache and a
// zero exit code.
func NewDefaultIO() *DefaultIO {
	return &DefaultIO{realCaseCache: make(map[string]string)}
}

// newFSE wraps an OS error into a [FileSystemException] carrying path,
// mirroring how Dart's FileSystemException always pairs message with path.
func newFSE(err error, path string) *FileSystemException {
	return &FileSystemException{
		Message: err.Error(),
		Path:    path,
		Err:     err,
	}
}

// ReadFile returns the raw bytes at name. UTF-8 decoding is the compile
// entry's job: Dart's readFile decodes inline and throws a SassException
// ("Invalid UTF-8" with a SourceFile span at the first bad byte), which has
// no equivalent at this layer since Go hands back undecided bytes.
func (d *DefaultIO) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, newFSE(err, name)
	}
	return data, nil
}

// WriteFile writes contents to path, matching Dart's writeAsStringSync
// except that Dart encodes a string as UTF-8 where Go takes raw bytes.
func (d *DefaultIO) WriteFile(path string, contents []byte) error {
	if err := os.WriteFile(path, contents, 0644); err != nil {
		return newFSE(err, path)
	}
	return nil
}

// DeleteFile removes the file at path, matching Dart's deleteSync.
func (d *DefaultIO) DeleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return newFSE(err, path)
	}
	return nil
}

// ReadStdin blocks reading standard input until EOF. Dart awaits the same
// stream through the system encoding; Go returns the raw bytes instead.
func (d *DefaultIO) ReadStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// FileExists reports whether a regular file (not a directory) exists at
// path, matching Dart's File.existsSync.
func (d *DefaultIO) FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirExists reports whether a directory exists at path, matching Dart's
// Directory.existsSync.
func (d *DefaultIO) DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// LinkExists reports whether a symbolic link exists at path. It uses Lstat
// so the link itself is probed rather than its target, matching Dart's
// Link.existsSync.
func (d *DefaultIO) LinkExists(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// EnsureDir creates path and any missing ancestors, matching Dart's
// createSync(recursive: true).
func (d *DefaultIO) EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return newFSE(err, path)
	}
	return nil
}

// ListDir returns the files (not sub-directories) under path, descending
// transitively when recursive is true. Dart filters the same listing down
// to File entities; Go skips directory entries during the walk instead,
// pruning subtrees early for the non-recursive case.
func (d *DefaultIO) ListDir(path string, recursive bool) ([]string, error) {
	var result []string
	walkFn := func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			result = append(result, p)
		}
		if !recursive && p != path && entry.IsDir() {
			return fs.SkipDir
		}
		return nil
	}
	if err := filepath.WalkDir(path, walkFn); err != nil {
		return nil, newFSE(err, path)
	}
	return result, nil
}

// Realpath returns the resolved physical path, matching Dart's realpath.
//
// Matches Dart: realpath in lib/src/io/vm.dart (via dart:io.FileSystemEntity.resolveSymbolicLinksSync)
func (d *DefaultIO) Realpath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", newFSE(err, path)
	}
	return resolved, nil
}

// ModificationTime returns the mtime of path. A missing file surfaces as a
// *FileSystemException, matching Dart's explicit not-found throw out of
// FileStat.statSync.
func (d *DefaultIO) ModificationTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, newFSE(err, path)
	}
	return info.ModTime(), nil
}

// GetEnvironmentVariable reads name from the process environment, matching
// Dart's Platform.environment lookup. Absence reports as "" (Dart yields
// null at this layer).
func (d *DefaultIO) GetEnvironmentVariable(name string) string {
	return os.Getenv(name)
}

// ExitCode returns the stored exit code. It is per-instance state, unlike
// Dart's process-wide dart:io exitCode re-export.
func (d *DefaultIO) ExitCode() int {
	return d.exitCode
}

// SetExitCode stores the exit code without exiting, so callers (and tests)
// can decide when to act on it.
func (d *DefaultIO) SetExitCode(code int) {
	d.exitCode = code
}

// PrintOutput writes message verbatim to standard output (no newline
// added). Go-only helper for the CLI runner; Dart prints through safePrint
// instead.
func (d *DefaultIO) PrintOutput(message string) {
	_, _ = fmt.Fprint(os.Stdout, message)
}

// SafePrint writes message plus a newline to standard output, matching
// Dart's print passthrough. Thread-safe per the interface contract.
func (d *DefaultIO) SafePrint(message string) {
	_, _ = fmt.Fprintln(os.Stdout, message)
}

// PrintError writes message plus a newline to standard error, matching
// Dart's stderr.writeln passthrough. Thread-safe per the interface contract.
func (d *DefaultIO) PrintError(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
}

// IsWindows reports whether the process runs on Windows, matching Dart's
// Platform.isWindows (read off the build target here instead).
func (d *DefaultIO) IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS reports whether the process runs on macOS, matching Dart's
// Platform.isMacOS.
func (d *DefaultIO) IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// HasTerminal reports whether stdout is an interactive terminal, matching
// Dart's stdout.hasTerminal probe.
func (d *DefaultIO) HasTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// SupportsAnsiEscapes reports whether stdout likely renders ANSI escapes.
// A TERM of "dumb" always opts out; otherwise a terminal probe decides.
// Dart trusts stdout.supportsAnsiEscapes on Windows only (TERM has false
// negatives elsewhere) and returns true for other terminals — Go's probe
// keeps the Windows-conditional shape but reads the terminal directly.
func (d *DefaultIO) SupportsAnsiEscapes() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	fd := int(os.Stdout.Fd())
	return term.IsTerminal(fd)
}

// Stat returns file info for name. Go-only infrastructure backing the
// import cache and import-path resolution; Dart's interface.dart declares
// no equivalent (its statSync use stays inside modificationTime).
func (d *DefaultIO) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, newFSE(err, name)
	}
	return info, nil
}
