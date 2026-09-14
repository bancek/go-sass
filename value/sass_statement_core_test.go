package value

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func coreSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

// --- ContentBlock ---

func TestContentBlockConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(" using ($a) { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 15)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(fs, 8, 12))
	cb := NewContentBlock(params, nil, span)

	if cb.Declaration() == nil {
		t.Fatal("Declaration() returned nil")
	}
}

func TestContentBlockString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("{ }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 3)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	cb := NewContentBlock(params, []Statement{}, span)

	got, err := cb.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "{") {
		t.Error("String() should contain '{'")
	}
}

// --- VariableDeclaration ---

func TestVariableDeclarationConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	vd, err := NewVariableDeclaration("name", expr, span, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vd.Name() != "name" {
		t.Errorf("Name() = %q, want %q", vd.Name(), "name")
	}
}

func TestVariableDeclarationNameSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	vd, err := NewVariableDeclaration("name", expr, span, nil, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := vd.NameSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("NameSpan() returned nil")
	}
}

func TestVariableDeclarationGlobalNamespace(t *testing.T) {
	ns := "mod"
	_, err := NewVariableDeclaration("name", nil, nil, &ns, false, true, nil)
	if err == nil {
		t.Error("Expected error for namespace + !global")
	}
}

// --- MediaRule ---

func TestMediaRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@media screen { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 17)
	query := NewInterpolationPlain("screen", sasscommon.NewFileSpan(fs, 7, 13))
	mr := NewMediaRule(query, nil, span)

	if mr.Query != query {
		t.Error("Query field should match constructor arg")
	}
}

func TestMediaRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@media screen { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 17)
	query := NewInterpolationPlain("screen", sasscommon.NewFileSpan(fs, 7, 13))
	mr := NewMediaRule(query, []Statement{}, span)

	got, err := mr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@media") {
		t.Errorf("String() = %q, should contain '@media'", got)
	}
}

// --- SupportsRule ---

func TestSupportsRuleConstruction(t *testing.T) {
	span := supportSpan("@supports (a: b) { }", 0, 19)
	decl := NewSupportsDeclaration(
		NewStringExpressionPlain("a", supportSpan("a", 11, 12), false),
		NewStringExpressionPlain("b", supportSpan("b", 14, 15), false),
		supportSpan("(a: b)", 10, 16),
	)
	sr := NewSupportsRule(decl, nil, span)

	if sr.Condition == nil {
		t.Error("Condition should not be nil")
	}
}

func TestSupportsRuleString(t *testing.T) {
	span := supportSpan("@supports (a: b) { }", 0, 19)
	decl := NewSupportsDeclaration(
		NewStringExpressionPlain("a", supportSpan("a", 11, 12), false),
		NewStringExpressionPlain("b", supportSpan("b", 14, 15), false),
		supportSpan("(a: b)", 10, 16),
	)
	sr := NewSupportsRule(decl, []Statement{}, span)

	got, err := sr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@supports") {
		t.Errorf("String() = %q, should contain '@supports'", got)
	}
}

// --- AtRule ---

func TestAtRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@unknown;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 9)
	name := NewInterpolationPlain("unknown", sasscommon.NewFileSpan(fs, 1, 8))
	ar := NewAtRule(name, span, nil, nil)

	got, err := ar.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestAtRuleStringNoChildren(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@unknown;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 9)
	name := NewInterpolationPlain("unknown", sasscommon.NewFileSpan(fs, 1, 8))
	ar := NewAtRule(name, span, nil, nil)

	got, err := ar.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ";") {
		t.Errorf("String() = %q, should end with ';'", got)
	}
}

func TestAtRuleStringWithChildren(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@unknown { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	name := NewInterpolationPlain("unknown", sasscommon.NewFileSpan(fs, 1, 8))
	ar := NewAtRule(name, span, nil, []Statement{})

	got, err := ar.String()
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(got, ";") {
		t.Errorf("String() = %q, should end with '}'", got)
	}
}

// --- Declaration ---

func TestDeclarationConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("color: red;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	name := NewInterpolationPlain("color", sasscommon.NewFileSpan(fs, 0, 5))
	val := NewStringExpressionPlain("red", sasscommon.NewFileSpan(fs, 7, 10), false)
	d := NewDeclaration(name, val, span)

	if !d.ParsedAsSassScript() {
		t.Error("ParsedAsSassScript() should be true")
	}
}

func TestDeclarationString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("color: red;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	name := NewInterpolationPlain("color", sasscommon.NewFileSpan(fs, 0, 5))
	val := NewStringExpressionPlain("red", sasscommon.NewFileSpan(fs, 7, 10), false)
	d := NewDeclaration(name, val, span)

	got, err := d.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ";") {
		t.Errorf("String() = %q, should end with ';'", got)
	}
}

