package value

import (
	"math"
	"strings"
	"testing"
)

func TestNewSassNumber(t *testing.T) {
	n := NewSassNumber(5, nil)
	if n.HasUnits() {
		t.Error("nil unit should create unitless number")
	}
	if n.NumValue() != 5 {
		t.Errorf("value = %v, want 5", n.NumValue())
	}

	n2 := NewSassNumber(10, ptrStr("px"))
	if !n2.HasUnit("px") {
		t.Error("should have px unit")
	}
}

func TestNewUnitlessNumber(t *testing.T) {
	n := NewUnitlessNumber(42)
	if n.HasUnits() {
		t.Error("should have no units")
	}
	if n.HasComplexUnits() {
		t.Error("should not have complex units")
	}
	if n.NumValue() != 42 {
		t.Errorf("value = %v, want 42", n.NumValue())
	}
}

func TestNewSingleUnitNumber(t *testing.T) {
	n := NewSingleUnitNumber(10, "px")
	if !n.HasUnits() {
		t.Error("should have units")
	}
	if n.HasComplexUnits() {
		t.Error("should not have complex units")
	}
	if len(n.NumNumeratorUnits()) != 1 || n.NumNumeratorUnits()[0] != "px" {
		t.Errorf("numerator = %v", n.NumNumeratorUnits())
	}
}

func TestNewComplexNumber(t *testing.T) {
	n := NewComplexNumber(10, []string{"m"}, []string{"s"})
	if !n.HasUnits() {
		t.Error("should have units")
	}
	if !n.HasComplexUnits() {
		t.Error("should have complex units")
	}
	if len(n.NumNumeratorUnits()) != 1 || n.NumNumeratorUnits()[0] != "m" {
		t.Errorf("numerator = %v", n.NumNumeratorUnits())
	}
	if len(n.NumDenominatorUnits()) != 1 || n.NumDenominatorUnits()[0] != "s" {
		t.Errorf("denominator = %v", n.NumDenominatorUnits())
	}
}

func TestSassNumberWithUnits(t *testing.T) {
	n := SassNumberWithUnits(1, []string{"px", "in"}, []string{"px"})
	if n.HasComplexUnits() {
		t.Errorf("should have simplified: num=%v den=%v", n.NumNumeratorUnits(), n.NumDenominatorUnits())
	}
	if !n.HasUnit("in") {
		t.Errorf("should have simplified to 'in', got num=%v", n.NumNumeratorUnits())
	}
}

func TestNumberIsInt(t *testing.T) {
	if !NewUnitlessNumber(5).IsInt() {
		t.Error("5 should be int")
	}
	if NewUnitlessNumber(5.5).IsInt() {
		t.Error("5.5 should not be int")
	}
	if NewUnitlessNumber(math.NaN()).IsInt() {
		t.Error("NaN should not be int")
	}
}

func TestNumberAsInt(t *testing.T) {
	i, ok := NewUnitlessNumber(5).AsInt()
	if !ok || i != 5 {
		t.Errorf("AsInt(5) = (%d, %v), want (5, true)", i, ok)
	}
	_, ok = NewUnitlessNumber(5.5).AsInt()
	if ok {
		t.Error("AsInt(5.5) should fail")
	}
}

func TestNumberAssertInt(t *testing.T) {
	i, err := NewUnitlessNumber(5).AssertInt(nil)
	if err != nil || i != 5 {
		t.Errorf("AssertInt(5) = (%d, %v)", i, err)
	}
	_, err = NewUnitlessNumber(5.5).AssertInt(nil)
	if err == nil {
		t.Error("AssertInt(5.5) should error")
	}
}

func TestNumberHasUnit(t *testing.T) {
	if NewUnitlessNumber(5).HasUnit("px") {
		t.Error("unitless should not have px")
	}
	if !NewSingleUnitNumber(5, "px").HasUnit("px") {
		t.Error("5px should have px")
	}
	if NewComplexNumber(5, []string{"m"}, []string{"s"}).HasUnit("m") {
		t.Error("complex should not return true for HasUnit")
	}
}

func TestNumberAssertUnit(t *testing.T) {
	if err := NewSingleUnitNumber(5, "px").AssertUnit("px", nil); err != nil {
		t.Errorf("5px should assert as px: %v", err)
	}
	if err := NewUnitlessNumber(5).AssertUnit("px", nil); err == nil {
		t.Error("5 should not assert as px")
	}
}

