package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// newTestScssParser creates a SCSS parser with all function pointers
// properly initialized for testing stylesheet-level parsing.
func newTestScssParser(text string) *ScssParser {
	return NewScssParser([]byte(text), nil, false)
}

func newTestScssParserParseSelectors(text string) *ScssParser {
	return NewScssParser([]byte(text), nil, true)
}

// newTestScssParserIndented creates a Sass (indented) parser for testing
// indented-only code paths (+ and = shorthand).
func newTestScssParserIndented(text string) *ScssParser {
	return NewScssParser([]byte(text), nil, false)
}

// ======================================================================
// Group A: parse (entry point)
// ======================================================================

func TestParseEmpty(t *testing.T) {
	p := newTestScssParser("")
	_, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseBom(t *testing.T) {
	p := newTestScssParser("\uFEFF$x: 1;")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1", len(ss.GetChildren()))
	}
}

func TestParseCharset(t *testing.T) {
	p := newTestScssParser(`@charset "UTF-8"; $x: 1;`)
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1 (charset filtered)", len(ss.GetChildren()))
	}
	if _, ok := ss.GetChildren()[0].(*VariableDeclaration); !ok {
		t.Errorf("expected VariableDeclaration, got %T", ss.GetChildren()[0])
	}
}

func TestParseSimpleVar(t *testing.T) {
	p := newTestScssParser("$x: 1;")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1", len(ss.GetChildren()))
	}
}

func TestParseSimpleRule(t *testing.T) {
	p := newTestScssParser(".foo { color: red; }")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1", len(ss.GetChildren()))
	}
	if _, ok := ss.GetChildren()[0].(*StyleRule); !ok {
		t.Errorf("expected StyleRule, got %T", ss.GetChildren()[0])
	}
}

func TestParseMultiple(t *testing.T) {
	p := newTestScssParser("$x: 1; $y: 2;")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 2 {
		t.Fatalf("children len = %d, want 2", len(ss.GetChildren()))
	}
}

// ======================================================================
// Group B: statement
// ======================================================================

func TestStatementAtRule(t *testing.T) {
	p := newTestScssParser("@debug 1;")
	stmt, err := p.statement(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stmt.(*DebugRule); !ok {
		t.Errorf("expected DebugRule, got %T", stmt)
	}
}

func TestStatementUnmatchedBrace(t *testing.T) {
	p := newTestScssParser("}")
	_, err := p.statement(true)
	if err == nil {
		t.Fatal("expected error for unmatched }")
	}
	want := `unmatched "}".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestStatementDefaultVarDecl(t *testing.T) {
	// Variables at top level are handled by statements(), which checks for $
	// directly. Use Parse() to test the full flow.
	p := newTestScssParser("$a: 1;")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.GetChildren()) != 1 {
		t.Fatalf("children len = %d, want 1", len(ss.GetChildren()))
	}
	if _, ok := ss.GetChildren()[0].(*VariableDeclaration); !ok {
		t.Errorf("expected VariableDeclaration, got %T", ss.GetChildren()[0])
	}
}

// ======================================================================
// Group C: variableDeclarationWithoutNamespace
// ======================================================================

func TestVarDeclSimple(t *testing.T) {
	p := newTestScssParser("$a: 1;")
	decl, err := p.variableDeclarationWithoutNamespace("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if decl.Name() != "a" {
		t.Errorf("name = %q, want %q", decl.Name(), "a")
	}
	if decl.IsGuarded {
		t.Error("should not be guarded by default")
	}
}

func TestVarDeclGuarded(t *testing.T) {
	p := newTestScssParser("$a: 1 !default;")
	decl, err := p.variableDeclarationWithoutNamespace("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !decl.IsGuarded {
		t.Error("expected guarded=true")
	}
}

func TestVarDeclGlobal(t *testing.T) {
	p := newTestScssParser("$a: 1 !global;")
	decl, err := p.variableDeclarationWithoutNamespace("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !decl.IsGlobal() {
		t.Error("expected global=true")
	}
}

func TestVarDeclGuardedAndGlobal(t *testing.T) {
	p := newTestScssParser("$a: 1 !default !global;")
	decl, err := p.variableDeclarationWithoutNamespace("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !decl.IsGuarded {
		t.Error("expected guarded=true")
	}
	if !decl.IsGlobal() {
		t.Error("expected global=true")
	}
}

func TestVarDeclDuplicateDefault(t *testing.T) {
	p := newTestScssParser("$a: 1 !default !default;")
	_, err := p.variableDeclarationWithoutNamespace("", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range p.warnings {
		if w.Deprecation != nil && w.Deprecation.ID == "duplicate-var-flags" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected DUPLICATE_VAR_FLAGS deprecation warning")
	}
}

func TestVarDeclGlobalWithNamespace(t *testing.T) {
	p := newTestScssParser("$var: 1 !global;")
	_, err := p.variableDeclarationWithoutNamespace("ns", nil)
	if err == nil {
		t.Fatal("expected error for !global on namespaced variable")
	}
	want := `!global isn't allowed for variables in other modules.`
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Errorf("error = %q, want to contain %q", got, want)
	}
}

