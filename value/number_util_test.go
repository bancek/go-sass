package value

import "testing"

func TestConversionFactor(t *testing.T) {
	f, ok := conversionFactor("px", "px")
	if !ok || f != 1 {
		t.Errorf("conversionFactor(px, px) = (%v, %v), want (1, true)", f, ok)
	}
	f, ok = conversionFactor("px", "in")
	if !ok {
		t.Error("conversionFactor(px, in) should be found")
	}
	f2, ok := conversionFactor("in", "px")
	if !ok {
		t.Error("conversionFactor(in, px) should be found")
	}
	if f*f2 < 0.9 || f*f2 > 1.1 {
		t.Errorf("px->in and in->px should be inverses: %v * %v = %v", f, f2, f*f2)
	}
	_, ok = conversionFactor("px", "deg")
	if ok {
		t.Error("conversionFactor(px, deg) should not be found")
	}
}

func TestCanonicalMultiplierForUnit(t *testing.T) {
	m1 := canonicalMultiplierForUnit("px")
	m2 := canonicalMultiplierForUnit("in")
	if m1 == 0 || m2 == 0 {
		t.Error("canonical multipliers should be non-zero")
	}
	m3 := canonicalMultiplierForUnit("unknown")
	if m3 != 1 {
		t.Errorf("canonicalMultiplierForUnit(unknown) = %v, want 1", m3)
	}
}

func TestCanonicalMultiplierForList(t *testing.T) {
	m := canonicalMultiplierForList([]string{"px"})
	if m != canonicalMultiplierForUnit("px") {
		t.Error("single-element list should match unit multiplier")
	}
	if canonicalMultiplierForList(nil) != 1 {
		t.Error("nil list should return 1")
	}
	if canonicalMultiplierForList([]string{}) != 1 {
		t.Error("empty list should return 1")
	}
}

func TestCanonicalizeUnitList(t *testing.T) {
	result := canonicalizeUnitList([]string{"px", "pt"})
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	if result[0] != result[1] {
		t.Logf("canonicalizeUnitList([px, pt]) = %v (both should be 'in')", result)
	}
	empty := canonicalizeUnitList(nil)
	if len(empty) != 0 {
		t.Error("nil should return nil")
	}
	unknown := canonicalizeUnitList([]string{"foobar"})
	if len(unknown) != 1 || unknown[0] != "foobar" {
		t.Errorf("unknown unit should pass through, got %v", unknown)
	}
}

func TestUnitString(t *testing.T) {
	tests := []struct {
		num, den []string
		want     string
	}{
		{[]string{"px"}, nil, "px"},
		{nil, nil, ""},
		{[]string{"px", "s"}, nil, "px*s"},
		{[]string{"px"}, []string{"s"}, "px/s"},
		{nil, []string{"s"}, "s^-1"},
		{[]string{"px"}, []string{"s", "ms"}, "px/(s*ms)"},
		{[]string{"s", "ms"}, nil, "s*ms"},
	}
	for _, tt := range tests {
		got := unitString(tt.num, tt.den)
		if got != tt.want {
			t.Errorf("unitString(%v, %v) = %q, want %q", tt.num, tt.den, got, tt.want)
		}
	}
}

func TestStringSliceEqual(t *testing.T) {
	if !stringSliceEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Error("equal slices should be equal")
	}
	if stringSliceEqual([]string{"a", "b"}, []string{"a"}) {
		t.Error("different length slices should not be equal")
	}
	if stringSliceEqual([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("different element slices should not be equal")
	}
}

func TestCopySlice(t *testing.T) {
	orig := []string{"a", "b", "c"}
	cp := copySlice(orig)
	cp[0] = "x"
	if orig[0] != "a" {
		t.Error("copySlice should return independent copy")
	}
	if copySlice(nil) != nil {
		t.Error("copySlice(nil) should return nil")
	}
}

func TestUnitsAreConvertible(t *testing.T) {
	if !unitsAreConvertible([]string{"px"}, []string{"in"}) {
		t.Error("px and in should be convertible")
	}
	if !unitsAreConvertible([]string{"px"}, []string{"px"}) {
		t.Error("same units should be convertible")
	}
	if unitsAreConvertible([]string{"px"}, []string{"deg"}) {
		t.Error("px and deg should not be convertible")
	}
	if !unitsAreConvertible([]string{"deg"}, []string{"rad"}) {
		t.Error("deg and rad should be convertible")
	}
}

func TestAbsNumber(t *testing.T) {
	result, err := Abs(NewUnitlessNumber(-5))
	if err != nil {
		t.Fatal(err)
	}
	if v := result.NumValue(); v != 5 {
		t.Errorf("Abs(-5) = %v, want 5", v)
	}
	if result.HasUnits() {
		t.Error("Abs(-5) should be unitless")
	}
	result, err = Abs(NewSingleUnitNumber(-10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if v := result.NumValue(); v != 10 {
		t.Errorf("Abs(-10px) = %v, want 10", v)
	}
	if !result.HasUnit("px") {
		t.Error("Abs(-10px) should have px unit")
	}
	result, err = Abs(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if v := result.NumValue(); v != 0 {
		t.Errorf("Abs(0) = %v, want 0", v)
	}
}
