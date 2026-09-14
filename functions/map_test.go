// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package functions

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// --- helpers ---

func mkStr(s string) *value.SassString { return &value.SassString{Text: s, HasQuotes: false} }

func mkNum(v float64) value.Value { return value.NewUnitlessNumber(v) }

func mkList(t *testing.T, sep value.ListSeparator, contents ...value.Value) *value.SassList {
	t.Helper()
	l, err := value.NewSassList(contents, sep, false)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// mkMap builds a SassMap from alternating key/value pairs, preserving order.
func mkMap(pairs ...value.Value) *value.SassMap {
	m := value.EmptySassMap()
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i], pairs[i+1])
	}
	return m
}

func inspect(t *testing.T, v value.Value) string {
	t.Helper()
	s, err := value.SerializeValueInspect(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func callMapFn(t *testing.T, fn *BuiltInCallable, positional int, names map[string]struct{}, args []value.Value) (value.Value, error) {
	t.Helper()
	res, err := fn.CallbackFor(positional, names)
	if err != nil {
		t.Fatalf("CallbackFor(%d): %v", positional, err)
	}
	return res.Fn(nil, args)
}

func callMapFnOK(t *testing.T, fn *BuiltInCallable, positional int, names map[string]struct{}, args []value.Value) value.Value {
	t.Helper()
	got, err := callMapFn(t, fn, positional, names, args)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertInspect(t *testing.T, got value.Value, want string) {
	t.Helper()
	if s := inspect(t, got); s != want {
		t.Errorf("inspect = %q, want %q", s, want)
	}
}

func assertErrMsg(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// --- map.get ---

func TestMapGetTopLevel(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, mapGetFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, got, "1")
}

func TestMapGetMissingKey(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapGetFunction(), 2, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma)})
	if got != value.Null {
		t.Errorf("get missing key = %v, want Null", got)
	}
}

func TestMapGetNested(t *testing.T) {
	inner2 := mkMap(mkStr("c"), mkNum(3))
	inner1 := mkMap(mkStr("b"), inner2)
	m := mkMap(mkStr("a"), inner1)
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"), mkStr("c"))})
	assertInspect(t, got, "3")
}

func TestMapGetNestedIntermediateMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.Null {
		t.Errorf("get with missing intermediate = %v, want Null", got)
	}
}

func TestMapGetNestedIntermediateNotAMap(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.Null {
		t.Errorf("get with non-map intermediate = %v, want Null", got)
	}
}

func TestMapGetNestedIntermediateEmptyList(t *testing.T) {
	// An empty list is NOT a *SassMap for get's type assertion, so lookup stops.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.Null {
		t.Errorf("get through empty list = %v, want Null", got)
	}
}

func TestMapGetNestedLastMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("x"))})
	if got != value.Null {
		t.Errorf("get with missing last key = %v, want Null", got)
	}
}

func TestMapGetReturnsMapValue(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapGetFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	if _, ok := got.(*value.SassMap); !ok {
		t.Fatalf("result should be a *SassMap, got %T", got)
	}
	assertInspect(t, got, "(b: 1)")
}

func TestMapGetTwoArgsOnly(t *testing.T) {
	// Defensive path: len(args) == 2, no rest argument at all.
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapGetFunction(), 2, nil, []value.Value{m, mkStr("a")})
	assertInspect(t, got, "1")
}

func TestMapGetRestArgumentList(t *testing.T) {
	// The rest argument is a SassArgumentList during real evaluation.
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(4)))
	rest, err := value.NewSassArgumentList(
		[]value.Value{mkStr("b")}, orderedmap.New[string, value.Value](), value.ListSeparatorComma)
	if err != nil {
		t.Fatal(err)
	}
	got := callMapFnOK(t, mapGetFunction(), 3, nil, []value.Value{m, mkStr("a"), rest})
	assertInspect(t, got, "4")
}

func TestMapGetRestScalarAsList(t *testing.T) {
	// AsList on a scalar returns [self], so the scalar becomes a single key.
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapGetFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkNum(5)})
	if got != value.Null {
		t.Errorf("get = %v, want Null", got)
	}
}

