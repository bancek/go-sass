package value

import (
	"testing"
)

func TestAlmostAnyValuePlainText(t *testing.T) {
	p := newTestStylesheetParser("foo")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != "foo" {
		t.Errorf("AsPlain() = %v, want 'foo'", s)
	}
}

func TestAlmostAnyValueQuotedString(t *testing.T) {
	p := newTestStylesheetParser(`"hello"`)
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != `"hello"` {
		t.Errorf("String() = %q, want %q", s, `"hello"`)
	}
}

func TestAlmostAnyValueSingleQuoted(t *testing.T) {
	p := newTestStylesheetParser(`'world'`)
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != `'world'` {
		t.Errorf("String() = %q, want %q", s, `'world'`)
	}
}

func TestAlmostAnyValueInterpolation(t *testing.T) {
	p := newTestStylesheetParser("#{foo}")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if interp.IsPlain() {
		t.Error("expected non-plain interpolation")
	}
}

func TestAlmostAnyValueEscapeBackslash(t *testing.T) {
	p := newTestStylesheetParser(`\a`)
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != `\a` {
		t.Errorf("AsPlain() = %v, want \\a", s)
	}
}

func TestAlmostAnyValueLoudCommentIncluded(t *testing.T) {
	p := newTestStylesheetParser("a/* comment */b")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a/* comment */b" {
		t.Errorf("String() = %q, want %q", s, "a/* comment */b")
	}
}

func TestAlmostAnyValueLoudCommentOmitted(t *testing.T) {
	p := newTestStylesheetParser("a/* comment */b")
	interp, err := p.almostAnyValue(true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "ab" {
		t.Errorf("String() = %q, want %q", s, "ab")
	}
}

func TestAlmostAnyValueSilentCommentIncluded(t *testing.T) {
	p := newTestStylesheetParser("a// comment\nb")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a// comment\nb" {
		t.Errorf("String() = %q, want %q", s, "a// comment\nb")
	}
}

func TestAlmostAnyValueSilentCommentOmitted(t *testing.T) {
	p := newTestStylesheetParser("a// comment\nb")
	interp, err := p.almostAnyValue(true)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a\nb" {
		t.Errorf("String() = %q, want %q", s, "a\nb")
	}
}

func TestAlmostAnyValueUrlFunction(t *testing.T) {
	p := newTestStylesheetParser("url(foo)")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "url(foo)" {
		t.Errorf("String() = %q, want %q", s, "url(foo)")
	}
}

func TestAlmostAnyValueUrlPrefixFunction(t *testing.T) {
	p := newTestStylesheetParser("url-prefix(foo)")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "url-prefix(foo)" {
		t.Errorf("String() = %q, want %q", s, "url-prefix(foo)")
	}
}

func TestAlmostAnyValueUnotURL(t *testing.T) {
	p := newTestStylesheetParser("underline")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "underline" {
		t.Errorf("String() = %q, want %q", s, "underline")
	}
}

func TestAlmostAnyValueTerminatesOnBang(t *testing.T) {
	p := newTestStylesheetParser("foo!bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "foo" {
		t.Errorf("String() = %q, want %q", s, "foo")
	}
}

func TestAlmostAnyValueTerminatesOnSemicolon(t *testing.T) {
	p := newTestStylesheetParser("foo;bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "foo" {
		t.Errorf("String() = %q, want %q", s, "foo")
	}
}

func TestAlmostAnyValueTerminatesOnOpenBrace(t *testing.T) {
	p := newTestStylesheetParser("foo{bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "foo" {
		t.Errorf("String() = %q, want %q", s, "foo")
	}
}

func TestAlmostAnyValueTerminatesOnCloseBrace(t *testing.T) {
	p := newTestStylesheetParser("foo}bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "foo" {
		t.Errorf("String() = %q, want %q", s, "foo")
	}
}

