// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package functions

import (
	"math"
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/value"
)

// --- helpers ---

func mathUnit(v float64, unit string) value.Value {
	return value.NewSingleUnitNumber(v, unit)
}

func mathComplex(v float64, numUnits, denUnits []string) value.Value {
	return value.NewComplexNumber(v, numUnits, denUnits)
}

func mathBIC(t *testing.T, c sasscallable.Callable) *BuiltInCallable {
	t.Helper()
	b, ok := c.(*BuiltInCallable)
	if !ok {
		t.Fatalf("expected *BuiltInCallable, got %T", c)
	}
	return b
}

// mathEvalWarn calls fn with a recording logger and returns the result, error,
// and any recorded warnings.
func mathEvalWarn(t *testing.T, fn *BuiltInCallable, args ...value.Value) (value.Value, error, *recordingLogger) {
	t.Helper()
	rl := &recordingLogger{}
	ec := &evalcontext.EvaluationContext{Logger: rl}
	res, err := fn.CallbackFor(len(args), nil)
	if err != nil {
		t.Fatalf("CallbackFor(%d): %v", len(args), err)
	}
	paramCount := 0
	if res.Params != nil {
		paramCount = len(res.Params.Parameters)
	}
	for len(args) < paramCount {
		args = append(args, value.Null)
	}
	got, err := res.Fn(ec, args)
	return got, err, rl
}

func assertNoWarnings(t *testing.T, rl *recordingLogger) {
	t.Helper()
	if len(rl.messages) != 0 {
		t.Errorf("warnings = %q, want none", rl.messages)
	}
}

func assertSingleWarning(t *testing.T, rl *recordingLogger, want string, dep *deprecation.Deprecation) {
	t.Helper()
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d (%q), want 1", len(rl.messages), rl.messages)
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
	if rl.deprecations[0] != dep {
		t.Errorf("deprecation = %v, want %v", rl.deprecations[0], dep)
	}
}

func assertNumWithUnit(t *testing.T, v value.Value, wantValue float64, wantUnit string) {
	t.Helper()
	n, ok := v.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", v)
	}
	if math.Abs(n.NumValue()-wantValue) > 1e-9 {
		t.Errorf("NumValue() = %v, want %v", n.NumValue(), wantValue)
	}
	if wantUnit == "" {
		if n.HasUnits() {
			t.Errorf("expected no units, got %q", n.UnitString())
		}
	} else if !n.HasUnit(wantUnit) {
		t.Errorf("unit = %q, want %q", n.UnitString(), wantUnit)
	}
}

// --- GlobalMathFunctions ---

func TestGlobalMathFunctionsNames(t *testing.T) {
	fns := GlobalMathFunctions()
	want := []string{"abs", "ceil", "floor", "max", "min", "percentage", "random", "round", "unit", "comparable", "unitless"}
	if len(fns) != len(want) {
		t.Fatalf("len = %d, want %d", len(fns), len(want))
	}
	for i, name := range want {
		if fns[i].Name() != name {
			t.Errorf("fns[%d].Name() = %q, want %q", i, fns[i].Name(), name)
		}
	}
}

func TestGlobalMathFunctionsEmitDeprecationWarnings(t *testing.T) {
	fns := GlobalMathFunctions()
	tests := []struct {
		index    int
		wantName string
		args     []value.Value
	}{
		{1, "math.ceil", []value.Value{num(4.2)}},
		{2, "math.floor", []value.Value{num(4.8)}},
		{3, "math.max", []value.Value{commaList(num(1))}},
		{4, "math.min", []value.Value{commaList(num(1))}},
		{5, "math.percentage", []value.Value{num(0.5)}},
		{6, "math.random", []value.Value{value.Null}},
		{7, "math.round", []value.Value{num(1.4)}},
		{8, "math.unit", []value.Value{num(1)}},
		{9, "math.compatible", []value.Value{num(1), num(2)}},
		{10, "math.is-unitless", []value.Value{num(1)}},
	}
	for _, tt := range tests {
		fn := mathBIC(t, fns[tt.index])
		_, err, rl := mathEvalWarn(t, fn, tt.args...)
		if err != nil {
			t.Fatalf("%s: %v", tt.wantName, err)
		}
		want := strings.Join([]string{
			`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
			`Use ` + tt.wantName + ` instead.`,
			``,
			`More info and automated migrator: https://sass-lang.com/d/import`,
		}, "\n")
		assertSingleWarning(t, rl, want, deprecation.GlobalBuiltin)
	}
}

