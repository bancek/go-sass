package functions

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/value"
)

// --- helpers ---

func metaGlobalBuiltinWarningMsg(name string) string {
	return strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use meta.` + name + ` instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
}

func featureExistsWarningMsg() string {
	return strings.Join([]string{
		`The feature-exists() function is deprecated.`,
		``,
		`More info: https://sass-lang.com/d/feature-exists`,
	}, "\n")
}

func mkCalc(name string, args ...any) *value.SassCalculation {
	return &value.SassCalculation{Name: name, Arguments: args}
}

func mkArgListWithKeywords(t *testing.T, pairs ...any) *value.SassArgumentList {
	t.Helper()
	keywords := orderedmap.New[string, value.Value]()
	for i := 0; i < len(pairs); i += 2 {
		keywords.Put(pairs[i].(string), pairs[i+1].(value.Value))
	}
	al, err := value.NewSassArgumentList(nil, keywords, value.ListSeparatorComma)
	if err != nil {
		t.Fatal(err)
	}
	return al
}

func assertQuotedString(t *testing.T, v value.Value, want string) {
	t.Helper()
	s, ok := v.(*value.SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", v)
	}
	if s.Text != want {
		t.Errorf("Text = %q, want %q", s.Text, want)
	}
	if !s.HasQuotes {
		t.Error("expected quoted string")
	}
}

func assertUnquotedString(t *testing.T, v value.Value, want string) {
	t.Helper()
	s, ok := v.(*value.SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", v)
	}
	if s.Text != want {
		t.Errorf("Text = %q, want %q", s.Text, want)
	}
	if s.HasQuotes {
		t.Error("expected unquoted string")
	}
}

// --- SharedMetaFunctions ---

func TestSharedMetaFunctionsNames(t *testing.T) {
	fns := SharedMetaFunctions()
	want := []string{"feature-exists", "inspect", "type-of", "keywords"}
	if len(fns) != len(want) {
		t.Fatalf("len = %d, want %d", len(fns), len(want))
	}
	for i, name := range want {
		if fns[i].Name() != name {
			t.Errorf("fns[%d].Name() = %q, want %q", i, fns[i].Name(), name)
		}
	}
}

func TestSharedMetaInspectEmitsDeprecationWarning(t *testing.T) {
	fns := SharedMetaFunctions()
	inspectFn := mathBIC(t, fns[1])
	got, err, rl := selEval(t, inspectFn, num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "1")
	assertSingleWarning(t, rl, metaGlobalBuiltinWarningMsg("inspect"), deprecation.GlobalBuiltin)
}

func TestSharedMetaTypeOfEmitsDeprecationWarning(t *testing.T) {
	fns := SharedMetaFunctions()
	typeOf := mathBIC(t, fns[2])
	got, err, rl := selEval(t, typeOf, num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "number")
	assertSingleWarning(t, rl, metaGlobalBuiltinWarningMsg("type-of"), deprecation.GlobalBuiltin)
}

func TestSharedMetaKeywordsEmitsDeprecationWarning(t *testing.T) {
	fns := SharedMetaFunctions()
	keywords := mathBIC(t, fns[3])
	got, err, rl := selEval(t, keywords, mkArgListWithKeywords(t, "a", mkNum(1)))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(a: 1)")
	assertSingleWarning(t, rl, metaGlobalBuiltinWarningMsg("keywords"), deprecation.GlobalBuiltin)
}

func TestSharedMetaFeatureExistsEmitsBothWarnings(t *testing.T) {
	// The wrapper emits GlobalBuiltin first, then the callback emits
	// FeatureExists.
	fns := SharedMetaFunctions()
	featureExists := mathBIC(t, fns[0])
	got, err, rl := selEval(t, featureExists, str("at-error"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)
	if len(rl.messages) != 2 {
		t.Fatalf("warnings = %d (%q), want 2", len(rl.messages), rl.messages)
	}
	if rl.messages[0] != metaGlobalBuiltinWarningMsg("feature-exists") {
		t.Errorf("warning[0] = %q, want %q", rl.messages[0], metaGlobalBuiltinWarningMsg("feature-exists"))
	}
	if rl.deprecations[0] != deprecation.GlobalBuiltin {
		t.Errorf("deprecation[0] = %v, want GlobalBuiltin", rl.deprecations[0])
	}
	if rl.messages[1] != featureExistsWarningMsg() {
		t.Errorf("warning[1] = %q, want %q", rl.messages[1], featureExistsWarningMsg())
	}
	if rl.deprecations[1] != deprecation.FeatureExists {
		t.Errorf("deprecation[1] = %v, want FeatureExists", rl.deprecations[1])
	}
}

// --- MetaModule ---

func TestMetaModuleURL(t *testing.T) {
	m := MetaModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:meta" {
		t.Errorf("URL = %q, want %q", url, "sass:meta")
	}
}

