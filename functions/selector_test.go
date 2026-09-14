package functions

import (
	"errors"
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// AssertNotBogus routes deprecation warnings through the EvaluationContext,
// which must satisfy value.WarnLogger.
var _ value.WarnLogger = (*evalcontext.EvaluationContext)(nil)

// --- helpers ---

// selEval calls fn with a recording logger and returns the result, error, and
// recorded warnings. Selector functions need a logger-backed ec for
// AssertNotBogus warnings and parse deprecations.
func selEval(t *testing.T, fn *BuiltInCallable, args ...value.Value) (value.Value, error, *recordingLogger) {
	t.Helper()
	return mathEvalWarn(t, fn, args...)
}

func selEvalOK(t *testing.T, fn *BuiltInCallable, args ...value.Value) value.Value {
	t.Helper()
	got, err, _ := selEval(t, fn, args...)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func selSlashList(t *testing.T, items ...value.Value) *value.SassList {
	t.Helper()
	l, err := value.NewSassList(items, value.ListSeparatorSlash, false)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func selArgList(t *testing.T, sep value.ListSeparator, items ...value.Value) *value.SassArgumentList {
	t.Helper()
	al, err := value.NewSassArgumentList(items, nil, sep)
	if err != nil {
		t.Fatal(err)
	}
	return al
}

// assertScriptErr asserts err is a *SassScriptException with the exact
// message and argument name.
func assertScriptErr(t *testing.T, err error, wantMsg, wantArgName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", wantMsg)
	}
	sse, ok := errors.AsType[*sasscommon.SassScriptException](err)
	if !ok {
		t.Fatalf("expected *SassScriptException, got %T: %v", err, err)
	}
	if sse.Message != wantMsg {
		t.Errorf("Message = %q, want %q", sse.Message, wantMsg)
	}
	if sse.ArgumentName != wantArgName {
		t.Errorf("ArgumentName = %q, want %q", sse.ArgumentName, wantArgName)
	}
}

func notValidSelectorMsg(v string) string {
	return strings.Join([]string{
		v + ` is not a valid selector: it must be a string,`,
		`a list of strings, or a list of lists of strings.`,
	}, "\n")
}

func bogusWarningMsg(prefix, selector string) string {
	return strings.Join([]string{
		prefix + selector + ` is not valid CSS.`,
		`This will be an error in Dart Sass 2.0.0.`,
		``,
		`More info: https://sass-lang.com/d/bogus-combinators`,
	}, "\n")
}

func globalBuiltinWarningMsg(name string) string {
	return strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use selector.` + name + ` instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
}

// --- GlobalSelectorFunctions ---

func TestGlobalSelectorFunctionsNames(t *testing.T) {
	fns := GlobalSelectorFunctions()
	want := []string{
		"is-superselector", "simple-selectors", "selector-parse", "selector-nest",
		"selector-append", "selector-extend", "selector-replace", "selector-unify",
	}
	if len(fns) != len(want) {
		t.Fatalf("len = %d, want %d", len(fns), len(want))
	}
	for i, name := range want {
		if fns[i].Name() != name {
			t.Errorf("fns[%d].Name() = %q, want %q", i, fns[i].Name(), name)
		}
	}
}

func TestGlobalSelectorParseEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	parse := mathBIC(t, fns[2])
	got, err, rl := selEval(t, parse, str("c"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(c,)")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("parse"), deprecation.GlobalBuiltin)
}

func TestGlobalIsSuperselectorEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	isSuper := mathBIC(t, fns[0])
	got, err, rl := selEval(t, isSuper, str(".c"), str(".c.d"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("is-superselector"), deprecation.GlobalBuiltin)
}

func TestGlobalSimpleSelectorsEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	simple := mathBIC(t, fns[1])
	got, err, rl := selEval(t, simple, str(".foo.bar"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, ".foo, .bar")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("simple-selectors"), deprecation.GlobalBuiltin)
}

func TestGlobalSelectorNestEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	nest := mathBIC(t, fns[3])
	got, err, rl := selEval(t, nest, commaList(str("c"), str("d")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(c d,)")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("nest"), deprecation.GlobalBuiltin)
}

func TestGlobalSelectorAppendEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	appendFn := mathBIC(t, fns[4])
	got, err, rl := selEval(t, appendFn, commaList(str(".c"), str(".d")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(.c.d,)")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("append"), deprecation.GlobalBuiltin)
}

func TestGlobalSelectorExtendEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	extendFn := mathBIC(t, fns[5])
	got, err, rl := selEval(t, extendFn, str("c"), str("c"), str("e"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "c, e")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("extend"), deprecation.GlobalBuiltin)
}

func TestGlobalSelectorReplaceEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	replaceFn := mathBIC(t, fns[6])
	got, err, rl := selEval(t, replaceFn, str("c"), str("c"), str("d"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(d,)")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("replace"), deprecation.GlobalBuiltin)
}

func TestGlobalSelectorUnifyEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalSelectorFunctions()
	unifyFn := mathBIC(t, fns[7])
	got, err, rl := selEval(t, unifyFn, str(".c"), str(".d"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(.c.d,)")
	assertSingleWarning(t, rl, globalBuiltinWarningMsg("unify"), deprecation.GlobalBuiltin)
}

// --- SelectorModule ---

func TestSelectorModuleURL(t *testing.T) {
	m := SelectorModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:selector" {
		t.Errorf("URL = %q, want %q", url, "sass:selector")
	}
}

func TestSelectorModuleFunctions(t *testing.T) {
	m := SelectorModule()
	want := []string{
		"is-superselector", "simple-selectors", "parse", "nest",
		"append", "extend", "replace", "unify",
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

func TestSelectorModuleNoMixinsNoVariables(t *testing.T) {
	m := SelectorModule()
	if m.Mixins().Len() != 0 {
		t.Errorf("mixins = %d, want 0", m.Mixins().Len())
	}
	if m.Variables().Len() != 0 {
		t.Errorf("variables = %d, want 0", m.Variables().Len())
	}
}

func TestSelectorModuleParseEmitsNoWarnings(t *testing.T) {
	got, err, rl := selEval(t, selectorParseModuleFunction(), str("c"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(c,)")
	assertNoWarnings(t, rl)
}

// --- parse ---

func TestSelectorParseString(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), str("c d, e f"))
	assertInspect(t, got, "c d, e f")
}

func TestSelectorParseSingle(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), str("c"))
	assertInspect(t, got, "(c,)")
}

func TestSelectorParseQuotedString(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), quoted("c d"))
	assertInspect(t, got, "(c d,)")
}

func TestSelectorParseCommaList(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), commaList(str("c"), str("d e")))
	assertInspect(t, got, "c, d e")
}

