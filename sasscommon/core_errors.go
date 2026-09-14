// Copyright (c) 2012, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (standard) dart:core

import "fmt"

// ArgumentError reports an invalid argument passed to a function. Name
// optionally identifies the offending parameter, matching Dart's
// ArgumentError.name.
//
// It ports Dart's ArgumentError/ArgumentError.value from dart:core, raised
// throughout the span algebra for cross-file or out-of-range operations.
//
// Matches Dart: ArgumentError / ArgumentError.value (dart:core)
type ArgumentError struct {
	// Message describes why the argument is invalid.
	Message string
	// Name optionally names the offending argument (Dart's name field).
	Name string // argument name (optional, matches Dart ArgumentError.name)
}

// Error renders "Invalid argument (name): message" when the argument is
// named, or "Invalid argument(s): message" otherwise.
func (e *ArgumentError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("Invalid argument (%s): %s", e.Name, e.Message)
	}
	return fmt.Sprintf("Invalid argument(s): %s", e.Message)
}

// StateError reports an operation that is invalid for the object's current
// state (for example consuming a scanner that already failed).
//
// It ports Dart's StateError from dart:core.
//
// Matches Dart: StateError (dart:core)
type StateError struct {
	// Message describes the invalid-state operation.
	Message string
}

// Error returns the message unchanged; state errors carry no extra fields.
func (e *StateError) Error() string { return e.Message }

// RangeError reports a value outside its valid range: a negative offset, an
// offset past end of file, or an invalid subspan window. Name optionally
// identifies the offending parameter, matching Dart's RangeError.name.
//
// It ports Dart's RangeError/RangeError.range from dart:core.
//
// Matches Dart: RangeError / RangeError.range (dart:core)
type RangeError struct {
	// Message describes the out-of-range value.
	Message string
	Name    string // parameter name (optional)
}

// Error returns the message unchanged; range details live in the message.
func (e *RangeError) Error() string { return e.Message }

// UnsupportedError reports an operation the value does not support.
//
// It ports Dart's UnsupportedError from dart:core.
//
// Matches Dart: UnsupportedError (dart:core)
type UnsupportedError struct {
	// Message describes the unsupported operation.
	Message string
}

// Error returns the message unchanged.
func (e *UnsupportedError) Error() string { return e.Message }
