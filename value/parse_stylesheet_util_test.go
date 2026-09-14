package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// --- lookingAtInterpolatedIdentifier ---

func TestLookingAtInterpolatedIdentifierNameStart(t *testing.T) {
	p := newTestStylesheetParser("foo")
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for name-start char")
	}
}

func TestLookingAtInterpolatedIdentifierBackslash(t *testing.T) {
	p := newTestStylesheetParser(`\41`)
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for backslash")
	}
}

func TestLookingAtInterpolatedIdentifierHashBrace(t *testing.T) {
	p := newTestStylesheetParser(`#{`)
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for #{")
	}
}

func TestLookingAtInterpolatedIdentifierDashNameStart(t *testing.T) {
	p := newTestStylesheetParser("-a")
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for -a")
	}
}

func TestLookingAtInterpolatedIdentifierDashHashBrace(t *testing.T) {
	p := newTestStylesheetParser("-#{")
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for -#{")
	}
}

func TestLookingAtInterpolatedIdentifierDoubleDash(t *testing.T) {
	p := newTestStylesheetParser("--")
	if !p.lookingAtInterpolatedIdentifier() {
		t.Error("expected true for --")
	}
}

func TestLookingAtInterpolatedIdentifierNonMatch(t *testing.T) {
	p := newTestStylesheetParser("123")
	if p.lookingAtInterpolatedIdentifier() {
		t.Error("expected false for digits")
	}
}

func TestLookingAtInterpolatedIdentifierNonMatchHash(t *testing.T) {
	p := newTestStylesheetParser("#foo")
	if p.lookingAtInterpolatedIdentifier() {
		t.Error("expected false for # without {")
	}
}

func TestLookingAtInterpolatedIdentifierEOF(t *testing.T) {
	p := newTestStylesheetParser("")
	if p.lookingAtInterpolatedIdentifier() {
		t.Error("expected false for EOF")
	}
}

// --- lookingAtPotentialPropertyHack ---

func TestLookingAtPotentialPropertyHackColon(t *testing.T) {
	p := newTestStylesheetParser(":")
	if !p.lookingAtPotentialPropertyHack() {
		t.Error("expected true for :")
	}
}

func TestLookingAtPotentialPropertyHackAsterisk(t *testing.T) {
	p := newTestStylesheetParser("*")
	if !p.lookingAtPotentialPropertyHack() {
		t.Error("expected true for *")
	}
}

func TestLookingAtPotentialPropertyHackDot(t *testing.T) {
	p := newTestStylesheetParser(".")
	if !p.lookingAtPotentialPropertyHack() {
		t.Error("expected true for .")
	}
}

func TestLookingAtPotentialPropertyHackHash(t *testing.T) {
	p := newTestStylesheetParser("#foo")
	if !p.lookingAtPotentialPropertyHack() {
		t.Error("expected true for # without {")
	}
}

func TestLookingAtPotentialPropertyHackHashBrace(t *testing.T) {
	p := newTestStylesheetParser("#{")
	if p.lookingAtPotentialPropertyHack() {
		t.Error("expected false for #{")
	}
}

func TestLookingAtPotentialPropertyHackNonMatch(t *testing.T) {
	p := newTestStylesheetParser("foo")
	if p.lookingAtPotentialPropertyHack() {
		t.Error("expected false for non-hack char")
	}
}

// --- lookingAtInterpolatedIdentifierBody ---

func TestLookingAtInterpolatedIdentifierBodyName(t *testing.T) {
	p := newTestStylesheetParser("foo")
	if !p.lookingAtInterpolatedIdentifierBody() {
		t.Error("expected true for name char")
	}
}

func TestLookingAtInterpolatedIdentifierBodyBackslash(t *testing.T) {
	p := newTestStylesheetParser(`\41`)
	if !p.lookingAtInterpolatedIdentifierBody() {
		t.Error("expected true for backslash")
	}
}

func TestLookingAtInterpolatedIdentifierBodyHashBrace(t *testing.T) {
	p := newTestStylesheetParser(`#{`)
	if !p.lookingAtInterpolatedIdentifierBody() {
		t.Error("expected true for #{")
	}
}

func TestLookingAtInterpolatedIdentifierBodyNonMatch(t *testing.T) {
	p := newTestStylesheetParser(".")
	if p.lookingAtInterpolatedIdentifierBody() {
		t.Error("expected false for .")
	}
}

func TestLookingAtInterpolatedIdentifierBodyEOF(t *testing.T) {
	p := newTestStylesheetParser("")
	if p.lookingAtInterpolatedIdentifierBody() {
		t.Error("expected false for EOF")
	}
}

