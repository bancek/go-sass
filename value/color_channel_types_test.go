package value

import "testing"

func TestNewColorChannel(t *testing.T) {
	ch := NewColorChannel("lightness", false, "%")
	if ch.Name != "lightness" {
		t.Errorf("Name = %q, want %q", ch.Name, "lightness")
	}
	if ch.IsPolarAngle {
		t.Error("IsPolarAngle should be false")
	}
	if ch.AssociatedUnit != "%" {
		t.Errorf("AssociatedUnit = %q, want %q", ch.AssociatedUnit, "%")
	}
}

func TestNewColorChannelPolarAngle(t *testing.T) {
	ch := NewColorChannel("hue", true, "deg")
	if !ch.IsPolarAngle {
		t.Error("IsPolarAngle should be true")
	}
	if ch.AssociatedUnit != "deg" {
		t.Errorf("AssociatedUnit = %q, want %q", ch.AssociatedUnit, "deg")
	}
}

func TestColorChannelIsAnalogousRed(t *testing.T) {
	red := NewColorChannel("red", false, "")
	x := NewColorChannel("x", false, "")
	if !red.IsAnalogous(x) {
		t.Error("red should be analogous to x")
	}
	if !x.IsAnalogous(red) {
		t.Error("x should be analogous to red")
	}
	if !red.IsAnalogous(red) {
		t.Error("red should be analogous to red")
	}
	if !x.IsAnalogous(x) {
		t.Error("x should be analogous to x")
	}
}

func TestColorChannelIsAnalogousGreen(t *testing.T) {
	green := NewColorChannel("green", false, "")
	y := NewColorChannel("y", false, "")
	if !green.IsAnalogous(y) {
		t.Error("green should be analogous to y")
	}
	if !y.IsAnalogous(green) {
		t.Error("y should be analogous to green")
	}
}

func TestColorChannelIsAnalogousBlue(t *testing.T) {
	blue := NewColorChannel("blue", false, "")
	z := NewColorChannel("z", false, "")
	if !blue.IsAnalogous(z) {
		t.Error("blue should be analogous to z")
	}
	if !z.IsAnalogous(blue) {
		t.Error("z should be analogous to blue")
	}
}

func TestColorChannelIsAnalogousChromaSaturation(t *testing.T) {
	chroma := NewColorChannel("chroma", false, "")
	saturation := NewColorChannel("saturation", false, "")
	if !chroma.IsAnalogous(saturation) {
		t.Error("chroma should be analogous to saturation")
	}
	if !saturation.IsAnalogous(chroma) {
		t.Error("saturation should be analogous to chroma")
	}
	if !chroma.IsAnalogous(chroma) {
		t.Error("chroma should be analogous to chroma")
	}
}

func TestColorChannelIsAnalogousLightness(t *testing.T) {
	lightness := NewColorChannel("lightness", false, "%")
	otherLightness := NewColorChannel("lightness", false, "")
	if !lightness.IsAnalogous(otherLightness) {
		t.Error("lightness should be analogous to lightness")
	}
}

func TestColorChannelIsAnalogousHue(t *testing.T) {
	hue := NewColorChannel("hue", true, "deg")
	otherHue := NewColorChannel("hue", true, "")
	if !hue.IsAnalogous(otherHue) {
		t.Error("hue should be analogous to hue")
	}
}

func TestColorChannelIsAnalogousNotAnalogous(t *testing.T) {
	red := NewColorChannel("red", false, "")
	green := NewColorChannel("green", false, "")
	if red.IsAnalogous(green) {
		t.Error("red should not be analogous to green")
	}
	lightness := NewColorChannel("lightness", false, "%")
	hue := NewColorChannel("hue", true, "deg")
	if lightness.IsAnalogous(hue) {
		t.Error("lightness should not be analogous to hue")
	}
	chroma := NewColorChannel("chroma", false, "")
	red2 := NewColorChannel("red", false, "")
	if chroma.IsAnalogous(red2) {
		t.Error("chroma should not be analogous to red")
	}
}

func TestNewLinearChannel(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1)
	if ch.Name != "red" {
		t.Errorf("Name = %q, want %q", ch.Name, "red")
	}
	if ch.Min != 0 {
		t.Errorf("Min = %v, want 0", ch.Min)
	}
	if ch.Max != 1 {
		t.Errorf("Max = %v, want 1", ch.Max)
	}
	if ch.IsPolarAngle {
		t.Error("IsPolarAngle should be false")
	}
	if ch.RequiresPercent {
		t.Error("RequiresPercent should be false (default)")
	}
	if ch.LowerClamped {
		t.Error("LowerClamped should be false (default)")
	}
	if ch.UpperClamped {
		t.Error("UpperClamped should be false (default)")
	}
}

