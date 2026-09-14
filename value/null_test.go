package value

import "testing"

func TestNullSingleton(t *testing.T) {
	if Null == nil {
		t.Error("Null should not be nil")
	}
}

func TestNullIsTruthy(t *testing.T) {
	if Null.IsTruthy() {
		t.Error("Null.IsTruthy() should be false")
	}
}

func TestNullIsBlank(t *testing.T) {
	if !Null.IsBlank() {
		t.Error("Null.IsBlank() should be true")
	}
}

func TestNullRealNull(t *testing.T) {
	if got := Null.RealNull(); got != nil {
		t.Errorf("Null.RealNull() should return nil, got %v", got)
	}
}

func TestNullUnaryNot(t *testing.T) {
	result, err := Null.UnaryNot()
	if err != nil {
		t.Fatal(err)
	}
	if result != SassTrue {
		t.Error("Null.UnaryNot() should return SassTrue")
	}
}

func TestNullEquals(t *testing.T) {
	if !Null.Equals(Null) {
		t.Error("null should equal null")
	}
	if Null.Equals(SassTrue) {
		t.Error("null should not equal true")
	}
	if Null.Equals(&SassString{Text: "", HasQuotes: false}) {
		t.Error("null should not equal empty string")
	}
	if Null.Equals(NewUnitlessNumber(0)) {
		t.Error("null should not equal 0")
	}
}

func TestNullHashCode(t *testing.T) {
	if got := Null.HashCode(); got != 0 {
		t.Errorf("Null.HashCode() = %d, want 0", got)
	}
}

func TestNullListProps(t *testing.T) {
	if Null.Separator() != ListSeparatorUndecided {
		t.Error("null separator should be Undecided")
	}
	if Null.HasBrackets() {
		t.Error("null should not have brackets")
	}
	list, err := Null.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != Null {
		t.Error("AsList should return [self]")
	}
	if Null.LengthAsList() != 1 {
		t.Error("LengthAsList should be 1")
	}
}

func TestNullToCssString(t *testing.T) {
	got, err := Null.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("Null.ToCssString(true) = %q, want %q", got, "")
	}
	got, err = Null.ToCssString(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("Null.ToCssString(false) = %q, want %q", got, "")
	}
}

func TestNullString(t *testing.T) {
	got, err := Null.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "null" {
		t.Errorf("Null.String() = %q, want %q", got, "null")
	}
}

func TestNullOperators(t *testing.T) {
	got, err := Null.Plus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("Null + number should not return nil")
	}

	got, err = Null.UnaryPlus()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("UnaryPlus on null should not return nil")
	}
}
