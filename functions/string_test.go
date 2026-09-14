package functions

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/value"
)

// --- helpers ---

func stringGlobalBuiltinWarningMsg(name string) string {
	return strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use string.` + name + ` instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
}

// negOne is the evaluated default for $end-at: -1. The test harness pads
// missing arguments with Null, but the real evaluator always passes the
// evaluated default expression.
func negOne() value.Value { return value.NewUnitlessNumber(-1) }

// --- GlobalStringFunctions ---

func TestGlobalStringFunctionsNames(t *testing.T) {
	fns := GlobalStringFunctions()
	want := []string{
		"unquote", "quote", "to-upper-case", "to-lower-case", "unique-id",
		"str-length", "str-insert", "str-index", "str-slice",
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

func TestGlobalUnquoteEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	unquote := mathBIC(t, fns[0])
	got, err, rl := selEval(t, unquote, quoted("foo"))
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "foo")
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("unquote"), deprecation.GlobalBuiltin)
}

func TestGlobalStrLengthEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	strLength := mathBIC(t, fns[5])
	got, err, rl := selEval(t, strLength, str("abcd"))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, got, 4)
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("length"), deprecation.GlobalBuiltin)
}

func TestGlobalStrInsertEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	strInsert := mathBIC(t, fns[6])
	got, err, rl := selEval(t, strInsert, str("abcd"), str("X"), num(1))
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "Xabcd")
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("insert"), deprecation.GlobalBuiltin)
}

func TestGlobalStrIndexEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	strIndex := mathBIC(t, fns[7])
	got, err, rl := selEval(t, strIndex, str("abcd"), str("bc"))
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, got, 2)
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("index"), deprecation.GlobalBuiltin)
}

func TestGlobalStrSliceEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	strSlice := mathBIC(t, fns[8])
	got, err, rl := selEval(t, strSlice, str("abcd"), num(2), negOne())
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "bcd")
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("slice"), deprecation.GlobalBuiltin)
}

func TestGlobalUniqueIDEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalStringFunctions()
	uniqueID := mathBIC(t, fns[4])
	_, err, rl := selEval(t, uniqueID)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleWarning(t, rl, stringGlobalBuiltinWarningMsg("unique-id"), deprecation.GlobalBuiltin)
}

// --- StringModule ---

func TestStringModuleURL(t *testing.T) {
	m := StringModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:string" {
		t.Errorf("URL = %q, want %q", url, "sass:string")
	}
}

