// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/utils.dart

// IsPrivateMember returns whether name is a private Sass member: one
// starting with "-" or "_". This is the negation of IsPublic, provided
// for call sites that test privateness directly.
//
// Matches Dart: !isPublic (utils.dart)
func IsPrivateMember(name string) bool {
	return !IsPublic(name)
}

// IsPublic returns whether name is a public member name: one starting
// with neither "-" nor "_".
//
// Assumes that name is a valid Sass identifier.
//
// Matches Dart: isPublic (utils.dart)
func IsPublic(name string) bool {
	start := name[0]
	return start != '-' && start != '_'
}

// Pluralize returns name if number is 1, or the plural of name otherwise.
//
// By default the plural just appends "s"; if plural is passed, it is used
// instead.
//
// Matches Dart: pluralize (utils.dart)
func Pluralize(name string, number int, plural *string) string {
	if number == 1 {
		return name
	}
	if plural != nil {
		return *plural
	}
	return name + "s"
}