func TestMapGetEmptyListAsMap(t *testing.T) {
	got := callMapFnOK(t, mapGetFunction(), 2, nil,
		[]value.Value{mkList(t, value.ListSeparatorSpace), mkStr("a"), mkList(t, value.ListSeparatorComma)})
	if got != value.Null {
		t.Errorf("get on empty list = %v, want Null", got)
	}
}

func TestMapGetNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapGetFunction(), 2, nil,
		[]value.Value{mkNum(3), mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

// --- map.set ---

func TestMapSetThreeArgsNewKey(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, setFunction(), 3, nil, []value.Value{m, mkStr("b"), mkNum(2)})
	assertInspect(t, got, "(a: 1, b: 2)")
}

func TestMapSetThreeArgsExistingKey(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, setFunction(), 3, nil, []value.Value{m, mkStr("a"), mkNum(3)})
	assertInspect(t, got, "(a: 3, b: 2)")
}

func TestMapSetThreeArgsDoesNotMutateOriginal(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	callMapFnOK(t, setFunction(), 3, nil, []value.Value{m, mkStr("a"), mkNum(9)})
	assertInspect(t, m, "(a: 1)")
}

func TestMapSetThreeArgsOnEmptyListMap(t *testing.T) {
	got := callMapFnOK(t, setFunction(), 3, nil,
		[]value.Value{mkList(t, value.ListSeparatorSpace), mkStr("a"), mkNum(1)})
	assertInspect(t, got, "(a: 1)")
}

func TestMapSetThreeArgsNotAMap(t *testing.T) {
	_, err := callMapFn(t, setFunction(), 3, nil, []value.Value{mkNum(3), mkStr("a"), mkNum(1)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

func TestMapSetRestEmpty(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	_, err := callMapFn(t, setFunction(), 1, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "Expected $args to contain a key.")
}

func TestMapSetRestOneItem(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	_, err := callMapFn(t, setFunction(), 2, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"))})
	assertErrMsg(t, err, "Expected $args to contain a value.")
}

func TestMapSetRestTwoItems(t *testing.T) {
	// Two rest items: one key plus the value, so a top-level set.
	m := mkMap(mkStr("a"), mkNum(1))
	res, err := setFunction().CallbackFor(4, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := res.Fn(nil, []value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("b"), mkNum(2))})
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(a: 1, b: 2)")
}

func TestMapSetRestNestedCreate(t *testing.T) {
	m := value.EmptySassMap()
	got := callMapFnOK(t, setFunction(), 4, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkNum(1))})
	assertInspect(t, got, "(a: (b: 1))")
}

func TestMapSetRestNestedExisting(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)), mkStr("c"), mkNum(2))
	got := callMapFnOK(t, setFunction(), 4, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkNum(9))})
	assertInspect(t, got, "(a: (b: 9), c: 2)")
}

func TestMapSetRestOverwritesNonMapIntermediate(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, setFunction(), 4, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkNum(2))})
	assertInspect(t, got, "(a: (b: 2))")
}

func TestMapSetRestDeepCreate(t *testing.T) {
	m := value.EmptySassMap()
	got := callMapFnOK(t, setFunction(), 5, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkStr("c"), mkNum(1))})
	assertInspect(t, got, "(a: (b: (c: 1)))")
}

func TestMapSetRestEmptyListIntermediate(t *testing.T) {
	// TryMap converts an empty list intermediate into an empty map.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, setFunction(), 4, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkNum(1))})
	assertInspect(t, got, "(a: (b: 1))")
}

func TestMapSetRestNotAMap(t *testing.T) {
	_, err := callMapFn(t, setFunction(), 4, nil,
		[]value.Value{mkNum(3), mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkNum(1))})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

// --- map.merge ---

func TestMapMergeBasic(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	map2 := mkMap(mkStr("c"), mkNum(3))
	got := callMapFnOK(t, mapMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: 1, b: 2, c: 3)")
}

func TestMapMergeOverlapKeepsPosition(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	map2 := mkMap(mkStr("b"), mkNum(9), mkStr("c"), mkNum(3))
	got := callMapFnOK(t, mapMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: 1, b: 9, c: 3)")
}

func TestMapMergeEmptyFirst(t *testing.T) {
	got := callMapFnOK(t, mapMergeFunction(), 2, nil,
		[]value.Value{mkList(t, value.ListSeparatorSpace), mkMap(mkStr("a"), mkNum(1))})
	assertInspect(t, got, "(a: 1)")
}

