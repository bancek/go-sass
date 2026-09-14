package value

import (
	"testing"
)

func TestHueToRgb(t *testing.T) {
	m1, m2 := 0.2, 0.7
	tests := []struct {
		hue  float64
		want string
	}{
		{hue: 0.0 / 6.0, want: "0.2"},
		{hue: 0.5 / 6.0, want: "0.45"},
		{hue: 1.0 / 6.0, want: "0.7"},
		{hue: 2.0 / 6.0, want: "0.7"},
		{hue: 3.0 / 6.0, want: "0.7"},
		{hue: 3.5 / 6.0, want: "0.45"},
		{hue: 4.0 / 6.0, want: "0.2"},
		{hue: 5.0 / 6.0, want: "0.2"},
		{hue: 1.0, want: "0.2"},
	}
	for _, tt := range tests {
		if got := ws(hueToRgb(m1, m2, tt.hue)); got != tt.want {
			t.Errorf("hueToRgb(%v, %v, %v) = %s, want %s", m1, m2, tt.hue, got, tt.want)
		}
	}
}

func TestHueToRgbWraps(t *testing.T) {
	m1, m2 := 0.2, 0.7
	if got := ws(hueToRgb(m1, m2, 0.5/6.0)); got != "0.45" {
		t.Errorf("hueToRgb(%v, %v, 0.5/6) = %s, want 0.45", m1, m2, got)
	}
	if got := ws(hueToRgb(m1, m2, -5.5/6.0)); got != "0.45" {
		t.Errorf("hueToRgb(%v, %v, -5.5/6) = %s, want 0.45", m1, m2, got)
	}
	if got := ws(hueToRgb(m1, m2, 1.0/6.0)); got != "0.7" {
		t.Errorf("hueToRgb(%v, %v, 1/6) = %s, want 0.7", m1, m2, got)
	}
	if got := ws(hueToRgb(m1, m2, 7.0/6.0)); got != "0.7" {
		t.Errorf("hueToRgb(%v, %v, 7/6) = %s, want 0.7", m1, m2, got)
	}
}

func TestSrgbAndDisplayP3Roundtrip(t *testing.T) {
	for _, v := range []float64{0.0, 0.05, 0.5, 1.0, -0.5} {
		lin := srgbAndDisplayP3ToLinear(v)
		round := srgbAndDisplayP3FromLinear(lin)
		if ws(v) != ws(round) {
			t.Errorf("sRGB roundtrip(%s) = %s, want %s", ws(v), ws(round), ws(v))
		}
	}
	for _, v := range []float64{0.0, 0.005, 0.5, 1.0, -0.5} {
		round := srgbAndDisplayP3ToLinear(srgbAndDisplayP3FromLinear(v))
		if ws(v) != ws(round) {
			t.Errorf("sRGB reverse roundtrip(%s) = %s, want %s", ws(v), ws(round), ws(v))
		}
	}
}

func TestA98Roundtrip(t *testing.T) {
	for _, v := range []float64{0.0, 0.5, 1.0, -0.3} {
		lin := a98ToLinear(v)
		round := a98FromLinear(lin)
		if ws(v) != ws(round) {
			t.Errorf("A98 roundtrip(%s) = %s, want %s", ws(v), ws(round), ws(v))
		}
	}
}

func TestProphotoRoundtrip(t *testing.T) {
	for _, v := range []float64{0.0, 0.001, 16.0 / 512.0, 0.5, 1.0} {
		lin := prophotoToLinear(v)
		round := prophotoFromLinear(lin)
		if ws(v) != ws(round) {
			t.Errorf("ProPhoto roundtrip(%s) = %s, want %s", ws(v), ws(round), ws(v))
		}
	}
}

func TestRec2020Roundtrip(t *testing.T) {
	for _, v := range []float64{0.0, 0.001, 0.1, 0.5, 1.0, -0.3} {
		lin := rec2020ToLinear(v)
		round := rec2020FromLinear(lin)
		if ws(v) != ws(round) {
			t.Errorf("Rec2020 roundtrip(%s) = %s, want %s", ws(v), ws(round), ws(v))
		}
	}
}

func TestCubeRootPreservingSign(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{v: 0.0, want: "0"},
		{v: 1.0, want: "1"},
		{v: 8.0, want: "2"},
		{v: -8.0, want: "-2"},
		{v: 27.0, want: "3"},
		{v: -27.0, want: "-3"},
	}
	for _, tt := range tests {
		if got := ws(cubeRootPreservingSign(tt.v)); got != tt.want {
			t.Errorf("cubeRootPreservingSign(%v) = %s, want %s", tt.v, got, tt.want)
		}
	}
}

func TestLabConstants(t *testing.T) {
	if got := ws(LabKappa); got != "903.2962962963" {
		t.Errorf("LabKappa = %s, want 903.2962962963", got)
	}
	if got := ws(LabEpsilon); got != "0.0088564517" {
		t.Errorf("LabEpsilon = %s, want 0.0088564517", got)
	}
}