func TestSelectorParseSpaceList(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), spaceList(str(".c"), str(".d")))
	assertInspect(t, got, "(.c .d,)")
}

func TestSelectorParseNestedList(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(),
		commaList(spaceList(str("c"), str("d")), str("e")))
	assertInspect(t, got, "c d, e")
}

func TestSelectorParseArgumentList(t *testing.T) {
	// Matches Dart: SassArgumentList extends SassList, so an argument list is
	// a valid selector structure.
	got := selEvalOK(t, selectorParseModuleFunction(),
		selArgList(t, value.ListSeparatorComma, str("c"), str("d")))
	assertInspect(t, got, "c, d")
}

func TestSelectorParseNestedArgumentList(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(),
		commaList(selArgList(t, value.ListSeparatorSpace, str("c"), str("d")), str("e")))
	assertInspect(t, got, "c d, e")
}

func TestSelectorParseStructure(t *testing.T) {
	got := selEvalOK(t, selectorParseModuleFunction(), str("c d, e"))
	if got.Separator() != value.ListSeparatorComma {
		t.Errorf("outer separator = %v, want comma", got.Separator())
	}
	if got.HasBrackets() {
		t.Error("outer list should not have brackets")
	}
	items, err := got.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("outer length = %d, want 2", len(items))
	}
	inner, ok := items[0].(*value.SassList)
	if !ok {
		t.Fatalf("inner = %T, want *SassList", items[0])
	}
	if inner.Separator() != value.ListSeparatorSpace {
		t.Errorf("inner separator = %v, want space", inner.Separator())
	}
	innerItems, err := inner.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(innerItems) != 2 {
		t.Fatalf("inner length = %d, want 2", len(innerItems))
	}
	s, ok := innerItems[0].(*value.SassString)
	if !ok {
		t.Fatalf("element = %T, want *SassString", innerItems[0])
	}
	if s.Text != "c" || s.HasQuotes {
		t.Errorf("element = %q (quotes %v), want unquoted %q", s.Text, s.HasQuotes, "c")
	}
}

