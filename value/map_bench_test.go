package value

import (
	"fmt"
	"testing"
)

// --- Helpers ---

func makeStringMap(n int) *SassMap {
	m := EmptySassMap()
	for i := range n {
		m.Set(
			&SassString{Text: fmt.Sprintf("key-%06d", i), HasQuotes: false},
			&SassString{Text: fmt.Sprintf("val-%06d", i), HasQuotes: false},
		)
	}
	return m
}

func makeNestedMap(n int) *SassMap {
	m := EmptySassMap()
	for i := range n {
		m.Set(
			&SassString{Text: fmt.Sprintf("k-%06d", i), HasQuotes: false},
			&SassString{Text: fmt.Sprintf("v-%06d", i), HasQuotes: false},
		)
	}
	return m
}

func makeStringList(n int) *SassList {
	contents := make([]Value, n)
	for i := range n {
		contents[i] = &SassString{Text: fmt.Sprintf("item-%06d", i), HasQuotes: false}
	}
	l, _ := NewSassList(contents, ListSeparatorComma, false)
	return l
}

// --- Standard benchmarks ---

func BenchmarkMapGet_10(b *testing.B)    { benchMapGet(b, 10) }
func BenchmarkMapGet_100(b *testing.B)   { benchMapGet(b, 100) }
func BenchmarkMapGet_1000(b *testing.B)  { benchMapGet(b, 1000) }
func BenchmarkMapGet_10000(b *testing.B) { benchMapGet(b, 10000) }

func BenchmarkMapSet_100(b *testing.B)  { benchMapSet(b, 100) }
func BenchmarkMapSet_1000(b *testing.B) { benchMapSet(b, 1000) }

func BenchmarkMapDelete_100(b *testing.B)  { benchMapDelete(b, 100) }
func BenchmarkMapDelete_1000(b *testing.B) { benchMapDelete(b, 1000) }

func BenchmarkMapContains_100(b *testing.B)  { benchMapContains(b, 100) }
func BenchmarkMapContains_1000(b *testing.B) { benchMapContains(b, 1000) }

func BenchmarkMapEqual_10(b *testing.B)   { benchMapEqual(b, 10) }
func BenchmarkMapEqual_100(b *testing.B)  { benchMapEqual(b, 100) }
func BenchmarkMapEqual_1000(b *testing.B) { benchMapEqual(b, 1000) }

func BenchmarkMapEntries_100(b *testing.B)  { benchMapEntries(b, 100) }
func BenchmarkMapEntries_1000(b *testing.B) { benchMapEntries(b, 1000) }

func BenchmarkMapAsList_100(b *testing.B)  { benchMapAsList(b, 100) }
func BenchmarkMapAsList_1000(b *testing.B) { benchMapAsList(b, 1000) }

// --- Composite key benchmarks: map as key ---

func BenchmarkMapAsKey_SetGet_10(b *testing.B)  { benchMapAsKeySetGet(b, 10) }
func BenchmarkMapAsKey_SetGet_100(b *testing.B) { benchMapAsKeySetGet(b, 100) }

func BenchmarkMapAsKey_SetOverwrite_10(b *testing.B)  { benchMapAsKeySetOverwrite(b, 10) }
func BenchmarkMapAsKey_SetOverwrite_100(b *testing.B) { benchMapAsKeySetOverwrite(b, 100) }

func BenchmarkMapAsKey_Get_10(b *testing.B)  { benchMapAsKeyGet(b, 10) }
func BenchmarkMapAsKey_Get_100(b *testing.B) { benchMapAsKeyGet(b, 100) }

// --- Composite key benchmarks: list of strings as key ---

func BenchmarkListAsKey_SetGet_10(b *testing.B)  { benchListAsKeySetGet(b, 10) }
func BenchmarkListAsKey_SetGet_100(b *testing.B) { benchListAsKeySetGet(b, 100) }

func BenchmarkListAsKey_SetOverwrite_10(b *testing.B)  { benchListAsKeySetOverwrite(b, 10) }
func BenchmarkListAsKey_SetOverwrite_100(b *testing.B) { benchListAsKeySetOverwrite(b, 100) }

func BenchmarkListAsKey_Get_10(b *testing.B)  { benchListAsKeyGet(b, 10) }
func BenchmarkListAsKey_Get_100(b *testing.B) { benchListAsKeyGet(b, 100) }

// --- Standard helpers ---

func benchMapGet(b *testing.B, n int) {
	m := makeStringMap(n)
	key := &SassString{Text: fmt.Sprintf("key-%06d", n/2), HasQuotes: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(key)
	}
}

func benchMapSet(b *testing.B, n int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := EmptySassMap()
		for j := range n {
			m.Set(
				&SassString{Text: fmt.Sprintf("k-%d-%d", i, j), HasQuotes: false},
				&SassString{Text: fmt.Sprintf("v-%d-%d", i, j), HasQuotes: false},
			)
		}
	}
}

func benchMapDelete(b *testing.B, n int) {
	key := &SassString{Text: fmt.Sprintf("key-%06d", n/2), HasQuotes: false}
	val := &SassString{Text: fmt.Sprintf("val-%06d", n/2), HasQuotes: false}
	m := makeStringMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Delete(key)
		m.Set(key, val)
	}
}

func benchMapContains(b *testing.B, n int) {
	m := makeStringMap(n)
	key := &SassString{Text: fmt.Sprintf("key-%06d", n/2), HasQuotes: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Contains(key)
	}
}

func benchMapEqual(b *testing.B, n int) {
	m1 := makeStringMap(n)
	m2 := makeStringMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m1.Equals(m2)
	}
}

func benchMapEntries(b *testing.B, n int) {
	m := makeStringMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k, v := range m.Entries() {
			_ = k
			_ = v
		}
	}
}

func benchMapAsList(b *testing.B, n int) {
	m := makeStringMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.AsList()
	}
}

// --- Map-as-key helpers ---

func benchMapAsKeySetGet(b *testing.B, n int) {
	inner := makeNestedMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := EmptySassMap()
		m.Set(inner, &SassString{Text: "val", HasQuotes: false})
		m.Get(inner)
	}
}

func benchMapAsKeySetOverwrite(b *testing.B, n int) {
	inner1 := makeNestedMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := EmptySassMap()
		m.Set(inner1, &SassString{Text: "v1", HasQuotes: false})
		inner2 := makeNestedMap(n)
		m.Set(inner2, &SassString{Text: "v2", HasQuotes: false})
	}
}

func benchMapAsKeyGet(b *testing.B, n int) {
	inner := makeNestedMap(n)
	outer := EmptySassMap()
	outer.Set(inner, &SassString{Text: "val", HasQuotes: false})
	lookup := makeNestedMap(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outer.Get(lookup)
	}
}

// --- List-as-key helpers ---

func benchListAsKeySetGet(b *testing.B, n int) {
	listKey := makeStringList(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := EmptySassMap()
		m.Set(listKey, &SassString{Text: "val", HasQuotes: false})
		m.Get(listKey)
	}
}

func benchListAsKeySetOverwrite(b *testing.B, n int) {
	list1 := makeStringList(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := EmptySassMap()
		m.Set(list1, &SassString{Text: "v1", HasQuotes: false})
		list2 := makeStringList(n)
		m.Set(list2, &SassString{Text: "v2", HasQuotes: false})
	}
}

func benchListAsKeyGet(b *testing.B, n int) {
	list1 := makeStringList(n)
	outer := EmptySassMap()
	outer.Set(list1, &SassString{Text: "val", HasQuotes: false})
	lookup := makeStringList(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outer.Get(lookup)
	}
}
