// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/util/fuzzy_equality.dart

// FuzzyEquality is an equality implementation for float64 that compares
// with FuzzyEquals and hashes with FuzzyHashCode, so maps keyed by numbers
// honor Sass's fuzzy-equality semantics.
//
// Matches Dart: FuzzyEquality (util/fuzzy_equality.dart, implementing
// Equality<double>)
type FuzzyEquality struct{}

// Equals returns whether a and b are FuzzyEquals.
//
// Matches Dart: FuzzyEquality.equals
func (FuzzyEquality) Equals(a, b float64) bool { return FuzzyEquals(a, b) }

// Hash returns the FuzzyHashCode of n, consistent with Equals.
//
// Matches Dart: FuzzyEquality.hash
func (FuzzyEquality) Hash(n float64) int { return FuzzyHashCode(n) }

// IsValidKey returns whether o is a float64 and so a valid key for this
// equality.
//
// Matches Dart: FuzzyEquality.isValidKey
func (FuzzyEquality) IsValidKey(o any) bool { _, ok := o.(float64); return ok }