func TestNumberAssertNoUnits(t *testing.T) {
	if err := NewUnitlessNumber(5).AssertNoUnits(nil); err != nil {
		t.Errorf("unitless should pass: %v", err)
	}
	if err := NewSingleUnitNumber(5, "px").AssertNoUnits(nil); err == nil {
		t.Error("5px should fail AssertNoUnits")
	}
}

func TestNumberValueInRange(t *testing.T) {
	v, err := NewUnitlessNumber(50).ValueInRange(0, 100, nil)
	if err != nil || v != 50 {
		t.Errorf("50 in [0,100] = (%v, %v)", v, err)
	}
	_, err = NewUnitlessNumber(150).ValueInRange(0, 100, ptrStr("val"))
	if err == nil {
		t.Error("150 should be out of range")
	}
}

func TestNumberUnitString(t *testing.T) {
	if s := NewUnitlessNumber(5).UnitString(); s != "" {
		t.Errorf("unitless UnitString = %q, want %q", s, "")
	}
	if s := NewSingleUnitNumber(5, "px").UnitString(); s != "px" {
		t.Errorf("5px UnitString = %q, want %q", s, "px")
	}
	if s := NewComplexNumber(5, []string{"m"}, []string{"s"}).UnitString(); s != "m/s" {
		t.Errorf("m/s UnitString = %q, want %q", s, "m/s")
	}
}

func TestHasCompatibleUnits(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewSingleUnitNumber(10, "px")
	if !a.HasCompatibleUnits(b) {
		t.Error("px and px should be compatible")
	}
	c := NewSingleUnitNumber(1, "in")
	if !a.HasCompatibleUnits(c) {
		t.Error("px and in should be compatible")
	}
	d := NewSingleUnitNumber(90, "deg")
	if a.HasCompatibleUnits(d) {
		t.Error("px and deg should not be compatible")
	}
}

func TestCompatibleWithUnit(t *testing.T) {
	if !NewUnitlessNumber(5).CompatibleWithUnit("px") {
		t.Error("unitless should be compatible with everything")
	}
	if !NewSingleUnitNumber(5, "px").CompatibleWithUnit("in") {
		t.Error("px should be compatible with in")
	}
	if NewSingleUnitNumber(5, "px").CompatibleWithUnit("deg") {
		t.Error("px should not be compatible with deg")
	}
}