func TestAlmostAnyValueBracketTracking(t *testing.T) {
	p := newTestStylesheetParser("(foo)bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(foo)bar" {
		t.Errorf("String() = %q, want %q", s, "(foo)bar")
	}
}

func TestAlmostAnyValueSquareBracketTracking(t *testing.T) {
	p := newTestStylesheetParser("[foo]bar")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "[foo]bar" {
		t.Errorf("String() = %q, want %q", s, "[foo]bar")
	}
}

func TestAlmostAnyValueOpenBraceTerminatesEvenInParens(t *testing.T) {
	p := newTestStylesheetParser("(foo{bar})baz")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(foo" {
		t.Errorf("String() = %q, want %q", s, "(foo")
	}
}

func TestAlmostAnyValueMixedBrackets(t *testing.T) {
	p := newTestStylesheetParser("([x])")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "([x])" {
		t.Errorf("String() = %q, want %q", s, "([x])")
	}
}

func TestAlmostAnyValueNewlineSCSS(t *testing.T) {
	p := newTestStylesheetParser("hello\nworld")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello\nworld" {
		t.Errorf("String() = %q, want %q", s, "hello\nworld")
	}
}

func TestAlmostAnyValueNewlineIndentedTerminates(t *testing.T) {
	p := newTestStylesheetParser("hello\nworld")
	p.indented = true
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestAlmostAnyValueNewlineIndentedInsideBrackets(t *testing.T) {
	p := newTestStylesheetParser("(hello\nworld)")
	p.indented = true
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(hello\nworld)" {
		t.Errorf("String() = %q, want %q", s, "(hello\nworld)")
	}
}

func TestAlmostAnyValueEmpty(t *testing.T) {
	p := newTestStylesheetParser("")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != "" {
		t.Errorf("AsPlain() = %v, want ''", s)
	}
}

func TestAlmostAnyValueIdentifier(t *testing.T) {
	p := newTestStylesheetParser("my-var")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "my-var" {
		t.Errorf("String() = %q, want %q", s, "my-var")
	}
}