// --- parse errors (selectorString) ---

func TestSelectorParseNumberError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector")
}

func TestSelectorParseNullError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), value.Null)
	assertScriptErr(t, err, notValidSelectorMsg("null"), "selector")
}

func TestSelectorParseBooleanError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), value.SassTrue)
	assertScriptErr(t, err, notValidSelectorMsg("true"), "selector")
}

func TestSelectorParseMapError(t *testing.T) {
	m := mkMap(mkStr("c"), mkStr("d"))
	_, err, _ := selEval(t, selectorParseModuleFunction(), m)
	assertScriptErr(t, err, notValidSelectorMsg("(c: d)"), "selector")
}

func TestSelectorParseEmptyListError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), commaList())
	assertScriptErr(t, err, notValidSelectorMsg("()"), "selector")
}

func TestSelectorParseSlashListError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), selSlashList(t, str("c"), str("d")))
	assertScriptErr(t, err, notValidSelectorMsg("(c / d)"), "selector")
}

func TestSelectorParseNestedCommaListError(t *testing.T) {
	// Matches sass-spec: core_functions/selector/parse/error inner_comma.
	_, err, _ := selEval(t, selectorParseModuleFunction(), commaList(commaList(str("c"))))
	assertScriptErr(t, err, notValidSelectorMsg("((c,),)"), "selector")
}

func TestSelectorParseTooNestedError(t *testing.T) {
	// Matches sass-spec: core_functions/selector/parse/error too_nested.
	_, err, _ := selEval(t, selectorParseModuleFunction(),
		commaList(spaceList(spaceList(str("c")))))
	assertScriptErr(t, err, notValidSelectorMsg("(c,)"), "selector")
}

func TestSelectorParseOuterSpaceError(t *testing.T) {
	// Matches sass-spec: core_functions/selector/parse/error outer_space.
	_, err, _ := selEval(t, selectorParseModuleFunction(), spaceList(spaceList(str("c"))))
	assertScriptErr(t, err, notValidSelectorMsg("(c)"), "selector")
}

func TestSelectorParseCommaListNumberError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), commaList(str("c"), num(1)))
	assertScriptErr(t, err, notValidSelectorMsg("(c, 1)"), "selector")
}

func TestSelectorParseSpaceListNumberError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), spaceList(str("c"), num(1)))
	assertScriptErr(t, err, notValidSelectorMsg("(c 1)"), "selector")
}

func TestSelectorParseNestedSlashListError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(),
		commaList(str("c"), selSlashList(t, str("d"), str("e"))))
	assertScriptErr(t, err, notValidSelectorMsg("(c, d / e)"), "selector")
}

func TestSelectorParseParentError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), str("&"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector")
}

func TestSelectorParseInvalidError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), str("[c"))
	assertScriptErr(t, err, "expected more input.", "selector")
}

func TestSelectorParseExtraError(t *testing.T) {
	_, err, _ := selEval(t, selectorParseModuleFunction(), str("c {"))
	assertScriptErr(t, err, "expected selector.", "selector")
}

// --- is-superselector ---