func TestDeclarationNested(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("color: red { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 14)
	name := NewInterpolationPlain("color", sasscommon.NewFileSpan(fs, 0, 5))
	val := NewStringExpressionPlain("red", sasscommon.NewFileSpan(fs, 7, 10), false)
	d := NewDeclarationNested(name, []Statement{}, span, val)

	if d.GetChildren() == nil {
		t.Fatal("Children should not be nil for nested declaration")
	}
}

// --- StyleRule ---

func TestStyleRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(".foo { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 8)
	sel := NewInterpolationPlain(".foo", sasscommon.NewFileSpan(fs, 0, 4))
	sr := NewStyleRule(sel, nil, span)

	if sr.Selector != sel {
		t.Error("Selector field should match constructor arg")
	}
}

func TestStyleRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte(".foo { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 8)
	sel := NewInterpolationPlain(".foo", sasscommon.NewFileSpan(fs, 0, 4))
	sr := NewStyleRule(sel, []Statement{}, span)

	got, err := sr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, ".foo") {
		t.Errorf("String() = %q, should contain '.foo'", got)
	}
}

// --- UseRule ---

func TestUseRuleConstruction(t *testing.T) {
	u, _ := url.Parse("https://example.com/module")
	span := supportSpan("@use 'mod';", 0, 10)
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ur.URL() != u {
		t.Error("URL() should return constructor arg")
	}
}

func TestUseRuleString(t *testing.T) {
	u, _ := url.Parse("https://example.com/module")
	span := supportSpan("@use 'mod';", 0, 10)
	ur, err := NewUseRule(u, nil, span, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ur.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "@use ") {
		t.Errorf("String() = %q, should start with '@use '", got)
	}
}

func TestUseRuleGuardedConfig(t *testing.T) {
	u, _ := url.Parse("https://example.com/module")
	span := supportSpan("@use 'mod';", 0, 10)
	cfg := NewConfiguredVariable("x", nil, span, true)
	_, err := NewUseRule(u, nil, span, []*ConfiguredVariable{cfg})
	if err == nil {
		t.Error("Expected error for guarded configuration variable")
	}
}

// --- ForwardRule ---

func TestForwardRuleConstruction(t *testing.T) {
	u, _ := url.Parse("https://example.com/module")
	span := supportSpan("@forward 'mod';", 0, 14)
	fr := NewForwardRule(u, span, nil, nil)

	if fr.URL() != u {
		t.Error("URL() should return constructor arg")
	}
}

func TestForwardRuleString(t *testing.T) {
	u, _ := url.Parse("https://example.com/module")
	span := supportSpan("@forward 'mod';", 0, 14)
	fr := NewForwardRule(u, span, nil, nil)

	got, err := fr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "@forward ") {
		t.Errorf("String() = %q, should start with '@forward '", got)
	}
}

// --- IncludeRule ---

func TestIncludeRuleConstruction(t *testing.T) {
	span := supportSpan("@include foo;", 0, 12)
	args := NewArgumentListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	ir := NewIncludeRule("foo", args, span, nil, nil)

	if ir.Name() != "foo" {
		t.Errorf("Name() = %q, want %q", ir.Name(), "foo")
	}
}

func TestIncludeRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@include foo;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	args := NewArgumentListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	ir := NewIncludeRule("foo", args, span, nil, nil)

	got, err := ir.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "@include foo") {
		t.Errorf("String() = %q, should start with '@include foo'", got)
	}
}

// --- MixinRule ---

func TestMixinRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@mixin foo { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 14)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	mr := NewMixinRule("foo", params, nil, span, nil)

	if mr.Name() != "foo" {
		t.Errorf("Name() = %q, want %q", mr.Name(), "foo")
	}
}

func TestMixinRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@mixin foo { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 14)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	mr := NewMixinRule("foo", params, []Statement{}, span, nil)

	got, err := mr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@mixin") {
		t.Errorf("String() = %q, should contain '@mixin'", got)
	}
}