func TestMapMergeEmptySecond(t *testing.T) {
	got := callMapFnOK(t, mapMergeFunction(), 2, nil,
		[]value.Value{mkMap(mkStr("a"), mkNum(1)), mkList(t, value.ListSeparatorSpace)})
	assertInspect(t, got, "(a: 1)")
}

func TestMapMergeDoesNotMutateOriginals(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1))
	map2 := mkMap(mkStr("b"), mkNum(2))
	callMapFnOK(t, mapMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, map1, "(a: 1)")
	assertInspect(t, map2, "(b: 2)")
}

func TestMapMergeMap1NotAMap(t *testing.T) {
	_, err := callMapFn(t, mapMergeFunction(), 2, nil,
		[]value.Value{mkNum(1), mkMap(mkStr("a"), mkNum(1))})
	assertErrMsg(t, err, "$map1: 1 is not a map.")
}

func TestMapMergeMap2NotAMap(t *testing.T) {
	_, err := callMapFn(t, mapMergeFunction(), 2, nil,
		[]value.Value{mkMap(mkStr("a"), mkNum(1)), mkNum(1)})
	assertErrMsg(t, err, "$map2: 1 is not a map.")
}

func TestMapMergeRestEmpty(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	_, err := callMapFn(t, mapMergeFunction(), 1, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "Expected $args to contain a key.")
}

func TestMapMergeRestOneItem(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	res, err := mapMergeFunction().CallbackFor(3, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = res.Fn(nil, []value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"))})
	assertErrMsg(t, err, "Expected $args to contain a map.")
}

func TestMapMergeRestNested(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)), mkStr("c"), mkNum(2))
	got := callMapFnOK(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("b2"), mkNum(9)))})
	assertInspect(t, got, "(a: (b: 1, b2: 9), c: 2)")
}

func TestMapMergeRestNestedOverlap(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1), mkStr("y"), mkNum(2)))
	got := callMapFnOK(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("y"), mkNum(9)))})
	assertInspect(t, got, "(a: (x: 1, y: 9))")
}

func TestMapMergeRestOldValueNotAMap(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("x"), mkNum(2)))})
	assertInspect(t, got, "(a: (x: 2))")
}

func TestMapMergeRestMissingKey(t *testing.T) {
	m := value.EmptySassMap()
	got := callMapFnOK(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("x"), mkNum(2)))})
	assertInspect(t, got, "(a: (x: 2))")
}

func TestMapMergeRestEmptyListOldValue(t *testing.T) {
	// TryMap converts the empty-list old value to an empty map, which is then merged.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("x"), mkNum(2)))})
	assertInspect(t, got, "(a: (x: 2))")
}

func TestMapMergeRestDeepPath(t *testing.T) {
	inner := mkMap(mkStr("b"), mkMap(mkStr("x"), mkNum(1)))
	m := mkMap(mkStr("a"), inner)
	got := callMapFnOK(t, mapMergeFunction(), 4, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkStr("b"), mkMap(mkStr("y"), mkNum(2)))})
	assertInspect(t, got, "(a: (b: (x: 1, y: 2)))")
}

func TestMapMergeRestLastNotAMap(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	_, err := callMapFn(t, mapMergeFunction(), 3, nil,
		[]value.Value{m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkNum(5))})
	assertErrMsg(t, err, "$map2: 5 is not a map.")
}

func TestMapMergeRestMap1NotAMap(t *testing.T) {
	_, err := callMapFn(t, mapMergeFunction(), 3, nil,
		[]value.Value{mkNum(3), mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap())})
	assertErrMsg(t, err, "$map1: 3 is not a map.")
}

// --- map.remove ---

func TestMapRemoveNoKeys(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapRemoveFunction(), 1, nil, []value.Value{m})
	assertInspect(t, got, "(a: 1)")
}

func TestMapRemoveNoKeysEmptyList(t *testing.T) {
	got := callMapFnOK(t, mapRemoveFunction(), 1, nil,
		[]value.Value{mkList(t, value.ListSeparatorSpace)})
	if _, ok := got.(*value.SassMap); !ok {
		t.Fatalf("result should be a *SassMap, got %T", got)
	}
	assertInspect(t, got, "()")
}

