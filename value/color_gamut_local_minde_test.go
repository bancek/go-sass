package value

import (
	"testing"
)

func TestLocalMinde(t *testing.T) {
	c, _ := NewColorSRGB(1.5, 0.1, 0.1, 1)
	result, _ := localMindeGamutMap(c)
	if got := ws(result.Channel0()); got != "1" {
		t.Errorf("localMinde(1.5,0.1,0.1) ch0 = %s, want 1", got)
	}
	if got := ws(result.Channel1()); got != "0.7237814013" {
		t.Errorf("localMinde(1.5,0.1,0.1) ch1 = %s, want 0.7237814013", got)
	}
	if got := ws(result.Channel2()); got != "0.6738285316" {
		t.Errorf("localMinde(1.5,0.1,0.1) ch2 = %s, want 0.6738285316", got)
	}

	c2, _ := NewColorSRGB(2.0, -0.5, 0.0, 1)
	result2, _ := localMindeGamutMap(c2)
	if got := ws(result2.Channel0()); got != "1" {
		t.Errorf("localMinde(2.0,-0.5,0) ch0 = %s, want 1", got)
	}
	if got := ws(result2.Channel1()); got != "1" {
		t.Errorf("localMinde(2.0,-0.5,0) ch1 = %s, want 1", got)
	}
	if got := ws(result2.Channel2()); got != "1" {
		t.Errorf("localMinde(2.0,-0.5,0) ch2 = %s, want 1", got)
	}
}

func TestDeltaEOK(t *testing.T) {
	c1, _ := NewColorSRGB(1.0, 0.0, 0.0, 1)
	c2, _ := NewColorSRGB(0.9, 0.0, 0.0, 1)
	de, _ := deltaEOK(c1, c2)
	if got := ws(de); got != "0.0519781045" {
		t.Errorf("deltaEOK = %s, want 0.0519781045", got)
	}
}
