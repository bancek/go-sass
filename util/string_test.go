package util

import "testing"

func TestIsHighSurrogate(t *testing.T) {
	// High surrogates are in range U+D800..U+DBFF
	if !IsHighSurrogate(0xD800) {
		t.Error("0xD800 should be a high surrogate")
	}
	if !IsHighSurrogate(0xDBFF) {
		t.Error("0xDBFF should be a high surrogate")
	}
	if IsHighSurrogate(0xD7FF) {
		t.Error("0xD7FF should not be a high surrogate")
	}
	if IsHighSurrogate(0xDC00) {
		t.Error("0xDC00 should not be a high surrogate")
	}
}

func TestIsLowSurrogate(t *testing.T) {
	// Low surrogates are in range U+DC00..U+DFFF
	if !IsLowSurrogate(0xDC00) {
		t.Error("0xDC00 should be a low surrogate")
	}
	if !IsLowSurrogate(0xDFFF) {
		t.Error("0xDFFF should be a low surrogate")
	}
	if IsLowSurrogate(0xDBFF) {
		t.Error("0xDBFF should not be a low surrogate")
	}
	if IsLowSurrogate(0xE000) {
		t.Error("0xE000 should not be a low surrogate")
	}
}

func TestIsPrivateUseBMP(t *testing.T) {
	if !IsPrivateUseBMP(0xE000) {
		t.Error("0xE000 should be private use BMP")
	}
	if !IsPrivateUseBMP(0xF8FF) {
		t.Error("0xF8FF should be private use BMP")
	}
	if IsPrivateUseBMP(0xDFFF) {
		t.Error("0xDFFF should not be private use BMP")
	}
	if IsPrivateUseBMP(0xF900) {
		t.Error("0xF900 should not be private use BMP")
	}
}

func TestIsPrivateUseHighSurrogate(t *testing.T) {
	if !IsPrivateUseHighSurrogate(0xDB80) {
		t.Error("0xDB80 should be private use high surrogate")
	}
	if !IsPrivateUseHighSurrogate(0xDBFF) {
		t.Error("0xDBFF should be private use high surrogate")
	}
	if IsPrivateUseHighSurrogate(0xD800) {
		t.Error("0xD800 should not be private use high surrogate")
	}
}

func TestCombineSurrogates(t *testing.T) {
	// U+10000 = D800 + DC00 → 65536
	got := CombineSurrogates(0xD800, 0xDC00)
	if got != 0x10000 {
		t.Errorf("CombineSurrogates(D800, DC00) = %x, want 10000", got)
	}
	// U+10FFFF = DBFF + DFFF → 1114111
	got = CombineSurrogates(0xDBFF, 0xDFFF)
	if got != 0x10FFFF {
		t.Errorf("CombineSurrogates(DBFF, DFFF) = %x, want 10FFFF", got)
	}
}