func TestMapRemoveOneKey(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, mapRemoveFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, got, "(b: 2)")
}

func TestMapRemoveMultipleKeys(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2), mkStr("c"), mkNum(3))
	got := callMapFnOK(t, mapRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	assertInspect(t, got, "(c: 3)")
}

func TestMapRemoveMissingKeysIgnored(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma, mkStr("y"))})
	assertInspect(t, got, "(a: 1)")
}

func TestMapRemoveDoesNotMutateOriginal(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	callMapFnOK(t, mapRemoveFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, m, "(a: 1, b: 2)")
}

func TestMapRemoveKeyPassedByName(t *testing.T) {
	// The second overload exists so $key can be passed by name.
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, mapRemoveFunction(), 1, map[string]struct{}{"key": {}},
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, got, "(b: 2)")
}

func TestMapRemoveNoKeysNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapRemoveFunction(), 1, nil, []value.Value{mkNum(3)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

func TestMapRemoveWithKeysNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapRemoveFunction(), 2, nil,
		[]value.Value{mkNum(3), mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

// --- map.keys ---

func TestMapKeys(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, mapKeysFunction(), 1, nil, []value.Value{m})
	list, ok := got.(*value.SassList)
	if !ok {
		t.Fatalf("keys should return a *SassList, got %T", got)
	}
	if list.Separator() != value.ListSeparatorComma {
		t.Errorf("separator = %v, want comma", list.Separator())
	}
	if list.HasBrackets() {
		t.Error("keys list should not have brackets")
	}
	assertInspect(t, got, "a, b")
}

func TestMapKeysSingle(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapKeysFunction(), 1, nil, []value.Value{m})
	assertInspect(t, got, "(a,)")
}

func TestMapKeysEmpty(t *testing.T) {
	got := callMapFnOK(t, mapKeysFunction(), 1, nil, []value.Value{value.EmptySassMap()})
	list, ok := got.(*value.SassList)
	if !ok {
		t.Fatalf("keys should return a *SassList, got %T", got)
	}
	if list.LengthAsList() != 0 {
		t.Errorf("length = %d, want 0", list.LengthAsList())
	}
	if list.Separator() != value.ListSeparatorComma {
		t.Errorf("separator = %v, want comma", list.Separator())
	}
	assertInspect(t, got, "()")
}

func TestMapKeysNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapKeysFunction(), 1, nil, []value.Value{mkNum(1)})
	assertErrMsg(t, err, "$map: 1 is not a map.")
}

// --- map.values ---

func TestMapValues(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, mapValuesFunction(), 1, nil, []value.Value{m})
	list, ok := got.(*value.SassList)
	if !ok {
		t.Fatalf("values should return a *SassList, got %T", got)
	}
	if list.Separator() != value.ListSeparatorComma {
		t.Errorf("separator = %v, want comma", list.Separator())
	}
	if list.HasBrackets() {
		t.Error("values list should not have brackets")
	}
	assertInspect(t, got, "1, 2")
}

func TestMapValuesSingle(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapValuesFunction(), 1, nil, []value.Value{m})
	assertInspect(t, got, "(1,)")
}

func TestMapValuesEmpty(t *testing.T) {
	got := callMapFnOK(t, mapValuesFunction(), 1, nil, []value.Value{value.EmptySassMap()})
	list, ok := got.(*value.SassList)
	if !ok {
		t.Fatalf("values should return a *SassList, got %T", got)
	}
	if list.LengthAsList() != 0 {
		t.Errorf("length = %d, want 0", list.LengthAsList())
	}
	assertInspect(t, got, "()")
}

func TestMapValuesNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapValuesFunction(), 1, nil, []value.Value{mkNum(1)})
	assertErrMsg(t, err, "$map: 1 is not a map.")
}

// --- map.has-key ---

func TestMapHasKeyTrue(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapHasKeyFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	if got != value.SassTrue {
		t.Errorf("has-key = %v, want SassTrue", got)
	}
}

func TestMapHasKeyFalse(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapHasKeyFunction(), 2, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma)})
	if got != value.SassFalse {
		t.Errorf("has-key = %v, want SassFalse", got)
	}
}

