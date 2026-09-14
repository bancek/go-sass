package sassmath

import (
	"math"
	"testing"
)

func hex(v float64) uint64 { return math.Float64bits(v) }

func TestMatrixMulNoFMA(t *testing.T) {
	// From color-fixes.md: matrixMul uses float64() on every
	// intermediate multiply-add to prevent Go's ARM64 FMA.
	// Golden values from Dart (C libm) — see spec/rust-port.md POC #6.

	m := [9]float64{0.5, 0.3, 0.2, 0.1, 0.4, 0.5, 0.3, 0.3, 0.4}
	v0, v1, v2 := 1.0, 0.5, 0.2

	// WITH float64() cast on every mul-add — matches Dart.
	r := float64(m[0]*v0) + float64(m[1]*v1) + float64(m[2]*v2)
	g := float64(m[3]*v0) + float64(m[4]*v1) + float64(m[5]*v2)
	b := float64(m[6]*v0) + float64(m[7]*v1) + float64(m[8]*v2)

	// Dart golden values:
	// matmul_r: 0x3fe6147ae147ae15
	// matmul_g: 0x3fd999999999999a
	// matmul_b: 0x3fe0f5c28f5c28f6
	if got := hex(r); got != 0x3fe6147ae147ae15 {
		t.Errorf("matmul_r: expected 0x3fe6147ae147ae15, got 0x%016x", got)
	}
	if got := hex(g); got != 0x3fd999999999999a {
		t.Errorf("matmul_g: expected 0x3fd999999999999a, got 0x%016x", got)
	}
	if got := hex(b); got != 0x3fe0f5c28f5c28f6 {
		t.Errorf("matmul_b: expected 0x3fe0f5c28f5c28f6, got 0x%016x", got)
	}
}

func TestMatrixMulWithoutFloat64Cast(t *testing.T) {
	// WITHOUT float64() cast — Go compiler may fuse into FMA on ARM64.
	// This test documents whether FMA is active on the current arch.
	m := [9]float64{0.5, 0.3, 0.2, 0.1, 0.4, 0.5, 0.3, 0.3, 0.4}
	v0, v1, v2 := 1.0, 0.5, 0.2

	r := m[0]*v0 + m[1]*v1 + m[2]*v2
	g := m[3]*v0 + m[4]*v1 + m[5]*v2
	b := m[6]*v0 + m[7]*v1 + m[8]*v2

	t.Logf("matmul without float64() casts: r=0x%016x g=0x%016x b=0x%016x",
		hex(r), hex(g), hex(b))
	t.Logf("Dart golden:                 r=0x3fe6147ae147ae15 g=0x3fd999999999999a b=0x3fe0f5c28f5c28f6")
}

func TestM116F1Minus16(t *testing.T) {
	// From color-fixes.md §FMA notes:
	// Dart: 116*f1 - 16 = 0 (exact). Go with FMA: -2.22e-16.
	f1 := 1.1379310344827585

	// WITH float64() on intermediate — prevents FMA.
	result := float64(116*f1) - 16
	if got := hex(result); got != 0x405cfffffffffffe {
		t.Errorf("116*f1-16: expected 0x405cfffffffffffe, got 0x%016x", got)
	}

	// WITHOUT float64() — may differ on ARM64.
	noCast := 116*f1 - 16
	t.Logf("116*f1-16 without cast: 0x%016x", hex(noCast))
	t.Logf("Dart golden:            0x405cfffffffffffe")
}

func TestTrigGoldenValues(t *testing.T) {
	pi := 3.141592653589793

	// Dart golden values (C libm, same as Rust f64::sin/cos):
	// sin(PI):    0x3ca1a62633145c07
	// cos(PI):    0xbff0000000000000
	// cos(PI/2):  0x3c91a62633145c07
	// atan2(1,0): 0x3ff921fb54442d18
	// atan2(0,-1):0x400921fb54442d18

	t.Run("Sin", func(t *testing.T) {
		got := hex(Sin(pi))
		t.Logf("Sin(PI):  Go=0x%016x  Dart=0x3ca1a62633145c07", got)
	})

	t.Run("Cos", func(t *testing.T) {
		got := hex(Cos(pi))
		t.Logf("Cos(PI):  Go=0x%016x  Dart=0xbff0000000000000", got)
	})

	t.Run("Cos_HalfPi", func(t *testing.T) {
		got := hex(Cos(pi / 2))
		t.Logf("Cos(PI/2): Go=0x%016x  Dart=0x3c91a62633145c07", got)
	})

	t.Run("Atan2_1_0", func(t *testing.T) {
		got := hex(Atan2(1, 0))
		t.Logf("Atan2(1,0): Go=0x%016x  Dart=0x3ff921fb54442d18", got)
	})

	t.Run("Atan2_0_n1", func(t *testing.T) {
		got := hex(Atan2(0, -1))
		t.Logf("Atan2(0,-1): Go=0x%016x  Dart=0x400921fb54442d18", got)
	})
}

func TestCustomFunctionPointers(t *testing.T) {
	called := false
	CustomSqrt = func(x float64) float64 {
		called = true
		return x * x
	}
	defer func() { CustomSqrt = nil }()

	if got := Sqrt(4.0); got != 16.0 {
		t.Errorf("CustomSqrt(4): expected 16, got %v", got)
	}
	if !called {
		t.Error("CustomSqrt was not called")
	}
}

func TestAllFunctionsExist(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(float64) float64
		x    float64
	}{
		{"Sin", Sin, 0},
		{"Cos", Cos, 0},
		{"Tan", Tan, 0},
		{"Atan", Atan, 0},
		{"Asin", Asin, 0},
		{"Acos", Acos, 1},
		{"Log", Log, 1},
		{"Sqrt", Sqrt, 4},
		{"Abs", Abs, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.fn(tc.x)
		})
	}

	for _, tc := range []struct {
		name string
		fn   func(float64, float64) float64
		x, y float64
	}{
		{"Pow", Pow, 2, 3},
		{"Atan2", Atan2, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.fn(tc.x, tc.y)
		})
	}
}
