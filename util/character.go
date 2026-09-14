// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package util

// dart-source: lib/src/util/character.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// The highest code point allowed in CSS.
//
// Matches Dart: maxAllowedCharacter
const MaxAllowedCharacter = 0x10FFFF

// The difference between upper- and lowercase ASCII letters. OR-ing an
// uppercase ASCII letter with this yields its lowercase equivalent.
//
// Matches Dart: _asciiCaseBit
const asciiCaseBit = 0x20

// IsAlphanumeric returns whether ch is a letter or a number.
//
// Matches Dart: CharacterExtension.isAlphanumeric
func IsAlphanumeric(ch int) bool {
	return IsAlphabetic(ch) || IsDigit(ch)
}

// IsWhitespace returns whether ch is an ASCII whitespace character.
//
// Matches Dart: NullableCharacterExtension.isWhitespace
func IsWhitespace(ch int) bool {
	return IsSpaceOrTab(ch) || IsNewline(ch)
}

// IsNewline returns whether ch is an ASCII newline: line feed, carriage
// return, or form feed.
//
// Matches Dart: NullableCharacterExtension.isNewline
func IsNewline(ch int) bool {
	return ch == '\n' || ch == '\r' || ch == '\f'
}

// IsSpaceOrTab returns whether ch is a space or a tab character.
//
// Matches Dart: NullableCharacterExtension.isSpaceOrTab
func IsSpaceOrTab(ch int) bool {
	return ch == ' ' || ch == '\t'
}

// IsPrivate returns whether identifier is module-private, that is whether
// its first code unit is "-" or "_".
//
// Assumes identifier is a valid, non-empty Sass identifier.
//
// Matches Dart: isPrivate
func IsPrivate(identifier string) bool {
	first := identifier[0]
	return first == '-' || first == '_'
}

// AsHex returns the value of character as a hex digit.
//
// Assumes character is a hex digit.
//
// Matches Dart: asHex
func AsHex(character int) int {
	switch {
	case character <= '9':
		return character - '0'
	case character <= 'F':
		return 10 + character - 'A'
	default:
		return 10 + character - 'a'
	}
}

// AsDecimal returns the value of character as a decimal digit.
//
// Assumes character is a decimal digit.
//
// Matches Dart: asDecimal
func AsDecimal(character int) int {
	return character - '0'
}

// DecimalCharFor returns the decimal digit for number.
//
// Assumes number is less than 10.
//
// Matches Dart: decimalCharFor
func DecimalCharFor(number int) int {
	return '0' + number
}

// HexCharFor returns the hex digit character for a number 0-15.
//
// Matches Dart: hexCharFor
func HexCharFor(number int) int {
	if number >= 0x10 {
		panic("number must be less than 16")
	}
	if number < 0xA {
		return '0' + number
	}
	return 'a' - 0xA + number
}

// Opposite returns the right-hand version of character, assuming it is a
// left-hand brace-like character: "(", "{", or "[".
//
// Matches Dart: opposite
func Opposite(character int) (int, error) {
	switch character {
	case '(':
		return ')', nil
	case '{':
		return '}', nil
	case '[':
		return ']', nil
	default:
		return 0, &sasscommon.ArgumentError{Message: fmt.Sprintf("%q isn't a brace-like character", rune(character))}
	}
}

// ToUpperCase returns character converted to upper case if it is an ASCII
// lowercase letter, and unchanged otherwise.
//
// Matches Dart: toUpperCase
func ToUpperCase(character int) int {
	if character >= 'a' && character <= 'z' {
		return character &^ asciiCaseBit
	}
	return character
}

// ToLowerCase returns character converted to lower case if it is an ASCII
// uppercase letter, and unchanged otherwise.
//
// Matches Dart: toLowerCase
func ToLowerCase(character int) int {
	if character >= 'A' && character <= 'Z' {
		return character | asciiCaseBit
	}
	return character
}

// CharacterEqualsIgnoreCase returns whether character1 and character2 are
// the same modulo ASCII case. A single differing bit equal to the case bit
// proves nothing by itself; one side must additionally be an ASCII letter
// for the pair to count as equivalent.
//
// Matches Dart: characterEqualsIgnoreCase
func CharacterEqualsIgnoreCase(character1 int, character2 int) bool {
	if character1 == character2 {
		return true
	}
	if character1^character2 != asciiCaseBit {
		return false
	}
	upperCase1 := character1 &^ asciiCaseBit
	return upperCase1 >= 'A' && upperCase1 <= 'Z'
}

// EqualsLetterIgnoreCase behaves like CharacterEqualsIgnoreCase but is
// optimized for a letter already known to be a lowercase ASCII letter.
//
// Matches Dart: equalsLetterIgnoreCase
func EqualsLetterIgnoreCase(letter int, actual int) bool {
	return (actual | asciiCaseBit) == letter
}

// IsHex returns whether ch is a hexadecimal digit.
//
// Matches Dart: CharacterExtension.isHex (nullable extension returns false
// for absent code units; Go callers pass -1 through the same predicate)
func IsHex(ch int) bool {
	return IsDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// IsDigit returns whether ch is an ASCII number.
//
// Matches Dart: CharacterExtension.isDigit
func IsDigit(ch int) bool {
	return ch >= '0' && ch <= '9'
}

// IsNameStart returns whether ch is legal as the start of a Sass
// identifier: "_", a letter, or any non-ASCII character.
//
// Matches Dart: CharacterExtension.isNameStart
func IsNameStart(ch int) bool {
	return ch == '_' || IsAlphabetic(ch) || ch >= 0x0080
}

// IsName returns whether ch is legal in the body of a Sass identifier: a
// name-start character, a digit, or "-".
//
// Matches Dart: CharacterExtension.isName
func IsName(ch int) bool {
	return IsNameStart(ch) || IsDigit(ch) || ch == '-'
}

// IsAlphabetic returns whether ch is an ASCII letter.
//
// Matches Dart: CharacterExtension.isAlphabetic
func IsAlphabetic(ch int) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