func TestMetaModuleFunctions(t *testing.T) {
	m := MetaModule()
	want := []string{
		"feature-exists", "inspect", "type-of", "keywords",
		"calc-name", "calc-args", "accepts-content",
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

func TestMetaModuleNoMixinsNoVariables(t *testing.T) {
	m := MetaModule()
	if m.Mixins().Len() != 0 {
		t.Errorf("mixins = %d, want 0", m.Mixins().Len())
	}
	if m.Variables().Len() != 0 {
		t.Errorf("variables = %d, want 0", m.Variables().Len())
	}
}

// --- feature-exists ---

func TestFeatureExistsSupportedFeatures(t *testing.T) {
	features := []string{
		"global-variable-shadowing",
		"extend-selector-pseudoclass",
		"units-level-3",
		"at-error",
		"custom-property",
	}
	for _, feature := range features {
		got, err, rl := selEval(t, featureExistsFunction(), str(feature))
		if err != nil {
			t.Fatalf("%s: %v", feature, err)
		}
		assertBool(t, got, true)
		assertSingleWarning(t, rl, featureExistsWarningMsg(), deprecation.FeatureExists)
	}
}

func TestFeatureExistsUnknownFeature(t *testing.T) {
	got, err, rl := selEval(t, featureExistsFunction(), str("unknown-feature"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
	assertSingleWarning(t, rl, featureExistsWarningMsg(), deprecation.FeatureExists)
}

func TestFeatureExistsQuotedString(t *testing.T) {
	got, err, _ := selEval(t, featureExistsFunction(), quoted("at-error"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)
}

func TestFeatureExistsTypeError(t *testing.T) {
	// The deprecation warning is emitted before the type check.
	_, err, rl := selEval(t, featureExistsFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "feature")
	assertSingleWarning(t, rl, featureExistsWarningMsg(), deprecation.FeatureExists)
}

// --- inspect ---

func TestInspectNumber(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), num(1))
	assertUnquotedString(t, got, "1")
}

func TestInspectQuotedString(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), quoted("foo"))
	assertUnquotedString(t, got, `"foo"`)
}

func TestInspectUnquotedString(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), str("foo"))
	assertUnquotedString(t, got, "foo")
}

func TestInspectCommaList(t *testing.T) {
	// SerializeValueInspect — no Dart toString parens.
	got := selEvalOK(t, inspectFunction(), commaList(num(1), num(2), num(3)))
	assertUnquotedString(t, got, "1, 2, 3")
}

func TestInspectSingleCommaList(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), commaList(num(1)))
	assertUnquotedString(t, got, "(1,)")
}

func TestInspectEmptyList(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), commaList())
	assertUnquotedString(t, got, "()")
}

func TestInspectNull(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), value.Null)
	assertUnquotedString(t, got, "null")
}

func TestInspectBoolean(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), value.SassTrue)
	assertUnquotedString(t, got, "true")
}

func TestInspectMap(t *testing.T) {
	got := selEvalOK(t, inspectFunction(), mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2)))
	assertUnquotedString(t, got, "(a: 1, b: 2)")
}

// --- type-of ---

func TestTypeOf(t *testing.T) {
	arglist, err := value.NewSassArgumentList(nil, nil, value.ListSeparatorComma)
	if err != nil {
		t.Fatal(err)
	}
	color := colorMust(value.NewColorForSpace(value.RgbColorSpace, [3]float64{255, 0, 0}, 1, [4]bool{}))
	cases := []struct {
		name string
		v    value.Value
		want string
	}{
		{"arglist", arglist, "arglist"},
		{"bool", value.SassTrue, "bool"},
		{"color", color, "color"},
		{"list", commaList(num(1), num(2)), "list"},
		{"map", mkMap(mkStr("a"), mkNum(1)), "map"},
		{"null", value.Null, "null"},
		{"number", num(1), "number"},
		{"function", value.NewSassFunction(nil), "function"},
		{"mixin", value.NewSassMixin(nil), "mixin"},
		{"calculation", mkCalc("calc", value.NewUnitlessNumber(1)), "calculation"},
		{"string", str("foo"), "string"},
		{"quoted string", quoted("foo"), "string"},
	}
	for _, tc := range cases {
		got := selEvalOK(t, typeOfFunction(), tc.v)
		assertUnquotedString(t, got, tc.want)
	}
}