func TestAlmostAnyValueUnexpectedCloseParenError(t *testing.T) {
	p := newTestStylesheetParser(")")
	_, err := p.almostAnyValue(false)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Unexpected ")".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAlmostAnyValueUnexpectedCloseBracketError(t *testing.T) {
	p := newTestStylesheetParser("]")
	_, err := p.almostAnyValue(false)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Unexpected "]".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAlmostAnyValueEmptyparensNoClose(t *testing.T) {
	p := newTestStylesheetParser("(")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(" {
		t.Errorf("String() = %q, want %q", s, "(")
	}
}

func TestAlmostAnyValueNestedBrackets(t *testing.T) {
	p := newTestStylesheetParser("((foo))")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "((foo))" {
		t.Errorf("String() = %q, want %q", s, "((foo))")
	}
}

func TestAlmostAnyValueSlash(t *testing.T) {
	p := newTestStylesheetParser("/")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "/" {
		t.Errorf("String() = %q, want %q", s, "/")
	}
}

func TestAlmostAnyValueSpecialChars(t *testing.T) {
	p := newTestStylesheetParser("@%.")
	interp, err := p.almostAnyValue(false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "@%." {
		t.Errorf("String() = %q, want %q", s, "@%.")
	}
}

// =============================================================================
// interpolatedDeclarationValue tests
// =============================================================================

func TestInterpolatedDeclarationValueBasic(t *testing.T) {
	p := newTestStylesheetParser("hello")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestInterpolatedDeclarationValueColonTerminatesWhenDisallowed(t *testing.T) {
	p := newTestStylesheetParser("hello:world")
	opts := declarationValueOpts{allowColon: false, allowOpenBrace: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestInterpolatedDeclarationValueColonAllowedByDefault(t *testing.T) {
	p := newTestStylesheetParser("hello:world")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello:world" {
		t.Errorf("String() = %q, want %q", s, "hello:world")
	}
}

func TestInterpolatedDeclarationValueColonInsideBrackets(t *testing.T) {
	p := newTestStylesheetParser("(a:b)")
	opts := declarationValueOpts{allowColon: false, allowOpenBrace: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(a:b)" {
		t.Errorf("String() = %q, want %q", s, "(a:b)")
	}
}

func TestInterpolatedDeclarationValueSemicolonTerminates(t *testing.T) {
	p := newTestStylesheetParser("hello;world")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestInterpolatedDeclarationValueSemicolonAllowedWhenTrue(t *testing.T) {
	p := newTestStylesheetParser("hello;world")
	opts := declarationValueOpts{allowSemicolon: true, allowColon: true, allowOpenBrace: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello;world" {
		t.Errorf("String() = %q, want %q", s, "hello;world")
	}
}

func TestInterpolatedDeclarationValueOpenBraceTerminatesWhenDisallowed(t *testing.T) {
	p := newTestStylesheetParser("hello{world")
	opts := declarationValueOpts{allowOpenBrace: false, allowColon: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestInterpolatedDeclarationValueOpenBraceAllowedByDefault(t *testing.T) {
	p := newTestStylesheetParser("hello+world")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello+world" {
		t.Errorf("String() = %q, want %q", s, "hello+world")
	}
}

func TestInterpolatedDeclarationValueOpenBraceInsideParens(t *testing.T) {
	p := newTestStylesheetParser("(a{b})")
	opts := declarationValueOpts{allowOpenBrace: true, allowColon: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(a{b})" {
		t.Errorf("String() = %q, want %q", s, "(a{b})")
	}
}

func TestInterpolatedDeclarationValueSpaceCollapsing(t *testing.T) {
	p := newTestStylesheetParser("hello   world")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello world" {
		t.Errorf("String() = %q, want %q", s, "hello world")
	}
}

func TestInterpolatedDeclarationValueSpaceCollapseTabSequence(t *testing.T) {
	p := newTestStylesheetParser("hello\t \tworld")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	// Two spaces/tabs before the third: first two are silently consumed,
	// last one before 'w' is written.
	if s != "hello\tworld" {
		t.Errorf("String() = %q, want %q", s, "hello\tworld")
	}
}

func TestInterpolatedDeclarationValueSpaceNotCollapsedAfterNewline(t *testing.T) {
	p := newTestStylesheetParser("hello\n  world")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello\n  world" {
		t.Errorf("String() = %q, want %q", s, "hello\n  world")
	}
}

func TestInterpolatedDeclarationValueEscape(t *testing.T) {
	p := newTestStylesheetParser(`\61 bc`)
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	// \61 = 'a' (hex escape), trailing space consumed by escape
	if s != "abc" {
		t.Errorf("String() = %q, want %q", s, "abc")
	}
}

func TestInterpolatedDeclarationValueQuotedString(t *testing.T) {
	p := newTestStylesheetParser(`"hello"`)
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != `"hello"` {
		t.Errorf("String() = %q, want %q", s, `"hello"`)
	}
}

func TestInterpolatedDeclarationValueInterpolation(t *testing.T) {
	p := newTestStylesheetParser("before#{x}after")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "before#{x}after" {
		t.Errorf("String() = %q, want %q", s, "before#{x}after")
	}
}

func TestInterpolatedDeclarationValueLoudComment(t *testing.T) {
	p := newTestStylesheetParser("a/* comment */b")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "a/* comment */b" {
		t.Errorf("String() = %q, want %q", s, "a/* comment */b")
	}
}

func TestInterpolatedDeclarationValueSilentCommentOmitted(t *testing.T) {
	p := newTestStylesheetParser("hello// comment\nworld")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello\nworld" {
		t.Errorf("String() = %q, want %q", s, "hello\nworld")
	}
}

func TestInterpolatedDeclarationValueSilentCommentPassedAsText(t *testing.T) {
	p := newTestStylesheetParser("hello// comment\nworld")
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: false}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello// comment\nworld" {
		t.Errorf("String() = %q, want %q", s, "hello// comment\nworld")
	}
}

func TestInterpolatedDeclarationValueUrlFunction(t *testing.T) {
	p := newTestStylesheetParser("url(http://x.com)")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "url(http://x.com)" {
		t.Errorf("String() = %q, want %q", s, "url(http://x.com)")
	}
}

func TestInterpolatedDeclarationValueUrlPrefixFunction(t *testing.T) {
	p := newTestStylesheetParser("url-prefix(http://)")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "url-prefix(http://)" {
		t.Errorf("String() = %q, want %q", s, "url-prefix(http://)")
	}
}

func TestInterpolatedDeclarationValueEndAfterOfTerminates(t *testing.T) {
	p := newTestStylesheetParser("1 of 2")
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: true, endAfterOf: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "1 of" {
		t.Errorf("String() = %q, want %q", s, "1 of")
	}
}

func TestInterpolatedDeclarationValueEndAfterOfNoMatch(t *testing.T) {
	p := newTestStylesheetParser("offset")
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: true, endAfterOf: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "offset" {
		t.Errorf("String() = %q, want %q", s, "offset")
	}
}

func TestInterpolatedDeclarationValueEndAfterOfInsideBrackets(t *testing.T) {
	p := newTestStylesheetParser("(1 of 2)")
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: true, endAfterOf: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(1 of 2)" {
		t.Errorf("String() = %q, want %q", s, "(1 of 2)")
	}
}

func TestInterpolatedDeclarationValueAllowEmptyFalseError(t *testing.T) {
	p := newTestStylesheetParser("")
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: true}
	_, err := p.interpolatedDeclarationValue(opts)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `Expected token.`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestInterpolatedDeclarationValueAllowEmptyTrue(t *testing.T) {
	p := newTestStylesheetParser("")
	opts := declarationValueOpts{allowEmpty: true, allowColon: true, allowOpenBrace: true, silentComments: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != "" {
		t.Errorf("AsPlain() = %v, want ''", s)
	}
}

func TestInterpolatedDeclarationValueNestedBrackets(t *testing.T) {
	p := newTestStylesheetParser("(a (b) c)")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(a (b) c)" {
		t.Errorf("String() = %q, want %q", s, "(a (b) c)")
	}
}

func TestInterpolatedDeclarationValueMixedBracketTypes(t *testing.T) {
	p := newTestStylesheetParser("([x])")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "([x])" {
		t.Errorf("String() = %q, want %q", s, "([x])")
	}
}

func TestInterpolatedDeclarationValueNewlineSCSS(t *testing.T) {
	p := newTestStylesheetParser("hello\nworld")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello\nworld" {
		t.Errorf("String() = %q, want %q", s, "hello\nworld")
	}
}