// --- lookingAtExpression ---

func TestLookingAtExpressionDot(t *testing.T) {
	p := newTestStylesheetParser(".5")
	if !p.lookingAtExpression() {
		t.Error("expected true for .")
	}
}

func TestLookingAtExpressionDoubleDot(t *testing.T) {
	p := newTestStylesheetParser("..")
	if p.lookingAtExpression() {
		t.Error("expected false for ..")
	}
}

func TestLookingAtExpressionExclamation(t *testing.T) {
	p := newTestStylesheetParser("!")
	if !p.lookingAtExpression() {
		t.Error("expected true for !")
	}
}

func TestLookingAtExpressionExclamationI(t *testing.T) {
	p := newTestStylesheetParser("!important")
	if !p.lookingAtExpression() {
		t.Error("expected true for !i")
	}
}

func TestLookingAtExpressionExclamationWhitespace(t *testing.T) {
	p := newTestStylesheetParser("! ")
	if !p.lookingAtExpression() {
		t.Error("expected true for ! followed by whitespace")
	}
}

func TestLookingAtExpressionParen(t *testing.T) {
	p := newTestStylesheetParser("(")
	if !p.lookingAtExpression() {
		t.Error("expected true for (")
	}
}

func TestLookingAtExpressionSlash(t *testing.T) {
	p := newTestStylesheetParser("/")
	if !p.lookingAtExpression() {
		t.Error("expected true for /")
	}
}

func TestLookingAtExpressionBracket(t *testing.T) {
	p := newTestStylesheetParser("[")
	if !p.lookingAtExpression() {
		t.Error("expected true for [")
	}
}

func TestLookingAtExpressionSingleQuote(t *testing.T) {
	p := newTestStylesheetParser("'")
	if !p.lookingAtExpression() {
		t.Error("expected true for '")
	}
}

func TestLookingAtExpressionDoubleQuote(t *testing.T) {
	p := newTestStylesheetParser(`"`)
	if !p.lookingAtExpression() {
		t.Error("expected true for \"")
	}
}

func TestLookingAtExpressionHash(t *testing.T) {
	p := newTestStylesheetParser("#")
	if !p.lookingAtExpression() {
		t.Error("expected true for #")
	}
}

func TestLookingAtExpressionPlus(t *testing.T) {
	p := newTestStylesheetParser("+")
	if !p.lookingAtExpression() {
		t.Error("expected true for +")
	}
}

func TestLookingAtExpressionMinus(t *testing.T) {
	p := newTestStylesheetParser("-")
	if !p.lookingAtExpression() {
		t.Error("expected true for -")
	}
}

