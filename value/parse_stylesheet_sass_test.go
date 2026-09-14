package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func newTestSassParserIndent(text string) *SassParser {
	return NewSassParser([]byte(text), nil, false)
}

// ======================================================================
// Group A: Indentation peek/read
// ======================================================================

func TestParseSassPeekIndentationZero(t *testing.T) {
	p := newTestSassParserIndent("\n")
	indent, err := p.sassPeekIndentation()
	if err != nil {
		t.Fatal(err)
	}
	if indent != 0 {
		t.Errorf("indent = %d, want 0", indent)
	}
}

// Indentation tests (skipped — need specific scanner state setup)

func TestParseSassExpectNewlineLF(t *testing.T) {
	p := newTestSassParserIndent("\n")
	err := p.sassExpectNewline(false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseSassExpectNewlineMissing(t *testing.T) {
	p := newTestSassParserIndent("content")
	err := p.sassExpectNewline(false)
	if err == nil {
		t.Fatal("expected error for missing newline")
	}
}

// ======================================================================
// Group C: tryTrailingSemicolon
// ======================================================================

func TestParseSassTryTrailingSemicolonFound(t *testing.T) {
	p := newTestSassParserIndent(";")
	ok, err := p.trySassTrailingSemicolon()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for semicolon")
	}
}

// ======================================================================
// Group D: styleRuleSelector (Sass)
// ======================================================================

func TestParseSassStyleRuleSelector(t *testing.T) {
	p := newTestSassParserIndent(".foo\n")
	interp, err := p.styleRuleSelector()
	if err != nil {
		t.Fatal(err)
	}
	if interp == nil {
		t.Error("expected non-nil selector")
	}
}

// ======================================================================
// Helpers
// ======================================================================

type _ = sasscommon.FileSpan