func TestIsSuperselectorTrue(t *testing.T) {
	got, err, rl := selEval(t, isSuperselectorFunction(), str(".c"), str(".c.d"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, true)
	assertNoWarnings(t, rl)
}

func TestIsSuperselectorFalse(t *testing.T) {
	got := selEvalOK(t, isSuperselectorFunction(), str(".c.d"), str(".c"))
	assertBool(t, got, false)
}

func TestIsSuperselectorEqual(t *testing.T) {
	got := selEvalOK(t, isSuperselectorFunction(), str("c"), str("c"))
	assertBool(t, got, true)
}

func TestIsSuperselectorList(t *testing.T) {
	got := selEvalOK(t, isSuperselectorFunction(), str("c, d"), str("c"))
	assertBool(t, got, true)
}

func TestIsSuperselectorBogusSuper(t *testing.T) {
	got, err, rl := selEval(t, isSuperselectorFunction(), str("> c"), str("c"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
	assertSingleWarning(t, rl, bogusWarningMsg("$super: ", "> c"), deprecation.BogusCombinators)
}

func TestIsSuperselectorBogusSub(t *testing.T) {
	got, err, rl := selEval(t, isSuperselectorFunction(), str("c"), str("d + ~ c"))
	if err != nil {
		t.Fatal(err)
	}
	assertBool(t, got, false)
	assertSingleWarning(t, rl, bogusWarningMsg("$sub: ", "d + ~ c"), deprecation.BogusCombinators)
}

func TestIsSuperselectorSuperTypeError(t *testing.T) {
	_, err, _ := selEval(t, isSuperselectorFunction(), num(1), str("c"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "super")
}

func TestIsSuperselectorSubTypeError(t *testing.T) {
	_, err, _ := selEval(t, isSuperselectorFunction(), str("c"), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "sub")
}

func TestIsSuperselectorSuperParseError(t *testing.T) {
	_, err, _ := selEval(t, isSuperselectorFunction(), str("[c"), str("d"))
	assertScriptErr(t, err, "expected more input.", "super")
}

func TestIsSuperselectorSubParentError(t *testing.T) {
	_, err, _ := selEval(t, isSuperselectorFunction(), str("c"), str("&"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "sub")
}

// --- simple-selectors ---

func TestSimpleSelectors(t *testing.T) {
	got, err, rl := selEval(t, simpleSelectorsFunction(), str(".foo.bar"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, ".foo, .bar")
	assertNoWarnings(t, rl)
}

func TestSimpleSelectorsTriple(t *testing.T) {
	got := selEvalOK(t, simpleSelectorsFunction(), str(".foo.bar.baz"))
	assertInspect(t, got, ".foo, .bar, .baz")
}

func TestSimpleSelectorsPseudo(t *testing.T) {
	got := selEvalOK(t, simpleSelectorsFunction(), str("c:hover"))
	assertInspect(t, got, "c, :hover")
}

func TestSimpleSelectorsSingle(t *testing.T) {
	got := selEvalOK(t, simpleSelectorsFunction(), str("c"))
	assertInspect(t, got, "(c,)")
}

func TestSimpleSelectorsSeparator(t *testing.T) {
	got := selEvalOK(t, simpleSelectorsFunction(), str(".foo.bar"))
	if got.Separator() != value.ListSeparatorComma {
		t.Errorf("separator = %v, want comma", got.Separator())
	}
	if got.HasBrackets() {
		t.Error("list should not have brackets")
	}
}

func TestSimpleSelectorsMultipleComplexError(t *testing.T) {
	_, err, _ := selEval(t, simpleSelectorsFunction(), str("c, d"))
	assertScriptErr(t, err, "Selector c, d must be a compound selector.", "selector")
}

func TestSimpleSelectorsCombinatorError(t *testing.T) {
	_, err, _ := selEval(t, simpleSelectorsFunction(), str("c > d"))
	assertScriptErr(t, err, "Selector c > d must be a compound selector.", "selector")
}

func TestSimpleSelectorsDescendantError(t *testing.T) {
	_, err, _ := selEval(t, simpleSelectorsFunction(), str("c d"))
	assertScriptErr(t, err, "Selector c d must be a compound selector.", "selector")
}

func TestSimpleSelectorsParseError(t *testing.T) {
	_, err, _ := selEval(t, simpleSelectorsFunction(), str("[c"))
	assertScriptErr(t, err, "expected more input.", "selector")
}

func TestSimpleSelectorsTypeError(t *testing.T) {
	_, err, _ := selEval(t, simpleSelectorsFunction(), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector")
}

func TestSimpleSelectorsParentError(t *testing.T) {
	// assertCompoundSelector always disallows parent selectors.
	_, err, _ := selEval(t, simpleSelectorsFunction(), str("&.c"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector")
}

// --- nest ---

func TestNestSingle(t *testing.T) {
	got, err, rl := selEval(t, selectorNestModuleFunction(), commaList(str("c")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(c,)")
	assertNoWarnings(t, rl)
}

func TestNestMany(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(),
		commaList(str("c"), str("d"), str("e"), str("f"), str("g")))
	assertInspect(t, got, "(c d e f g,)")
}

func TestNestList(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c, d"), str("e")))
	assertInspect(t, got, "c e, d e")
}

func TestNestParentAlone(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("&")))
	assertInspect(t, got, "(&,)")
}

func TestNestParentAloneSecond(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c"), str("&")))
	assertInspect(t, got, "(c,)")
}

func TestNestParentCompound(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c"), str("&.d")))
	assertInspect(t, got, "(c.d,)")
}

func TestNestParentSuffix(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c"), str("&d")))
	assertInspect(t, got, "(cd,)")
}

func TestNestComplexParent(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c d"), str("e &.f")))
	assertInspect(t, got, "(e c d.f,)")
}

func TestNestSelectorPseudo(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c"), str(":is(&)")))
	assertInspect(t, got, "(:is(c),)")
}

func TestNestSelectorPseudoAlone(t *testing.T) {
	// A parent selector inside a selector pseudo survives a nil parent.
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str(":is(&)")))
	assertInspect(t, got, "(:is(&),)")
}

func TestNestSelectorPseudoSuffixError(t *testing.T) {
	// The parent-with-suffix check recurses into selector pseudos.
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(str(":is(&c)")))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	se, ok := errors.AsType[*sasscommon.SassException](err)
	if !ok {
		t.Fatalf("expected *SassException, got %T: %v", err, err)
	}
	want := "A top-level selector may not contain a parent selector with a suffix."
	if se.Message != want {
		t.Errorf("Message = %q, want %q", se.Message, want)
	}
}

func TestNestLeadingCombinator(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("> c"), str("d")))
	assertInspect(t, got, "(> c d,)")
}

func TestNestTrailingCombinator(t *testing.T) {
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c ~"), str("d")))
	assertInspect(t, got, "(c ~ d,)")
}

func TestNestSelectorPseudoNonInitialSimple(t *testing.T) {
	// The pseudo is not the first simple in its compound; the parser-level
	// allowParent still applies to the pseudo's inner selector list.
	got := selEvalOK(t, selectorNestModuleFunction(), commaList(str("c"), str("d:is(&)")))
	assertInspect(t, got, "(d:is(c),)")
}

func TestSelectorParsePseudoParentError(t *testing.T) {
	// allowParent=false also applies inside selector pseudos.
	_, err, _ := selEval(t, selectorParseModuleFunction(), str(":is(&)"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector")
}

func TestNestEmptyError(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList())
	assertScriptErr(t, err, "$selectors: At least one selector must be passed.", "")
}

func TestNestSuffixParentError(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(str("&c")))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	se, ok := errors.AsType[*sasscommon.SassException](err)
	if !ok {
		t.Fatalf("expected *SassException, got %T: %v", err, err)
	}
	want := "A top-level selector may not contain a parent selector with a suffix."
	if se.Message != want {
		t.Errorf("Message = %q, want %q", se.Message, want)
	}
}

func TestNestNonInitialParentError(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(str("c"), str("[d]&")))
	assertScriptErr(t, err, `"&" may only used at the beginning of a compound selector.`, "")
}

func TestNestTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(num(1)))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "")
}

