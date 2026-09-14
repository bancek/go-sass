// Copyright (c) 2013, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// Package sourcemap implements version 3 source map generation: the VLQ
// encoder here plus the JSON builder in sourcemap.go.
//
// The VLQ layer is ported from the external package:source_maps encoder:
// signed values fold their sign into the least-significant bit, then emit
// as 5-bit chunks with a continuation bit, least-significant chunk first,
// each chunk mapped through the Base64 alphabet.
package sourcemap

// dart-source: (external) package:source_maps/source_maps.dart (src/vlq.dart: encodeVlq/decodeVlq)

import "fmt"

// The Base64 alphabet that each 5-bit VLQ chunk maps through.
//
// Matches Dart: base64Digits in package:source_maps src/vlq.dart
const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

const (
	// Number of payload bits per VLQ chunk.
	//
	// Matches Dart: vlqBaseShift
	vlqBaseShift = 5
	// Mask for the payload bits of a VLQ chunk.
	//
	// Matches Dart: vlqBaseMask
	vlqBaseMask = (1 << 5) - 1
	// Flag marking that another VLQ chunk follows.
	//
	// Matches Dart: vlqContinuationBit
	vlqContinuationBit = 1 << 5
)

// Bounds of the signed 32-bit range the VLQ codec accepts. Values outside
// it are rejected on encode and on decode.
//
// Matches Dart: maxInt32/minInt32 in package:source_maps src/vlq.dart
const maxInt32 = 1<<31 - 1
const minInt32 = -1 << 31

// Reverse lookup from Base64 character to its 6-bit value, -1 for
// characters outside the alphabet.
//
// Matches Dart: _digits in package:source_maps src/vlq.dart
var base64Decode [256]int8

func init() {
	for i := range base64Decode {
		base64Decode[i] = -1
	}
	for i := range len(base64Chars) {
		base64Decode[base64Chars[i]] = int8(i)
	}
}

// encodeVLQ encodes value as a Base64 VLQ string: the sign folds into the
// least-significant bit (so 1 encodes as 2 and -1 as 3), then 5-bit chunks
// emit least-significant first with the continuation bit set on every
// chunk but the last. Values outside the signed 32-bit range are rejected.
//
// Matches Dart: encodeVlq in package:source_maps src/vlq.dart
func encodeVLQ(value int) (string, error) {
	if value < minInt32 || value > maxInt32 {
		return "", fmt.Errorf("sourcemap: value %d out of 32-bit range", value)
	}

	signBit := 0
	if value < 0 {
		signBit = 1
		value = -value
	}
	value = (value << 1) | signBit

	buf := make([]byte, 0, 7)
	for {
		digit := value & vlqBaseMask
		value >>= vlqBaseShift
		if value > 0 {
			digit |= vlqContinuationBit
		}
		buf = append(buf, base64Chars[digit])
		if value == 0 {
			break
		}
	}
	return string(buf), nil
}

// decodeVLQ decodes a single VLQ-encoded value from s starting at pos,
// returning the decoded value and the position just past it. Chunks
// accumulate until a chunk without the continuation bit; the
// least-significant bit then unfolds back into the sign. Incomplete
// values, characters outside the Base64 alphabet, and results outside the
// signed 32-bit range are errors.
//
// Matches Dart: decodeVlq in package:source_maps src/vlq.dart
func decodeVLQ(s string, pos int) (int, int, error) {
	result := 0
	shift := 0
	for {
		if pos >= len(s) {
			return 0, pos, fmt.Errorf("incomplete VLQ value")
		}
		c := s[pos]
		digit := int(base64Decode[c])
		if digit < 0 {
			return 0, pos, fmt.Errorf("invalid character in VLQ encoding: %c", c)
		}
		pos++
		stop := (digit & vlqContinuationBit) == 0
		digit &= vlqBaseMask
		result += digit << shift
		shift += vlqBaseShift
		if stop {
			break
		}
	}

	negate := (result & 1) == 1
	result >>= 1
	if negate {
		result = -result
	}

	if result < minInt32 || result > maxInt32 {
		return 0, pos, fmt.Errorf("decoded value out of 32-bit range: %d", result)
	}

	return result, pos, nil
}