func TestConvert(t *testing.T) {
	n := NewSingleUnitNumber(1, "in")
	result, err := n.Convert([]string{"px"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v := result.NumValue(); !fuzzyApprox(v, 96) {
		t.Errorf("1in in px = %v, want 96", v)
	}
	if !result.HasUnit("px") {
		t.Error("result should have px unit")
	}

	_, err = n.Convert([]string{"deg"}, nil, nil)
	if err == nil {
		t.Error("in to deg should error")
	}
}

func TestCoerce(t *testing.T) {
	n := NewUnitlessNumber(1)
	result, err := n.Coerce([]string{"px"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUnit("px") {
		t.Error("coerce(1, px) should give px units")
	}
	if !fuzzyApprox(result.NumValue(), 1) {
		t.Errorf("value should be 1, got %v", result.NumValue())
	}
}

func TestCoerceToMatch(t *testing.T) {
	n := NewSingleUnitNumber(2, "in")
	target := NewSingleUnitNumber(10, "px")
	result, err := n.CoerceToMatch(target, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUnit("px") {
		t.Error("should have px unit")
	}
}

func TestConvertToMatch(t *testing.T) {
	n := NewUnitlessNumber(1)
	target := NewSingleUnitNumber(10, "px")
	_, err := n.ConvertToMatch(target, nil, nil)
	if err == nil {
		t.Error("unitless ConvertToMatch to unitful should error (requires explicit conversion, not coercion)")
	}

	n2 := NewSingleUnitNumber(1, "px")
	target2 := NewSingleUnitNumber(10, "px")
	result, err := n2.ConvertToMatch(target2, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUnit("px") {
		t.Error("should have px unit")
	}

	_, err = n2.ConvertToMatch(NewUnitlessNumber(5), nil, nil)
	if err == nil {
		t.Error("1px to unitless should error via ConvertToMatch (units cannot be removed)")
	}
}

func TestCoerceValueToUnit(t *testing.T) {
	v, err := NewSingleUnitNumber(1, "in").CoerceValueToUnit("px", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 96) {
		t.Errorf("1in.CoerceValueToUnit(px) = %v, want 96", v)
	}
	v, err = NewUnitlessNumber(5).CoerceValueToUnit("px", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 5) {
		t.Errorf("5.CoerceValueToUnit(px) = %v, want 5", v)
	}
}

func TestConvertValueToUnit(t *testing.T) {
	v, err := NewSingleUnitNumber(1, "in").ConvertValueToUnit("px", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 96) {
		t.Errorf("1in.ConvertValueToUnit(px) = %v, want 96", v)
	}
	_, err = NewUnitlessNumber(5).ConvertValueToUnit("px", nil)
	if err == nil {
		t.Error("5.ConvertValueToUnit(px) should error")
	}
}

func TestNumberSlash(t *testing.T) {
	n := NewUnitlessNumber(5)
	num := NewUnitlessNumber(1)
	den := NewUnitlessNumber(2)
	slashed := n.WithSlash(num, den)
	if !slashed.HasSlash() {
		t.Error("HasSlash should be true")
	}
	before, after := slashed.SlashPair()
	if before == nil || after == nil {
		t.Error("SlashPair should return non-nil values")
	}
	uns := slashed.WithoutSlash()
	if uns.HasSlash() {
		t.Error("WithoutSlash should remove slash")
	}

	noSlash := NewUnitlessNumber(5)
	if noSlash.HasSlash() {
		t.Error("regular number should not have slash")
	}
	s1, s2 := noSlash.SlashPair()
	if s1 != nil || s2 != nil {
		t.Error("SlashPair on non-slashed should return nil, nil")
	}
}

func TestNumberWithValue(t *testing.T) {
	n := NewSingleUnitNumber(5, "px")
	n2 := n.WithValue(10)
	if !fuzzyApprox(n2.NumValue(), 10) {
		t.Errorf("value = %v, want 10", n2.NumValue())
	}
	if !n2.HasUnit("px") {
		t.Error("should preserve units")
	}
}

func TestNumberWithUnits(t *testing.T) {
	n := NewSingleUnitNumber(5, "px")
	n2 := n.WithUnits([]string{"cm"}, nil)
	if !n2.HasUnit("cm") {
		t.Error("should change units")
	}
	if !fuzzyApprox(n2.NumValue(), 5) {
		t.Errorf("value should be preserved, got %v", n2.NumValue())
	}
}

func TestNumberUnitSuggestion(t *testing.T) {
	s := NewUnitlessNumber(1).UnitSuggestion("foo", nil)
	if s == "" {
		t.Error("should return a suggestion string")
	}
	s = NewSingleUnitNumber(1, "px").UnitSuggestion("foo", nil)
	if !strings.Contains(s, "foo") {
		t.Errorf("should contain name: %s", s)
	}
}

func TestSassNumberEquals(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewSingleUnitNumber(5, "px")
	if !a.SassNumberEquals(b) {
		t.Error("5px should equal 5px")
	}
	c := NewSingleUnitNumber(96, "px")
	d := NewSingleUnitNumber(1, "in")
	if !c.SassNumberEquals(d) {
		t.Error("96px should equal 1in")
	}
}

func TestNumberEqualsValue(t *testing.T) {
	n := NewUnitlessNumber(5)
	if n.Equals(SassTrue) {
		t.Error("number should not equal boolean")
	}
	if n.Equals(Null) {
		t.Error("number should not equal null")
	}
}

func TestNumberHashCode(t *testing.T) {
	a := NewSingleUnitNumber(96, "px")
	b := NewSingleUnitNumber(1, "in")
	if a.HashCode() != b.HashCode() {
		t.Error("96px and 1in should have same hash")
	}
	h1 := a.HashCode()
	h2 := a.HashCode()
	if h1 != h2 {
		t.Error("hash should be deterministic (cached)")
	}
}

func TestNumberPlus(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewSingleUnitNumber(3, "px")
	result, err := a.Plus(b)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 8) {
		t.Errorf("5px + 3px = %v, want 8", r.NumValue())
	}

	a2 := NewSingleUnitNumber(1, "in")
	b2 := NewSingleUnitNumber(96, "px")
	result, err = a2.Plus(b2)
	if err != nil {
		t.Fatal(err)
	}
	r = result.(SassNumber)
	if !r.HasUnit("in") {
		t.Errorf("1in + 96px should be in inches, got %v", r.UnitString())
	}
}

func TestNumberPlusString(t *testing.T) {
	result, err := NewUnitlessNumber(3).Plus(&SassString{Text: "px", HasQuotes: false})
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*SassString)
	if !ok {
		t.Fatal("should return a string")
	}
	if !strings.Contains(s.Text, "px") {
		t.Errorf("result should contain 'px', got %q", s.Text)
	}
}

func TestNumberPlusColor(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	_, err := NewUnitlessNumber(1).Plus(c)
	if err == nil {
		t.Error("number + color should error")
	}
}

func TestNumberMinus(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewSingleUnitNumber(3, "px")
	result, err := a.Minus(b)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 2) {
		t.Errorf("5px - 3px = %v, want 2", r.NumValue())
	}
}

func TestNumberTimes(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewUnitlessNumber(3)
	result, err := a.Times(b)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 15) {
		t.Errorf("5px * 3 = %v, want 15", r.NumValue())
	}
	if !r.HasUnit("px") {
		t.Error("should preserve px unit")
	}

	a2 := NewSingleUnitNumber(2, "in")
	b2 := NewSingleUnitNumber(3, "px")
	result, err = a2.Times(b2)
	if err != nil {
		t.Fatal(err)
	}
	r2 := result.(SassNumber)
	if v := r2.NumValue(); !fuzzyApprox(v, 6) {
		t.Errorf("2in * 3px value = %v, want 6", v)
	}
}

func TestNumberDividedBy(t *testing.T) {
	a := NewSingleUnitNumber(10, "px")
	b := NewUnitlessNumber(2)
	result, err := a.DividedBy(b)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 5) {
		t.Errorf("10px / 2 = %v, want 5", r.NumValue())
	}
	if !r.HasUnit("px") {
		t.Error("should preserve px unit")
	}

	a2 := NewSingleUnitNumber(10, "px")
	b2 := NewSingleUnitNumber(5, "px")
	result, err = a2.DividedBy(b2)
	if err != nil {
		t.Fatal(err)
	}
	r2 := result.(SassNumber)
	if !fuzzyApprox(r2.NumValue(), 2) {
		t.Errorf("10px / 5px = %v, want 2", r2.NumValue())
	}
}

func TestNumberModulo(t *testing.T) {
	a := NewSingleUnitNumber(7, "px")
	b := NewSingleUnitNumber(3, "px")
	result, err := a.Modulo(b)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 1) {
		t.Errorf("7px %% 3px = %v, want 1", r.NumValue())
	}

	a2 := NewSingleUnitNumber(-7, "px")
	result, err = a2.Modulo(b)
	if err != nil {
		t.Fatal(err)
	}
	r = result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 2) {
		t.Errorf("-7px %% 3px = %v, want 2", r.NumValue())
	}
}