// --- keywords ---

func TestKeywords(t *testing.T) {
	al := mkArgListWithKeywords(t, "a", mkNum(1), "b", mkNum(2))
	got := selEvalOK(t, keywordsFunction(), al)
	assertInspect(t, got, "(a: 1, b: 2)")
}

func TestKeywordsKeysAreUnquotedStrings(t *testing.T) {
	al := mkArgListWithKeywords(t, "key", mkStr("val"))
	got := selEvalOK(t, keywordsFunction(), al)
	m, ok := got.(*value.SassMap)
	if !ok {
		t.Fatalf("expected *SassMap, got %T", got)
	}
	if m.LengthAsList() != 1 {
		t.Fatalf("len = %d, want 1", m.LengthAsList())
	}
	for k := range m.Entries() {
		assertUnquotedString(t, k, "key")
	}
}

func TestKeywordsEmpty(t *testing.T) {
	al := mkArgListWithKeywords(t)
	got := selEvalOK(t, keywordsFunction(), al)
	assertInspect(t, got, "()")
}

func TestKeywordsMarksAccessed(t *testing.T) {
	al := mkArgListWithKeywords(t, "a", mkNum(1))
	if al.WereKeywordsAccessed() {
		t.Fatal("keywords should not be accessed yet")
	}
	selEvalOK(t, keywordsFunction(), al)
	if !al.WereKeywordsAccessed() {
		t.Error("keywords() should mark keywords as accessed")
	}
}

func TestKeywordsListError(t *testing.T) {
	_, err, _ := selEval(t, keywordsFunction(), commaList(num(1), num(2)))
	assertScriptErr(t, err, "(1, 2) is not an argument list.", "args")
}

func TestKeywordsStringError(t *testing.T) {
	_, err, _ := selEval(t, keywordsFunction(), str("foo"))
	assertScriptErr(t, err, "foo is not an argument list.", "args")
}

func TestKeywordsNumberError(t *testing.T) {
	_, err, _ := selEval(t, keywordsFunction(), num(1))
	assertScriptErr(t, err, "1 is not an argument list.", "args")
}

// --- calc-name ---

func TestCalcName(t *testing.T) {
	calc := mkCalc("calc", value.NewUnitlessNumber(1))
	got := selEvalOK(t, mathBIC(t, calcNameFunction()), calc)
	assertQuotedString(t, got, "calc")
}

func TestCalcNameMax(t *testing.T) {
	calc := mkCalc("max", value.NewUnitlessNumber(1), value.NewUnitlessNumber(2))
	got := selEvalOK(t, mathBIC(t, calcNameFunction()), calc)
	assertQuotedString(t, got, "max")
}

func TestCalcNameNumberError(t *testing.T) {
	_, err, _ := selEval(t, mathBIC(t, calcNameFunction()), num(1))
	assertScriptErr(t, err, "1 is not a calculation.", "calc")
}

func TestCalcNameStringError(t *testing.T) {
	_, err, _ := selEval(t, mathBIC(t, calcNameFunction()), str("calc(1px)"))
	assertScriptErr(t, err, "calc(1px) is not a calculation.", "calc")
}

// --- calc-args ---

func TestCalcArgsNumbers(t *testing.T) {
	calc := mkCalc("max", value.NewUnitlessNumber(1), value.NewSingleUnitNumber(2, "px"))
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	assertInspect(t, got, "1, 2px")
	assertListSep(t, got, value.ListSeparatorComma)
}

func TestCalcArgsSingle(t *testing.T) {
	calc := mkCalc("calc", value.NewSingleUnitNumber(1, "px"))
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	assertInspect(t, got, "(1px,)")
	assertListLen(t, got, 1)
}

