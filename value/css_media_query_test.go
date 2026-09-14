package value

import (
	"strings"
	"testing"
)

// ---- Constructor tests ----

func TestNewCssMediaQueryTypeBasic(t *testing.T) {
	typ := "screen"
	q := NewCssMediaQueryType(&typ, nil, nil)

	if q.Modifier != nil {
		t.Error("expected nil modifier")
	}
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", q.Type)
	}
	if !q.Conjunction {
		t.Error("expected conjunction to be true")
	}
	if len(q.Conditions) != 0 {
		t.Errorf("expected empty conditions, got %v", q.Conditions)
	}
}

func TestNewCssMediaQueryTypeWithModifier(t *testing.T) {
	typ := "screen"
	mod := "only"
	q := NewCssMediaQueryType(&typ, &mod, nil)

	if q.Modifier == nil || *q.Modifier != "only" {
		t.Errorf("expected modifier 'only', got %v", q.Modifier)
	}
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", q.Type)
	}
}

func TestNewCssMediaQueryTypeWithConditions(t *testing.T) {
	typ := "screen"
	conditions := []string{"(min-width: 700px)", "(color)"}
	q := NewCssMediaQueryType(&typ, nil, conditions)

	if len(q.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(q.Conditions))
	}
	if q.Conditions[0] != "(min-width: 700px)" {
		t.Errorf("expected first condition '(min-width: 700px)', got %q", q.Conditions[0])
	}
	if q.Conditions[1] != "(color)" {
		t.Errorf("expected second condition '(color)', got %q", q.Conditions[1])
	}
	// Verify the conditions slice is independent
	conditions[0] = "modified"
	if q.Conditions[0] == "modified" {
		t.Error("conditions slice should be independent copy")
	}
}

func TestNewCssMediaQueryConditionSingle(t *testing.T) {
	q, err := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if q.Modifier != nil {
		t.Error("expected nil modifier")
	}
	if q.Type != nil {
		t.Error("expected nil type")
	}
	if !q.Conjunction {
		t.Error("expected conjunction to be true")
	}
	if len(q.Conditions) != 1 || q.Conditions[0] != "(color)" {
		t.Errorf("expected [(color)], got %v", q.Conditions)
	}
}

func TestNewCssMediaQueryConditionMultiWithConjunction(t *testing.T) {
	c := true
	q, err := NewCssMediaQueryCondition([]string{"(color)", "(min-width: 700px)"}, &c)
	if err != nil {
		t.Fatal(err)
	}

	if !q.Conjunction {
		t.Error("expected conjunction to be true")
	}
	if len(q.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(q.Conditions))
	}
}

func TestNewCssMediaQueryConditionMultiDisjunction(t *testing.T) {
	c := false
	q, err := NewCssMediaQueryCondition([]string{"(color)", "(monochrome)"}, &c)
	if err != nil {
		t.Fatal(err)
	}

	if q.Conjunction {
		t.Error("expected conjunction to be false")
	}
}

