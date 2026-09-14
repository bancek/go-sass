// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package util

import (
	"testing"
)

func TestWriteNumberToString(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{v: 0, want: "0"},
		{v: 1, want: "1"},
		{v: 255, want: "255"},
		{v: 0.5, want: "0.5"},
		{v: 100.0, want: "100"},
		{v: -42, want: "-42"},
		{v: -0.5, want: "-0.5"},
		{v: 0.1, want: "0.1"},
		{v: 1.5, want: "1.5"},
		{v: 0.123456789, want: "0.123456789"},
		{v: 0.1234567890, want: "0.123456789"},
		{v: 0.12345678919, want: "0.1234567892"},
		{v: 1.0, want: "1"},
		{v: 0.0, want: "0"},
	}
	for _, tt := range tests {
		got := WriteNumberToString(tt.v)
		if got != tt.want {
			t.Errorf("WriteNumberToString(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestWriteNumberToStringIntegerFuzzy(t *testing.T) {
	tests := []float64{
		0.000000000001,
		-0.000000000001,
		255.000000000001,
		1.000000000001,
	}
	for _, v := range tests {
		got := WriteNumberToString(v)
		// Fuzzy integers within 1e-11 should round to the exact integer.
		i := int(v)
		if got != WriteNumberToString(float64(i)) {
			t.Errorf("WriteNumberToString(%v) = %q, want same as integer %v", v, got, i)
		}
	}
}

func TestWriteNumberToStringExponent(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{v: 0.000001, want: "0.000001"},
		{v: 1e-7, want: "0.0000001"},
		{v: 1000000, want: "1000000"},
		{v: 1.2345678912345e-6, want: "0.0000012346"},
		// Precision truncates to 10 decimal places
		{v: 0.123456789012345, want: "0.123456789"},
		{v: 0.12345678906789, want: "0.1234567891"},
	}
	for _, tt := range tests {
		got := WriteNumberToString(tt.v)
		if got != tt.want {
			t.Errorf("WriteNumberToString(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