func TestMapHasKeyNullValue(t *testing.T) {
	// A key whose value is null still counts as present.
	m := mkMap(mkStr("a"), value.Null)
	got := callMapFnOK(t, mapHasKeyFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	if got != value.SassTrue {
		t.Errorf("has-key with null value = %v, want SassTrue", got)
	}
}

func TestMapHasKeyNestedTrue(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapHasKeyFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.SassTrue {
		t.Errorf("nested has-key = %v, want SassTrue", got)
	}
}

func TestMapHasKeyNestedLastMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapHasKeyFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("x"))})
	if got != value.SassFalse {
		t.Errorf("nested has-key = %v, want SassFalse", got)
	}
}

func TestMapHasKeyNestedIntermediateMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, mapHasKeyFunction(), 3, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.SassFalse {
		t.Errorf("nested has-key = %v, want SassFalse", got)
	}
}

func TestMapHasKeyNestedIntermediateNotAMap(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapHasKeyFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.SassFalse {
		t.Errorf("nested has-key = %v, want SassFalse", got)
	}
}

func TestMapHasKeyNestedIntermediateEmptyList(t *testing.T) {
	// An empty list is NOT a *SassMap for has-key's type assertion.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, mapHasKeyFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	if got != value.SassFalse {
		t.Errorf("nested has-key = %v, want SassFalse", got)
	}
}

func TestMapHasKeyTwoArgsOnly(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, mapHasKeyFunction(), 2, nil, []value.Value{m, mkStr("a")})
	if got != value.SassTrue {
		t.Errorf("has-key = %v, want SassTrue", got)
	}
}

func TestMapHasKeyNotAMap(t *testing.T) {
	_, err := callMapFn(t, mapHasKeyFunction(), 2, nil,
		[]value.Value{mkNum(3), mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

// --- map.deep-merge ---

func TestDeepMergeFlat(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	map2 := mkMap(mkStr("b"), mkNum(9), mkStr("c"), mkNum(3))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: 1, b: 9, c: 3)")
}

func TestDeepMergeEmptyMap1(t *testing.T) {
	map2 := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{value.EmptySassMap(), map2})
	assertInspect(t, got, "(a: 1)")
}

func TestDeepMergeEmptyMap2(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, value.EmptySassMap()})
	assertInspect(t, got, "(a: 1)")
}

func TestDeepMergeBothEmpty(t *testing.T) {
	got := callMapFnOK(t, deepMergeFunction(), 2, nil,
		[]value.Value{value.EmptySassMap(), value.EmptySassMap()})
	assertInspect(t, got, "()")
}

func TestDeepMergeNestedMaps(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1), mkStr("y"), mkNum(2)))
	map2 := mkMap(mkStr("a"), mkMap(mkStr("y"), mkNum(9), mkStr("z"), mkNum(3)))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: (x: 1, y: 9, z: 3))")
}

func TestDeepMergeTwoLevels(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkMap(mkStr("b"), mkMap(mkStr("x"), mkNum(1))))
	map2 := mkMap(mkStr("a"), mkMap(mkStr("b"), mkMap(mkStr("y"), mkNum(2))))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: (b: (x: 1, y: 2)))")
}

func TestDeepMergeValEmptyMapKeepsExisting(t *testing.T) {
	// Merging a nested empty map into an existing nested map leaves it unchanged.
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1)))
	map2 := mkMap(mkStr("a"), value.EmptySassMap())
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: (x: 1))")
}

func TestDeepMergeValEmptyListKeepsExisting(t *testing.T) {
	// An empty list value converts to an empty map via TryMap and merges as one.
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1)))
	map2 := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: (x: 1))")
}

func TestDeepMergeExistingNotAMap(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1))
	map2 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(2)))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: (x: 2))")
}

func TestDeepMergeValNotAMap(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1)))
	map2 := mkMap(mkStr("a"), mkNum(5))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(a: 5)")
}

func TestDeepMergeNewKeysAppended(t *testing.T) {
	map1 := mkMap(mkStr("b"), mkNum(2))
	map2 := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, got, "(b: 2, a: 1)")
}

