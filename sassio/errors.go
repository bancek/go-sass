// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sassio

// dart-source: lib/src/io/interface.dart (errors: FileSystemException)

import (
	"errors"
	"io/fs"
	"syscall"
)

// FileSystemException is an error indicating that a filesystem operation failed.
//
// On the Dart VM, the native FileSystemException carries an OSError with an
// error code (errno). Because Go has no shared errno across its host targets,
// the wrapped cause in Err is classified portably instead: [IsNotExist],
// [IsPermission], and [IsExist] probe both the fs.Err* sentinels and the
// raw syscall errno, so the checks keep working where errno reads as zero.
//
// Matches Dart: FileSystemException in lib/src/io/interface.dart (the
// VM version re-exports dart:io.FileSystemException which has osError.errorCode).
type FileSystemException struct {
	// Message is the human-readable description of the failure.
	Message string
	// Path is the filesystem path the operation targeted, when known.
	Path string
	// Err is the underlying OS error, for errors.Is/errors.As inspection.
	Err error
}

// Error renders the message, suffixed with the path when one is known,
// mirroring how Dart's FileSystemException always pairs message with path.
func (e *FileSystemException) Error() string {
	if e.Path != "" {
		return e.Message + ": " + e.Path
	}
	return e.Message
}

// Unwrap exposes the underlying OS error for errors.Is/errors.As.
func (e *FileSystemException) Unwrap() error {
	return e.Err
}

// IsNotExist reports whether the error indicates a file or directory does not
// exist. It checks the portable fs.ErrNotExist sentinel first and falls back
// to the raw ENOENT errno, since errno reads as zero on some targets.
// Matches Dart's OSError errorCode classification for missing files.
func (e *FileSystemException) IsNotExist() bool {
	if errors.Is(e.Err, fs.ErrNotExist) {
		return true
	}
	if errors.Is(e.Err, syscall.ENOENT) {
		return true
	}
	return false
}

// IsPermission reports whether the error indicates a permission denial
// (fs.ErrPermission sentinel, else the raw EACCES errno).
func (e *FileSystemException) IsPermission() bool {
	if errors.Is(e.Err, fs.ErrPermission) {
		return true
	}
	if errors.Is(e.Err, syscall.EACCES) {
		return true
	}
	return false
}

// IsExist reports whether the error indicates a file already exists
// (fs.ErrExist sentinel, else the raw EEXIST errno).
func (e *FileSystemException) IsExist() bool {
	if errors.Is(e.Err, fs.ErrExist) {
		return true
	}
	if errors.Is(e.Err, syscall.EEXIST) {
		return true
	}
	return false
}