func TestXYZChannels(t *testing.T) {
	if len(XYZChannels) != 3 {
		t.Fatalf("len(XYZChannels) = %d, want 3", len(XYZChannels))
	}
	if XYZChannels[0].Name != "x" {
		t.Errorf("XYZChannels[0].Name = %q, want %q", XYZChannels[0].Name, "x")
	}
	if XYZChannels[1].Name != "y" {
		t.Errorf("XYZChannels[1].Name = %q, want %q", XYZChannels[1].Name, "y")
	}
	if XYZChannels[2].Name != "z" {
		t.Errorf("XYZChannels[2].Name = %q, want %q", XYZChannels[2].Name, "z")
	}
}

func TestLabToLch(t *testing.T) {
	lightness := 50.0
	a := 1.0
	b := 0.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.Space() != LchColorSpace {
		t.Errorf("space = %v, want Lch", c.Space().Name())
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("lightness = %s, want 50", got)
	}
	if got := ws(c.Channel1()); got != "1" {
		t.Errorf("chroma = %s, want 1", got)
	}
	if got := ws(c.Channel2()); got != "0" {
		t.Errorf("hue = %s, want 0", got)
	}
	if c.Alpha() != 1.0 {
		t.Errorf("alpha = %v, want 1", c.Alpha())
	}
}

func TestLabToLchHue90(t *testing.T) {
	lightness := 50.0
	a := 0.0
	b := 1.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("lightness = %s, want 50", got)
	}
	if got := ws(c.Channel1()); got != "1" {
		t.Errorf("chroma = %s, want 1", got)
	}
	if got := ws(c.Channel2()); got != "90" {
		t.Errorf("hue = %s, want 90", got)
	}
}

func TestLabToLchHue180(t *testing.T) {
	lightness := 50.0
	a := -1.0
	b := 0.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("lightness = %s, want 50", got)
	}
	if got := ws(c.Channel1()); got != "1" {
		t.Errorf("chroma = %s, want 1", got)
	}
	if got := ws(c.Channel2()); got != "180" {
		t.Errorf("hue = %s, want 180", got)
	}
}

func TestLabToLchHue270(t *testing.T) {
	lightness := 50.0
	a := 0.0
	b := -1.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("lightness = %s, want 50", got)
	}
	if got := ws(c.Channel1()); got != "1" {
		t.Errorf("chroma = %s, want 1", got)
	}
	if got := ws(c.Channel2()); got != "270" {
		t.Errorf("hue = %s, want 270", got)
	}
}

func TestLabToLchMissingChroma(t *testing.T) {
	lightness := 50.0
	a := 1.0
	b := 0.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.Channel1OrNil() != nil {
		t.Error("chroma should be missing when missingChroma=true")
	}
}

func TestLabToLchMissingHue(t *testing.T) {
	lightness := 50.0
	a := 1.0
	b := 0.0
	alpha := 1.0
	c, err := labToLch(LchColorSpace, &lightness, &a, &b, &alpha, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Channel2OrNil() != nil {
		t.Error("hue should be missing when missingHue=true")
	}
}

func TestSrgbToLinearNegativeHandle(t *testing.T) {
	got := ws(srgbAndDisplayP3ToLinear(-0.5))
	want := "-0.2140411405"
	if got != want {
		t.Errorf("srgbToLinear(-0.5) = %s, want %s", got, want)
	}
}

func TestProphotoThreshold(t *testing.T) {
	below := 16.0 / 512.0 / 2.0
	above := 16.0 / 512.0 * 2.0
	if got := ws(prophotoToLinear(below)); got != "0.0009765625" {
		t.Errorf("prophotoToLinear(below) = %s, want 0.0009765625", got)
	}
	if got := ws(prophotoToLinear(above)); got != "0.0068011763" {
		t.Errorf("prophotoToLinear(above) = %s, want 0.0068011763", got)
	}
}

func TestRec2020Gamma240(t *testing.T) {
	// Since dart-sass 1.102 (#2729) rec2020 uses the plain 2.4-gamma power law.
	// Goldens below are Dart CLI output for color(rec2020 0.5 0.25 0.75) -> srgb.
	c, err := convertColor(Rec2020ColorSpace, SrgbColorSpace, 0.5, 0.25, 0.75, 1.0, [4]bool{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "0.5439373396" {
		t.Errorf("channel0 = %s, want 0.5439373396", got)
	}
	if got := ws(c.Channel1()); got != "0.1170946921" {
		t.Errorf("channel1 = %s, want 0.1170946921", got)
	}
	if got := ws(c.Channel2()); got != "0.7697591314" {
		t.Errorf("channel2 = %s, want 0.7697591314", got)
	}
}