func TestDeepMergeDoesNotMutateOriginals(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1)))
	map2 := mkMap(mkStr("a"), mkMap(mkStr("y"), mkNum(2)))
	callMapFnOK(t, deepMergeFunction(), 2, nil, []value.Value{map1, map2})
	assertInspect(t, map1, "(a: (x: 1))")
	assertInspect(t, map2, "(a: (y: 2))")
}

func TestDeepMergeMap1NotAMap(t *testing.T) {
	_, err := callMapFn(t, deepMergeFunction(), 2, nil,
		[]value.Value{mkNum(1), value.EmptySassMap()})
	assertErrMsg(t, err, "$map1: 1 is not a map.")
}

func TestDeepMergeMap2NotAMap(t *testing.T) {
	_, err := callMapFn(t, deepMergeFunction(), 2, nil,
		[]value.Value{value.EmptySassMap(), mkNum(1)})
	assertErrMsg(t, err, "$map2: 1 is not a map.")
}

// --- deepMergeImpl (internal) ---

func TestDeepMergeImplEmptyMap1ReturnsMap2(t *testing.T) {
	map2 := mkMap(mkStr("a"), mkNum(1))
	got := deepMergeImpl(value.EmptySassMap(), map2)
	if got != map2 {
		t.Error("deepMergeImpl with empty map1 should return map2 itself")
	}
}

func TestDeepMergeImplEmptyMap2ReturnsMap1(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkNum(1))
	got := deepMergeImpl(map1, value.EmptySassMap())
	if got != map1 {
		t.Error("deepMergeImpl with empty map2 should return map1 itself")
	}
}

func TestDeepMergeImplRecursive(t *testing.T) {
	map1 := mkMap(mkStr("a"), mkMap(mkStr("x"), mkNum(1)), mkStr("b"), mkNum(2))
	map2 := mkMap(mkStr("a"), mkMap(mkStr("y"), mkNum(3)))
	got := deepMergeImpl(map1, map2)
	assertInspect(t, got, "(a: (x: 1, y: 3), b: 2)")
}

// --- map.deep-remove ---

func TestDeepRemoveTopLevel(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	got := callMapFnOK(t, deepRemoveFunction(), 2, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, got, "(b: 2)")
}

func TestDeepRemoveTopLevelMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, deepRemoveFunction(), 2, nil,
		[]value.Value{m, mkStr("x"), mkList(t, value.ListSeparatorComma)})
	assertInspect(t, got, "(a: 1)")
}

func TestDeepRemoveNested(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1), mkStr("c"), mkNum(2)))
	got := callMapFnOK(t, deepRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	assertInspect(t, got, "(a: (c: 2))")
}

func TestDeepRemoveNestedLastMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	got := callMapFnOK(t, deepRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("x"))})
	assertInspect(t, got, "(a: (b: 1))")
}

func TestDeepRemoveIntermediateNotAMap(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	got := callMapFnOK(t, deepRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	assertInspect(t, got, "(a: 1)")
}

func TestDeepRemoveIntermediatePathMissing(t *testing.T) {
	m := mkMap(mkStr("x"), mkNum(1))
	got := callMapFnOK(t, deepRemoveFunction(), 4, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"), mkStr("c"))})
	assertInspect(t, got, "(x: 1)")
}

func TestDeepRemoveEmptyListIntermediate(t *testing.T) {
	// TryMap turns the empty-list intermediate into an empty map, so the
	// traversal continues and materializes (b: null) along the way.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := callMapFnOK(t, deepRemoveFunction(), 4, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"), mkStr("c"))})
	assertInspect(t, got, "(a: (b: null))")
}

func TestDeepRemoveThreeLevels(t *testing.T) {
	inner := mkMap(mkStr("c"), mkNum(1), mkStr("d"), mkNum(2))
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), inner))
	got := callMapFnOK(t, deepRemoveFunction(), 4, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"), mkStr("c"))})
	assertInspect(t, got, "(a: (b: (d: 2)))")
}

func TestDeepRemoveDoesNotMutateOriginal(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	callMapFnOK(t, deepRemoveFunction(), 3, nil,
		[]value.Value{m, mkStr("a"), mkList(t, value.ListSeparatorComma, mkStr("b"))})
	assertInspect(t, m, "(a: (b: 1))")
}

