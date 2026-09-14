// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/util/character.dart
//
// The surrogate helpers below live on Dart's CharacterExtension as getters
// used in pattern matches; Go keeps them as free functions on code-unit ints
// because Go strings are UTF-8 and surrogates only arise when bridging
// UTF-16 data (for example identifier escaping).

// IsHighSurrogate returns whether ch is the start of a UTF-16 surrogate
// pair, that is whether it matches 0b110110XXXXXXXXXX.
//
// Matches Dart: CharacterExtension.isHighSurrogate
func IsHighSurrogate(ch int) bool {
	return ch>>10 == 0x36
}

// IsLowSurrogate returns whether ch is the end of a UTF-16 surrogate
// pair, that is whether it matches 0b110111XXXXXXXXXX.
//
// Matches Dart: CharacterExtension.isLowSurrogate
func IsLowSurrogate(ch int) bool {
	return ch>>10 == 0x37
}

// IsPrivateUseBMP returns whether ch is a Unicode private-use code point
// in the Basic Multilingual Plane (U+E000 to U+F8FF).
//
// Matches Dart: CharacterExtension.isPrivateUseBMP
func IsPrivateUseBMP(ch int) bool {
	return ch >= 0xE000 && ch <= 0xF8FF
}

// IsPrivateUseHighSurrogate returns whether ch is the high surrogate for
// a code point in a Unicode private-use supplementary plane (high
// surrogates 0xDB80 to 0xDBFF, exactly the range 0b110110111XXXXXXX).
//
// Matches Dart: CharacterExtension.isPrivateUseHighSurrogate
func IsPrivateUseHighSurrogate(ch int) bool {
	return ch>>7 == 0x1B7
}

// CombineSurrogates combines a UTF-16 high and low surrogate pair into a
// single code point. Only the low ten bits of each surrogate carry payload
// (0x3FF masks out the six tag bits), shifted above the 0x10000 base.
//
// Matches Dart: combineSurrogates
func CombineSurrogates(highSurrogate, lowSurrogate int) int {
	return 0x10000 + ((highSurrogate & 0x3FF) << 10) + (lowSurrogate & 0x3FF)
}
