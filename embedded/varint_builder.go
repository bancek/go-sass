// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/util/varint_builder.dart

import (
	"fmt"
)

// VarintBuilder builds up unsigned varints byte-by-byte for the
// length-delimited packet framing.
//
// One builder parses one varint; Reset recycles it for the next. Callers
// feed raw stream bytes in order and watch the done flag in Add's second
// return.
//
// Matches Dart: class VarintBuilder in util/varint_builder.dart.
type VarintBuilder struct {
	// maxLength is the maximum length in bits of the varint being parsed.
	// It corresponds to Dart's _maxLength field.
	maxLength int
	// name identifies the value being parsed in error messages. It
	// corresponds to Dart's nullable _name field (empty means unnamed).
	name string
	// value accumulates the parsed payload bits. It corresponds to Dart's
	// _value field.
	value int
	// bits counts how many payload bits have been consumed so far. It
	// corresponds to Dart's _bits field.
	bits int
	// done reports whether the varint has completed (or overrun). It
	// corresponds to Dart's _done field.
	done bool
}

// NewVarintBuilder creates a builder accepting at most maxLength bits, using
// name in over-length error messages when non-empty.
//
// Dart's name is an optional positional parameter; Go takes it as a plain
// string with "" meaning unnamed.
//
// Matches Dart: VarintBuilder constructor in util/varint_builder.dart.
func NewVarintBuilder(maxLength int, name string) *VarintBuilder {
	return &VarintBuilder{maxLength: maxLength, name: name}
}

// Add parses b as a continuation of the varint.
//
// Each byte contributes its 7 low bits (masked with 0x7f) while the high bit
// marks continuation. When a byte without the high bit arrives the varint is
// complete and Add returns its value with done true; otherwise it returns
// done false to request more bytes. An over-long varint or a call after
// completion returns an error — Dart throws a ProtocolError for the former
// and a StateError for the latter, but Go unifies both as errors and leaves
// it to packet/frame callers to wrap them as PARSE protocol errors. The
// trailing-bytes check matters because maxLength is usually not a multiple
// of 7, so the final byte must not smuggle bits past the limit.
//
// Matches Dart: VarintBuilder.add in util/varint_builder.dart.
func (v *VarintBuilder) Add(b byte) (int, bool, error) {
	if v.done {
		return 0, false, fmt.Errorf("VarintBuilder.Add() has already returned a value.")
	}

	// Varints encode data in the 7 lower bits of each byte.
	v.value += int(b&0x7f) << v.bits
	v.bits += 7

	// If the byte has its high bit set, more bytes need to be consumed.
	if b > 0x7f {
		if v.bits >= v.maxLength {
			v.done = true
			return 0, true, fmt.Errorf("varint %swas longer than %d bits", v.nameMsg(), v.maxLength)
		}
		return 0, false, nil
	}

	v.done = true
	if v.bits > v.maxLength && v.value >= 1<<v.maxLength {
		return v.value, true, fmt.Errorf("varint %swas longer than %d bits", v.nameMsg(), v.maxLength)
	}
	return v.value, true, nil
}

// Reset returns the builder to its initial state so it can parse another
// varint, clearing the accumulated value, bit count, and completion flag.
//
// Matches Dart: VarintBuilder.reset in util/varint_builder.dart.
func (v *VarintBuilder) Reset() {
	v.value = 0
	v.bits = 0
	v.done = false
}

// nameMsg renders the builder's name with a trailing space for error
// messages, or an empty string when unnamed. Go-local split of Dart's inline
// interpolation in _tooLong.
func (v *VarintBuilder) nameMsg() string {
	if v.name == "" {
		return ""
	}
	return v.name + " "
}