func TestDeepRemoveNotAMap(t *testing.T) {
	_, err := callMapFn(t, deepRemoveFunction(), 2, nil,
		[]value.Value{mkNum(3), mkStr("a"), mkList(t, value.ListSeparatorComma)})
	assertErrMsg(t, err, "$map: 3 is not a map.")
}

// --- modifyMap (internal) ---

func TestModifyMapNoKeys(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	var received value.Value
	got := modifyMap(m, nil, func(old value.Value) value.Value {
		received = old
		return mkNum(42)
	}, true)
	if received != value.Value(m) {
		t.Error("modify should receive the map itself when no keys are given")
	}
	assertInspect(t, got, "42")
}

func TestModifyMapSingleKeyExisting(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1), mkStr("b"), mkNum(2))
	var received value.Value
	got := modifyMap(m, []value.Value{mkStr("a")}, func(old value.Value) value.Value {
		received = old
		return mkNum(9)
	}, true)
	assertInspect(t, received, "1")
	assertInspect(t, got, "(a: 9, b: 2)")
	assertInspect(t, m, "(a: 1, b: 2)")
}

func TestModifyMapSingleKeyMissing(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(1))
	var received value.Value
	got := modifyMap(m, []value.Value{mkStr("x")}, func(old value.Value) value.Value {
		received = old
		return mkNum(9)
	}, true)
	if received != value.Null {
		t.Errorf("modify should receive Null for a missing key, got %v", received)
	}
	assertInspect(t, got, "(a: 1, x: 9)")
}

func TestModifyMapNestedAddNesting(t *testing.T) {
	m := value.EmptySassMap()
	var received value.Value
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		received = old
		return mkNum(1)
	}, true)
	if received != value.Null {
		t.Errorf("modify should receive Null, got %v", received)
	}
	assertInspect(t, got, "(a: (b: 1))")
}

func TestModifyMapNestedExistingPath(t *testing.T) {
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	var received value.Value
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		received = old
		return mkNum(9)
	}, true)
	assertInspect(t, received, "1")
	assertInspect(t, got, "(a: (b: 9))")
}

func TestModifyMapNestedNonMapIntermediateAddNesting(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(5))
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		return mkNum(9)
	}, true)
	assertInspect(t, got, "(a: (b: 9))")
}

func TestModifyMapNestedMissingNoNesting(t *testing.T) {
	m := mkMap(mkStr("x"), mkNum(1))
	called := false
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		called = true
		return mkNum(9)
	}, false)
	if called {
		t.Error("modify should not be called when nesting is missing and addNesting is false")
	}
	assertInspect(t, got, "(x: 1)")
}

func TestModifyMapNestedNonMapIntermediateNoNesting(t *testing.T) {
	m := mkMap(mkStr("a"), mkNum(5))
	called := false
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		called = true
		return mkNum(9)
	}, false)
	if called {
		t.Error("modify should not be called for a non-map intermediate with addNesting false")
	}
	assertInspect(t, got, "(a: 5)")
}

func TestModifyMapEmptyListIntermediateNoNesting(t *testing.T) {
	// TryMap converts an empty list to an empty map even when addNesting is
	// false, so traversal continues.
	m := mkMap(mkStr("a"), mkList(t, value.ListSeparatorSpace))
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b")}, func(old value.Value) value.Value {
		return mkNum(9)
	}, false)
	assertInspect(t, got, "(a: (b: 9))")
}

func TestModifyMapDeepNesting(t *testing.T) {
	m := value.EmptySassMap()
	got := modifyMap(m, []value.Value{mkStr("a"), mkStr("b"), mkStr("c")}, func(old value.Value) value.Value {
		return mkNum(9)
	}, true)
	assertInspect(t, got, "(a: (b: (c: 9)))")
}

// --- GlobalMapFunctions ---

func TestGlobalMapFunctionsNames(t *testing.T) {
	fns := GlobalMapFunctions()
	want := []string{"map-get", "map-merge", "map-remove", "map-keys", "map-values", "map-has-key"}
	if len(fns) != len(want) {
		t.Fatalf("len = %d, want %d", len(fns), len(want))
	}
	for i, fn := range fns {
		if fn.Name() != want[i] {
			t.Errorf("fns[%d].Name() = %q, want %q", i, fn.Name(), want[i])
		}
	}
}