func TestNewCssMediaQueryConditionMultiNilConjunctionError(t *testing.T) {
	_, err := NewCssMediaQueryCondition([]string{"(a)", "(b)"}, nil)
	if err == nil {
		t.Fatal("expected error for multi-condition with nil conjunction")
	}
	if !strings.Contains(err.Error(), "conjunction may not be null") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- MatchesAllTypes tests ----

func TestMatchesAllTypesNilType(t *testing.T) {
	q, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	if !q.MatchesAllTypes() {
		t.Error("nil type should match all types")
	}
}

func TestMatchesAllTypesAllLower(t *testing.T) {
	typ := "all"
	q := NewCssMediaQueryType(&typ, nil, nil)
	if !q.MatchesAllTypes() {
		t.Error("'all' type should match all types")
	}
}

func TestMatchesAllTypesAllUpper(t *testing.T) {
	typ := "ALL"
	q := NewCssMediaQueryType(&typ, nil, nil)
	if !q.MatchesAllTypes() {
		t.Error("'ALL' type should match all types (case-insensitive)")
	}
}

func TestMatchesAllTypesAllMixed(t *testing.T) {
	typ := "aLl"
	q := NewCssMediaQueryType(&typ, nil, nil)
	if !q.MatchesAllTypes() {
		t.Error("'aLl' type should match all types (case-insensitive)")
	}
}

func TestMatchesAllTypesScreen(t *testing.T) {
	typ := "screen"
	q := NewCssMediaQueryType(&typ, nil, nil)
	if q.MatchesAllTypes() {
		t.Error("'screen' should not match all types")
	}
}

// ---- String tests ----

func TestStringTypeOnly(t *testing.T) {
	typ := "screen"
	q := NewCssMediaQueryType(&typ, nil, nil)
	got := q.String()
	if got != "screen" {
		t.Errorf("String() = %q, want 'screen'", got)
	}
}

func TestStringModifierType(t *testing.T) {
	typ := "screen"
	mod := "only"
	q := NewCssMediaQueryType(&typ, &mod, nil)
	got := q.String()
	if got != "only screen" {
		t.Errorf("String() = %q, want 'only screen'", got)
	}
}

func TestStringTypeAndConditions(t *testing.T) {
	typ := "screen"
	q := NewCssMediaQueryType(&typ, nil, []string{"(min-width: 700px)", "(color)"})
	got := q.String()
	if got != "screen and (min-width: 700px) and (color)" {
		t.Errorf("String() = %q, want 'screen and (min-width: 700px) and (color)'", got)
	}
}

func TestStringModifierTypeAndConditions(t *testing.T) {
	typ := "screen"
	mod := "not"
	q := NewCssMediaQueryType(&typ, &mod, []string{"(color)"})
	got := q.String()
	if got != "not screen and (color)" {
		t.Errorf("String() = %q, want 'not screen and (color)'", got)
	}
}

func TestStringConditionsOnlyAnd(t *testing.T) {
	c := true
	q, _ := NewCssMediaQueryCondition([]string{"(min-width: 700px)", "(color)"}, &c)
	got := q.String()
	if got != "(min-width: 700px) and (color)" {
		t.Errorf("String() = %q, want '(min-width: 700px) and (color)'", got)
	}
}

func TestStringConditionsOnlyOr(t *testing.T) {
	c := false
	q, _ := NewCssMediaQueryCondition([]string{"(color)", "(monochrome)"}, &c)
	got := q.String()
	if got != "(color) or (monochrome)" {
		t.Errorf("String() = %q, want '(color) or (monochrome)'", got)
	}
}

func TestStringSingleCondition(t *testing.T) {
	q, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	got := q.String()
	if got != "(color)" {
		t.Errorf("String() = %q, want '(color)'", got)
	}
}

func TestStringEmpty(t *testing.T) {
	q := &CssMediaQuery{Conjunction: true}
	got := q.String()
	if got != "" {
		t.Errorf("String() = %q, want ''", got)
	}
}

// ---- Equality tests ----

func TestMediaQueriesEqualIdentical(t *testing.T) {
	typ := "screen"
	a := NewCssMediaQueryType(&typ, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&typ, nil, []string{"(color)"})

	if !MediaQueriesEqual(a, b) {
		t.Error("identical queries should be equal")
	}
}

func TestMediaQueriesEqualDifferentType(t *testing.T) {
	screen := "screen"
	print := "print"
	a := NewCssMediaQueryType(&screen, nil, nil)
	b := NewCssMediaQueryType(&print, nil, nil)

	if MediaQueriesEqual(a, b) {
		t.Error("queries with different types should not be equal")
	}
}

func TestMediaQueriesEqualOneNilType(t *testing.T) {
	screen := "screen"
	a := NewCssMediaQueryType(&screen, nil, nil)
	b, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)

	if MediaQueriesEqual(a, b) {
		t.Error("query with type vs nil type should not be equal")
	}
}

func TestMediaQueriesEqualBothNilType(t *testing.T) {
	a, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	b, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)

	if !MediaQueriesEqual(a, b) {
		t.Error("both nil type with same conditions should be equal")
	}
}

func TestMediaQueriesEqualDifferentConditions(t *testing.T) {
	a, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	b, _ := NewCssMediaQueryCondition([]string{"(min-width: 700px)"}, nil)

	if MediaQueriesEqual(a, b) {
		t.Error("different conditions should not be equal")
	}
}

func TestMediaQueriesEqualDifferentConditionOrder(t *testing.T) {
	typ := "screen"
	a := NewCssMediaQueryType(&typ, nil, []string{"(color)", "(grid)"})
	b := NewCssMediaQueryType(&typ, nil, []string{"(grid)", "(color)"})

	if MediaQueriesEqual(a, b) {
		t.Error("different condition order should not be equal")
	}
}

