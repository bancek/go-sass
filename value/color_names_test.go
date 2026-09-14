package value

import (
	"testing"
)

func TestColorNameForInts(t *testing.T) {
	if name := ColorNameForInts(255, 0, 0); name != "red" {
		t.Errorf("ColorNameForInts(255,0,0) = %q, want %q", name, "red")
	}
	if name := ColorNameForInts(0, 0, 0); name != "black" {
		t.Errorf("ColorNameForInts(0,0,0) = %q, want %q", name, "black")
	}
	if name := ColorNameForInts(255, 255, 255); name != "white" {
		t.Errorf("ColorNameForInts(255,255,255) = %q, want %q", name, "white")
	}
	if name := ColorNameForInts(1, 2, 3); name != "" {
		t.Errorf("unknown color should return empty: got %q", name)
	}
}

func TestColorNameFor(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	name, err := ColorNameFor(c)
	if err != nil {
		t.Fatal(err)
	}
	if name != "red" {
		t.Errorf("ColorNameFor(red) = %q", name)
	}

	c2, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	name2, err := ColorNameFor(c2)
	if err == nil {
		t.Error("non-legacy color should error")
	}
	_ = name2
}

func TestColorNameForGrayWins(t *testing.T) {
	// gray wins over grey (both 0x80,0x80,0x80)
	if name := ColorNameForInts(0x80, 0x80, 0x80); name != "gray" {
		t.Errorf("ColorNameForInts(128,128,128) = %q, want %q", name, "gray")
	}
}

func TestTransparentAndBlack(t *testing.T) {
	// rgba(0,0,0,0) should be "transparent" per Dart semantics
	tr, _ := NewColorRGB(0, 0, 0, 0)
	name := colorNameForSassColor(tr)
	if name != "transparent" {
		t.Errorf("rgba(0,0,0,0) = %q, want %q", name, "transparent")
	}

	// ColorNameFor should also return transparent for alpha=0
	name2, err := ColorNameFor(tr)
	if err != nil {
		t.Fatal(err)
	}
	if name2 != "transparent" {
		t.Errorf("ColorNameFor(rgba(0,0,0,0)) = %q, want %q", name2, "transparent")
	}

	// rgb(0,0,0,1) should be "black" (opaque)
	bl, _ := NewColorRGB(0, 0, 0, 1)
	name3, _ := ColorNameFor(bl)
	if name3 != "black" {
		t.Errorf("rgb(0,0,0,1) = %q, want %q", name3, "black")
	}
}
