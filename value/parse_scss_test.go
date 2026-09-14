package value

import (
	"strings"
	"testing"
)

func newTestScssParserForTest(text string) *ScssParser {
	return NewScssParser([]byte(text), nil, false)
}

// ======================================================================
// Group A: scssSilentComment
// ======================================================================

func TestScssSilentCommentSimple(t *testing.T) {
	p := newTestScssParserForTest("// comment\n")
	comment, err := p.scssSilentComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

func TestScssSilentCommentMultiLine(t *testing.T) {
	p := newTestScssParserForTest("// line1\n  // line2\n")
	comment, err := p.scssSilentComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

func TestScssSilentCommentEof(t *testing.T) {
	p := newTestScssParserForTest("// no newline")
	comment, err := p.scssSilentComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

// ======================================================================
// Group B: scssLoudComment
// ======================================================================

func TestScssLoudCommentSimple(t *testing.T) {
	p := newTestScssParserForTest("/* comment */")
	comment, err := p.scssLoudComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

func TestScssLoudCommentWithInterpolation(t *testing.T) {
	p := newTestScssParserForTest("/* #{1 + 2} */")
	comment, err := p.scssLoudComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

func TestScssLoudCommentCRNormalize(t *testing.T) {
	p := newTestScssParserForTest("/* line\r\n*/")
	comment, err := p.scssLoudComment()
	if err != nil {
		t.Fatal(err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
}

// ======================================================================
// Group C: scssSilentComment — plainCss error
// ======================================================================

func TestScssSilentCommentPlainCss(t *testing.T) {
	p := newTestScssParserForTest("// comment\n")
	p.plainCss = true
	_, err := p.scssSilentComment()
	if err == nil {
		t.Fatal("expected error for silent comment in plain CSS")
	}
	want := "Silent comments aren't allowed in plain CSS."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ======================================================================
// Group D: expectStatementSeparator (SCSS)
// ======================================================================

func TestScssExpectStatementSeparatorSemicolon(t *testing.T) {
	p := newTestScssParserForTest("; next")
	err := p.expectStatementSeparator("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestScssExpectStatementSeparatorBrace(t *testing.T) {
	p := newTestScssParserForTest("} next")
	err := p.expectStatementSeparator("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestScssExpectStatementSeparatorEOF(t *testing.T) {
	p := newTestScssParserForTest("")
	err := p.expectStatementSeparator("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestScssExpectStatementSeparatorError(t *testing.T) {
	p := newTestScssParserForTest("x")
	err := p.expectStatementSeparator("test")
	if err == nil {
		t.Fatal("expected error for missing separator")
	}
}

// ======================================================================
// Group E: scanElse (SCSS)
// ======================================================================

func TestScssScanElseFound(t *testing.T) {
	p := newTestScssParserForTest(" @else { }")
	found, err := p.scanElse(0)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected found=true for @else")
	}
}

func TestScssScanElseIfDeprecation(t *testing.T) {
	p := newTestScssParserForTest(" @elseif true { }")
	_, err := p.scanElse(0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range p.warnings {
		if w.Deprecation != nil && w.Deprecation.ID == "elseif" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ELSEIF deprecation warning")
	}
}

// ======================================================================
// Group F: children (SCSS)
// ======================================================================

func TestScssChildrenEmpty(t *testing.T) {
	p := newTestScssParserForTest("{}")
	children, err := p.children(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Errorf("len = %d, want 0", len(children))
	}
}

func TestScssChildrenVarDecl(t *testing.T) {
	p := newTestScssParserForTest("{ $a: 1; }")
	children, err := p.children(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 {
		t.Fatalf("len = %d, want 1", len(children))
	}
	if _, ok := children[0].(*VariableDeclaration); !ok {
		t.Errorf("expected VariableDeclaration, got %T", children[0])
	}
}

func TestScssChildrenSemicolon(t *testing.T) {
	p := newTestScssParserForTest("{ ; }")
	children, err := p.children(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Errorf("len = %d, want 0 (semicolons filtered)", len(children))
	}
}

func TestScssChildrenMissingBrace(t *testing.T) {
	p := newTestScssParserForTest("{")
	_, err := p.children(func() (Statement, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error for missing }")
	}
}

// ======================================================================
// Group G: statements (SCSS)
// ======================================================================

func TestScssStatementsEmpty(t *testing.T) {
	p := newTestScssParserForTest("")
	stmts, err := p.statements(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 0 {
		t.Errorf("len = %d, want 0", len(stmts))
	}
}

func TestScssStatementsVarDecl(t *testing.T) {
	p := newTestScssParserForTest("$a: 1;")
	stmts, err := p.statements(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("len = %d, want 1", len(stmts))
	}
}

func TestScssStatementsMultipleVars(t *testing.T) {
	p := newTestScssParserForTest("$a: 1; $b: 2;")
	stmts, err := p.statements(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 2 {
		t.Fatalf("len = %d, want 2", len(stmts))
	}
}

func TestScssStatementsSilentComment(t *testing.T) {
	p := newTestScssParserForTest("// comment\n$a: 1;")
	stmts, err := p.statements(func() (Statement, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 2 {
		t.Fatalf("len = %d, want 2 (silent comment is a statement)", len(stmts))
	}
}

// ======================================================================
// Group H: styleRuleSelector (SCSS)
// ======================================================================

func TestScssStyleRuleSelector(t *testing.T) {
	p := newTestScssParserForTest(".foo { }")
	interp, err := p.styleRuleSelector()
	if err != nil {
		t.Fatal(err)
	}
	if interp == nil {
		t.Error("expected non-nil selector")
	}
}
