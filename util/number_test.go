package util

import (
	"math"
	"strings"
	"testing"
)

func TestFuzzyEquals(t *testing.T) {
	if !FuzzyEquals(1.0, 1.0) {
		t.Error("1.0 and 1.0 should be fuzzy equal")
	}
	if !FuzzyEquals(1.0, 1.0+1e-12) {
		t.Error("1.0 and 1.0+1e-12 should be fuzzy equal (within tolerance)")
	}
	if FuzzyEquals(1.0, 1.0+1e-10) {
		t.Error("1.0 and 1.0+1e-10 should not be fuzzy equal")
	}
}

func TestFuzzyEqualsNullable(t *testing.T) {
	if !FuzzyEqualsNullable(1.0, 1.0, false, false) {
		t.Error("both present, same value")
	}
	if !FuzzyEqualsNullable(0, 0, true, true) {
		t.Error("both missing should be equal")
	}
	if FuzzyEqualsNullable(1.0, 1.0, true, false) {
		t.Error("one missing should not be equal")
	}
}

func TestFuzzyHashCode(t *testing.T) {
	// Same value → same hash
	h1 := FuzzyHashCode(1.0)
	h2 := FuzzyHashCode(1.0 + 1e-12)
	if h1 != h2 {
		t.Errorf("fuzzy equal values should have same hash: %d vs %d", h1, h2)
	}
	// Inf/NaN → 0
	if got := FuzzyHashCode(math.Inf(1)); got != 0 {
		t.Errorf("FuzzyHashCode(Inf) = %d, want 0", got)
	}
}

func TestFuzzyLessThan(t *testing.T) {
	if !FuzzyLessThan(1.0, 2.0) {
		t.Error("1.0 < 2.0")
	}
	if FuzzyLessThan(2.0, 1.0) {
		t.Error("2.0 not < 1.0")
	}
	if FuzzyLessThan(1.0, 1.0+1e-12) {
		t.Error("fuzzy equal should not be less than")
	}
}

func TestFuzzyGreaterThan(t *testing.T) {
	if !FuzzyGreaterThan(2.0, 1.0) {
		t.Error("2.0 > 1.0")
	}
	if FuzzyGreaterThan(1.0, 2.0) {
		t.Error("1.0 not > 2.0")
	}
}

func TestFuzzyInRange(t *testing.T) {
	if !FuzzyInRange(5.0, 0.0, 10.0) {
		t.Error("5.0 should be in [0, 10]")
	}
	if !FuzzyInRange(0.0, 0.0, 10.0) {
		t.Error("0.0 should be in [0, 10]")
	}
	if FuzzyInRange(11.0, 0.0, 10.0) {
		t.Error("11.0 should not be in [0, 10]")
	}
}

func TestFuzzyIsInt(t *testing.T) {
	if !FuzzyIsInt(5.0) {
		t.Error("5.0 should be fuzzy int")
	}
	if !FuzzyIsInt(5.0 + 1e-12) {
		t.Error("5.0+1e-12 should be fuzzy int")
	}
	if FuzzyIsInt(5.5) {
		t.Error("5.5 should not be fuzzy int")
	}
	if FuzzyIsInt(math.Inf(1)) {
		t.Error("Inf should not be fuzzy int")
	}
}

func TestFuzzyAsInt(t *testing.T) {
	val, ok := FuzzyAsInt(5.0)
	if !ok || val != 5 {
		t.Errorf("FuzzyAsInt(5.0) = (%d, %v), want (5, true)", val, ok)
	}
	val, ok = FuzzyAsInt(5.0 + 1e-12)
	if !ok || val != 5 {
		t.Errorf("FuzzyAsInt(5.0+1e-12) = (%d, %v), want (5, true)", val, ok)
	}
	_, ok = FuzzyAsInt(5.5)
	if ok {
		t.Error("FuzzyAsInt(5.5) should not be ok")
	}
	_, ok = FuzzyAsInt(math.Inf(1))
	if ok {
		t.Error("FuzzyAsInt(Inf) should not be ok")
	}
}

func TestAsIntForSerialize(t *testing.T) {
	// Dart `_asInt` (#2800): exact integers in both modes, fuzzy integers
	// only in inspect mode.
	if v, ok := AsIntForSerialize(2.0, false); !ok || v != 2 {
		t.Errorf("AsIntForSerialize(2.0, false) = (%d, %v), want (2, true)", v, ok)
	}
	if v, ok := AsIntForSerialize(2.0, true); !ok || v != 2 {
		t.Errorf("AsIntForSerialize(2.0, true) = (%d, %v), want (2, true)", v, ok)
	}
	if _, ok := AsIntForSerialize(2.000000000001, false); ok {
		t.Error("AsIntForSerialize(2.000000000001, false) should not be ok")
	}
	if v, ok := AsIntForSerialize(2.000000000001, true); !ok || v != 2 {
		t.Errorf("AsIntForSerialize(2.000000000001, true) = (%d, %v), want (2, true)", v, ok)
	}
	if _, ok := AsIntForSerialize(2.5, false); ok {
		t.Error("AsIntForSerialize(2.5, false) should not be ok")
	}
	if _, ok := AsIntForSerialize(2.5, true); ok {
		t.Error("AsIntForSerialize(2.5, true) should not be ok")
	}
	if _, ok := AsIntForSerialize(math.NaN(), false); ok {
		t.Error("AsIntForSerialize(NaN, false) should not be ok")
	}
	if _, ok := AsIntForSerialize(math.Inf(1), true); ok {
		t.Error("AsIntForSerialize(Inf, true) should not be ok")
	}
}

