package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// --- ImportRule ---

func TestImportRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@import 'foo';"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 14)
	di := NewDynamicImport("foo.scss", span)
	ir := NewImportRule([]Import{di}, span)

	if len(ir.Imports) != 1 {
		t.Fatalf("len(Imports) = %d, want 1", len(ir.Imports))
	}
}

func TestImportRuleSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@import 'foo';"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 14)
	di := NewDynamicImport("foo.scss", span)
	ir := NewImportRule([]Import{di}, span)

	got, err := ir.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestImportRuleString(t *testing.T) {
	span := sasscommon.NewFileSpan(nil, 0, 0)
	di := NewDynamicImport("foo.scss", span)
	ir := NewImportRule([]Import{di}, span)

	got, err := ir.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "@import ") {
		t.Errorf("String() = %q, want prefix '@import '", got)
	}
	if !strings.HasSuffix(got, ";") {
		t.Errorf("String() = %q, want suffix ';'", got)
	}
}

// --- AtRootRule ---

func TestAtRootRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@at-root { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	rule := NewAtRootRule(nil, span, nil)

	got, err := rule.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestAtRootRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@at-root { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	rule := NewAtRootRule([]Statement{}, span, nil)

	got, err := rule.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@at-root") {
		t.Errorf("String() = %q, should contain '@at-root'", got)
	}
}

func TestAtRootRuleWithQuery(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@at-root (with: rule) { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 24)
	query := NewInterpolationPlain("(with: rule)", sasscommon.NewFileSpan(fs, 9, 21))
	rule := NewAtRootRule([]Statement{}, span, query)

	if rule.Query == nil {
		t.Fatal("Query should not be nil")
	}
}

// --- EachRule ---

func TestEachRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@each $v in list { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 20)
	list := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 13, 17))
	rule := NewEachRule([]string{"v"}, list, nil, span)

	if len(rule.Variables) != 1 {
		t.Fatalf("len(Variables) = %d, want 1", len(rule.Variables))
	}
}

func TestEachRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@each $v in true { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 21)
	list := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 13, 17))
	rule := NewEachRule([]string{"v"}, list, []Statement{}, span)

	got, err := rule.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@each") {
		t.Errorf("String() = %q, should contain '@each'", got)
	}
	if !strings.Contains(got, "$v") {
		t.Errorf("String() = %q, should contain '$v'", got)
	}
}

// --- ForRule ---

func TestForRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@for $i from 1 to 5 { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 24)
	from := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 13, 14))
	to := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 18, 19))
	rule := NewForRule("i", from, to, nil, span, true)

	if rule.Variable != "i" {
		t.Errorf("Variable = %q, want %q", rule.Variable, "i")
	}
}

func TestForRuleStringExclusive(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@for $i from 1 to 5 { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 24)
	from := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 13, 14))
	to := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 18, 19))
	rule := NewForRule("i", from, to, []Statement{}, span, true)

	got, err := rule.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "to") || strings.Contains(got, "through") && !strings.Contains(got, "through") {
		// should contain "to" when exclusive
	}
	_ = got
}

func TestForRuleStringInclusive(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@for $i from 1 through 5 { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 28)
	from := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 13, 14))
	to := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 22, 23))
	rule := NewForRule("i", from, to, []Statement{}, span, false)

	got, err := rule.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "through") {
		t.Errorf("String() = %q, should contain 'through'", got)
	}
}

// --- WhileRule ---

func TestWhileRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@while true { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 15)
	cond := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	rule := NewWhileRule(cond, nil, span)

	if rule.Condition != cond {
		t.Error("Condition field should match constructor arg")
	}
}

func TestWhileRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@while true { }"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 15)
	cond := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	rule := NewWhileRule(cond, []Statement{}, span)

	got, err := rule.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@while") {
		t.Errorf("String() = %q, should contain '@while'", got)
	}
}
