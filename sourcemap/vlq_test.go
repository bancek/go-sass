// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package sourcemap

import (
	"strings"
	"testing"
)

func TestEncodeVLQSimple(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "C"},
		{2, "E"},
		{3, "G"},
		{100, "oG"},
		{0, "A"},
		{-1, "D"},
		{minInt32, "hgggggE"},
	}

	for _, tt := range tests {
		got, err := encodeVLQ(tt.input)
		if err != nil {
			t.Fatalf("encodeVLQ(%d) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("encodeVLQ(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDecodeVLQSimple(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"C", 1},
		{"E", 2},
		{"G", 3},
		{"oG", 100},
		{"A", 0},
		{"D", -1},
		{"hgggggE", minInt32},
	}

	for _, tt := range tests {
		got, pos, err := decodeVLQ(tt.input, 0)
		if err != nil {
			t.Fatalf("decodeVLQ(%q) unexpected error: %v", tt.input, err)
		}
		if pos != len(tt.input) {
			t.Errorf("decodeVLQ(%q) pos = %d, want %d", tt.input, pos, len(tt.input))
		}
		if got != tt.want {
			t.Errorf("decodeVLQ(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestEncodeDecodeRange(t *testing.T) {
	for i := -10000; i < 10000; i++ {
		encoded, err := encodeVLQ(i)
		if err != nil {
			t.Fatalf("encodeVLQ(%d) unexpected error: %v", i, err)
		}
		decoded, pos, err := decodeVLQ(encoded, 0)
		if err != nil {
			t.Fatalf("decodeVLQ(encodeVLQ(%d)) unexpected error: %v", i, err)
		}
		if pos != len(encoded) {
			t.Errorf("decodeVLQ(encodeVLQ(%d)) pos = %d, want %d", i, pos, len(encoded))
		}
		if decoded != i {
			t.Errorf("decodeVLQ(encodeVLQ(%d)) = %d, want %d", i, decoded, i)
		}
	}
}

func TestEncodeDecodeBoundaries(t *testing.T) {
	boundaries := []int{
		maxInt32 - 1,
		minInt32 + 1,
		maxInt32,
		minInt32,
	}

	for _, val := range boundaries {
		encoded, err := encodeVLQ(val)
		if err != nil {
			t.Fatalf("encodeVLQ(%d) unexpected error: %v", val, err)
		}
		decoded, _, err := decodeVLQ(encoded, 0)
		if err != nil {
			t.Fatalf("decodeVLQ(encodeVLQ(%d)) unexpected error: %v", val, err)
		}
		if decoded != val {
			t.Errorf("decodeVLQ(encodeVLQ(%d)) = %d, want %d", val, decoded, val)
		}
	}
}

func TestEncodeOutOfRange(t *testing.T) {
	values := []int{maxInt32 + 1, maxInt32 + 2, minInt32 - 1, minInt32 - 2}
	for _, v := range values {
		_, err := encodeVLQ(v)
		if err == nil {
			t.Errorf("expected error for %d, got nil", v)
		}
	}
}

func TestDecodeOutOfRange(t *testing.T) {
	// These values would be valid VLQ but exceed 32-bit range
	// If we allowed more than 32 bits, these would be the encodings:
	tests := []string{
		"ggggggE", // maxInt32 + 1
		"igggggE", // -(minInt32 - 1)
		"jgggggE",
		"lgggggE",
	}

	for _, tc := range tests {
		_, _, err := decodeVLQ(tc, 0)
		if err == nil {
			t.Errorf("decodeVLQ(%q) expected error", tc)
		}
	}
}

func TestEncodeDecodeSequential(t *testing.T) {
	// Encode multiple values and decode them sequentially
	var buf strings.Builder
	values := []int{4, 0, 1, 4, 0} // "IACIA"
	for _, v := range values {
		enc, err := encodeVLQ(v)
		if err != nil {
			t.Fatalf("encodeVLQ(%d) unexpected error: %v", v, err)
		}
		buf.WriteString(enc)
	}
	if buf.String() != "IACIA" {
		t.Errorf("sequential encode = %q, want %q", buf.String(), "IACIA")
	}

	pos := 0
	for _, want := range values {
		got, newPos, err := decodeVLQ(buf.String(), pos)
		if err != nil {
			t.Fatalf("decodeVLQ at pos %d: %v", pos, err)
		}
		if got != want {
			t.Errorf("decodeVLQ at pos %d = %d, want %d", pos, got, want)
		}
		pos = newPos
	}
}

func TestEncodeDecodeRandom(t *testing.T) {
	values := []int{
		42, -42, 127, -128, 255, 1024, -2048, 65535, 1<<20 - 1, -(1 << 20),
	}
	for _, v := range values {
		enc, err := encodeVLQ(v)
		if err != nil {
			t.Fatalf("roundtrip %d encode: %v", v, err)
		}
		dec, _, err := decodeVLQ(enc, 0)
		if err != nil {
			t.Fatalf("roundtrip %d decode: %v", v, err)
		}
		if dec != v {
			t.Errorf("roundtrip %d: got %d", v, dec)
		}
	}
}