func TestNestTypeErrorLater(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(str("c"), num(1)))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "")
}

func TestNestInvalidError(t *testing.T) {
	_, err, _ := selEval(t, selectorNestModuleFunction(), commaList(str("[c")))
	assertScriptErr(t, err, "expected more input.", "")
}

// --- append ---

func TestAppendClasses(t *testing.T) {
	got, err, rl := selEval(t, selectorAppendModuleFunction(), commaList(str(".c"), str(".d")))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(.c.d,)")
	assertNoWarnings(t, rl)
}

func TestAppendClassesDouble(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str(".c, .d"), str(".e, .f")))
	assertInspect(t, got, ".c.e, .c.f, .d.e, .d.f")
}

func TestAppendSuffix(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str(".c"), str("d")))
	assertInspect(t, got, "(.cd,)")
}

func TestAppendSuffixMultiple(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str(".c, .d"), str("e, f")))
	assertInspect(t, got, ".ce, .cf, .de, .df")
}

func TestAppendDescendant(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str("c d"), str("e f")))
	assertInspect(t, got, "(c de f,)")
}

func TestAppendOneArg(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str(".c.d")))
	assertInspect(t, got, "(.c.d,)")
}

func TestAppendManyArgs(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str(".c"), str(".d"), str(".e")))
	assertInspect(t, got, "(.c.d.e,)")
}