func TestStringModuleFunctions(t *testing.T) {
	m := StringModule()
	want := []string{
		"unquote", "quote", "to-upper-case", "to-lower-case", "unique-id",
		"length", "insert", "index", "slice", "split",
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

func TestStringModuleNoMixinsNoVariables(t *testing.T) {
	m := StringModule()
	if m.Mixins().Len() != 0 {
		t.Errorf("mixins = %d, want 0", m.Mixins().Len())
	}
	if m.Variables().Len() != 0 {
		t.Errorf("variables = %d, want 0", m.Variables().Len())
	}
}

// --- unquote ---

func TestUnquoteQuoted(t *testing.T) {
	got, err, rl := selEval(t, unquoteFunction(), quoted("foo"))
	if err != nil {
		t.Fatal(err)
	}
	assertUnquotedString(t, got, "foo")
	assertNoWarnings(t, rl)
}

func TestUnquoteUnquotedReturnsSameValue(t *testing.T) {
	input := str("foo")
	got := selEvalOK(t, unquoteFunction(), input)
	if got != input {
		t.Error("unquote of an unquoted string should return the same value")
	}
}

func TestUnquoteEmpty(t *testing.T) {
	got := selEvalOK(t, unquoteFunction(), quoted(""))
	assertUnquotedString(t, got, "")
}

func TestUnquoteTypeError(t *testing.T) {
	_, err, _ := selEval(t, unquoteFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

// --- quote ---

func TestQuoteUnquoted(t *testing.T) {
	got := selEvalOK(t, quoteFunction(), str("foo"))
	assertQuotedString(t, got, "foo")
}

func TestQuoteQuotedReturnsSameValue(t *testing.T) {
	input := quoted("foo")
	got := selEvalOK(t, quoteFunction(), input)
	if got != input {
		t.Error("quote of a quoted string should return the same value")
	}
}

func TestQuoteTypeError(t *testing.T) {
	_, err, _ := selEval(t, quoteFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

// --- to-upper-case / to-lower-case ---

func TestToUpperCase(t *testing.T) {
	got := selEvalOK(t, toUpperCaseFunction(), str("aBc123"))
	assertUnquotedString(t, got, "ABC123")
}

func TestToUpperCaseAsciiOnly(t *testing.T) {
	// Only ASCII a-z are uppercased, matching Dart's toUpperCase(codeUnit).
	got := selEvalOK(t, toUpperCaseFunction(), str("aBc123äö"))
	assertUnquotedString(t, got, "ABC123äö")
}

func TestToUpperCaseKeepsQuotes(t *testing.T) {
	got := selEvalOK(t, toUpperCaseFunction(), quoted("abc"))
	assertQuotedString(t, got, "ABC")
}

func TestToUpperCaseTypeError(t *testing.T) {
	_, err, _ := selEval(t, toUpperCaseFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

func TestToLowerCase(t *testing.T) {
	got := selEvalOK(t, toLowerCaseFunction(), str("AbC123"))
	assertUnquotedString(t, got, "abc123")
}

func TestToLowerCaseAsciiOnly(t *testing.T) {
	got := selEvalOK(t, toLowerCaseFunction(), str("AbC123ÄÖ"))
	assertUnquotedString(t, got, "abc123ÄÖ")
}

func TestToLowerCaseKeepsQuotes(t *testing.T) {
	got := selEvalOK(t, toLowerCaseFunction(), quoted("ABC"))
	assertQuotedString(t, got, "abc")
}

func TestToLowerCaseTypeError(t *testing.T) {
	_, err, _ := selEval(t, toLowerCaseFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

// --- unique-id ---

func TestUniqueIDFormat(t *testing.T) {
	got := selEvalOK(t, uniqueIDFunction())
	s, ok := got.(*value.SassString)
	if !ok {
		t.Fatalf("expected *SassString, got %T", got)
	}
	if s.HasQuotes {
		t.Error("unique-id should be unquoted")
	}
	if len(s.Text) != 7 {
		t.Fatalf("len = %d (%q), want 7", len(s.Text), s.Text)
	}
	if s.Text[0] != 'u' {
		t.Errorf("prefix = %q, want 'u'", s.Text[0])
	}
	for _, r := range s.Text[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z')) {
			t.Errorf("invalid base-36 char %q in %q", r, s.Text)
		}
	}
}

func TestUniqueIDChanges(t *testing.T) {
	first := selEvalOK(t, uniqueIDFunction()).(*value.SassString).Text
	second := selEvalOK(t, uniqueIDFunction()).(*value.SassString).Text
	if first == second {
		t.Errorf("two unique-id calls returned the same value %q", first)
	}
}

func TestPadLeft(t *testing.T) {
	if got := padLeft("ab", 6, '0'); got != "0000ab" {
		t.Errorf("padLeft = %q, want %q", got, "0000ab")
	}
	if got := padLeft("abcdef", 6, '0'); got != "abcdef" {
		t.Errorf("padLeft = %q, want %q", got, "abcdef")
	}
	if got := padLeft("abcdefg", 6, '0'); got != "abcdefg" {
		t.Errorf("padLeft = %q, want %q", got, "abcdefg")
	}
}

// --- str-length / length ---

func TestStrLength(t *testing.T) {
	got := selEvalOK(t, strLengthFunction(), str("abcd"))
	assertNum(t, got, 4)
}

func TestStrLengthEmpty(t *testing.T) {
	got := selEvalOK(t, strLengthFunction(), str(""))
	assertNum(t, got, 0)
}

func TestStrLengthCodepoints(t *testing.T) {
	// Length counts codepoints, not bytes.
	got := selEvalOK(t, strLengthFunction(), str("a\U0001F46Db"))
	assertNum(t, got, 3)
}

func TestStrLengthTypeError(t *testing.T) {
	_, err, _ := selEval(t, strLengthFunction(), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

// --- str-insert / insert ---

func TestStrInsert(t *testing.T) {
	cases := []struct {
		index float64
		want  string
	}{
		{1, "Xabcd"},
		{3, "abXcd"},
		{5, "abcdX"},
		{100, "abcdX"},
		{0, "Xabcd"},
		{-1, "abcdX"},
		{-2, "abcXd"},
		{-5, "Xabcd"},
		{-100, "Xabcd"},
	}
	for _, tc := range cases {
		got := selEvalOK(t, strInsertFunction(), str("abcd"), str("X"), num(tc.index))
		s, ok := got.(*value.SassString)
		if !ok {
			t.Fatalf("index %v: expected *SassString, got %T", tc.index, got)
		}
		if s.Text != tc.want {
			t.Errorf("insert(abcd, X, %v) = %q, want %q", tc.index, s.Text, tc.want)
		}
	}
}

func TestStrInsertKeepsQuotes(t *testing.T) {
	got := selEvalOK(t, strInsertFunction(), quoted("abcd"), str("X"), num(1))
	assertQuotedString(t, got, "Xabcd")
}

func TestStrInsertCodepoints(t *testing.T) {
	got := selEvalOK(t, strInsertFunction(), str("a\U0001F46Db"), str("X"), num(3))
	assertUnquotedString(t, got, "a\U0001F46DXb")
}

func TestStrInsertStringTypeError(t *testing.T) {
	_, err, _ := selEval(t, strInsertFunction(), num(1), str("X"), num(1))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

func TestStrInsertInsertTypeError(t *testing.T) {
	_, err, _ := selEval(t, strInsertFunction(), str("ab"), num(1), num(1))
	assertScriptErr(t, err, "1 is not a string.", "insert")
}

func TestStrInsertIndexTypeError(t *testing.T) {
	_, err, _ := selEval(t, strInsertFunction(), str("ab"), str("X"), str("x"))
	assertScriptErr(t, err, "x is not a number.", "index")
}

func TestStrInsertIndexUnitsError(t *testing.T) {
	_, err, _ := selEval(t, strInsertFunction(), str("ab"), str("X"), mathUnit(1, "px"))
	assertScriptErr(t, err, "Expected 1px to have no units.", "index")
}

func TestStrInsertIndexNonIntError(t *testing.T) {
	_, err, _ := selEval(t, strInsertFunction(), str("ab"), str("X"), num(1.5))
	assertScriptErr(t, err, "1.5 is not an int.", "index")
}

// --- str-index / index ---

func TestStrIndexFound(t *testing.T) {
	got := selEvalOK(t, strIndexFunction(), str("abcd"), str("bc"))
	assertNum(t, got, 2)
}

func TestStrIndexNotFound(t *testing.T) {
	got := selEvalOK(t, strIndexFunction(), str("abcd"), str("x"))
	assertNull(t, got)
}

func TestStrIndexFirst(t *testing.T) {
	got := selEvalOK(t, strIndexFunction(), str("abcd"), str("a"))
	assertNum(t, got, 1)
}

func TestStrIndexCodepoints(t *testing.T) {
	// The result is a codepoint index, not a byte index.
	got := selEvalOK(t, strIndexFunction(), str("a\U0001F46Dbc"), str("b"))
	assertNum(t, got, 3)
}

func TestStrIndexEmptySubstring(t *testing.T) {
	got := selEvalOK(t, strIndexFunction(), str("abcd"), str(""))
	assertNum(t, got, 1)
}

func TestStrIndexStringTypeError(t *testing.T) {
	_, err, _ := selEval(t, strIndexFunction(), num(1), str("a"))
	assertScriptErr(t, err, "1 is not a string.", "string")
}

func TestStrIndexSubstringTypeError(t *testing.T) {
	_, err, _ := selEval(t, strIndexFunction(), str("ab"), num(1))
	assertScriptErr(t, err, "1 is not a string.", "substring")
}

// --- str-slice / slice ---

func TestStrSlice(t *testing.T) {
	cases := []struct {
		start, end float64
		want       string
	}{
		{2, 3, "bc"},
		{2, -1, "bcd"},
		{-3, -2, "bc"},
		{1, 0, ""},
		{2, 1, ""},
		{2, 10, "bcd"},
		{-100, -1, "abcd"},
		{-100, -100, ""},
		{1, -1, "abcd"},
		{4, 4, "d"},
	}
	for _, tc := range cases {
		got := selEvalOK(t, strSliceFunction(), str("abcd"), num(tc.start), num(tc.end))
		s, ok := got.(*value.SassString)
		if !ok {
			t.Fatalf("slice(abcd, %v, %v): expected *SassString, got %T", tc.start, tc.end, got)
		}
		if s.Text != tc.want {
			t.Errorf("slice(abcd, %v, %v) = %q, want %q", tc.start, tc.end, s.Text, tc.want)
		}
	}
}

func TestStrSliceKeepsQuotes(t *testing.T) {
	got := selEvalOK(t, strSliceFunction(), quoted("abcd"), num(2), num(3))
	assertQuotedString(t, got, "bc")
}

func TestStrSliceEmptyResultKeepsQuotes(t *testing.T) {
	got := selEvalOK(t, strSliceFunction(), quoted("abcd"), num(1), num(0))
	assertQuotedString(t, got, "")
}

func TestStrSliceCodepoints(t *testing.T) {
	got := selEvalOK(t, strSliceFunction(), str("a\U0001F46Dbc"), num(2), num(3))
	assertUnquotedString(t, got, "\U0001F46Db")
}

func TestStrSliceStringTypeError(t *testing.T) {
	_, err, _ := selEval(t, strSliceFunction(), num(1), num(1), negOne())
	assertScriptErr(t, err, "1 is not a string.", "string")
}

func TestStrSliceStartTypeError(t *testing.T) {
	_, err, _ := selEval(t, strSliceFunction(), str("ab"), str("x"), negOne())
	assertScriptErr(t, err, "x is not a number.", "start-at")
}

func TestStrSliceExplicitNullEndError(t *testing.T) {
	// Matches Dart: an explicit null $end-at is a type error — the default -1
	// only comes from the parameter's default expression.
	_, err, _ := selEval(t, strSliceFunction(), str("ab"), num(1), value.Null)
	assertScriptErr(t, err, "null is not a number.", "end-at")
}

func TestStrSliceEndAssertedBeforeStartUnits(t *testing.T) {
	// Matches Dart: both numbers are asserted before either no-units check.
	_, err, _ := selEval(t, strSliceFunction(), str("ab"), mathUnit(1, "px"), value.Null)
	assertScriptErr(t, err, "null is not a number.", "end-at")
}

func TestStrSliceStartUnitsError(t *testing.T) {
	_, err, _ := selEval(t, strSliceFunction(), str("ab"), mathUnit(1, "px"), num(2))
	assertScriptErr(t, err, "Expected 1px to have no units.", "start-at")
}

func TestStrSliceEndUnitsError(t *testing.T) {
	_, err, _ := selEval(t, strSliceFunction(), str("ab"), num(1), mathUnit(2, "px"))
	assertScriptErr(t, err, "Expected 2px to have no units.", "end-at")
}

func TestStrSliceStartNonIntError(t *testing.T) {
	// Matches Dart: start.assertInt() has no argument name...
	_, err, _ := selEval(t, strSliceFunction(), str("abcd"), num(1.5), negOne())
	assertScriptErr(t, err, "1.5 is not an int.", "")
}

func TestStrSliceEndNonIntError(t *testing.T) {
	// ...and neither does end.assertInt(), which runs first.
	_, err, _ := selEval(t, strSliceFunction(), str("abcd"), num(1.5), num(2.5))
	assertScriptErr(t, err, "2.5 is not an int.", "")
}

// --- split ---

func TestSplit(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted("a,b,c"), quoted(","), value.Null)
	assertInspect(t, got, `["a", "b", "c"]`)
	assertListSep(t, got, value.ListSeparatorComma)
	assertListBrackets(t, got, true)
}

func TestSplitUnquoted(t *testing.T) {
	got := selEvalOK(t, splitFunction(), str("a,b,c"), str(","), value.Null)
	assertInspect(t, got, "[a, b, c]")
}

func TestSplitLimit(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted("a,b,c"), quoted(","), num(1))
	assertInspect(t, got, `["a", "b,c"]`)
}

func TestSplitLimitLargerThanChunks(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted("a b"), quoted(" "), num(5))
	assertInspect(t, got, `["a", "b"]`)
}

func TestSplitEmptySeparator(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted("abc"), quoted(""), value.Null)
	assertInspect(t, got, `["a", "b", "c"]`)
}

func TestSplitEmptySeparatorCodepoints(t *testing.T) {
	got := selEvalOK(t, splitFunction(), str("a\U0001F46Db"), str(""), value.Null)
	assertListLen(t, got, 3)
}

func TestSplitEmptyString(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted(""), quoted(","), value.Null)
	assertInspect(t, got, "[]")
	assertListBrackets(t, got, true)
}

func TestSplitTrailingSeparator(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted("a,b,c,"), quoted(","), value.Null)
	assertInspect(t, got, `["a", "b", "c", ""]`)
}

func TestSplitLeadingSeparator(t *testing.T) {
	got := selEvalOK(t, splitFunction(), quoted(",a"), quoted(","), value.Null)
	assertInspect(t, got, `["", "a"]`)
}

func TestSplitSeparatorNotFound(t *testing.T) {
	// A singleton comma-separated list keeps the trailing comma even when
	// bracketed (verified against dart-sass).
	got := selEvalOK(t, splitFunction(), quoted("abc"), quoted("x"), value.Null)
	assertInspect(t, got, `["abc",]`)
}

func TestSplitStringTypeError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), num(1), str(","), value.Null)
	assertScriptErr(t, err, "1 is not a string.", "string")
}

func TestSplitSeparatorTypeError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), str("ab"), num(1), value.Null)
	assertScriptErr(t, err, "1 is not a string.", "separator")
}

