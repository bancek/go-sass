package value

import (
	"testing"
)

func TestLabSpaceMetadata(t *testing.T) {
	if LabColorSpace.Name() != "lab" {
		t.Error()
	}
	if LabColorSpace.IsBounded() != false {
		t.Error()
	}
	if LabColorSpace.IsLegacy() != false {
		t.Error()
	}
	if LabColorSpace.IsPolar() != false {
		t.Error()
	}
}

func TestLabConvert(t *testing.T) {
	l, a, b := 50.0, 0.0, 0.0
	alpha := 1.0
	c, _ := LabColorSpace.Convert(SrgbColorSpace, &l, &a, &b, &alpha)
	if got := ws(c.Channel0()); got != "0.4663266093" {
		t.Errorf("lab->Srgb ch0 = %s, want 0.4663266093", got)
	}
	if got := ws(c.Channel1()); got != "0.4663266093" {
		t.Errorf("lab->Srgb ch1 = %s, want 0.4663266093", got)
	}
	if got := ws(c.Channel2()); got != "0.4663266093" {
		t.Errorf("lab->Srgb ch2 = %s, want 0.4663266093", got)
	}
	c2, _ := LabColorSpace.Convert(LchColorSpace, &l, &a, &b, &alpha)
	if got := ws(c2.Channel0()); got != "50" {
		t.Errorf("lab->Lch ch0 = %s, want 50", got)
	}
	if got := ws(c2.Channel1()); got != "0" {
		t.Errorf("lab->Lch ch1 = %s, want 0", got)
	}
	if got := ws(c2.Channel2()); got != "0" {
		t.Errorf("lab->Lch ch2 = %s, want 0", got)
	}
}

// Analogous sets of missing channels survive conversion (#2810): the
// changelog example color.to-space(lch(50% none none), lab) is
// lab(50% none none), not lab(50% 0 0).
func TestAnalogousMissingPropagateLchToLab(t *testing.T) {
	c, err := convertColor(LchColorSpace, LabColorSpace, 50, 0, 0, 1, [4]bool{false, true, true, false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("ch0 = %s, want 50", got)
	}
	if !c.IsChannel1Missing() {
		t.Error("ch1 should be missing")
	}
	if !c.IsChannel2Missing() {
		t.Error("ch2 should be missing")
	}
}

func TestAnalogousMissingPropagateLabToLch(t *testing.T) {
	c, err := convertColor(LabColorSpace, LchColorSpace, 50, 0, 0, 1, [4]bool{false, true, true, false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws(c.Channel0()); got != "50" {
		t.Errorf("ch0 = %s, want 50", got)
	}
	if !c.IsChannel1Missing() {
		t.Error("ch1 should be missing")
	}
	if !c.IsChannel2Missing() {
		t.Error("ch2 should be missing")
	}
}
