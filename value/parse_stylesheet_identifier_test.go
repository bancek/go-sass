package value

import (
	"testing"
)

func newTestStylesheetParser(text string) *StylesheetParser {
	return NewStylesheetParser([]byte(text), nil, false, nil)
}

func TestInterpolatedIdentifierSimple(t *testing.T) {
	p := newTestStylesheetParser("foo")
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	if !interp.IsPlain() {
		t.Error("expected plain interpolation")
	}
	if s := interp.AsPlain(); s == nil || *s != "foo" {
		t.Errorf("AsPlain() = %v, want 'foo'", s)
	}
}

func TestInterpolatedIdentifierDoubleDash(t *testing.T) {
	p := newTestStylesheetParser("--prop")
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	if s := interp.AsPlain(); s == nil || *s != "--prop" {
		t.Errorf("AsPlain() = %v, want '--prop'", s)
	}
}

func TestInterpolatedIdentifierUnderscore(t *testing.T) {
	p := newTestStylesheetParser("_foo")
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	if s := interp.AsPlain(); s == nil || *s != "_foo" {
		t.Errorf("AsPlain() = %v, want '_foo'", s)
	}
}

func TestInterpolatedIdentifierEscape(t *testing.T) {
	// \41 = 'A'. Hex escape consumes up to 6 digits, so need
	// non-hex terminator. Just test \41 alone (single-char input).
	p := newTestStylesheetParser(`\41`)
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	plain := interp.AsPlain()
	if plain == nil {
		t.Fatal("AsPlain() returned nil")
	}
	if *plain != `\41` {
		t.Errorf("AsPlain() = %q, want %q", *plain, `\41`)
	}

	expr, err := newTestStylesheetParser(`\41`).interpolatedIdentifier(true)
	if err != nil {
		t.Fatal(err)
	}
	exprPlain := expr.AsPlain()
	if exprPlain == nil || *exprPlain != "A" {
		t.Errorf("expression AsPlain() = %v, want %q", exprPlain, "A")
	}
}

func TestInterpolatedIdentifierEscapeBody(t *testing.T) {
	// f\6fo -> escape \6f = 'o', body reads 'o' -> "foo"
	p := newTestStylesheetParser(`f\6fo`)
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	plain := interp.AsPlain()
	if plain == nil {
		t.Fatal("AsPlain() returned nil")
	}
	if *plain != `f\6fo` {
		t.Errorf("AsPlain() = %q, want %q", *plain, `f\6fo`)
	}

	expr, err := newTestStylesheetParser(`f\6fo`).interpolatedIdentifier(true)
	if err != nil {
		t.Fatal(err)
	}
	exprPlain := expr.AsPlain()
	if exprPlain == nil || *exprPlain != "foo" {
		t.Errorf("expression AsPlain() = %v, want %q", exprPlain, "foo")
	}
}

func TestInterpolatedIdentifierErrorEmpty(t *testing.T) {
	p := newTestStylesheetParser("")
	_, err := p.interpolatedIdentifier(false)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestInterpolatedIdentifierErrorDigit(t *testing.T) {
	p := newTestStylesheetParser("1foo")
	_, err := p.interpolatedIdentifier(false)
	if err == nil {
		t.Error("expected error for digit-start input")
	}
}

func TestInterpolatedIdentifierBodySimple(t *testing.T) {
	p := newTestStylesheetParser("foo")
	p.scanner.ScanChar('f')
	interp, err := p.interpolatedIdentifierBody(false)
	if err != nil {
		t.Fatal(err)
	}
	if s := interp.AsPlain(); s == nil || *s != "oo" {
		t.Errorf("AsPlain() = %v, want 'oo'", s)
	}
}

func TestInterpolatedIdentifierBodyErrorEmpty(t *testing.T) {
	p := newTestStylesheetParser(".")
	p.scanner.ScanChar('.')
	_, err := p.interpolatedIdentifierBody(false)
	if err == nil {
		t.Error("expected error for empty body")
	}
}

// Tests below use #{} interpolation and depend on _expression (File 14).
// They will be #[ignore] in Rust until the expression parser is ported.

func TestInterpolatedIdentifierInterpolation(t *testing.T) {
	p := newTestStylesheetParser(`#{$var}rest`)
	interp, err := p.interpolatedIdentifier(false)
	if err != nil {
		t.Fatal(err)
	}
	if interp.IsPlain() {
		t.Error("expected non-plain interpolation")
	}
	if initial := interp.InitialPlain(); initial != "" {
		t.Errorf("InitialPlain() = %q, want ''", initial)
	}
}

func TestInterpolatedIdentifierBodyInterpolation(t *testing.T) {
	p := newTestStylesheetParser(`f#{$var}`)
	p.scanner.ScanChar('f')
	interp, err := p.interpolatedIdentifierBody(false)
	if err != nil {
		t.Fatal(err)
	}
	if interp.IsPlain() {
		t.Error("expected non-plain interpolation body")
	}
}

func TestSingleInterpolation(t *testing.T) {
	p := newTestStylesheetParser(`#{$var}`)
	expr, span, err := p.singleInterpolation()
	if err != nil {
		t.Fatal(err)
	}
	if expr == nil {
		t.Error("expected non-nil expression")
	}
	_ = span
}

func TestSingleInterpolationPlainCss(t *testing.T) {
	p := newTestStylesheetParser(`#{$var}`)
	p.plainCss = true
	_, _, err := p.singleInterpolation()
	if err == nil {
		t.Error("expected error for interpolation in plain CSS")
	}
}
