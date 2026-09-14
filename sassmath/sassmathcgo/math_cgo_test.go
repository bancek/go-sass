//go:build cgomath

package sassmathcgo

import (
	"math"
	"testing"

	"github.com/bancek/go-sass/sassmath"
)

func bits(v float64) uint64 { return math.Float64bits(v) }

func TestCgoTrigMatchesDart(t *testing.T) {
	SassMathCgo()

	pi := 3.141592653589793

	t.Run("Sin", func(t *testing.T) {
		got := bits(sassmath.Sin(pi))
		want := uint64(0x3ca1a62633145c07)
		if got != want {
			t.Errorf("Sin(PI): got 0x%016x, want 0x%016x (Dart golden)", got, want)
		}
	})

	t.Run("Cos", func(t *testing.T) {
		got := bits(sassmath.Cos(pi))
		want := uint64(0xbff0000000000000)
		if got != want {
			t.Errorf("Cos(PI): got 0x%016x, want 0x%016x (Dart golden)", got, want)
		}
	})

	t.Run("Cos_HalfPi", func(t *testing.T) {
		got := bits(sassmath.Cos(pi / 2))
		want := uint64(0x3c91a62633145c07)
		if got != want {
			t.Errorf("Cos(PI/2): got 0x%016x, want 0x%016x (Dart golden)", got, want)
		}
	})

	t.Run("Atan2_1_0", func(t *testing.T) {
		got := bits(sassmath.Atan2(1, 0))
		want := uint64(0x3ff921fb54442d18)
		if got != want {
			t.Errorf("Atan2(1,0): got 0x%016x, want 0x%016x (Dart golden)", got, want)
		}
	})

	t.Run("Atan2_0_n1", func(t *testing.T) {
		got := bits(sassmath.Atan2(0, -1))
		want := uint64(0x400921fb54442d18)
		if got != want {
			t.Errorf("Atan2(0,-1): got 0x%016x, want 0x%016x (Dart golden)", got, want)
		}
	})

	t.Run("MatrixMul", func(t *testing.T) {
		m := [9]float64{0.5, 0.3, 0.2, 0.1, 0.4, 0.5, 0.3, 0.3, 0.4}
		v0, v1, v2 := 1.0, 0.5, 0.2

		r := float64(m[0]*v0) + float64(m[1]*v1) + float64(m[2]*v2)
		g := float64(m[3]*v0) + float64(m[4]*v1) + float64(m[5]*v2)
		b := float64(m[6]*v0) + float64(m[7]*v1) + float64(m[8]*v2)

		assertBits(t, "matmul_r", bits(r), 0x3fe6147ae147ae15)
		assertBits(t, "matmul_g", bits(g), 0x3fd999999999999a)
		assertBits(t, "matmul_b", bits(b), 0x3fe0f5c28f5c28f6)
	})
}

func assertBits(t *testing.T, label string, got, want uint64) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got 0x%016x, want 0x%016x (Dart golden)", label, got, want)
	}
}
