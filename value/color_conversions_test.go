package value

import (
	"testing"
)

func TestD50(t *testing.T) {
	if got := ws(D50[0]); got != "0.9642956764" {
		t.Errorf("D50[0] = %s, want 0.9642956764", got)
	}
	if got := ws(D50[1]); got != "1" {
		t.Errorf("D50[1] = %s, want 1", got)
	}
	if got := ws(D50[2]); got != "0.8251046025" {
		t.Errorf("D50[2] = %s, want 0.8251046025", got)
	}
}

func TestMatrixMul(t *testing.T) {
	r1, g1, b1 := matrixMul(linearSrgbToXyzD65[:], 1.0, 0.0, 0.0)
	if got := ws(r1); got != "0.4123907993" {
		t.Errorf("matrixMul(1,0,0) r = %s, want 0.4123907993", got)
	}
	if got := ws(g1); got != "0.2126390059" {
		t.Errorf("matrixMul(1,0,0) g = %s, want 0.2126390059", got)
	}
	if got := ws(b1); got != "0.0193308187" {
		t.Errorf("matrixMul(1,0,0) b = %s, want 0.0193308187", got)
	}
	r2, g2, b2 := matrixMul(linearSrgbToXyzD65[:], 0.0, 1.0, 0.0)
	if got := ws(r2); got != "0.3575843394" {
		t.Errorf("matrixMul(0,1,0) r = %s, want 0.3575843394", got)
	}
	if got := ws(g2); got != "0.7151686788" {
		t.Errorf("matrixMul(0,1,0) g = %s, want 0.7151686788", got)
	}
	if got := ws(b2); got != "0.1191947798" {
		t.Errorf("matrixMul(0,1,0) b = %s, want 0.1191947798", got)
	}
	r3, g3, b3 := matrixMul(linearSrgbToXyzD65[:], 0.0, 0.0, 1.0)
	if got := ws(r3); got != "0.1804807884" {
		t.Errorf("matrixMul(0,0,1) r = %s, want 0.1804807884", got)
	}
	if got := ws(g3); got != "0.0721923154" {
		t.Errorf("matrixMul(0,0,1) g = %s, want 0.0721923154", got)
	}
	if got := ws(b3); got != "0.9505321522" {
		t.Errorf("matrixMul(0,0,1) b = %s, want 0.9505321522", got)
	}
}

func TestConvertColor(t *testing.T) {
	c1, _ := convertColor(SrgbColorSpace, LabColorSpace, 1.0, 0.0, 0.0, 1.0, [4]bool{false, false, false, false}, nil)
	if got := ws(c1.Channel0()); got != "54.2905414047" {
		t.Errorf("sRGB->Lab ch0 = %s, want 54.2905414047", got)
	}
	if got := ws(c1.Channel1()); got != "80.8049281704" {
		t.Errorf("sRGB->Lab ch1 = %s, want 80.8049281704", got)
	}
	if got := ws(c1.Channel2()); got != "69.8909647686" {
		t.Errorf("sRGB->Lab ch2 = %s, want 69.8909647686", got)
	}

	c2, _ := convertColor(SrgbColorSpace, LchColorSpace, 1.0, 0.0, 0.0, 1.0, [4]bool{false, false, false, false}, nil)
	if got := ws(c2.Channel0()); got != "54.2905414047" {
		t.Errorf("sRGB->Lch ch0 = %s, want 54.2905414047", got)
	}
	if got := ws(c2.Channel1()); got != "106.8371816032" {
		t.Errorf("sRGB->Lch ch1 = %s, want 106.8371816032", got)
	}
	if got := ws(c2.Channel2()); got != "40.857656505" {
		t.Errorf("sRGB->Lch ch2 = %s, want 40.857656505", got)
	}

	c3, _ := convertColor(SrgbColorSpace, OklchColorSpace, 0.0, 1.0, 0.0, 1.0, [4]bool{false, false, false, false}, nil)
	if got := ws(c3.Channel0()); got != "0.8664396175" {
		t.Errorf("sRGB->Oklch ch0 = %s, want 0.8664396175", got)
	}
	if got := ws(c3.Channel1()); got != "0.2948272245" {
		t.Errorf("sRGB->Oklch ch1 = %s, want 0.2948272245", got)
	}
	if got := ws(c3.Channel2()); got != "142.4953450414" {
		t.Errorf("sRGB->Oklch ch2 = %s, want 142.4953450414", got)
	}

	c4, _ := convertColor(LabColorSpace, LchColorSpace, 50.0, 25.0, -25.0, 1.0, [4]bool{false, false, false, false}, nil)
	if got := ws(c4.Channel0()); got != "50" {
		t.Errorf("Lab->Lch ch0 = %s, want 50", got)
	}
	if got := ws(c4.Channel1()); got != "35.3553390593" {
		t.Errorf("Lab->Lch ch1 = %s, want 35.3553390593", got)
	}
	if got := ws(c4.Channel2()); got != "315" {
		t.Errorf("Lab->Lch ch2 = %s, want 315", got)
	}

	c5, _ := convertColor(SrgbColorSpace, SrgbColorSpace, 0.3, 0.6, 0.9, 1.0, [4]bool{false, false, false, false}, nil)
	if got := ws(c5.Channel0()); got != "0.3" {
		t.Errorf("sRGB->sRGB ch0 = %s, want 0.3", got)
	}
	if got := ws(c5.Channel1()); got != "0.6" {
		t.Errorf("sRGB->sRGB ch1 = %s, want 0.6", got)
	}
	if got := ws(c5.Channel2()); got != "0.9" {
		t.Errorf("sRGB->sRGB ch2 = %s, want 0.9", got)
	}
}