func TestAppendListFormat(t *testing.T) {
	// Matches sass-spec: format/input/initial — parsed-format input.
	got := selEvalOK(t, selectorAppendModuleFunction(),
		commaList(commaList(str("c"), str("d e")), str("f")))
	assertInspect(t, got, "cf, d ef")
}

func TestAppendLeadingCombinatorParent(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str("> c"), str("d")))
	assertInspect(t, got, "(> cd,)")
}

func TestAppendTrailingCombinatorChild(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str("c"), str("d ~")))
	assertInspect(t, got, "(cd ~,)")
}

func TestAppendMiddleCombinators(t *testing.T) {
	got := selEvalOK(t, selectorAppendModuleFunction(), commaList(str("c > > d"), str("e")))
	assertInspect(t, got, "(c > > de,)")
}

func TestAppendUniversalError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(str(".c"), str("*")))
	assertScriptErr(t, err, "Can't append * to .c.", "")
}

func TestAppendLeadingCombinatorError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(str(".c"), str("> .d")))
	assertScriptErr(t, err, "Can't append > .d to .c.", "")
}

func TestAppendCombinatorOnlyError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(),
		commaList(str(".c"), str(">"), str(".d")))
	assertScriptErr(t, err, "Can't append > to .c.", "")
}

func TestAppendNamespaceError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(str("c"), str("|d")))
	assertScriptErr(t, err, "Can't append |d to c.", "")
}

func TestAppendMultipleTargetsError(t *testing.T) {
	// The result string in the error lists all components of the accumulated selector.
	_, err, _ := selEval(t, selectorAppendModuleFunction(),
		commaList(str(".c, .d"), str("*")))
	assertScriptErr(t, err, "Can't append * to .c, .d.", "")
}

func TestAppendTrailingCombinatorParentError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(str(".c ~"), str(".d")))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	mse, ok := errors.AsType[*sasscommon.MultiSpanSassException](err)
	if !ok {
		t.Fatalf("expected *MultiSpanSassException, got %T: %v", err, err)
	}
	want := `Selector ".c ~" can't be used as a parent in a compound selector.`
	if mse.Message != want {
		t.Errorf("Message = %q, want %q", mse.Message, want)
	}
	if mse.PrimaryLabel != "outer selector" {
		t.Errorf("PrimaryLabel = %q, want %q", mse.PrimaryLabel, "outer selector")
	}
}

func TestAppendEmptyError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList())
	assertScriptErr(t, err, "$selectors: At least one selector must be passed.", "")
}

func TestAppendTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(num(1)))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "")
}

func TestAppendParentError(t *testing.T) {
	// append parses with allowParent=false.
	_, err, _ := selEval(t, selectorAppendModuleFunction(), commaList(str("&"), str("c")))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "")
}

// --- extend ---

func TestExtendEqual(t *testing.T) {
	got, err, rl := selEval(t, selectorExtendModuleFunction(), str("c"), str("c"), str("e"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "c, e")
	assertNoWarnings(t, rl)
}

func TestExtendUnequal(t *testing.T) {
	got := selEvalOK(t, selectorExtendModuleFunction(), str("c"), str("d"), str("e"))
	assertInspect(t, got, "(c,)")
}

func TestExtendCompound(t *testing.T) {
	got := selEvalOK(t, selectorExtendModuleFunction(), str(".c.d"), str(".c"), str(".e"))
	assertInspect(t, got, ".c.d, .d.e")
}

func TestExtendParent(t *testing.T) {
	got := selEvalOK(t, selectorExtendModuleFunction(), str(".c .d"), str(".c"), str(".e"))
	assertInspect(t, got, ".c .d, .e .d")
}

func TestExtendListExtender(t *testing.T) {
	got := selEvalOK(t, selectorExtendModuleFunction(), str(".c .d"), str(".c"), str(".e, .f"))
	assertInspect(t, got, ".c .d, .e .d, .f .d")
}

func TestExtendBogusSelectorWarning(t *testing.T) {
	got, err, rl := selEval(t, selectorExtendModuleFunction(), str("> .c"), str(".c"), str(".d"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "> .c, > .d")
	assertSingleWarning(t, rl, bogusWarningMsg("$selector: ", "> .c"), deprecation.BogusCombinators)
}

func TestExtendSelectorTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorExtendModuleFunction(), num(1), str("c"), str("d"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector")
}

func TestExtendExtendeeTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorExtendModuleFunction(), str("c"), num(1), str("d"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "extendee")
}

func TestExtendExtenderTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorExtendModuleFunction(), str("c"), str("d"), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "extender")
}

