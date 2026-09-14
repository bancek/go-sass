package value

import (
	"strings"
	"testing"
)

func TestEmptySassMap(t *testing.T) {
	m := EmptySassMap()
	if m.LengthAsList() != 0 {
		t.Errorf("length = %d, want 0", m.LengthAsList())
	}
	if !m.IsTruthy() {
		t.Error("map should be truthy")
	}
}

func TestNewSassMap(t *testing.T) {
	m := NewSassMap(map[Value]Value{
		&SassString{Text: "a", HasQuotes: true}: NewUnitlessNumber(1),
		&SassString{Text: "b", HasQuotes: true}: NewUnitlessNumber(2),
	})
	if m.LengthAsList() != 2 {
		t.Errorf("length = %d, want 2", m.LengthAsList())
	}
}

func TestNewSassMap_DupKeys(t *testing.T) {
	a1 := &SassString{Text: "a"}
	a2 := &SassString{Text: "a"}
	m := NewSassMap(map[Value]Value{
		a1: NewUnitlessNumber(1),
		a2: NewUnitlessNumber(2),
	})
	if m.LengthAsList() != 1 {
		t.Fatalf("duplicate keys: length = %d, want 1", m.LengthAsList())
	}
	v, _ := m.Get(&SassString{Text: "a"})
	n, ok := v.(SassNumber)
	if !ok {
		t.Fatalf("value should be a number, got %T", v)
	}
	if n.NumValue() != 1 && n.NumValue() != 2 {
		t.Errorf("value should be 1 or 2 (map iteration order): got %v", n.NumValue())
	}
}

func TestMapGet(t *testing.T) {
	k := &SassString{Text: "a", HasQuotes: true}
	m := NewSassMap(map[Value]Value{k: NewUnitlessNumber(42)})
	v, found := m.Get(k)
	n, ok := v.(SassNumber)
	if !found || !ok || n.NumValue() != 42 {
		t.Errorf("Get existing = (%v, %v)", v, found)
	}
	_, ok = m.Get(&SassString{Text: "b"})
	if ok {
		t.Error("Get missing should return false")
	}
}

func TestMapSet(t *testing.T) {
	m := EmptySassMap()
	k := &SassString{Text: "a", HasQuotes: true}
	m.Set(k, NewUnitlessNumber(1))
	if m.LengthAsList() != 1 {
		t.Error("Set should add entry")
	}
	if !m.Contains(k) {
		t.Error("Contains should be true after Set")
	}
	m.Set(k, NewUnitlessNumber(2))
	if m.LengthAsList() != 1 {
		t.Error("Set on existing key should not increase length")
	}
	v, _ := m.Get(k)
	n, _ := v.(SassNumber)
	if n.NumValue() != 2 {
		t.Error("Set should update value")
	}
}

func TestMapDelete(t *testing.T) {
	k := &SassString{Text: "a", HasQuotes: true}
	m := NewSassMap(map[Value]Value{k: NewUnitlessNumber(1)})
	m.Delete(k)
	if m.Contains(k) {
		t.Error("Contains should be false after Delete")
	}
	if m.LengthAsList() != 0 {
		t.Error("length should be 0 after Delete")
	}
}

func TestMapCopy(t *testing.T) {
	k := &SassString{Text: "a", HasQuotes: true}
	m := NewSassMap(map[Value]Value{k: NewUnitlessNumber(1)})
	c := m.Copy()
	c.Set(k, NewUnitlessNumber(99))
	v, _ := m.Get(k)
	n, _ := v.(SassNumber)
	if n.NumValue() != 1 {
		t.Error("modifying copy should not affect original")
	}
}

func TestMapEntries(t *testing.T) {
	m := NewSassMap(map[Value]Value{
		&SassString{Text: "a", HasQuotes: true}: NewUnitlessNumber(1),
		&SassString{Text: "b", HasQuotes: true}: NewUnitlessNumber(2),
	})
	count := 0
	for key, val := range m.Entries() {
		if key == nil || val == nil {
			t.Error("entries should not be nil")
		}
		count++
	}
	if count != 2 {
		t.Errorf("entries count = %d, want 2", count)
	}
}

