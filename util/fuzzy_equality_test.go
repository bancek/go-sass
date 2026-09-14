package util

import "testing"

func TestFuzzyEqualityEquals(t *testing.T) {
	var fe FuzzyEquality
	if !fe.Equals(1.0, 1.0) {
		t.Error("1.0 should equal 1.0")
	}
	if !fe.Equals(1.0, 1.0+1e-12) {
		t.Error("1.0 should equal 1.0+1e-12 (fuzzy)")
	}
	if fe.Equals(1.0, 1.1) {
		t.Error("1.0 should not equal 1.1")
	}
}

func TestFuzzyEqualityHash(t *testing.T) {
	var fe FuzzyEquality
	h1 := fe.Hash(1.0)
	h2 := fe.Hash(1.0 + 1e-12)
	if h1 != h2 {
		t.Errorf("fuzzy equal values should have same hash: %d vs %d", h1, h2)
	}
}

func TestFuzzyEqualityIsValidKey(t *testing.T) {
	var fe FuzzyEquality
	if !fe.IsValidKey(1.0) {
		t.Error("float64 should be valid key")
	}
	if !fe.IsValidKey(float64(0)) {
		t.Error("0.0 should be valid key")
	}
	// Non-float64 should return false
	if fe.IsValidKey("not a float") {
		t.Error("string should not be valid key")
	}
	if fe.IsValidKey(42) {
		t.Error("int should not be valid key")
	}
}
