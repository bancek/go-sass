// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassio

// dart-source: lib/src/io/interface.dart (interface only)

import (
	"os"
	"time"
)

// IO is the host I/O seam through which the compiler reads files,
// canonicalizes paths, and reports filesystem state. It may be implemented
// by host code; the compile path threads it through the options and the
// evaluator rather than reading ambient process state.
//
// Contract docs live here. Platform-query, UTF-8, and trace mechanics live
// on DefaultIO (default_io.go), matching how Dart keeps the interface in
// io/interface.dart and the VM behavior in io/vm.dart. Dart's watchDir has
// no counterpart here (watch mode is out of scope) and Dart's exitCode
// re-export becomes per-instance state on DefaultIO.
type IO interface {
	// ReadFile reads the file at name, returning its raw bytes.
	//
	// A failing read surfaces as a *FileSystemException carrying the path;
	// UTF-8 decoding (and Dart's "Invalid UTF-8" span error) is the caller's
	// job, done at the compile entry — unlike Dart's readFile, which decodes
	// inline.
	//
	// Matches Dart: readFile
	ReadFile(name string) ([]byte, error)

	// WriteFile writes contents to the file at path.
	//
	// A failing write surfaces as a *FileSystemException carrying the path;
	// Dart's writeFile encodes a string as UTF-8 where Go takes raw bytes.
	//
	// Matches Dart: writeFile
	WriteFile(path string, contents []byte) error

	// DeleteFile deletes the file at path.
	//
	// Matches Dart: deleteFile
	DeleteFile(path string) error

	// ReadStdin reads all data from standard input until EOF.
	//
	// Dart's readStdin is async (a Future decoded with the system encoding);
	// Go blocks synchronously and returns raw bytes.
	//
	// Matches Dart: readStdin
	ReadStdin() ([]byte, error)

	// FileExists reports whether a file exists at path.
	//
	// Matches Dart: fileExists
	FileExists(path string) bool

	// DirExists reports whether a directory exists at path.
	//
	// Matches Dart: dirExists
	DirExists(path string) bool

	// LinkExists reports whether a symbolic link exists at path.
	//
	// Matches Dart: linkExists
	LinkExists(path string) bool

	// EnsureDir creates a directory at path, including any necessary parents.
	//
	// Matches Dart: ensureDir
	EnsureDir(path string) error

	// ListDir lists the files (not sub-directories) in the directory at path.
	// If recursive is true, lists files in directories transitively as well.
	//
	// Dart spells the flag as a named parameter defaulting to false; Go
	// takes it positionally.
	//
	// Matches Dart: listDir
	ListDir(path string, recursive bool) ([]string, error)

	// Realpath returns the resolved physical path of path on disk, with
	// symbolic links resolved and matching the case of the physical file.
	//
	// Unlike Canonicalize this resolves symlinks (Dart notes the File
	// method works on directories too).
	//
	// Matches Dart: realpath
	Realpath(path string) (string, error)

	// ModificationTime returns the modification time of the file at path.
	//
	// A missing file surfaces as a *FileSystemException, matching Dart's
	// not-found throw out of statSync.
	//
	// Matches Dart: modificationTime
	ModificationTime(path string) (time.Time, error)

	// GetEnvironmentVariable returns the value of the environment variable
	// with the given name, or an empty string if not set.
	//
	// Dart returns null when unset; Go reports the same absence as "".
	//
	// Matches Dart: getEnvironmentVariable
	GetEnvironmentVariable(name string) string

	// ExitCode returns the current exit code.
	//
	// Dart re-exports the process-wide dart:io exitCode; Go keeps it as
	// per-instance DefaultIO state so compilations stay isolated.
	//
	// Matches Dart: exitCode getter
	ExitCode() int

	// SetExitCode sets the exit code.
	//
	// Matches Dart: exitCode setter
	SetExitCode(code int)

	// Canonicalize returns the canonical form of path on disk: absolute,
	// symlink-resolved, and on case-insensitive systems (macOS, Windows) with
	// the correct filesystem case. This ensures source map URLs match the real
	// filenames on disk.
	//
	// The lexical half (absolute + case correction, symlink names preserved)
	// is implemented in default_io_canonicalize.go; see there for why
	// symlinks must not resolve.
	//
	// Matches Dart: io.canonicalize in lib/src/io.dart
	Canonicalize(path string) (string, error)

	// PrintOutput writes message to standard output without adding a newline.
	//
	// Go-only helper used by the CLI runner (multi-file output, version
	// text); Dart's interface.dart has no equivalent — its executable layer
	// prints through safePrint instead.
	PrintOutput(message string)

	// SafePrint prints message followed by a newline to standard output.
	//
	// This method is thread-safe.
	//
	// Matches Dart: safePrint
	SafePrint(message string)

	// PrintError prints message followed by a newline to standard error.
	//
	// This method is thread-safe.
	//
	// Matches Dart: printError
	PrintError(message string)

	// IsWindows reports whether the current process is running on Windows.
	//
	// Matches Dart: isWindows
	IsWindows() bool

	// IsMacOS reports whether the current process is running on macOS.
	//
	// Matches Dart: isMacOS
	IsMacOS() bool

	// HasTerminal reports whether stdout is connected to an interactive terminal.
	//
	// Matches Dart: hasTerminal
	HasTerminal() bool

	// SupportsAnsiEscapes reports whether the terminal supports ANSI escape
	// sequences.
	//
	// See DefaultIO for the TERM-dumb/Windows trust mechanics behind this.
	//
	// Matches Dart: supportsAnsiEscapes
	SupportsAnsiEscapes() bool

	// Stat returns file info for name. This is Go-specific infrastructure
	// needed by import_cache and resolve_import_path; Dart's interface.dart
	// does not have an equivalent.
	Stat(name string) (os.FileInfo, error)
}
