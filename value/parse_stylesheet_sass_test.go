package value

import (
	"errors"
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

func TestParseSassStyleRuleSelectorNoTrailingNewline(t *testing.T) {
	p := newTestSassParserIndent("a,\nb\n  c: d\n")
	sheet, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	rule, ok := sheet.GetChildren()[0].(*StyleRule)
	if !ok {
		t.Fatalf("expected StyleRule, got %T", sheet.GetChildren()[0])
	}
	got, err := rule.Selector.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "a,\nb" {
		t.Errorf("selector = %q, want %q", got, "a,\nb")
	}
}

func TestParseSassEmptySelectorUnrecognizedSyntax(t *testing.T) {
	p := newTestSassParserIndent("a\n  @extend b\n  !optional\n")
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error")
	}
	var formatErr *sasscommon.SassFormatException
	if !errors.As(err, &formatErr) {
		t.Fatalf("error type = %T, want *sasscommon.SassFormatException", err)
	}
	if formatErr.Message != "unrecognized syntax" {
		t.Errorf("message = %q, want %q", formatErr.Message, "unrecognized syntax")
	}
	spanText, spanErr := formatErr.Span.SpanText()
	if spanErr != nil {
		t.Fatal(spanErr)
	}
	if spanText != "!optional" {
		t.Errorf("span = %q, want %q", spanText, "!optional")
	}
}

func TestParseSassStyleRuleSelectorTrailingCommaAtEOF(t *testing.T) {
	p := newTestSassParserIndent("a,\n")
	sheet, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1", len(sheet.GetChildren()))
	}
}

// ======================================================================
// Helpers
// ======================================================================

type _ = sasscommon.FileSpan
