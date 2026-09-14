package linkedhashmap

import (
	"fmt"
	"slices"
	"testing"
)

// Test key types satisfying HashCoder.

type intKey int

func (k intKey) HashCode() int { return int(k) }

type strKey string

func (k strKey) HashCode() int {
	h := 0
	for _, c := range string(k) {
		h = h*31 + int(c)
	}
	return h
}

// Equality functions for test key types.

func intEquals(a, b intKey) bool { return a == b }
func strEquals(a, b strKey) bool { return a == b }

func TestPutGet(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Errorf("Get(a) = %d, %v; want 1, true", v, ok)
	}
}

func TestGetMissing(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	if v, ok := m.Get("x"); ok || v != 0 {
		t.Errorf("Get(x) = %d, %v; want 0, false", v, ok)
	}
}

func TestPutOverwrite(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("a", 10)

	if v, ok := m.Get("a"); !ok || v != 10 {
		t.Errorf("Get(a) = %d, %v; want 10, true", v, ok)
	}
	keys := slices.Collect(m.Keys())
	if !slices.Equal(keys, []strKey{"a", "b"}) {
		t.Errorf("Keys() = %v; want [a b] (position unchanged after overwrite)", keys)
	}
}

func TestDelete(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)
	m.Delete("b")

	if m.Has("b") {
		t.Error("Has(b) = true after Delete; want false")
	}
	keys := slices.Collect(m.Keys())
	if !slices.Equal(keys, []strKey{"a", "c"}) {
		t.Errorf("Keys() = %v; want [a c]", keys)
	}
}

func TestDeleteMissing(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Delete("x")
	if m.Len() != 1 {
		t.Errorf("Len() = %d; want 1", m.Len())
	}
}

func TestDeleteReinsert(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Delete("a")
	m.Put("a", 10)

	keys := slices.Collect(m.Keys())
	if !slices.Equal(keys, []strKey{"b", "a"}) {
		t.Errorf("Keys() = %v; want [b a] (re-inserted key moves to tail)", keys)
	}
	if v, ok := m.Get("a"); !ok || v != 10 {
		t.Errorf("Get(a) = %d, %v; want 10, true", v, ok)
	}
}

func TestOrder100(t *testing.T) {
	m := New[intKey, int](intEquals)
	n := 100
	for i := range n {
		m.Put(intKey(i), i*10)
	}
	keys := slices.Collect(m.Keys())
	for i := range n {
		if keys[i] != intKey(i) {
			t.Fatalf("Keys()[%d] = %d; want %d", i, keys[i], i)
		}
	}
	i := 0
	for k, v := range m.Entries() {
		if k != intKey(i) || v != i*10 {
			t.Fatalf("Entries()[%d] = (%d, %d); want (%d, %d)", i, k, v, i, i*10)
		}
		i++
	}
	if i != n {
		t.Errorf("Entries() yielded %d pairs; want %d", i, n)
	}
}

// collider is a test type where HashCode returns a fixed hash independent of id.
type collider struct {
	id   int
	hash int
}

func (c collider) HashCode() int { return c.hash }

func TestHashCollision(t *testing.T) {
	m := New[collider, string](func(a, b collider) bool { return a.id == b.id })
	m.Put(collider{id: 1, hash: 42}, "one")
	m.Put(collider{id: 2, hash: 42}, "two")

	if v, ok := m.Get(collider{id: 1, hash: 42}); !ok || v != "one" {
		t.Errorf("Get(1) = %q, %v; want %q, true", v, ok, "one")
	}
	if v, ok := m.Get(collider{id: 2, hash: 42}); !ok || v != "two" {
		t.Errorf("Get(2) = %q, %v; want %q, true", v, ok, "two")
	}
	if m.Len() != 2 {
		t.Errorf("Len() = %d; want 2", m.Len())
	}
}