func TestGlobalMathAbsWarnsGlobalBuiltin(t *testing.T) {
	got, err, rl := mathEvalWarn(t, absFunctionGlobal(), mathUnit(-3, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3px")
	want := strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use math.abs instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
	assertSingleWarning(t, rl, want, deprecation.GlobalBuiltin)
}

func TestGlobalMathAbsPercentDeprecation(t *testing.T) {
	got, err, rl := mathEvalWarn(t, absFunctionGlobal(), mathUnit(-50, "%"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "50%")
	want := strings.Join([]string{
		`Passing percentage units to the global abs() function is deprecated.`,
		`In the future, this will emit a CSS abs() function to be resolved by the browser.`,
		`To preserve current behavior: math.abs(-50%)`,
		`To emit a CSS abs() now: abs(#{-50%})`,
		`More info: https://sass-lang.com/d/abs-percent`,
	}, "\n")
	assertSingleWarning(t, rl, want, deprecation.AbsPercent)
}

func TestGlobalMathAbsNonNumber(t *testing.T) {
	_, err, _ := mathEvalWarn(t, absFunctionGlobal(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- MathModule ---

func TestMathModuleURL(t *testing.T) {
	m := MathModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:math" {
		t.Errorf("URL = %q, want %q", url, "sass:math")
	}
}

func TestMathModuleFunctionNames(t *testing.T) {
	m := MathModule()
	want := []string{
		"abs", "acos", "asin", "atan", "atan2", "ceil", "clamp", "cos",
		"compatible", "div", "floor", "hypot", "is-unitless", "log", "max",
		"min", "percentage", "pow", "random", "round", "sin", "sqrt", "tan",
		"unit",
	}
	if m.Functions().Len() != len(want) {
		t.Fatalf("functions = %d, want %d", m.Functions().Len(), len(want))
	}
	var names []string
	for name := range m.Functions().Keys() {
		names = append(names, name)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("functions[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestMathModuleNoMixins(t *testing.T) {
	m := MathModule()
	if m.Mixins().Len() != 0 {
		t.Errorf("mixins = %d, want 0", m.Mixins().Len())
	}
}

func TestMathModuleVariables(t *testing.T) {
	m := MathModule()
	wantOrder := []string{"e", "pi", "epsilon", "max-safe-integer", "min-safe-integer", "max-number", "min-number"}
	if m.Variables().Len() != len(wantOrder) {
		t.Fatalf("variables = %d, want %d", m.Variables().Len(), len(wantOrder))
	}
	var names []string
	for name := range m.Variables().Keys() {
		names = append(names, name)
	}
	for i, name := range names {
		if name != wantOrder[i] {
			t.Errorf("variables[%d] = %q, want %q", i, name, wantOrder[i])
		}
	}
	wantValues := map[string]float64{
		"e":                math.E,
		"pi":               math.Pi,
		"epsilon":          2.220446049250313e-16,
		"max-safe-integer": 9007199254740991,
		"min-safe-integer": -9007199254740991,
		"max-number":       math.MaxFloat64,
		"min-number":       math.SmallestNonzeroFloat64,
	}
	for name, wantVal := range wantValues {
		v, ok := m.Variables().Get(name)
		if !ok {
			t.Fatalf("variable %q not found", name)
		}
		n, ok := v.(value.SassNumber)
		if !ok {
			t.Fatalf("variable %q is %T, want SassNumber", name, v)
		}
		if n.NumValue() != wantVal {
			t.Errorf("variable %q = %v, want %v", name, n.NumValue(), wantVal)
		}
		if n.HasUnits() {
			t.Errorf("variable %q should be unitless", name)
		}
	}
}

// --- abs (module) ---

func TestMathAbs(t *testing.T) {
	got, err := eval(absFunction(), mathUnit(-10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "10px")

	got, err = eval(absFunction(), mathUnit(10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "10px")

	got, err = eval(absFunction(), num(-5))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5")
}

func TestMathAbsPreservesComplexUnits(t *testing.T) {
	got, err := eval(absFunction(), mathComplex(-5, []string{"px"}, []string{"s"}))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(5px / 1s)")
}

func TestMathAbsModuleDoesNotWarn(t *testing.T) {
	_, err, rl := mathEvalWarn(t, absFunction(), mathUnit(-50, "%"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoWarnings(t, rl)
}

func TestMathAbsNonNumber(t *testing.T) {
	_, err := eval(absFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- ceil ---

func TestMathCeil(t *testing.T) {
	got, err := eval(ceilFunction(), num(4.2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5")

	got, err = eval(ceilFunction(), mathUnit(4.2, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5px")

	got, err = eval(ceilFunction(), num(-4.2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "-4")

	got, err = eval(ceilFunction(), num(4))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "4")

	// Negative zero serializes as "0".
	got, err = eval(ceilFunction(), num(-0.1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")
}

func TestMathCeilPreservesComplexUnits(t *testing.T) {
	got, err := eval(ceilFunction(), mathComplex(5.5, []string{"px"}, []string{"s"}))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(6px / 1s)")
}

func TestMathCeilNonNumber(t *testing.T) {
	_, err := eval(ceilFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- floor ---

func TestMathFloor(t *testing.T) {
	got, err := eval(floorFunction(), num(4.8))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "4")

	got, err = eval(floorFunction(), mathUnit(-4.2, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "-5px")

	got, err = eval(floorFunction(), num(4))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "4")

	got, err = eval(floorFunction(), num(0.9))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")
}

func TestMathFloorNonNumber(t *testing.T) {
	_, err := eval(floorFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- round ---

func TestMathRound(t *testing.T) {
	got, err := eval(roundFunction(), num(2.5))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	got, err = eval(roundFunction(), num(-2.5))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "-3")

	got, err = eval(roundFunction(), num(2.4))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2")

	got, err = eval(roundFunction(), mathUnit(2.6, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3px")
}

func TestMathRoundHalfCases(t *testing.T) {
	// Rounds half away from zero.
	tests := []struct {
		in   float64
		want string
	}{
		{0.5, "1"},
		{-0.5, "-1"},
		{1.5, "2"},
		{-1.5, "-2"},
	}
	for _, tt := range tests {
		got, err := eval(roundFunction(), num(tt.in))
		if err != nil {
			t.Fatal(err)
		}
		assertInspect(t, got, tt.want)
	}
}

func TestMathRoundNonNumber(t *testing.T) {
	_, err := eval(roundFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- max ---

func TestMathMax(t *testing.T) {
	got, err := eval(maxFunction(), commaList(num(1), num(2), num(3)))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	got, err = eval(maxFunction(), commaList(mathUnit(1, "px"), mathUnit(2, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2px")

	// 1in == 96px > 50px: keeps original 1in.
	got, err = eval(maxFunction(), commaList(mathUnit(1, "in"), mathUnit(50, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1in")

	// Unitless coerces to unitful.
	got, err = eval(maxFunction(), commaList(num(1), mathUnit(2, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2px")

	got, err = eval(maxFunction(), commaList(mathUnit(1, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1px")
}

func TestMathMaxEmpty(t *testing.T) {
	_, err := eval(maxFunction(), commaList())
	assertErrMsg(t, err, "At least one argument must be passed.")
}

func TestMathMaxNonNumberElement(t *testing.T) {
	_, err := eval(maxFunction(), commaList(num(1), str("foo")))
	assertErrMsg(t, err, "foo is not a number.")

	_, err = eval(maxFunction(), commaList(num(1), quoted("foo")))
	assertErrMsg(t, err, `"foo" is not a number.`)
}

func TestMathMaxIncompatibleUnits(t *testing.T) {
	_, err := eval(maxFunction(), commaList(mathUnit(1, "px"), mathUnit(2, "s")))
	assertErrMsg(t, err, "1px and 2s have incompatible units.")
}

// --- min ---

func TestMathMin(t *testing.T) {
	got, err := eval(minFunction(), commaList(num(3), num(1), num(2)))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	// 1000ms < 2s: keeps original 1000ms.
	got, err = eval(minFunction(), commaList(mathUnit(2, "s"), mathUnit(1000, "ms")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1000ms")

	got, err = eval(minFunction(), commaList(mathUnit(3, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3px")
}

func TestMathMinEmpty(t *testing.T) {
	_, err := eval(minFunction(), commaList())
	assertErrMsg(t, err, "At least one argument must be passed.")
}

func TestMathMinNonNumberElement(t *testing.T) {
	_, err := eval(minFunction(), commaList(num(1), str("foo")))
	assertErrMsg(t, err, "foo is not a number.")
}

func TestMathMinIncompatibleUnits(t *testing.T) {
	_, err := eval(minFunction(), commaList(mathUnit(1, "px"), mathUnit(2, "s")))
	assertErrMsg(t, err, "1px and 2s have incompatible units.")
}

// --- percentage ---

func TestMathPercentage(t *testing.T) {
	got, err := eval(percentageFunction(), num(0.5))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "50%")

	got, err = eval(percentageFunction(), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "100%")
}

func TestMathPercentageWithUnits(t *testing.T) {
	_, err := eval(percentageFunction(), mathUnit(0.5, "px"))
	assertErrMsg(t, err, "$number: Expected 0.5px to have no units.")
}

func TestMathPercentageNonNumber(t *testing.T) {
	_, err := eval(percentageFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- acos ---

func TestMathAcos(t *testing.T) {
	got, err := eval(mathBIC(t, acosFunction()), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0deg")

	got, err = eval(mathBIC(t, acosFunction()), num(-1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "180deg")

	// Out of domain: NaN with deg unit.
	got, err = eval(mathBIC(t, acosFunction()), num(2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(NaN * 1deg)")
}

func TestMathAcosWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, acosFunction()), mathUnit(2, "px"))
	assertErrMsg(t, err, "$number: Expected 2px to have no units.")
}

func TestMathAcosNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, acosFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- asin ---

func TestMathAsin(t *testing.T) {
	got, err := eval(mathBIC(t, asinFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0deg")

	got, err = eval(mathBIC(t, asinFunction()), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "90deg")
}

func TestMathAsinWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, asinFunction()), mathUnit(1, "px"))
	assertErrMsg(t, err, "$number: Expected 1px to have no units.")
}

// --- atan ---

func TestMathAtan(t *testing.T) {
	got, err := eval(mathBIC(t, atanFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0deg")

	got, err = eval(mathBIC(t, atanFunction()), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "45deg")
}

func TestMathAtanWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, atanFunction()), mathUnit(1, "px"))
	assertErrMsg(t, err, "$number: Expected 1px to have no units.")
}

// --- atan2 ---

func TestMathAtan2(t *testing.T) {
	got, err := eval(mathBIC(t, atan2Function()), num(1), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "90deg")

	got, err = eval(mathBIC(t, atan2Function()), num(0), num(-1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "180deg")

	// Units are converted to match: 96px == 1in.
	got, err = eval(mathBIC(t, atan2Function()), mathUnit(1, "in"), mathUnit(96, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "45deg")
}

func TestMathAtan2IncompatibleUnits(t *testing.T) {
	_, err := eval(mathBIC(t, atan2Function()), mathUnit(1, "px"), mathUnit(1, "s"))
	assertErrMsg(t, err, "$x: 1s and $y: 1px have incompatible units.")
}

func TestMathAtan2MixedUnitless(t *testing.T) {
	_, err := eval(mathBIC(t, atan2Function()), mathUnit(1, "px"), num(1))
	assertErrMsg(t, err, "$x: 1 and $y: 1px have incompatible units (one has units and the other doesn't).")
}

func TestMathAtan2NonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, atan2Function()), str("foo"), num(1))
	assertErrMsg(t, err, "$y: foo is not a number.")

	_, err = eval(mathBIC(t, atan2Function()), num(1), str("foo"))
	assertErrMsg(t, err, "$x: foo is not a number.")
}

// --- clamp ---

func TestMathClamp(t *testing.T) {
	got, err := eval(mathBIC(t, clampFunction()), num(1), num(2), num(3))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2")

	got, err = eval(mathBIC(t, clampFunction()), num(1), num(0), num(3))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	got, err = eval(mathBIC(t, clampFunction()), num(1), num(5), num(3))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	// min >= max: returns min.
	got, err = eval(mathBIC(t, clampFunction()), num(3), num(2), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	got, err = eval(mathBIC(t, clampFunction()), mathUnit(1, "px"), mathUnit(2, "px"), mathUnit(3, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2px")

	// Compatible units are converted: 50px < 1in (96px), so min is returned.
	got, err = eval(mathBIC(t, clampFunction()), mathUnit(1, "in"), mathUnit(50, "px"), mathUnit(2, "in"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1in")
}

func TestMathClampIncompatibleUnits(t *testing.T) {
	_, err := eval(mathBIC(t, clampFunction()), mathUnit(1, "px"), mathUnit(2, "s"), mathUnit(3, "px"))
	assertErrMsg(t, err, "$number: 2s and $min: 1px have incompatible units.")

	_, err = eval(mathBIC(t, clampFunction()), mathUnit(1, "px"), mathUnit(2, "px"), mathUnit(3, "s"))
	assertErrMsg(t, err, "$max: 3s and $min: 1px have incompatible units.")
}

func TestMathClampMixedUnitless(t *testing.T) {
	_, err := eval(mathBIC(t, clampFunction()), num(0), mathUnit(1, "px"), num(2))
	assertErrMsg(t, err, "$number: 1px and $min: 0 have incompatible units (one has units and the other doesn't).")
}

func TestMathClampNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, clampFunction()), str("foo"), num(2), num(3))
	assertErrMsg(t, err, "$min: foo is not a number.")

	_, err = eval(mathBIC(t, clampFunction()), num(1), str("foo"), num(3))
	assertErrMsg(t, err, "$number: foo is not a number.")

	_, err = eval(mathBIC(t, clampFunction()), num(1), num(2), str("foo"))
	assertErrMsg(t, err, "$max: foo is not a number.")
}

// --- cos ---

func TestMathCos(t *testing.T) {
	got, err := eval(mathBIC(t, cosFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	got, err = eval(mathBIC(t, cosFunction()), mathUnit(0, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	got, err = eval(mathBIC(t, cosFunction()), mathUnit(180, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "-1")

	got, err = eval(mathBIC(t, cosFunction()), num(math.Pi))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "-1")
}

func TestMathCosNonAngleUnit(t *testing.T) {
	_, err := eval(mathBIC(t, cosFunction()), mathUnit(10, "px"))
	assertErrMsg(t, err, "$number: Expected 10px to have an angle unit (deg, grad, rad, turn).")
}

func TestMathCosNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, cosFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- sin ---

func TestMathSin(t *testing.T) {
	got, err := eval(mathBIC(t, sinFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")

	got, err = eval(mathBIC(t, sinFunction()), num(math.Pi/2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	got, err = eval(mathBIC(t, sinFunction()), mathUnit(90, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")
}

func TestMathSinNonAngleUnit(t *testing.T) {
	_, err := eval(mathBIC(t, sinFunction()), mathUnit(10, "px"))
	assertErrMsg(t, err, "$number: Expected 10px to have an angle unit (deg, grad, rad, turn).")
}

// --- tan ---

func TestMathTan(t *testing.T) {
	got, err := eval(mathBIC(t, tanFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")

	got, err = eval(mathBIC(t, tanFunction()), mathUnit(45, "deg"))
	if err != nil {
		t.Fatal(err)
	}
	// C libm value (installed via sassmathcgo in math_cgo_test.go, matching
	// Dart and the Rust port). Go's native math.Tan would give
	// 0.9999999999999998. The raw value is 0.9999999999999999, but inspect
	// serializes fuzzy integers as integers (Dart `_asInt`, #2800) — Dart
	// 1.104 prints `1` here too.
	assertInspect(t, got, "1")
}

func TestMathTanNonAngleUnit(t *testing.T) {
	_, err := eval(mathBIC(t, tanFunction()), mathUnit(10, "px"))
	assertErrMsg(t, err, "$number: Expected 10px to have an angle unit (deg, grad, rad, turn).")
}

// --- compatible (module) ---

func TestMathCompatible(t *testing.T) {
	got, err := eval(mathBIC(t, compatibleFunction()), mathUnit(1, "px"), mathUnit(1, "in"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(mathBIC(t, compatibleFunction()), mathUnit(1, "px"), mathUnit(1, "s"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)

	got, err = eval(mathBIC(t, compatibleFunction()), num(1), mathUnit(1, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(mathBIC(t, compatibleFunction()), mathUnit(1, "px"), mathComplex(1, []string{"px"}, []string{"s"}))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
}

func TestMathCompatibleNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, compatibleFunction()), str("foo"), num(1))
	assertErrMsg(t, err, "$number1: foo is not a number.")

	_, err = eval(mathBIC(t, compatibleFunction()), num(1), str("foo"))
	assertErrMsg(t, err, "$number2: foo is not a number.")
}

// --- div ---

func TestMathDiv(t *testing.T) {
	got, err, rl := mathEvalWarn(t, mathBIC(t, divFunction()), num(10), num(2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5")
	assertNoWarnings(t, rl)

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), mathUnit(10, "px"), num(2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5px")
	assertNoWarnings(t, rl)

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), mathUnit(10, "px"), mathUnit(5, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2")
	assertNoWarnings(t, rl)

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), mathUnit(10, "px"), mathUnit(2, "s"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(5px / 1s)")
	assertNoWarnings(t, rl)
}

func TestMathDivNonNumberWarns(t *testing.T) {
	got, err, rl := mathEvalWarn(t, mathBIC(t, divFunction()), str("a"), str("b"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "a/b")
	want := strings.Join([]string{
		`math.div() will only support number arguments in a future release.`,
		`Use list.slash() instead for a slash separator.`,
	}, "\n")
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d, want 1", len(rl.messages))
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
	if rl.deprecations[0] != nil {
		t.Errorf("deprecation = %v, want nil (plain warning)", rl.deprecations[0])
	}

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), num(10), str("b"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "10/b")
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d, want 1", len(rl.messages))
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
}

func TestMathDivByZero(t *testing.T) {
	// Division by zero yields Infinity/NaN numbers, not errors.
	got, err, rl := mathEvalWarn(t, mathBIC(t, divFunction()), num(1), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(infinity)")
	assertNoWarnings(t, rl)

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), num(-1), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(-infinity)")
	assertNoWarnings(t, rl)

	got, err, rl = mathEvalWarn(t, mathBIC(t, divFunction()), num(0), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(NaN)")
	assertNoWarnings(t, rl)
}

// --- hypot ---

func TestMathHypot(t *testing.T) {
	got, err := eval(mathBIC(t, hypotFunction()), commaList(num(3), num(4)))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5")

	got, err = eval(mathBIC(t, hypotFunction()), commaList(mathUnit(3, "px"), mathUnit(4, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5px")

	// Units are converted to match the first number's units.
	got, err = eval(mathBIC(t, hypotFunction()), commaList(mathUnit(1, "in"), mathUnit(96, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1.4142135623730951in")

	got, err = eval(mathBIC(t, hypotFunction()), commaList(mathUnit(5, "px")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "5px")
}

func TestMathHypotThreeArgs(t *testing.T) {
	// hypot(2, 3, 6) = sqrt(4 + 9 + 36) = 7
	got, err := eval(mathBIC(t, hypotFunction()), commaList(num(2), num(3), num(6)))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "7")
}

func TestMathHypotEmpty(t *testing.T) {
	_, err := eval(mathBIC(t, hypotFunction()), commaList())
	assertErrMsg(t, err, "At least one argument must be passed.")
}

func TestMathHypotNonNumberElement(t *testing.T) {
	_, err := eval(mathBIC(t, hypotFunction()), commaList(num(3), str("foo")))
	assertErrMsg(t, err, "foo is not a number.")
}

func TestMathHypotIncompatibleUnits(t *testing.T) {
	_, err := eval(mathBIC(t, hypotFunction()), commaList(mathUnit(3, "px"), mathUnit(4, "s")))
	assertErrMsg(t, err, "$numbers[2]: 4s and $numbers[1]: 3px have incompatible units.")
}

func TestMathHypotMixedUnitless(t *testing.T) {
	_, err := eval(mathBIC(t, hypotFunction()), commaList(mathUnit(3, "px"), num(4)))
	assertErrMsg(t, err, "$numbers[2]: 4 and $numbers[1]: 3px have incompatible units (one has units and the other doesn't).")
}

// --- is-unitless ---

func TestMathIsUnitless(t *testing.T) {
	got, err := eval(mathBIC(t, isUnitlessModuleFunction()), num(5))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(mathBIC(t, isUnitlessModuleFunction()), mathUnit(5, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)

	got, err = eval(mathBIC(t, isUnitlessModuleFunction()), mathComplex(5, []string{"px"}, []string{"s"}))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
}

func TestMathIsUnitlessNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, isUnitlessModuleFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// isUnitlessFunction is currently unused by the module/global lists but kept
// to match the Dart source structure; lock its behavior too.
func TestMathIsUnitlessStandaloneFunction(t *testing.T) {
	got, err := eval(mathBIC(t, isUnitlessFunction()), num(5))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(mathBIC(t, isUnitlessFunction()), mathUnit(5, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)

	_, err = eval(mathBIC(t, isUnitlessFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- log ---

func TestMathLog(t *testing.T) {
	got, err := eval(mathBIC(t, logFunction()), num(math.E))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")

	got, err = eval(mathBIC(t, logFunction()), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")

	got, err = eval(mathBIC(t, logFunction()), num(100), num(10))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2")

	got, err = eval(mathBIC(t, logFunction()), num(8), num(2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	got, err = eval(mathBIC(t, logFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(-infinity)")
}

func TestMathLogWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, logFunction()), mathUnit(10, "px"))
	assertErrMsg(t, err, "$number: Expected 10px to have no units.")
}

func TestMathLogBaseWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, logFunction()), num(10), mathUnit(2, "px"))
	assertErrMsg(t, err, "$base: Expected 2px to have no units.")
}

func TestMathLogNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, logFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")

	_, err = eval(mathBIC(t, logFunction()), quoted("foo"))
	assertErrMsg(t, err, `$number: "foo" is not a number.`)

	_, err = eval(mathBIC(t, logFunction()), num(10), str("foo"))
	assertErrMsg(t, err, "$base: foo is not a number.")
}

// --- pow ---

func TestMathPow(t *testing.T) {
	got, err := eval(mathBIC(t, powFunction()), num(2), num(10))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1024")

	got, err = eval(mathBIC(t, powFunction()), num(4), num(0.5))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "2")

	got, err = eval(mathBIC(t, powFunction()), num(2), num(-2))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0.25")
}

func TestMathPowWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, powFunction()), mathUnit(2, "px"), num(3))
	assertErrMsg(t, err, "$base: Expected 2px to have no units.")

	_, err = eval(mathBIC(t, powFunction()), num(2), mathUnit(3, "px"))
	assertErrMsg(t, err, "$exponent: Expected 3px to have no units.")
}

func TestMathPowNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, powFunction()), str("foo"), num(2))
	assertErrMsg(t, err, "$base: foo is not a number.")

	_, err = eval(mathBIC(t, powFunction()), num(2), str("foo"))
	assertErrMsg(t, err, "$exponent: foo is not a number.")
}

// --- sqrt ---

func TestMathSqrt(t *testing.T) {
	got, err := eval(mathBIC(t, sqrtFunction()), num(9))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "3")

	got, err = eval(mathBIC(t, sqrtFunction()), num(0))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "0")

	got, err = eval(mathBIC(t, sqrtFunction()), num(-1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "calc(NaN)")
}

func TestMathSqrtWithUnits(t *testing.T) {
	_, err := eval(mathBIC(t, sqrtFunction()), mathUnit(9, "px"))
	assertErrMsg(t, err, "$number: Expected 9px to have no units.")
}

func TestMathSqrtNonNumber(t *testing.T) {
	_, err := eval(mathBIC(t, sqrtFunction()), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- unit ---

func TestMathUnit(t *testing.T) {
	tests := []struct {
		arg  value.Value
		want string
	}{
		{num(5), ""},
		{mathUnit(5, "px"), "px"},
		{mathComplex(5, []string{"px", "s"}, nil), "px*s"},
		{mathComplex(5, []string{"px"}, []string{"s"}), "px/s"},
		{mathComplex(5, nil, []string{"s"}), "s^-1"},
		{mathComplex(5, nil, []string{"s", "ms"}), "(s*ms)^-1"},
		{mathComplex(5, []string{"px"}, []string{"s", "ms"}), "px/(s*ms)"},
	}
	for _, tt := range tests {
		got, err := eval(unitFunction(), tt.arg)
		if err != nil {
			t.Fatal(err)
		}
		s, ok := got.(*value.SassString)
		if !ok {
			t.Fatalf("expected *SassString, got %T", got)
		}
		if s.Text != tt.want {
			t.Errorf("unit() = %q, want %q", s.Text, tt.want)
		}
		if !s.HasQuotes {
			t.Error("unit() should return a quoted string")
		}
	}
}

func TestMathUnitNonNumber(t *testing.T) {
	_, err := eval(unitFunction(), str("foo"))
	assertErrMsg(t, err, "$number: foo is not a number.")
}

// --- unitString / joinStrings helpers ---

func TestMathUnitStringHelper(t *testing.T) {
	tests := []struct {
		n    value.SassNumber
		want string
	}{
		{value.NewUnitlessNumber(1), ""},
		{value.NewSingleUnitNumber(1, "px"), "px"},
		{value.NewComplexNumber(1, []string{"px", "s"}, nil), "px*s"},
		{value.NewComplexNumber(1, []string{"px"}, []string{"s"}), "px/s"},
		{value.NewComplexNumber(1, nil, []string{"s"}), "s^-1"},
		{value.NewComplexNumber(1, nil, []string{"s", "ms"}), "(s*ms)^-1"},
		{value.NewComplexNumber(1, []string{"px"}, []string{"s", "ms"}), "px/(s*ms)"},
	}
	for _, tt := range tests {
		if got := unitString(tt.n); got != tt.want {
			t.Errorf("unitString = %q, want %q", got, tt.want)
		}
	}
}

func TestMathJoinStrings(t *testing.T) {
	if got := joinStrings(nil, "*"); got != "" {
		t.Errorf("joinStrings(nil) = %q, want %q", got, "")
	}
	if got := joinStrings([]string{"a"}, "*"); got != "a" {
		t.Errorf("joinStrings([a]) = %q, want %q", got, "a")
	}
	if got := joinStrings([]string{"a", "b", "c"}, "*"); got != "a*b*c" {
		t.Errorf("joinStrings([a b c]) = %q, want %q", got, "a*b*c")
	}
}

// --- comparable (global) ---

func TestMathComparableGlobalName(t *testing.T) {
	if name := comparableFunctionGlobal().Name(); name != "comparable" {
		t.Errorf("Name() = %q, want %q", name, "comparable")
	}
}

func TestMathComparableGlobal(t *testing.T) {
	got, err := eval(comparableFunctionGlobal(), mathUnit(1, "px"), mathUnit(1, "in"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(comparableFunctionGlobal(), mathUnit(1, "px"), mathUnit(1, "s"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
}

// --- unitless (global) ---

func TestMathUnitlessGlobalName(t *testing.T) {
	if name := unitlessFunctionGlobal().Name(); name != "unitless" {
		t.Errorf("Name() = %q, want %q", name, "unitless")
	}
}

func TestMathUnitlessGlobal(t *testing.T) {
	got, err := eval(unitlessFunctionGlobal(), num(5))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)

	got, err = eval(unitlessFunctionGlobal(), mathUnit(5, "px"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
}

// --- random ---

func TestMathRandomNull(t *testing.T) {
	for range 10 {
		got, err := eval(randomFunction())
		if err != nil {
			t.Fatal(err)
		}
		n, ok := got.(value.SassNumber)
		if !ok {
			t.Fatalf("expected SassNumber, got %T", got)
		}
		if n.NumValue() < 0 || n.NumValue() >= 1 {
			t.Errorf("random() = %v, want in [0, 1)", n.NumValue())
		}
		if n.HasUnits() {
			t.Error("random() should be unitless")
		}
	}
}

func TestMathRandomLimitOne(t *testing.T) {
	got, err := eval(randomFunction(), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")
}

func TestMathRandomLimit(t *testing.T) {
	for range 20 {
		got, err := eval(randomFunction(), num(10))
		if err != nil {
			t.Fatal(err)
		}
		n, ok := got.(value.SassNumber)
		if !ok {
			t.Fatalf("expected SassNumber, got %T", got)
		}
		if !n.IsInt() {
			t.Errorf("random(10) = %v, want an int", n.NumValue())
		}
		if n.NumValue() < 1 || n.NumValue() > 10 {
			t.Errorf("random(10) = %v, want in [1, 10]", n.NumValue())
		}
		if n.HasUnits() {
			t.Error("random(10) should be unitless")
		}
	}
}

func TestMathRandomZeroLimit(t *testing.T) {
	_, err := eval(randomFunction(), num(0))
	assertErrMsg(t, err, "$limit: Must be greater than 0, was 0.")
}

func TestMathRandomNegativeLimit(t *testing.T) {
	_, err := eval(randomFunction(), num(-5))
	assertErrMsg(t, err, "$limit: Must be greater than 0, was -5.")
}

func TestMathRandomNonIntLimit(t *testing.T) {
	_, err := eval(randomFunction(), num(2.5))
	assertErrMsg(t, err, "$limit: 2.5 is not an int.")
}

func TestMathRandomNonNumber(t *testing.T) {
	_, err := eval(randomFunction(), str("foo"))
	assertErrMsg(t, err, "$limit: foo is not a number.")
}

func TestMathRandomUnitsDeprecation(t *testing.T) {
	got, err, rl := mathEvalWarn(t, randomFunction(), mathUnit(10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := got.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", got)
	}
	if n.NumValue() < 1 || n.NumValue() > 10 {
		t.Errorf("random(10px) = %v, want in [1, 10]", n.NumValue())
	}
	if n.HasUnits() {
		t.Error("random(10px) result should ignore units")
	}
	want := strings.Join([]string{
		`math.random() will no longer ignore $limit units (10px) in a future release.`,
		``,
		`Recommendation: math.random(math.div($limit, 1px)) * 1px`,
		``,
		`To preserve current behavior: math.random(math.div($limit, 1px))`,
		``,
		`More info: https://sass-lang.com/d/function-units`,
	}, "\n")
	assertSingleWarning(t, rl, want, deprecation.FunctionUnits)
}

func TestMathRandomUnitsZeroStillErrors(t *testing.T) {
	// The units deprecation warning fires before the range check.
	_, err, rl := mathEvalWarn(t, randomFunction(), mathUnit(0, "px"))
	assertErrMsg(t, err, "$limit: Must be greater than 0, was 0px.")
	if len(rl.messages) != 1 {
		t.Errorf("warnings = %d, want 1 (deprecation before error)", len(rl.messages))
	}
}

func TestGlobalMathRandomUnitsEmitsBothWarnings(t *testing.T) {
	// The wrapped global random() emits the GlobalBuiltin warning first, then
	// the FunctionUnits warning from the callback body.
	fns := GlobalMathFunctions()
	rnd := mathBIC(t, fns[6])
	got, err, rl := mathEvalWarn(t, rnd, mathUnit(10, "px"))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := got.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", got)
	}
	if n.NumValue() < 1 || n.NumValue() > 10 {
		t.Errorf("random(10px) = %v, want in [1, 10]", n.NumValue())
	}
	if len(rl.messages) != 2 {
		t.Fatalf("warnings = %d (%q), want 2", len(rl.messages), rl.messages)
	}
	wantGlobal := strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use math.random instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
	if rl.messages[0] != wantGlobal {
		t.Errorf("warning[0] = %q, want %q", rl.messages[0], wantGlobal)
	}
	if rl.deprecations[0] != deprecation.GlobalBuiltin {
		t.Errorf("deprecation[0] = %v, want GlobalBuiltin", rl.deprecations[0])
	}
	wantUnits := strings.Join([]string{
		`math.random() will no longer ignore $limit units (10px) in a future release.`,
		``,
		`Recommendation: math.random(math.div($limit, 1px)) * 1px`,
		``,
		`To preserve current behavior: math.random(math.div($limit, 1px))`,
		``,
		`More info: https://sass-lang.com/d/function-units`,
	}, "\n")
	if rl.messages[1] != wantUnits {
		t.Errorf("warning[1] = %q, want %q", rl.messages[1], wantUnits)
	}
	if rl.deprecations[1] != deprecation.FunctionUnits {
		t.Errorf("deprecation[1] = %v, want FunctionUnits", rl.deprecations[1])
	}
}