func TestNewLinearChannelPercentAuto(t *testing.T) {
	ch := NewLinearChannel("lightness", 0, 100)
	if ch.AssociatedUnit != "%" {
		t.Errorf("AssociatedUnit = %q, want %q", ch.AssociatedUnit, "%")
	}
}

func TestNewLinearChannelNoPercentAuto(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1)
	if ch.AssociatedUnit != "" {
		t.Errorf("AssociatedUnit = %q, want empty", ch.AssociatedUnit)
	}
}

func TestNewLinearChannelWithOptions(t *testing.T) {
	ch := NewLinearChannel("lightness", 0, 100,
		WithRequiresPercent,
		WithLowerClamped,
		WithUpperClamped,
	)
	if !ch.RequiresPercent {
		t.Error("RequiresPercent should be true")
	}
	if !ch.LowerClamped {
		t.Error("LowerClamped should be true")
	}
	if !ch.UpperClamped {
		t.Error("UpperClamped should be true")
	}
}

func TestNewLinearChannelWithConventionallyPercent(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1)
	ch2 := NewLinearChannel("red", 0, 1, WithConventionallyPercent)
	if ch2.AssociatedUnit != "%" {
		t.Errorf("AssociatedUnit = %q, want %q", ch2.AssociatedUnit, "%")
	}
	if ch.AssociatedUnit != "" {
		t.Error("original AssociatedUnit should still be empty")
	}
}

func TestNewLinearChannelWithNoPercent(t *testing.T) {
	ch := NewLinearChannel("lightness", 0, 100, WithNoPercent)
	if ch.AssociatedUnit != "" {
		t.Errorf("AssociatedUnit = %q, want empty after WithNoPercent", ch.AssociatedUnit)
	}
}

func TestAlphaChannel(t *testing.T) {
	if AlphaChannel.Name != "alpha" {
		t.Errorf("AlphaChannel.Name = %q, want %q", AlphaChannel.Name, "alpha")
	}
	if AlphaChannel.Min != 0 {
		t.Errorf("AlphaChannel.Min = %v, want 0", AlphaChannel.Min)
	}
	if AlphaChannel.Max != 1 {
		t.Errorf("AlphaChannel.Max = %v, want 1", AlphaChannel.Max)
	}
	if AlphaChannel.IsPolarAngle {
		t.Error("AlphaChannel.IsPolarAngle should be false")
	}
}

func TestHueChannelInfo(t *testing.T) {
	ch := HueChannelInfo()
	if ch.Name != "hue" {
		t.Errorf("Name = %q, want %q", ch.Name, "hue")
	}
	if !ch.IsPolarAngle {
		t.Error("IsPolarAngle should be true")
	}
	if ch.AssociatedUnit != "deg" {
		t.Errorf("AssociatedUnit = %q, want %q", ch.AssociatedUnit, "deg")
	}
	if ch.RequiresPercent {
		t.Error("RequiresPercent should default to false")
	}
	if ch.LowerClamped {
		t.Error("LowerClamped should default to false")
	}
	if ch.UpperClamped {
		t.Error("UpperClamped should default to false")
	}
}

func TestLinearChannelRequiresPercentOption(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1, WithRequiresPercent)
	if !ch.RequiresPercent {
		t.Error("RequiresPercent should be true")
	}
	if ch.LowerClamped || ch.UpperClamped {
		t.Error("other options should not be affected")
	}
}

func TestLinearChannelLowerClampedOption(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1, WithLowerClamped)
	if !ch.LowerClamped {
		t.Error("LowerClamped should be true")
	}
}

func TestLinearChannelUpperClampedOption(t *testing.T) {
	ch := NewLinearChannel("red", 0, 1, WithUpperClamped)
	if !ch.UpperClamped {
		t.Error("UpperClamped should be true")
	}
}

func TestLinearChannelAllOptionCombo(t *testing.T) {
	ch := NewLinearChannel("lightness", 0, 100,
		WithRequiresPercent,
		WithLowerClamped,
		WithUpperClamped,
		WithNoPercent,
	)
	if !ch.RequiresPercent {
		t.Error("RequiresPercent should be true")
	}
	if !ch.LowerClamped {
		t.Error("LowerClamped should be true")
	}
	if !ch.UpperClamped {
		t.Error("UpperClamped should be true")
	}
	if ch.AssociatedUnit != "" {
		t.Errorf("AssociatedUnit = %q, want empty (WithNoPercent overrides auto)", ch.AssociatedUnit)
	}
}