func TestSplitLimitZeroError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), str("a,b"), str(","), num(0))
	assertScriptErr(t, err, "Must be 1 or greater, was 0.", "limit")
}

func TestSplitLimitNegativeError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), str("a,b"), str(","), num(-1))
	assertScriptErr(t, err, "Must be 1 or greater, was -1.", "limit")
}

func TestSplitLimitNonIntError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), str("a,b"), str(","), num(1.5))
	assertScriptErr(t, err, "1.5 is not an int.", "limit")
}

func TestSplitLimitTypeError(t *testing.T) {
	_, err, _ := selEval(t, splitFunction(), str("a,b"), str(","), str("x"))
	assertScriptErr(t, err, "x is not a number.", "limit")
}

// --- codepoint helpers ---

func TestCodepointForIndex(t *testing.T) {
	cases := []struct {
		index, length, want int
	}{
		{0, 4, 0},
		{1, 4, 0},
		{4, 4, 3},
		{5, 4, 4},
		{100, 4, 4},
		{-1, 4, 3},
		{-4, 4, 0},
		{-5, 4, 0},
		{-100, 4, 0},
	}
	for _, tc := range cases {
		if got := codepointForIndex(tc.index, tc.length); got != tc.want {
			t.Errorf("codepointForIndex(%d, %d) = %d, want %d", tc.index, tc.length, got, tc.want)
		}
	}
}

