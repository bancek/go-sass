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
	indent, err := p.sassPeekIndentation(false)
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

func TestParseSassIndentedFirstStatementSpan(t *testing.T) {
	// The error span covers the indentation of the first non-blank line, not
	// the whole document start. A document whose first line is blank or
	// whitespace-only must still be rejected.
	tests := []struct {
		source string
		start  int
		end    int
	}{
		{"  a: b\n", 0, 2},
		{"\n  a: b\n", 1, 3},
		{"  \t\n  a: b\n", 4, 6},
	}

	for _, tt := range tests {
		p := newTestSassParserIndent(tt.source)
		_, err := p.statements(func() (Statement, error) { return nil, nil })
		if err == nil {
			t.Errorf("source %q: expected an error", tt.source)
			continue
		}
		if err.Error() != "Indenting at the beginning of the document is illegal." {
			t.Errorf("source %q: error = %q", tt.source, err.Error())
			continue
		}
		var scanErr *sasscommon.ScanError
		if !errors.As(err, &scanErr) {
			t.Errorf("source %q: error = %T, want *sasscommon.ScanError", tt.source, err)
			continue
		}
		start, startErr := scanErr.Span.StartLocation()
		if startErr != nil {
			t.Fatal(startErr)
		}
		end, endErr := scanErr.Span.EndLocation()
		if endErr != nil {
			t.Fatal(endErr)
		}
		if start.Offset != tt.start || end.Offset != tt.end {
			t.Errorf("source %q: span = (%d, %d), want (%d, %d)",
				tt.source, start.Offset, end.Offset, tt.start, tt.end)
		}
	}
}

// ======================================================================
// Helpers
// ======================================================================

type _ = sasscommon.FileSpan
