package orderedmap_test

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
)

// orderedset-like and map types must satisfy the key-only view so callers can
// pass whichever they hold.
var (
	_ orderedmap.MapKeySet[string] = orderedmap.New[string, int]()
	_ orderedmap.MapKeySet[string] = orderedmap.New[string, struct{}]()
)

func TestMapKeySetKeysInOrder(t *testing.T) {
	m := orderedmap.New[string, int]()
	for _, key := range []string{"b", "a", "c"} {
		m.Put(key, 0)
	}

	var keys []string
	for key := range m.Keys() {
		keys = append(keys, key)
	}
	if len(keys) != 3 || keys[0] != "b" || keys[1] != "a" || keys[2] != "c" {
		t.Errorf("Keys() = %v, want [b a c]", keys)
	}
	if !m.Has("a") || m.Has("z") {
		t.Errorf("Has mismatch")
	}
	if m.Len() != 3 {
		t.Errorf("Len() = %d, want 3", m.Len())
	}
}