func TestMediaQueriesEqualIgnoresConjunction(t *testing.T) {
	typ := "screen"
	a := NewCssMediaQueryType(&typ, nil, []string{"(color)"})
	b := &CssMediaQuery{
		Type:        &typ,
		Conjunction: false,
		Conditions:  []string{"(color)"},
	}

	if !MediaQueriesEqual(a, b) {
		t.Error("equality should ignore conjunction field")
	}
}

// ---- HashCode tests ----

func TestHashCodeEqualQueries(t *testing.T) {
	typ := "screen"
	a := NewCssMediaQueryType(&typ, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&typ, nil, []string{"(color)"})

	if a.HashCode() != b.HashCode() {
		t.Errorf("equal queries should have same hash: %d vs %d", a.HashCode(), b.HashCode())
	}
}

func TestHashCodeStable(t *testing.T) {
	typ := "screen"
	q := NewCssMediaQueryType(&typ, nil, []string{"(color)"})
	h1 := q.HashCode()
	h2 := q.HashCode()
	if h1 != h2 {
		t.Errorf("hash should be stable: %d vs %d", h1, h2)
	}
}

// ---- Merge tests ----

func TestMergeBothNilType(t *testing.T) {
	a, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)
	b, _ := NewCssMediaQueryCondition([]string{"(min-width: 700px)"}, nil)

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Type != nil {
		t.Error("expected nil type in merged query")
	}
	if len(merged.Query.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(merged.Query.Conditions))
	}
}

func TestMergeOneNotOneNormalSameTypeSubset(t *testing.T) {
	screen := "screen"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, nil, []string{"(color)", "(grid)"})

	result := a.Merge(b)
	if result != MediaQueryMergeResultEmpty {
		t.Errorf("expected Empty, got %T", result)
	}
}

func TestMergeOneNotOneNormalSameTypeNoSubset(t *testing.T) {
	screen := "screen"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, nil, []string{"(grid)"})

	result := a.Merge(b)
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeOneNotOneNormalBothMatchAll(t *testing.T) {
	all := "all"
	not := "not"
	screen := "screen"
	a := NewCssMediaQueryType(&all, &not, nil)
	b := NewCssMediaQueryType(&screen, nil, nil)

	result := a.Merge(b)
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeOneNotOneNormalDifferentTypes(t *testing.T) {
	screen := "screen"
	print := "print"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, nil)
	b := NewCssMediaQueryType(&print, nil, nil)

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Modifier != nil {
		t.Errorf("expected nil modifier (non-'not' side), got %q", *merged.Query.Modifier)
	}
	if merged.Query.Type == nil || *merged.Query.Type != "print" {
		t.Errorf("expected type 'print' (non-'not' side), got %v", merged.Query.Type)
	}
}

func TestMergeBothNotSameTypeSuperset(t *testing.T) {
	screen := "screen"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, []string{"(color)", "(grid)"})
	b := NewCssMediaQueryType(&screen, &not, []string{"(color)"})

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if len(merged.Query.Conditions) != 2 {
		t.Fatalf("expected 2 conditions (more), got %d", len(merged.Query.Conditions))
	}
}

func TestMergeBothNotSameTypeNoSuperset(t *testing.T) {
	screen := "screen"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, &not, []string{"(grid)"})

	result := a.Merge(b)
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeBothNotDifferentTypes(t *testing.T) {
	screen := "screen"
	print := "print"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, nil)
	b := NewCssMediaQueryType(&print, &not, nil)

	result := a.Merge(b)
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeFirstMatchesAllTypes(t *testing.T) {
	all := "all"
	screen := "screen"
	a := NewCssMediaQueryType(&all, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, nil, []string{"(grid)"})

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Type == nil || *merged.Query.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", merged.Query.Type)
	}
	if len(merged.Query.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(merged.Query.Conditions))
	}
}

func TestMergeSecondMatchesAllTypes(t *testing.T) {
	screen := "screen"
	all := "all"
	a := NewCssMediaQueryType(&screen, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&all, nil, []string{"(grid)"})

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Type == nil || *merged.Query.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", merged.Query.Type)
	}
}

func TestMergeBothAll(t *testing.T) {
	all1 := "all"
	all2 := "all"
	a := NewCssMediaQueryType(&all1, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&all2, nil, []string{"(grid)"})

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	// both match all types, our type is not nil (it's "all") → use their type ("all")
	if merged.Query.Type == nil || *merged.Query.Type != "all" {
		t.Errorf("expected type 'all', got %v", merged.Query.Type)
	}
}