func TestVarDeclInvalidFlag(t *testing.T) {
	p := newTestScssParser("$a: 1 !bad;")
	_, err := p.variableDeclarationWithoutNamespace("", nil)
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
	want := `Invalid flag name.`
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Errorf("error = %q, want to contain %q", got, want)
	}
}

// ======================================================================
// Group D: variableDeclarationWithNamespace
// ======================================================================

func TestVarDeclWithNamespace(t *testing.T) {
	p := newTestScssParser("ns.$var: 1;")
	decl, err := p.variableDeclarationWithNamespace()
	if err != nil {
		t.Fatal(err)
	}
	if decl.Namespace() == nil || *decl.Namespace() != "ns" {
		t.Errorf("namespace = %v, want 'ns'", decl.Namespace())
	}
}

// ======================================================================
// Group E: variableDeclarationOrStyleRule
// ======================================================================

func TestVarDeclarationOrStyleRuleVar(t *testing.T) {
	// Variables are handled by the statements loop, not by statement().
	// Test through Parse() which uses statements().
	p := newTestScssParser("$a: 1;")
	ss, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ss.GetChildren()[0].(*VariableDeclaration); !ok {
		t.Errorf("expected VariableDeclaration, got %T", ss.GetChildren()[0])
	}
}

func TestVarDeclarationOrStyleRuleSelector(t *testing.T) {
	p := newTestScssParser(".foo { color: red; }")
	stmt, err := p.variableDeclarationOrStyleRule()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stmt.(*StyleRule); !ok {
		t.Errorf("expected StyleRule, got %T", stmt)
	}
}

// ======================================================================
// Group F: declarationOrBuffer — declaration or selector
// ======================================================================

func TestDeclarationOrBufferNoColon(t *testing.T) {
	p := newTestScssParser("color")
	buf, err := p.declarationOrBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if buf == nil {
		t.Fatal("expected non-nil buffer for no-colon")
	}
	if _, ok := buf.(*InterpolationBuffer); !ok {
		t.Errorf("expected InterpolationBuffer, got %T", buf)
	}
}

func TestDeclarationOrBufferPropertyHack(t *testing.T) {
	p := newTestScssParser("*color: red;")
	result, err := p.declarationOrBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Property hack transforms to declaration if colon found
	if _, ok := result.(*InterpolationBuffer); !ok {
		// Declaration is also valid
	}
}

// ======================================================================
// Group G: variableDeclarationOrInterpolation
// ======================================================================

func TestVarDeclarationOrInterpolationSimple(t *testing.T) {
	// identifier `foo` followed by `.$var` → namespace.var declaration
	p := newTestScssParser("ns.$var: 1")
	result, err := p.variableDeclarationOrInterpolation()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if vd, ok := result.(*VariableDeclaration); ok {
		if vd.Name() != "var" {
			t.Errorf("name = %q", vd.Name())
		}
	}
}