func TestMapAsList(t *testing.T) {
	m := NewSassMap(map[Value]Value{
		&SassString{Text: "a", HasQuotes: true}: NewUnitlessNumber(1),
	})
	result, err := m.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("AsList length = %d, want 1", len(result))
	}
	pair, ok := result[0].(*SassList)
	if !ok {
		t.Fatal("AsList element should be a list")
	}
	if pair.LengthAsList() != 2 {
		t.Error("each pair should have 2 elements")
	}
}

func TestMapEquals(t *testing.T) {
	m1 := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	m2 := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	if !m1.Equals(m2) {
		t.Error("identical maps should be equal")
	}
	m3 := NewSassMap(map[Value]Value{&SassString{Text: "b"}: NewUnitlessNumber(1)})
	if m1.Equals(m3) {
		t.Error("different maps should not be equal")
	}
	if m1.Equals(NewUnitlessNumber(1)) {
		t.Error("map should not equal number")
	}
}

func TestMapEqualsEmptyVsList(t *testing.T) {
	m := EmptySassMap()
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)
	if !m.Equals(emptyList) {
		t.Error("empty map should equal empty list")
	}
	m2 := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	if m2.Equals(emptyList) {
		t.Error("non-empty map should not equal empty list")
	}
}

func TestMapHashCode(t *testing.T) {
	m := EmptySassMap()
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)
	if m.HashCode() != emptyList.HashCode() {
		t.Error("empty map hash should match empty list hash")
	}
	m2 := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	h1 := m2.HashCode()
	h2 := m2.HashCode()
	if h1 != h2 {
		t.Error("hash should be deterministic (cached)")
	}
}

func TestMapSetInvalidatesHashCache(t *testing.T) {
	k := &SassString{Text: "a", HasQuotes: true}
	m := NewSassMap(map[Value]Value{k: NewUnitlessNumber(1)})
	h1 := m.HashCode()
	m.Set(k, NewUnitlessNumber(2))
	h2 := m.HashCode()
	if h1 == h2 {
		t.Error("Set should invalidate hash cache")
	}
}

func TestMapSeparator(t *testing.T) {
	if EmptySassMap().Separator() != ListSeparatorUndecided {
		t.Error("empty map separator should be Undecided")
	}
	m := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	if m.Separator() != ListSeparatorComma {
		t.Error("non-empty map separator should be Comma")
	}
}

func TestMapNewSassMapFromEntries(t *testing.T) {
	m := NewSassMapFromEntries(func(yield func(Value, Value) bool) {
		yield(&SassString{Text: "a"}, NewUnitlessNumber(1))
		yield(&SassString{Text: "b"}, NewUnitlessNumber(2))
	})
	if m.LengthAsList() != 2 {
		t.Errorf("length = %d, want 2", m.LengthAsList())
	}
}

func TestMapToCssString(t *testing.T) {
	m := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	_, err := m.ToCssString(false)
	if err == nil {
		t.Error("map ToCssString should error")
	}
	if !strings.Contains(err.Error(), "isn't a valid CSS value") {
		t.Errorf("error should mention invalid CSS value: %v", err)
	}
}

func TestMapString(t *testing.T) {
	m := NewSassMap(map[Value]Value{
		&SassString{Text: "a", HasQuotes: true}: NewUnitlessNumber(1),
		&SassString{Text: "b", HasQuotes: true}: NewUnitlessNumber(2),
	})
	got, err := m.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\"a\"") || !strings.Contains(got, "1") {
		t.Errorf("String() = %q, should contain key and value", got)
	}
	empty := EmptySassMap()
	got, err = empty.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "()" {
		t.Errorf("empty map String() = %q, want %q", got, "()")
	}
}

func TestMapOperators(t *testing.T) {
	m := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	_, err := m.Plus(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Plus should error for map (CSS serialization fails)")
	}
	_, err = m.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error")
	}
}

func TestMapTryMap(t *testing.T) {
	m := NewSassMap(map[Value]Value{&SassString{Text: "a"}: NewUnitlessNumber(1)})
	if m.TryMap() != m {
		t.Error("TryMap on a map should return the map itself")
	}
	empty := EmptySassMap()
	if empty.TryMap() != empty {
		t.Error("TryMap on an empty map should return the map itself")
	}
}