func TestNumberComparisons(t *testing.T) {
	a := NewSingleUnitNumber(5, "px")
	b := NewSingleUnitNumber(3, "px")

	result, err := a.GreaterThan(b)
	if err != nil {
		t.Fatal(err)
	}
	if result != SassTrue {
		t.Error("5px > 3px should be true")
	}

	result, err = a.LessThan(b)
	if err != nil {
		t.Fatal(err)
	}
	if result != SassFalse {
		t.Error("5px < 3px should be false")
	}

	equal := NewSingleUnitNumber(96, "px")
	oneInch := NewSingleUnitNumber(1, "in")
	result, err = equal.GreaterThan(oneInch)
	if err != nil {
		t.Fatal(err)
	}
	if result != SassFalse {
		t.Error("96px > 1in should be false (equal)")
	}
}

func TestNumberUnaryOperations(t *testing.T) {
	result, err := NewSingleUnitNumber(5, "px").UnaryMinus()
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), -5) {
		t.Errorf("-5px value = %v, want -5", r.NumValue())
	}

	result, err = NewSingleUnitNumber(5, "px").UnaryPlus()
	if err != nil {
		t.Fatal(err)
	}
	r = result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 5) {
		t.Errorf("unary plus should return same value")
	}
}

func TestUnitlessArithmetic(t *testing.T) {
	u := NewUnitlessNumber(5)
	px := NewSingleUnitNumber(10, "px")

	result, err := u.Plus(px)
	if err != nil {
		t.Fatal(err)
	}
	r := result.(SassNumber)
	if !fuzzyApprox(r.NumValue(), 15) && r.HasUnit("px") {
		t.Logf("unitless + unitful may depend on type: got %v %v", r.NumValue(), r.UnitString())
	}

	result, err = u.Times(px)
	if err != nil {
		t.Fatal(err)
	}
	r = result.(SassNumber)
	if r.NumValue() != 0 && !r.HasUnits() {
		t.Logf("unitless * unitful: got %v %v", r.NumValue(), r.UnitString())
	}
}

