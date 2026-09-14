package value

import (
	"testing"
)

func TestClipGamut(t *testing.T) {
	c, _ := NewColorSRGB(1.5, -0.5, 0.5, 1)
	result := clipGamutMap(c)
	if got := ws(result.Channel0()); got != "1" {
		t.Errorf("clip(1.5,-0.5,0.5) ch0 = %s, want 1", got)
	}
	if got := ws(result.Channel1()); got != "0" {
		t.Errorf("clip(1.5,-0.5,0.5) ch1 = %s, want 0", got)
	}
	if got := ws(result.Channel2()); got != "0.5" {
		t.Errorf("clip(1.5,-0.5,0.5) ch2 = %s, want 0.5", got)
	}

	c2, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	result2 := clipGamutMap(c2)
	if got := ws(result2.Channel0()); got != "0.5" {
		t.Errorf("clip(0.5,0.5,0.5) ch0 = %s, want 0.5", got)
	}
	if got := ws(result2.Channel1()); got != "0.5" {
		t.Errorf("clip(0.5,0.5,0.5) ch1 = %s, want 0.5", got)
	}
	if got := ws(result2.Channel2()); got != "0.5" {
		t.Errorf("clip(0.5,0.5,0.5) ch2 = %s, want 0.5", got)
	}
}
