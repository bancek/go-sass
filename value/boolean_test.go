package value

import (
	"strings"
	"testing"
)

func TestBooleanSingleton(t *testing.T) {
	if SassTrue != NewSassBoolean(true) {
		t.Error("NewSassBoolean(true) should return SassTrue singleton")
	}
	if SassFalse != NewSassBoolean(false) {
		t.Error("NewSassBoolean(false) should return SassFalse singleton")
	}
}

func TestBooleanIsTruthy(t *testing.T) {
	if !SassTrue.IsTruthy() {
		t.Error("SassTrue.IsTruthy() should be true")
	}
	if SassFalse.IsTruthy() {
		t.Error("SassFalse.IsTruthy() should be false")
	}
}

func TestBooleanUnaryNot(t *testing.T) {
	result, err := SassTrue.UnaryNot()
	if err != nil {
		t.Fatal(err)
	}
	if result != SassFalse {
		t.Error("SassTrue.UnaryNot() should return SassFalse")
	}
	result, err = SassFalse.UnaryNot()
	if err != nil {
		t.Fatal(err)
	}
	if result != SassTrue {
		t.Error("SassFalse.UnaryNot() should return SassTrue")
	}
}

func TestBooleanEquals(t *testing.T) {
	if !SassTrue.Equals(SassTrue) {
		t.Error("true should equal true")
	}
	if !SassFalse.Equals(SassFalse) {
		t.Error("false should equal false")
	}
	if SassTrue.Equals(SassFalse) {
		t.Error("true should not equal false")
	}
	if SassTrue.Equals(Null) {
		t.Error("true should not equal null")
	}
	if SassTrue.Equals(NewUnitlessNumber(1)) {
		t.Error("true should not equal a number")
	}
}

func TestBooleanHashCode(t *testing.T) {
	if got := SassTrue.HashCode(); got != 1 {
		t.Errorf("SassTrue.HashCode() = %d, want 1", got)
	}
	if got := SassFalse.HashCode(); got != 0 {
		t.Errorf("SassFalse.HashCode() = %d, want 0", got)
	}
}

func TestBooleanListProps(t *testing.T) {
	if SassTrue.Separator() != ListSeparatorUndecided {
		t.Error("boolean separator should be Undecided")
	}
	if SassTrue.HasBrackets() {
		t.Error("boolean should not have brackets")
	}
	list, err := SassTrue.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != SassTrue {
		t.Error("AsList should return [self]")
	}
	if SassTrue.LengthAsList() != 1 {
		t.Error("LengthAsList should be 1")
	}
	if SassTrue.IsBlank() {
		t.Error("boolean should not be blank")
	}
}

func TestBooleanToCssString(t *testing.T) {
	got, err := SassTrue.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("SassTrue.ToCssString(true) = %q, want %q", got, "true")
	}
	got, err = SassTrue.ToCssString(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("SassTrue.ToCssString(false) = %q, want %q", got, "true")
	}
	got, err = SassFalse.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "false" {
		t.Errorf("SassFalse.ToCssString(true) = %q, want %q", got, "false")
	}
}

func TestBooleanString(t *testing.T) {
	got, err := SassTrue.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("SassTrue.String() = %q, want %q", got, "true")
	}
	got, err = SassFalse.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "false" {
		t.Errorf("SassFalse.String() = %q, want %q", got, "false")
	}
}

func TestBooleanRealNull(t *testing.T) {
	if SassTrue.RealNull() != SassTrue {
		t.Error("SassTrue.RealNull() should return self")
	}
}

func TestBooleanOperators(t *testing.T) {
	got, err := SassTrue.Plus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	s, err := got.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, "true") {
		t.Errorf("Plus should concatenate string representations, got %q", s)
	}

	got, err = SassTrue.Minus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	s, err = got.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "true") {
		t.Errorf("Minus should hyphenate, got %q", s)
	}

	_, err = SassTrue.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error for booleans")
	}

	got, err = SassTrue.DividedBy(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	s, err = got.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "/") {
		t.Errorf("DividedBy should produce string with slash, got %q", s)
	}

	_, err = SassTrue.Modulo(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Modulo should error for booleans")
	}

	_, err = SassTrue.GreaterThan(NewUnitlessNumber(1))
	if err == nil {
		t.Error("GreaterThan should error for booleans")
	}
}
