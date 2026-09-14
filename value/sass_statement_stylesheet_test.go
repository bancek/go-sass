package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// --- Stylesheet ---

func TestStylesheetConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("a { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 5)
	ss := NewStylesheet(nil, span)

	got, err := ss.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestStylesheetString(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	ss := NewStylesheet([]Statement{}, span)

	got, err := ss.String()
	if err != nil {
		t.Fatal(err)
	}
	// Empty stylesheet produces empty string
	_ = got
}

func TestStylesheetUsesEmpty(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	ss := NewStylesheet([]Statement{}, span)

	uses := ss.Uses()
	if len(uses) != 0 {
		t.Errorf("len(Uses()) = %d, want 0", len(uses))
	}
	forwards := ss.Forwards()
	if len(forwards) != 0 {
		t.Errorf("len(Forwards()) = %d, want 0", len(forwards))
	}
}

func TestStylesheetIsPlainCss(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	warnings := []ParseTimeWarning{}
	gv := map[string]sasscommon.FileSpan{}
	ss := NewStylesheetDetailed([]Statement{}, span, warnings, true, gv)

	if !ss.IsPlainCss() {
		t.Error("IsPlainCss() should be true")
	}
}

func TestStylesheetWarnings(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	warnings := []ParseTimeWarning{
		{Message: "test warning", Span: span},
	}
	gv := map[string]sasscommon.FileSpan{}
	ss := NewStylesheetDetailed([]Statement{}, span, warnings, false, gv)

	got := ss.ParseTimeWarnings()
	if len(got) != 1 {
		t.Fatalf("len(ParseTimeWarnings()) = %d, want 1", len(got))
	}
	if got[0].Message != "test warning" {
		t.Errorf("ParseTimeWarnings()[0].Message = %q, want %q", got[0].Message, "test warning")
	}
}

func TestStylesheetUsesForwardsExtraction(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	useRule, _ := NewUseRule(nil, nil, span, nil)
	fwdRule := NewForwardRule(nil, span, nil, nil)
	ss := NewStylesheet([]Statement{useRule, fwdRule}, span)

	uses := ss.Uses()
	if len(uses) != 1 {
		t.Errorf("len(Uses()) = %d, want 1", len(uses))
	}
	forwards := ss.Forwards()
	if len(forwards) != 1 {
		t.Errorf("len(Forwards()) = %d, want 1", len(forwards))
	}
}

func TestStylesheetStringWithChildren(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@debug true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	dr := NewDebugRule(expr, span)
	ss := NewStylesheet([]Statement{dr}, span)

	got, err := ss.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@debug") {
		t.Errorf("String() = %q, should contain '@debug'", got)
	}
}
