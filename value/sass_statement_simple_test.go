package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func simpleSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

// --- SilentComment ---

func TestSilentCommentConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("// comment"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	sc := NewSilentComment("// comment", span)

	if sc.Text != "// comment" {
		t.Errorf("Text = %q, want %q", sc.Text, "// comment")
	}
}

func TestSilentCommentSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("// comment"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	sc := NewSilentComment("// comment", span)

	got, err := sc.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestSilentCommentString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("// comment"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	sc := NewSilentComment("// comment", span)

	got := sc.String()
	if got != "// comment" {
		t.Errorf("String() = %q, want %q", got, "// comment")
	}
}

func TestSilentCommentDocComment(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("/// doc"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 7)
	sc := NewSilentComment("/// doc", span)

	got := sc.DocComment()
	if got == nil {
		t.Fatal("DocComment() returned nil")
	}
	if *got != "doc" {
		t.Errorf("DocComment() = %q, want %q", *got, "doc")
	}
}

func TestSilentCommentDocCommentMultiLine(t *testing.T) {
	sc := NewSilentComment("/// line 1\n/// line 2\nnon-doc line", nil)

	got := sc.DocComment()
	if got == nil {
		t.Fatal("DocComment() returned nil")
	}
	if *got != "line 1\nline 2" {
		t.Errorf("DocComment() = %q, want %q", *got, "line 1\nline 2")
	}
}

// --- LoudComment ---

func TestLoudCommentConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("/* comment */"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 13)
	text := NewInterpolationPlain("/* comment */", span)
	lc := NewLoudComment(text)

	if lc.Text != text {
		t.Error("Text field should match constructor arg")
	}
}

func TestLoudCommentSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("/* comment */"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 13)
	text := NewInterpolationPlain("/* comment */", span)
	lc := NewLoudComment(text)

	got, err := lc.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestLoudCommentString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("/* comment */"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 13)
	text := NewInterpolationPlain("/* comment */", span)
	lc := NewLoudComment(text)

	got, err := lc.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/* comment */" {
		t.Errorf("String() = %q, want %q", got, "/* comment */")
	}
}

// --- ContentRule ---

func TestContentRuleConstruction(t *testing.T) {
	span := simpleSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)

	if cr.Arguments != args {
		t.Error("Arguments field should match constructor arg")
	}
}

func TestContentRuleSpan(t *testing.T) {
	span := simpleSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)

	got, err := cr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestContentRuleStringEmpty(t *testing.T) {
	span := simpleSpan("@content;", 0, 9)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)

	got, err := cr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "@content;" {
		t.Errorf("String() = %q, want %q", got, "@content;")
	}
}

func TestContentRuleStringWithArgs(t *testing.T) {
	span := simpleSpan("@content($a);", 0, 13)
	args := NewArgumentListEmpty(sasscommon.NewSimpleFileSpan(nil, 0, 0))
	cr := NewContentRule(args, span)

	got, err := cr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("String() should not be empty")
	}
}

// --- DebugRule ---

func TestDebugRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@debug true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	dr := NewDebugRule(expr, span)

	if dr.Expression != expr {
		t.Error("Expression field should match constructor arg")
	}
}

func TestDebugRuleSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@debug true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	dr := NewDebugRule(expr, span)

	got, err := dr.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestDebugRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@debug true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	dr := NewDebugRule(expr, span)

	got, err := dr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "@debug true;" {
		t.Errorf("String() = %q, want %q", got, "@debug true;")
	}
}

// --- ErrorRule ---

func TestErrorRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@error \"msg\";"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewStringExpressionPlain("msg", sasscommon.NewFileSpan(fs, 7, 10), true)
	er := NewErrorRule(expr, span)

	if er.Expression != expr {
		t.Error("Expression field should match constructor arg")
	}
}

func TestErrorRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@error \"msg\";"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	expr := NewStringExpressionPlain("msg", sasscommon.NewFileSpan(fs, 7, 10), true)
	er := NewErrorRule(expr, span)

	got, err := er.String()
	if err != nil {
		t.Fatal(err)
	}
	expected := "@error \"msg\";"
	if got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

// --- WarnRule ---

func TestWarnRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@warn true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 6, 10))
	wr := NewWarnRule(expr, span)

	if wr.Expression != expr {
		t.Error("Expression field should match constructor arg")
	}
}

func TestWarnRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@warn true;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	expr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 6, 10))
	wr := NewWarnRule(expr, span)

	got, err := wr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "@warn true;" {
		t.Errorf("String() = %q, want %q", got, "@warn true;")
	}
}

// --- ReturnRule ---

func TestReturnRuleConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@return 42;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	expr := NewNumberExpression(42, sasscommon.NewFileSpan(fs, 8, 10), nil)
	rr := NewReturnRule(expr, span)

	if rr.Expression != expr {
		t.Error("Expression field should match constructor arg")
	}
}

func TestReturnRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@return 42;"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	expr := NewNumberExpression(42, sasscommon.NewFileSpan(fs, 8, 10), nil)
	rr := NewReturnRule(expr, span)

	got, err := rr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "@return 42;" {
		t.Errorf("String() = %q, want %q", got, "@return 42;")
	}
}
