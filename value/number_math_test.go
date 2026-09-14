package value

import (
	"math"
	"testing"
)

func TestSqrtNumber(t *testing.T) {
	r, err := SqrtNumber(NewUnitlessNumber(9))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 3) {
		t.Errorf("sqrt(9) = %v, want 3", v.NumValue())
	}
	_, err = SqrtNumber(NewSingleUnitNumber(10, "px"))
	if err == nil {
		t.Error("sqrt(10px) should error")
	}
}

func TestSinNumber(t *testing.T) {
	r, err := SinNumber(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("sin(0) = %v, want 0", v.NumValue())
	}
	r, err = SinNumber(NewSingleUnitNumber(90, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 1) {
		t.Errorf("sin(90deg) = %v, want 1", v.NumValue())
	}
	r, err = SinNumber(NewSingleUnitNumber(3.141592653589793, "rad"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("sin(pi rad) = %v, want ~0", v.NumValue())
	}
}

func TestCosNumber(t *testing.T) {
	r, err := CosNumber(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 1) {
		t.Errorf("cos(0) = %v, want 1", v.NumValue())
	}
	r, err = CosNumber(NewSingleUnitNumber(90, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("cos(90deg) = %v, want ~0", v.NumValue())
	}
}

func TestTanNumber(t *testing.T) {
	r, err := TanNumber(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("tan(0) = %v, want 0", v.NumValue())
	}
	r, err = TanNumber(NewSingleUnitNumber(45, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 1) {
		t.Errorf("tan(45deg) = %v, want ~1", v.NumValue())
	}
}

func TestAtanNumber(t *testing.T) {
	r, err := AtanNumber(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("atan(0) = %v, want 0deg", v.NumValue())
	}
	if !r.HasUnit("deg") {
		t.Error("atan result should be in degrees")
	}
}

func TestAsinNumber(t *testing.T) {
	r, err := AsinNumber(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("asin(0) = %v, want 0deg", v.NumValue())
	}
}

func TestAcosNumber(t *testing.T) {
	r, err := AcosNumber(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("acos(1) = %v, want 0deg", v.NumValue())
	}
}

func TestAbsNumber_math(t *testing.T) {
	r, err := AbsNumber(NewSingleUnitNumber(-10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 10) {
		t.Errorf("Abs(-10px) = %v, want 10", v.NumValue())
	}
	if !r.HasUnit("px") {
		t.Error("Abs should preserve units")
	}
}

func TestLogNumber(t *testing.T) {
	r, err := LogNumber(NewUnitlessNumber(math.E), nil)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 1) {
		t.Errorf("ln(e) = %v, want 1", v.NumValue())
	}
	r, err = LogNumber(NewUnitlessNumber(100), NewUnitlessNumber(10))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 2) {
		t.Errorf("log10(100) = %v, want 2", v.NumValue())
	}
	_, err = LogNumber(NewSingleUnitNumber(10, "px"), nil)
	if err == nil {
		t.Error("log of number with units should error")
	}
}

func TestPowNumber(t *testing.T) {
	r, err := PowNumber(NewUnitlessNumber(2), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 8) {
		t.Errorf("pow(2, 3) = %v, want 8", v.NumValue())
	}
	_, err = PowNumber(NewSingleUnitNumber(2, "px"), NewUnitlessNumber(3))
	if err == nil {
		t.Error("pow with units should error")
	}
}

func TestAtan2Number(t *testing.T) {
	r, err := Atan2Number(NewUnitlessNumber(1), NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 90) {
		t.Errorf("atan2(1, 0) = %v, want 90deg", v.NumValue())
	}
	if !r.HasUnit("deg") {
		t.Error("atan2 result should be in degrees")
	}
	r, err = Atan2Number(NewUnitlessNumber(0), NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.(SassNumber); !approx(v.NumValue(), 0) {
		t.Errorf("atan2(0, 1) = %v, want 0deg", v.NumValue())
	}
}

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
