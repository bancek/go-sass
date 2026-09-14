package value

import (
	"math"
	"testing"
)

// === Category A: Types & Basic Methods ===

func TestCalcArgument_Equals(t *testing.T) {
	c1, _ := NewCalc(NewUnitlessNumber(1))
	c2, _ := NewCalc(NewUnitlessNumber(1))
	if !c1.Equals(c2) {
		t.Error("equal calculations should be equal")
	}
	c3, _ := NewCalc(NewUnitlessNumber(2))
	if c1.Equals(c3) {
		t.Error("different calculations should not be equal")
	}
}

func TestCalculationOperator_Name(t *testing.T) {
	if name := CalculationOperatorPlus.Name(); name != "plus" {
		t.Errorf("Plus.Name() = %q, want %q", name, "plus")
	}
	if name := CalculationOperatorMinus.Name(); name != "minus" {
		t.Errorf("Minus.Name() = %q, want %q", name, "minus")
	}
	if name := CalculationOperatorTimes.Name(); name != "times" {
		t.Errorf("Times.Name() = %q, want %q", name, "times")
	}
	if name := CalculationOperatorDividedBy.Name(); name != "divided by" {
		t.Errorf("DividedBy.Name() = %q, want %q", name, "divided by")
	}
}

func TestCalculationOperator_Operator(t *testing.T) {
	if op := CalculationOperatorPlus.Operator(); op != "+" {
		t.Errorf("Plus.Operator() = %q, want +", op)
	}
	if op := CalculationOperatorMinus.Operator(); op != "-" {
		t.Errorf("Minus.Operator() = %q, want -", op)
	}
	if op := CalculationOperatorTimes.Operator(); op != "*" {
		t.Errorf("Times.Operator() = %q, want *", op)
	}
	if op := CalculationOperatorDividedBy.Operator(); op != "/" {
		t.Errorf("DividedBy.Operator() = %q, want /", op)
	}
}

func TestCalculationOperator_Precedence(t *testing.T) {
	if p := CalculationOperatorPlus.Precedence(); p != 1 {
		t.Errorf("Plus.Precedence() = %d, want 1", p)
	}
	if p := CalculationOperatorMinus.Precedence(); p != 1 {
		t.Errorf("Minus.Precedence() = %d, want 1", p)
	}
	if p := CalculationOperatorTimes.Precedence(); p != 2 {
		t.Errorf("Times.Precedence() = %d, want 2", p)
	}
	if p := CalculationOperatorDividedBy.Precedence(); p != 2 {
		t.Errorf("DividedBy.Precedence() = %d, want 2", p)
	}
}

func TestCalculation_IsSpecialNumber(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if !c.IsSpecialNumber() {
		t.Error("IsSpecialNumber should be true")
	}
}

func TestCalculation_IsTruthy(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if !c.IsTruthy() {
		t.Error("IsTruthy should be true")
	}
}

func TestCalculation_Separator(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if c.Separator() != ListSeparatorUndecided {
		t.Error("Separator should be Undecided")
	}
}

func TestCalculation_HasBrackets(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if c.HasBrackets() {
		t.Error("HasBrackets should be false")
	}
}

func TestCalculation_LengthAsList(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if c.LengthAsList() != 1 {
		t.Error("LengthAsList should return 1")
	}
}

func TestCalculation_IsBlank(t *testing.T) {
	c := NewUnsimplified("calc", NewUnitlessNumber(1))
	if c.IsBlank() {
		t.Error("IsBlank should be false")
	}
}

func TestCalculation_Equals(t *testing.T) {
	c1 := NewUnsimplified("calc", NewUnitlessNumber(1))
	c2 := NewUnsimplified("calc", NewUnitlessNumber(1))
	if !c1.Equals(c2) {
		t.Error("same name+args should be equal")
	}
	if c1.Equals(NewUnsimplified("min", NewUnitlessNumber(1))) {
		t.Error("different name should not be equal")
	}
	if c1.Equals(NewUnsimplified("calc", NewUnitlessNumber(2))) {
		t.Error("different args should not be equal")
	}
	if c1.Equals(SassTrue) {
		t.Error("different type should not be equal")
	}
}