type recordingLogger struct {
	messages     []string
	deprecations []*deprecation.Deprecation
}

func (r *recordingLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	r.messages = append(r.messages, message)
	r.deprecations = append(r.deprecations, nil)
}

func (r *recordingLogger) Debug(message string, span *sasscommon.FileSpan) {}

func (r *recordingLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace) error {
	r.messages = append(r.messages, message)
	r.deprecations = append(r.deprecations, dep)
	return nil
}

func TestGlobalMapGetEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalMapFunctions()
	mapGet, ok := fns[0].(*BuiltInCallable)
	if !ok {
		t.Fatalf("map-get should be a *BuiltInCallable, got %T", fns[0])
	}
	rl := &recordingLogger{}
	ec := &evalcontext.EvaluationContext{Logger: rl}
	m := mkMap(mkStr("a"), mkNum(1))
	res, err := mapGet.CallbackFor(2, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := res.Fn(ec, []value.Value{m, mkStr("a")})
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "1")
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d, want 1", len(rl.messages))
	}
	want := strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use map.get instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
	if rl.deprecations[0] != deprecation.GlobalBuiltin {
		t.Errorf("deprecation = %v, want GlobalBuiltin", rl.deprecations[0])
	}
}

func TestGlobalMapHasKeyEmitsDeprecationWarning(t *testing.T) {
	fns := GlobalMapFunctions()
	hasKey, ok := fns[5].(*BuiltInCallable)
	if !ok {
		t.Fatalf("map-has-key should be a *BuiltInCallable, got %T", fns[5])
	}
	rl := &recordingLogger{}
	ec := &evalcontext.EvaluationContext{Logger: rl}
	m := mkMap(mkStr("a"), mkNum(1))
	res, err := hasKey.CallbackFor(2, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := res.Fn(ec, []value.Value{m, mkStr("a")})
	if err != nil {
		t.Fatal(err)
	}
	if got != value.SassTrue {
		t.Errorf("map-has-key = %v, want SassTrue", got)
	}
	want := strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use map.has-key instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d, want 1", len(rl.messages))
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
}

func TestGlobalMapMergeRestOverloadEmitsDeprecationWarning(t *testing.T) {
	// WithDeprecationWarning wraps every overload, including the rest overload.
	fns := GlobalMapFunctions()
	mapMerge, ok := fns[1].(*BuiltInCallable)
	if !ok {
		t.Fatalf("map-merge should be a *BuiltInCallable, got %T", fns[1])
	}
	rl := &recordingLogger{}
	ec := &evalcontext.EvaluationContext{Logger: rl}
	m := mkMap(mkStr("a"), mkMap(mkStr("b"), mkNum(1)))
	res, err := mapMerge.CallbackFor(3, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := res.Fn(ec, []value.Value{
		m, mkList(t, value.ListSeparatorComma, mkStr("a"), mkMap(mkStr("c"), mkNum(2)))})
	if err != nil {
		t.Fatal(err)
	}
	assertInspect(t, got, "(a: (b: 1, c: 2))")
	want := strings.Join([]string{
		`Global built-in functions are deprecated and will be removed in Dart Sass 3.0.0.`,
		`Use map.merge instead.`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/import`,
	}, "\n")
	if len(rl.messages) != 1 {
		t.Fatalf("warnings = %d, want 1", len(rl.messages))
	}
	if rl.messages[0] != want {
		t.Errorf("warning = %q, want %q", rl.messages[0], want)
	}
}

// --- MapModule ---

func TestMapModuleURL(t *testing.T) {
	m := MapModule()
	url, err := m.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "sass:map" {
		t.Errorf("URL = %q, want %q", url, "sass:map")
	}
}

func TestMapModuleFunctions(t *testing.T) {
	m := MapModule()
	want := []string{"get", "set", "merge", "remove", "keys", "values", "has-key", "deep-merge", "deep-remove"}
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

func TestMapModuleNoMixinsNoVariables(t *testing.T) {
	m := MapModule()
	if m.Mixins().Len() != 0 {
		t.Errorf("mixins = %d, want 0", m.Mixins().Len())
	}
	if m.Variables().Len() != 0 {
		t.Errorf("variables = %d, want 0", m.Variables().Len())
	}
}
