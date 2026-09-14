package value

import "testing"

func TestHashCombine(t *testing.T) {
	if hashCombine(0, 0) != hashCombine(0, 0) {
		t.Error("hashCombine must be deterministic")
	}
	h1 := hashCombine(1, 2)
	h2 := hashCombine(2, 1)
	if h1 == h2 {
		t.Error("hashCombine should produce different results for different inputs")
	}
	h3 := hashCombine(1, 2)
	if h1 != h3 {
		t.Error("hashCombine must be deterministic")
	}
}

func TestStringHashCode(t *testing.T) {
	if got := stringHashCode(""); got != 0 {
		t.Errorf("stringHashCode(\"\") = %d, want 0", got)
	}
	h := stringHashCode("hello")
	if h == 0 {
		t.Error("stringHashCode(\"hello\") should be non-zero")
	}
	if stringHashCode("hello") != stringHashCode("hello") {
		t.Error("stringHashCode must be deterministic")
	}
	if stringHashCode("hello") == stringHashCode("Hello") {
		t.Error("different strings should produce different hash codes")
	}
}

func TestBoolHashCode(t *testing.T) {
	if got := boolHashCode(true); got != 1231 {
		t.Errorf("boolHashCode(true) = %d, want 1231", got)
	}
	if got := boolHashCode(false); got != 1237 {
		t.Errorf("boolHashCode(false) = %d, want 1237", got)
	}
}

func TestListHash(t *testing.T) {
	if got := listHash(nil); got != 0 {
		t.Errorf("listHash(nil) = %d, want 0", got)
	}
	if got := listHash([]Value{}); got != 0 {
		t.Errorf("listHash([]) = %d, want 0", got)
	}
	a := NewUnitlessNumber(1)
	b := NewUnitlessNumber(2)
	h := listHash([]Value{a, b})
	expected := hashCombine(hashCombine(0, a.HashCode()), b.HashCode())
	if h != expected {
		t.Errorf("listHash([a, b]) = %d, want %d", h, expected)
	}
}

func TestMapHash(t *testing.T) {
	if got := mapHash(nil); got != 0 {
		t.Errorf("mapHash(nil) = %d, want 0", got)
	}
	if got := mapHash([]MapEntry{}); got != 0 {
		t.Errorf("mapHash([]) = %d, want 0", got)
	}
	k := NewUnitlessNumber(1)
	v := NewUnitlessNumber(10)
	h := mapHash([]MapEntry{{k, v}})
	expected := hashCombine(hashCombine(0, k.HashCode()), v.HashCode())
	if h != expected {
		t.Errorf("mapHash([{k,v}]) = %d, want %d", h, expected)
	}
}

func TestHashPtr(t *testing.T) {
	x := 42
	if got := hashPtr[int](nil); got != 0 {
		t.Errorf("hashPtr(nil) = %d, want 0", got)
	}
	h1 := hashPtr[int](&x)
	if h1 == 0 {
		t.Error("hashPtr(&x) should be non-zero")
	}
	if h1 != hashPtr[int](&x) {
		t.Error("hashPtr must be deterministic for the same pointer")
	}
}