func TestCodepointForIndexAllowNegative(t *testing.T) {
	if got := codepointForIndexAllowNegative(-5, 4); got != -1 {
		t.Errorf("codepointForIndexAllowNegative(-5, 4) = %d, want -1", got)
	}
	if got := codepointForIndexAllowNegative(-100, 4); got != -96 {
		t.Errorf("codepointForIndexAllowNegative(-100, 4) = %d, want -96", got)
	}
}

func TestCodepointIndexToCodeUnitIndex(t *testing.T) {
	s := "a\U0001F46Db"
	cases := []struct{ codepoint, want int }{
		{0, 0},
		{1, 1},
		{2, 5},
		{3, 6},
	}
	for _, tc := range cases {
		if got := codepointIndexToCodeUnitIndex(s, tc.codepoint); got != tc.want {
			t.Errorf("codepointIndexToCodeUnitIndex(%q, %d) = %d, want %d", s, tc.codepoint, got, tc.want)
		}
	}
}

func TestCodeUnitIndexToCodepointIndex(t *testing.T) {
	s := "a\U0001F46Db"
	cases := []struct{ codeUnit, want int }{
		{0, 0},
		{1, 1},
		{5, 2},
		{6, 3},
	}
	for _, tc := range cases {
		if got := codeUnitIndexToCodepointIndex(s, tc.codeUnit); got != tc.want {
			t.Errorf("codeUnitIndexToCodepointIndex(%q, %d) = %d, want %d", s, tc.codeUnit, got, tc.want)
		}
	}
}