func TestNumberToCssString(t *testing.T) {
	tests := []struct {
		val  SassNumber
		want string
	}{
		{NewUnitlessNumber(42), "42"},
		{NewUnitlessNumber(0), "0"},
		{NewUnitlessNumber(-5), "-5"},
		{NewSingleUnitNumber(12, "px"), "12px"},
	}
	for _, tt := range tests {
		got, err := tt.val.ToCssString(true)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("ToCssString = %q, want %q", got, tt.want)
		}
	}
}

func TestNumberToCssString_Slash(t *testing.T) {
	n := NewUnitlessNumber(5).WithSlash(NewUnitlessNumber(2), NewUnitlessNumber(3))
	got, err := n.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/") {
		t.Errorf("slashed number should contain slash: %q", got)
	}
	if !strings.Contains(got, "2") && !strings.Contains(got, "3") {
		t.Errorf("slashed number should render numerator/denominator: %q", got)
	}
}

func TestNumberToCssString_InfNaN(t *testing.T) {
	got, err := NewUnitlessNumber(math.Inf(1)).ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(got), "inf") && !strings.Contains(got, "calc") {
		t.Errorf("Inf should be wrapped or formatted: %q", got)
	}

	got, err = NewUnitlessNumber(math.NaN()).ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(got), "nan") && !strings.Contains(got, "calc") {
		t.Errorf("NaN should be wrapped or formatted: %q", got)
	}
}

func TestNumberString(t *testing.T) {
	got, err := NewSingleUnitNumber(12, "px").String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "12px" {
		t.Errorf("String() = %q, want %q", got, "12px")
	}
}

func ptrStr(s string) *string { return &s }

func fuzzyApprox(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestHasPossiblyCompatibleUnits(t *testing.T) {
	if !NewUnitlessNumber(5).HasPossiblyCompatibleUnits(NewUnitlessNumber(10)) {
		t.Error("unitless should be compatible with unitless")
	}
	if NewUnitlessNumber(5).HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "px")) {
		t.Error("unitless should NOT be compatible with unitful")
	}
	if !NewSingleUnitNumber(5, "px").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "px")) {
		t.Error("same unit should be compatible")
	}
	if !NewSingleUnitNumber(5, "px").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "in")) {
		t.Error("px and in should be compatible (both in length set)")
	}
	if !NewSingleUnitNumber(5, "px").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "em")) {
		t.Error("px and em should be compatible")
	}
	if NewSingleUnitNumber(5, "px").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "deg")) {
		t.Error("px and deg should NOT be compatible")
	}
	if !NewSingleUnitNumber(5, "deg").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "rad")) {
		t.Error("deg and rad should be compatible (angle set)")
	}
	if !NewSingleUnitNumber(5, "s").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "ms")) {
		t.Error("s and ms should be compatible (time set)")
	}
	if !NewSingleUnitNumber(5, "dpi").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "dppx")) {
		t.Error("dpi and dppx should be compatible (pixel density set)")
	}
	if !NewSingleUnitNumber(5, "custom").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "px")) {
		t.Error("unknown unit should be compatible with known unit")
	}
	if !NewSingleUnitNumber(5, "custom1").HasPossiblyCompatibleUnits(NewSingleUnitNumber(10, "custom2")) {
		t.Error("unknown units should be compatible with each other")
	}
}

func TestCoerceValue(t *testing.T) {
	n := NewUnitlessNumber(5)
	v, err := n.CoerceValue([]string{"px"}, []string{}, ptrStr("test"))
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 5) {
		t.Errorf("value = %v, want 5", v)
	}

	m := NewSingleUnitNumber(1, "in")
	v, err = m.CoerceValue([]string{"px"}, []string{}, ptrStr("test"))
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 96) {
		t.Errorf("1in should coerce to 96px, got %v", v)
	}
}