func TestInterpolatedDeclarationValueNewlineIndentedTerminates(t *testing.T) {
	p := newTestStylesheetParser("hello\nworld")
	p.indented = true
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("String() = %q, want %q", s, "hello")
	}
}

func TestInterpolatedDeclarationValueNewlineIndentedConsumeNewlinesTrue(t *testing.T) {
	p := newTestStylesheetParser("hello\nworld")
	p.indented = true
	opts := declarationValueOpts{allowColon: true, allowOpenBrace: true, silentComments: true, consumeNewlines: true}
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello\nworld" {
		t.Errorf("String() = %q, want %q", s, "hello\nworld")
	}
}

func TestInterpolatedDeclarationValueNewlineIndentedInsideBrackets(t *testing.T) {
	p := newTestStylesheetParser("(hello\nworld)")
	p.indented = true
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(hello\nworld)" {
		t.Errorf("String() = %q, want %q", s, "(hello\nworld)")
	}
}

func TestInterpolatedDeclarationValueConsecutiveNewlinesCollapsed(t *testing.T) {
	p := newTestStylesheetParser("a\n\nb")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	// First \n: Writeln -> \n. Second \n: peek(-1) is \n -> skipped.
	if s != "a\nb" {
		t.Errorf("String() = %q, want %q", s, "a\nb")
	}
}

func TestInterpolatedDeclarationValueUnclosedBracketError(t *testing.T) {
	p := newTestStylesheetParser("(")
	opts := declarationValueOptsDefaults()
	_, err := p.interpolatedDeclarationValue(opts)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `expected ")".`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestInterpolatedDeclarationValueSemicolonInsideBrackets(t *testing.T) {
	p := newTestStylesheetParser("(a;b)")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	if s != "(a;b)" {
		t.Errorf("String() = %q, want %q", s, "(a;b)")
	}
}

func TestInterpolatedDeclarationValueComplexMixed(t *testing.T) {
	p := newTestStylesheetParser("a url(x) \"b\" c/* comment */d #{e} f")
	opts := declarationValueOptsDefaults()
	interp, err := p.interpolatedDeclarationValue(opts)
	if err != nil {
		t.Fatal(err)
	}
	s, err := interp.String()
	if err != nil {
		t.Fatal(err)
	}
	want := "a url(x) \"b\" c/* comment */d #{e} f"
	if s != want {
		t.Errorf("String() = %q, want %q", s, want)
	}
}
