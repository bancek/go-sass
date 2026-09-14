package value

import "testing"

func TestIsLegacy(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	if !c.IsLegacy() {
		t.Error("RGB should be legacy")
	}
	srgb, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	if srgb.IsLegacy() {
		t.Error("sRGB should not be legacy")
	}
}

func TestIsInGamut(t *testing.T) {
	c, _ := NewColorSRGB(0.5, 0.5, 0.5, 1)
	if !c.IsInGamut() {
		t.Error("in-range sRGB should be in gamut")
	}
	lab, _ := NewColorLab(50, 0, 0, 1)
	if !lab.IsInGamut() {
		t.Error("Lab is unbounded, always in gamut")
	}
}

func TestHasMissingChannel(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	if c.HasMissingChannel() {
		t.Error("full color should not have missing channels")
	}
	miss, _ := NewColorForSpace(RgbColorSpace, [3]float64{0, 0, 0}, 1, [4]bool{true, false, false, false})
	if !miss.HasMissingChannel() {
		t.Error("should have missing channel")
	}
}

func TestChannelByName(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	v, err := c.ChannelByName("red")
	if err != nil {
		t.Fatal(err)
	}
	if v != 255 {
		t.Errorf("ChannelByName(red) = %v", v)
	}
	_, err = c.ChannelByName("bogus")
	if err == nil {
		t.Error("unknown channel should error")
	}
}

func TestIsChannelMissingByName(t *testing.T) {
	miss, _ := NewColorForSpace(RgbColorSpace, [3]float64{0, 0, 0}, 1, [4]bool{true, false, false, false})
	missing, err := miss.IsChannelMissingByName("red")
	if err != nil {
		t.Fatal(err)
	}
	if !missing {
		t.Error("red should be missing")
	}
}

func TestChangeAlpha(t *testing.T) {
	c, _ := NewColorRGB(255, 0, 0, 1)
	c2, err := c.ChangeAlpha(0.5)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Alpha() != 0.5 {
		t.Errorf("alpha = %v, want 0.5", c2.Alpha())
	}
}

func TestChangeChannels(t *testing.T) {
	c, _ := NewColorLab(50, 25, -25, 1)
	v := 75.0
	result, err := c.ChangeChannels(map[string]float64{"lightness": v}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Channel0() != 75 {
		t.Errorf("lightness = %v, want 75", result.Channel0())
	}
}

func TestChannelInfo(t *testing.T) {
	ch := ChannelInfo(SrgbColorSpace, 0)
	if ch.Name != "red" {
		t.Errorf("name = %q, want red", ch.Name)
	}
	alphaCh := ChannelInfo(SrgbColorSpace, 3)
	if alphaCh.Name != "alpha" {
		t.Errorf("name = %q, want alpha", alphaCh.Name)
	}
}