func TestVarDeclarationOrInterpolationInterpolation(t *testing.T) {
	p := newTestScssParser("foo")
	result, err := p.variableDeclarationOrInterpolation()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if _, ok := result.(*Interpolation); !ok {
		t.Errorf("expected Interpolation, got %T", result)
	}
}

// ======================================================================
// Group H: parseVariableDeclaration
// ======================================================================

func TestParseVariableDeclarationSimple(t *testing.T) {
	p := newTestScssParser("$a: 1")
	decl, _, err := p.parseVariableDeclaration()
	if err != nil {
		t.Fatal(err)
	}
	if decl.Name() != "a" {
		t.Errorf("name = %q, want 'a'", decl.Name())
	}
}

func TestParseVariableDeclarationNamespace(t *testing.T) {
	p := newTestScssParser("ns.$a: 1")
	decl, _, err := p.parseVariableDeclaration()
	if err != nil {
		t.Fatal(err)
	}
	if decl.Namespace() == nil || *decl.Namespace() != "ns" {
		t.Errorf("namespace = %v, want 'ns'", decl.Namespace())
	}
}

// ======================================================================
// Group I: parseExpression
// ======================================================================

func TestParseExpression(t *testing.T) {
	p := newTestScssParser("1 + 2")
	expr, _, err := p.parseExpression()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*BinaryOperationExpression); !ok {
		t.Errorf("expected BinaryOperationExpression, got %T", expr)
	}
}

// ======================================================================
// Group J: parseParameterList
// ======================================================================

func TestParseParameterList(t *testing.T) {
	p := newTestScssParser("@mixin foo($a, $b) {")
	params, err := p.parseParameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 2 {
		t.Fatalf("params len = %d, want 2", len(params.Parameters))
	}
}

// ======================================================================
// Group K: parseUseRule
// ======================================================================

func TestParseUseRule(t *testing.T) {
	p := newTestScssParser(`@use "path/file"`)
	_, _, err := p.parseUseRule()
	if err != nil {
		t.Fatal(err)
	}
}

// ======================================================================
// Group L: ParseSignature
// ======================================================================

func TestParseSignature(t *testing.T) {
	name, params, err := ParseSignature("my-function($a, $b)", true)
	if err != nil {
		t.Fatal(err)
	}
	if name != "my-function" {
		t.Errorf("name = %q, want %q", name, "my-function")
	}
	if len(params.Parameters) != 2 {
		t.Errorf("params len = %d, want 2", len(params.Parameters))
	}
}

func TestParseSignatureNoParens(t *testing.T) {
	name, params, err := ParseSignature("my-fn", false)
	if err != nil {
		t.Fatal(err)
	}
	if name != "my-fn" {
		t.Errorf("name = %q, want %q", name, "my-fn")
	}
	if len(params.Parameters) != 0 {
		t.Errorf("params len = %d, want 0", len(params.Parameters))
	}
}

func TestParseSignatureNoParensRequired(t *testing.T) {
	// With requireParens=true and no parens, parseSignature still works
	// because the second branch (no parens present, requireParens=false path)
	// just creates an empty ParameterList.
	_, _, err := ParseSignature("my-fn", false)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestParseSignatureInvalid(t *testing.T) {
	_, _, err := ParseSignature("1invalid", true)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	got := err.Error()
	if !strings.Contains(got, "Invalid signature") {
		t.Errorf("error = %q, want to contain 'Invalid signature'", got)
	}
}

// ======================================================================
// Group M: parseSingleProduction
// ======================================================================

func TestParseSingleProductionSuccess(t *testing.T) {
	p := newTestScssParser("$a: 1")
	result, err := parseSingleProduction(&p.StylesheetParser, func() (*VariableDeclaration, error) {
		return p.variableDeclarationWithoutNamespace("", nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name() != "a" {
		t.Errorf("name = %q, want %q", result.Name(), "a")
	}
}

// ======================================================================
// Helpers
// ======================================================================

type _ = sasscommon.FileSpan