func TestCopy(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)

	c := m.Copy()
	m.Put("c", 3)
	m.Delete("a")

	if c.Len() != 2 {
		t.Errorf("Copy Len() = %d; want 2", c.Len())
	}
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Errorf("Copy Get(a) = %d, %v; want 1, true", v, ok)
	}
	if c.Has("c") {
		t.Error("Copy Has(c) = true; want false (original mutates, copy unchanged)")
	}
}

func TestCopyOrder(t *testing.T) {
	m := New[intKey, int](intEquals)
	for i := range 5 {
		m.Put(intKey(i), i)
	}
	c := m.Copy()
	orig := slices.Collect(m.Keys())
	copied := slices.Collect(c.Keys())
	if !slices.Equal(orig, copied) {
		t.Errorf("Copy Keys() = %v; want %v", copied, orig)
	}
}

func TestHas(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Delete("b")

	if !m.Has("a") {
		t.Error("Has(a) = false; want true")
	}
	if m.Has("b") {
		t.Error("Has(b) = true after Delete; want false")
	}
	if m.Has("x") {
		t.Error("Has(x) = true; want false")
	}
}

func TestLen(t *testing.T) {
	m := New[strKey, int](strEquals)
	if m.Len() != 0 {
		t.Errorf("Len() = %d; want 0", m.Len())
	}
	m.Put("a", 1)
	if m.Len() != 1 {
		t.Errorf("Len() = %d after Put; want 1", m.Len())
	}
	m.Put("a", 2)
	if m.Len() != 1 {
		t.Errorf("Len() = %d after overwrite; want 1", m.Len())
	}
	m.Put("b", 3)
	if m.Len() != 2 {
		t.Errorf("Len() = %d after second Put; want 2", m.Len())
	}
	m.Delete("a")
	if m.Len() != 1 {
		t.Errorf("Len() = %d after Delete; want 1", m.Len())
	}
}

func TestNilMap(t *testing.T) {
	var m *LinkedHashMap[strKey, int]
	if v, ok := m.Get("a"); ok || v != 0 {
		t.Errorf("nil.Get = %d, %v; want 0, false", v, ok)
	}
	if m.Has("a") {
		t.Error("nil.Has = true; want false")
	}
	if l := m.Len(); l != 0 {
		t.Errorf("nil.Len = %d; want 0", l)
	}
	for range m.Keys() {
		t.Error("nil.Keys yielded; want empty")
	}
	for range m.Entries() {
		t.Error("nil.Entries yielded; want empty")
	}
	m.Delete("a")
	if c := m.Copy(); c != nil {
		t.Error("nil.Copy = non-nil; want nil")
	}
}

func TestDeleteEmpty(t *testing.T) {
	m := New[strKey, int](strEquals)
	m.Delete("a")
}

func TestPutDeleteManyRoundtrip(t *testing.T) {
	m := New[strKey, string](strEquals)
	n := 500
	for i := range n {
		k := strKey(fmt.Sprintf("key-%d", i))
		m.Put(k, fmt.Sprintf("val-%d", i))
	}
	if m.Len() != n {
		t.Fatalf("Len() = %d; want %d", m.Len(), n)
	}
	for i := range n {
		k := strKey(fmt.Sprintf("key-%d", i))
		if v, ok := m.Get(k); !ok || v != fmt.Sprintf("val-%d", i) {
			t.Errorf("Get(%s) = %q, %v; want val-%d, true", k, v, ok, i)
		}
	}
	for i := 0; i < n; i += 2 {
		m.Delete(strKey(fmt.Sprintf("key-%d", i)))
	}
	if m.Len() != n/2 {
		t.Errorf("Len() = %d after deleting half; want %d", m.Len(), n/2)
	}
	for i := range n {
		k := strKey(fmt.Sprintf("key-%d", i))
		if i%2 == 0 {
			if m.Has(k) {
				t.Errorf("Has(%s) = true after Delete; want false", k)
			}
		} else {
			if v, ok := m.Get(k); !ok || v != fmt.Sprintf("val-%d", i) {
				t.Errorf("Get(%s) = %q, %v; want val-%d, true", k, v, ok, i)
			}
		}
	}
}