func TestCalcArgsOperation(t *testing.T) {
	// Non-Value arguments are stringified via their String() method
	// (Dart: argument.toString()).
	op := &value.CalculationOperation{
		Operator: value.CalculationOperatorPlus,
		Left:     value.NewSingleUnitNumber(1, "px"),
		Right:    value.NewSingleUnitNumber(2, "px"),
	}
	calc := mkCalc("calc", op)
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	assertListLen(t, got, 1)
	items, err := got.AsList()
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, items[0], "1px + 2px")
}

func TestCalcArgsInterpolation(t *testing.T) {
	// Matches Dart: CalculationInterpolation.toString() => value.
	calc := mkCalc("calc", &value.CalculationInterpolation{Value: "var(--x)"})
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	items, err := got.AsList()
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, items[0], "var(--x)")
}

func TestCalcArgsStringArgument(t *testing.T) {
	// String arguments are Values and pass through unchanged.
	calc := mkCalc("calc", str("100% - 10px"))
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	items, err := got.AsList()
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, items[0], "100% - 10px")
}

func TestCalcArgsNestedCalculation(t *testing.T) {
	inner := mkCalc("calc", value.NewSingleUnitNumber(1, "px"))
	calc := mkCalc("max", inner, value.NewSingleUnitNumber(2, "px"))
	got := selEvalOK(t, mathBIC(t, calcArgsFunction()), calc)
	items, err := got.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if _, ok := items[0].(*value.SassCalculation); !ok {
		t.Errorf("items[0] = %T, want *SassCalculation", items[0])
	}
}

func TestCalcArgsTypeError(t *testing.T) {
	_, err, _ := selEval(t, mathBIC(t, calcArgsFunction()), num(1))
	assertScriptErr(t, err, "1 is not a calculation.", "calc")
}

// --- accepts-content ---

func TestAcceptsContentBuiltInTrue(t *testing.T) {
	bic := metaFunction("m", "", func(_ *evalcontext.EvaluationContext, _ []value.Value) (value.Value, error) {
		return value.Null, nil
	})
	bic.SetAcceptsContent(true)
	mixin := value.NewSassMixin(bic)
	got := selEvalOK(t, mathBIC(t, acceptsContentFunction()), mixin)
	assertBool(t, got, true)
}

func TestAcceptsContentBuiltInFalse(t *testing.T) {
	bic := metaFunction("m", "", func(_ *evalcontext.EvaluationContext, _ []value.Value) (value.Value, error) {
		return value.Null, nil
	})
	mixin := value.NewSassMixin(bic)
	got := selEvalOK(t, mathBIC(t, acceptsContentFunction()), mixin)
	assertBool(t, got, false)
}

func TestAcceptsContentUserDefinedWithContent(t *testing.T) {
	mr := value.NewMixinRule("m", nil, []value.Statement{
		value.NewContentRule(nil, nil),
	}, nil, nil)
	udc := NewUserDefinedCallable(mr, nil, false)
	mixin := value.NewSassMixin(udc)
	got := selEvalOK(t, mathBIC(t, acceptsContentFunction()), mixin)
	assertBool(t, got, true)
}

func TestAcceptsContentUserDefinedWithoutContent(t *testing.T) {
	mr := value.NewMixinRule("m", nil, nil, nil, nil)
	udc := NewUserDefinedCallable(mr, nil, false)
	mixin := value.NewSassMixin(udc)
	got := selEvalOK(t, mathBIC(t, acceptsContentFunction()), mixin)
	assertBool(t, got, false)
}

func TestAcceptsContentUserDefinedNotMixinPanics(t *testing.T) {
	// Matches Dart: a UserDefinedCallable whose declaration is not a MixinRule
	// throws UnsupportedError. Go panics via the default branch.
	fr := value.NewFunctionRule("m", nil, nil, nil, nil)
	udc := NewUserDefinedCallable(fr, nil, false)
	mixin := value.NewSassMixin(udc)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-mixin UserDefinedCallable")
		}
	}()
	selEvalOK(t, mathBIC(t, acceptsContentFunction()), mixin)
}

func TestAcceptsContentTypeError(t *testing.T) {
	_, err, _ := selEval(t, mathBIC(t, acceptsContentFunction()), num(1))
	assertScriptErr(t, err, "1 is not a mixin reference.", "mixin")
}

func TestAcceptsContentStringError(t *testing.T) {
	_, err, _ := selEval(t, mathBIC(t, acceptsContentFunction()), str("m"))
	assertScriptErr(t, err, "m is not a mixin reference.", "mixin")
}
