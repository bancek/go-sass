package orderedmap

import (
	"slices"
	"testing"
)

func TestLinkedMapClonePreservesOrderAndIsIndependent(t *testing.T) {
	original := New[string, int]()
	original.Put("first", 1)
	original.Put("second", 2)

	clone := original.Clone()
	clone.Put("third", 3)
	clone.Put("first", 10)

	var originalKeys []string
	for key := range original.Keys() {
		originalKeys = append(originalKeys, key)
	}
	if want := []string{"first", "second"}; !slices.Equal(originalKeys, want) {
		t.Errorf("original keys = %v, want %v", originalKeys, want)
	}
	if got, _ := original.Get("first"); got != 1 {
		t.Errorf("original first = %d, want 1", got)
	}

	var cloneKeys []string
	for key := range clone.Keys() {
		cloneKeys = append(cloneKeys, key)
	}
	if want := []string{"first", "second", "third"}; !slices.Equal(cloneKeys, want) {
		t.Errorf("clone keys = %v, want %v", cloneKeys, want)
	}
	if got, _ := clone.Get("first"); got != 10 {
		t.Errorf("clone first = %d, want 10", got)
	}
}

func TestNilLinkedMapCloneIsEmpty(t *testing.T) {
	var original *LinkedMap[string, int]
	clone := original.Clone()
	if clone == nil || clone.Len() != 0 {
		t.Errorf("Clone() = %v, want empty map", clone)
	}
}