func TestFuzzyRound(t *testing.T) {
	tests := []struct {
		n    float64
		want int64
	}{
		{1.0, 1},
		{1.4, 1},
		{1.6, 2},
		{-1.4, -1},
		{-1.6, -2},
	}
	for _, tt := range tests {
		got, err := FuzzyRound(tt.n)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("FuzzyRound(%v) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestFuzzyRoundRound0_5(t *testing.T) {
	got, _ := FuzzyRound(0.5)
	if got != 1 {
		t.Errorf("FuzzyRound(0.5) = %d, want 1", got)
	}
	got, _ = FuzzyRound(1.5)
	if got != 2 {
		t.Errorf("FuzzyRound(1.5) = %d, want 2", got)
	}
}

func TestModuloLikeSass(t *testing.T) {
	// Positive divisor
	if got := ModuloLikeSass(7.0, 3.0); got != 1.0 {
		t.Errorf("ModuloLikeSass(7,3) = %v, want 1", got)
	}
	if got := ModuloLikeSass(-7.0, 3.0); got != 2.0 {
		t.Errorf("ModuloLikeSass(-7,3) = %v, want 2", got)
	}
	// Negative divisor
	if got := ModuloLikeSass(7.0, -3.0); got != -2.0 {
		t.Errorf("ModuloLikeSass(7,-3) = %v, want -2", got)
	}
	if got := ModuloLikeSass(-7.0, -3.0); got != -1.0 {
		t.Errorf("ModuloLikeSass(-7,-3) = %v, want -1", got)
	}
}

func TestModuloLikeSassZero(t *testing.T) {
	// 0 divisor
	if !math.IsNaN(ModuloLikeSass(1.0, 0.0)) {
		t.Error("ModuloLikeSass(x, 0) should be NaN")
	}
}

func TestSignIncludingZero(t *testing.T) {
	if SignIncludingZero(1.0) != 1.0 {
		t.Error("SignIncludingZero(1.0) should be 1.0")
	}
	if SignIncludingZero(-1.0) != -1.0 {
		t.Error("SignIncludingZero(-1.0) should be -1.0")
	}
	// +0 and -0
	pz := math.Copysign(0, 1)
	nz := math.Copysign(0, -1)
	if SignIncludingZero(pz) != 1.0 {
		t.Error("SignIncludingZero(+0) should be 1.0")
	}
	if SignIncludingZero(nz) != -1.0 {
		t.Error("SignIncludingZero(-0) should be -1.0")
	}
}

func TestClampLikeCss(t *testing.T) {
	if got := ClampLikeCss(5.0, 0.0, 10.0); got != 5.0 {
		t.Errorf("ClampLikeCss(5, 0, 10) = %v, want 5", got)
	}
	if got := ClampLikeCss(-1.0, 0.0, 10.0); got != 0.0 {
		t.Errorf("ClampLikeCss(-1, 0, 10) = %v, want 0", got)
	}
	if got := ClampLikeCss(15.0, 0.0, 10.0); got != 10.0 {
		t.Errorf("ClampLikeCss(15, 0, 10) = %v, want 10", got)
	}
	// NaN → lower bound
	if got := ClampLikeCss(math.NaN(), 0.0, 10.0); got != 0.0 {
		t.Errorf("ClampLikeCss(NaN, 0, 10) = %v, want 0", got)
	}
	// Dart's num.clamp yields +0.0 for -0.0 (#2840).
	negZero := math.Copysign(0, -1)
	if got := ClampLikeCss(negZero, 0.0, 1.0); math.Float64bits(got) != math.Float64bits(0) {
		t.Errorf("ClampLikeCss(-0.0, 0, 1) = %v, want +0.0", got)
	}
}

func TestIsNegativeZero(t *testing.T) {
	negZero := math.Copysign(0, -1)
	if !IsNegativeZero(negZero) {
		t.Error("IsNegativeZero(-0.0) = false, want true")
	}
	if IsNegativeZero(0.0) {
		t.Error("IsNegativeZero(0.0) = true, want false")
	}
	if IsNegativeZero(1.0) || IsNegativeZero(-1.0) || IsNegativeZero(math.NaN()) {
		t.Error("IsNegativeZero should be false for non-zero values")
	}
}

func TestNormalizeLinear(t *testing.T) {
	negZero := math.Copysign(0, -1)
	if got := NormalizeLinear(math.NaN()); math.Float64bits(got) != math.Float64bits(0) {
		t.Error("NormalizeLinear(NaN) should be +0")
	}
	if got := NormalizeLinear(negZero); math.Float64bits(got) != math.Float64bits(0) {
		t.Error("NormalizeLinear(-0.0) should be +0")
	}
	if got := NormalizeLinear(1.5); got != 1.5 {
		t.Errorf("NormalizeLinear(1.5) = %v, want 1.5", got)
	}
	if got := NormalizeLinear(math.Inf(1)); !math.IsInf(got, 1) {
		t.Errorf("NormalizeLinear(Inf) = %v, want +Inf", got)
	}
}

func TestWriteNumberNegativeZero(t *testing.T) {
	negZero := math.Copysign(0, -1)
	if got := WriteNumberToString(negZero); got != "-0" {
		t.Errorf("WriteNumberToString(-0.0) = %q, want %q", got, "-0")
	}
	var buf strings.Builder
	WriteNumberTo(&buf, negZero, true, true)
	if got := buf.String(); got != "-0" {
		t.Errorf("WriteNumberTo(-0.0, inspect, compressed) = %q, want %q", got, "-0")
	}
}