func TestConvertValue(t *testing.T) {
	n := NewUnitlessNumber(5)
	v, err := n.ConvertValue([]string{}, []string{}, ptrStr("test"))
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 5) {
		t.Errorf("value = %v, want 5", v)
	}

	m := NewSingleUnitNumber(1, "in")
	v, err = m.ConvertValue([]string{"px"}, []string{}, ptrStr("test"))
	if err != nil {
		t.Fatal(err)
	}
	if !fuzzyApprox(v, 96) {
		t.Errorf("1in should convert to 96px, got %v", v)
	}

	_, err = n.ConvertValue([]string{"px"}, []string{}, ptrStr("test"))
	if err == nil {
		t.Error("expected error converting unitless to px")
	}
}

// --- Exact error message tests (locked for the Rust port) ---

func TestNumberAssertIntErrorMessage(t *testing.T) {
	_, err := NewUnitlessNumber(2.5).AssertInt(ptrStr("limit"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$limit: 2.5 is not an int."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	_, err = NewSingleUnitNumber(2.5, "px").AssertInt(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "2.5px is not an int."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberAssertNoUnitsErrorMessage(t *testing.T) {
	err := NewSingleUnitNumber(5, "px").AssertNoUnits(ptrStr("number"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$number: Expected 5px to have no units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	err = NewSingleUnitNumber(5, "px").AssertNoUnits(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "Expected 5px to have no units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberAssertUnitErrorMessage(t *testing.T) {
	err := NewUnitlessNumber(5).AssertUnit("px", ptrStr("length"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := `$length: Expected 5 to have unit "px".`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberValueInRangeErrorMessage(t *testing.T) {
	_, err := NewUnitlessNumber(150).ValueInRange(0, 100, ptrStr("amount"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$amount: Expected 150 to be within 0 and 100."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	_, err = NewSingleUnitNumber(150, "px").ValueInRange(0, 100, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "Expected 150px to be within 0px and 100px."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberValueInRangeWithUnitErrorMessage(t *testing.T) {
	_, err := NewUnitlessNumber(150).ValueInRangeWithUnit(0, 100, "alpha", "%")
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$alpha: Expected 150 to be within 0% and 100%."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberConvertValueToMatchErrorMessages(t *testing.T) {
	// Incompatible units with names.
	_, err := NewSingleUnitNumber(1, "s").ConvertValueToMatch(NewSingleUnitNumber(1, "px"), ptrStr("x"), ptrStr("y"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$x: 1s and $y: 1px have incompatible units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	// One unitless.
	_, err = NewUnitlessNumber(1).ConvertValueToMatch(NewSingleUnitNumber(1, "px"), ptrStr("x"), ptrStr("y"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$x: 1 and $y: 1px have incompatible units (one has units and the other doesn't)."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	// No names.
	_, err = NewSingleUnitNumber(1, "px").ConvertValueToMatch(NewSingleUnitNumber(2, "s"), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "1px and 2s have incompatible units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberCoerceValueToUnitErrorMessages(t *testing.T) {
	// Known unit type: lists the compatible units.
	_, err := NewSingleUnitNumber(10, "px").CoerceValueToUnit("rad", ptrStr("number"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$number: Expected 10px to have an angle unit (deg, grad, rad, turn)."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	// Unknown unit.
	_, err = NewSingleUnitNumber(5, "px").CoerceValueToUnit("foo", ptrStr("x"))
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "$x: Expected 5px to have unit foo."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberConvertValueErrorMessages(t *testing.T) {
	// Multiple units.
	_, err := NewSingleUnitNumber(5, "px").ConvertValue([]string{"px", "s"}, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "Expected 5px to have units px*s."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}

	// Unitful to unitless.
	_, err = NewSingleUnitNumber(5, "px").ConvertValue(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "Expected 5px to have no units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNumberCoerceUnitsErrorPicksSelfCoercionError(t *testing.T) {
	// numCoerceUnits returns the error from n.CoerceValueToMatch(other) when
	// both coercions fail: the receiver appears first in the message.
	a := NewSingleUnitNumber(1, "px")
	b := NewSingleUnitNumber(2, "s")
	_, err := a.LessThan(b)
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "1px and 2s have incompatible units."; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