func TestMixinRuleNameNormalization(t *testing.T) {
	// Matches Dart: CallableDeclaration converts underscores to hyphens for
	// name, while originalName is preserved.
	span := coreSpan("@mixin foo_bar { }", 0, 18)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	mr := NewMixinRule("foo_bar", params, nil, span, nil)

	if mr.Name() != "foo-bar" {
		t.Errorf("Name() = %q, want %q", mr.Name(), "foo-bar")
	}
	if mr.Declaration().OriginalName() != "foo_bar" {
		t.Errorf("OriginalName = %q, want %q", mr.Declaration().OriginalName(), "foo_bar")
	}
}

func TestContentBlockName(t *testing.T) {
	// Matches Dart: ContentBlock passes "@content" as its name.
	span := coreSpan("{ }", 0, 3)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	cb := NewContentBlock(params, nil, span)

	if cb.Name() != "@content" {
		t.Errorf("Name() = %q, want %q", cb.Name(), "@content")
	}
}

// --- CallableDeclaration.HasContent ---

func TestHasContentFalse(t *testing.T) {
	span := coreSpan("@mixin m { }", 0, 12)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	mr := NewMixinRule("m", params, nil, span, nil)

	if mr.HasContent() {
		t.Error("HasContent() = true, want false")
	}
}

func TestHasContentDirectChild(t *testing.T) {
	span := coreSpan("@mixin m { @content; }", 0, 22)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	content := NewContentRule(nil, coreSpan("@content;", 0, 9))
	mr := NewMixinRule("m", params, []Statement{content}, span, nil)

	if !mr.HasContent() {
		t.Error("HasContent() = false, want true")
	}
}

func TestHasContentNested(t *testing.T) {
	// The @content search recurses through child statements
	// (Dart: _HasContentVisitor via StatementSearchVisitor).
	span := coreSpan("@mixin m { @if true { @content; } }", 0, 35)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	content := NewContentRule(nil, coreSpan("@content;", 0, 9))
	clause := NewIfClause(NewBooleanExpression(true, coreSpan("true", 0, 4)), []Statement{content})
	ifRule := NewIfRule([]*IfClause{clause}, coreSpan("@if true { @content; }", 0, 22), nil)
	mr := NewMixinRule("m", params, []Statement{ifRule}, span, nil)

	if !mr.HasContent() {
		t.Error("HasContent() = false, want true")
	}
}

func TestHasContentLazyCached(t *testing.T) {
	span := coreSpan("@mixin m { @content; }", 0, 22)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	content := NewContentRule(nil, coreSpan("@content;", 0, 9))
	mr := NewMixinRule("m", params, []Statement{content}, span, nil)

	if !mr.HasContent() {
		t.Fatal("first HasContent() = false, want true")
	}
	if !mr.HasContent() {
		t.Error("second HasContent() = false, want true (cached)")
	}
}

// --- FunctionRule ---

func TestFunctionRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@function foo() { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 18)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	fr := NewFunctionRule("foo", params, nil, span, nil)

	if fr.Name() != "foo" {
		t.Errorf("Name() = %q, want %q", fr.Name(), "foo")
	}
}

func TestFunctionRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@function foo() { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 18)
	params := NewParameterListEmpty(sasscommon.NewFileSpan(nil, 0, 0))
	fr := NewFunctionRule("foo", params, []Statement{}, span, nil)

	got, err := fr.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@function") {
		t.Errorf("String() = %q, should contain '@function'", got)
	}
}

// --- IfRule ---

func TestIfRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@if true { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	cond := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 4, 8))
	clause := NewIfClause(cond, nil)
	ir := NewIfRule([]*IfClause{clause}, span, nil)

	if len(ir.Clauses) != 1 {
		t.Fatalf("len(Clauses) = %d, want 1", len(ir.Clauses))
	}
}

func TestIfRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@if true { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	cond := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 4, 8))
	clause := NewIfClause(cond, []Statement{})
	ir := NewIfRule([]*IfClause{clause}, span, nil)

	got, err := ir.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@if") {
		t.Errorf("String() = %q, should contain '@if'", got)
	}
}

func TestIfRuleWithElse(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@if true { } @else { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 22)
	cond := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 4, 8))
	clause := NewIfClause(cond, []Statement{})
	last := NewElseClause([]Statement{})
	ir := NewIfRule([]*IfClause{clause}, span, last)

	if ir.LastClause == nil {
		t.Fatal("LastClause should not be nil")
	}
}