func TestExtendParentSelectorError(t *testing.T) {
	_, err, _ := selEval(t, selectorExtendModuleFunction(), str("&"), str("c"), str("d"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector")
}

func TestExtendExtenderParseError(t *testing.T) {
	_, err, _ := selEval(t, selectorExtendModuleFunction(), str("c"), str("d"), str("[e"))
	assertScriptErr(t, err, "expected more input.", "extender")
}

// --- replace ---

func TestReplaceSimple(t *testing.T) {
	got, err, rl := selEval(t, selectorReplaceModuleFunction(), str("c"), str("c"), str("d"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(d,)")
	assertNoWarnings(t, rl)
}

func TestReplaceCompound(t *testing.T) {
	got := selEvalOK(t, selectorReplaceModuleFunction(), str("c.d"), str("c"), str("e"))
	assertInspect(t, got, "(e.d,)")
}

func TestReplaceComplex(t *testing.T) {
	got := selEvalOK(t, selectorReplaceModuleFunction(), str("c d"), str("d"), str("e f"))
	assertInspect(t, got, "c e f, e c f")
}

func TestReplaceSelectorPseudo(t *testing.T) {
	got := selEvalOK(t, selectorReplaceModuleFunction(), str(":is(c)"), str("c"), str("d"))
	assertInspect(t, got, "(:is(d),)")
}

func TestReplaceNoOp(t *testing.T) {
	got := selEvalOK(t, selectorReplaceModuleFunction(), str("c"), str("d"), str("e"))
	assertInspect(t, got, "(c,)")
}

func TestReplaceSelectorTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorReplaceModuleFunction(), num(1), str("c"), str("d"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector")
}

func TestReplaceOriginalTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorReplaceModuleFunction(), str("c"), num(1), str("d"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "original")
}

func TestReplaceReplacementTypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorReplaceModuleFunction(), str("c"), str("d"), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "replacement")
}

func TestReplaceParentSelectorError(t *testing.T) {
	_, err, _ := selEval(t, selectorReplaceModuleFunction(), str("&"), str("c"), str("d"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector")
}

// --- unify ---

func TestUnifySame(t *testing.T) {
	got, err, rl := selEval(t, selectorUnifyModuleFunction(), str(".c"), str(".c"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(.c,)")
	assertNoWarnings(t, rl)
}

func TestUnifyDifferentClasses(t *testing.T) {
	got := selEvalOK(t, selectorUnifyModuleFunction(), str(".c"), str(".d"))
	assertInspect(t, got, "(.c.d,)")
}

func TestUnifyIdsNull(t *testing.T) {
	got := selEvalOK(t, selectorUnifyModuleFunction(), str("#c"), str("#d"))
	assertNull(t, got)
}

func TestUnifyDescendantNull(t *testing.T) {
	got := selEvalOK(t, selectorUnifyModuleFunction(), str("c d"), str("e f"))
	assertNull(t, got)
}

func TestUnifyDescendantSame(t *testing.T) {
	got := selEvalOK(t, selectorUnifyModuleFunction(), str("c d"), str("c d"))
	assertInspect(t, got, "(c d,)")
}

func TestUnifyBogusSelector1Warning(t *testing.T) {
	got, err, rl := selEval(t, selectorUnifyModuleFunction(), str("c ~"), str("c"))
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(c ~,)")
	assertSingleWarning(t, rl, bogusWarningMsg("$selector1: ", "c ~"), deprecation.BogusCombinators)
}

func TestUnifySelector1TypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorUnifyModuleFunction(), num(1), str("c"))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector1")
}

func TestUnifySelector2TypeError(t *testing.T) {
	_, err, _ := selEval(t, selectorUnifyModuleFunction(), str("c"), num(1))
	assertScriptErr(t, err, notValidSelectorMsg("1"), "selector2")
}

func TestUnifySelector1ParentError(t *testing.T) {
	_, err, _ := selEval(t, selectorUnifyModuleFunction(), str("&"), str("c"))
	assertScriptErr(t, err, "Parent selectors aren't allowed here.", "selector1")
}