func TestLookingAtExpressionBackslash(t *testing.T) {
	p := newTestStylesheetParser(`\`)
	if !p.lookingAtExpression() {
		t.Error("expected true for \\")
	}
}

func TestLookingAtExpressionDollar(t *testing.T) {
	p := newTestStylesheetParser("$")
	if !p.lookingAtExpression() {
		t.Error("expected true for $")
	}
}

func TestLookingAtExpressionAmpersand(t *testing.T) {
	p := newTestStylesheetParser("&")
	if !p.lookingAtExpression() {
		t.Error("expected true for &")
	}
}

func TestLookingAtExpressionPercent(t *testing.T) {
	p := newTestStylesheetParser("%")
	if !p.lookingAtExpression() {
		t.Error("expected true for %")
	}
}

func TestLookingAtExpressionNameStart(t *testing.T) {
	p := newTestStylesheetParser("a")
	if !p.lookingAtExpression() {
		t.Error("expected true for name-start char")
	}
}

func TestLookingAtExpressionDigit(t *testing.T) {
	p := newTestStylesheetParser("1")
	if !p.lookingAtExpression() {
		t.Error("expected true for digit")
	}
}

func TestLookingAtExpressionNonMatch(t *testing.T) {
	p := newTestStylesheetParser(";")
	if p.lookingAtExpression() {
		t.Error("expected false for ;")
	}
}

func TestLookingAtExpressionEOF(t *testing.T) {
	p := newTestStylesheetParser("")
	if p.lookingAtExpression() {
		t.Error("expected false for EOF")
	}
}

// --- urlString ---

func TestUrlStringValid(t *testing.T) {
	p := newTestStylesheetParser(`"https://example.com/style.scss"`)
	url, err := p.urlString()
	if err != nil {
		t.Fatal(err)
	}
	if url.Scheme != "https" {
		t.Errorf("scheme = %q, want %q", url.Scheme, "https")
	}
	if url.Host != "example.com" {
		t.Errorf("host = %q, want %q", url.Host, "example.com")
	}
	if url.Path != "/style.scss" {
		t.Errorf("path = %q, want %q", url.Path, "/style.scss")
	}
}

func TestUrlStringInvalid(t *testing.T) {
	// An invalid percent-encoded sequence causes url.Parse to fail
	p := newTestStylesheetParser(`"%ZZ"`)
	_, err := p.urlString()
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestUrlStringSassScheme(t *testing.T) {
	p := newTestStylesheetParser(`"sass:color"`)
	url, err := p.urlString()
	if err != nil {
		t.Fatal(err)
	}
	if url.Scheme != "sass" {
		t.Errorf("scheme = %q, want %q", url.Scheme, "sass")
	}
	if url.Path != "color" {
		t.Errorf("path = %q, want %q", url.Path, "color")
	}
}

// --- publicIdentifier ---

func TestPublicIdentifierValid(t *testing.T) {
	p := newTestStylesheetParser("myvar")
	result, err := p.publicIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	if result != "myvar" {
		t.Errorf("result = %q, want %q", result, "myvar")
	}
}

func TestPublicIdentifierUnderscorePrefix(t *testing.T) {
	p := newTestStylesheetParser("_myvar")
	_, err := p.publicIdentifier()
	if err == nil {
		t.Error("expected error for underscore-prefixed identifier")
	}
}

func TestPublicIdentifierDashPrefix(t *testing.T) {
	p := newTestStylesheetParser("-myvar")
	_, err := p.publicIdentifier()
	if err == nil {
		t.Error("expected error for dash-prefixed identifier")
	}
}

// --- assertPublic ---

func TestAssertPublicPublic(t *testing.T) {
	p := newTestStylesheetParser("")
	span := p.scanner.EmptySpan()
	err := p.assertPublic("myvar", func() sasscommon.FileSpan { return span })
	if err != nil {
		t.Errorf("unexpected error for public identifier: %v", err)
	}
}

func TestAssertPublicPrivate(t *testing.T) {
	p := newTestStylesheetParser("")
	span := p.scanner.EmptySpan()
	err := p.assertPublic("_myvar", func() sasscommon.FileSpan { return span })
	if err == nil {
		t.Error("expected error for private identifier")
	}
}

func TestAssertPublicPrivateDash(t *testing.T) {
	p := newTestStylesheetParser("")
	span := p.scanner.EmptySpan()
	err := p.assertPublic("-myvar", func() sasscommon.FileSpan { return span })
	if err == nil {
		t.Error("expected error for dash-prefixed identifier")
	}
}

// --- addOrInject ---

func TestAddOrInjectUnquotedString(t *testing.T) {
	// An unquoted string expression should have its text inlined via addInterpolation
	p := newTestStylesheetParser(`hello`)
	interp, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	expr := &StringExpression{Text: interp, HasQuotes: false}

	var buf InterpolationBuffer
	err = p.addOrInject(&buf, expr)
	if err != nil {
		t.Fatal(err)
	}
	result, err := buf.Interpolation(p.scanner.EmptySpan())
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsPlain() {
		t.Error("expected plain interpolation after addOrInject with unquoted string")
	}
	if s := result.AsPlain(); s == nil || *s != "hello" {
		t.Errorf("AsPlain() = %v, want 'hello'", s)
	}
}

func TestAddOrInjectQuotedString(t *testing.T) {
	p := newTestStylesheetParser("")
	span := p.scanner.EmptySpan()
	// Manually construct a quoted StringExpression
	interp := NewInterpolationPlain("hello", span)
	expr := &StringExpression{Text: interp, HasQuotes: true}

	var buf InterpolationBuffer
	err := p.addOrInject(&buf, expr)
	if err != nil {
		t.Fatal(err)
	}
	// The buffer should have the expression (not the text inlined)
	result, err := buf.Interpolation(p.scanner.EmptySpan())
	if err != nil {
		t.Fatal(err)
	}
	if result.IsPlain() {
		t.Error("expected non-plain interpolation after addOrInject with quoted string")
	}
}

func TestAddOrInjectInterpolationExpression(t *testing.T) {
	// Create a simple unquoted interpolation, which should be inlined
	p := newTestStylesheetParser(`hello`)
	interp, err := p.interpolatedIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	expr := &StringExpression{Text: interp, HasQuotes: false}

	var buf InterpolationBuffer
	err = p.addOrInject(&buf, expr)
	if err != nil {
		t.Fatal(err)
	}
	if buf.IsEmpty() {
		t.Error("expected non-empty buffer")
	}
}

// --- withChildren is tested via `children` which depends on File 16+17+18 ---
// Tests will be added when children_impl is ported.