func TestMergeBothAllOneNilType(t *testing.T) {
	all := "all"
	a, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil) // nil type
	b := NewCssMediaQueryType(&all, nil, []string{"(grid)"})    // "all" type

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	// a.MatchesAllTypes() → true (nil type), other is "all" and a's type is nil → type = nil
	if merged.Query.Type != nil {
		t.Errorf("expected nil type (both match all and ours is nil), got %v", merged.Query.Type)
	}
}

func TestMergeSameTypes(t *testing.T) {
	screen := "screen"
	a := NewCssMediaQueryType(&screen, nil, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, nil, []string{"(grid)"})

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Type == nil || *merged.Query.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", merged.Query.Type)
	}
	if len(merged.Query.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(merged.Query.Conditions))
	}
}

func TestMergeDifferentTypes(t *testing.T) {
	screen := "screen"
	print := "print"
	a := NewCssMediaQueryType(&screen, nil, nil)
	b := NewCssMediaQueryType(&print, nil, nil)

	result := a.Merge(b)
	if result != MediaQueryMergeResultEmpty {
		t.Errorf("expected Empty, got %T", result)
	}
}

func TestMergeNonConjunction(t *testing.T) {
	a, _ := NewCssMediaQueryCondition([]string{"(color)"}, nil)             // conjunction = true
	b := &CssMediaQuery{Conjunction: false, Conditions: []string{"(grid)"}} // conjunction = false

	result := a.Merge(b)
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeCaseInsensitiveTypes(t *testing.T) {
	screen := "SCREEN"
	screenLower := "screen"
	a := NewCssMediaQueryType(&screen, nil, nil)
	b := NewCssMediaQueryType(&screenLower, nil, nil)

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	// Should keep original casing from 'a' since ourType matches
	if merged.Query.Type == nil || *merged.Query.Type != "SCREEN" {
		t.Errorf("expected original casing 'SCREEN', got %v", merged.Query.Type)
	}
}

func TestMergeCaseInsensitiveModifier(t *testing.T) {
	screen := "screen"
	notMod := "NOT"
	a := NewCssMediaQueryType(&screen, &notMod, []string{"(color)"})
	b := NewCssMediaQueryType(&screen, nil, nil)

	result := a.Merge(b)
	// NOT is treated as "not" after lowercasing; "(color)" is not subset of empty
	// positive conditions, so result is unrepresentable
	if result != MediaQueryMergeResultUnrepresentable {
		t.Errorf("expected Unrepresentable, got %T", result)
	}
}

func TestMergeBothNotCaseInsensitiveTypes(t *testing.T) {
	screen := "SCREEN"
	screenLower := "screen"
	not := "not"
	a := NewCssMediaQueryType(&screen, &not, nil)
	b := NewCssMediaQueryType(&screenLower, &not, nil)

	result := a.Merge(b)
	merged, ok := result.(*MediaQuerySuccessfulMergeResult)
	if !ok {
		t.Fatalf("expected successful merge, got %T", result)
	}
	if merged.Query.Type == nil || *merged.Query.Type != "SCREEN" {
		t.Errorf("expected type 'SCREEN', got %v", merged.Query.Type)
	}
}

func TestMergeNotVsNormalCaseInsensitiveModifier(t *testing.T) {
	screen := "screen"
	notMod := "NOT"
	a := NewCssMediaQueryType(&screen, &notMod, nil)
	b := NewCssMediaQueryType(&screen, nil, nil)

	result := a.Merge(b)
	// "NOT" lowercased = "not"; both nil conditions → allContained(nil, nil) = true → Empty
	if result != MediaQueryMergeResultEmpty {
		t.Errorf("expected Empty, got %T", result)
	}
}

// ---- ParseList tests ----

func TestParseListValid(t *testing.T) {
	queries, err := ParseList("screen and (min-width: 700px)", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}
	q := queries[0]
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("expected type 'screen', got %v", q.Type)
	}
	if len(q.Conditions) != 1 || q.Conditions[0] != "(min-width: 700px)" {
		t.Errorf("expected condition '(min-width: 700px)', got %v", q.Conditions)
	}
}

func TestParseListMultipleQueries(t *testing.T) {
	queries, err := ParseList("screen and (color), print", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}
}

func TestParseListInvalid(t *testing.T) {
	_, err := ParseList("not a valid media query!!!", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}