func TestCalculation_HashCode(t *testing.T) {
	c1 := NewUnsimplified("calc", NewUnitlessNumber(1))
	c2 := NewUnsimplified("calc", NewUnitlessNumber(1))
	if c1.HashCode() != c2.HashCode() {
		t.Error("same name+args should have same hash")
	}
}

// === Category B: Simplify ===

func TestSimplify_CalcOfCalc(t *testing.T) {
	inner, err := NewCalc(NewUnitlessNumber(5))
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewCalc(inner)
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("calc(calc(5)) should simplify to 5, got %v", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestSimplify_CalcOfCalcWithUnits(t *testing.T) {
	inner, err := NewCalc(NewSingleUnitNumber(16, "px"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewCalc(inner)
	if err != nil {
		t.Fatal(err)
	}
	// NewCalc(simplify(16px)) → SassNumber(16px) — calc(16px) flattens
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 16) {
			t.Errorf("calc(calc(16px)) should simplify to 16px, got %v", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestSimplify_CalcOfCalcOfCalc(t *testing.T) {
	inner, _ := NewCalc(NewUnitlessNumber(10))
	middle, _ := NewCalc(inner)
	result, err := NewCalc(middle)
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 10) {
			t.Errorf("calc(calc(calc(10))) should simplify to 10, got %v", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

// === Category C: Constructors (return SassNumber) ===

func TestNewCalc_Unitless(t *testing.T) {
	v, err := NewCalc(NewUnitlessNumber(42))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 42) {
			t.Errorf("value = %v, want 42", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewCalc_Unitful(t *testing.T) {
	v, err := NewCalc(NewSingleUnitNumber(16, "px"))
	if err != nil {
		t.Fatal(err)
	}
	// NewCalc simplifies: 16px is already a number, returns it directly
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 16) {
			t.Errorf("value = %v, want 16", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewCalc_String(t *testing.T) {
	arg := &SassString{Text: "var(--x)", HasQuotes: false}
	v, err := NewCalc(arg)
	if err != nil {
		t.Fatal(err)
	}
	if calc, ok := v.(*SassCalculation); ok {
		if calc.Name != "calc" {
			t.Errorf("name = %q, want calc", calc.Name)
		}
	} else {
		t.Errorf("expected SassCalculation, got %T", v)
	}
}

func TestNewMin_ThreeUnitless(t *testing.T) {
	v, err := NewMin(NewUnitlessNumber(10), NewUnitlessNumber(5), NewUnitlessNumber(20))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("min(10, 5, 20) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewMin_SameUnits(t *testing.T) {
	v, err := NewMin(NewSingleUnitNumber(10, "px"), NewSingleUnitNumber(5, "px"), NewSingleUnitNumber(20, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("min(10px, 5px, 20px) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewMin_Incompatible(t *testing.T) {
	v, err := NewMin(NewSingleUnitNumber(10, "px"), NewSingleUnitNumber(1, "em"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation for incompatible units, got %T", v)
	}
}

func TestNewMin_Empty(t *testing.T) {
	_, err := NewMin()
	if err == nil {
		t.Error("expected error for empty min()")
	}
}

func TestNewMin_Mixed(t *testing.T) {
	arg := &SassString{Text: "x", HasQuotes: false}
	v, err := NewMin(NewUnitlessNumber(1), arg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation, got %T", v)
	}
}

func TestNewMax_ThreeUnitless(t *testing.T) {
	v, err := NewMax(NewUnitlessNumber(10), NewUnitlessNumber(5), NewUnitlessNumber(20))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 20) {
			t.Errorf("max(10, 5, 20) = %v, want 20", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewMax_Empty(t *testing.T) {
	_, err := NewMax()
	if err == nil {
		t.Error("expected error for empty max()")
	}
}

func TestNewClamp_Valid(t *testing.T) {
	v, err := NewClamp(NewUnitlessNumber(0), NewUnitlessNumber(5), NewUnitlessNumber(10))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("clamp(0, 5, 10) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewClamp_BelowMin(t *testing.T) {
	v, err := NewClamp(NewUnitlessNumber(0), NewUnitlessNumber(-5), NewUnitlessNumber(10))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 0) {
			t.Errorf("clamp(0, -5, 10) = %v, want 0", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewClamp_AboveMax(t *testing.T) {
	v, err := NewClamp(NewUnitlessNumber(0), NewUnitlessNumber(15), NewUnitlessNumber(10))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 10) {
			t.Errorf("clamp(0, 15, 10) = %v, want 10", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewHypot_Unitless(t *testing.T) {
	v, err := NewHypot(NewUnitlessNumber(3), NewUnitlessNumber(4))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("hypot(3, 4) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewHypot_SameUnits(t *testing.T) {
	v, err := NewHypot(NewSingleUnitNumber(3, "px"), NewSingleUnitNumber(4, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("hypot(3px, 4px) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewHypot_Empty(t *testing.T) {
	_, err := NewHypot()
	if err == nil {
		t.Error("expected error for empty hypot()")
	}
}

func TestNewSqrt_Unitless(t *testing.T) {
	v, err := NewSqrt(NewUnitlessNumber(16))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 4) {
			t.Errorf("sqrt(16) = %v, want 4", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewSqrt_HasUnits(t *testing.T) {
	_, err := NewSqrt(NewSingleUnitNumber(16, "px"))
	if err == nil {
		t.Error("expected error for sqrt() with units")
	}
}

func TestNewSqrt_NonNumber(t *testing.T) {
	arg := &SassString{Text: "x", HasQuotes: false}
	v, err := NewSqrt(arg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation, got %T", v)
	}
}

func TestNewSin_Cos_Tan_Zeros(t *testing.T) {
	v, err := NewSin(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 0) {
			t.Errorf("sin(0) = %v, want 0", num.NumValue())
		}
	}

	v, err = NewCos(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 1) {
			t.Errorf("cos(0) = %v, want 1", num.NumValue())
		}
	}

	v, err = NewTan(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 0) {
			t.Errorf("tan(0) = %v, want 0", num.NumValue())
		}
	}
}

func TestNewAbs_Unitless(t *testing.T) {
	v, err := NewAbs(NewUnitlessNumber(-5))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("abs(-5) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewAbs_NonNumber(t *testing.T) {
	arg := &SassString{Text: "x", HasQuotes: false}
	v, err := NewAbs(arg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation, got %T", v)
	}
}

func TestNewSign_Positive(t *testing.T) {
	v, err := NewSign(NewUnitlessNumber(7))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 1) {
			t.Errorf("sign(7) = %v, want 1", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewSign_Negative(t *testing.T) {
	v, err := NewSign(NewUnitlessNumber(-7))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), -1) {
			t.Errorf("sign(-7) = %v, want -1", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewSign_Zero(t *testing.T) {
	v, err := NewSign(NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 0) {
			t.Errorf("sign(0) = %v, want 0", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewSign_NonNumber(t *testing.T) {
	arg := &SassString{Text: "x", HasQuotes: false}
	v, err := NewSign(arg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation, got %T", v)
	}
}

func TestNewPow(t *testing.T) {
	v, err := NewPow(NewUnitlessNumber(2), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 8) {
			t.Errorf("pow(2, 3) = %v, want 8", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewPow_WithUnits(t *testing.T) {
	_, err := NewPow(NewSingleUnitNumber(2, "px"), NewUnitlessNumber(3))
	if err == nil {
		t.Error("expected error for pow with units")
	}
}

func TestNewLog(t *testing.T) {
	v, err := NewLog(NewUnitlessNumber(math.E), nil)
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 1) {
			t.Errorf("log(e) = %v, want 1", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewLog_Base(t *testing.T) {
	v, err := NewLog(NewUnitlessNumber(8), NewUnitlessNumber(2))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 3) {
			t.Errorf("log2(8) = %v, want 3", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewAtan2(t *testing.T) {
	v, err := NewAtan2(NewUnitlessNumber(0), NewUnitlessNumber(-1))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 180) {
			t.Errorf("atan2(0, -1) = %v, want 180", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewRem(t *testing.T) {
	v, err := NewRem(NewUnitlessNumber(5), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 2) {
			t.Errorf("rem(5, 3) = %v, want 2", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewMod(t *testing.T) {
	v, err := NewMod(NewUnitlessNumber(5), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 2) {
			t.Errorf("mod(5, 3) = %v, want 2", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewRound_SingleUnitless(t *testing.T) {
	v, err := NewRound(NewUnitlessNumber(3.7), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 4) {
			t.Errorf("round(3.7) = %v, want 4", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewRound_Nearest_WithStep(t *testing.T) {
	strat := &SassString{Text: "nearest", HasQuotes: false}
	v, err := NewRound(strat, NewUnitlessNumber(3.5), NewUnitlessNumber(2))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 4) {
			t.Errorf("round(nearest, 3.5, 2) = %v, want 4", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewRound_Up_WithStep(t *testing.T) {
	strat := &SassString{Text: "up", HasQuotes: false}
	v, err := NewRound(strat, NewUnitlessNumber(3.1), NewUnitlessNumber(2))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 4) {
			t.Errorf("round(up, 3.1, 2) = %v, want 4", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewCalcSize(t *testing.T) {
	c, err := NewCalcSize(NewSingleUnitNumber(100, "px"), NewSingleUnitNumber(50, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "calc-size" {
		t.Errorf("name = %q, want calc-size", c.Name)
	}
	if len(c.Arguments) != 2 {
		t.Errorf("arguments = %d, want 2", len(c.Arguments))
	}
}

// === Category D: Operators ===

func TestCalculation_PlusString(t *testing.T) {
	c := NewUnsimplified("calc", NewSingleUnitNumber(10, "px"))
	s := &SassString{Text: "px", HasQuotes: false}
	result, err := c.Plus(s)
	if err != nil {
		t.Fatal(err)
	}
	if str, ok := result.(*SassString); ok {
		if str.Text != "calc(10px)px" {
			t.Errorf("Plus string = %q, want calc(10px)px", str.Text)
		}
	} else {
		t.Errorf("expected SassString, got %T", result)
	}
}

func TestCalculation_PlusNumber(t *testing.T) {
	c := NewUnsimplified("calc", NewSingleUnitNumber(10, "px"))
	_, err := c.Plus(NewUnitlessNumber(5))
	if err == nil {
		t.Error("expected error for calc + number")
	}
}

func TestCalculation_Minus(t *testing.T) {
	c := NewUnsimplified("calc", NewSingleUnitNumber(10, "px"))
	_, err := c.Minus(NewUnitlessNumber(5))
	if err == nil {
		t.Error("expected error for calc - number")
	}
}

func TestCalculation_UnaryPlus(t *testing.T) {
	c := NewUnsimplified("calc", NewSingleUnitNumber(10, "px"))
	_, err := c.UnaryPlus()
	if err == nil {
		t.Error("expected error for +calc")
	}
}

func TestCalculation_UnaryMinus(t *testing.T) {
	c := NewUnsimplified("calc", NewSingleUnitNumber(10, "px"))
	_, err := c.UnaryMinus()
	if err == nil {
		t.Error("expected error for -calc")
	}
}

// === Category E: Operate ===

func TestOperate_Plus(t *testing.T) {
	result, err := Operate(CalculationOperatorPlus, NewUnitlessNumber(1), NewUnitlessNumber(2))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 3) {
			t.Errorf("1 + 2 = %v, want 3", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_Times(t *testing.T) {
	result, err := Operate(CalculationOperatorTimes, NewUnitlessNumber(2), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 6) {
			t.Errorf("2 * 3 = %v, want 6", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_PlusSameUnits(t *testing.T) {
	result, err := Operate(CalculationOperatorPlus, NewSingleUnitNumber(1, "px"), NewSingleUnitNumber(2, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 3) {
			t.Errorf("1px + 2px = %v, want 3", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_PlusIncompatible(t *testing.T) {
	result, err := Operate(CalculationOperatorPlus, NewSingleUnitNumber(1, "px"), NewSingleUnitNumber(1, "em"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*CalculationOperation); !ok {
		t.Errorf("expected CalculationOperation, got %T", result)
	}
}

func TestOperate_TimesUnitful(t *testing.T) {
	result, err := Operate(CalculationOperatorTimes, NewSingleUnitNumber(2, "px"), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 6) {
			t.Errorf("2px * 3 = %v, want 6", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_TimesIncompatible(t *testing.T) {
	result, err := Operate(CalculationOperatorTimes, NewSingleUnitNumber(2, "px"), NewSingleUnitNumber(3, "px"))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 6) {
			t.Errorf("2px * 3px = %v, want 6", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_DividedBy(t *testing.T) {
	result, err := Operate(CalculationOperatorDividedBy, NewUnitlessNumber(6), NewUnitlessNumber(2))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 3) {
			t.Errorf("6 / 2 = %v, want 3", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_Minus(t *testing.T) {
	result, err := Operate(CalculationOperatorMinus, NewUnitlessNumber(5), NewUnitlessNumber(3))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 2) {
			t.Errorf("5 - 3 = %v, want 2", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

func TestOperate_NegativeRight(t *testing.T) {
	result, err := Operate(CalculationOperatorPlus, NewUnitlessNumber(5), NewUnitlessNumber(-3))
	if err != nil {
		t.Fatal(err)
	}
	// 5 + (-3) simplifies to SassNumber(2) — both are unitless numbers
	if num, ok := result.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 2) {
			t.Errorf("5 + (-3) = %v, want 2", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", result)
	}
}

// === Category F: Edge Cases & Regressions ===

func TestQuotedStrategyIsNotRounded(t *testing.T) {
	q := &SassString{Text: "nearest", HasQuotes: true}
	_, err := NewRound(q, NewUnitlessNumber(3), NewUnitlessNumber(1))
	if err == nil {
		t.Fatal("expected error for quoted strategy string")
	}
}

func TestQuotedStringInCalcErrors(t *testing.T) {
	q := &SassString{Text: "hello", HasQuotes: true}
	_, err := NewCalc(q)
	if err == nil {
		t.Fatal("expected error for quoted string in calc")
	}
}

func TestQuotedStringRoundStrategy(t *testing.T) {
	// Quoted "nearest" should NOT be treated as a strategy
	q := &SassString{Text: "nearest", HasQuotes: true}
	v, err := NewRound(q, NewUnitlessNumber(3), NewUnitlessNumber(1))
	if err == nil {
		t.Errorf("expected error, got %T", v)
	}
	if v != nil {
		t.Error("expected nil result")
	}
}

func TestNewHypot_AllUnitless(t *testing.T) {
	v, err := NewHypot(NewUnitlessNumber(3), NewUnitlessNumber(4))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 5) {
			t.Errorf("hypot(3, 4) = %v, want 5", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestNewHypot_ZeroZero(t *testing.T) {
	v, err := NewHypot(NewUnitlessNumber(0), NewUnitlessNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	if num, ok := v.(SassNumber); ok {
		if !fuzzyApprox(num.NumValue(), 0) {
			t.Errorf("hypot(0, 0) = %v, want 0", num.NumValue())
		}
	} else {
		t.Errorf("expected SassNumber, got %T", v)
	}
}

func TestQuotedStringNotSpecialVariable(t *testing.T) {
	// Quoted "var(--x)" should NOT be treated as a special variable
	q := &SassString{Text: "var(--x)", HasQuotes: true}
	v, err := NewRound(q, NewUnitlessNumber(5), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*SassCalculation); !ok {
		t.Errorf("expected SassCalculation for non-special-variable, got %T", v)
	}
}

func TestCalcWithQuotedString(t *testing.T) {
	qs := &SassString{Text: "hello", HasQuotes: true}
	_, err := NewCalc(qs)
	if err == nil {
		t.Fatal("expected error for quoted string in calc")
	}
}

func TestCalcWithQuotedStringInMin(t *testing.T) {
	qs := &SassString{Text: "hello", HasQuotes: true}
	_, err := NewMin(qs)
	if err == nil {
		t.Fatal("expected error for quoted string in min")
	}
}
